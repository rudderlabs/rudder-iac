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

// ==================== Transformation Tests ====================

func TestTransformationsCreateWithPublishFalse(t *testing.T) {
	ctx := context.Background()

	calls := []testutils.Call{
		{
			Validate: func(req *http.Request) bool {
				// Verify POST method and publish=false query param
				if req.URL.Query().Get("publish") != "false" {
					t.Error("expected publish=false query parameter")
					return false
				}
				return testutils.ValidateRequest(t, req, "POST", "https://api.rudderstack.com/transformations", `{
					"name": "My Transformation",
					"description": "Test transformation",
					"code": "function transformEvent(event) { return event; }",
					"language": "javascript",
					"externalId": "my-transformation"
				}`)
			},
			ResponseStatus: 200,
			ResponseBody: `{
				"id": "trans123",
				"versionId": "ver123",
				"name": "My Transformation",
				"description": "Test transformation",
				"code": "function transformEvent(event) { return event; }",
				"language": "javascript",
				"externalId": "my-transformation"
			}`,
		},
	}

	httpClient := testutils.NewMockHTTPClient(t, calls...)

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	req := &client.CreateTransformationRequest{
		Name:        "My Transformation",
		Description: "Test transformation",
		Code:        "function transformEvent(event) { return event; }",
		Language:    "javascript",
		ExternalID:  "my-transformation",
	}

	transformation, err := c.Transformations.Create(ctx, req, false)
	require.NoError(t, err)

	assert.Equal(t, "trans123", transformation.ID)
	assert.Equal(t, "ver123", transformation.VersionID)
	assert.Equal(t, "My Transformation", transformation.Name)
	assert.Equal(t, "my-transformation", transformation.ExternalID)

	httpClient.AssertNumberOfCalls()
}

func TestTransformationsCreateWithPublishTrue(t *testing.T) {
	ctx := context.Background()

	calls := []testutils.Call{
		{
			Validate: func(req *http.Request) bool {
				// Verify POST method and publish=true query param
				if req.URL.Query().Get("publish") != "true" {
					t.Error("expected publish=true query parameter")
					return false
				}
				return testutils.ValidateRequest(t, req, "POST", "https://api.rudderstack.com/transformations", "")
			},
			ResponseStatus: 200,
			ResponseBody: `{
				"id": "trans123",
				"versionId": "ver123",
				"name": "My Transformation",
				"language": "javascript"
			}`,
		},
	}

	httpClient := testutils.NewMockHTTPClient(t, calls...)

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	req := &client.CreateTransformationRequest{
		Name:     "My Transformation",
		Code:     "function transformEvent(event) { return event; }",
		Language: "javascript",
	}

	transformation, err := c.Transformations.Create(ctx, req, true)
	require.NoError(t, err)

	assert.Equal(t, "trans123", transformation.ID)

	httpClient.AssertNumberOfCalls()
}

func TestTransformationsUpdate(t *testing.T) {
	ctx := context.Background()

	calls := []testutils.Call{
		{
			Validate: func(req *http.Request) bool {
				// Verify POST method (not PUT) and publish query param
				if req.Method != "POST" {
					t.Errorf("expected POST method, got %s", req.Method)
					return false
				}
				if req.URL.Query().Get("publish") != "false" {
					t.Error("expected publish=false query parameter")
					return false
				}
				return testutils.ValidateRequest(t, req, "POST", "https://api.rudderstack.com/transformations/trans123", `{
					"name": "Updated Transformation",
					"description": "Updated description",
					"code": "function transformEvent(event) { return event; }",
					"language": "javascript"
				}`)
			},
			ResponseStatus: 200,
			ResponseBody: `{
				"id": "trans123",
				"versionId": "ver456",
				"name": "Updated Transformation",
				"description": "Updated description",
				"language": "javascript"
			}`,
		},
	}

	httpClient := testutils.NewMockHTTPClient(t, calls...)

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	req := &client.CreateTransformationRequest{
		Name:        "Updated Transformation",
		Description: "Updated description",
		Code:        "function transformEvent(event) { return event; }",
		Language:    "javascript",
	}

	transformation, err := c.Transformations.Update(ctx, "trans123", req, false)
	require.NoError(t, err)

	assert.Equal(t, "trans123", transformation.ID)
	assert.Equal(t, "ver456", transformation.VersionID)
	assert.Equal(t, "Updated Transformation", transformation.Name)

	httpClient.AssertNumberOfCalls()
}

func TestTransformationsGet(t *testing.T) {
	ctx := context.Background()

	calls := []testutils.Call{
		{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/transformations/trans123", "")
			},
			ResponseStatus: 200,
			ResponseBody: `{
				"id": "trans123",
				"versionId": "ver123",
				"name": "My Transformation",
				"description": "Test transformation",
				"code": "function transformEvent(event) { return event; }",
				"language": "javascript",
				"imports": ["lib1", "lib2"],
				"externalId": "my-transformation"
			}`,
		},
	}

	httpClient := testutils.NewMockHTTPClient(t, calls...)

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	transformation, err := c.Transformations.Get(ctx, "trans123")
	require.NoError(t, err)

	assert.Equal(t, "trans123", transformation.ID)
	assert.Equal(t, "ver123", transformation.VersionID)
	assert.Equal(t, "My Transformation", transformation.Name)
	assert.Equal(t, "javascript", transformation.Language)
	assert.Equal(t, []string{"lib1", "lib2"}, transformation.Imports)

	httpClient.AssertNumberOfCalls()
}

func TestTransformationsList(t *testing.T) {
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
						"id": "trans1",
						"versionId": "ver1",
						"name": "Transformation 1",
						"language": "javascript"
					},
					{
						"id": "trans2",
						"versionId": "ver2",
						"name": "Transformation 2",
						"language": "python"
					}
				]
			}`,
		},
	}

	httpClient := testutils.NewMockHTTPClient(t, calls...)

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	response, err := c.Transformations.List(ctx)
	require.NoError(t, err)

	assert.Len(t, response.Transformations, 2)
	assert.Equal(t, "trans1", response.Transformations[0].ID)
	assert.Equal(t, "Transformation 1", response.Transformations[0].Name)
	assert.Equal(t, "trans2", response.Transformations[1].ID)
	assert.Equal(t, "Transformation 2", response.Transformations[1].Name)

	httpClient.AssertNumberOfCalls()
}

func TestTransformationsDelete(t *testing.T) {
	ctx := context.Background()

	calls := []testutils.Call{
		{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(t, req, "DELETE", "https://api.rudderstack.com/transformations/trans123", "")
			},
			ResponseStatus: 204,
			ResponseBody:   "",
		},
	}

	httpClient := testutils.NewMockHTTPClient(t, calls...)

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	err = c.Transformations.Delete(ctx, "trans123")
	require.NoError(t, err)

	httpClient.AssertNumberOfCalls()
}

// ==================== Library Tests ====================

func TestLibrariesCreateWithPublishFalse(t *testing.T) {
	ctx := context.Background()

	calls := []testutils.Call{
		{
			Validate: func(req *http.Request) bool {
				if req.URL.Query().Get("publish") != "false" {
					t.Error("expected publish=false query parameter")
					return false
				}
				return testutils.ValidateRequest(t, req, "POST", "https://api.rudderstack.com/transformationLibraries", `{
					"name": "My Library",
					"description": "Test library",
					"code": "export function add(a, b) { return a + b; }",
					"language": "javascript",
					"importName": "myLib",
					"externalId": "my-library"
				}`)
			},
			ResponseStatus: 200,
			ResponseBody: `{
				"id": "lib123",
				"versionId": "ver123",
				"name": "My Library",
				"description": "Test library",
				"code": "export function add(a, b) { return a + b; }",
				"language": "javascript",
				"importName": "myLib",
				"externalId": "my-library"
			}`,
		},
	}

	httpClient := testutils.NewMockHTTPClient(t, calls...)

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	req := &client.CreateLibraryRequest{
		Name:        "My Library",
		Description: "Test library",
		Code:        "export function add(a, b) { return a + b; }",
		Language:    "javascript",
		ImportName:  "myLib",
		ExternalID:  "my-library",
	}

	library, err := c.TransformationLibraries.Create(ctx, req, false)
	require.NoError(t, err)

	assert.Equal(t, "lib123", library.ID)
	assert.Equal(t, "ver123", library.VersionID)
	assert.Equal(t, "My Library", library.Name)
	assert.Equal(t, "myLib", library.HandleName)

	httpClient.AssertNumberOfCalls()
}

func TestLibrariesCreateWithPublishTrue(t *testing.T) {
	ctx := context.Background()

	calls := []testutils.Call{
		{
			Validate: func(req *http.Request) bool {
				if req.URL.Query().Get("publish") != "true" {
					t.Error("expected publish=true query parameter")
					return false
				}
				return testutils.ValidateRequest(t, req, "POST", "https://api.rudderstack.com/transformationLibraries", "")
			},
			ResponseStatus: 200,
			ResponseBody: `{
				"id": "lib123",
				"versionId": "ver123",
				"name": "My Library",
				"language": "javascript",
				"importName": "myLib"
			}`,
		},
	}

	httpClient := testutils.NewMockHTTPClient(t, calls...)

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	req := &client.CreateLibraryRequest{
		Name:       "My Library",
		Code:       "export function add(a, b) { return a + b; }",
		Language:   "javascript",
		ImportName: "myLib",
	}

	library, err := c.TransformationLibraries.Create(ctx, req, true)
	require.NoError(t, err)

	assert.Equal(t, "lib123", library.ID)

	httpClient.AssertNumberOfCalls()
}

func TestLibrariesUpdate(t *testing.T) {
	ctx := context.Background()

	calls := []testutils.Call{
		{
			Validate: func(req *http.Request) bool {
				// Verify POST method (not PUT)
				if req.Method != "POST" {
					t.Errorf("expected POST method, got %s", req.Method)
					return false
				}
				if req.URL.Query().Get("publish") != "false" {
					t.Error("expected publish=false query parameter")
					return false
				}
				return testutils.ValidateRequest(t, req, "POST", "https://api.rudderstack.com/transformationLibraries/lib123", "")
			},
			ResponseStatus: 200,
			ResponseBody: `{
				"id": "lib123",
				"versionId": "ver456",
				"name": "Updated Library",
				"language": "javascript",
				"importName": "myLib"
			}`,
		},
	}

	httpClient := testutils.NewMockHTTPClient(t, calls...)

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	req := &client.CreateLibraryRequest{
		Name:       "Updated Library",
		Code:       "export function add(a, b) { return a + b; }",
		Language:   "javascript",
		ImportName: "myLib",
	}

	library, err := c.TransformationLibraries.Update(ctx, "lib123", req, false)
	require.NoError(t, err)

	assert.Equal(t, "lib123", library.ID)
	assert.Equal(t, "ver456", library.VersionID)

	httpClient.AssertNumberOfCalls()
}

func TestLibrariesGet(t *testing.T) {
	ctx := context.Background()

	calls := []testutils.Call{
		{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/transformationLibraries/lib123", "")
			},
			ResponseStatus: 200,
			ResponseBody: `{
				"id": "lib123",
				"versionId": "ver123",
				"name": "My Library",
				"language": "javascript",
				"importName": "myLib",
				"externalId": "my-library"
			}`,
		},
	}

	httpClient := testutils.NewMockHTTPClient(t, calls...)

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	library, err := c.TransformationLibraries.Get(ctx, "lib123")
	require.NoError(t, err)

	assert.Equal(t, "lib123", library.ID)
	assert.Equal(t, "myLib", library.HandleName)
	assert.Equal(t, "my-library", library.ExternalID)

	httpClient.AssertNumberOfCalls()
}

func TestLibrariesList(t *testing.T) {
	ctx := context.Background()

	calls := []testutils.Call{
		{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/transformationLibraries", "")
			},
			ResponseStatus: 200,
			ResponseBody: `{
				"libraries": [
					{"id": "lib1", "name": "Library 1", "importName": "lib1"},
					{"id": "lib2", "name": "Library 2", "importName": "lib2"}
				]
			}`,
		},
	}

	httpClient := testutils.NewMockHTTPClient(t, calls...)

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	response, err := c.TransformationLibraries.List(ctx)
	require.NoError(t, err)

	assert.Len(t, response.Libraries, 2)
	assert.Equal(t, "lib1", response.Libraries[0].ID)
	assert.Equal(t, "lib2", response.Libraries[1].ID)

	httpClient.AssertNumberOfCalls()
}

func TestLibrariesDelete(t *testing.T) {
	ctx := context.Background()

	calls := []testutils.Call{
		{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(t, req, "DELETE", "https://api.rudderstack.com/transformationLibraries/lib123", "")
			},
			ResponseStatus: 204,
			ResponseBody:   "",
		},
	}

	httpClient := testutils.NewMockHTTPClient(t, calls...)

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	err = c.TransformationLibraries.Delete(ctx, "lib123")
	require.NoError(t, err)

	httpClient.AssertNumberOfCalls()
}

// ==================== Batch Publish Tests ====================

func TestBatchPublishTransformationsOnly(t *testing.T) {
	ctx := context.Background()

	calls := []testutils.Call{
		{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(t, req, "POST", "https://api.rudderstack.com/transformations/libraries/publish", `{
					"transformations": [{"versionId": "ver1"}, {"versionId": "ver2"}],
					"libraries": []
				}`)
			},
			ResponseStatus: 200,
			ResponseBody:   `{"published": true}`,
		},
	}

	httpClient := testutils.NewMockHTTPClient(t, calls...)

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	req := &client.BatchPublishRequest{
		Transformations: []client.TransformationVersionInput{
			{VersionID: "ver1"},
			{VersionID: "ver2"},
		},
		Libraries: []client.LibraryVersionInput{},
	}

	response, err := c.Transformations.BatchPublish(ctx, req)
	require.NoError(t, err)

	assert.True(t, response.Published)

	httpClient.AssertNumberOfCalls()
}

func TestBatchPublishLibrariesOnly(t *testing.T) {
	ctx := context.Background()

	calls := []testutils.Call{
		{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(t, req, "POST", "https://api.rudderstack.com/transformations/libraries/publish", `{
					"transformations": [],
					"libraries": [{"versionId": "lib-ver1"}, {"versionId": "lib-ver2"}]
				}`)
			},
			ResponseStatus: 200,
			ResponseBody:   `{"published": true}`,
		},
	}

	httpClient := testutils.NewMockHTTPClient(t, calls...)

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	req := &client.BatchPublishRequest{
		Transformations: []client.TransformationVersionInput{},
		Libraries: []client.LibraryVersionInput{
			{VersionID: "lib-ver1"},
			{VersionID: "lib-ver2"},
		},
	}

	response, err := c.Transformations.BatchPublish(ctx, req)
	require.NoError(t, err)

	assert.True(t, response.Published)

	httpClient.AssertNumberOfCalls()
}

func TestBatchPublishBoth(t *testing.T) {
	ctx := context.Background()

	calls := []testutils.Call{
		{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(t, req, "POST", "https://api.rudderstack.com/transformations/libraries/publish", `{
					"transformations": [{"versionId": "ver1"}],
					"libraries": [{"versionId": "lib-ver1"}]
				}`)
			},
			ResponseStatus: 200,
			ResponseBody:   `{"published": true}`,
		},
	}

	httpClient := testutils.NewMockHTTPClient(t, calls...)

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	req := &client.BatchPublishRequest{
		Transformations: []client.TransformationVersionInput{
			{VersionID: "ver1"},
		},
		Libraries: []client.LibraryVersionInput{
			{VersionID: "lib-ver1"},
		},
	}

	response, err := c.Transformations.BatchPublish(ctx, req)
	require.NoError(t, err)

	assert.True(t, response.Published)

	httpClient.AssertNumberOfCalls()
}

// ==================== Error Cases ====================

func TestTransformationsCreateAPIError(t *testing.T) {
	ctx := context.Background()

	calls := []testutils.Call{
		{
			ResponseStatus: 400,
			ResponseBody:   `{"error": "Bad Request", "code": "VALIDATION_ERROR"}`,
		},
	}

	httpClient := testutils.NewMockHTTPClient(t, calls...)

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	req := &client.CreateTransformationRequest{
		Name:     "Test",
		Code:     "code",
		Language: "javascript",
	}

	_, err = c.Transformations.Create(ctx, req, false)
	require.Error(t, err)

	httpClient.AssertNumberOfCalls()
}

func TestTransformationsGetMalformedResponse(t *testing.T) {
	ctx := context.Background()

	calls := []testutils.Call{
		{
			ResponseStatus: 200,
			ResponseBody:   `{malformed_json`,
		},
	}

	httpClient := testutils.NewMockHTTPClient(t, calls...)

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	_, err = c.Transformations.Get(ctx, "trans123")
	require.Error(t, err)

	httpClient.AssertNumberOfCalls()
}

func TestLibrariesCreateAPIError(t *testing.T) {
	ctx := context.Background()

	calls := []testutils.Call{
		{
			ResponseStatus: 500,
			ResponseBody:   `{"error": "Internal Server Error"}`,
		},
	}

	httpClient := testutils.NewMockHTTPClient(t, calls...)

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	req := &client.CreateLibraryRequest{
		Name:       "Test",
		Code:       "code",
		Language:   "javascript",
		ImportName: "test",
	}

	_, err = c.TransformationLibraries.Create(ctx, req, false)
	require.Error(t, err)

	httpClient.AssertNumberOfCalls()
}

func TestBatchPublishAPIError(t *testing.T) {
	ctx := context.Background()

	calls := []testutils.Call{
		{
			ResponseStatus: 400,
			ResponseBody:   `{"error": "Validation failed"}`,
		},
	}

	httpClient := testutils.NewMockHTTPClient(t, calls...)

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	req := &client.BatchPublishRequest{
		Transformations: []client.TransformationVersionInput{{VersionID: "ver1"}},
		Libraries:       []client.LibraryVersionInput{},
	}

	_, err = c.Transformations.BatchPublish(ctx, req)
	require.Error(t, err)

	httpClient.AssertNumberOfCalls()
}

func TestTransformationsDeleteAPIError(t *testing.T) {
	ctx := context.Background()

	calls := []testutils.Call{
		{
			ResponseStatus: 404,
			ResponseBody:   `{"error": "Not Found"}`,
		},
	}

	httpClient := testutils.NewMockHTTPClient(t, calls...)

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	err = c.Transformations.Delete(ctx, "nonexistent")
	require.Error(t, err)

	httpClient.AssertNumberOfCalls()
}

func TestLibrariesDeleteAPIError(t *testing.T) {
	ctx := context.Background()

	calls := []testutils.Call{
		{
			ResponseStatus: 404,
			ResponseBody:   `{"error": "Not Found"}`,
		},
	}

	httpClient := testutils.NewMockHTTPClient(t, calls...)

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	err = c.TransformationLibraries.Delete(ctx, "nonexistent")
	require.Error(t, err)

	httpClient.AssertNumberOfCalls()
}
