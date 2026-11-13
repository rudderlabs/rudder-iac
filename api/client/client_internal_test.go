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
		{name: "missing growth stays capped at the last retry", value: "", retry: maxRateLimitRetries, want: maxRateLimitWait},
		{name: "header ignores retry", value: "2", retry: 4, want: 2 * time.Second},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, retryAfterDelay(tc.value, tc.retry))
		})
	}
}

// not parallel: it shrinks the package-level budget
func TestClientDoStopsRetryingWhenBudgetRunsOut(t *testing.T) {
	budget := rateLimitBudget
	rateLimitBudget = 1500 * time.Millisecond
	t.Cleanup(func() { rateLimitBudget = budget })

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Retry-After", "1")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	c, err := New("some-access-token", WithBaseURL(server.URL))
	require.NoError(t, err)

	data, err := c.Do(context.Background(), http.MethodPost, "path", bytes.NewReader([]byte(`{}`)))
	assert.Equal(t, &APIError{HTTPStatusCode: http.StatusTooManyRequests}, err)
	assert.Nil(t, data)
	assert.Equal(t, int32(2), calls.Load())
}
