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

func TestSubmitSourcePreview(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		t.Parallel()

		request := &retl.PreviewSubmitRequest{
			AccountID:    "acc123",
			FetchRows:    true,
			FetchColumns: true,
			RowLimit:     100,
			SQL:          "SELECT * FROM users LIMIT 100",
			WorkspaceID:  "ws123",
		}

		httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
			Validate: func(req *http.Request) bool {
				expected := `{"accountId":"acc123","fetchRows":true,"fetchColumns":true,"rowLimit":100,"sql":"SELECT * FROM users LIMIT 100","workspaceId":"ws123"}`
				return testutils.ValidateRequest(t, req, "POST", "https://api.rudderstack.com/v2/retl-sources/preview/role/test-role/submit", expected)
			},
			ResponseStatus: 200,
			ResponseBody: `{
				"data": {
					"requestId": "req123",
					"error": null
				},
				"success": true
			}`,
		})

		c, err := client.New("test-token", client.WithHTTPClient(httpClient))
		require.NoError(t, err)

		retlClient := retl.NewRudderRETLStore(c)

		response, err := retlClient.SubmitSourcePreview(context.Background(), "test-role", request)
		require.NoError(t, err)

		assert.True(t, response.Success)
		assert.Equal(t, "req123", response.Data.RequestID)
		assert.Nil(t, response.Data.Error)

		httpClient.AssertNumberOfCalls()
	})

	t.Run("WithError", func(t *testing.T) {
		t.Parallel()

		request := &retl.PreviewSubmitRequest{
			AccountID:    "acc123",
			FetchRows:    true,
			FetchColumns: true,
			RowLimit:     100,
			SQL:          "SELECT * FROM users LIMIT 100",
			WorkspaceID:  "ws123",
		}

		httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
			Validate: func(req *http.Request) bool {
				expected := `{"accountId":"acc123","fetchRows":true,"fetchColumns":true,"rowLimit":100,"sql":"SELECT * FROM users LIMIT 100","workspaceId":"ws123"}`
				return testutils.ValidateRequest(t, req, "POST", "https://api.rudderstack.com/v2/retl-sources/preview/role/test-role/submit", expected)
			},
			ResponseStatus: 200,
			ResponseBody: `{
				"data": {
					"requestId": "req123",
					"error": {
						"message": "SQL syntax error",
						"code": "SQL_ERROR"
					}
				},
				"success": false
			}`,
		})

		c, err := client.New("test-token", client.WithHTTPClient(httpClient))
		require.NoError(t, err)

		retlClient := retl.NewRudderRETLStore(c)

		response, err := retlClient.SubmitSourcePreview(context.Background(), "test-role", request)
		require.NoError(t, err)

		assert.False(t, response.Success)
		assert.Equal(t, "req123", response.Data.RequestID)
		assert.NotNil(t, response.Data.Error)
		assert.Equal(t, "SQL syntax error", response.Data.Error.Message)
		assert.Equal(t, "SQL_ERROR", response.Data.Error.Code)

		httpClient.AssertNumberOfCalls()
	})

	t.Run("EmptyRole", func(t *testing.T) {
		t.Parallel()

		request := &retl.PreviewSubmitRequest{
			AccountID:    "acc123",
			FetchRows:    true,
			FetchColumns: true,
			RowLimit:     100,
			SQL:          "SELECT * FROM users LIMIT 100",
			WorkspaceID:  "ws123",
		}

		httpClient := testutils.NewMockHTTPClient(t)
		c, err := client.New("test-token", client.WithHTTPClient(httpClient))
		require.NoError(t, err)

		retlClient := retl.NewRudderRETLStore(c)

		response, err := retlClient.SubmitSourcePreview(context.Background(), "", request)
		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "role cannot be empty")
	})

	t.Run("HTTPError", func(t *testing.T) {
		t.Parallel()

		request := &retl.PreviewSubmitRequest{
			AccountID:    "acc123",
			FetchRows:    true,
			FetchColumns: true,
			RowLimit:     100,
			SQL:          "SELECT * FROM users LIMIT 100",
			WorkspaceID:  "ws123",
		}

		httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
			Validate: func(req *http.Request) bool {
				expected := `{"accountId":"acc123","fetchRows":true,"fetchColumns":true,"rowLimit":100,"sql":"SELECT * FROM users LIMIT 100","workspaceId":"ws123"}`
				return testutils.ValidateRequest(t, req, "POST", "https://api.rudderstack.com/v2/retl-sources/preview/role/test-role/submit", expected)
			},
			ResponseError: assert.AnError,
		})

		c, err := client.New("test-token", client.WithHTTPClient(httpClient))
		require.NoError(t, err)

		retlClient := retl.NewRudderRETLStore(c)

		response, err := retlClient.SubmitSourcePreview(context.Background(), "test-role", request)
		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "submitting source preview")

		httpClient.AssertNumberOfCalls()
	})

	t.Run("InvalidJSONResponse", func(t *testing.T) {
		t.Parallel()

		request := &retl.PreviewSubmitRequest{
			AccountID:    "acc123",
			FetchRows:    true,
			FetchColumns: true,
			RowLimit:     100,
			SQL:          "SELECT * FROM users LIMIT 100",
			WorkspaceID:  "ws123",
		}

		httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
			Validate: func(req *http.Request) bool {
				expected := `{"accountId":"acc123","fetchRows":true,"fetchColumns":true,"rowLimit":100,"sql":"SELECT * FROM users LIMIT 100","workspaceId":"ws123"}`
				return testutils.ValidateRequest(t, req, "POST", "https://api.rudderstack.com/v2/retl-sources/preview/role/test-role/submit", expected)
			},
			ResponseStatus: 200,
			ResponseBody:   `{"invalid": json}`,
		})

		c, err := client.New("test-token", client.WithHTTPClient(httpClient))
		require.NoError(t, err)

		retlClient := retl.NewRudderRETLStore(c)

		response, err := retlClient.SubmitSourcePreview(context.Background(), "test-role", request)
		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "unmarshalling response")

		httpClient.AssertNumberOfCalls()
	})

}

func TestGetSourcePreviewResult(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		t.Parallel()

		httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/v2/retl-sources/preview/role/test-role/result/req123", "")
			},
			ResponseStatus: 200,
			ResponseBody: `{
				"data": {
					"state": "completed",
					"result": {
						"success": true,
						"errorDetails": null,
						"data": {
							"columns": [
								{
									"name": "id",
									"type": "integer",
									"rawType": "int4"
								},
								{
									"name": "name",
									"type": "string",
									"rawType": "varchar"
								}
							],
							"rows": [
								{
									"id": 1,
									"name": "John Doe"
								},
								{
									"id": 2,
									"name": "Jane Smith"
								}
							],
							"rowCount": 2
						}
					}
				},
				"success": true
			}`,
		})

		c, err := client.New("test-token", client.WithHTTPClient(httpClient))
		require.NoError(t, err)

		retlClient := retl.NewRudderRETLStore(c)

		response, err := retlClient.GetSourcePreviewResult(context.Background(), "test-role", "req123")
		require.NoError(t, err)

		assert.True(t, response.Success)
		assert.Equal(t, "completed", response.Data.State)
		assert.True(t, response.Data.Result.Success)
		assert.Nil(t, response.Data.Result.ErrorDetails)
		assert.NotNil(t, response.Data.Result.Data)
		assert.Equal(t, 2, len(response.Data.Result.Data.Columns))
		assert.Equal(t, 2, len(response.Data.Result.Data.Rows))
		assert.Equal(t, 2, response.Data.Result.Data.RowCount)

		// Check column details
		assert.Equal(t, "id", response.Data.Result.Data.Columns[0].Name)
		assert.Equal(t, "integer", response.Data.Result.Data.Columns[0].Type)
		assert.Equal(t, "int4", response.Data.Result.Data.Columns[0].RawType)

		assert.Equal(t, "name", response.Data.Result.Data.Columns[1].Name)
		assert.Equal(t, "string", response.Data.Result.Data.Columns[1].Type)
		assert.Equal(t, "varchar", response.Data.Result.Data.Columns[1].RawType)

		// Check row data
		assert.Equal(t, float64(1), response.Data.Result.Data.Rows[0]["id"])
		assert.Equal(t, "John Doe", response.Data.Result.Data.Rows[0]["name"])
		assert.Equal(t, float64(2), response.Data.Result.Data.Rows[1]["id"])
		assert.Equal(t, "Jane Smith", response.Data.Result.Data.Rows[1]["name"])

		httpClient.AssertNumberOfCalls()
	})

	t.Run("WithError", func(t *testing.T) {
		t.Parallel()

		httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/v2/retl-sources/preview/role/test-role/result/req123", "")
			},
			ResponseStatus: 200,
			ResponseBody: `{
				"data": {
					"state": "failed",
					"result": {
						"success": false,
						"errorDetails": {
							"message": "Database connection failed",
							"code": "DB_CONNECTION_ERROR"
						},
						"data": null
					}
				},
				"success": false
			}`,
		})

		c, err := client.New("test-token", client.WithHTTPClient(httpClient))
		require.NoError(t, err)

		retlClient := retl.NewRudderRETLStore(c)

		response, err := retlClient.GetSourcePreviewResult(context.Background(), "test-role", "req123")
		require.NoError(t, err)

		assert.False(t, response.Success)
		assert.Equal(t, "failed", response.Data.State)
		assert.False(t, response.Data.Result.Success)
		assert.NotNil(t, response.Data.Result.ErrorDetails)
		assert.Equal(t, "Database connection failed", response.Data.Result.ErrorDetails.Message)
		assert.Equal(t, "DB_CONNECTION_ERROR", response.Data.Result.ErrorDetails.Code)
		assert.Nil(t, response.Data.Result.Data)

		httpClient.AssertNumberOfCalls()
	})

	t.Run("EmptyRole", func(t *testing.T) {
		t.Parallel()

		httpClient := testutils.NewMockHTTPClient(t)
		c, err := client.New("test-token", client.WithHTTPClient(httpClient))
		require.NoError(t, err)

		retlClient := retl.NewRudderRETLStore(c)

		response, err := retlClient.GetSourcePreviewResult(context.Background(), "", "req123")
		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "role cannot be empty")
	})

	t.Run("EmptyResultID", func(t *testing.T) {
		t.Parallel()

		httpClient := testutils.NewMockHTTPClient(t)
		c, err := client.New("test-token", client.WithHTTPClient(httpClient))
		require.NoError(t, err)

		retlClient := retl.NewRudderRETLStore(c)

		response, err := retlClient.GetSourcePreviewResult(context.Background(), "test-role", "")
		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "result ID cannot be empty")
	})

	t.Run("HTTPError", func(t *testing.T) {
		t.Parallel()

		httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/v2/retl-sources/preview/role/test-role/result/req123", "")
			},
			ResponseError: assert.AnError,
		})

		c, err := client.New("test-token", client.WithHTTPClient(httpClient))
		require.NoError(t, err)

		retlClient := retl.NewRudderRETLStore(c)

		response, err := retlClient.GetSourcePreviewResult(context.Background(), "test-role", "req123")
		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "getting source preview result")

		httpClient.AssertNumberOfCalls()
	})

	t.Run("InvalidJSONResponse", func(t *testing.T) {
		t.Parallel()

		httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/v2/retl-sources/preview/role/test-role/result/req123", "")
			},
			ResponseStatus: 200,
			ResponseBody:   `{"invalid": json}`,
		})

		c, err := client.New("test-token", client.WithHTTPClient(httpClient))
		require.NoError(t, err)

		retlClient := retl.NewRudderRETLStore(c)

		response, err := retlClient.GetSourcePreviewResult(context.Background(), "test-role", "req123")
		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "unmarshalling response")

		httpClient.AssertNumberOfCalls()
	})

	t.Run("EmptyData", func(t *testing.T) {
		t.Parallel()

		httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/v2/retl-sources/preview/role/test-role/result/req123", "")
			},
			ResponseStatus: 200,
			ResponseBody: `{
				"data": {
					"state": "completed",
					"result": {
						"success": true,
						"errorDetails": null,
						"data": {
							"columns": [],
							"rows": [],
							"rowCount": 0
						}
					}
				},
				"success": true
			}`,
		})

		c, err := client.New("test-token", client.WithHTTPClient(httpClient))
		require.NoError(t, err)

		retlClient := retl.NewRudderRETLStore(c)

		response, err := retlClient.GetSourcePreviewResult(context.Background(), "test-role", "req123")
		require.NoError(t, err)

		assert.True(t, response.Success)
		assert.Equal(t, "completed", response.Data.State)
		assert.True(t, response.Data.Result.Success)
		assert.Nil(t, response.Data.Result.ErrorDetails)
		assert.NotNil(t, response.Data.Result.Data)
		assert.Equal(t, 0, len(response.Data.Result.Data.Columns))
		assert.Equal(t, 0, len(response.Data.Result.Data.Rows))
		assert.Equal(t, 0, response.Data.Result.Data.RowCount)

		httpClient.AssertNumberOfCalls()
	})
}
