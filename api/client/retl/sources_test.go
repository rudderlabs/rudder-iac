package retl_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/rudderlabs/rudder-iac/api/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateRetlSource(t *testing.T) {
	sourceConfig := retl.RETLSQLModelConfig{
		PrimaryKey:  "id",
		Sql:         "SELECT * FROM users",
		Description: "Test source",
	}
	externalID := "ext-123"

	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			expected := `{"name":"Test Source","config":{"primaryKey":"id","sql":"SELECT * FROM users","description":"Test source"},"sourceType":"model","sourceDefinitionName":"postgres","accountId":"acc123", "enabled":true, "externalId":"ext-123"}`
			return testutils.ValidateRequest(t, req, "POST", "https://api.rudderstack.com/v2/retl-sources", expected)
		},
		ResponseStatus: 200,
		ResponseBody: `{
			"id": "src1",
			"name": "Test Source",
			"config": {"primaryKey":"id","sql":"SELECT * FROM users","description":"Test source"},
			"enabled": true,
			"sourceType": "model",
			"sourceDefinitionName": "postgres",
			"accountId": "acc123",
			"createdAt": "2023-07-01T12:00:00Z",
			"updatedAt": "2023-07-01T12:00:00Z",
			"externalId": "ext-123"
		}`,
	})

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	retlClient := retl.NewRudderRETLStore(c)

	source := &retl.RETLSourceCreateRequest{
		Name:                 "Test Source",
		Config:               sourceConfig,
		SourceType:           retl.ModelSourceType,
		SourceDefinitionName: "postgres",
		AccountID:            "acc123",
		Enabled:              true,
		ExternalID:           &externalID,
	}

	created, err := retlClient.CreateRetlSource(context.Background(), source)
	require.NoError(t, err)

	assert.Equal(t, "src1", created.ID)
	assert.Equal(t, "Test Source", created.Name)
	assert.Equal(t, sourceConfig, created.Config)
	assert.Equal(t, true, created.IsEnabled)
	assert.Equal(t, retl.ModelSourceType, created.SourceType)
	assert.Equal(t, "postgres", created.SourceDefinitionName)
	assert.Equal(t, "acc123", created.AccountID)
	assert.NotNil(t, created.CreatedAt)
	assert.NotNil(t, created.UpdatedAt)

	httpClient.AssertNumberOfCalls()
}

func TestUpdateRetlSource(t *testing.T) {
	sourceConfig := retl.RETLSQLModelConfig{
		PrimaryKey:  "id",
		Sql:         "SELECT * FROM users",
		Description: "Test source",
	}

	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			expected := `{"name":"Updated Source","config":{"primaryKey":"id","sql":"SELECT * FROM users","description":"Test source"},"enabled":true,"accountId":"acc123"}`
			return testutils.ValidateRequest(t, req, "PUT", "https://api.rudderstack.com/v2/retl-sources/src1", expected)
		},
		ResponseStatus: 200,
		ResponseBody: `{
			"id": "src1",
			"name": "Updated Source",
			"config": {"primaryKey":"id","sql":"SELECT * FROM users","description":"Test source"},
			"enabled": true,
			"sourceType": "model",
			"sourceDefinitionName": "postgres",
			"accountId": "acc123",
			"createdAt": "2023-07-01T12:00:00Z",
			"updatedAt": "2023-07-01T13:00:00Z"
		}`,
	})

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	retlClient := retl.NewRudderRETLStore(c)

	source := &retl.RETLSourceUpdateRequest{
		Name:      "Updated Source",
		Config:    sourceConfig,
		IsEnabled: true,
		AccountID: "acc123",
	}

	updated, err := retlClient.UpdateRetlSource(context.Background(), "src1", source)
	require.NoError(t, err)

	assert.Equal(t, "src1", updated.ID)
	assert.Equal(t, "Updated Source", updated.Name)
	assert.Equal(t, sourceConfig, updated.Config)
	assert.Equal(t, true, updated.IsEnabled)
	assert.Equal(t, retl.ModelSourceType, updated.SourceType)
	assert.Equal(t, "postgres", updated.SourceDefinitionName)
	assert.Equal(t, "acc123", updated.AccountID)

	httpClient.AssertNumberOfCalls()
}

func TestGetRetlSource(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/v2/retl-sources/src1", "")
		},
		ResponseStatus: 200,
		ResponseBody: `{
			"id": "src1",
			"name": "Test Source",
			"config": {"primaryKey":"id","sql":"SELECT * FROM users","description":"Test source"},
			"enabled": true,
			"sourceType": "model",
			"sourceDefinitionName": "postgres",
			"accountId": "acc123",
			"createdAt": "2023-07-01T12:00:00Z",
			"updatedAt": "2023-07-01T12:00:00Z"
		}`,
	})

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	retlClient := retl.NewRudderRETLStore(c)

	source, err := retlClient.GetRetlSource(context.Background(), "src1")
	require.NoError(t, err)

	assert.Equal(t, "src1", source.ID)
	assert.Equal(t, "Test Source", source.Name)
	assert.Equal(t, true, source.IsEnabled)
	assert.Equal(t, retl.ModelSourceType, source.SourceType)
	assert.Equal(t, "postgres", source.SourceDefinitionName)
	assert.Equal(t, "acc123", source.AccountID)

	httpClient.AssertNumberOfCalls()
}

func TestDeleteRetlSource(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return testutils.ValidateRequest(t, req, "DELETE", "https://api.rudderstack.com/v2/retl-sources/src1", "")
		},
		ResponseStatus: 204,
		ResponseBody:   "",
	})

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	retlClient := retl.NewRudderRETLStore(c)

	err = retlClient.DeleteRetlSource(context.Background(), "src1")
	require.NoError(t, err)

	httpClient.AssertNumberOfCalls()
}

func TestListRetlSources(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/v2/retl-sources", "")
		},
		ResponseStatus: 200,
		ResponseBody: `{
			"data": [
				{
					"id": "src1",
					"name": "Source 1",
					"enabled": true,
					"sourceType": "model",
					"sourceDefinitionName": "postgres",
					"accountId": "acc123"
				},
				{
					"id": "src2",
					"name": "Source 2",
					"enabled": false,
					"sourceType": "model",
					"sourceDefinitionName": "mysql",
					"accountId": "acc123"
				}
			]
		}`,
	})

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	retlClient := retl.NewRudderRETLStore(c)

	sources, err := retlClient.ListRetlSources(context.Background(), nil)
	require.NoError(t, err)

	assert.Len(t, sources.Data, 2)
	assert.Equal(t, "src1", sources.Data[0].ID)
	assert.Equal(t, "Source 1", sources.Data[0].Name)
	assert.Equal(t, true, sources.Data[0].IsEnabled)
	assert.Equal(t, retl.ModelSourceType, sources.Data[0].SourceType)
	assert.Equal(t, "postgres", sources.Data[0].SourceDefinitionName)
	assert.Equal(t, "acc123", sources.Data[0].AccountID)

	assert.Equal(t, "src2", sources.Data[1].ID)
	assert.Equal(t, "Source 2", sources.Data[1].Name)
	assert.Equal(t, false, sources.Data[1].IsEnabled)
	assert.Equal(t, retl.ModelSourceType, sources.Data[1].SourceType)
	assert.Equal(t, "mysql", sources.Data[1].SourceDefinitionName)
	assert.Equal(t, "acc123", sources.Data[1].AccountID)

	httpClient.AssertNumberOfCalls()
}

func TestNextRetlSources(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/v2/retl-sources", "")
		},
		ResponseStatus: 200,
		ResponseBody: `{
			"data": [
				{
					"id": "src3",
					"name": "Source 3",
					"enabled": true,
					"sourceType": "model",
					"sourceDefinitionName": "postgres",
					"accountId": "acc123"
				}
			]
		}`,
	})

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	retlClient := retl.NewRudderRETLStore(c)

	sources, err := retlClient.ListRetlSources(context.Background(), nil)
	require.NoError(t, err)

	assert.Len(t, sources.Data, 1)
	assert.Equal(t, "src3", sources.Data[0].ID)
	assert.Equal(t, "Source 3", sources.Data[0].Name)
	assert.Equal(t, true, sources.Data[0].IsEnabled)
	assert.Equal(t, retl.ModelSourceType, sources.Data[0].SourceType)
	assert.Equal(t, "postgres", sources.Data[0].SourceDefinitionName)
	assert.Equal(t, "acc123", sources.Data[0].AccountID)

	httpClient.AssertNumberOfCalls()
}

func TestUpdateRetlSourceEmptyID(t *testing.T) {
	c, err := client.New("test-token")
	require.NoError(t, err)

	retlClient := retl.NewRudderRETLStore(c)

	source := &retl.RETLSourceUpdateRequest{
		Name:      "Updated Source",
		IsEnabled: true,
		AccountID: "acc123",
	}

	_, err = retlClient.UpdateRetlSource(context.Background(), "", source)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "source ID cannot be empty")
}

func TestDeleteRetlSourceEmptyID(t *testing.T) {
	c, err := client.New("test-token")
	require.NoError(t, err)

	retlClient := retl.NewRudderRETLStore(c)

	err = retlClient.DeleteRetlSource(context.Background(), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "source ID cannot be empty")
}

func TestGetRetlSourceEmptyID(t *testing.T) {
	c, err := client.New("test-token")
	require.NoError(t, err)

	retlClient := retl.NewRudderRETLStore(c)

	_, err = retlClient.GetRetlSource(context.Background(), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "source ID cannot be empty")
}

func TestCreateRetlSourceAPIError(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return testutils.ValidateRequest(t, req, "POST", "https://api.rudderstack.com/v2/retl-sources", "")
		},
		ResponseStatus: 400,
		ResponseBody:   `{"error":"Bad Request"}`,
	})

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	retlClient := retl.NewRudderRETLStore(c)

	sourceConfig := retl.RETLSQLModelConfig{
		PrimaryKey: "id",
		Sql:        "SELECT * FROM users",
	}

	source := &retl.RETLSourceCreateRequest{
		Name:                 "Test Source",
		Config:               sourceConfig,
		SourceType:           "postgres",
		SourceDefinitionName: "PostgreSQL",
		AccountID:            "acc123",
	}

	_, err = retlClient.CreateRetlSource(context.Background(), source)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "creating RETL source")

	httpClient.AssertNumberOfCalls()
}

func TestGetRetlSourceMalformedResponse(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/v2/retl-sources/src1", "")
		},
		ResponseStatus: 200,
		ResponseBody:   `{malformed_json`,
	})

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	retlClient := retl.NewRudderRETLStore(c)

	_, err = retlClient.GetRetlSource(context.Background(), "src1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshalling response")

	httpClient.AssertNumberOfCalls()
}

func TestListRetlSourcesAPIError(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/v2/retl-sources", "")
		},
		ResponseStatus: 500,
		ResponseBody:   `{"error":"Internal Server Error"}`,
	})

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	retlClient := retl.NewRudderRETLStore(c)

	_, err = retlClient.ListRetlSources(context.Background(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "listing RETL sources")

	httpClient.AssertNumberOfCalls()
}

func TestUpdateRetlSourceMalformedResponse(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return testutils.ValidateRequest(t, req, "PUT", "https://api.rudderstack.com/v2/retl-sources/src1", "")
		},
		ResponseStatus: 200,
		ResponseBody:   `{invalid_json`,
	})

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	retlClient := retl.NewRudderRETLStore(c)

	source := &retl.RETLSourceUpdateRequest{
		Name:      "Updated Source",
		IsEnabled: true,
		AccountID: "acc123",
	}

	_, err = retlClient.UpdateRetlSource(context.Background(), "src1", source)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshalling response")

	httpClient.AssertNumberOfCalls()
}

func TestListRetlSourcesMalformedResponse(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/v2/retl-sources", "")
		},
		ResponseStatus: 200,
		ResponseBody:   `{invalid_json`,
	})

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	retlClient := retl.NewRudderRETLStore(c)

	_, err = retlClient.ListRetlSources(context.Background(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshalling response")

	httpClient.AssertNumberOfCalls()
}

func TestDeleteRetlSourceAPIError(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return testutils.ValidateRequest(t, req, "DELETE", "https://api.rudderstack.com/v2/retl-sources/src1", "")
		},
		ResponseStatus: 500,
		ResponseBody:   `{"error":"Internal Server Error"}`,
	})

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	retlClient := retl.NewRudderRETLStore(c)

	err = retlClient.DeleteRetlSource(context.Background(), "src1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "deleting RETL source")

	httpClient.AssertNumberOfCalls()
}

func TestCreateRetlSourceMalformedResponse(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return testutils.ValidateRequest(t, req, "POST", "https://api.rudderstack.com/v2/retl-sources", "")
		},
		ResponseStatus: 200,
		ResponseBody:   `{invalid_json`,
	})

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	retlClient := retl.NewRudderRETLStore(c)

	sourceConfig := retl.RETLSQLModelConfig{
		PrimaryKey: "id",
		Sql:        "SELECT * FROM users",
	}

	source := &retl.RETLSourceCreateRequest{
		Name:                 "Test Source",
		Config:               sourceConfig,
		SourceType:           "postgres",
		SourceDefinitionName: "PostgreSQL",
		AccountID:            "acc123",
	}

	_, err = retlClient.CreateRetlSource(context.Background(), source)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshalling response")

	httpClient.AssertNumberOfCalls()
}

func TestCreateRetlSourceInvalidRequest(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return testutils.ValidateRequest(t, req, "POST", "https://api.rudderstack.com/v2/retl-sources", "")
		},
		ResponseStatus: 400,
		ResponseBody:   `{"error":"Invalid request"}`,
	})

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	retlClient := retl.NewRudderRETLStore(c)

	source := &retl.RETLSourceCreateRequest{
		// Missing required fields
	}

	_, err = retlClient.CreateRetlSource(context.Background(), source)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "creating RETL source")

	httpClient.AssertNumberOfCalls()
}

func TestGetRetlSourceAPIError(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/v2/retl-sources/src1", "")
		},
		ResponseStatus: 500,
		ResponseBody:   `{"error":"Internal Server Error"}`,
	})

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	retlClient := retl.NewRudderRETLStore(c)

	_, err = retlClient.GetRetlSource(context.Background(), "src1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "getting RETL source")

	httpClient.AssertNumberOfCalls()
}

func TestUpdateRetlSourceAPIError(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return testutils.ValidateRequest(t, req, "PUT", "https://api.rudderstack.com/v2/retl-sources/src1", "")
		},
		ResponseStatus: 500,
		ResponseBody:   `{"error":"Internal Server Error"}`,
	})

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	retlClient := retl.NewRudderRETLStore(c)

	source := &retl.RETLSourceUpdateRequest{
		Name:      "Updated Source",
		IsEnabled: true,
		AccountID: "acc123",
	}

	_, err = retlClient.UpdateRetlSource(context.Background(), "src1", source)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "updating RETL source")

	httpClient.AssertNumberOfCalls()
}
