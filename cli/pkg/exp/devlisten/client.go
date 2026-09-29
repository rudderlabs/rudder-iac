package devlisten

import (
	"context"
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

// clientPollWait is the per-request wait Client.Wait uses while it loops.
const clientPollWait = 10 * time.Second

var (
	// ErrServerChanged matches a 409 server_changed: the serverId the client
	// sent is not the running server's.
	ErrServerChanged = errors.New("server changed")
	// ErrCaptureGap matches a 410 capture_gap. The tracer never evicts, so
	// it never answers 410.
	ErrCaptureGap = errors.New("capture gap")
)

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
	}
	return false
}

type Client struct {
	baseURL  string
	http     *http.Client
	serverID string
}

type ClientOption func(*Client)

func WithHTTPClient(hc *http.Client) ClientOption { return func(c *Client) { c.http = hc } }

// WithServerID pins the client to one server: every query carries serverId
// and a restarted server answers ErrServerChanged.
func WithServerID(id string) ClientOption { return func(c *Client) { c.serverID = id } }

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

// Info is the /info object (contract section 4.1).
type Info struct {
	Ready
	WriteKeys     []string   `json:"writeKeys"`
	RecordVersion int        `json:"recordVersion"`
	Store         StoreStats `json:"store"`
}

type StoreStats struct {
	Requests int `json:"requests"`
	Events   int `json:"events"`
	Control  int `json:"control"`
}

// Info reads /info. The first successful call pins the serverId when no
// WithServerID was given.
func (c *Client) Info(ctx context.Context) (Info, error) {
	var info Info
	if err := c.get(ctx, "/_dev/v1/info", nil, &info); err != nil {
		return Info{}, err
	}
	if c.serverID == "" {
		c.serverID = info.ServerID
	}
	return info, nil
}

// Query selects events. Wait is the server-side long-poll of one request
// (at most 110 s); zero asks for a snapshot.
type Query struct {
	Since uint64
	Limit int
	Event []string
	Type  []string
	Min   int
	Wait  time.Duration
}

// Values is the exact query string the client sends, without serverId.
func (q Query) Values() url.Values {
	v := url.Values{}
	if q.Since > 0 {
		v.Set("since", strconv.FormatUint(q.Since, 10))
	}
	if q.Limit > 0 {
		v.Set("limit", strconv.Itoa(q.Limit))
	}
	if q.Min > 0 {
		v.Set("min", strconv.Itoa(q.Min))
	}
	if q.Wait > 0 {
		v.Set("wait", q.Wait.String())
	}
	for _, e := range q.Event {
		v.Add("event", e)
	}
	for _, t := range q.Type {
		v.Add("type", t)
	}
	return v
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
	Events     []Event    `json:"events"`
}

type Unfiltered struct {
	Requests       int `json:"requests"`
	FailedRequests int `json:"failedRequests"`
	Events         int `json:"events"`
	Control        int `json:"control"`
}

// Event is one flattened event. String fields are "" when the wire value is
// null; Raw keeps the item bytes when that difference matters.
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
	if err := c.get(ctx, "/_dev/v1/events", q.Values(), &page); err != nil {
		return Page{}, err
	}
	return page, nil
}

// Wait loops 10 s long-polls until a page holds at least q.Min events
// (default 1) or ctx ends. q.Wait is ignored.
func (c *Client) Wait(ctx context.Context, q Query) (Page, error) {
	q.Min = max(q.Min, 1)
	for {
		q.Wait = pollWait(ctx)
		page, err := c.Events(ctx, q)
		switch {
		case ctx.Err() != nil:
			return page, fmt.Errorf("waiting for %v since seq %d: %w (unfiltered: %d requests, %d events, %d control)",
				q.Event, q.Since, ctx.Err(), page.Unfiltered.Requests, page.Unfiltered.Events, page.Unfiltered.Control)
		case err != nil:
			return Page{}, err
		case len(page.Events) >= q.Min:
			return page, nil
		}
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

func (c *Client) get(ctx context.Context, path string, values url.Values, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint(path, values), nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("requesting %s: %w", path, err)
	}
	defer resp.Body.Close()
	return decodeResponse(path, resp, out)
}

// endpoint adds the pinned serverId to every query except /info, which is
// how a client learns the serverId in the first place.
func (c *Client) endpoint(path string, values url.Values) string {
	if values == nil {
		values = url.Values{}
	}
	if c.serverID != "" && path != "/_dev/v1/info" {
		values.Set("serverId", c.serverID)
	}
	if len(values) == 0 {
		return c.baseURL + path
	}
	return c.baseURL + path + "?" + values.Encode()
}

func decodeResponse(path string, resp *http.Response, out any) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}
	if resp.StatusCode != http.StatusOK {
		return decodeAPIError(resp.StatusCode, body)
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decoding %s: %w", path, err)
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
