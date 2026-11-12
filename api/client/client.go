package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

var ErrResourceNotFound = fmt.Errorf("resource not found")

type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

type Client struct {
	baseURL             string
	urlMu               sync.RWMutex
	regionMu            sync.Mutex
	discoverRegion      bool
	discoveredWorkspace *Workspace
	accessToken         string
	userAgent           string
	httpClient          HTTPClient

	Sources      *sources
	Destinations *destinations
	Connections  *connections
	Accounts     *accounts
	Workspaces   *workspaces
}

const (
	BASE_URL    = "https://api.rudderstack.com"
	BASE_URL_EU = "https://api.eu.rudderstack.com"
)

// BaseURLForRegion returns a regional API URL only when the region is known.
func BaseURLForRegion(region string) (string, bool) {
	if strings.EqualFold(strings.TrimSpace(region), "EU") {
		return BASE_URL_EU, true
	}
	return BASE_URL, false
}

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
	if client.discoverRegion {
		client.discoverRegion = isDefaultBaseURL(client.baseURL)
		if client.discoverRegion {
			client.baseURL = BASE_URL
		}
	}

	if client.accessToken == "" {
		return nil, ErrEmptyAccessToken
	}

	return client, nil
}

func (c *Client) URL(path string) string {
	c.urlMu.RLock()
	defer c.urlMu.RUnlock()
	if len(path) == 0 {
		return c.baseURL
	}

	return fmt.Sprintf("%s/%s", c.baseURL, strings.TrimPrefix(path, "/"))
}

func (c *Client) Do(ctx context.Context, method, path string, body io.Reader) ([]byte, error) {
	if c.discoverRegion {
		if err := c.ensureRegion(ctx); err != nil {
			return nil, fmt.Errorf("fetching workspace information: %w", err)
		}
	}
	return c.do(ctx, method, path, body)
}

func (c *Client) do(ctx context.Context, method, path string, body io.Reader) ([]byte, error) {
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

// Discovery runs on demand so local-only commands can construct providers
// without requiring the workspace endpoint.
func (c *Client) ensureRegion(ctx context.Context) error {
	c.regionMu.Lock()
	defer c.regionMu.Unlock()
	if c.discoveredWorkspace != nil {
		return nil
	}

	workspace, err := c.Workspaces.getByAuthToken(ctx)
	if err != nil {
		return err
	}
	if workspace == nil {
		return fmt.Errorf("workspace is missing")
	}
	if regionalURL, switchRegion := BaseURLForRegion(workspace.Region); switchRegion {
		c.urlMu.Lock()
		c.baseURL = regionalURL
		c.urlMu.Unlock()
	}
	c.discoveredWorkspace = workspace
	return nil
}

// isDefaultBaseURL accepts harmless spellings of the US endpoint while leaving
// custom gateways and URL paths under explicit caller control.
func isDefaultBaseURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || !strings.EqualFold(u.Scheme, "https") || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return false
	}
	return strings.EqualFold(u.Hostname(), "api.rudderstack.com") && (u.Port() == "" || u.Port() == "443")
}

func (c *Client) doRequest(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	attempts := 1
	if isRetryableMethod(method) {
		attempts = 3
	}

	var err error
	for attempt := range attempts {
		req, reqErr := http.NewRequestWithContext(ctx, method, c.URL(path), body)
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

		timer := time.NewTimer(time.Duration(attempt+1) * 100 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}

	return nil, err
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
