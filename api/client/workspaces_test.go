package client_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/api/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkspaces_GetByAuthToken_Success(t *testing.T) {
	ctx := context.Background()

	dataPlaneURL := "https://dataplane.example.com"
	calls := []testutils.Call{
		{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/v2/workspace", "")
			},
			ResponseStatus: 200,
			ResponseBody: `{
				"workspace": {
					"id": "ws_123abc",
					"name": "Production",
					"environment": "PRODUCTION",
					"status": "ACTIVE",
					"region": "US",
					"dataPlaneURL": "https://dataplane.example.com"
				}
			}`,
		},
	}

	httpClient := testutils.NewMockHTTPClient(t, calls...)

	c, err := client.New("some-access-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	workspace, err := c.Workspaces.GetByAuthToken(ctx)
	require.NoError(t, err)
	assert.NotNil(t, workspace)
	assert.Equal(t, "ws_123abc", workspace.ID)
	assert.Equal(t, "Production", workspace.Name)
	assert.Equal(t, "PRODUCTION", workspace.Environment)
	assert.Equal(t, "ACTIVE", workspace.Status)
	assert.Equal(t, "US", workspace.Region)
	assert.NotNil(t, workspace.DataPlaneURL)
	assert.Equal(t, dataPlaneURL, *workspace.DataPlaneURL)

	httpClient.AssertNumberOfCalls()
}

func TestWorkspaces_GetByAuthToken_NullDataPlaneURL(t *testing.T) {
	ctx := context.Background()

	calls := []testutils.Call{
		{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/v2/workspace", "")
			},
			ResponseStatus: 200,
			ResponseBody: `{
				"workspace": {
					"id": "ws_456def",
					"name": "Development",
					"environment": "DEVELOPMENT",
					"status": "ACTIVE",
					"region": "EU"
				}
			}`,
		},
	}

	httpClient := testutils.NewMockHTTPClient(t, calls...)

	c, err := client.New("some-access-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	workspace, err := c.Workspaces.GetByAuthToken(ctx)
	require.NoError(t, err)
	assert.NotNil(t, workspace)
	assert.Equal(t, "ws_456def", workspace.ID)
	assert.Equal(t, "Development", workspace.Name)
	assert.Equal(t, "DEVELOPMENT", workspace.Environment)
	assert.Equal(t, "ACTIVE", workspace.Status)
	assert.Equal(t, "EU", workspace.Region)
	assert.Nil(t, workspace.DataPlaneURL)

	httpClient.AssertNumberOfCalls()
}

func TestWorkspaces_GetByAuthToken_Unauthorized(t *testing.T) {
	ctx := context.Background()

	calls := []testutils.Call{
		{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/v2/workspace", "")
			},
			ResponseStatus: 401,
			ResponseBody: `{
				"error": "This endpoint requires a workspace-level access token",
				"code": "UNAUTHORIZED"
			}`,
		},
	}

	httpClient := testutils.NewMockHTTPClient(t, calls...)

	c, err := client.New("some-access-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	workspace, err := c.Workspaces.GetByAuthToken(ctx)
	require.Error(t, err)
	assert.Nil(t, workspace)

	// Verify it's an APIError with correct status code
	apiErr, ok := err.(*client.APIError)
	assert.True(t, ok)
	assert.Equal(t, 401, apiErr.HTTPStatusCode)
	assert.Contains(t, apiErr.Message, "workspace-level access token")

	httpClient.AssertNumberOfCalls()
}

func TestWorkspaces_GetByAuthToken_APIError(t *testing.T) {
	ctx := context.Background()

	calls := []testutils.Call{
		{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/v2/workspace", "")
			},
			ResponseStatus: 500,
			ResponseBody: `{
				"error": "Internal server error",
				"code": "INTERNAL_ERROR"
			}`,
		},
	}

	httpClient := testutils.NewMockHTTPClient(t, calls...)

	c, err := client.New("some-access-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	workspace, err := c.Workspaces.GetByAuthToken(ctx)
	require.Error(t, err)
	assert.Nil(t, workspace)

	// Verify it's an APIError
	apiErr, ok := err.(*client.APIError)
	assert.True(t, ok)
	assert.Equal(t, 500, apiErr.HTTPStatusCode)

	httpClient.AssertNumberOfCalls()
}
