package client_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/stretchr/testify/assert"
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

// mockHTTPClient is a mock implementation of HTTPClient for testing
type mockHTTPClient struct {
	statusCode int
	response   string
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: m.statusCode,
		Body:       io.NopCloser(strings.NewReader(m.response)),
	}, nil
}

func TestClientErrorHandling(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		response       string
		expectedMsg    string
		expectedCode   string
	}{
		{
			name:         "500 Internal Server Error",
			statusCode:   500,
			response:     `{"error": "Database connection failed", "code": "DB_ERROR"}`,
			expectedMsg:  "Something went wrong. Try again shortly.",
			expectedCode: "",
		},
		{
			name:         "503 Service Unavailable",
			statusCode:   503,
			response:     `{"error": "Service temporarily unavailable", "code": "SERVICE_DOWN"}`,
			expectedMsg:  "Something went wrong. Try again shortly.",
			expectedCode: "",
		},
		{
			name:         "400 Bad Request",
			statusCode:   400,
			response:     `{"error": "Invalid request", "code": "BAD_REQUEST"}`,
			expectedMsg:  "Invalid request",
			expectedCode: "BAD_REQUEST",
		},
		{
			name:         "404 Not Found",
			statusCode:   404,
			response:     `{"error": "Resource not found", "code": "NOT_FOUND"}`,
			expectedMsg:  "Resource not found",
			expectedCode: "NOT_FOUND",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockHTTPClient{
				statusCode: tt.statusCode,
				response:   tt.response,
			}

			c, err := client.New("test-token", client.WithHTTPClient(mockClient))
			assert.NoError(t, err)

			_, err = c.Do(context.Background(), "GET", "/test", nil)
			assert.Error(t, err)

			apiError, ok := err.(*client.APIError)
			assert.True(t, ok, "error should be of type APIError")
			assert.Equal(t, tt.statusCode, apiError.HTTPStatusCode)
			assert.Equal(t, tt.expectedMsg, apiError.Message)
			assert.Equal(t, tt.expectedCode, apiError.ErrorCode)
		})
	}
}
