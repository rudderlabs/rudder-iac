package client_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/api/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientEmptyAccessToken(t *testing.T) {
	_, err := client.New("")
	assert.Equal(t, client.ErrEmptyAccessToken, err, "error should be ErrEmptyAccessToken")
}

func TestClientURL(t *testing.T) {
	c, err := client.New("some-access-token")
	assert.NoError(t, err)
	assert.Equal(t, "https://api.rudderstack.com", c.URL(""))
	assert.Equal(t, "https://api.rudderstack.com/path", c.URL("path"))
	assert.Equal(t, "https://api.rudderstack.com/path", c.URL("/path"))
	assert.Equal(t, "https://api.rudderstack.com/path/more", c.URL("/path/more"))
}

func TestClientDoRetriesSafeMethodTransportErrors(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t,
		testutils.Call{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/path", "")
			},
			ResponseError: errors.New("read: connection reset by peer"),
		},
		testutils.Call{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/path", "")
			},
			ResponseStatus: http.StatusOK,
			ResponseBody:   "ok",
		},
	)

	c, err := client.New("some-access-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	data, err := c.Do(context.Background(), http.MethodGet, "path", nil)
	require.NoError(t, err)
	assert.Equal(t, "ok", string(data))
	httpClient.AssertNumberOfCalls()
}

func TestClientDoDoesNotRetryMutatingMethodTransportErrors(t *testing.T) {
	transportErr := errors.New("read: connection reset by peer")
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return testutils.ValidateRequest(t, req, "POST", "https://api.rudderstack.com/path", "")
		},
		ResponseError: transportErr,
	})

	c, err := client.New("some-access-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	data, err := c.Do(context.Background(), http.MethodPost, "path", nil)
	require.ErrorIs(t, err, transportErr)
	assert.Nil(t, data)
	httpClient.AssertNumberOfCalls()
}

func TestClientDoDoesNotRetryNonTransientSafeMethodErrors(t *testing.T) {
	transportErr := errors.New("HTTP error")
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/path", "")
		},
		ResponseError: transportErr,
	})

	c, err := client.New("some-access-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	data, err := c.Do(context.Background(), http.MethodGet, "path", nil)
	require.ErrorIs(t, err, transportErr)
	assert.Nil(t, data)
	httpClient.AssertNumberOfCalls()
}

func TestClientDoRetriesRateLimitedWrite(t *testing.T) {
	const body = `{"name":"some-name","description":"some description"}`

	var (
		mu     sync.Mutex
		bodies []string
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, err := io.ReadAll(r.Body)
		assert.NoError(t, err)

		mu.Lock()
		bodies = append(bodies, string(data))
		attempt := len(bodies)
		mu.Unlock()

		if attempt == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	c, err := client.New("some-access-token", client.WithBaseURL(server.URL))
	require.NoError(t, err)

	data, err := c.Do(context.Background(), http.MethodPost, "path", bytes.NewReader([]byte(body)))
	require.NoError(t, err)
	assert.Equal(t, "ok", string(data))

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, []string{body, body}, bodies)
}

func TestClientDoRateLimitWaitHonoursContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "3")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	c, err := client.New("some-access-token", client.WithBaseURL(server.URL))
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(200*time.Millisecond, cancel)
	defer cancel()

	start := time.Now()
	data, err := c.Do(ctx, http.MethodPost, "path", bytes.NewReader([]byte(`{}`)))
	require.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, data)
	assert.Less(t, time.Since(start), time.Second)
}

func TestClientDoReturnsRateLimitErrorWhenWaitOutlastsDeadline(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "3")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	c, err := client.New("some-access-token", client.WithBaseURL(server.URL))
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	start := time.Now()
	data, err := c.Do(ctx, http.MethodPost, "path", bytes.NewReader([]byte(`{}`)))
	assert.Equal(t, &client.APIError{HTTPStatusCode: http.StatusTooManyRequests}, err)
	assert.Nil(t, data)
	assert.Less(t, time.Since(start), 500*time.Millisecond)
}

func TestClientDoReturnsRateLimitErrorWhenRetriesRunOut(t *testing.T) {
	var (
		mu    sync.Mutex
		calls int
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		calls++
		mu.Unlock()

		w.Header().Set("Retry-After", "0")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	c, err := client.New("some-access-token", client.WithBaseURL(server.URL))
	require.NoError(t, err)

	data, err := c.Do(context.Background(), http.MethodPut, "path", bytes.NewReader([]byte(`{}`)))
	assert.Equal(t, &client.APIError{HTTPStatusCode: http.StatusTooManyRequests}, err)
	assert.Nil(t, data)

	mu.Lock()
	defer mu.Unlock()
	// a near-zero wait is retried at most 100 times: the first call plus 100 retries
	assert.Equal(t, 101, calls)
}

func TestClientDoGetsConcurrentWritersThroughASlidingWindowLimit(t *testing.T) {
	const (
		workers = 30
		items   = 60
		limit   = 8
		window  = time.Second
	)

	var (
		mu       sync.Mutex
		accepted []time.Time
	)
	// admits limit requests per sliding window and, like the server's limiter, asks for a full window's wait
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		now := time.Now()
		recent := accepted[:0]
		for _, at := range accepted {
			if now.Sub(at) < window {
				recent = append(recent, at)
			}
		}
		accepted = recent
		if len(accepted) >= limit {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		accepted = append(accepted, now)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	c, err := client.New("some-access-token", client.WithBaseURL(server.URL))
	require.NoError(t, err)

	// workers pull from a shared queue, as apply does, so a worker that just succeeded sends again at once and can
	// keep taking free slots from a call that lost an early round
	queue := make(chan int, items)
	for i := range items {
		queue <- i
	}
	close(queue)

	errs := make([]error, items)
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range queue {
				_, errs[i] = c.Do(context.Background(), http.MethodPost, "path", bytes.NewReader([]byte(`{}`)))
			}
		}()
	}
	wg.Wait()

	assert.Equal(t, make([]error, items), errs)
}

type closeTrackingReader struct {
	io.Reader
	closed bool
}

func (r *closeTrackingReader) Close() error {
	r.closed = true
	return nil
}

func TestClientDoClosesRequestBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	c, err := client.New("some-access-token", client.WithBaseURL(server.URL))
	require.NoError(t, err)

	body := &closeTrackingReader{Reader: bytes.NewReader([]byte(`{}`))}
	_, err = c.Do(context.Background(), http.MethodPost, "path", body)
	require.NoError(t, err)
	assert.True(t, body.closed)
}
