package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten/internal/store"
)

// recordRoots are the record keys a requests fields path may start with.
var recordRoots = []string{
	"recordVersion", "serverId", "seq", "kind", "receivedAt", "route", "transport", "statusCode",
	"failed", "outcome", "writeKey", "writeKeyPrefix", "writeKeySuffix", "writeKeySha256", "sourceId", "rejection", "hint", "request", "response", "events",
}

// compactOmitted are the record parts the compact view leaves out.
var compactOmitted = []string{"request.headers", "request.body", "response", "events.message", "events.enrichedMessage"}

type requestsPage struct {
	APIVersion string `json:"apiVersion"`
	ServerID   string `json:"serverId"`
	Since      uint64 `json:"since"`
	Cursor     uint64 `json:"cursor"`
	// EvictedThrough above since means the store dropped part of the window.
	EvictedThrough uint64            `json:"evictedThrough"`
	HasMore        bool              `json:"hasMore"`
	TimedOut       bool              `json:"timedOut"`
	WaitedMs       int64             `json:"waitedMs"`
	Unfiltered     unfiltered        `json:"unfiltered"`
	View           string            `json:"view"`
	Omitted        *omitted          `json:"omitted"`
	Truncated      *truncated        `json:"truncated"`
	Next           string            `json:"next"`
	Links          pageLinks         `json:"links"`
	Requests       []json.RawMessage `json:"requests"`
}

type requestsQuery struct {
	filters
	kind     string
	failed   *bool
	view     string
	sentView string
	fields   []string
}

func parseRequestsQuery(values map[string][]string) (requestsQuery, *apiError) {
	p, err := parseParams(values, "requests")
	if err != nil {
		return requestsQuery{}, err
	}
	q := requestsQuery{
		filters:  parseFilters(p),
		kind:     p.oneOf("kind", "ingestion", "ingestion", "control", "all"),
		failed:   p.optBool("failed"),
		view:     p.oneOf("view", viewList, viewList, viewCompact, viewFull),
		sentView: p.single("view"),
		fields:   p.list("fields"),
	}
	if len(q.fields) == 0 {
		return q, p.err
	}
	kept, bad := checkFields(q.fields, recordRoots, "request.body")
	fixed := q.args(newCommand(routeCommands["requests"]).num("since", q.since), false).fields(kept).flag("json")
	switch {
	case bad != "":
		failFields(p, bad, recordRoots, fixed.String())
	case q.sentView != "":
		p.failWith("fields", fixed.String(), nil, "cannot combine with view; fields replaces the view")
	}
	q.view = viewFields
	return q, p.err
}

func (q requestsQuery) matches(rec store.Record) bool {
	return (q.kind == "all" || rec.Kind == q.kind) && q.filters.matchesRecord(rec) &&
		(q.failed == nil || rec.Failed == *q.failed)
}

// args adds the caller's filters and, withShape, the output shape.
func (q requestsQuery) args(c *command, withShape bool) *command {
	q.filters.args(c)
	if q.kind != "ingestion" {
		c.quoted("kind", q.kind)
	}
	c.quoted("status-code", q.statusCode...)
	if q.failed != nil {
		c.parts = append(c.parts, "--failed="+strconv.FormatBool(*q.failed))
	}
	if withShape {
		if q.sentView != "" {
			c.quoted("view", q.sentView)
		}
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
		q.resolveSince(h.store)
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
			return group{seq: rec.Seq, items: []json.RawMessage{renderRecord(rec, q.view, q.fields)}}
		})
	})
	switch {
	case err != nil:
		writeError(w, *err)
	case ok:
		page := requestsPage{
			APIVersion: APIVersion, ServerID: h.id.ServerID, Since: q.since, TimedOut: timedOut,
			WaitedMs: h.now().Sub(start).Milliseconds(), Unfiltered: scanned.unfiltered, View: q.view,
			EvictedThrough: scanned.evictedThrough,
		}
		body := fit(scanned, q.filters, func(groups []group, cursor uint64, hasMore bool, t *truncated) []byte {
			page.Cursor, page.HasMore, page.Truncated = cursor, hasMore, t
			page.Requests = flatten(groups)
			page.Omitted = q.omitted(groups)
			page.Next = q.args(newCommand(routeCommands["requests"]).num("since", cursor), true).flag("json").String()
			page.Links = pageLinks{Next: nextURL("requests", r.URL.Query(), cursor)}
			return encode(page)
		}, func(cursor uint64, oversized bool) string {
			c := newCommand(routeCommands["requests"])
			if oversized {
				return q.args(c.num("since", q.since), false).bare("fields", "request.headers").flag("json").String()
			}
			return q.args(c.num("since", cursor), true).flag("json").String()
		})
		linkNext(w, page.HasMore, page.Links.Next)
		writeRaw(w, http.StatusOK, body)
	}
}

func (q requestsQuery) omitted(groups []group) *omitted {
	show := nextSummary
	if len(groups) > 0 {
		show = showCommand(groups[0].seq).bare("fields", "request.body").flag("json").String()
	}
	switch q.view {
	case viewFull:
		return nil
	case viewList:
		c := q.args(newCommand(routeCommands["requests"]).num("since", q.since), false)
		return &omitted{Fields: []string{"receivedAt", "rejection", "events"}, Context: []string{},
			Next: c.bare("view", viewCompact).flag("json").String()}
	case viewFields:
		return &omitted{Fields: []string{}, Context: []string{}, Next: show}
	}
	return &omitted{Fields: compactOmitted, Context: []string{}, Next: show}
}

const nextSummary = "rudder-cli dev events list --json"

func showCommand(seq uint64) *command {
	return newCommand(routeCommands["requests/{seq}"] + " " + strconv.FormatUint(seq, 10))
}

// requestCompact is one line per request: what happened, and the names of
// its events. method tells a preflight from the request it precedes.
type requestCompact struct {
	Seq        uint64           `json:"seq"`
	ReceivedAt time.Time        `json:"receivedAt"`
	Method     string           `json:"method"`
	Route      string           `json:"route"`
	StatusCode int              `json:"statusCode"`
	Outcome    string           `json:"outcome"`
	Kind       string           `json:"kind"`
	Rejection  *store.Rejection `json:"rejection"`
	Events     []eventName      `json:"events"`
}

type eventName struct {
	Idx   int     `json:"idx"`
	Type  *string `json:"type"`
	Event *string `json:"event"`
}

type requestListItem struct {
	Seq        uint64 `json:"seq"`
	Kind       string `json:"kind"`
	Method     string `json:"method"`
	Route      string `json:"route"`
	StatusCode int    `json:"statusCode"`
	Outcome    string `json:"outcome"`
	EventCount int    `json:"eventCount"`
}

// recordKeep are the paths every requests projection keeps.
var recordKeep = []string{"seq", "request.method"}

func renderRecord(rec store.Record, view string, fields []string) json.RawMessage {
	switch view {
	case viewFields:
		return encode(projectJSON(encode(rec), fields, recordKeep))
	case viewFull:
		return encode(rec)
	case viewList:
		return encode(requestListItem{Seq: rec.Seq, Kind: rec.Kind, Method: rec.Request.Method, Route: rec.Route,
			StatusCode: rec.StatusCode, Outcome: rec.Outcome, EventCount: len(rec.Events)})
	}
	item := requestCompact{Seq: rec.Seq, ReceivedAt: rec.ReceivedAt, Method: rec.Request.Method, Route: rec.Route,
		StatusCode: rec.StatusCode, Outcome: rec.Outcome, Kind: rec.Kind, Rejection: rec.Rejection,
		Events: make([]eventName, len(rec.Events))}
	for i, ev := range rec.Events {
		item.Events[i] = eventName{Idx: ev.Idx, Type: ev.Type, Event: ev.Event}
	}
	return dropNulls(encode(item))
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
				"; it was never captured or the store evicted it",
			Next: strp(nextSummary)})
		return
	}
	body := renderShow(rec, q)
	if q.maxBytes > 0 && len(body) > q.maxBytes {
		body = encode(recordTruncated{Seq: q.seq, Truncated: &truncated{By: "maxBytes", RequestBytes: len(body),
			Next: showCommand(q.seq).bare("fields", "request.headers").bare("fields", "response").flag("json").String()}})
	}
	writeRaw(w, http.StatusOK, body)
}

type recordQuery struct {
	seq      uint64
	view     string
	fields   []string
	maxBytes int
}

// recordShow is a record whose events carry their scalars only: the body
// already holds each message.
type recordShow struct {
	store.Record
	Events  []eventScalars `json:"events"`
	Omitted *omitted       `json:"omitted"`
}

type eventScalars struct {
	Idx         int     `json:"idx"`
	Type        *string `json:"type"`
	Event       *string `json:"event"`
	UserID      *string `json:"userId"`
	AnonymousID *string `json:"anonymousId"`
	MessageID   *string `json:"messageId"`
}

func renderShow(rec store.Record, q recordQuery) []byte {
	switch {
	case len(q.fields) > 0:
		return encode(projectJSON(encode(rec), q.fields, recordKeep))
	case q.view == viewFull:
		return encode(rec)
	}
	show := recordShow{Record: rec, Events: make([]eventScalars, len(rec.Events)), Omitted: &omitted{
		Fields: []string{"events.message", "events.enrichedMessage"}, Context: []string{},
		Next: showCommand(rec.Seq).bare("fields", "request.body").flag("json").String(),
	}}
	for i, ev := range rec.Events {
		show.Events[i] = eventScalars{Idx: ev.Idx, Type: ev.Type, Event: ev.Event, UserID: ev.UserID,
			AnonymousID: ev.AnonymousID, MessageID: ev.MessageID}
	}
	return encode(show)
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
	q := recordQuery{seq: seq, fields: p.list("fields"), maxBytes: p.maxBytes(),
		view: p.oneOf("view", viewCompact, viewCompact, viewFull)}
	if kept, bad := checkFields(q.fields, recordRoots, "request.body"); bad != "" {
		failFields(p, bad, recordRoots, showCommand(seq).fields(kept).flag("json").String())
	}
	if p.err != nil {
		return q, p.err
	}
	return q, h.checkServerID(p.single("serverId"))
}
