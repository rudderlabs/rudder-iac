package api

import (
	"path"
	"slices"
	"strings"
	"time"

	"github.com/tidwall/gjson"

	"github.com/rudderlabs/rudder-iac/cli/internal/devlisten/store"
)

const (
	outcomeAccepted = "accepted"
	outcomeRejected = "rejected"
	// outcomeDropped is a pixel request: it answers the GIF with 200 even
	// when its event fails.
	outcomeDropped = "dropped"
)

func outcome(r *store.Record) string {
	switch {
	case r.Response.StatusCode < 200 || r.Response.StatusCode > 299:
		return outcomeRejected
	case r.Rejection != nil:
		return outcomeDropped
	}
	return outcomeAccepted
}

func accepted(r *store.Record) bool {
	return r.Kind == "ingestion" && outcome(r) == outcomeAccepted
}

// eventKeys are the top-level keys the filters and counts read. The last of
// duplicate keys wins, as in RudderStack.
type eventKeys struct {
	typ, event, userID, anonymousID, channel, library string
	hasUserID, hasAnonymousID                         bool
}

func readEvent(message []byte, route string) eventKeys {
	var k eventKeys
	gjson.ParseBytes(message).ForEach(func(key, value gjson.Result) bool {
		switch key.Str {
		case "type":
			k.typ = value.String()
		case "event":
			k.event = value.String()
		case "userId":
			k.userID, k.hasUserID = value.String(), true
		case "anonymousId":
			k.anonymousID, k.hasAnonymousID = value.String(), true
		case "channel":
			k.channel = value.String()
		case "context":
			k.library = value.Get("library.name").String()
		}
		return true
	})
	if k.typ == "" {
		k.typ = routeType(route)
	}
	return k
}

// routeType is the type RudderStack gives an event that arrived on a
// single-event route without one.
func routeType(route string) string {
	switch t := path.Base(route); t {
	case "batch", "import":
		return ""
	default:
		return t
	}
}

// window holds the records a read covers: after the cursor, up to the
// store cursor taken before the read, and, for a time, received at or after
// it. A slow upload that started before the time completes with a later
// seq, so the time alone keeps it out.
type window struct {
	since  uint64
	floor  time.Time
	cursor uint64
}

func (h *Handler) window(s Since) window {
	w := window{since: s.Cursor, cursor: h.store.Cursor()}
	if s.At.IsZero() {
		return w
	}
	w.floor, w.since = s.At, w.cursor
	records, _ := h.store.Since(0)
	for _, r := range records {
		if r.Seq > w.cursor {
			break
		}
		if !r.ReceivedAt.Before(s.At) {
			w.since = r.Seq - 1
			break
		}
	}
	return w
}

func (w window) holds(r *store.Record) bool {
	return r.Seq > w.since && r.Seq <= w.cursor && !r.ReceivedAt.Before(w.floor)
}

// records returns the records in the window, oldest first.
func (h *Handler) records(w window) []*store.Record {
	all, _ := h.store.Since(w.since)
	out := all[:0]
	for _, r := range all {
		if w.holds(r) {
			out = append(out, r)
		}
	}
	return out
}

// keyMatches matches a literal write key by its masked form, so a caller
// never types the mask.
func keyMatches(keys []string, r *store.Record) bool {
	if len(keys) == 0 {
		return true
	}
	for _, key := range keys {
		if store.MaskWriteKey(key) == r.WriteKey {
			return true
		}
	}
	return false
}

func (q *eventsQuery) eventMatches(k eventKeys) bool {
	switch {
	case len(q.types) > 0 && !slices.Contains(q.types, k.typ):
		return false
	case q.userID != nil && (!k.hasUserID || k.userID != *q.userID):
		return false
	case q.anonymousID != nil && (!k.hasAnonymousID || k.anonymousID != *q.anonymousID):
		return false
	case len(q.events) == 0:
		return true
	}
	for _, name := range q.events {
		if prefix, ok := strings.CutSuffix(name, "*"); ok && strings.HasPrefix(k.event, prefix) || name == k.event {
			return true
		}
	}
	return false
}

// matches returns the indexes of the events in r that the filters select.
func (q *eventsQuery) matches(r *store.Record) []int {
	if !keyMatches(q.writeKeys, r) {
		return nil
	}
	var out []int
	for i, ev := range r.Events {
		if q.eventMatches(readEvent(ev.Message, r.Route)) {
			out = append(out, i)
		}
	}
	return out
}

// acceptedMatches counts what a long-poll waits for: the accepted events
// the filters select. A rejected copy never counts.
func (q *eventsQuery) acceptedMatches(w window) func(*store.Record) int {
	return func(r *store.Record) int {
		if !accepted(r) || r.Seq <= w.since || r.ReceivedAt.Before(w.floor) {
			return 0
		}
		return len(q.matches(r))
	}
}
