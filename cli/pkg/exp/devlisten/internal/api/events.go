package api

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten/internal/store"
)

const (
	viewCounts  = "counts"
	viewList    = "list"
	viewCompact = "compact"
	viewFull    = "full"
	viewFields  = "fields"
)

// eventRoots are the item keys a fields path may start with.
var eventRoots = []string{
	"seq", "idx", "receivedAt", "route", "transport", "statusCode", "outcome", "writeKey", "type", "event",
	"userId", "anonymousId", "messageId", "sentAt", "originalTimestamp", "properties", "traits", "message",
	"enrichedMessage",
}

type eventsQuery struct {
	filters
	view     string
	fields   []string
	sentView string
	event    []string
	typ      []string
	userID   string
	anonID   string
}

func parseEventsQuery(values map[string][]string) (eventsQuery, *apiError) {
	p, err := parseParams(values, "events")
	if err != nil {
		return eventsQuery{}, err
	}
	q := eventsQuery{
		filters:  parseFilters(p),
		sentView: p.single("view"),
		view:     p.oneOf("view", viewList, viewCounts, viewList, viewCompact, viewFull),
		fields:   p.list("fields"),
		event:    p.list("event"),
		typ:      p.list("type"),
		userID:   p.single("userId"),
		anonID:   p.single("anonymousId"),
	}
	q.checkShape(p)
	return q, p.err
}

// checkShape validates fields; fields replaces the view. A conflict names
// the corrected command in next.
func (q *eventsQuery) checkShape(p *params) {
	if len(q.fields) == 0 {
		return
	}
	kept, bad := checkFields(q.fields, eventRoots, "properties")
	switch {
	case bad != "":
		failFields(p, bad, eventRoots, q.fixed(kept))
	case q.sentView != "":
		p.failWith("fields", q.fixed(kept), nil, "cannot combine with view; fields replaces the view")
	}
	q.view = viewFields
}

// fixed is the caller's command with fields as the only shape.
func (q eventsQuery) fixed(fields []fieldArg) string {
	c := q.filterArgs(newCommand(routeCommands["events"]).num("since", q.since))
	return c.fields(fields).flag("json").String()
}

// filterArgs adds the caller's filters to a next command.
func (q eventsQuery) filterArgs(c *command) *command {
	q.filters.args(c)
	c.quoted("event", q.event...).quoted("type", q.typ...).quoted("status-code", q.statusCode...)
	if q.userID != "" {
		c.quoted("user-id", q.userID)
	}
	if q.anonID != "" {
		c.quoted("anonymous-id", q.anonID)
	}
	return c
}

// args adds the caller's filters and output shape to a next command.
func (q eventsQuery) args(c *command, withShape bool) *command {
	q.filterArgs(c)
	if withShape {
		if q.sentView != "" {
			c.quoted("view", q.sentView)
		}
		c.quoted("fields", q.fields...)
		q.filters.maxBytesArg(c)
	}
	return c
}

// matchesEvent applies the event-level filters.
func (q eventsQuery) matchesEvent(ev store.Event) bool {
	return matches(ev.Event, q.event) && matches(ev.Type, q.typ) &&
		equals(ev.UserID, q.userID) && equals(ev.AnonymousID, q.anonID)
}

func (q eventsQuery) matches(rec store.Record, ev store.Event) bool {
	return q.filters.matchesRecord(rec) && q.matchesEvent(ev)
}

// matches ORs the values of one repeatable filter; no values match all. A
// value that ends in * matches every name with that prefix.
func matches(field *string, want []string) bool {
	return len(want) == 0 || field != nil && slices.ContainsFunc(want, func(w string) bool {
		if prefix, ok := strings.CutSuffix(w, "*"); ok {
			return strings.HasPrefix(*field, prefix)
		}
		return w == *field
	})
}

func equals(field *string, want string) bool {
	return want == "" || field != nil && *field == want
}

type unfiltered struct {
	Requests       int `json:"requests"`
	FailedRequests int `json:"failedRequests"`
	Events         int `json:"events"`
	Control        int `json:"control"`
}

type omitted struct {
	Fields  []string `json:"fields"`
	Context []string `json:"context"`
	Next    string   `json:"next"`
}

// eventsPage is the one /events envelope. Its keys never change with the
// view: counts leaves events empty, the other views fill it.
type eventsPage struct {
	APIVersion string `json:"apiVersion"`
	ServerID   string `json:"serverId"`
	Since      uint64 `json:"since"`
	Cursor     uint64 `json:"cursor"`
	// EvictedThrough above since means the store dropped part of the window.
	EvictedThrough uint64     `json:"evictedThrough"`
	HasMore        bool       `json:"hasMore"`
	TimedOut       bool       `json:"timedOut"`
	WaitedMs       int64      `json:"waitedMs"`
	Summary        summary    `json:"summary"`
	View           string     `json:"view"`
	Omitted        *omitted   `json:"omitted"`
	Truncated      *truncated `json:"truncated"`
	// Next is the CLI command that continues after this answer; Links.Next
	// is the same call as a URL.
	Next   string            `json:"next"`
	Links  pageLinks         `json:"links"`
	Events []json.RawMessage `json:"events"`
}

// events answers /events, long-polling until min matches exist, the wait
// ends, the client leaves or the server stops (contract section 4.5).
func (h *Handler) events(w http.ResponseWriter, r *http.Request) {
	q, err := parseEventsQuery(r.URL.Query())
	if err == nil {
		err = h.checkServerID(q.serverID)
	}
	if err != nil {
		writeError(w, *err)
		return
	}

	start := h.now()
	var window store.View
	scanned, timedOut, ok, err := h.poll(r.Context(), q.filters, func(view store.View) scanResult {
		window = view
		return scan(view, q.since, q.limit, func(rec store.Record) group { return h.eventGroup(rec, q) })
	})
	switch {
	case err != nil:
		writeError(w, *err)
	case ok:
		page := eventsPage{
			APIVersion: APIVersion, ServerID: h.id.ServerID, Since: q.since, TimedOut: timedOut,
			WaitedMs: h.now().Sub(start).Milliseconds(), Summary: h.summarize(window.Records, q), View: q.view,
			EvictedThrough: scanned.evictedThrough, Events: []json.RawMessage{},
		}
		continueAt := func(cursor uint64) {
			page.Next = q.args(newCommand(routeCommands["events"]).num("since", cursor), true).flag("json").String()
			page.Links = pageLinks{Next: nextURL("events", r.URL.Query(), cursor)}
		}
		if q.view == viewCounts {
			page.Cursor = max(q.since, window.Cursor)
			page.Omitted = q.omitted(nil)
			continueAt(page.Cursor)
			writeRaw(w, http.StatusOK, encode(page))
			return
		}
		body := fit(scanned, q.filters, func(groups []group, cursor uint64, hasMore bool, t *truncated) []byte {
			page.Cursor, page.HasMore, page.Truncated = cursor, hasMore, t
			page.Events = flatten(groups)
			page.Omitted = q.omitted(groups)
			continueAt(cursor)
			return encode(page)
		}, q.truncatedNext)
		linkNext(w, page.HasMore, page.Links.Next)
		writeRaw(w, http.StatusOK, body)
	}
}

func (q eventsQuery) truncatedNext(cursor uint64, oversized bool) string {
	c := newCommand(routeCommands["events"])
	if oversized {
		return q.args(c.num("since", q.since), false).bare("fields", "properties").flag("json").String()
	}
	return q.args(c.num("since", cursor), true).flag("json").String()
}

// omitted names what the view left out and the next rung of the ladder:
// counts, list, compact, fields, then one whole request.
func (q eventsQuery) omitted(groups []group) *omitted {
	c := q.args(newCommand(routeCommands["events"]).num("since", q.since), false)
	switch q.view {
	case viewFull:
		return nil
	case viewCounts:
		return &omitted{Fields: []string{"events"}, Context: []string{}, Next: c.flag("json").String()}
	case viewList:
		return &omitted{Fields: []string{"properties", "traits", "context", "message", "enrichedMessage"},
			Context: []string{}, Next: c.bare("view", viewCompact).flag("json").String()}
	case viewFields:
		next := c.bare("view", viewFull).flag("json").String()
		if len(groups) > 0 {
			next = showCommand(groups[0].seq).bare("fields", "request.body").flag("json").String()
		}
		return &omitted{Fields: []string{}, Context: []string{}, Next: next}
	}
	return &omitted{Fields: []string{"message", "enrichedMessage"}, Context: strippedKeys(groups),
		Next: c.bare("fields", "properties").flag("json").String()}
}

func strippedKeys(groups []group) []string {
	keys := []string{}
	for _, g := range groups {
		for _, k := range g.stripped {
			if !slices.Contains(keys, k) {
				keys = append(keys, k)
			}
		}
	}
	slices.Sort(keys)
	return keys
}

func (h *Handler) checkServerID(serverID string) *apiError {
	if serverID == "" || serverID == h.id.ServerID {
		return nil
	}
	return &apiError{status: http.StatusConflict, Code: "server_changed",
		Message: "serverId " + serverID + " is not the running server; repeat the action with a fresh cursor",
		Details: map[string]any{"serverId": h.id.ServerID, "startedAt": h.id.StartedAt},
		Next:    strp(nextFresh)}
}

func countRecords(records []store.Record) unfiltered {
	var u unfiltered
	for _, rec := range records {
		if rec.Kind == "control" {
			u.Control++
			continue
		}
		u.Requests++
		u.Events += len(rec.Events)
		if rec.Failed {
			u.FailedRequests++
		}
	}
	return u
}

// eventGroup renders the matching events of one request in the query view.
func (h *Handler) eventGroup(rec store.Record, q eventsQuery) group {
	g := group{seq: rec.Seq}
	for _, ev := range rec.Events {
		if !q.matches(rec, ev) {
			continue
		}
		item, stripped := renderEvent(rec, ev, q)
		g.items = append(g.items, item)
		g.stripped = append(g.stripped, stripped...)
	}
	return g
}

func renderEvent(rec store.Record, ev store.Event, q eventsQuery) (json.RawMessage, []string) {
	base := newBaseItem(rec, ev)
	switch q.view {
	case viewCounts, viewList:
		return encode(listItem{Seq: base.Seq, Idx: base.Idx, ReceivedAt: base.ReceivedAt, Type: base.Type,
			Event: base.Event, UserID: base.UserID, WriteKey: base.WriteKey, StatusCode: base.StatusCode}), nil
	case viewFull:
		return encode(fullItem{baseItem: base, Message: ev.Message, EnrichedMessage: ev.EnrichedMessage}), nil
	case viewFields:
		full := encode(fullItem{baseItem: base, Message: ev.Message, EnrichedMessage: ev.EnrichedMessage})
		return encode(projectJSON(full, q.fields, []string{"seq", "idx", "type", "event"})), nil
	}
	item := compactItem{baseItem: base}
	var stripped []string
	item.Context, stripped = compactContext(ev.Message)
	return dropNulls(encode(item)), stripped
}
