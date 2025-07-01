package retl_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/rudderlabs/rudder-iac/api/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateRetlSource(t *testing.T) {
	sourceConfig := json.RawMessage(`{"host":"localhost","port":5432}`)

	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			expected := `{"name":"Test Source","config":{"host":"localhost","port":5432},"enabled":true,"sourceType":"postgres","sourceDefinitionName":"PostgreSQL","accountId":"acc123"}`
			return testutils.ValidateRequest(t, req, "POST", "https://api.rudderstack.com/v2/retl-sources", expected)
		},
		ResponseStatus: 200,
		ResponseBody: `{
			"source": {
				"id": "src1",
				"name": "Test Source",
				"config": {"host":"localhost","port":5432},
				"enabled": true,
				"sourceType": "postgres",
				"sourceDefinitionName": "PostgreSQL",
				"accountId": "acc123",
				"createdAt": "2023-07-01T12:00:00Z",
				"updatedAt": "2023-07-01T12:00:00Z"
			}
		}`,
	})

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	retlClient := retl.NewRudderRETLStore(c)

	source := &retl.RETLSource{
		Name:                 "Test Source",
		Config:               sourceConfig,
		IsEnabled:            true,
		SourceType:           "postgres",
		SourceDefinitionName: "PostgreSQL",
		AccountID:            "acc123",
	}

	created, err := retlClient.CreateRetlSource(context.Background(), source)
	require.NoError(t, err)

	assert.Equal(t, "src1", created.ID)
	assert.Equal(t, "Test Source", created.Name)
	assert.Equal(t, sourceConfig, created.Config)
	assert.Equal(t, true, created.IsEnabled)
	assert.Equal(t, "postgres", created.SourceType)
	assert.Equal(t, "PostgreSQL", created.SourceDefinitionName)
	assert.Equal(t, "acc123", created.AccountID)
	assert.NotNil(t, created.CreatedAt)
	assert.NotNil(t, created.UpdatedAt)

	httpClient.AssertNumberOfCalls()
}

func TestUpdateRetlSource(t *testing.T) {
	sourceConfig := json.RawMessage(`{"host":"localhost","port":5432,"dbname":"testdb"}`)

	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			expected := `{"id":"src1","name":"Updated Source","config":{"host":"localhost","port":5432,"dbname":"testdb"},"enabled":true,"sourceType":"postgres","sourceDefinitionName":"PostgreSQL","accountId":"acc123"}`
			return testutils.ValidateRequest(t, req, "PUT", "https://api.rudderstack.com/v2/retl-sources/src1", expected)
		},
		ResponseStatus: 200,
		ResponseBody: `{
			"source": {
				"id": "src1",
				"name": "Updated Source",
				"config": {"host":"localhost","port":5432,"dbname":"testdb"},
				"enabled": true,
				"sourceType": "postgres",
				"sourceDefinitionName": "PostgreSQL",
				"accountId": "acc123",
				"createdAt": "2023-07-01T12:00:00Z",
				"updatedAt": "2023-07-01T13:00:00Z"
			}
		}`,
	})

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	retlClient := retl.NewRudderRETLStore(c)

	source := &retl.RETLSource{
		ID:                   "src1",
		Name:                 "Updated Source",
		Config:               sourceConfig,
		IsEnabled:            true,
		SourceType:           "postgres",
		SourceDefinitionName: "PostgreSQL",
		AccountID:            "acc123",
	}

	updated, err := retlClient.UpdateRetlSource(context.Background(), source)
	require.NoError(t, err)

	assert.Equal(t, "src1", updated.ID)
	assert.Equal(t, "Updated Source", updated.Name)
	assert.Equal(t, sourceConfig, updated.Config)
	assert.Equal(t, true, updated.IsEnabled)

	httpClient.AssertNumberOfCalls()
}

func TestGetRetlSource(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/v2/retl-sources/src1", "")
		},
		ResponseStatus: 200,
		ResponseBody: `{
			"source": {
				"id": "src1",
				"name": "Test Source",
				"config": {"host":"localhost","port":5432},
				"enabled": true,
				"sourceType": "postgres",
				"sourceDefinitionName": "PostgreSQL",
				"accountId": "acc123",
				"createdAt": "2023-07-01T12:00:00Z",
				"updatedAt": "2023-07-01T12:00:00Z"
			}
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
	assert.Equal(t, "postgres", source.SourceType)

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
			"sources": [
				{
					"id": "src1",
					"name": "Source 1",
					"enabled": true,
					"sourceType": "postgres",
					"sourceDefinitionName": "PostgreSQL",
					"accountId": "acc123"
				},
				{
					"id": "src2",
					"name": "Source 2",
					"enabled": false,
					"sourceType": "mysql",
					"sourceDefinitionName": "MySQL",
					"accountId": "acc123"
				}
			]
		}`,
	})

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	retlClient := retl.NewRudderRETLStore(c)

	sources, err := retlClient.ListRetlSources(context.Background())
	require.NoError(t, err)

	assert.Len(t, sources.Sources, 2)
	assert.Equal(t, "src1", sources.Sources[0].ID)
	assert.Equal(t, "Source 1", sources.Sources[0].Name)
	assert.Equal(t, true, sources.Sources[0].IsEnabled)

	assert.Equal(t, "src2", sources.Sources[1].ID)
	assert.Equal(t, "Source 2", sources.Sources[1].Name)
	assert.Equal(t, false, sources.Sources[1].IsEnabled)

	httpClient.AssertNumberOfCalls()
}

func TestNextRetlSources(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/v2/retl-sources", "")
		},
		ResponseStatus: 200,
		ResponseBody: `{
			"sources": [
				{
					"id": "src3",
					"name": "Source 3",
					"enabled": true,
					"sourceType": "postgres",
					"sourceDefinitionName": "PostgreSQL",
					"accountId": "acc123"
				}
			]
		}`,
	})

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	retlClient := retl.NewRudderRETLStore(c)

	sources, err := retlClient.ListRetlSources(context.Background())
	require.NoError(t, err)

	assert.Len(t, sources.Sources, 1)
	assert.Equal(t, "src3", sources.Sources[0].ID)
	assert.Equal(t, "Source 3", sources.Sources[0].Name)

	httpClient.AssertNumberOfCalls()
}
