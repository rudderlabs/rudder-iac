package validator

import (
	"encoding/json"
	"io"
	"maps"
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
		// Only Close flushes, so every call lands in one batch in send order;
		// smaller batches are sent concurrently and can arrive in any order.
		// The SDK also flushes early once a batch passes MaxBatchBytes
		// (~500 KB by default), so that limit is lifted too, to the 4 MB
		// most that v4.3.0 and later accept.
		BatchSize:     1000,
		MaxBatchBytes: 4 << 20,
		Interval:      time.Hour,
		Callback:      rec,
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

func TestTrackSmoke(t *testing.T) {
	wire, succeeded := capture(t, func(client analytics.Client) {
		require.NoError(t, examples.New(client).TrackSomeTrackEvent(
			examples.Identity{UserID: "user-123"},
			examples.TrackSomeTrackEventProperties{SomeString: "hello", SomeInteger: 42, SomeBoolean: examples.Ptr(true)},
		))
	})

	require.Len(t, wire, 1)
	assertMessage(t, `{
		"type": "track",
		"channel": "server",
		"event": "Some Track Event",
		"userId": "user-123",
		"properties": {"someString": "hello", "someInteger": 42, "someBoolean": true},
		"context": {
			"ruddertyper": {
				"platform": "go",
				"rudderCLIVersion": "1.0.0",
				"trackingPlanId": "plan_examples",
				"trackingPlanVersion": 0
			}
		}
	}`, wire[0])
	// Callback receives the snapshot itself, so this checks its Go value types
	// (int64(42)), which the wire JSON cannot show.
	require.Len(t, succeeded, 1)
	require.IsType(t, analytics.Track{}, succeeded[0])
	assert.Equal(t, analytics.Properties{"someString": "hello", "someInteger": int64(42), "someBoolean": true}, succeeded[0].(analytics.Track).Properties)
}
