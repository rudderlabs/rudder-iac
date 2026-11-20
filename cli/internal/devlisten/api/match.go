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
// store cursor of the read, and, for a time, received at or after
// it. A slow upload that started before the time completes with a later
// seq, so the time alone keeps it out.
type window struct {
	since  uint64
	floor  time.Time
	cursor uint64
}

func (w window) holds(r *store.Record) bool {
	return r.Seq > w.since && r.Seq <= w.cursor && !r.ReceivedAt.Before(w.floor)
}

// lost reports whether the store evicted a record of the window: one after
// the cursor or, for a time, one received at or after it. Every read uses
// it, so the summary, the stream and the CLI agree on which windows are
// incomplete. Both values come from the snapshot that gave the records.
func (w window) lost(evictedThrough uint64, evictedAt time.Time) bool {
	// evictedAt is the zero time until something is evicted, and a caller
	// may name an earlier time.
	if evictedThrough == 0 {
		return false
	}
	if w.floor.IsZero() {
		return evictedThrough > w.since
	}
	return !evictedAt.Before(w.floor)
}

// reading is a window with its records, oldest first, the store totals and
// whether the window lost a record, all of one moment of the store.
type reading struct {
	window
	records []*store.Record
	stats   store.Stats
	lost    bool
}

// read takes the window of s from one snapshot of the store. Every read
// handler gets its records this way and reads nothing else of the store, so
// a request stored during the read cannot show in one part of an answer and
// not in another.
func (h *Handler) read(s Since) reading {
	if s.At.IsZero() {
		return h.reread(window{since: s.Cursor})
	}
	snap := h.store.Snapshot(0)
	w := window{since: snap.Cursor, floor: s.At}
	for _, r := range snap.Records {
		if !r.ReceivedAt.Before(s.At) {
			w.since = r.Seq - 1
			break
		}
	}
	return w.reading(snap)
}

// reread reads a window again, from the same seq, after a wait.
func (h *Handler) reread(w window) reading {
	return w.reading(h.store.Snapshot(w.since))
}

func (w window) reading(snap store.Snapshot) reading {
	w.cursor = snap.Cursor
	records := snap.Records[:0]
	for _, r := range snap.Records {
		if w.holds(r) {
			records = append(records, r)
		}
	}
	return reading{window: w, records: records, stats: snap.Stats, lost: w.lost(snap.Stats.EvictedThrough, snap.EvictedAt)}
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
