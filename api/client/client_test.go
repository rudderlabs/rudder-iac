package client_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

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
