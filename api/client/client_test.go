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

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	start := time.Now()
	data, err := c.Do(ctx, http.MethodPost, "path", bytes.NewReader([]byte(`{}`)))
	require.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Nil(t, data)
	assert.Less(t, time.Since(start), time.Second)
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
	assert.Equal(t, 6, calls)
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
