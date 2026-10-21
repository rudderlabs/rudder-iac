package ingest

import (
	"crypto/md5" //nolint:gosec // rudderId is an MD5-based identifier, not a secret.
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/tidwall/gjson"
)

// maxRequestIPLen bounds request_ip, because the sender controls
// X-Forwarded-For and each event of a batch carries it.
const maxRequestIPLen = 64

// Enrichment holds the fields RudderStack adds to an accepted event. It stays
// beside the event, so Event.Message keeps the bytes the SDK sent.
type Enrichment struct {
	// MessageID is set only when RudderStack replaces the event's messageId.
	MessageID  string `json:"messageId,omitempty"`
	ReceivedAt string `json:"receivedAt"`
	// RequestIP is set only when the event has no request_ip.
	RequestIP string `json:"request_ip,omitempty"`
	RudderID  string `json:"rudderId"`
	// Type is set on a single-event route, whose type RudderStack overwrites.
	Type string `json:"type,omitempty"`
}

// enrich reads each event from sanitized, the messages without \u0000
// escapes.
func (h *Handler) enrich(c *Capture, sanitized [][]byte, reqType string) {
	var (
		receivedAt = c.ReceivedAt.Format(time.RFC3339Nano)
		// A copy, so the record does not keep the whole header value alive.
		ip        = strings.Clone(requestIP(c.Request))
		eventType string
	)
	if reqType != "batch" && reqType != "import" {
		eventType = reqType
	}

	for i := range c.Events {
		v := topLevel(gjson.ParseBytes(sanitized[i]), "messageId", "userId", "anonymousId", "request_ip")

		e := &Enrichment{
			ReceivedAt: receivedAt,
			RudderID:   rudderID(sanitizeAndTrim(identity(v["userId"])), sanitizeAndTrim(identity(v["anonymousId"]))),
			Type:       eventType,
		}
		sent := v["messageId"].String()
		if id := sanitizeAndTrim(sent); id == "" {
			e.MessageID = h.newUUID()
		} else if id != sent {
			// A copy, so the record does not keep the whole parsed event alive.
			e.MessageID = strings.Clone(id)
		}
		if !v["request_ip"].Exists() {
			e.RequestIP = ip
		}
		c.Events[i].Enrichment = e
	}
}

// identity rounds a number to a float64, because the rudderId RudderStack
// derives shows that rounding.
func identity(v gjson.Result) string {
	if v.Type != gjson.Number {
		return v.String()
	}
	f, _ := strconv.ParseFloat(v.Raw, 64)
	b, _ := json.Marshal(f)
	return string(b)
}

// requestIP copies rudder-go-kit httputil.GetRequestIP and answers 0.0.0.0
// when it finds no address.
func requestIP(r *http.Request) string {
	ip, _, _ := strings.Cut(r.Header.Get("X-Forwarded-For"), ",")
	if ip != "" {
		ip = strings.ReplaceAll(ip, " ", "")
	} else if i := strings.LastIndexByte(r.RemoteAddr, ':'); i >= 0 {
		ip = r.RemoteAddr[:i]
	}
	if ip == "" {
		return "0.0.0.0"
	}
	return ip[:min(len(ip), maxRequestIPLen)]
}

// rudderID copies rudder-go-kit uuid.GetMD5UUID.
func rudderID(userID, anonymousID string) string {
	sum := md5.Sum([]byte(userID + ":" + anonymousID)) //nolint:gosec // see the import
	sum[6] = sum[6]&0x0f | 0x40
	sum[8] = sum[8]&0x3f | 0x80
	return uuid.UUID(sum).String()
}
