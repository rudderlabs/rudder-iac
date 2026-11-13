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

func TestBaseURLForRegion(t *testing.T) {
	for _, tc := range []struct {
		region, url  string
		switchRegion bool
	}{
		{"EU", client.BASE_URL_EU, true},
		{"eu", client.BASE_URL_EU, true},
		{" EU ", client.BASE_URL_EU, true},
		{"US", client.BASE_URL, false},
		{"", client.BASE_URL, false},
		{"APAC", client.BASE_URL, false},
		{"eu-west", client.BASE_URL, false},
	} {
		t.Run(tc.region, func(t *testing.T) {
			url, switchRegion := client.BaseURLForRegion(tc.region)
			assert.Equal(t, tc.url, url)
			assert.Equal(t, tc.switchRegion, switchRegion)
		})
	}
}

func TestRegionDiscoveryRoutesFirstRequestAndCachesWorkspace(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t,
		testutils.Call{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(t, req, "GET", client.BASE_URL+"/v2/workspace", "")
			},
			ResponseStatus: http.StatusOK,
			ResponseBody:   `{"workspace":{"id":"ws-eu","region":"EU"}}`,
		},
		testutils.Call{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(t, req, "GET", client.BASE_URL_EU+"/v2/sources", "")
			},
			ResponseStatus: http.StatusOK,
			ResponseBody:   `{"sources":[]}`,
		},
	)
	c, err := client.New("test-token", client.WithHTTPClient(httpClient), client.WithRegionDiscovery())
	require.NoError(t, err)
	assert.Equal(t, client.BASE_URL, c.URL(""))
	_, err = c.Do(t.Context(), http.MethodGet, "/v2/sources", nil)
	require.NoError(t, err)
	workspace, err := c.Workspaces.GetByAuthToken(t.Context())
	require.NoError(t, err)
	assert.Equal(t, &client.Workspace{ID: "ws-eu", Region: "EU"}, workspace)
	assert.Equal(t, client.BASE_URL_EU, c.URL(""))
	httpClient.AssertNumberOfCalls()
}

func TestRegionDiscoveryRecognizesDefaultURLSpellings(t *testing.T) {
	for _, rawURL := range []string{"https://api.rudderstack.com/", "HTTPS://API.RUDDERSTACK.COM", "https://api.rudderstack.com:443"} {
		t.Run(rawURL, func(t *testing.T) {
			httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
				Validate: func(req *http.Request) bool {
					return testutils.ValidateRequest(t, req, "GET", client.BASE_URL+"/v2/workspace", "")
				},
				ResponseStatus: http.StatusOK,
				ResponseBody:   `{"workspace":{"region":"EU"}}`,
			})
			c, err := client.New("test-token", client.WithBaseURL(rawURL), client.WithHTTPClient(httpClient), client.WithRegionDiscovery())
			require.NoError(t, err)
			_, err = c.Workspaces.GetByAuthToken(t.Context())
			require.NoError(t, err)
			assert.Equal(t, client.BASE_URL_EU, c.URL(""))
			httpClient.AssertNumberOfCalls()
		})
	}
}

func TestRegionDiscoveryLeavesCustomPathUntouched(t *testing.T) {
	const apiURL = "https://api.rudderstack.com/custom"
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return testutils.ValidateRequest(t, req, "GET", apiURL+"/v2/workspace", "")
		},
		ResponseStatus: http.StatusOK,
		ResponseBody:   `{"workspace":{"region":"EU"}}`,
	})
	c, err := client.New("test-token", client.WithBaseURL(apiURL), client.WithHTTPClient(httpClient), client.WithRegionDiscovery())
	require.NoError(t, err)
	_, err = c.Workspaces.GetByAuthToken(t.Context())
	require.NoError(t, err)
	assert.Equal(t, apiURL, c.URL(""))
	httpClient.AssertNumberOfCalls()
}

func TestRegionDiscoveryLeavesOverrideUntouchedRegardlessOfOptionOrder(t *testing.T) {
	const apiURL = "https://api.custom.test"
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return testutils.ValidateRequest(t, req, "GET", apiURL+"/v2/workspace", "")
		},
		ResponseStatus: http.StatusOK,
		ResponseBody:   `{"workspace":{"region":"EU"}}`,
	})
	c, err := client.New("test-token", client.WithRegionDiscovery(), client.WithBaseURL(apiURL), client.WithHTTPClient(httpClient))
	require.NoError(t, err)
	_, err = c.Workspaces.GetByAuthToken(t.Context())
	require.NoError(t, err)
	assert.Equal(t, apiURL, c.URL(""))
	httpClient.AssertNumberOfCalls()
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
