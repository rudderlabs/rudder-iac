package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"slices"
	"strconv"

	"github.com/tidwall/gjson"
)

func (h *Handler) events(w http.ResponseWriter, r *http.Request) {
	query, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		h.fail(w, http.StatusBadRequest, "invalid_parameter", "The query string is malformed: "+err.Error()+".",
			eventsHelp(query))
		return
	}
	q, perr := parseEventsQuery(query, h.now())
	if perr != nil {
		h.failParam(w, http.StatusBadRequest, perr.code, perr.param, perr.message, eventsHelp(query))
		return
	}
	if q.serverID != "" && q.serverID != h.cfg.Identity.ServerID {
		h.serverChanged(w)
		return
	}
	if q.view == "counts" {
		h.summary(w, r, q)
		return
	}
	h.stream(w, r, q)
}

func eventsHelp(query url.Values) string {
	if query.Get("view") == "counts" {
		return "rudder-cli dev events --help"
	}
	return "rudder-cli dev events list --help"
}

func (h *Handler) serverChanged(w http.ResponseWriter) {
	writeJSON(w, http.StatusConflict, map[string]apiError{"error": {
		Status:  http.StatusConflict,
		Code:    "server_changed",
		Message: "Another listener answers at this URL, or it restarted. Start again from a fresh cursor.",
		Details: map[string]any{"serverId": h.cfg.Identity.ServerID, "startedAt": h.cfg.Identity.StartedAt},
		Next:    "rudder-cli dev events --json",
	}})
}

// stream writes the accepted events after the cursor, one per line. A page
// holds whole requests only, so the next cursor never splits one.
func (h *Handler) stream(w http.ResponseWriter, r *http.Request, q *eventsQuery) {
	win := h.window(q.since)
	var (
		body    bytes.Buffer
		n       int
		cursor  = win.cursor
		scanned uint64
		hasMore bool
	)
	for _, rec := range h.records(win) {
		var page []int
		if accepted(rec) {
			page = q.matches(rec)
		}
		if len(page) > 0 && n > 0 && n+len(page) > q.limit {
			cursor, hasMore = scanned, true
			break
		}
		for _, i := range page {
			body.Write(q.line(rec.Events[i].Message))
			body.WriteByte('\n')
		}
		n += len(page)
		scanned = rec.Seq
	}

	header := w.Header()
	header.Set("Content-Type", "application/x-ndjson")
	header.Set("X-Dev-Cursor", strconv.FormatUint(cursor, 10))
	header.Set("X-Dev-Has-More", strconv.FormatBool(hasMore))
	if hasMore {
		next := url.Values{}
		for k, v := range q.raw {
			next[k] = v
		}
		next.Set("since", strconv.FormatUint(cursor, 10))
		header.Set("Link", "<"+base+"events?"+next.Encode()+`>; rel="next"`)
	}
	w.WriteHeader(http.StatusOK)
	if r.Method != http.MethodHead {
		_, _ = w.Write(body.Bytes())
	}
}

// line is one event on one line. Selection only drops keys; compaction only
// drops whitespace between tokens, so a pretty-printed event stays one line.
func (q *eventsQuery) line(message []byte) []byte {
	selected := message
	switch {
	case len(q.fields) > 0:
		selected = selectFields(gjson.ParseBytes(message), fieldTree(q.fields))
	case q.view == "compact":
		selected = compactView(gjson.ParseBytes(message))
	}
	var out bytes.Buffer
	if err := json.Compact(&out, selected); err != nil {
		// The ingestion handler stores only events that parse, so this
		// keeps a line even if that ever changes.
		return bytes.ReplaceAll(selected, []byte("\n"), nil)
	}
	return out.Bytes()
}

// autoContext lists the context keys an SDK collects on its own, by the
// event spec names, and the app-set traits, ip and session keys.
var autoContext = []string{
	"app", "campaign", "device", "library", "locale", "network", "os", "page", "screen", "timezone", "userAgent",
	"traits", "ip", "sessionId", "sessionStart", "consentManagement",
}

func compactView(event gjson.Result) []byte {
	return object(event, func(key, value gjson.Result) (string, bool) {
		if key.Str != "context" || !value.IsObject() {
			return value.Raw, true
		}
		context := object(value, func(key, value gjson.Result) (string, bool) {
			return value.Raw, !slices.Contains(autoContext, key.Str)
		})
		return string(context), string(context) != "{}"
	})
}

// object rebuilds o from its raw keys and the values keep returns, so keys,
// key order and value bytes stay as sent.
func object(o gjson.Result, keep func(key, value gjson.Result) (string, bool)) []byte {
	var b bytes.Buffer
	b.WriteByte('{')
	o.ForEach(func(key, value gjson.Result) bool {
		raw, ok := keep(key, value)
		if !ok {
			return true
		}
		if b.Len() > 1 {
			b.WriteByte(',')
		}
		b.WriteString(key.Raw)
		b.WriteByte(':')
		b.WriteString(raw)
		return true
	})
	b.WriteByte('}')
	return b.Bytes()
}

// field is one segment of the fields paths. all keeps the whole value; an
// earlier or later shorter path wins over a longer one below it.
type field struct {
	key      string
	all      bool
	children []*field
}

func fieldTree(paths [][]string) []*field {
	var root []*field
	for _, segments := range paths {
		level := &root
		for i, key := range segments {
			idx := slices.IndexFunc(*level, func(f *field) bool { return f.key == key })
			if idx < 0 {
				*level = append(*level, &field{key: key})
				idx = len(*level) - 1
			}
			f := (*level)[idx]
			if i == len(segments)-1 {
				f.all, f.children = true, nil
			}
			if f.all {
				break
			}
			level = &f.children
		}
	}
	return root
}

// selectFields keeps the selected paths, nested as in the event, in the
// order the caller named them. An absent path is left out.
func selectFields(event gjson.Result, fields []*field) []byte {
	var b bytes.Buffer
	b.WriteByte('{')
	for _, f := range fields {
		value, ok := lastValue(event, f.key)
		if !ok {
			continue
		}
		raw := value.Raw
		if !f.all {
			if !value.IsObject() {
				continue
			}
			raw = string(selectFields(value, f.children))
			if raw == "{}" {
				continue
			}
		}
		if b.Len() > 1 {
			b.WriteByte(',')
		}
		key, _ := json.Marshal(f.key)
		b.Write(key)
		b.WriteByte(':')
		b.WriteString(raw)
	}
	b.WriteByte('}')
	return b.Bytes()
}

// lastValue reads a key by name, not by gjson path syntax, so a key with a
// dot, a star or a hash selects itself. The last of duplicate keys wins.
func lastValue(o gjson.Result, key string) (gjson.Result, bool) {
	var (
		found gjson.Result
		ok    bool
	)
	o.ForEach(func(k, v gjson.Result) bool {
		if k.Str == key {
			found, ok = v, true
		}
		return true
	})
	return found, ok
}
