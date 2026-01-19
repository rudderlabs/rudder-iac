package transformations_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/api/client/transformations"
	"github.com/rudderlabs/rudder-iac/api/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper to convert string to *string
func strPtr(s string) *string {
	return &s
}

func TestCreateTransformation(t *testing.T) {
	tests := []struct {
		name    string
		publish bool
	}{
		{
			name:    "create transformation without publish",
			publish: false,
		},
		{
			name:    "create transformation with publish",
			publish: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			calls := []testutils.Call{
				{
					Validate: func(req *http.Request) bool {
						expectedURL := "https://api.rudderstack.com/transformations?publish=false"
						if tt.publish {
							expectedURL = "https://api.rudderstack.com/transformations?publish=true"
						}
						if req.URL.String() != expectedURL {
							return false
						}
						return testutils.ValidateRequest(t, req, "POST", expectedURL, `{
							"name": "test-transformation",
							"description": "Test transformation description",
							"code": "function transform(event) { return event; }",
							"language": "javascript",
							"externalId": "ext-123"
						}`)
					},
					ResponseStatus: 200,
					ResponseBody: `{
						"id": "trans-123",
						"versionId": "ver-456",
						"name": "test-transformation",
						"description": "Test transformation description",
						"code": "function transform(event) { return event; }",
						"language": "javascript",
						"imports": ["lib-1", "lib-2"],
						"workspaceId": "ws-789",
						"externalId": "ext-123"
					}`,
				},
			}

			httpClient := testutils.NewMockHTTPClient(t, calls...)
			c, err := client.New("some-access-token", client.WithHTTPClient(httpClient))
			require.NoError(t, err)

			store := transformations.NewRudderTransformationStore(c)
			req := &transformations.CreateTransformationRequest{
				Name:        "test-transformation",
				Description: strPtr("Test transformation description"),
				Code:        "function transform(event) { return event; }",
				Language:    "javascript",
				ExternalID:  "ext-123",
			}

			result, err := store.CreateTransformation(ctx, req, tt.publish)
			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, "trans-123", result.ID)
			assert.Equal(t, "ver-456", result.VersionID)
			assert.Equal(t, "test-transformation", result.Name)
			assert.Equal(t, "Test transformation description", result.Description)
			assert.Equal(t, "function transform(event) { return event; }", result.Code)
			assert.Equal(t, "javascript", result.Language)
			assert.Equal(t, []string{"lib-1", "lib-2"}, result.Imports)
			assert.Equal(t, "ws-789", result.WorkspaceID)
			assert.Equal(t, "ext-123", result.ExternalID)

			httpClient.AssertNumberOfCalls()
		})
	}
}

func TestUpdateTransformation(t *testing.T) {
	tests := []struct {
		name    string
		publish bool
	}{
		{
			name:    "update transformation without publish",
			publish: false,
		},
		{
			name:    "update transformation with publish",
			publish: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			calls := []testutils.Call{
				{
					Validate: func(req *http.Request) bool {
						expectedURL := "https://api.rudderstack.com/transformations/trans-123?publish=false"
						if tt.publish {
							expectedURL = "https://api.rudderstack.com/transformations/trans-123?publish=true"
						}
						if req.URL.String() != expectedURL {
							return false
						}
						return testutils.ValidateRequest(t, req, "POST", expectedURL, `{
							"name": "updated-transformation",
							"description": "Updated description",
							"code": "function transform(event) { return event; }",
							"language": "javascript"
						}`)
					},
					ResponseStatus: 200,
					ResponseBody: `{
						"id": "trans-123",
						"versionId": "ver-789",
						"name": "updated-transformation",
						"description": "Updated description",
						"code": "function transform(event) { return event; }",
						"language": "javascript",
						"imports": [],
						"workspaceId": "ws-789",
						"externalId": "ext-123"
					}`,
				},
			}

			httpClient := testutils.NewMockHTTPClient(t, calls...)
			c, err := client.New("some-access-token", client.WithHTTPClient(httpClient))
			require.NoError(t, err)

			store := transformations.NewRudderTransformationStore(c)
			req := &transformations.UpdateTransformationRequest{
				Name:        "updated-transformation",
				Description: strPtr("Updated description"),
				Code:        "function transform(event) { return event; }",
				Language:    "javascript",
			}

			result, err := store.UpdateTransformation(ctx, "trans-123", req, tt.publish)
			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, "trans-123", result.ID)
			assert.Equal(t, "ver-789", result.VersionID)
			assert.Equal(t, "updated-transformation", result.Name)

			httpClient.AssertNumberOfCalls()
		})
	}
}

func TestGetTransformation(t *testing.T) {
	ctx := context.Background()

	calls := []testutils.Call{
		{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/transformations/trans-123", "")
			},
			ResponseStatus: 200,
			ResponseBody: `{
				"id": "trans-123",
				"versionId": "ver-456",
				"name": "test-transformation",
				"description": "Test description",
				"code": "function transform(event) { return event; }",
				"language": "javascript",
				"imports": ["lib-1"],
				"workspaceId": "ws-789",
				"externalId": "ext-123"
			}`,
		},
	}

	httpClient := testutils.NewMockHTTPClient(t, calls...)
	c, err := client.New("some-access-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	store := transformations.NewRudderTransformationStore(c)
	result, err := store.GetTransformation(ctx, "trans-123")
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "trans-123", result.ID)
	assert.Equal(t, "ver-456", result.VersionID)
	assert.Equal(t, "test-transformation", result.Name)
	assert.Equal(t, "Test description", result.Description)

	httpClient.AssertNumberOfCalls()
}

func TestListTransformations(t *testing.T) {
	ctx := context.Background()

	calls := []testutils.Call{
		{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/transformations", "")
			},
			ResponseStatus: 200,
			ResponseBody: `{
				"transformations": [
					{
						"id": "trans-1",
						"versionId": "ver-1",
						"name": "transformation-1",
						"code": "code1",
						"language": "javascript",
						"imports": [],
						"workspaceId": "ws-1"
					},
					{
						"id": "trans-2",
						"versionId": "ver-2",
						"name": "transformation-2",
						"code": "code2",
						"language": "javascript",
						"imports": ["lib-1"],
						"workspaceId": "ws-1"
					}
				]
			}`,
		},
	}

	httpClient := testutils.NewMockHTTPClient(t, calls...)
	c, err := client.New("some-access-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	store := transformations.NewRudderTransformationStore(c)
	results, err := store.ListTransformations(ctx)
	require.NoError(t, err)
	assert.NotNil(t, results)
	assert.Len(t, results, 2)
	assert.Equal(t, "trans-1", results[0].ID)
	assert.Equal(t, "transformation-1", results[0].Name)
	assert.Equal(t, "trans-2", results[1].ID)
	assert.Equal(t, "transformation-2", results[1].Name)

	httpClient.AssertNumberOfCalls()
}

func TestDeleteTransformation(t *testing.T) {
	ctx := context.Background()

	calls := []testutils.Call{
		{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(t, req, "DELETE", "https://api.rudderstack.com/transformations/trans-123", "")
			},
			ResponseStatus: 200,
			ResponseBody:   `{}`,
		},
	}

	httpClient := testutils.NewMockHTTPClient(t, calls...)
	c, err := client.New("some-access-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	store := transformations.NewRudderTransformationStore(c)
	err = store.DeleteTransformation(ctx, "trans-123")
	require.NoError(t, err)

	httpClient.AssertNumberOfCalls()
}

func TestCreateLibrary(t *testing.T) {
	tests := []struct {
		name    string
		publish bool
	}{
		{
			name:    "create library without publish",
			publish: false,
		},
		{
			name:    "create library with publish",
			publish: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			calls := []testutils.Call{
				{
					Validate: func(req *http.Request) bool {
						expectedURL := "https://api.rudderstack.com/libraries?publish=false"
						if tt.publish {
							expectedURL = "https://api.rudderstack.com/libraries?publish=true"
						}
						if req.URL.String() != expectedURL {
							return false
						}
						return testutils.ValidateRequest(t, req, "POST", expectedURL, `{
							"name": "test-library",
							"description": "Test library description",
							"code": "function helper() { return true; }",
							"language": "javascript",
							"externalId": "lib-ext-123"
						}`)
					},
					ResponseStatus: 200,
					ResponseBody: `{
						"id": "lib-123",
						"versionId": "lib-ver-456",
						"name": "test-library",
						"description": "Test library description",
						"code": "function helper() { return true; }",
						"language": "javascript",
						"handleName": "testLibrary",
						"workspaceId": "ws-789",
						"externalId": "lib-ext-123"
					}`,
				},
			}

			httpClient := testutils.NewMockHTTPClient(t, calls...)
			c, err := client.New("some-access-token", client.WithHTTPClient(httpClient))
			require.NoError(t, err)

			store := transformations.NewRudderTransformationStore(c)
			req := &transformations.CreateLibraryRequest{
				Name:        "test-library",
				Description: strPtr("Test library description"),
				Code:        "function helper() { return true; }",
				Language:    "javascript",
				ExternalID:  "lib-ext-123",
			}

			result, err := store.CreateLibrary(ctx, req, tt.publish)
			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, "lib-123", result.ID)
			assert.Equal(t, "lib-ver-456", result.VersionID)
			assert.Equal(t, "test-library", result.Name)
			assert.Equal(t, "Test library description", result.Description)
			assert.Equal(t, "function helper() { return true; }", result.Code)
			assert.Equal(t, "javascript", result.Language)
			assert.Equal(t, "testLibrary", result.HandleName)
			assert.Equal(t, "ws-789", result.WorkspaceID)
			assert.Equal(t, "lib-ext-123", result.ExternalID)

			httpClient.AssertNumberOfCalls()
		})
	}
}

func TestUpdateLibrary(t *testing.T) {
	tests := []struct {
		name    string
		publish bool
	}{
		{
			name:    "update library without publish",
			publish: false,
		},
		{
			name:    "update library with publish",
			publish: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			calls := []testutils.Call{
				{
					Validate: func(req *http.Request) bool {
						expectedURL := "https://api.rudderstack.com/libraries/lib-123?publish=false"
						if tt.publish {
							expectedURL = "https://api.rudderstack.com/libraries/lib-123?publish=true"
						}
						if req.URL.String() != expectedURL {
							return false
						}
						return testutils.ValidateRequest(t, req, "POST", expectedURL, `{
							"name": "updated-library",
							"description": "Updated library description",
							"code": "function helper() { return false; }",
							"language": "javascript"
						}`)
					},
					ResponseStatus: 200,
					ResponseBody: `{
						"id": "lib-123",
						"versionId": "lib-ver-789",
						"name": "updated-library",
						"description": "Updated library description",
						"code": "function helper() { return false; }",
						"language": "javascript",
						"handleName": "updatedLibrary",
						"workspaceId": "ws-789",
						"externalId": "lib-ext-123"
					}`,
				},
			}

			httpClient := testutils.NewMockHTTPClient(t, calls...)
			c, err := client.New("some-access-token", client.WithHTTPClient(httpClient))
			require.NoError(t, err)

			store := transformations.NewRudderTransformationStore(c)
			req := &transformations.UpdateLibraryRequest{
				Name:        "updated-library",
				Description: strPtr("Updated library description"),
				Code:        "function helper() { return false; }",
				Language:    "javascript",
			}

			result, err := store.UpdateLibrary(ctx, "lib-123", req, tt.publish)
			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, "lib-123", result.ID)
			assert.Equal(t, "lib-ver-789", result.VersionID)
			assert.Equal(t, "updated-library", result.Name)

			httpClient.AssertNumberOfCalls()
		})
	}
}

func TestGetLibrary(t *testing.T) {
	ctx := context.Background()

	calls := []testutils.Call{
		{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/libraries/lib-123", "")
			},
			ResponseStatus: 200,
			ResponseBody: `{
				"id": "lib-123",
				"versionId": "lib-ver-456",
				"name": "test-library",
				"description": "Test library description",
				"code": "function helper() { return true; }",
				"language": "javascript",
				"handleName": "testLibrary",
				"workspaceId": "ws-789",
				"externalId": "lib-ext-123"
			}`,
		},
	}

	httpClient := testutils.NewMockHTTPClient(t, calls...)
	c, err := client.New("some-access-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	store := transformations.NewRudderTransformationStore(c)
	result, err := store.GetLibrary(ctx, "lib-123")
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "lib-123", result.ID)
	assert.Equal(t, "lib-ver-456", result.VersionID)
	assert.Equal(t, "test-library", result.Name)
	assert.Equal(t, "Test library description", result.Description)
	assert.Equal(t, "testLibrary", result.HandleName)

	httpClient.AssertNumberOfCalls()
}

func TestListLibraries(t *testing.T) {
	ctx := context.Background()

	calls := []testutils.Call{
		{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/libraries", "")
			},
			ResponseStatus: 200,
			ResponseBody: `{
				"libraries": [
					{
						"id": "lib-1",
						"versionId": "lib-ver-1",
						"name": "library-1",
						"code": "code1",
						"language": "javascript",
						"handleName": "library1",
						"workspaceId": "ws-1"
					},
					{
						"id": "lib-2",
						"versionId": "lib-ver-2",
						"name": "library-2",
						"code": "code2",
						"language": "javascript",
						"handleName": "library2",
						"workspaceId": "ws-1"
					}
				]
			}`,
		},
	}

	httpClient := testutils.NewMockHTTPClient(t, calls...)
	c, err := client.New("some-access-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	store := transformations.NewRudderTransformationStore(c)
	results, err := store.ListLibraries(ctx)
	require.NoError(t, err)
	assert.NotNil(t, results)
	assert.Len(t, results, 2)
	assert.Equal(t, "lib-1", results[0].ID)
	assert.Equal(t, "library-1", results[0].Name)
	assert.Equal(t, "library1", results[0].HandleName)
	assert.Equal(t, "lib-2", results[1].ID)
	assert.Equal(t, "library-2", results[1].Name)
	assert.Equal(t, "library2", results[1].HandleName)

	httpClient.AssertNumberOfCalls()
}

func TestDeleteLibrary(t *testing.T) {
	ctx := context.Background()

	calls := []testutils.Call{
		{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(t, req, "DELETE", "https://api.rudderstack.com/libraries/lib-123", "")
			},
			ResponseStatus: 200,
			ResponseBody:   `{}`,
		},
	}

	httpClient := testutils.NewMockHTTPClient(t, calls...)
	c, err := client.New("some-access-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	store := transformations.NewRudderTransformationStore(c)
	err = store.DeleteLibrary(ctx, "lib-123")
	require.NoError(t, err)

	httpClient.AssertNumberOfCalls()
}

func TestBatchPublish(t *testing.T) {
	ctx := context.Background()

	calls := []testutils.Call{
		{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(t, req, "POST", "https://api.rudderstack.com/transformations/libraries/publish", `{
					"transformations": [
						{
							"versionId": "trans-ver-1",
							"testInput": [{"event": "test1"}]
						},
						{
							"versionId": "trans-ver-2",
							"testInput": [{"event": "test2"}]
						}
					],
					"libraries": [
						{
							"versionId": "lib-ver-1"
						},
						{
							"versionId": "lib-ver-2"
						}
					]
				}`)
			},
			ResponseStatus: 200,
			ResponseBody: `{
				"published": true
			}`,
		},
	}

	httpClient := testutils.NewMockHTTPClient(t, calls...)
	c, err := client.New("some-access-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	store := transformations.NewRudderTransformationStore(c)
	req := &transformations.BatchPublishRequest{
		Transformations: []transformations.BatchPublishTransformation{
			{
				VersionID: "trans-ver-1",
				TestInput: []any{map[string]string{"event": "test1"}},
			},
			{
				VersionID: "trans-ver-2",
				TestInput: []any{map[string]string{"event": "test2"}},
			},
		},
		Libraries: []transformations.BatchPublishLibrary{
			{
				VersionID: "lib-ver-1",
			},
			{
				VersionID: "lib-ver-2",
			},
		},
	}

	err = store.BatchPublish(ctx, req)
	require.NoError(t, err)

	httpClient.AssertNumberOfCalls()
}

func TestUpdateTransformationClearDescription(t *testing.T) {
	// This test proves we can explicitly set description to empty string
	// which is different from omitting the field entirely
	ctx := context.Background()

	calls := []testutils.Call{
		{
			Validate: func(req *http.Request) bool {
				// Verify that description is included in JSON as empty string
				expectedURL := "https://api.rudderstack.com/transformations/trans-123?publish=false"
				if req.URL.String() != expectedURL {
					return false
				}
				return testutils.ValidateRequest(t, req, "POST", expectedURL, `{
					"name": "transformation",
					"description": "",
					"code": "function transform(event) { return event; }",
					"language": "javascript"
				}`)
			},
			ResponseStatus: 200,
			ResponseBody: `{
				"id": "trans-123",
				"versionId": "ver-new",
				"name": "transformation",
				"description": "",
				"code": "function transform(event) { return event; }",
				"language": "javascript",
				"imports": [],
				"workspaceId": "ws-789"
			}`,
		},
	}

	httpClient := testutils.NewMockHTTPClient(t, calls...)
	c, err := client.New("some-access-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	store := transformations.NewRudderTransformationStore(c)
	req := &transformations.UpdateTransformationRequest{
		Name:        "transformation",
		Description: strPtr(""), // Explicitly set to empty - this clears the description!
		Code:        "function transform(event) { return event; }",
		Language:    "javascript",
	}

	result, err := store.UpdateTransformation(ctx, "trans-123", req, false)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "", result.Description)

	httpClient.AssertNumberOfCalls()
}

