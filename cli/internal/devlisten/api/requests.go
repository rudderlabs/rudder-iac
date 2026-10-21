package api

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/tidwall/gjson"

	"github.com/rudderlabs/rudder-iac/cli/internal/devlisten/ingest"
	"github.com/rudderlabs/rudder-iac/cli/internal/devlisten/store"
)

// defaultMaxBytes keeps a page of request records small enough for an agent
// to read in one call.
const defaultMaxBytes = 24_000

var (
	requestsParams = []string{
		"since", "serverId", "limit", "kind", "statusCode", "writeKey", "failed", "messageId", "view", "fields", "maxBytes",
	}
	recordParams = []string{"serverId", "view", "fields", "maxBytes"}
	statusCode   = regexp.MustCompile(`^[1-5]([0-9][0-9]|xx)$`)
)

type requestsQuery struct {
	raw         url.Values
	since       Since
	serverID    string
	limit       int
	kind        string
	statusCodes []string
	writeKeys   []string
	failed      *bool
	messageID   *string
	view        string
	fields      [][]string
	maxBytes    int
}

// parseRequestsQuery reads the parameters of /requests, or of one record
// when single is set.
func parseRequestsQuery(query url.Values, now time.Time, single bool) (*requestsQuery, *paramError) {
	route, allowed, views := base+"requests", requestsParams, []string{"list", "compact", "full"}
	if single {
		route, allowed, views = base+"requests/{seq}", recordParams, []string{"compact", "full"}
	}
	if err := checkNames(query, route, allowed, []string{"fields", "statusCode", "writeKey"}); err != nil {
		return nil, err
	}
	q := &requestsQuery{
		raw:         query,
		serverID:    query.Get("serverId"),
		limit:       defaultLimit,
		kind:        "ingestion",
		statusCodes: query["statusCode"],
		writeKeys:   query["writeKey"],
		view:        views[0],
		maxBytes:    defaultMaxBytes,
	}
	var err *paramError
	if q.since, err = parseSinceParam(query, now); err != nil {
		return nil, err
	}
	if query.Has("view") {
		q.view = query.Get("view")
		if !slices.Contains(views, q.view) {
			return nil, invalid("view", "%q is not %s", q.view, strings.Join(views, ", "))
		}
	}
	if q.fields, err = parseFields(query); err != nil {
		return nil, err
	}
	for _, path := range q.fields {
		if !slices.Contains(recordRoots, path[0]) {
			err := invalid("fields", "%q is not a key of the request record", path[0])
			err.details = map[string]any{"validRoots": recordRoots}
			return nil, err
		}
	}
	if query.Has("limit") {
		if q.limit, err = parseCount(query, "limit", 0, MaxLimit); err != nil {
			return nil, err
		}
	}
	if query.Has("maxBytes") {
		if q.maxBytes, err = parseCount(query, "maxBytes", 0, 0); err != nil {
			return nil, err
		}
	}
	if query.Has("kind") {
		q.kind = query.Get("kind")
		if !slices.Contains([]string{"ingestion", "control", "all"}, q.kind) {
			return nil, invalid("kind", "%q is not ingestion, control or all", q.kind)
		}
	}
	for _, code := range q.statusCodes {
		if !statusCode.MatchString(code) {
			return nil, invalid("statusCode", "%q is not a status such as 401 or a class such as 4xx", code)
		}
	}
	if query.Has("failed") {
		failed, ok := map[string]bool{"true": true, "false": false}[query.Get("failed")]
		if !ok {
			return nil, invalid("failed", "%q is not true or false", query.Get("failed"))
		}
		q.failed = &failed
	}
	if v, ok := query["messageId"]; ok {
		q.messageID = &v[0]
	}
	return q, nil
}

func (q *requestsQuery) matches(r *store.Record) bool {
	switch {
	case q.kind != "all" && r.Kind != q.kind:
		return false
	case !keyMatches(q.writeKeys, r):
		return false
	case q.failed != nil && (outcome(r) != outcomeAccepted) != *q.failed:
		return false
	case len(q.statusCodes) > 0 && !slices.ContainsFunc(q.statusCodes, func(code string) bool {
		status := r.Response.StatusCode
		return code == strconv.Itoa(status) || strings.HasSuffix(code, "xx") && int(code[0]-'0') == status/100
	}):
		return false
	case q.messageID != nil:
		return slices.ContainsFunc(r.Events, func(ev ingest.Event) bool {
			id := ownKeys(ev.Message).MessageID
			return id != nil && gjson.ParseBytes(id).String() == *q.messageID
		})
	}
	return true
}

// requests answers the request records after the cursor in a JSON envelope.
// Unlike the stream it has a byte cap, because a record holds the raw body.
func (h *Handler) requests(w http.ResponseWriter, r *http.Request) {
	q, ok := h.parseRequests(w, r, false)
	if !ok {
		return
	}
	win := h.window(q.since)
	var matched []*store.Record
	for _, rec := range h.records(win) {
		if q.matches(rec) {
			matched = append(matched, rec)
		}
	}
	env := requestsEnvelope{
		APIVersion:     h.cfg.Identity.APIVersion,
		ServerID:       h.cfg.Identity.ServerID,
		Since:          win.since,
		Cursor:         win.cursor,
		EvictedThrough: h.store.Stats().EvictedThrough,
		Total:          len(matched),
		View:           q.view,
		Requests:       []json.RawMessage{},
	}
	switch {
	case q.fields != nil:
		env.View = "fields"
	case q.view != "full":
		env.Omitted = &omitted{Fields: []string{"request.body"}}
	}

	page := matched
	if q.limit < len(matched) {
		page = matched[:q.limit]
		if q.limit > 0 {
			env.HasMore, env.Cursor = true, matched[q.limit].Seq-1
		}
	}
	for _, rec := range page {
		env.Requests = append(env.Requests, h.render(rec, q.view, q.fields))
	}
	env.Returned = len(env.Requests)
	h.link(&env, q)
	if q.maxBytes > 0 && size(env) > q.maxBytes {
		h.fit(&env, page, q)
	}
	if env.Links.Next != nil {
		w.Header().Set("Link", "<"+base+*env.Links.Next+`>; rel="next"`)
	}
	writeJSON(w, http.StatusOK, env)
}

// size is the byte count writeJSON sends for env, its newline included.
func size(env requestsEnvelope) int {
	out, _ := encode(env)
	return len(out) + 1
}

// link sets links.next from the cursor.
func (h *Handler) link(env *requestsEnvelope, q *requestsQuery) {
	env.Links.Next = nil
	if env.HasMore {
		next := "requests?" + nextQuery(q.raw, env.Cursor)
		env.Links.Next = &next
	}
}

// fit keeps the longest run of whole records that fits in maxBytes. It sets
// the metadata of each candidate page before it measures, so the bytes it
// measures are the bytes sent.
func (h *Handler) fit(env *requestsEnvelope, page []*store.Record, q *requestsQuery) {
	items := env.Requests
	if len(items) == 0 {
		return
	}
	env.HasMore = true
	env.Requests = []json.RawMessage{}
	cut := func(n int) {
		env.Cursor, env.Returned = page[n-1].Seq, n
		h.link(env, q)
		env.Truncated = &truncation{
			By: "maxBytes", Kept: n, MatchedAfter: env.Total - n, Next: h.curl(*env.Links.Next),
		}
	}
	kept, itemBytes := 0, -1
	for n := 1; n < len(items); n++ {
		// n items and the n-1 commas between them.
		itemBytes += len(items[n-1]) + 1
		cut(n)
		if size(*env)+itemBytes > q.maxBytes {
			break
		}
		kept = n
	}
	if kept > 0 {
		cut(kept)
		env.Requests = items[:kept]
		return
	}
	// The first record alone does not fit. The cursor passes it, so a pager
	// that follows links.next ends; truncated.next reads the record.
	env.Cursor, env.Returned = page[0].Seq, 0
	env.HasMore = env.Total > 1
	h.link(env, q)
	env.Truncated = &truncation{By: "maxBytes", RequestBytes: len(items[0]), Next: h.wholeBody(page[0].Seq)}
}

func (h *Handler) wholeBody(seq uint64) string {
	return h.curl("requests/" + strconv.FormatUint(seq, 10) + "?fields=request.body&maxBytes=0")
}

// request answers one record. Its default view leaves out each event's
// message, because the body holds the same bytes.
func (h *Handler) request(w http.ResponseWriter, r *http.Request) {
	seq, _ := strconv.ParseUint(strings.TrimPrefix(r.URL.Path, base+"requests/"), 10, 64)
	rec := h.store.Get(seq)
	if rec == nil {
		h.fail(w, http.StatusNotFound, "not_found",
			"No request "+strconv.FormatUint(seq, 10)+": it was never received, or the store evicted it.", h.curl("requests"))
		return
	}
	q, ok := h.parseRequests(w, r, true)
	if !ok {
		return
	}
	view := q.view
	if view == "compact" {
		view = "record"
	}
	out := h.render(rec, view, q.fields)
	if q.maxBytes > 0 && len(out)+1 > q.maxBytes {
		writeJSON(w, http.StatusOK, struct {
			Seq       uint64      `json:"seq"`
			Truncated *truncation `json:"truncated"`
		}{seq, &truncation{By: "maxBytes", RequestBytes: len(out), Next: h.wholeBody(seq)}})
		return
	}
	writeJSON(w, http.StatusOK, json.RawMessage(out))
}

func (h *Handler) parseRequests(w http.ResponseWriter, r *http.Request, single bool) (*requestsQuery, bool) {
	query, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		h.fail(w, http.StatusBadRequest, "invalid_parameter", "The query string is malformed: "+err.Error()+".", h.curl(""))
		return nil, false
	}
	q, perr := parseRequestsQuery(query, h.now(), single)
	if perr != nil {
		h.failParam(w, perr, h.curl(""))
		return nil, false
	}
	if q.serverID != "" && q.serverID != h.cfg.Identity.ServerID {
		h.serverChanged(w)
		return nil, false
	}
	return q, true
}

type requestsEnvelope struct {
	APIVersion     string      `json:"apiVersion"`
	ServerID       string      `json:"serverId"`
	Since          uint64      `json:"since"`
	Cursor         uint64      `json:"cursor"`
	EvictedThrough uint64      `json:"evictedThrough"`
	HasMore        bool        `json:"hasMore"`
	Total          int         `json:"total"`
	Returned       int         `json:"returned"`
	View           string      `json:"view"`
	Omitted        *omitted    `json:"omitted"`
	Truncated      *truncation `json:"truncated"`
	Links          struct {
		Next *string `json:"next"`
	} `json:"links"`
	Requests []json.RawMessage `json:"requests"`
}

type omitted struct {
	Fields []string `json:"fields"`
}

type truncation struct {
	By           string `json:"by"`
	Kept         int    `json:"kept,omitempty"`
	MatchedAfter int    `json:"matchedAfter,omitempty"`
	RequestBytes int    `json:"requestBytes,omitempty"`
	Next         string `json:"next"`
}

// recordView is the request record of the API. Fields may be added; a
// removal, rename or type change needs a new recordVersion.
type recordView struct {
	RecordVersion  int            `json:"recordVersion"`
	ServerID       string         `json:"serverId"`
	Seq            uint64         `json:"seq"`
	Kind           string         `json:"kind"`
	ReceivedAt     time.Time      `json:"receivedAt"`
	Route          string         `json:"route"`
	Transport      string         `json:"transport"`
	StatusCode     int            `json:"statusCode"`
	Failed         bool           `json:"failed"`
	Outcome        string         `json:"outcome"`
	WriteKey       string         `json:"writeKey"`
	WriteKeySha256 *string        `json:"writeKeySha256"`
	SourceID       string         `json:"sourceId"`
	Rejection      *rejectionView `json:"rejection"`
	Hint           *string        `json:"hint"`
	Request        struct {
		Method         string              `json:"method"`
		Target         string              `json:"target"`
		Headers        map[string][]string `json:"headers"`
		DroppedHeaders int                 `json:"droppedHeaders"`
		RemoteAddr     string              `json:"remoteAddr"`
		BodyBytes      int                 `json:"bodyBytes"`
		Body           store.Body          `json:"body"`
		BodyComplete   bool                `json:"bodyComplete"`
	} `json:"request"`
	Response struct {
		StatusCode int                 `json:"statusCode"`
		Headers    map[string][]string `json:"headers"`
		Body       string              `json:"body"`
	} `json:"response"`
	Events []eventView `json:"events"`
	// Omitted names the identity keys too deep for a client to decode. They
	// render as null; the body still holds them.
	Omitted *omitted `json:"omitted,omitempty"`
}

// recordRoots are the keys a fields path may start with. Every fields
// answer carries omitted when it is set, so it is not a root.
var recordRoots = func() []string {
	t := reflect.TypeFor[recordView]()
	var out []string
	for i := range t.NumField() {
		if name, _, _ := strings.Cut(t.Field(i).Tag.Get("json"), ","); name != "omitted" {
			out = append(out, name)
		}
	}
	return out
}()

type rejectionView struct {
	Stage  string `json:"stage"`
	Reason string `json:"reason"`
	Idx    *int   `json:"idx"`
}

// eventKeysRaw are the identity keys of an event as the SDK sent them, or
// nil when absent. Enrichment holds what RudderStack would add.
type eventKeysRaw struct {
	MessageID   json.RawMessage `json:"messageId"`
	Type        json.RawMessage `json:"type"`
	Event       json.RawMessage `json:"event"`
	UserID      json.RawMessage `json:"userId"`
	AnonymousID json.RawMessage `json:"anonymousId"`
}

type eventView struct {
	Idx int `json:"idx"`
	eventKeysRaw
	Message    json.RawMessage    `json:"message,omitempty"`
	Enrichment *ingest.Enrichment `json:"enrichment"`
}

// ownKeys reads the keys by name; the last of duplicate keys wins.
func ownKeys(message []byte) eventKeysRaw {
	var k eventKeysRaw
	gjson.ParseBytes(message).ForEach(func(key, value gjson.Result) bool {
		raw := json.RawMessage(value.Raw)
		switch key.Str {
		case "messageId":
			k.MessageID = raw
		case "type":
			k.Type = raw
		case "event":
			k.Event = raw
		case "userId":
			k.UserID = raw
		case "anonymousId":
			k.AnonymousID = raw
		}
		return true
	})
	return k
}

func (h *Handler) record(r *store.Record, withMessages bool) recordView {
	v := recordView{
		RecordVersion: recordVersion,
		ServerID:      h.cfg.Identity.ServerID,
		Seq:           r.Seq,
		Kind:          r.Kind,
		ReceivedAt:    r.ReceivedAt,
		Route:         r.Route,
		Transport:     r.Transport,
		StatusCode:    r.Response.StatusCode,
		Outcome:       outcome(r),
		Rejection:     rejection(r),
		WriteKey:      r.WriteKey.Key,
		SourceID:      "dev-" + keyHash(r.WriteKey)[:12],
		Events:        make([]eventView, len(r.Events)),
	}
	v.Failed = v.Outcome != outcomeAccepted
	if r.WriteKey.Sha256 != "" {
		v.WriteKeySha256 = &r.WriteKey.Sha256
	}
	v.Request.Method, v.Request.Target = r.Request.Method, r.Request.Target
	v.Request.Headers, v.Request.DroppedHeaders = listValues(r.Request.Headers), r.Request.DroppedHeaders
	v.Request.RemoteAddr, v.Request.BodyBytes = r.Request.RemoteAddr, r.Request.BodyBytes
	v.Request.Body, v.Request.BodyComplete = r.Request.Body, r.Request.BodyComplete
	v.Response.StatusCode, v.Response.Headers, v.Response.Body = r.Response.StatusCode, listValues(r.Response.Headers), r.Response.Body
	var cut []string
	for i, ev := range r.Events {
		keys, deep := keysThatNest(ownKeys(ev.Message), i)
		cut = append(cut, deep...)
		v.Events[i] = eventView{Idx: i, eventKeysRaw: keys, Enrichment: ev.Enrichment}
		if withMessages {
			v.Events[i].Message = ev.Message
		}
	}
	if cut != nil {
		v.Omitted = &omitted{Fields: cut}
	}
	return v
}

// keyNesting is the nesting the record and the envelope add around an
// identity key: the envelope object and its requests array, then the
// record, its events array and the event.
const keyNesting = 5

// keysThatNest sets to nil each key that a client could not decode inside
// the envelope, because the record adds levels to what the SDK sent. It
// returns the paths of the keys it cleared.
func keysThatNest(k eventKeysRaw, idx int) (eventKeysRaw, []string) {
	var (
		cut  []string
		open = bytes.Repeat([]byte("["), keyNesting)
		shut = bytes.Repeat([]byte("]"), keyNesting)
	)
	for _, key := range []struct {
		name string
		raw  *json.RawMessage
	}{
		{"messageId", &k.MessageID}, {"type", &k.Type}, {"event", &k.Event},
		{"userId", &k.UserID}, {"anonymousId", &k.AnonymousID},
	} {
		if *key.raw == nil || json.Valid(slices.Concat(open, *key.raw, shut)) {
			continue
		}
		*key.raw = nil
		cut = append(cut, "events."+strconv.Itoa(idx)+"."+key.name)
	}
	return k, cut
}

func rejection(r *store.Record) *rejectionView {
	if r.Rejection == nil {
		return nil
	}
	return &rejectionView{Stage: r.Rejection.Stage, Reason: r.Rejection.Reason, Idx: r.Rejection.Idx}
}

// keyHash is the hash the /sourceConfig answer builds its source id from. A
// key of up to 8 characters is stored in clear, so its hash is computed here.
func keyHash(k store.WriteKey) string {
	if k.Sha256 != "" {
		return k.Sha256
	}
	sum := sha256.Sum256([]byte(k.Key))
	return hex.EncodeToString(sum[:])
}

// listValues gives each kept header the list form of HTTP; the store keeps
// the first value of each.
func listValues(h map[string]string) map[string][]string {
	out := make(map[string][]string, len(h))
	for name, value := range h {
		out[name] = []string{value}
	}
	return out
}

// render encodes one record in a view, or only the fields paths with seq
// and request.method. "record" is the default view of one record: the
// record without the messages.
func (h *Handler) render(r *store.Record, view string, fields [][]string) []byte {
	if fields != nil {
		keep := append([][]string{{"seq"}, {"request", "method"}, {"omitted"}}, fields...)
		return selectFields(gjson.ParseBytes(h.full(r)), fieldTree(keep))
	}
	var v any
	switch view {
	case "full":
		return h.full(r)
	case "list":
		v = struct {
			Seq        uint64 `json:"seq"`
			Kind       string `json:"kind"`
			Method     string `json:"method"`
			Route      string `json:"route"`
			StatusCode int    `json:"statusCode"`
			Outcome    string `json:"outcome"`
			EventCount int    `json:"eventCount"`
		}{r.Seq, r.Kind, r.Request.Method, r.Route, r.Response.StatusCode, outcome(r), len(r.Events)}
	case "compact":
		item := compactItem{
			Seq: r.Seq, ReceivedAt: r.ReceivedAt, Method: r.Request.Method, Route: r.Route,
			StatusCode: r.Response.StatusCode, Outcome: outcome(r), Kind: r.Kind,
			Rejection: rejection(r), Events: make([]compactEvent, len(r.Events)),
		}
		for i, ev := range r.Events {
			k, deep := keysThatNest(ownKeys(ev.Message), i)
			item.Events[i] = compactEvent{Idx: i, MessageID: k.MessageID, Type: k.Type, Event: k.Event}
			for _, path := range deep {
				switch path[strings.LastIndexByte(path, '.')+1:] {
				case "messageId", "type", "event":
					item.Omitted = appendOmitted(item.Omitted, path)
				}
			}
		}
		v = item
	default:
		v = h.record(r, false)
	}
	out, _ := encode(v)
	return out
}

// full encodes the record with each event as sent. Ingestion accepts JSON
// up to the depth limit of the encoder, and the record and the envelope add
// levels, so a record too deep to send in the envelope leaves out the
// messages. Its body is a string, so it still encodes.
func (h *Handler) full(r *store.Record) []byte {
	out, err := encode(h.record(r, true))
	if err == nil && json.Valid(slices.Concat([]byte("[["), out, []byte("]]"))) {
		return out
	}
	out, _ = encode(h.record(r, false))
	return out
}

type compactItem struct {
	Seq        uint64         `json:"seq"`
	ReceivedAt time.Time      `json:"receivedAt"`
	Method     string         `json:"method"`
	Route      string         `json:"route"`
	StatusCode int            `json:"statusCode"`
	Outcome    string         `json:"outcome"`
	Kind       string         `json:"kind"`
	Rejection  *rejectionView `json:"rejection"`
	Events     []compactEvent `json:"events"`
	Omitted    *omitted       `json:"omitted,omitempty"`
}

func appendOmitted(o *omitted, path string) *omitted {
	if o == nil {
		o = &omitted{}
	}
	o.Fields = append(o.Fields, path)
	return o
}

type compactEvent struct {
	Idx       int             `json:"idx"`
	MessageID json.RawMessage `json:"messageId"`
	Type      json.RawMessage `json:"type"`
	Event     json.RawMessage `json:"event"`
}

// encode writes JSON the way every answer of the API does: curl lines hold
// & and must paste as they print.
func encode(v any) ([]byte, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(b.Bytes(), []byte("\n")), nil
}
