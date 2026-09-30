// Package ingest answers SDK requests the way RudderStack ingestion does, and
// hands every request to a Sink.
package ingest

import (
	"encoding/json"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"
)

const (
	// MaxInFlight bounds memory, because any web page can post to the listener.
	MaxInFlight = 8

	// BodyReadTimeout stops a stalled upload from holding an in-flight slot for ever.
	BodyReadTimeout = 10 * time.Second

	// maxReqSize bounds a body, raw and decoded, so MaxInFlight bodies stay
	// small in memory.
	maxReqSize = 2000 * 1024

	// maxBatchEvents bounds the per-event slices that a batch of tiny events
	// allocates.
	maxBatchEvents = 10_000

	// maxPixelKeys and maxPixelKeyDepth bound a pixel query to keep memory small.
	maxPixelKeys     = 1000
	maxPixelKeyDepth = 32
)

// Sink receives each request before its answer is written, so a sender that
// has read its response always finds its request captured.
type Sink interface {
	Capture(c *Capture)
}

// Capture is one request and the answer it got.
type Capture struct {
	ReceivedAt time.Time
	// Kind is "ingestion" or "control".
	Kind string
	// Route is the path after the double-slash rewrite.
	Route string
	// Transport is "http", "beacon" or "pixel".
	Transport string
	// Request has its body already read.
	Request  *http.Request
	WriteKey string

	// Body is the raw entity, cut at the size limit.
	Body []byte
	// Decoded is Body after gzip, or nil when decoding failed.
	Decoded []byte
	// BodyComplete is false when reading or gzip decoding failed or hit the limit.
	BodyComplete bool
	Events       []Event

	StatusCode   int
	Header       http.Header
	ResponseBody []byte
	// Rejection is set on a refused pixel request too, although it answers 200.
	Rejection *Rejection
}

// Event is one event of a parsed request.
type Event struct {
	// Message is a slice of Capture.Decoded, never re-encoded, so a developer
	// sees exactly what the SDK sent.
	Message json.RawMessage
	// FromQuery is true on a pixel request, whose Message the handler built
	// from the query string.
	FromQuery bool
}

// Rejection names the stage that refused a request and the message
// RudderStack answers, without the trailing newline.
type Rejection struct {
	Stage  string
	Reason string
	// Idx is the index of the event that failed, or nil.
	Idx *int
}

type endpoint struct {
	method  string
	reqType string
}

var endpoints = map[string]endpoint{
	"/v1/alias":        {http.MethodPost, "alias"},
	"/v1/audiencelist": {http.MethodPost, "audiencelist"},
	"/v1/batch":        {http.MethodPost, "batch"},
	"/v1/group":        {http.MethodPost, "group"},
	"/v1/identify":     {http.MethodPost, "identify"},
	"/v1/import":       {http.MethodPost, "import"},
	"/v1/merge":        {http.MethodPost, "merge"},
	"/v1/page":         {http.MethodPost, "page"},
	"/v1/screen":       {http.MethodPost, "screen"},
	"/v1/track":        {http.MethodPost, "track"},
	"/beacon/v1/batch": {http.MethodPost, "batch"},
	"/pixel/v1/track":  {http.MethodGet, "track"},
	"/pixel/v1/page":   {http.MethodGet, "page"},
}

// Handler serves the ingestion routes.
type Handler struct {
	sink        Sink
	writeKeys   []string
	slots       chan struct{}
	readTimeout time.Duration
	now         func() time.Time
}

// New returns a handler that captures into sink. An empty writeKeys accepts
// every request, a missing key included; otherwise only those keys pass.
func New(sink Sink, writeKeys []string) *Handler {
	return &Handler{
		sink:        sink,
		writeKeys:   writeKeys,
		slots:       make(chan struct{}, MaxInFlight),
		readTimeout: BodyReadTimeout,
		now:         time.Now,
	}
}

// ServeHTTP checks a request in the order that gives each failure the answer
// RudderStack gives.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "//") {
		r.URL.Path = r.URL.Path[1:]
	}
	path := r.URL.Path
	c := &Capture{
		ReceivedAt: h.now().UTC(),
		Kind:       "control",
		Route:      path,
		Transport:  "http",
		Request:    r,
	}
	switch {
	case strings.HasPrefix(path, "/beacon/"):
		c.Transport = "beacon"
	case strings.HasPrefix(path, "/pixel/"):
		c.Transport = "pixel"
	}

	if isPreflight(r) {
		preflight(w.Header(), r)
		h.respond(w, c, reply{status: http.StatusNoContent})
		return
	}
	actualCORS(w.Header(), r)
	// A preflight stays control on any route: it carries no event.
	if _, ok := endpoints[path]; ok {
		c.Kind = "ingestion"
	}

	if len(r.URL.RawQuery) > maxReqSize {
		h.respond(w, c, reject(errQueryTooLarge))
		return
	}
	// Read before the in-flight check, so a 503 names its key too.
	if c.Transport == "http" {
		c.WriteKey, _, _ = r.BasicAuth()
	} else {
		c.WriteKey = queryValue(r.URL.RawQuery, "writeKey")
	}
	select {
	case h.slots <- struct{}{}:
		defer func() { <-h.slots }()
	default:
		// Without Connection: close, net/http reads the rest of a small body
		// before it answers, so a stalled body holds the 503 back.
		w.Header().Set("Connection", "close")
		h.respond(w, c, reject(errServiceUnavailable))
		return
	}

	h.respond(w, c, h.route(w, r, c))
}

func (h *Handler) route(w http.ResponseWriter, r *http.Request, c *Capture) reply {
	// Decode gzip before routing, so a bad gzip body fails the same way on
	// every route.
	early, late := h.readBody(w, r, c)
	if early != nil {
		return reject(*early)
	}

	ep, ok := endpoints[r.URL.Path]
	switch {
	case !ok:
		return reject(errUnknownPath)
	case r.Method != ep.method:
		return methodNotAllowed(ep.method)
	case c.Transport == "pixel":
		return h.pixel(r, c, ep.reqType)
	}
	return h.ingest(r, c, ep.reqType, late)
}

// ingest takes late, a body error that RudderStack answers only after the
// write key passes, so the key check runs first.
func (h *Handler) ingest(r *http.Request, c *Capture, reqType string, late *gwError) reply {
	missing := errNoWriteKeyInBasicAuth
	if c.Transport == "beacon" {
		missing = errNoWriteKeyInQuery
	}
	if e := h.checkKey(c.WriteKey, missing); e != nil {
		return reject(*e)
	}

	switch {
	case late != nil:
		return reject(*late)
	case r.Header.Get("Content-Encoding") != "gzip" && r.Body == http.NoBody:
		return reject(errRequestBodyNil)
	}
	return process(c, c.Decoded, reqType)
}

// pixel answers the GIF whatever the inner request gets, as RudderStack
// does, so a failure shows only in the capture.
func (h *Handler) pixel(r *http.Request, c *Capture, reqType string) reply {
	gif := reply{status: http.StatusOK, contentType: "image/gif", body: []byte(pixelGIF)}

	// Count before the parse, which holds every pair in memory.
	if strings.Count(r.URL.RawQuery, "&") >= maxPixelKeys {
		return reject(errPixelTooManyKeys)
	}
	query := r.URL.Query()
	for key := range query {
		if strings.Count(key, ".") >= maxPixelKeyDepth {
			return reject(errPixelKeyTooDeep)
		}
	}
	if e := h.checkKey(c.WriteKey, errNoWriteKeyInQuery); e != nil {
		gif.rejection = reject(*e).rejection
		return gif
	}

	payload, err := pixelPayload(query, reqType, c.ReceivedAt)
	if err != nil {
		gif.rejection = &Rejection{Stage: "parse", Reason: err.Error()}
		return gif
	}
	gif.rejection = process(c, payload, reqType).rejection
	for i := range c.Events {
		c.Events[i].FromQuery = true
	}
	return gif
}

// checkKey checks the write key only with an allowlist, so a local SDK works
// with any key.
func (h *Handler) checkKey(key string, missing gwError) *gwError {
	switch {
	case len(h.writeKeys) == 0:
		return nil
	case key == "":
		return &missing
	case !slices.Contains(h.writeKeys, key):
		return &errInvalidWriteKey
	}
	return nil
}

func process(c *Capture, payload []byte, reqType string) reply {
	events, e, idx := parseEvents(payload, reqType)
	c.Events = events
	if e != nil {
		rep := reject(*e)
		rep.rejection.Idx = idx
		return rep
	}
	return reply{status: http.StatusOK, body: []byte("ok")}
}

// queryValue returns what url.ParseQuery(rawQuery).Get(key) returns, but
// unescapes only the pair it returns, so a query of many pairs costs no memory.
func queryValue(rawQuery, key string) string {
	for rawQuery != "" {
		var pair string
		pair, rawQuery, _ = strings.Cut(rawQuery, "&")
		if strings.Contains(pair, ";") {
			continue
		}
		k, v, _ := strings.Cut(pair, "=")
		// A longer key cannot unescape to key, since an escaped byte takes three characters.
		if len(k) > 3*len(key) {
			continue
		}
		if k, err := url.QueryUnescape(k); err != nil || k != key {
			continue
		}
		if v, err := url.QueryUnescape(v); err == nil {
			return v
		}
	}
	return ""
}
