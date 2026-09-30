package api

import (
	"bytes"
	"cmp"
	"encoding/json"
	"slices"
	"strings"
	"time"

	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten/internal/store"
)

// strippedContextKeys are the message.context keys compact leaves out: the
// docs' automatically collected contextual fields, then ip and traits
// (contextual, not auto-collected; traits is shown on the item), then the
// SDK session and consent keys.
var strippedContextKeys = []string{
	"app", "campaign", "device", "library", "locale", "network", "os", "page", "screen", "timezone", "userAgent",
	"ip", "traits",
	"sessionId", "sessionStart", "consentManagement",
}

type baseItem struct {
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
}

type fullItem struct {
	baseItem
	Message         json.RawMessage `json:"message"`
	EnrichedMessage json.RawMessage `json:"enrichedMessage"`
}

type compactItem struct {
	baseItem
	Context json.RawMessage `json:"context"`
}

type listItem struct {
	Seq        uint64    `json:"seq"`
	Idx        int       `json:"idx"`
	ReceivedAt time.Time `json:"receivedAt"`
	Type       *string   `json:"type"`
	Event      *string   `json:"event"`
	UserID     *string   `json:"userId"`
	WriteKey   string    `json:"writeKey"`
	StatusCode int       `json:"statusCode"`
}

func newBaseItem(rec store.Record, ev store.Event) baseItem {
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

	return baseItem{
		Seq: rec.Seq, Idx: ev.Idx, ReceivedAt: rec.ReceivedAt, Route: rec.Route, Transport: rec.Transport,
		StatusCode: rec.StatusCode, Outcome: rec.Outcome, WriteKey: rec.WriteKey,
		Type: ev.Type, Event: ev.Event, UserID: ev.UserID, AnonymousID: ev.AnonymousID, MessageID: ev.MessageID,
		SentAt:            rawString(sentAt),
		OriginalTimestamp: rawString(msg.OriginalTimestamp),
		Properties:        nullIfEmpty(msg.Properties),
		Traits:            nullIfEmpty(traits),
	}
}

// compactContext returns message.context without the stripped keys, and the
// keys it removed. --fields message.context reads the whole context.
func compactContext(message json.RawMessage) (json.RawMessage, []string) {
	var msg struct {
		Context json.RawMessage `json:"context"`
	}
	_ = json.Unmarshal(message, &msg)
	if isNull(msg.Context) {
		return nullIfEmpty(msg.Context), nil
	}
	var ctx map[string]json.RawMessage
	if json.Unmarshal(msg.Context, &ctx) != nil {
		return msg.Context, nil
	}
	var stripped []string
	for key := range ctx {
		if slices.Contains(strippedContextKeys, key) {
			stripped = append(stripped, key)
			delete(ctx, key)
		}
	}
	if len(ctx) == 0 {
		return nil, stripped
	}
	return encode(ctx), stripped
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

// fieldArg is one --fields value of a corrected command. Caller values are
// quoted; generated ones are bare.
type fieldArg struct {
	value     string
	generated bool
}

// checkFields validates dotted paths against the allowed roots. It returns
// the paths to keep: valid ones as given, a context path moved under
// message, and fallback when nothing valid is left.
func checkFields(fields, roots []string, fallback string) (kept []fieldArg, bad string) {
	for _, f := range fields {
		segments := strings.Split(f, ".")
		switch {
		case !slices.Contains(segments, "") && slices.Contains(roots, segments[0]):
			kept = append(kept, fieldArg{value: f})
		case segments[0] == "context" && slices.Contains(roots, "message") && !slices.Contains(segments, ""):
			kept = append(kept, fieldArg{value: "message." + f, generated: true})
			bad = cmp.Or(bad, f)
		default:
			bad = cmp.Or(bad, f)
		}
	}
	if len(kept) == 0 {
		kept = []fieldArg{{value: fallback, generated: true}}
	}
	return kept, bad
}

func failFields(p *params, bad string, roots []string, next string) {
	p.failWith("fields", next, map[string]any{"validRoots": roots},
		"%q does not start with one of %s", bad, strings.Join(roots, ", "))
}

func (c *command) fields(args []fieldArg) *command {
	for _, a := range args {
		if a.generated {
			c.bare("fields", a.value)
			continue
		}
		c.quoted("fields", a.value)
	}
	return c
}

// projectJSON keeps the named dotted paths of a JSON object plus the keep
// keys. The output stays nested; an absent path is null; a path through an
// array applies to every element.
func projectJSON(raw json.RawMessage, fields, keep []string) any {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var src any
	_ = dec.Decode(&src)
	paths := make([][]string, 0, len(keep)+len(fields))
	for _, k := range keep {
		paths = append(paths, strings.Split(k, "."))
	}
	for _, f := range fields {
		paths = append(paths, strings.Split(f, "."))
	}
	return project(src, paths)
}

func project(src any, paths [][]string) any {
	switch v := src.(type) {
	case map[string]any:
		out := map[string]any{}
		for key, rests := range groupByHead(paths) {
			if slices.ContainsFunc(rests, func(r []string) bool { return len(r) == 0 }) {
				out[key] = v[key]
				continue
			}
			out[key] = project(v[key], rests)
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i, el := range v {
			out[i] = project(el, paths)
		}
		return out
	default:
		return nil
	}
}

func groupByHead(paths [][]string) map[string][][]string {
	groups := map[string][][]string{}
	for _, p := range paths {
		groups[p[0]] = append(groups[p[0]], p[1:])
	}
	return groups
}

// dropNulls removes the top-level keys whose value is null and keeps the key
// order. Compact lists such keys as optional, so a null says nothing.
func dropNulls(obj []byte) []byte {
	dec := json.NewDecoder(bytes.NewReader(obj))
	if tok, err := dec.Token(); err != nil || tok != json.Delim('{') {
		return obj
	}
	out := []byte{'{'}
	for dec.More() {
		key, err := dec.Token()
		if err != nil {
			return obj
		}
		var val json.RawMessage
		if err := dec.Decode(&val); err != nil {
			return obj
		}
		if isNull(val) {
			continue
		}
		if len(out) > 1 {
			out = append(out, ',')
		}
		out = append(append(append(out, encode(key)...), ':'), val...)
	}
	return append(out, '}')
}
