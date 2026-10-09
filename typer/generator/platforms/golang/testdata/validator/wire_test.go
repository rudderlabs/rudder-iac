package validator

import (
	"encoding/json"
	"io"
	"maps"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	analytics "github.com/rudderlabs/analytics-go/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/typer/generator/platforms/golang/testdata/validator/examples"
	"github.com/rudderlabs/rudder-iac/typer/generator/platforms/golang/testdata/validator/ruddertyper"
)

// recorder is both the data plane and the Config.Callback of a capture.
type recorder struct {
	t         *testing.T
	mu        sync.Mutex
	batches   [][]map[string]any
	succeeded []analytics.Message
}

func (r *recorder) ServeHTTP(_ http.ResponseWriter, req *http.Request) {
	var body struct {
		Batch []map[string]any `json:"batch"`
	}
	assert.NoError(r.t, decode(req.Body, &body))
	r.mu.Lock()
	defer r.mu.Unlock()
	r.batches = append(r.batches, body.Batch)
}

func (r *recorder) Success(msg analytics.Message) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.succeeded = append(r.succeeded, msg)
}

func (r *recorder) Failure(msg analytics.Message, err error) {
	r.t.Errorf("SDK failed to send %#v: %v", msg, err)
}

// capture runs send against a real SDK client and returns the messages of the
// one batch the SDK sent, in send order, and the messages it handed to
// Config.Callback.
func capture(t *testing.T, send func(analytics.Client)) ([]map[string]any, []analytics.Message) {
	t.Helper()
	var (
		rec = &recorder{t: t}
		srv = httptest.NewServer(rec)
	)
	defer srv.Close()

	client, err := analytics.NewWithConfig("write-key", analytics.Config{
		DataPlaneUrl: srv.URL,
		DisableGzip:  true,
		// Only Close flushes while a capture stays under the default ~500 KB
		// MaxBatchBytes, so every call lands in one batch in send order;
		// smaller batches are sent concurrently and can arrive in any order.
		BatchSize: 1000,
		Interval:  time.Hour,
		Callback:  rec,
	})
	require.NoError(t, err)
	send(client)
	require.NoError(t, client.Close())

	rec.mu.Lock()
	defer rec.mu.Unlock()
	require.Len(t, rec.batches, 1)
	return rec.batches[0], rec.succeeded
}

// assertMessage compares got whole with wantJSON, a JSON object. The SDK-filled
// messageId, originalTimestamp, sentAt, anonymousId and context.library are
// ignored unless wantJSON asserts them, so WithMessageID and WithTimestamp stay
// testable.
func assertMessage(t *testing.T, wantJSON string, got map[string]any) {
	t.Helper()
	var want map[string]any
	require.NoError(t, decode(strings.NewReader(wantJSON), &want))

	got = maps.Clone(got)
	for _, key := range []string{"messageId", "originalTimestamp", "sentAt", "anonymousId"} {
		if _, ok := want[key]; !ok {
			delete(got, key)
		}
	}
	wantContext, _ := want["context"].(map[string]any)
	_, wantLibrary := wantContext["library"]
	if gotContext, ok := got["context"].(map[string]any); ok && !wantLibrary {
		gotContext = maps.Clone(gotContext)
		delete(gotContext, "library")
		got["context"] = gotContext
	}
	assert.Equal(t, want, got)
}

// decode keeps numbers as their JSON text, so an int64 beyond 2^53 cannot
// match a rounded one.
func decode(r io.Reader, v any) error {
	d := json.NewDecoder(r)
	d.UseNumber()
	return d.Decode(v)
}

// assertPlain fails on any value in v that a snapshot must not hold, such as
// an int or a generated type.
func assertPlain(t *testing.T, v any) {
	t.Helper()
	switch v := v.(type) {
	case nil, string, bool, int64, float64:
	case map[string]any:
		for _, e := range v {
			assertPlain(t, e)
		}
	case []any:
		for _, e := range v {
			assertPlain(t, e)
		}
	default:
		t.Errorf("snapshot holds %T %v", v, v)
	}
}

// The context.ruddertyper objects of the examples and reference goldens.
const (
	exMeta  = `{"platform": "go", "rudderCLIVersion": "1.0.0", "trackingPlanId": "plan_examples", "trackingPlanVersion": 0}`
	refMeta = `{"platform": "go", "rudderCLIVersion": "1.0.0", "trackingPlanId": "plan_12345", "trackingPlanVersion": 13}`
)

func TestTrackWire(t *testing.T) {
	var (
		ex        *examples.RudderTyperAnalytics
		ref       *ruddertyper.RudderTyperAnalytics
		id        = examples.Identity{UserID: "user-123"}
		refID     = ruddertyper.Identity{UserID: "user-123"}
		someTrack = examples.TrackSomeTrackEventProperties{SomeString: "hello", SomeInteger: 42, SomeBoolean: examples.Ptr(false)}
		closed    = analytics.New("write-key", "http://127.0.0.1")
	)
	require.NoError(t, closed.Close())

	// The rows share one capture, which sets ex and ref when it starts, so the
	// batch holds one message per row that succeeds, in row order.
	tests := []struct {
		name    string
		call    func() error
		want    string
		err     error
		payload any // when set, json.Marshal(payload) must equal the properties sent
	}{
		{
			name:    "a set optional field sends its value, even false",
			call:    func() error { return ex.TrackSomeTrackEvent(id, someTrack) },
			payload: someTrack,
			want: `{"type": "track", "channel": "server", "event": "Some Track Event", "userId": "user-123",
				"properties": {"someString": "hello", "someInteger": 42, "someBoolean": false},
				"context": {"ruddertyper": ` + exMeta + `}}`,
		},
		{
			name: "a nil optional field omits its key and an unset required field sends its zero value",
			call: func() error { return ex.TrackSomeTrackEvent(id, examples.TrackSomeTrackEventProperties{}) },
			want: `{"type": "track", "channel": "server", "event": "Some Track Event", "userId": "user-123",
				"properties": {"someString": "", "someInteger": 0},
				"context": {"ruddertyper": ` + exMeta + `}}`,
		},
		{
			name: "an empty closed event sends no properties",
			call: func() error { return ref.TrackEmptyEventNoAdditionalProps(refID) },
			want: `{"type": "track", "channel": "server", "event": "Empty Event No Additional Props", "userId": "user-123",
				"context": {"ruddertyper": ` + refMeta + `}}`,
		},
		{
			name: "an empty open event sends no properties for nil",
			call: func() error { return ref.TrackEmptyEventWithAdditionalProps(refID, nil) },
			want: `{"type": "track", "channel": "server", "event": "Empty Event With Additional Props", "userId": "user-123",
				"context": {"ruddertyper": ` + refMeta + `}}`,
		},
		{
			name: "an empty open event sends no properties for an empty map",
			call: func() error { return ref.TrackEmptyEventWithAdditionalProps(refID, map[string]any{}) },
			want: `{"type": "track", "channel": "server", "event": "Empty Event With Additional Props", "userId": "user-123",
				"context": {"ruddertyper": ` + refMeta + `}}`,
		},
		{
			name: "an empty open event sends a non-empty map, with nil slices and maps inside it as [] and {}",
			call: func() error {
				return ref.TrackEmptyEventWithAdditionalProps(refID, map[string]any{
					"plan":   "pro",
					"nested": map[string]any{"list": []string(nil), "object": map[string]any(nil)},
				})
			},
			want: `{"type": "track", "channel": "server", "event": "Empty Event With Additional Props", "userId": "user-123",
				"properties": {"plan": "pro", "nested": {"list": [], "object": {}}},
				"context": {"ruddertyper": ` + refMeta + `}}`,
		},
		{
			// productId is required, so the properties are never empty.
			name: "the open rule's struct with nothing set sends only its required key",
			call: func() error { return ex.TrackSomeOpenTrackEvent(id, examples.TrackSomeOpenTrackEventProperties{}) },
			want: `{"type": "track", "channel": "server", "event": "Some Open Track Event", "userId": "user-123",
				"properties": {"productId": ""},
				"context": {"ruddertyper": ` + exMeta + `}}`,
		},
		{
			name: "amount as a number sends under amount, and AdditionalProperties sends undeclared keys as plain values, even ints and generated types, and ignores declared ones",
			call: func() error {
				return ex.TrackSomeOpenTrackEvent(id, examples.TrackSomeOpenTrackEventProperties{
					ProductID: "p1",
					Amount:    examples.Ptr(9.5),
					AdditionalProperties: map[string]any{
						"coupon": "SAVE", "productId": "override", "productName": "fill",
						"count": 3, "user": examples.TrackUserProperties{User: examples.Ptr("u")},
					},
				})
			},
			want: `{"type": "track", "channel": "server", "event": "Some Open Track Event", "userId": "user-123",
				"properties": {"productId": "p1", "amount": 9.5, "coupon": "SAVE", "count": 3, "user": {"user": "u"}},
				"context": {"ruddertyper": ` + exMeta + `}}`,
		},
		{
			name: "the collision pair's first event sends its own name",
			call: func() error {
				return ex.TrackEventWithNameCamelCase(id, examples.TrackEventWithNameCamelCaseProperties{})
			},
			want: `{"type": "track", "channel": "server", "event": "$eventWithNameCamelCase$!", "userId": "user-123",
				"context": {"ruddertyper": ` + exMeta + `}}`,
		},
		{
			name: "the collision pair's second event sends its own name",
			call: func() error {
				return ex.TrackEventWithNameCamelCase1(id, examples.TrackEventWithNameCamelCaseProperties1{})
			},
			want: `{"type": "track", "channel": "server", "event": "eventWithNameCamelCase", "userId": "user-123",
				"context": {"ruddertyper": ` + exMeta + `}}`,
		},
		{
			name: "amount as a string and the other colliding names send under their plan keys",
			call: func() error {
				return ex.TrackNameCollisions(id, examples.TrackNameCollisionsProperties{
					X1stPlace:     examples.Ptr[int64](1),
					Amount:        "ten",
					ToProperties1: examples.Ptr("t"),
					UserID:        "u1",
					UserID1:       examples.Ptr("u2"),
				})
			},
			want: `{"type": "track", "channel": "server", "event": "Name Collisions", "userId": "user-123",
				"properties": {"1st_place": 1, "amount": "ten", "toProperties": "t", "userId": "u1", "user_id": "u2"},
				"context": {"ruddertyper": ` + exMeta + `}}`,
		},
		{
			name: "options send integrations, originalTimestamp and messageId",
			call: func() error {
				return ex.TrackSomeEmptyTrackEvent(id,
					examples.WithIntegrations(analytics.Integrations{"All": false, "Amplitude": true}),
					examples.WithTimestamp(time.Date(2026, 9, 29, 10, 0, 0, 0, time.UTC)),
					examples.WithMessageID("msg-1"),
				)
			},
			want: `{"type": "track", "channel": "server", "event": "Some Empty Track Event", "userId": "user-123",
				"integrations": {"All": false, "Amplitude": true},
				"originalTimestamp": "2026-09-29T10:00:00Z",
				"messageId": "msg-1",
				"context": {"ruddertyper": ` + exMeta + `}}`,
		},
		{
			name: "WithAnalyticsContext keeps caller keys and ruddertyper wins",
			call: func() error {
				return ex.TrackSomeEmptyTrackEvent(id, examples.WithAnalyticsContext(analytics.Context{
					Locale: "en-US",
					Traits: analytics.Traits{"plan": "pro"},
					Extra:  map[string]any{"custom": 1, "ruddertyper": "overwritten"},
				}))
			},
			want: `{"type": "track", "channel": "server", "event": "Some Empty Track Event", "userId": "user-123",
				"context": {"locale": "en-US", "traits": {"plan": "pro"}, "custom": 1, "ruddertyper": ` + exMeta + `}}`,
		},
		{
			name: "a later option overrides an earlier one and nil options are skipped",
			call: func() error {
				return ex.TrackSomeEmptyTrackEvent(id,
					examples.WithMessageID("first"), nil, examples.WithMessageID("second"),
					examples.WithIntegrations(analytics.Integrations{"All": false}),
					examples.WithIntegrations(analytics.Integrations{"Amplitude": true}),
				)
			},
			want: `{"type": "track", "channel": "server", "event": "Some Empty Track Event", "userId": "user-123",
				"messageId": "second",
				"integrations": {"Amplitude": true},
				"context": {"ruddertyper": ` + exMeta + `}}`,
		},
		{
			name: "no identity returns the SDK's FieldError",
			call: func() error { return ex.TrackSomeEmptyTrackEvent(examples.Identity{}) },
			err:  analytics.FieldError{Type: "analytics.Track", Name: "UserId", Value: ""},
		},
		{
			name: "a closed client returns the SDK's ErrClosed",
			call: func() error { return examples.New(closed).TrackSomeEmptyTrackEvent(id) },
			err:  analytics.ErrClosed,
		},
		{
			name: "a self-containing map is invalid",
			call: func() error {
				m := map[string]any{}
				m["self"] = m
				return ex.TrackSomeEmptyTrackEventWithAdditionalProperties(id, m)
			},
			err: examples.ErrInvalidValue,
		},
		{
			name: "NaN in the payload is invalid",
			call: func() error {
				return ex.TrackSomeOpenTrackEvent(id, examples.TrackSomeOpenTrackEventProperties{Amount: examples.Ptr(math.NaN())})
			},
			err: examples.ErrInvalidValue,
		},
		{
			name: "NaN in WithIntegrations is invalid",
			call: func() error {
				return ex.TrackSomeEmptyTrackEvent(id, examples.WithIntegrations(analytics.Integrations{"x": math.NaN()}))
			},
			err: examples.ErrInvalidValue,
		},
		{
			name: "NaN in WithAnalyticsContext is invalid",
			call: func() error {
				return ex.TrackSomeEmptyTrackEvent(id, examples.WithAnalyticsContext(analytics.Context{Extra: map[string]any{"x": math.NaN()}}))
			},
			err: examples.ErrInvalidValue,
		},
		{
			name: "a WithTimestamp after year 9999 is invalid",
			call: func() error {
				return ex.TrackSomeEmptyTrackEvent(id, examples.WithTimestamp(time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)))
			},
			err: examples.ErrInvalidValue,
		},
	}

	errs := make([]error, len(tests))
	wire, succeeded := capture(t, func(client analytics.Client) {
		ex, ref = examples.New(client), ruddertyper.New(client)
		for i, tt := range tests {
			errs[i] = tt.call()
		}
	})

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err != nil {
				assert.ErrorIs(t, errs[i], tt.err)
				return
			}
			require.NoError(t, errs[i])
			require.NotEmpty(t, wire)
			msg := wire[0]
			wire = wire[1:]
			assertMessage(t, tt.want, msg)
			if tt.payload != nil {
				want, err := json.Marshal(tt.payload)
				require.NoError(t, err)
				got, err := json.Marshal(msg["properties"])
				require.NoError(t, err)
				assert.JSONEq(t, string(want), string(got))
			}
		})
	}
	assert.Empty(t, wire, "a call that failed sent a message")

	// Config.Callback receives the snapshot itself, so its Go value types show
	// what the wire JSON cannot.
	for _, msg := range succeeded {
		assertPlain(t, map[string]any(msg.(analytics.Track).Properties))
	}
}
