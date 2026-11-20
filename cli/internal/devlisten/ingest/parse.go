package ingest

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/tidwall/gjson"
)

// parseEvents never decodes the payload into Go values, because the
// in-flight cap bounds bytes, not values. It also returns each message without
// \u0000 escapes, for the checks and the enrichment to read, because
// RudderStack ignores them (rudder-server misc.SanitizeJSON).
func parseEvents(payload []byte, reqType string) ([]Event, [][]byte, *gwError, *int) {
	// gjson.Valid recurses without a depth limit; encoding/json.Valid does not.
	if !json.Valid(payload) {
		return nil, nil, &errInvalidJSON, nil
	}

	var events []Event
	if reqType == "batch" || reqType == "import" {
		var e *gwError
		if events, e = batchEvents(payload); e != nil {
			return nil, nil, e, nil
		}
	} else {
		if bytes.TrimLeft(payload, " \t\r\n")[0] != '{' {
			return nil, nil, &errNotRudderEvent, nil
		}
		events = []Event{{Message: payload}}
	}

	sanitized := make([][]byte, len(events))
	for i, ev := range events {
		sanitized[i] = ev.Message
		if bytes.Contains(ev.Message, nullEscape) {
			sanitized[i] = bytes.ReplaceAll(ev.Message, nullEscape, nil)
			if !json.Valid(sanitized[i]) {
				return events, nil, &errInvalidJSON, &i
			}
		}
		if e := checkEvent(sanitized[i]); e != nil {
			return events, nil, e, &i
		}
	}
	return events, sanitized, nil, nil
}

func batchEvents(payload []byte) ([]Event, *gwError) {
	batch := gjson.GetBytes(payload, "batch")
	if !batch.IsArray() {
		return nil, &errNotRudderEvent
	}
	var (
		events    []Event
		notObject bool
		tooMany   bool
	)
	batch.ForEach(func(_, v gjson.Result) bool {
		switch {
		case !v.IsObject():
			notObject = true
			return false
		case len(events) == maxBatchEvents:
			tooMany = true
			return false
		}
		events = append(events, Event{Message: payload[v.Index : v.Index+len(v.Raw)]})
		return true
	})
	switch {
	case notObject:
		return nil, &errNotRudderEvent
	case tooMany:
		return nil, &errTooManyEvents
	case len(events) == 0:
		return nil, &errEmptyBatch
	}
	return events, nil
}

func checkEvent(message []byte) *gwError {
	event := gjson.ParseBytes(message)
	switch {
	case !numbersFit(event.Raw):
		return &errInvalidJSON
	case nonIdentifiable(event):
		return &errNonIdentifiable
	}
	return nil
}

var nullEscape = []byte(`\u0000`)

// numbersFit exists because RudderStack answers invalid json when a number
// does not fit a float64.
func numbersFit(raw string) bool {
	for i := 0; i < len(raw); i++ {
		switch c := raw[i]; {
		case c == '"':
			for i++; raw[i] != '"'; i++ {
				if raw[i] == '\\' {
					i++
				}
			}
		case c == '-' || '0' <= c && c <= '9':
			end := i + 1
			for end < len(raw) && strings.IndexByte("+-.0123456789eE", raw[end]) >= 0 {
				end++
			}
			if _, err := strconv.ParseFloat(raw[i:end], 64); err != nil {
				return false
			}
			i = end - 1
		}
	}
	return true
}

// topLevel lets the last of duplicate keys win, as RudderStack does.
func topLevel(event gjson.Result, keys ...string) map[string]gjson.Result {
	values := make(map[string]gjson.Result, len(keys))
	event.ForEach(func(key, value gjson.Result) bool {
		if slices.Contains(keys, key.Str) {
			values[key.Str] = value
		}
		return true
	})
	return values
}

func nonIdentifiable(event gjson.Result) bool {
	v := topLevel(event, "type", "userId", "anonymousId")
	switch v["type"].String() {
	case "extract", "record":
		return false
	}
	return sanitizeAndTrim(v["userId"].String()) == "" && sanitizeAndTrim(v["anonymousId"].String()) == ""
}

// invisibleRunes is rudder-go-kit sanitize's list (v0.80.0). An identifier of
// these runes only counts as empty.
var invisibleRunes = map[rune]bool{
	'\u0000': true, '\u0009': true, '\u00A0': true, '\u00AD': true, '\u034F': true,
	'\u061C': true, '\u115F': true, '\u1160': true, '\u17B4': true, '\u17B5': true,
	'\u180E': true, '\u2000': true, '\u2001': true, '\u2002': true, '\u2003': true,
	'\u2004': true, '\u2005': true, '\u2006': true, '\u2007': true, '\u2008': true,
	'\u2009': true, '\u200A': true, '\u200B': true, '\u200C': true, '\u200D': true,
	'\u200E': true, '\u200F': true, '\u202F': true, '\u205F': true, '\u2060': true,
	'\u2061': true, '\u2062': true, '\u2063': true, '\u2064': true, '\u206A': true,
	'\u206B': true, '\u206C': true, '\u206D': true, '\u206E': true, '\u206F': true,
	'\u3000': true, '\u2800': true, '\u3164': true, '\uFEFF': true, '\uFFA0': true,
}

func sanitizeAndTrim(s string) string {
	return strings.TrimSpace(strings.Map(func(r rune) rune {
		if invisibleRunes[r] || !unicode.IsPrint(r) {
			return -1
		}
		return r
	}, s))
}

var quotedAnonymousID = regexp.MustCompile(`^"(.*)"$`)

// pixelPayload reads each query key as a dotted path, as RudderStack does.
func pixelPayload(query url.Values, reqType string, now time.Time) ([]byte, error) {
	ts := now.Format(time.RFC3339)
	payload := map[string]any{
		"channel":           "web",
		"integrations":      map[string]any{"All": true},
		"originalTimestamp": ts,
		"sentAt":            ts,
		"type":              reqType,
	}
	for key, values := range query {
		if key == "writeKey" {
			continue
		}
		value := values[0]
		if key == "anonymousId" {
			value = quotedAnonymousID.ReplaceAllString(value, "$1")
		}
		setPath(payload, strings.Split(key, "."), value)
	}

	switch {
	case reqType == "page" && query.Has("name") && query.Get("name") == "":
		payload["name"] = "Unknown Page"
	case reqType == "track" && query.Has("event") && query.Get("event") == "":
		return nil, errors.New("track: Mandatory field 'event' missing")
	}
	return json.Marshal(payload)
}

// setPath skips a path with an empty key, like jsonparser.SetValue.
func setPath(m map[string]any, path []string, value string) {
	if slices.Contains(path, "") {
		return
	}
	for _, key := range path[:len(path)-1] {
		next, ok := m[key].(map[string]any)
		if !ok {
			next = map[string]any{}
			m[key] = next
		}
		m = next
	}
	m[path[len(path)-1]] = value
}
