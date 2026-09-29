package api

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"time"

	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten/internal/store"
)

type eventsQuery struct {
	since    uint64
	serverID string
	limit    int
	min      int
	wait     time.Duration
	event    []string
	typ      []string
}

func parseEventsQuery(values map[string][]string) (eventsQuery, *apiError) {
	p, err := parseParams(values, "since", "serverId", "limit", "min", "wait", "event", "type")
	if err != nil {
		return eventsQuery{}, err
	}
	q := eventsQuery{
		since:    p.uint("since", 0),
		serverID: p.single("serverId"),
		limit:    p.intIn("limit", 100, 1, 1000),
		wait:     p.wait(),
		event:    p.list("event"),
		typ:      p.list("type"),
	}
	q.min = p.intIn("min", 1, 1, q.limit)
	return q, p.err
}

// answered is true when a page may be returned before the wait ends: a
// snapshot (wait 0) always is.
func (q eventsQuery) answered(page eventsPage) bool {
	return q.wait == 0 || len(page.Events) >= q.min
}

type unfiltered struct {
	Requests       int `json:"requests"`
	FailedRequests int `json:"failedRequests"`
	Events         int `json:"events"`
	Control        int `json:"control"`
}

type eventsPage struct {
	APIVersion string      `json:"apiVersion"`
	ServerID   string      `json:"serverId"`
	Since      uint64      `json:"since"`
	Cursor     uint64      `json:"cursor"`
	HasMore    bool        `json:"hasMore"`
	TimedOut   bool        `json:"timedOut"`
	WaitedMs   int64       `json:"waitedMs"`
	Unfiltered unfiltered  `json:"unfiltered"`
	Events     []eventItem `json:"events"`
}

type eventItem struct {
	Seq               uint64          `json:"seq"`
	Idx               int             `json:"idx"`
	ReceivedAt        time.Time       `json:"receivedAt"`
	Route             string          `json:"route"`
	Transport         string          `json:"transport"`
	StatusCode        int             `json:"statusCode"`
	Outcome           string          `json:"outcome"`
	WriteKey          string          `json:"writeKey"`
	Type              *string         `json:"type"`
	Event             *string         `json:"event"`
	UserID            *string         `json:"userId"`
	AnonymousID       *string         `json:"anonymousId"`
	MessageID         *string         `json:"messageId"`
	SentAt            *string         `json:"sentAt"`
	OriginalTimestamp *string         `json:"originalTimestamp"`
	Properties        json.RawMessage `json:"properties"`
	Traits            json.RawMessage `json:"traits"`
	Message           json.RawMessage `json:"message"`
	EnrichedMessage   json.RawMessage `json:"enrichedMessage"`
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
	page, ok, err := h.waitForPage(r.Context(), q)
	switch {
	case err != nil:
		writeError(w, *err)
	case ok:
		page.WaitedMs = h.now().Sub(start).Milliseconds()
		writeJSON(w, http.StatusOK, page)
	}
}

// waitForPage returns ok=false when the client left before an answer.
func (h *Handler) waitForPage(ctx context.Context, q eventsQuery) (eventsPage, bool, *apiError) {
	timer := time.NewTimer(q.wait)
	defer timer.Stop()
	for {
		view := h.store.Since(q.since)
		page := h.scanEvents(view, q)
		if q.answered(page) {
			return page, true, nil
		}
		select {
		case <-view.Changed:
		case <-timer.C:
			page = h.scanEvents(h.store.Since(q.since), q)
			page.TimedOut = len(page.Events) < q.min
			return page, true, nil
		case <-ctx.Done():
			return eventsPage{}, false, nil
		case <-h.store.Done():
			return eventsPage{}, false, &errShuttingDown
		}
	}
}

func (h *Handler) checkServerID(serverID string) *apiError {
	if serverID == "" || serverID == h.id.ServerID {
		return nil
	}
	return &apiError{status: http.StatusConflict, Code: "server_changed",
		Message: "serverId " + serverID + " is not the running server",
		Details: map[string]any{"serverId": h.id.ServerID, "startedAt": h.id.StartedAt}}
}

// scanEvents builds one page under contract section 4.4: it never splits a
// request, and the cursor never skips or repeats a request.
func (h *Handler) scanEvents(view store.View, q eventsQuery) eventsPage {
	page := eventsPage{
		APIVersion: APIVersion,
		ServerID:   h.id.ServerID,
		Since:      q.since,
		Cursor:     max(q.since, view.Cursor),
		Unfiltered: countRecords(view.Records),
		Events:     []eventItem{},
	}
	for i, rec := range view.Records {
		items := matchingItems(rec, q)
		if len(items) > 0 && len(page.Events) >= q.limit {
			page.HasMore = true
			page.Cursor = view.Records[i-1].Seq
			break
		}
		page.Events = append(page.Events, items...)
	}
	return page
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

func matchingItems(rec store.Record, q eventsQuery) []eventItem {
	var items []eventItem
	for _, ev := range rec.Events {
		if matches(ev.Event, q.event) && matches(ev.Type, q.typ) {
			items = append(items, newEventItem(rec, ev))
		}
	}
	return items
}

// matches ORs the values of one repeatable filter; no values match all.
func matches(field *string, want []string) bool {
	return len(want) == 0 || field != nil && slices.Contains(want, *field)
}

func newEventItem(rec store.Record, ev store.Event) eventItem {
	var msg struct {
		SentAt            json.RawMessage `json:"sentAt"`
		OriginalTimestamp json.RawMessage `json:"originalTimestamp"`
		Properties        json.RawMessage `json:"properties"`
		Traits            json.RawMessage `json:"traits"`
		Context           struct {
			Traits json.RawMessage `json:"traits"`
		} `json:"context"`
	}
	_ = json.Unmarshal(ev.Message, &msg)

	traits := msg.Traits
	if isNull(traits) {
		traits = msg.Context.Traits
	}
	sentAt := msg.SentAt
	if isNull(sentAt) {
		sentAt = batchSentAt(rec.Request.Body)
	}

	return eventItem{
		Seq: rec.Seq, Idx: ev.Idx, ReceivedAt: rec.ReceivedAt, Route: rec.Route, Transport: rec.Transport,
		StatusCode: rec.StatusCode, Outcome: rec.Outcome, WriteKey: rec.WriteKey,
		Type: ev.Type, Event: ev.Event, UserID: ev.UserID, AnonymousID: ev.AnonymousID, MessageID: ev.MessageID,
		SentAt:            rawString(sentAt),
		OriginalTimestamp: rawString(msg.OriginalTimestamp),
		Properties:        nullIfEmpty(msg.Properties),
		Traits:            nullIfEmpty(traits),
		Message:           ev.Message,
		EnrichedMessage:   ev.EnrichedMessage,
	}
}

// batchSentAt reads the batch-level sentAt that the Node SDK sends instead
// of a per-event one.
func batchSentAt(body string) json.RawMessage {
	var root struct {
		SentAt json.RawMessage `json:"sentAt"`
	}
	_ = json.Unmarshal([]byte(body), &root)
	return root.SentAt
}

func isNull(raw json.RawMessage) bool {
	return len(raw) == 0 || string(raw) == "null"
}

func nullIfEmpty(raw json.RawMessage) json.RawMessage {
	if isNull(raw) {
		return nil
	}
	return raw
}

func rawString(raw json.RawMessage) *string {
	if isNull(raw) {
		return nil
	}
	var s string
	if json.Unmarshal(raw, &s) != nil {
		s = string(raw)
	}
	return &s
}
