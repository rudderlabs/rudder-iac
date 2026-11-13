package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var ErrResourceNotFound = fmt.Errorf("resource not found")

type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

type Client struct {
	baseURL     string
	accessToken string
	userAgent   string
	httpClient  HTTPClient

	Sources      *sources
	Destinations *destinations
	Connections  *connections
	Accounts     *accounts
	Workspaces   *workspaces
}

const BASE_URL = "https://api.rudderstack.com"

var (
	ErrEmptyAccessToken  = fmt.Errorf("access token cannot be empty")
	ErrInvalidBaseURL    = fmt.Errorf("base url cannot be empty")
	ErrInvalidHTTPClient = fmt.Errorf("http client cannot be nil")
)

func New(accessToken string, options ...Option) (*Client, error) {
	client := &Client{
		baseURL:     BASE_URL,
		httpClient:  &http.Client{},
		accessToken: accessToken,
		userAgent:   "rudder-api-go/1.0.0",
	}

	client.Sources = &sources{service: client.service("/v2/sources")}
	client.Destinations = &destinations{service: client.service("/v2/destinations")}
	client.Connections = &connections{service: client.service("/v2/connections")}
	client.Accounts = &accounts{service: client.service("/v2/accounts")}
	client.Workspaces = &workspaces{client: client}

	for _, o := range options {
		if err := o(client); err != nil {
			return nil, err
		}
	}

	if client.accessToken == "" {
		return nil, ErrEmptyAccessToken
	}

	return client, nil
}

func (c *Client) URL(path string) string {
	if len(path) == 0 {
		return c.baseURL
	}

	return fmt.Sprintf("%s/%s", c.baseURL, strings.TrimPrefix(path, "/"))
}

func (c *Client) Do(ctx context.Context, method, path string, body io.Reader) ([]byte, error) {
	res, err := c.doRequest(ctx, method, path, body)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	// check if response has an error status code and parse API error accordingly
	if res.StatusCode < 200 || res.StatusCode > 299 {
		apiError := &APIError{HTTPStatusCode: res.StatusCode}
		if len(data) > 0 {
			err := json.Unmarshal(data, apiError)
			if err != nil {
				return nil, fmt.Errorf("could not parse error response from API: %w", err)
			}
		}

		return nil, apiError
	}

	return data, nil
}

const (
	maxRateLimitRetries = 5
	maxRateLimitWait    = 2 * time.Minute
	// header-less retries then wait about 62s in total, outlasting a minute-long limit window
	baseRateLimitWait = 2 * time.Second
)

// doRequest retries rate-limited responses for every method, as the server rejected them before acting on them.
func (c *Client) doRequest(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	var payload []byte
	if body != nil {
		var err error
		payload, err = io.ReadAll(body)
		// buffering detaches the body from the request, so the transport no longer closes it
		if closer, ok := body.(io.Closer); ok {
			_ = closer.Close()
		}
		if err != nil {
			return nil, fmt.Errorf("reading request body: %w", err)
		}
	}

	for retry := 0; ; retry++ {
		res, err := c.send(ctx, method, path, payload)
		if err != nil || res.StatusCode != http.StatusTooManyRequests || retry == maxRateLimitRetries {
			return res, err
		}

		wait := retryAfterDelay(res.Header.Get("Retry-After"), retry)
		// up to 10% jitter so concurrent callers told the same wait do not all retry at once; zero stays zero
		wait += rand.N(wait/10 + 1) //nolint:gosec // timing jitter, not a secret
		// a wait the context cannot outlast would only end in a deadline error, so surface the 429 instead
		if deadline, ok := ctx.Deadline(); ok && time.Until(deadline) < wait {
			return res, nil
		}

		_, _ = io.Copy(io.Discard, res.Body)
		_ = res.Body.Close()
		if err := sleep(ctx, wait); err != nil {
			return nil, err
		}
	}
}

func (c *Client) send(ctx context.Context, method, path string, payload []byte) (*http.Response, error) {
	attempts := 1
	if isRetryableMethod(method) {
		attempts = 3
	}

	var err error
	for attempt := range attempts {
		req, reqErr := http.NewRequestWithContext(ctx, method, c.URL(path), bytes.NewReader(payload))
		if reqErr != nil {
			return nil, reqErr
		}

		req.Header.Add("Content-Type", "application/json")
		req.Header.Add("User-Agent", c.userAgent)
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.accessToken))

		var res *http.Response
		res, err = c.httpClient.Do(req)
		if err == nil {
			return res, nil
		}
		if attempt == attempts-1 || ctx.Err() != nil || !isRetryableTransportError(err) {
			return nil, err
		}

		if err := sleep(ctx, time.Duration(attempt+1)*100*time.Millisecond); err != nil {
			return nil, err
		}
	}

	return nil, err
}

func sleep(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// retryAfterDelay reads a Retry-After value (delta-seconds or HTTP-date), capped so a bad value cannot stall a run.
// Without a usable value it backs off exponentially by retry, so the retries still span a longer limit window.
func retryAfterDelay(value string, retry int) time.Duration {
	value = strings.TrimSpace(value)
	// an out-of-range value saturates, so a huge positive one still lands on the cap
	if secs, err := strconv.Atoi(value); (err == nil || errors.Is(err, strconv.ErrRange)) && secs >= 0 {
		return time.Duration(min(secs, int(maxRateLimitWait/time.Second))) * time.Second
	}
	if date, err := http.ParseTime(value); err == nil {
		return min(max(time.Until(date), 0), maxRateLimitWait)
	}
	return min(baseRateLimitWait<<retry, maxRateLimitWait)
}

func isRetryableMethod(method string) bool {
	switch strings.ToUpper(method) {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}

func isRetryableTransportError(err error) bool {
	if err == nil {
		return false
	}

	var netErr net.Error
	if errors.As(err, &netErr) && (netErr.Timeout() || netErr.Temporary()) {
		return true
	}

	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "connection reset by peer") ||
		strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "unexpected eof")
}

func (c *Client) service(basePath string) *service {
	return &service{client: c, basePath: basePath}
}
