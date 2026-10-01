package api

import (
	"fmt"
	"maps"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"
)

const (
	// MaxWait keeps a long-poll under the 120 s limit of a typical agent
	// shell; a longer wait loops on the cursor.
	MaxWait = 110 * time.Second
	// MaxLimit bounds a stream page.
	MaxLimit     = 1000
	defaultLimit = 100
)

// Since is a parsed since parameter: a cursor, or a time when At is set.
type Since struct {
	Cursor uint64
	At     time.Time
}

// ParseSince reads a cursor, a duration back from now, or an RFC 3339 time.
func ParseSince(value string, now time.Time) (Since, error) {
	if cursor, err := strconv.ParseUint(value, 10, 64); err == nil {
		return Since{Cursor: cursor}, nil
	}
	if d, err := time.ParseDuration(value); err == nil {
		if d < 0 {
			return Since{}, fmt.Errorf("%q is negative; give a duration back from now, such as 5m", value)
		}
		return Since{At: now.Add(-d)}, nil
	}
	if at, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return Since{At: at}, nil
	}
	return Since{}, fmt.Errorf("%q is not a cursor, a duration such as 5m or an RFC 3339 time", value)
}

// ParseWait reads a wait parameter: a Go duration from 0 to MaxWait.
func ParseWait(value string) (time.Duration, error) {
	d, err := time.ParseDuration(value)
	switch {
	case err != nil:
		return 0, fmt.Errorf("%q is not a duration such as 30s", value)
	case d < 0:
		return 0, fmt.Errorf("%s is negative", value)
	case d > MaxWait:
		return 0, fmt.Errorf("%s exceeds the maximum of %gs", value, MaxWait.Seconds())
	}
	return d, nil
}

// CheckFieldPath rejects a fields value that would select nothing on
// purpose: a comma list, or a path with an empty segment.
func CheckFieldPath(path string) error {
	if strings.Contains(path, ",") {
		a, b, _ := strings.Cut(path, ",")
		return fmt.Errorf("repeat the parameter for each path, as fields=%s&fields=%s", a, b)
	}
	if slices.Contains(strings.Split(path, "."), "") {
		return fmt.Errorf("%q has an empty path segment", path)
	}
	return nil
}

// eventsQuery holds the parameters of /events.
type eventsQuery struct {
	raw      url.Values
	since    Since
	serverID string
	view     string
	fields   [][]string
	limit    int
	wait     time.Duration
	min      int

	events      []string
	types       []string
	writeKeys   []string
	userID      *string
	anonymousID *string
}

// paramError is a 400 answer that names its parameter.
type paramError struct {
	code    string
	param   string
	message string
}

var eventsParams = []string{
	"since", "serverId", "view", "fields", "limit", "event", "type", "writeKey", "userId", "anonymousId", "wait", "min",
}

// repeatable parameters OR their values; the others are singletons.
var repeatable = []string{"fields", "event", "type", "writeKey"}

func parseEventsQuery(query url.Values, now time.Time) (*eventsQuery, *paramError) {
	invalid := func(param, format string, args ...any) *paramError {
		return &paramError{code: "invalid_parameter", param: param, message: param + ": " + fmt.Sprintf(format, args...)}
	}

	for _, name := range slices.Sorted(maps.Keys(query)) {
		if !slices.Contains(eventsParams, name) {
			return nil, &paramError{code: "unknown_parameter", param: name, message: "/_dev/v1/events takes no parameter " + name + "."}
		}
		if len(query[name]) > 1 && !slices.Contains(repeatable, name) {
			return nil, invalid(name, "repeated; give it once")
		}
	}

	q := &eventsQuery{
		raw:       query,
		serverID:  query.Get("serverId"),
		view:      "full",
		limit:     defaultLimit,
		min:       1,
		events:    query["event"],
		types:     query["type"],
		writeKeys: query["writeKey"],
	}
	if v, ok := query["userId"]; ok {
		q.userID = &v[0]
	}
	if v, ok := query["anonymousId"]; ok {
		q.anonymousID = &v[0]
	}
	if query.Has("since") {
		since, err := ParseSince(query.Get("since"), now)
		if err != nil {
			return nil, invalid("since", "%s", err)
		}
		q.since = since
	}
	if query.Has("view") {
		q.view = query.Get("view")
		if !slices.Contains([]string{"full", "compact", "counts"}, q.view) {
			return nil, invalid("view", "%q is not full, compact or counts", q.view)
		}
	}
	counts := q.view == "counts"

	if query.Has("fields") {
		if query.Has("view") {
			return nil, invalid("fields", "cannot combine with view")
		}
		for _, path := range query["fields"] {
			if err := CheckFieldPath(path); err != nil {
				return nil, invalid("fields", "%s", err)
			}
			q.fields = append(q.fields, strings.Split(path, "."))
		}
	}
	if query.Has("limit") {
		if counts {
			return nil, invalid("limit", "applies to the stream, not to view=counts")
		}
		n, err := strconv.Atoi(query.Get("limit"))
		if err != nil || n < 1 || n > MaxLimit {
			return nil, invalid("limit", "%q is not a number from 1 to %d", query.Get("limit"), MaxLimit)
		}
		q.limit = n
	}
	if query.Has("wait") {
		if !counts {
			return nil, invalid("wait", "applies to view=counts only")
		}
		d, err := ParseWait(query.Get("wait"))
		if err != nil {
			return nil, invalid("wait", "%s", err)
		}
		q.wait = d
	}
	if query.Has("min") {
		if !counts {
			return nil, invalid("min", "applies to view=counts only")
		}
		n, err := strconv.Atoi(query.Get("min"))
		if err != nil || n < 1 {
			return nil, invalid("min", "%q is not a number of 1 or more", query.Get("min"))
		}
		q.min = n
	}
	return q, nil
}

// filtered is true when a filter on the events themselves is set. Such a
// filter keeps only the requests that carry a matching event.
func (q *eventsQuery) filtered() bool {
	return len(q.events) > 0 || len(q.types) > 0 || q.userID != nil || q.anonymousID != nil
}
