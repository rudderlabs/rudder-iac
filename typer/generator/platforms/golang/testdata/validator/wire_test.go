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
	t        *testing.T
	mu       sync.Mutex
	batches  [][]map[string]any
	callback []analytics.Message
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
	r.callback = append(r.callback, msg)
}

func (r *recorder) Failure(msg analytics.Message, err error) {
	r.t.Errorf("SDK failed to send %#v: %v", msg, err)
}

// capture runs send against a real SDK client and returns the messages of the
// one batch the SDK sent, in send order, and the messages it handed to
// Config.Callback.
func capture(t *testing.T, send func(analytics.Client)) ([]map[string]any, []analytics.Message) {
	t.Helper()
	rec := &recorder{t: t}
	srv := httptest.NewServer(rec)
	defer srv.Close()

	client, err := analytics.NewWithConfig("write-key", analytics.Config{
		DataPlaneUrl: srv.URL,
		DisableGzip:  true,
		// Only Close flushes, so every call lands in one batch in send order;
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
	return rec.batches[0], rec.callback
}

// assertMessage compares got whole with want, a JSON object. The SDK-filled
// messageId, originalTimestamp, sentAt, anonymousId and context.library are
// ignored unless want asserts them, so WithMessageID and WithTimestamp stay
// testable.
func assertMessage(t *testing.T, want string, got map[string]any) {
	t.Helper()
	var w map[string]any
	require.NoError(t, decode(strings.NewReader(want), &w))

	got = maps.Clone(got)
	for _, key := range []string{"messageId", "originalTimestamp", "sentAt", "anonymousId"} {
		if _, ok := w[key]; !ok {
			delete(got, key)
		}
	}
	wantContext, _ := w["context"].(map[string]any)
	_, wantLibrary := wantContext["library"]
	if gotContext, ok := got["context"].(map[string]any); ok && !wantLibrary {
		gotContext = maps.Clone(gotContext)
		delete(gotContext, "library")
		got["context"] = gotContext
	}
	assert.Equal(t, w, got)
}

// decode keeps numbers as their JSON text, so an int64 beyond 2^53 cannot
// match a rounded one.
func decode(r io.Reader, v any) error {
	d := json.NewDecoder(r)
	d.UseNumber()
	return d.Decode(v)
}

func TestTrackSmoke(t *testing.T) {
	msgs, sent := capture(t, func(client analytics.Client) {
		require.NoError(t, examples.New(client).TrackSomeTrackEvent(
			examples.Identity{UserID: "user-123"},
			examples.TrackSomeTrackEventProperties{SomeString: "hello", SomeInteger: 42, SomeBoolean: examples.Ptr(true)},
		))
	})

	require.Len(t, msgs, 1)
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
	}`, msgs[0])
	require.Len(t, sent, 1)
	assert.Equal(t, analytics.Properties{"someString": "hello", "someInteger": int64(42), "someBoolean": true}, sent[0].(analytics.Track).Properties)
}
