package api

import (
	"encoding/json"
	"net/http"
	"slices"

	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten/internal/store"
)

const (
	viewSummary = "summary"
	viewCompact = "compact"
	viewFull    = "full"
	viewFields  = "fields"

	includeContext    = "context"
	includeEnrichment = "enrichment"
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
	include  []string
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
		view:     p.oneOf("view", viewCompact, viewSummary, viewCompact, viewFull),
		fields:   p.list("fields"),
		include:  p.list("include"),
		event:    p.list("event"),
		typ:      p.list("type"),
		userID:   p.single("userId"),
		anonID:   p.single("anonymousId"),
	}
	q.checkShape(p)
	return q, p.err
}

// checkShape validates include and fields; fields replaces the view.
func (q *eventsQuery) checkShape(p *params) {
	for _, inc := range q.include {
		if inc != includeContext && inc != includeEnrichment {
			p.fail("include", "%q is not context or enrichment", inc)
		}
	}
	if len(q.fields) == 0 {
		return
	}
	checkFields(p, q.fields, eventRoots)
	switch {
	case q.sentView != "":
		p.fail("fields", "cannot combine with view; use one of them")
	case len(q.include) > 0:
		p.fail("include", "cannot combine with fields; name the paths in fields")
	}
	q.view = viewFields
}

func (q eventsQuery) includes(name string) bool { return slices.Contains(q.include, name) }

// args adds the caller's filters and output shape to a next command.
func (q eventsQuery) args(c *command, withShape bool) *command {
	q.filters.args(c)
	c.quoted("event", q.event...).quoted("type", q.typ...).quoted("route", q.route...).
		quoted("status-code", intStrings(q.statusCode)...)
	if q.userID != "" {
		c.quoted("user-id", q.userID)
	}
	if q.anonID != "" {
		c.quoted("anonymous-id", q.anonID)
	}
	if withShape {
		if q.sentView != "" {
			c.quoted("view", q.sentView)
		}
		c.quoted("fields", q.fields...)
		q.filters.maxBytesArg(c)
	}
	return c.quoted("include", q.include...)
}

func (q eventsQuery) matches(rec store.Record, ev store.Event) bool {
	return matches(ev.Event, q.event) && matches(ev.Type, q.typ) && q.filters.matchesRecord(rec) &&
		equals(ev.UserID, q.userID) && equals(ev.AnonymousID, q.anonID)
}

// matches ORs the values of one repeatable filter; no values match all.
func matches(field *string, want []string) bool {
	return len(want) == 0 || field != nil && slices.Contains(want, *field)
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

type eventsPage struct {
	APIVersion string            `json:"apiVersion"`
	ServerID   string            `json:"serverId"`
	Since      uint64            `json:"since"`
	Cursor     uint64            `json:"cursor"`
	HasMore    bool              `json:"hasMore"`
	TimedOut   bool              `json:"timedOut"`
	WaitedMs   int64             `json:"waitedMs"`
	Unfiltered unfiltered        `json:"unfiltered"`
	View       string            `json:"view"`
	Omitted    *omitted          `json:"omitted"`
	Truncated  *truncated        `json:"truncated"`
	Events     []json.RawMessage `json:"events"`
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
	scanned, timedOut, ok, err := h.poll(r.Context(), q.filters, func(view store.View) scanResult {
		return scan(view, q.since, q.limit, func(rec store.Record) group { return h.eventGroup(rec, q) })
	})
	switch {
	case err != nil:
		writeError(w, *err)
	case ok:
		page := eventsPage{
			APIVersion: APIVersion, ServerID: h.id.ServerID, Since: q.since, TimedOut: timedOut,
			WaitedMs: h.now().Sub(start).Milliseconds(), Unfiltered: scanned.unfiltered, View: q.view,
			Events: []json.RawMessage{},
		}
		writeRaw(w, http.StatusOK, fit(scanned, q.filters, func(groups []group, cursor uint64, hasMore bool, t *truncated) []byte {
			page.Cursor, page.HasMore, page.Truncated = cursor, hasMore, t
			page.Events = flatten(groups)
			page.Omitted = q.omitted(groups)
			return encode(page)
		}, q.truncatedNext))
	}
}

func (q eventsQuery) truncatedNext(cursor uint64, oversized bool) string {
	c := newCommand(routeCommands["events"])
	if oversized {
		return q.args(c.num("since", q.since), false).bare("fields", "properties").flag("json").String()
	}
	return q.args(c.num("since", cursor), true).flag("json").String()
}

func (q eventsQuery) omitted(groups []group) *omitted {
	c := q.args(newCommand(routeCommands["events"]).num("since", q.since), false)
	switch q.view {
	case viewFull:
		return nil
	case viewSummary:
		return &omitted{Fields: []string{"properties", "traits", "context", "message", "enrichedMessage"},
			Context: []string{}, Next: c.flag("json").String()}
	case viewFields:
		return &omitted{Fields: []string{}, Context: []string{}, Next: c.bare("view", viewFull).flag("json").String()}
	}
	stripped := strippedKeys(groups)
	if len(stripped) > 0 {
		c.bare("include", includeContext)
	} else {
		c.bare("view", viewFull)
	}
	return &omitted{Fields: []string{"message", "enrichedMessage"}, Context: stripped, Next: c.flag("json").String()}
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
		Next:    strp(nextInfo)}
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
	case viewSummary:
		return encode(summaryItem{Seq: base.Seq, Idx: base.Idx, Type: base.Type, Event: base.Event,
			UserID: base.UserID, StatusCode: base.StatusCode}), nil
	case viewFull:
		return encode(fullItem{baseItem: base, Message: ev.Message, EnrichedMessage: ev.EnrichedMessage}), nil
	case viewFields:
		full := encode(fullItem{baseItem: base, Message: ev.Message, EnrichedMessage: ev.EnrichedMessage})
		return encode(projectJSON(full, q.fields, []string{"seq", "idx", "type", "event"})), nil
	}
	item := compactItem{baseItem: base}
	var stripped []string
	item.Context, stripped = compactContext(ev.Message, q.includes(includeContext))
	if q.includes(includeEnrichment) {
		item.RequestIP, item.RudderID = enrichment(ev.EnrichedMessage)
	}
	return encode(item), stripped
}
