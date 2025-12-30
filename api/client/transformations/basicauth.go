package transformations

import (
	"encoding/base64"
	"fmt"
	"net/http"
)

// basicAuthHTTPClient wraps an HTTP client and adds basic auth headers
type basicAuthHTTPClient struct {
	httpClient  http.Client
	accessToken string
}

// NewBasicAuthHTTPClient creates an HTTP client wrapper that adds basic auth
// username is empty, password is the access token
func NewBasicAuthHTTPClient(accessToken string) *basicAuthHTTPClient {
	return &basicAuthHTTPClient{
		httpClient:  http.Client{},
		accessToken: accessToken,
	}
}

// Do implements the HTTPClient interface with basic auth
func (c *basicAuthHTTPClient) Do(req *http.Request) (*http.Response, error) {
	// Basic auth: username is empty, password is access token
	// Format: base64(":accessToken")
	credentials := base64.StdEncoding.EncodeToString([]byte(":" + c.accessToken))
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", credentials))

	return c.httpClient.Do(req)
}
