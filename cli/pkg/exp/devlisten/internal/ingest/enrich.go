package ingest

import (
	"crypto/md5" //nolint:gosec // rudderId is an MD5 UUID in the oracle; it is an identifier, not a secret.
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten/internal/store"
)

type enrichInput struct {
	reqType    string
	receivedAt time.Time
	requestIP  string
	newUUID    func() string
}

// enrich copies the oracle enricher chain in its order: messageId, type,
// receivedAt, request_ip, rudderId (SVC internal/gateway/enricher/enricher.go).
// The oracle first round-trips the event through `any` (rudder-server
// misc.SanitizeJSON), so numbers become float64; this copy does the same.
func enrich(event []byte, in enrichInput) (json.RawMessage, error) {
	var m map[string]any
	if err := json.Unmarshal(event, &m); err != nil {
		return nil, fmt.Errorf("decoding event: %w", err)
	}

	messageID := jsonString(m["messageId"])
	if sanitized := sanitizeAndTrim(messageID); sanitized == "" {
		m["messageId"] = in.newUUID()
	} else if sanitized != messageID {
		m["messageId"] = sanitized
	}

	switch in.reqType {
	case "batch", "import":
	default:
		m["type"] = in.reqType
	}

	m["receivedAt"] = in.receivedAt.Format(time.RFC3339Nano)

	if _, ok := m["request_ip"]; !ok {
		m["request_ip"] = in.requestIP
	}

	m["rudderId"] = rudderID(sanitizeAndTrim(jsonString(m["userId"])), sanitizeAndTrim(jsonString(m["anonymousId"])))

	out, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("encoding event: %w", err)
	}
	return out, nil
}

// jsonString mirrors rudder-go-kit jsonparser.GetStringOrEmpty: strings as
// is, numbers and booleans as their text, null and absent as "", objects and
// arrays as JSON.
func jsonString(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case float64, bool, json.Number:
		return fmt.Sprint(t)
	default:
		b, _ := json.Marshal(t)
		return string(b)
	}
}

// rudderID copies rudder-go-kit uuid.GetMD5UUID: the MD5 of
// userId + ":" + anonymousId with the RFC 4122 variant and version 4 bits.
func rudderID(userID, anonymousID string) string {
	sum := md5.Sum([]byte(userID + ":" + anonymousID)) //nolint:gosec // see import
	sum[8] = sum[8]&(0xff>>2) | (0x02 << 6)
	sum[6] = (sum[6] & 0x0f) | (4 << 4)
	return formatUUID(sum)
}

func newUUIDv4() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return formatUUID(b)
}

func formatUUID(b [16]byte) string {
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// requestIP copies SVC getRequestIP (handler.go:560-568) and rudder-go-kit
// httputil.GetRequestIP: the event's request_ip, then the first
// X-Forwarded-For entry, then the host part of RemoteAddr.
func requestIP(r *http.Request, event []byte) string {
	var probe struct {
		RequestIP any `json:"request_ip"`
	}
	_ = json.Unmarshal(event, &probe)
	if ip := jsonString(probe.RequestIP); ip != "" {
		return ip
	}
	if first := strings.Split(r.Header.Get("X-Forwarded-For"), ",")[0]; first != "" {
		return strings.ReplaceAll(first, " ", "")
	}
	parts := strings.Split(r.RemoteAddr, ":")
	if ip := strings.Join(parts[:len(parts)-1], ":"); ip != "" {
		return ip
	}
	return "0.0.0.0"
}

// newEvent builds the captured event. Scalars come from the enriched copy
// when there is one, so `type` and `messageId` show what the pipeline saw.
func newEvent(idx int, message, enriched json.RawMessage) store.Event {
	source := message
	if enriched != nil {
		source = enriched
	}
	var fields struct {
		Type, Event, UserID, AnonymousID, MessageID any
	}
	var m map[string]any
	_ = json.Unmarshal(source, &m)
	fields.Type, fields.Event, fields.UserID, fields.AnonymousID, fields.MessageID =
		m["type"], m["event"], m["userId"], m["anonymousId"], m["messageId"]

	return store.Event{
		Idx:             idx,
		Type:            nullableString(fields.Type),
		Event:           nullableString(fields.Event),
		UserID:          nullableString(fields.UserID),
		AnonymousID:     nullableString(fields.AnonymousID),
		MessageID:       nullableString(fields.MessageID),
		Message:         message,
		EnrichedMessage: enriched,
	}
}

func nullableString(v any) *string {
	if v == nil {
		return nil
	}
	s := jsonString(v)
	return &s
}
