package devlisten

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	// clientPollWait is the per-request wait WaitForEvents uses while it loops.
	clientPollWait = 10 * time.Second
	// clientRetryDelay is the least time between two polls that returned no
	// event, so WaitForEvents never spins.
	clientRetryDelay = 200 * time.Millisecond
)

var (
	// ErrServerChanged matches a 409 server_changed: the serverId the client
	// sent is not the running server's.
	ErrServerChanged = errors.New("server changed")
	// ErrCaptureGap matches a 410 capture_gap. This build never evicts, so
	// it never answers 410.
	ErrCaptureGap = errors.New("capture gap")
	// ErrNotFound matches a 404 not_found, such as an unknown request seq.
	ErrNotFound = errors.New("not found")
	// ErrOutputLimit is returned by WaitForEvents when one request is larger
	// than Query.MaxBytes, so no page can hold it.
	ErrOutputLimit = errors.New("output limit")
)

// View selects the shape of an /events item.
type View string

const (
	ViewSummary View = "summary"
	ViewCompact View = "compact"
	ViewFull    View = "full"
)

// Order is the page order. Only OrderAsc is served in this phase.
type Order string

const OrderAsc Order = "asc"

// Values of Query.Include.
const (
	IncludeContext    = "context"
	IncludeEnrichment = "enrichment"
)

// MaxBytesOff turns the server's page cap off (maxBytes=0). A zero
// MaxBytes sends nothing, so the server default of 24000 applies.
const MaxBytesOff = -1

// APIError is the error object of contract section 4.8.
type APIError struct {
	StatusCode int            `json:"-"`
	Code       string         `json:"code"`
	Message    string         `json:"message"`
	Param      string         `json:"param"`
	Details    map[string]any `json:"details"`
	Next       string         `json:"next"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("%s (%d): %s", e.Code, e.StatusCode, e.Message)
}

// MarshalJSON writes unused param and next as null, as the server does.
func (e *APIError) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Code    string         `json:"code"`
		Message string         `json:"message"`
		Param   *string        `json:"param"`
		Details map[string]any `json:"details"`
		Next    *string        `json:"next"`
	}{e.Code, e.Message, nullString(e.Param), e.Details, nullString(e.Next)})
}

func nullString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func (e *APIError) Is(target error) bool {
	switch target {
	case ErrServerChanged:
		return e.Code == "server_changed"
	case ErrCaptureGap:
		return e.Code == "capture_gap"
	case ErrNotFound:
		return e.Code == "not_found"
	}
	return false
}

type Client struct {
	baseURL  string
	http     *http.Client
	serverID string
	noPin    bool
}

type ClientOption func(*Client)

// WithHTTPClient replaces the HTTP client. nil is ignored.
func WithHTTPClient(hc *http.Client) ClientOption {
	return func(c *Client) {
		if hc != nil {
			c.http = hc
		}
	}
}

// WithServerID pins the client to one server: every query carries serverId
// and a restarted server answers ErrServerChanged.
func WithServerID(id string) ClientOption { return func(c *Client) { c.serverID = id } }

// WithoutServerIDPin stops Info from pinning the serverId it reads.
func WithoutServerIDPin() ClientOption { return func(c *Client) { c.noPin = true } }

func NewClient(baseURL string, opts ...ClientOption) *Client {
	// A transport per client keeps parallel tests from sharing idle
	// connections (rudder-iac .agents/knowledge/mistakes.md, DEX-677).
	transport := http.DefaultTransport.(*http.Transport).Clone()
	c := &Client{baseURL: strings.TrimRight(baseURL, "/"), http: &http.Client{Transport: transport}}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// URL is the base URL the client calls.
func (c *Client) URL() string { return c.baseURL }

// raw keeps the response bytes, so a caller can print what the server sent.
type raw struct {
	Raw json.RawMessage `json:"-"`
}

func (r *raw) setRaw(b []byte) { r.Raw = append(json.RawMessage(nil), bytes.TrimSpace(b)...) }

// Info is the /info object (contract section 4.1).
type Info struct {
	Ready
	WriteKeys     []string  `json:"writeKeys"`
	RecordVersion int       `json:"recordVersion"`
	Exposed       bool      `json:"exposed"`
	Store         InfoStore `json:"store"`
	raw
}

type StoreStats struct {
	Requests int `json:"requests"`
	Events   int `json:"events"`
	Control  int `json:"control"`
}

// InfoStore is the /info store object: counts, occupancy and evictions. The
// store keeps MaxRequests requests and MaxBytes bytes and drops the oldest
// whole requests first; EvictedThrough is the highest dropped seq.
type InfoStore struct {
	StoreStats
	Bytes          int    `json:"bytes"`
	Evicted        int    `json:"evicted"`
	EvictedThrough uint64 `json:"evictedThrough"`
	MaxRequests    int    `json:"maxRequests"`
	MaxBytes       int    `json:"maxBytes"`
}

// Info reads /info. The first successful call pins the serverId unless
// WithServerID or WithoutServerIDPin was given.
func (c *Client) Info(ctx context.Context) (Info, error) {
	var info Info
	if err := c.do(ctx, http.MethodGet, "/_dev/v1/info", nil, &info); err != nil {
		return Info{}, err
	}
	if c.serverID == "" && !c.noPin {
		c.serverID = info.ServerID
	}
	return info, nil
}

// Query selects events. Wait is the server-side long-poll of one Events
// request (at most 110 s); zero asks for a snapshot. Zero values send
// nothing, so the server defaults apply.
type Query struct {
	Since       uint64
	Limit       int
	Order       Order
	View        View
	Event       []string
	Type        []string
	Route       []string
	StatusCode  []int
	UserID      string
	AnonymousID string
	Include     []string
	Fields      []string
	MaxBytes    int
	Min         int
	Wait        time.Duration
}

// Values is the exact query string the client sends, without serverId.
func (q Query) Values() url.Values {
	v := url.Values{}
	setUint(v, "since", q.Since)
	setInt(v, "limit", q.Limit)
	setString(v, "order", string(q.Order))
	setString(v, "view", string(q.View))
	v["event"], v["type"], v["route"] = q.Event, q.Type, q.Route
	v["statusCode"] = intStrings(q.StatusCode)
	setString(v, "userId", q.UserID)
	setString(v, "anonymousId", q.AnonymousID)
	v["include"], v["fields"] = q.Include, q.Fields
	setMaxBytes(v, q.MaxBytes)
	setInt(v, "min", q.Min)
	setWait(v, q.Wait)
	return compact(v)
}

// RequestQuery selects requests; see Query for the zero-value rule.
// Failed is sent only when set, so false selects accepted requests.
type RequestQuery struct {
	Since      uint64
	Limit      int
	Order      Order
	Kind       string
	Route      []string
	StatusCode []int
	Failed     *bool
	Stage      string
	View       View
	Fields     []string
	MaxBytes   int
	Min        int
	Wait       time.Duration
}

func (q RequestQuery) Values() url.Values {
	v := url.Values{}
	setUint(v, "since", q.Since)
	setInt(v, "limit", q.Limit)
	setString(v, "order", string(q.Order))
	setString(v, "kind", q.Kind)
	v["route"], v["statusCode"] = q.Route, intStrings(q.StatusCode)
	if q.Failed != nil {
		v.Set("failed", strconv.FormatBool(*q.Failed))
	}
	setString(v, "stage", q.Stage)
	setString(v, "view", string(q.View))
	v["fields"] = q.Fields
	setMaxBytes(v, q.MaxBytes)
	setInt(v, "min", q.Min)
	setWait(v, q.Wait)
	return compact(v)
}

// RecordQuery shapes one /requests/{seq} record. View is ViewCompact (the
// default: events without their message copies) or ViewFull.
type RecordQuery struct {
	View     View
	Fields   []string
	MaxBytes int
}

func (q RecordQuery) Values() url.Values {
	v := url.Values{"fields": q.Fields}
	setString(v, "view", string(q.View))
	setMaxBytes(v, q.MaxBytes)
	return compact(v)
}

// SummaryQuery selects the records /summary counts. Each Expect value is
// NAME or NAME=COUNT; the answer then carries Expected.
type SummaryQuery struct {
	Since  uint64
	Expect []string
}

func (q SummaryQuery) Values() url.Values {
	v := url.Values{"expect": q.Expect}
	setUint(v, "since", q.Since)
	return compact(v)
}

func setUint(v url.Values, key string, n uint64) {
	if n > 0 {
		v.Set(key, strconv.FormatUint(n, 10))
	}
}

func setInt(v url.Values, key string, n int) {
	if n > 0 {
		v.Set(key, strconv.Itoa(n))
	}
}

func setString(v url.Values, key, s string) {
	if s != "" {
		v.Set(key, s)
	}
}

func setMaxBytes(v url.Values, n int) {
	switch {
	case n == MaxBytesOff:
		v.Set("maxBytes", "0")
	case n > 0:
		v.Set("maxBytes", strconv.Itoa(n))
	}
}

func setWait(v url.Values, d time.Duration) {
	if d > 0 {
		v.Set("wait", d.String())
	}
}

func intStrings(ns []int) []string {
	var out []string
	for _, n := range ns {
		out = append(out, strconv.Itoa(n))
	}
	return out
}

func compact(v url.Values) url.Values {
	for k, vs := range v {
		if len(vs) == 0 {
			delete(v, k)
		}
	}
	return v
}

// Omitted names what a non-full page left out and the call that shows it.
type Omitted struct {
	Fields  []string `json:"fields"`
	Context []string `json:"context"`
	Next    string   `json:"next"`
}

// Truncated is set when maxBytes cut the page. RequestBytes is set when the
// first request alone did not fit, and the page is then empty.
type Truncated struct {
	By           string `json:"by"`
	Kept         int    `json:"kept"`
	MatchedAfter int    `json:"matchedAfter"`
	RequestBytes int    `json:"requestBytes"`
	Next         string `json:"next"`
}

// Page is the /events envelope (contract section 4.3).
type Page struct {
	APIVersion string     `json:"apiVersion"`
	ServerID   string     `json:"serverId"`
	Since      uint64     `json:"since"`
	Cursor     uint64     `json:"cursor"`
	HasMore    bool       `json:"hasMore"`
	TimedOut   bool       `json:"timedOut"`
	WaitedMs   int64      `json:"waitedMs"`
	Unfiltered Unfiltered `json:"unfiltered"`
	View       View       `json:"view"`
	Omitted    *Omitted   `json:"omitted"`
	Truncated  *Truncated `json:"truncated"`
	Events     []Event    `json:"events"`
	raw
}

type Unfiltered struct {
	Requests       int `json:"requests"`
	FailedRequests int `json:"failedRequests"`
	Events         int `json:"events"`
	Control        int `json:"control"`
}

// Event is one flattened event. String fields are "" when the wire value is
// null; Raw keeps the item bytes when that difference matters. Message and
// EnrichedMessage are set under ViewFull only.
type Event struct {
	Seq               uint64          `json:"seq"`
	Idx               int             `json:"idx"`
	ReceivedAt        time.Time       `json:"receivedAt"`
	Route             string          `json:"route"`
	Transport         string          `json:"transport"`
	StatusCode        int             `json:"statusCode"`
	Outcome           string          `json:"outcome"`
	WriteKey          string          `json:"writeKey"`
	Type              string          `json:"type"`
	Name              string          `json:"event"`
	UserID            string          `json:"userId"`
	AnonymousID       string          `json:"anonymousId"`
	MessageID         string          `json:"messageId"`
	SentAt            string          `json:"sentAt"`
	OriginalTimestamp string          `json:"originalTimestamp"`
	Properties        map[string]any  `json:"properties"`
	Traits            map[string]any  `json:"traits"`
	Context           map[string]any  `json:"context"`
	Message           map[string]any  `json:"message"`
	EnrichedMessage   map[string]any  `json:"enrichedMessage"`
	Raw               json.RawMessage `json:"-"`
}

func (e *Event) UnmarshalJSON(b []byte) error {
	type plain Event
	var p plain
	if err := json.Unmarshal(b, &p); err != nil {
		return fmt.Errorf("decoding event: %w", err)
	}
	*e = Event(p)
	e.Raw = append(json.RawMessage(nil), b...)
	return nil
}

// MarshalJSON writes Raw back when it is set, so a decoded page re-encodes
// to the bytes the server sent, nulls included.
func (e Event) MarshalJSON() ([]byte, error) {
	if e.Raw != nil {
		return e.Raw, nil
	}
	type plain Event
	return json.Marshal(plain(e))
}

// Events runs one /events request. With q.Wait set it long-polls on the
// server and returns TimedOut at the deadline, without an error.
func (c *Client) Events(ctx context.Context, q Query) (Page, error) {
	var page Page
	if err := c.do(ctx, http.MethodGet, "/_dev/v1/events", q.Values(), &page); err != nil {
		return Page{}, err
	}
	return page, nil
}

// WaitForEvents loops 10 s long-polls until it holds at least q.Min events
// (default 1) or ctx ends. q.Wait is ignored. A page that MaxBytes cut short
// moves the cursor, so the events are collected across pages and a page is
// never asked for twice. The result spans pages when needed: Events holds
// every match, Cursor is the last page's, and Raw is nil then.
func (c *Client) WaitForEvents(ctx context.Context, q Query) (Page, error) {
	want, since := max(q.Min, 1), q.Since
	var (
		last   Page
		events []Event
		pages  int
	)
	for {
		q.Min = want - len(events)
		q.Wait = pollWait(ctx)
		start := time.Now()
		page, err := c.Events(ctx, q)
		switch {
		case ctx.Err() != nil:
			return collected(last, events, pages), fmt.Errorf(
				"waiting for %v since seq %d: %w (collected %d of %d events up to seq %d; "+
					"unfiltered after it: %d requests, %d events, %d control)",
				q.Event, since, ctx.Err(), len(events), want, q.Since,
				last.Unfiltered.Requests, last.Unfiltered.Events, last.Unfiltered.Control)
		case err != nil:
			return Page{}, err
		case oversized(page):
			return Page{}, fmt.Errorf("the request after seq %d is %d bytes, above MaxBytes: %w", page.Since,
				page.Truncated.RequestBytes, ErrOutputLimit)
		}
		last, events, pages = page, append(events, page.Events...), pages+1
		if len(events) >= want {
			return collected(last, events, pages), nil
		}
		if len(page.Events) == 0 {
			// No progress: space the polls, whatever made the server return.
			_ = sleepCtx(ctx, clientRetryDelay-time.Since(start))
		}
		q.Since = max(q.Since, page.Cursor)
	}
}

// oversized is a page whose first request alone is above MaxBytes: it holds
// no event and does not move the cursor.
func oversized(p Page) bool {
	return p.Truncated != nil && p.Truncated.RequestBytes > 0 && len(p.Events) == 0
}

func collected(last Page, events []Event, pages int) Page {
	if pages > 1 {
		last.Raw, last.Truncated = nil, nil
	}
	last.Events = events
	return last
}

// sleepCtx sleeps d, or less when ctx ends first; d <= 0 returns at once.
func sleepCtx(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// pollWait ends a poll just before ctx does, so the last page still arrives.
func pollWait(ctx context.Context) time.Duration {
	deadline, ok := ctx.Deadline()
	if !ok {
		return clientPollWait
	}
	return max(min(time.Until(deadline)-100*time.Millisecond, clientPollWait), time.Millisecond)
}

// Record is one captured request (contract section 3), decoded in part.
// Raw holds the full record as the server sent it.
type Record struct {
	Seq        uint64     `json:"seq"`
	Kind       string     `json:"kind"`
	Probe      bool       `json:"probe"`
	ReceivedAt time.Time  `json:"receivedAt"`
	Route      string     `json:"route"`
	Transport  string     `json:"transport"`
	StatusCode int        `json:"statusCode"`
	Failed     bool       `json:"failed"`
	Outcome    string     `json:"outcome"`
	WriteKey   string     `json:"writeKey"`
	Rejection  *Rejection `json:"rejection"`
	Events     []struct {
		Type  string `json:"type"`
		Event string `json:"event"`
	} `json:"events"`
	// Truncated is set, and every other field is zero, when the record is
	// larger than the requested maxBytes.
	Truncated *Truncated      `json:"truncated"`
	Raw       json.RawMessage `json:"-"`
}

func (r *Record) UnmarshalJSON(b []byte) error {
	type plain Record
	var p plain
	if err := json.Unmarshal(b, &p); err != nil {
		return fmt.Errorf("decoding record: %w", err)
	}
	*r = Record(p)
	r.Raw = append(json.RawMessage(nil), b...)
	return nil
}

type Rejection struct {
	Stage  string `json:"stage"`
	Reason string `json:"reason"`
}

// RequestPage is the /requests envelope.
type RequestPage struct {
	APIVersion string     `json:"apiVersion"`
	ServerID   string     `json:"serverId"`
	Since      uint64     `json:"since"`
	Cursor     uint64     `json:"cursor"`
	HasMore    bool       `json:"hasMore"`
	TimedOut   bool       `json:"timedOut"`
	Unfiltered Unfiltered `json:"unfiltered"`
	View       View       `json:"view"`
	Omitted    *Omitted   `json:"omitted"`
	Truncated  *Truncated `json:"truncated"`
	Requests   []Record   `json:"requests"`
	raw
}

func (c *Client) Requests(ctx context.Context, q RequestQuery) (RequestPage, error) {
	var page RequestPage
	if err := c.do(ctx, http.MethodGet, "/_dev/v1/requests", q.Values(), &page); err != nil {
		return RequestPage{}, err
	}
	return page, nil
}

// Request reads one whole record. It returns ErrNotFound for a seq the
// store does not hold.
func (c *Client) Request(ctx context.Context, seq uint64, q RecordQuery) (Record, error) {
	var rec Record
	path := "/_dev/v1/requests/" + strconv.FormatUint(seq, 10)
	if err := c.do(ctx, http.MethodGet, path, q.Values(), &rec); err != nil {
		return Record{}, err
	}
	return rec, nil
}

// Summary is the /summary object (contract section 4.7).
type Summary struct {
	APIVersion string `json:"apiVersion"`
	ServerID   string `json:"serverId"`
	Since      uint64 `json:"since"`
	Cursor     uint64 `json:"cursor"`
	Requests   struct {
		Total        int            `json:"total"`
		Failed       int            `json:"failed"`
		Probes       int            `json:"probes"`
		ByRoute      map[string]int `json:"byRoute"`
		ByStatusCode map[string]int `json:"byStatusCode"`
		ByOutcome    map[string]int `json:"byOutcome"`
		ByStage      map[string]int `json:"byStage"`
	} `json:"requests"`
	Events struct {
		Total   int            `json:"total"`
		ByType  map[string]int `json:"byType"`
		ByEvent map[string]int `json:"byEvent"`
	} `json:"events"`
	Control struct {
		Total              int `json:"total"`
		SourceConfig       int `json:"sourceConfig"`
		SourceConfigFailed int `json:"sourceConfigFailed"`
		Preflight          int `json:"preflight"`
		PluginPath         int `json:"pluginPath"`
		Other              int `json:"other"`
	} `json:"control"`
	BySource struct {
		ByChannel map[string]int       `json:"byChannel"`
		BySdk     map[string]SDKCounts `json:"bySdk"`
	} `json:"bySource"`
	Expected  []Expected  `json:"expected"`
	Diagnosis []Diagnosis `json:"diagnosis"`
	raw
}

// SDKCounts are the requests, events and control requests of one SDK
// family: browser, node, go, mobile or other.
type SDKCounts struct {
	Requests int `json:"requests"`
	Events   int `json:"events"`
	Control  int `json:"control"`
}

// Expected is the check of one SummaryQuery.Expect value. Want is nil when
// no count was given. Status is present, missing or count_mismatch.
type Expected struct {
	Event  string `json:"event"`
	Want   *int   `json:"want"`
	Got    int    `json:"got"`
	Status string `json:"status"`
}

type Diagnosis struct {
	Code    string   `json:"code"`
	Count   int      `json:"count"`
	Message string   `json:"message"`
	Events  []string `json:"events"`
	Next    string   `json:"next"`
}

func (c *Client) Summary(ctx context.Context, q SummaryQuery) (Summary, error) {
	var s Summary
	if err := c.do(ctx, http.MethodGet, "/_dev/v1/summary", q.Values(), &s); err != nil {
		return Summary{}, err
	}
	return s, nil
}

type ResetResult struct {
	Cursor  uint64     `json:"cursor"`
	Removed StoreStats `json:"removed"`
	raw
}

// Reset clears the store. seq keeps counting.
func (c *Client) Reset(ctx context.Context) (ResetResult, error) {
	var res ResetResult
	if err := c.do(ctx, http.MethodPost, "/_dev/v1/reset", nil, &res); err != nil {
		return ResetResult{}, err
	}
	return res, nil
}

type ShutdownResult struct {
	Stopping bool   `json:"stopping"`
	ServerID string `json:"serverId"`
	raw
}

// Shutdown asks the server to stop. It returns once the server accepted;
// the server drains after that.
func (c *Client) Shutdown(ctx context.Context) (ShutdownResult, error) {
	var res ShutdownResult
	if err := c.do(ctx, http.MethodPost, "/_dev/v1/shutdown", nil, &res); err != nil {
		return ShutdownResult{}, err
	}
	return res, nil
}

// Probe is the event Send posts. Zero fields take the defaults: event
// "probe", type "track", userId "dev-send". WriteKey is sent as given, so
// an empty key tests the auth rejection; use DefaultWriteKey otherwise.
type Probe struct {
	Event     string
	Type      string
	UserID    string
	WriteKey  string
	UserAgent string
}

type SendResult struct {
	StatusCode int    `json:"statusCode"`
	Body       string `json:"body"`
	Seq        uint64 `json:"seq"`
	Route      string `json:"route"`
}

// Send posts one probe event through the ingestion path, like an SDK, and
// finds the seq it was captured under. A non-2xx answer is not an error;
// the caller reads StatusCode.
func (c *Client) Send(ctx context.Context, p Probe) (SendResult, error) {
	p = p.withDefaults()
	info, err := c.Info(ctx)
	if err != nil {
		return SendResult{}, err
	}
	route := "/v1/" + p.Type
	body, _ := json.Marshal(map[string]any{
		"type": p.Type, "event": p.Event, "userId": p.UserID, "messageId": newMessageID(),
		"properties": map[string]any{"sentBy": "rudder-cli"},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+route, bytes.NewReader(body))
	if err != nil {
		return SendResult{}, fmt.Errorf("building probe: %w", err)
	}
	req.SetBasicAuth(p.WriteKey, "")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", p.UserAgent)
	resp, err := c.http.Do(req)
	if err != nil {
		return SendResult{}, fmt.Errorf("sending probe: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	res := SendResult{StatusCode: resp.StatusCode, Body: strings.TrimSpace(string(respBody)), Route: route}
	page, err := c.Requests(ctx, RequestQuery{Since: info.Cursor, Kind: "all", Route: []string{route}, View: ViewFull,
		MaxBytes: MaxBytesOff})
	if err != nil {
		return res, err
	}
	for _, rec := range page.Requests {
		if rec.Probe {
			res.Seq = rec.Seq
			break
		}
	}
	return res, nil
}

func (p Probe) withDefaults() Probe {
	def := func(s *string, v string) {
		if *s == "" {
			*s = v
		}
	}
	def(&p.Event, "probe")
	def(&p.Type, "track")
	def(&p.UserID, "dev-send")
	def(&p.UserAgent, "rudder-cli dev send/dev")
	return p
}

func newMessageID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func (c *Client) do(ctx context.Context, method, path string, values url.Values, out any) error {
	var body io.Reader
	if method == http.MethodPost {
		body = strings.NewReader("{}")
	}
	req, err := http.NewRequestWithContext(ctx, method, c.endpoint(path, values), body)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("requesting %s: %w", path, err)
	}
	defer resp.Body.Close()
	return decodeResponse(path, resp, out)
}

// endpoint adds the pinned serverId to every query except /info, which is
// how a client learns the serverId in the first place, and the POST routes.
func (c *Client) endpoint(path string, values url.Values) string {
	if values == nil {
		values = url.Values{}
	}
	if c.serverID != "" && takesServerID(path) {
		values.Set("serverId", c.serverID)
	}
	if len(values) == 0 {
		return c.baseURL + path
	}
	return c.baseURL + path + "?" + values.Encode()
}

func takesServerID(path string) bool {
	switch path {
	case "/_dev/v1/info", "/_dev/v1/reset", "/_dev/v1/shutdown":
		return false
	}
	return strings.HasPrefix(path, "/_dev/v1/")
}

func decodeResponse(path string, resp *http.Response, out any) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return decodeAPIError(resp.StatusCode, body)
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decoding %s: %w", path, err)
	}
	if r, ok := out.(interface{ setRaw([]byte) }); ok {
		r.setRaw(body)
	}
	return nil
}

func decodeAPIError(status int, body []byte) error {
	var wrapper struct {
		Error *APIError `json:"error"`
	}
	if json.Unmarshal(body, &wrapper) != nil || wrapper.Error == nil {
		return &APIError{StatusCode: status, Code: "unexpected_response", Message: strings.TrimSpace(string(body))}
	}
	wrapper.Error.StatusCode = status
	return wrapper.Error
}
