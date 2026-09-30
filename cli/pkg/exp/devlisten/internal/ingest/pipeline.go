package ingest

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten/internal/store"
)

const (
	// maxReqSize is the SVC default Gateway.maxReqSizeInKB (4000 KB).
	maxReqSize = 4000 * 1024
	// maxEventsInBatch is the SVC default (internal/ratelimit/resolver.go:21).
	maxEventsInBatch = 20000
)

// ingest runs one ingestion request through the oracle's stages in order:
// gzip, auth, body, parse, identity, enrichment, size
// (SVC internal/gateway/handler.go:230-558).
func (g *Gateway) ingest(r *http.Request, reqType, transport string, raw []byte, receivedAt time.Time) reply {
	rep := reply{kind: "ingestion", transport: transport, header: http.Header{}, decoded: raw}

	gzipped := r.Header.Get("Content-Encoding") == "gzip"
	streamErr, early := rep.decode(raw, gzipped)
	if early != nil {
		return rep.reject(*early)
	}

	writeKey, rejection := authenticate(r, transport)
	rep.writeKey = writeKey
	if rejection = g.checkKey(writeKey, rejection); rejection != nil {
		return rep.reject(*rejection)
	}

	switch {
	case streamErr:
		return rep.reject(errRequestBodyReadFailed)
	case !gzipped && r.ContentLength == 0:
		return rep.reject(errRequestBodyNil)
	}
	return g.process(r, rep, reqType, receivedAt)
}

// decode gunzips when asked and applies the size limit to the raw and the
// decoded body. early is a rejection that comes before auth; streamErr is a
// broken stream, which the oracle rejects after auth.
func (rep *reply) decode(raw []byte, gzipped bool) (streamErr bool, early *gwError) {
	if gzipped {
		var headerErr *gwError
		rep.decoded, headerErr, streamErr = gunzip(raw)
		if headerErr != nil {
			return false, headerErr
		}
	}
	if len(raw) > maxReqSize || len(rep.decoded) > maxReqSize {
		rep.decoded = nil
		return false, &errRequestBodyTooLarge
	}
	return streamErr, nil
}

// gunzip copies SVC internal/middleware/uncompress.go: a bad gzip header
// fails before auth; a stream that breaks later fails at body read, after auth.
// ISIZE is sender-controlled, so it only rejects early: the decoded stream is
// read through a limit of maxReqSize+1 bytes, and the caller rejects a longer
// one. A broken stream returns no partial output.
func gunzip(raw []byte) ([]byte, *gwError, bool) {
	zr, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		return nil, &errUncompress, false
	}
	if isize := binary.LittleEndian.Uint32(raw[len(raw)-4:]); isize > maxReqSize {
		return nil, &errRequestBodyTooLarge, false
	}
	decoded, err := io.ReadAll(io.LimitReader(zr, maxReqSize+1))
	switch {
	case len(decoded) > maxReqSize:
		return decoded, nil, false
	case err != nil:
		return nil, nil, true
	}
	return decoded, nil, false
}

// authenticate copies the beacon interceptor (SVC handler_beacon.go) and
// writeKeyAuth (SVC auth.go:52-74). Every non-empty key is accepted: the
// oracle's static mode resolves any key to an enabled source.
func authenticate(r *http.Request, transport string) (string, *gwError) {
	if transport == "beacon" {
		writeKey := r.URL.Query().Get("writeKey")
		if writeKey == "" {
			return "", &errNoWriteKeyInQuery
		}
		return writeKey, nil
	}
	writeKey, _, ok := r.BasicAuth()
	if !ok || writeKey == "" {
		return "", &errNoWriteKeyInBasicAuth
	}
	return writeKey, nil
}

func (g *Gateway) process(r *http.Request, rep reply, reqType string, receivedAt time.Time) reply {
	payload := rep.decoded
	if !json.Valid(payload) {
		return rep.reject(errInvalidJSON)
	}

	messages, perr := splitEvents(payload, reqType)
	if perr != nil {
		return rep.reject(*perr)
	}

	events, containsAudienceList, failedIdx := g.enrichAll(r, messages, reqType, receivedAt)
	rep.events = events
	if failedIdx >= 0 {
		return rep.rejectEvent(errNonIdentifiable, failedIdx)
	}
	if len(payload) > maxReqSize && !containsAudienceList {
		return rep.reject(errRequestBodyTooLarge)
	}

	rep.status = http.StatusOK
	rep.body = []byte("ok")
	return rep
}

// enrichAll checks identity and enriches each event in order. Like the
// oracle it stops at the first non-identifiable event, whose index it returns
// (-1 when all pass); events after it keep a nil enrichedMessage.
func (g *Gateway) enrichAll(r *http.Request, messages []json.RawMessage, reqType string, receivedAt time.Time) ([]store.Event, bool, int) {
	events := make([]store.Event, len(messages))
	for i, msg := range messages {
		events[i] = newEvent(i, msg, nil)
	}

	var containsAudienceList bool
	for i, msg := range messages {
		ids := identityOf(msg)
		if ids.nonIdentifiable() {
			return events, containsAudienceList, i
		}
		containsAudienceList = containsAudienceList || ids.eventType == "audiencelist"

		// enrich cannot fail here: json.Valid and isObject already hold.
		enriched, _ := enrich(msg, enrichInput{
			reqType:    reqType,
			receivedAt: receivedAt,
			requestIP:  requestIP(r, msg),
			newUUID:    g.newUUID,
		})
		events[i] = newEvent(i, msg, enriched)
	}
	return events, containsAudienceList, -1
}

// splitEvents copies the batch and single-event shape checks
// (SVC handler.go:250-294).
func splitEvents(payload []byte, reqType string) ([]json.RawMessage, *gwError) {
	if reqType == "batch" || reqType == "import" {
		return splitBatch(payload)
	}
	if !isObject(payload) {
		return nil, &errNotRudderEvent
	}
	return []json.RawMessage{payload}, nil
}

func splitBatch(payload []byte) ([]json.RawMessage, *gwError) {
	batch, ok := batchOf(payload)
	switch {
	case !ok:
		return nil, &errNotRudderEvent
	case len(batch) == 0:
		return nil, &errEmptyBatch
	case len(batch) > maxEventsInBatch:
		return nil, &errTooManyEventsInBatch
	}
	return batch, nil
}

// batchOf returns the `batch` array when it exists and holds only objects.
func batchOf(payload []byte) ([]json.RawMessage, bool) {
	var root struct {
		Batch []json.RawMessage `json:"batch"`
	}
	if !isObject(payload) || json.Unmarshal(payload, &root) != nil || root.Batch == nil {
		return nil, false
	}
	for _, msg := range root.Batch {
		if !isObject(msg) {
			return nil, false
		}
	}
	return root.Batch, true
}

func isObject(b []byte) bool {
	b = bytes.TrimLeft(b, " \t\r\n")
	return len(b) > 0 && b[0] == '{'
}

type identity struct {
	userID, anonymousID, eventType string
}

func identityOf(msg json.RawMessage) identity {
	var m map[string]any
	_ = json.Unmarshal(msg, &m)
	return identity{
		userID:      sanitizeAndTrim(jsonString(m["userId"])),
		anonymousID: sanitizeAndTrim(jsonString(m["anonymousId"])),
		eventType:   jsonString(m["type"]),
	}
}

// nonIdentifiable copies SVC isNonIdentifiable (handler.go:570-577) with
// allowReqsWithoutUserIDAndAnonymousID at its default, false.
func (id identity) nonIdentifiable() bool {
	switch id.eventType {
	case "extract", "record":
		return false
	default:
		return id.userID == "" && id.anonymousID == ""
	}
}
