package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

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
}

const BASE_URL_V2 = "https://api.rudderstack.com/v2"

var (
	ErrEmptyAccessToken  = fmt.Errorf("access token cannot be empty")
	ErrInvalidBaseURL    = fmt.Errorf("base url cannot be empty")
	ErrInvalidHTTPClient = fmt.Errorf("http client cannot be nil")
)

func New(accessToken string, options ...Option) (*Client, error) {
	client := &Client{
		baseURL:     BASE_URL_V2,
		httpClient:  &http.Client{},
		accessToken: accessToken,
		userAgent:   "rudder-api-go/1.0.0",
	}

	client.Sources = &sources{service: client.service("sources")}
	client.Destinations = &destinations{service: client.service("destinations")}
	client.Connections = &connections{service: client.service("connections")}

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
	req, err := http.NewRequestWithContext(ctx, method, c.URL(path), body)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("User-Agent", c.userAgent)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.accessToken))

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	data, err := ioutil.ReadAll(res.Body)
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

func (c *Client) service(basePath string) *service {
	return &service{client: c, basePath: basePath}
}
