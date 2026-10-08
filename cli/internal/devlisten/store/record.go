package store

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/url"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/rudderlabs/rudder-iac/cli/internal/devlisten/ingest"
)

// Record is one captured request and the answer it got.
type Record struct {
	Seq        uint64            `json:"seq"`
	ReceivedAt time.Time         `json:"receivedAt"`
	Kind       string            `json:"kind"`
	Route      string            `json:"route"`
	Transport  string            `json:"transport"`
	WriteKey   WriteKey          `json:"writeKey"`
	Request    Request           `json:"request"`
	Response   Response          `json:"response"`
	Rejection  *ingest.Rejection `json:"rejection"`
	Events     []ingest.Event    `json:"events"`

	size int
}

type Request struct {
	Method     string `json:"method"`
	Target     string `json:"target"`
	RemoteAddr string `json:"remoteAddr"`
	// Headers has the first value of each of keptHeaders.
	Headers map[string]string `json:"headers"`
	// DroppedHeaders counts the header values Headers leaves out.
	DroppedHeaders int `json:"droppedHeaders"`
	// Body is the entity after gzip decoding, or as received when decoding failed.
	Body Body `json:"body"`
	// BodyBytes is the size of the entity as received.
	BodyBytes    int  `json:"bodyBytes"`
	BodyComplete bool `json:"bodyComplete"`
}

type Response struct {
	StatusCode int               `json:"statusCode"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
}

// Body marshals as a JSON string, so a body that is not valid JSON still encodes.
type Body []byte

func (b Body) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(b))
}

// WriteKey shows a key of up to 8 characters in clear: such keys are labels
// like "web" or "api". A longer key may be a real one, so Key shows only its
// first and last 4 characters and Sha256 lets a reader match it.
type WriteKey struct {
	Key    string `json:"key"`
	Sha256 string `json:"sha256,omitempty"`
}

// MaskWriteKey is the only form of a key that leaves the listener.
func MaskWriteKey(key string) WriteKey {
	if len(key) <= 8 {
		// The key may be a slice of the whole decoded Authorization header.
		return WriteKey{Key: strings.Clone(key)}
	}
	sum := sha256.Sum256([]byte(key))
	shown := key[:4] + "..."
	// A suffix would leave fewer than four characters hidden.
	if len(key) >= 12 {
		shown += key[len(key)-4:]
	}
	return WriteKey{Key: shown, Sha256: hex.EncodeToString(sum[:])}
}

// maskQuery masks every writeKey value in target, however it is escaped,
// because a preflight carries the key there without an Authorization header.
func maskQuery(target string) string {
	path, query, ok := strings.Cut(target, "?")
	if !ok {
		return target
	}
	var b strings.Builder
	b.Grow(len(target))
	b.WriteString(path)
	b.WriteByte('?')
	for more := true; more; {
		var pair string
		pair, query, more = strings.Cut(query, "&")
		k, v, _ := strings.Cut(pair, "=")
		// A longer name cannot unescape to writeKey, since an escaped byte takes three characters.
		if len(k) <= 3*len("writeKey") {
			name, err := url.QueryUnescape(k)
			if err == nil && name == "writeKey" {
				key, err := url.QueryUnescape(v)
				if err != nil {
					// A stray % fails the unescape, but the raw value still holds the key.
					key = v
				}
				pair = k + "=" + url.QueryEscape(MaskWriteKey(key).Key)
			}
		}
		b.WriteString(pair)
		if more {
			b.WriteByte('&')
		}
	}
	return b.String()
}

// keptHeaders are the request headers that help debug an SDK. The rest,
// credentials and cookies included, are only counted.
var keptHeaders = []string{"User-Agent", "Content-Type", "Content-Encoding", "Origin", "Anonymousid", "X-Forwarded-For"}

// maxHeaderValue bounds each kept value, because the sender controls it.
const maxHeaderValue = 1024

// maxRefusedBody bounds the body kept for a request refused before parsing,
// such as one over the size cap or with a corrupt gzip stream. The sender
// picks its size, and any web page can post to the listener, so keeping a
// refused 2 MB body lets 32 posts evict every capture.
const maxRefusedBody = 1024

// maxTarget bounds the kept request target. The sender controls the query,
// and the route is in the first bytes.
const maxTarget = 1024

// firstValues keeps the first value of each header in names, or of every
// header when names is nil, and counts the values it leaves out.
func firstValues(h http.Header, names []string) (map[string]string, int) {
	out := map[string]string{}
	dropped := 0
	for name, values := range h {
		if len(values) == 0 {
			continue
		}
		if names != nil && !slices.Contains(names, name) {
			dropped += len(values)
			continue
		}
		v := values[0]
		out[name] = strings.Clone(v[:min(len(v), maxHeaderValue)])
		dropped += len(values) - 1
	}
	return out, dropped
}

func newRecord(c *ingest.Capture) *Record {
	key := MaskWriteKey(c.WriteKey)
	body := c.Decoded
	if body == nil {
		body = c.Body
	}
	if len(body) == 0 {
		body = nil
	}
	bodyComplete := c.BodyComplete
	// A body that was not read or decoded in full never reached the parser, so
	// no event points into it and a copy lets the large array go. A body that
	// parsed and failed, such as truncated JSON, stays whole so the sender can
	// see where it broke. BodyBytes still reports the real size.
	if !c.BodyComplete && c.StatusCode >= http.StatusBadRequest && len(c.Events) == 0 && len(body) > maxRefusedBody {
		body = slices.Clone(body[:maxRefusedBody])
		bodyComplete = false
	}
	responseBody := string(c.ResponseBody)
	if key.Sha256 != "" {
		responseBody = strings.ReplaceAll(responseBody, c.WriteKey, key.Key)
		// The /sourceConfig answer is json.Marshal output, which escapes <, >,
		// &, quotes and control characters, so the key may be there only escaped.
		quoted, _ := json.Marshal(c.WriteKey)
		if escaped := string(quoted[1 : len(quoted)-1]); escaped != c.WriteKey {
			responseBody = strings.ReplaceAll(responseBody, escaped, key.Key)
		}
	}
	requestHeaders, dropped := firstValues(c.Request.Header, keptHeaders)
	responseHeaders, _ := firstValues(c.Header, nil)
	target := maskQuery(c.Request.RequestURI)
	target = target[:min(len(target), maxTarget)]
	// Method, target and route are slices of the whole request line, so the
	// record keeps copies.
	rec := &Record{
		ReceivedAt: c.ReceivedAt,
		Kind:       c.Kind,
		Route:      strings.Clone(c.Route),
		Transport:  c.Transport,
		WriteKey:   key,
		Request: Request{
			Method:         strings.Clone(c.Request.Method),
			Target:         strings.Clone(target),
			RemoteAddr:     c.Request.RemoteAddr,
			Headers:        requestHeaders,
			DroppedHeaders: dropped,
			Body:           body,
			BodyBytes:      len(c.Body),
			BodyComplete:   bodyComplete,
		},
		Response: Response{
			StatusCode: c.StatusCode,
			Headers:    responseHeaders,
			Body:       responseBody,
		},
		Rejection: c.Rejection,
		Events:    c.Events,
	}
	rec.size = rec.bytes()
	return rec
}

var (
	recordOverhead     = int(reflect.TypeFor[Record]().Size())
	eventOverhead      = int(reflect.TypeFor[ingest.Event]().Size())
	enrichmentOverhead = int(reflect.TypeFor[ingest.Enrichment]().Size())
)

// mapEntryOverhead estimates a map entry with two string headers.
const mapEntryOverhead = 64

// bytes estimates the memory the record holds. Parsed events are slices of
// the body, so the body's backing array is charged once.
func (r *Record) bytes() int {
	n := recordOverhead + len(r.Route) +
		len(r.Request.Method) + len(r.Request.Target) + len(r.Request.RemoteAddr) + cap(r.Request.Body) +
		headerBytes(r.Request.Headers) + headerBytes(r.Response.Headers) + len(r.Response.Body) +
		cap(r.Events)*eventOverhead
	if r.Rejection != nil {
		n += len(r.Rejection.Reason)
	}
	for _, ev := range r.Events {
		if e := ev.Enrichment; e != nil {
			n += enrichmentOverhead + len(e.MessageID) + len(e.ReceivedAt) + len(e.RequestIP) + len(e.RudderID) + len(e.Type)
		}
		if ev.FromQuery {
			n += cap(ev.Message)
		}
	}
	return n
}

func headerBytes(h map[string]string) int {
	n := 0
	for name, value := range h {
		n += mapEntryOverhead + len(name) + len(value)
	}
	return n
}
