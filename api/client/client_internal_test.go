package client

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// skipWaits makes sleep return at once (still honouring the context), so tests can cover long waits quickly.
// Callers must not be parallel, because sleep is package-level.
func skipWaits(t *testing.T) {
	t.Helper()
	orig := sleep
	sleep = func(ctx context.Context, _ time.Duration) error { return ctx.Err() }
	t.Cleanup(func() { sleep = orig })
}

// rateLimitedServer answers 429 with the given Retry-After for the first rejections calls, then 200.
func rateLimitedServer(t *testing.T, retryAfter string, rejections int32) (*Client, *atomic.Int32) {
	t.Helper()
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) <= rejections {
			w.Header().Set("Retry-After", retryAfter)
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(server.Close)

	c, err := New("some-access-token", WithBaseURL(server.URL))
	require.NoError(t, err)
	return c, &calls
}

func TestClientDoKeepsRetryingRealWaitsPastTheImmediateRetryCap(t *testing.T) {
	skipWaits(t)
	// a call that keeps losing full-window rounds, well past the cap that only applies to near-zero waits
	c, calls := rateLimitedServer(t, "60", 3*maxImmediateRetries)

	data, err := c.Do(context.Background(), http.MethodPost, "path", bytes.NewReader([]byte(`{}`)))
	require.NoError(t, err)
	assert.Equal(t, "ok", string(data))
	assert.Equal(t, int32(3*maxImmediateRetries+1), calls.Load())
}

func TestClientDoRetriesAWaitThatFitsBeforeTheDeadlineDespiteJitter(t *testing.T) {
	skipWaits(t)
	// jitter is random, so repeat: before the fix most runs returned the 429 instead of retrying
	for range 20 {
		c, calls := rateLimitedServer(t, "60", 1)
		// the 60s wait fits, but 60s plus up to 6s of jitter usually would not
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second+500*time.Millisecond)

		data, err := c.Do(ctx, http.MethodPost, "path", bytes.NewReader([]byte(`{}`)))
		cancel()
		require.NoError(t, err)
		assert.Equal(t, "ok", string(data))
		assert.Equal(t, int32(2), calls.Load())
	}
}

func TestRetryAfterDelay(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name  string
		value string
		retry int
		want  time.Duration
	}{
		{name: "seconds", value: "2", want: 2 * time.Second},
		{name: "zero", value: "0", want: 0},
		{name: "minute-long window", value: "60", want: time.Minute},
		{name: "seconds above cap", value: "3600", want: maxRateLimitWait},
		{name: "seconds that overflow a duration", value: "9223372036854775807", want: maxRateLimitWait},
		{name: "negative", value: "-1", want: baseRateLimitWait},
		{name: "missing", value: "", want: baseRateLimitWait},
		{name: "invalid", value: "soon", want: baseRateLimitWait},
		{name: "missing on a later retry grows", value: "", retry: 4, want: 32 * time.Second},
		{name: "invalid on a later retry grows", value: "soon", retry: 2, want: 8 * time.Second},
		{name: "missing growth is capped", value: "", retry: 10, want: maxRateLimitWait},
		{name: "missing growth stays capped after many retries", value: "", retry: 1000, want: maxRateLimitWait},
		{name: "header ignores retry", value: "2", retry: 4, want: 2 * time.Second},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, retryAfterDelay(tc.value, tc.retry))
		})
	}
}
