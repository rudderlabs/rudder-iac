package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten/internal/store"
)

// recordRoots are the record keys a requests fields path may start with.
var recordRoots = []string{
	"recordVersion", "serverId", "seq", "kind", "probe", "receivedAt", "route", "transport", "statusCode",
	"failed", "outcome", "writeKey", "sourceId", "rejection", "hint", "request", "response", "events",
}

// heavyFields are left out of a /requests item unless fields names them.
var heavyFields = []string{"request.body", "request.bodyBase64", "events.enrichedMessage"}

type requestsPage struct {
	APIVersion string            `json:"apiVersion"`
	ServerID   string            `json:"serverId"`
	Since      uint64            `json:"since"`
	Cursor     uint64            `json:"cursor"`
	HasMore    bool              `json:"hasMore"`
	TimedOut   bool              `json:"timedOut"`
	WaitedMs   int64             `json:"waitedMs"`
	Unfiltered unfiltered        `json:"unfiltered"`
	Omitted    *omitted          `json:"omitted"`
	Truncated  *truncated        `json:"truncated"`
	Requests   []json.RawMessage `json:"requests"`
}

type requestsQuery struct {
	filters
	kind   string
	failed *bool
	stage  string
	fields []string
}

func parseRequestsQuery(values map[string][]string) (requestsQuery, *apiError) {
	p, err := parseParams(values, "requests")
	if err != nil {
		return requestsQuery{}, err
	}
	q := requestsQuery{
		filters: parseFilters(p),
		kind:    p.oneOf("kind", "ingestion", "ingestion", "control", "all"),
		failed:  p.optBool("failed"),
		stage:   p.single("stage"),
		fields:  p.list("fields"),
	}
	if kept, bad := checkFields(q.fields, recordRoots, "request.body"); bad != "" {
		c := q.args(newCommand(routeCommands["requests"]).num("since", q.since), false)
		failFields(p, bad, recordRoots, c.fields(kept).flag("json").String())
	}
	return q, p.err
}

func (q requestsQuery) matches(rec store.Record) bool {
	return (q.kind == "all" || rec.Kind == q.kind) && q.filters.matchesRecord(rec) &&
		(q.failed == nil || rec.Failed == *q.failed) && q.matchesStage(rec)
}

func (q requestsQuery) matchesStage(rec store.Record) bool {
	return q.stage == "" || rec.Rejection != nil && rec.Rejection.Stage == q.stage
}

// args adds the caller's filters and, withShape, the output shape.
func (q requestsQuery) args(c *command, withShape bool) *command {
	q.filters.args(c)
	if q.kind != "ingestion" {
		c.quoted("kind", q.kind)
	}
	c.quoted("route", q.route...).quoted("status-code", intStrings(q.statusCode)...)
	if q.failed != nil {
		c.parts = append(c.parts, "--failed="+strconv.FormatBool(*q.failed))
	}
	if q.stage != "" {
		c.quoted("stage", q.stage)
	}
	if withShape {
		c.quoted("fields", q.fields...)
		q.filters.maxBytesArg(c)
	}
	return c
}

// requests answers /requests: one item per request, heavy attributes left
// out unless fields names them (contract section 4.3, last paragraph).
func (h *Handler) requests(w http.ResponseWriter, r *http.Request) {
	q, err := parseRequestsQuery(r.URL.Query())
	if err == nil {
		err = h.checkServerID(q.serverID)
	}
	if err != nil {
		writeError(w, *err)
		return
	}

	start := h.now()
	scanned, timedOut, ok, err := h.poll(r.Context(), q.filters, func(view store.View) scanResult {
		return scan(view, q.since, q.limit, func(rec store.Record) group {
			if !q.matches(rec) {
				return group{}
			}
			return group{seq: rec.Seq, items: []json.RawMessage{renderRecord(rec, q.fields)}}
		})
	})
	switch {
	case err != nil:
		writeError(w, *err)
	case ok:
		page := requestsPage{
			APIVersion: APIVersion, ServerID: h.id.ServerID, Since: q.since, TimedOut: timedOut,
			WaitedMs: h.now().Sub(start).Milliseconds(), Unfiltered: scanned.unfiltered,
		}
		writeRaw(w, http.StatusOK, fit(scanned, q.filters, func(groups []group, cursor uint64, hasMore bool, t *truncated) []byte {
			page.Cursor, page.HasMore, page.Truncated = cursor, hasMore, t
			page.Requests = flatten(groups)
			page.Omitted = q.omitted(groups)
			return encode(page)
		}, func(cursor uint64, oversized bool) string {
			c := newCommand(routeCommands["requests"])
			if oversized {
				return q.args(c.num("since", q.since), false).bare("fields", "request.headers").flag("json").String()
			}
			return q.args(c.num("since", cursor), true).flag("json").String()
		}))
	}
}

func (q requestsQuery) omitted(groups []group) *omitted {
	next := nextSummary
	if len(groups) > 0 {
		next = showCommand(groups[0].seq).flag("json").String()
	}
	fields := heavyFields
	if len(q.fields) > 0 {
		fields = []string{}
	}
	return &omitted{Fields: fields, Context: []string{}, Next: next}
}

const nextSummary = "rudder-cli dev summary --json"

func showCommand(seq uint64) *command {
	return newCommand(routeCommands["requests/{seq}"] + " " + strconv.FormatUint(seq, 10))
}

// requestItem is a record without its heavy attributes.
type requestItem struct {
	store.Record
	Request requestHead     `json:"request"`
	Events  []eventNoEnrich `json:"events"`
}

type requestHead struct {
	Method          string      `json:"method"`
	Target          string      `json:"target"`
	Headers         http.Header `json:"headers"`
	RedactedHeaders []string    `json:"redactedHeaders"`
	RemoteAddr      string      `json:"remoteAddr"`
	BodyEncoding    string      `json:"bodyEncoding"`
	BodyBytes       int         `json:"bodyBytes"`
	BodyComplete    bool        `json:"bodyComplete"`
}

type eventNoEnrich struct {
	Idx         int             `json:"idx"`
	Type        *string         `json:"type"`
	Event       *string         `json:"event"`
	UserID      *string         `json:"userId"`
	AnonymousID *string         `json:"anonymousId"`
	MessageID   *string         `json:"messageId"`
	Message     json.RawMessage `json:"message"`
}

func renderRecord(rec store.Record, fields []string) json.RawMessage {
	if len(fields) > 0 {
		return encode(projectJSON(encode(rec), fields, []string{"seq"}))
	}
	req := rec.Request
	item := requestItem{Record: rec, Request: requestHead{
		Method: req.Method, Target: req.Target, Headers: req.Headers, RedactedHeaders: req.RedactedHeaders,
		RemoteAddr: req.RemoteAddr, BodyEncoding: req.BodyEncoding, BodyBytes: req.BodyBytes,
		BodyComplete: req.BodyComplete,
	}, Events: make([]eventNoEnrich, len(rec.Events))}
	for i, ev := range rec.Events {
		item.Events[i] = eventNoEnrich{Idx: ev.Idx, Type: ev.Type, Event: ev.Event, UserID: ev.UserID,
			AnonymousID: ev.AnonymousID, MessageID: ev.MessageID, Message: ev.Message}
	}
	return encode(item)
}

// recordTruncated replaces a record larger than maxBytes.
type recordTruncated struct {
	Seq       uint64     `json:"seq"`
	Truncated *truncated `json:"truncated"`
}

// request answers /requests/{seq} with one whole record.
func (h *Handler) request(w http.ResponseWriter, r *http.Request) {
	q, err := h.parseRecordQuery(r)
	if err != nil {
		writeError(w, *err)
		return
	}
	rec, ok := h.store.Get(q.seq)
	if !ok {
		writeError(w, apiError{status: http.StatusNotFound, Code: "not_found",
			Message: "no request with seq " + strconv.FormatUint(q.seq, 10) +
				"; it was never captured or was removed by reset",
			Next: strp(nextInfo)})
		return
	}
	body := encode(rec)
	if len(q.fields) > 0 {
		body = encode(projectJSON(body, q.fields, []string{"seq"}))
	}
	if q.maxBytes > 0 && len(body) > q.maxBytes {
		body = encode(recordTruncated{Seq: q.seq, Truncated: &truncated{By: "maxBytes", RequestBytes: len(body),
			Next: showCommand(q.seq).bare("fields", "request.headers").bare("fields", "response").flag("json").String()}})
	}
	writeRaw(w, http.StatusOK, body)
}

type recordQuery struct {
	seq      uint64
	fields   []string
	maxBytes int
}

func (h *Handler) parseRecordQuery(r *http.Request) (recordQuery, *apiError) {
	p, err := parseParams(r.URL.Query(), "requests/{seq}")
	if err != nil {
		return recordQuery{}, err
	}
	raw := strings.TrimPrefix(r.URL.Path, base+"requests/")
	seq, convErr := strconv.ParseUint(raw, 10, 64)
	if convErr != nil {
		p.fail("seq", "%q is not a request seq", raw)
	}
	q := recordQuery{seq: seq, fields: p.list("fields"), maxBytes: p.maxBytes()}
	if kept, bad := checkFields(q.fields, recordRoots, "request.body"); bad != "" {
		failFields(p, bad, recordRoots, showCommand(seq).fields(kept).flag("json").String())
	}
	if p.err != nil {
		return q, p.err
	}
	return q, h.checkServerID(p.single("serverId"))
}
