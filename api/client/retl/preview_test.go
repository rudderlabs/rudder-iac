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
			AccountID: "acc123",
			Limit:     100,
			SQL:       "SELECT * FROM users LIMIT 100",
		}

		httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
			Validate: func(req *http.Request) bool {
				expected := `{"accountId":"acc123","limit":100,"sql":"SELECT * FROM users LIMIT 100"}`
				return testutils.ValidateRequest(t, req, "POST", "https://api.rudderstack.com/v2/retl-sources/preview", expected)
			},
			ResponseStatus: 200,
			ResponseBody: `{
				"id": "req123"
			}`,
		})

		c, err := client.New("test-token", client.WithHTTPClient(httpClient))
		require.NoError(t, err)

		retlClient := retl.NewRudderRETLStore(c)

		response, err := retlClient.SubmitSourcePreview(context.Background(), request)
		require.NoError(t, err)

		assert.Equal(t, "req123", response.ID)

		httpClient.AssertNumberOfCalls()
	})

	t.Run("HTTPError", func(t *testing.T) {
		t.Parallel()

		request := &retl.PreviewSubmitRequest{
			AccountID: "acc123",
			Limit:     100,
			SQL:       "SELECT * FROM users LIMIT 100",
		}

		httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
			Validate: func(req *http.Request) bool {
				expected := `{"accountId":"acc123","limit":100,"sql":"SELECT * FROM users LIMIT 100"}`
				return testutils.ValidateRequest(t, req, "POST", "https://api.rudderstack.com/v2/retl-sources/preview", expected)
			},
			ResponseError: assert.AnError,
		})

		c, err := client.New("test-token", client.WithHTTPClient(httpClient))
		require.NoError(t, err)

		retlClient := retl.NewRudderRETLStore(c)

		response, err := retlClient.SubmitSourcePreview(context.Background(), request)
		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "submitting source preview")

		httpClient.AssertNumberOfCalls()
	})

	t.Run("InvalidJSONResponse", func(t *testing.T) {
		t.Parallel()

		request := &retl.PreviewSubmitRequest{
			AccountID: "acc123",
			Limit:     100,
			SQL:       "SELECT * FROM users LIMIT 100",
		}

		httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
			Validate: func(req *http.Request) bool {
				expected := `{"accountId":"acc123","limit":100,"sql":"SELECT * FROM users LIMIT 100"}`
				return testutils.ValidateRequest(t, req, "POST", "https://api.rudderstack.com/v2/retl-sources/preview", expected)
			},
			ResponseStatus: 200,
			ResponseBody:   `{"invalid": json}`,
		})

		c, err := client.New("test-token", client.WithHTTPClient(httpClient))
		require.NoError(t, err)

		retlClient := retl.NewRudderRETLStore(c)

		response, err := retlClient.SubmitSourcePreview(context.Background(), request)
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
				return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/v2/retl-sources/preview/req123", "")
			},
			ResponseStatus: 200,
			ResponseBody: `{
				"status": "completed",
				"rows": [
					{
						"id": 1,
						"name": "John Doe"
					},
					{
						"id": 2,
						"name": "Jane Smith"
					}
				]
			}`,
		})

		c, err := client.New("test-token", client.WithHTTPClient(httpClient))
		require.NoError(t, err)

		retlClient := retl.NewRudderRETLStore(c)

		response, err := retlClient.GetSourcePreviewResult(context.Background(), "req123")
		require.NoError(t, err)

		assert.Equal(t, retl.Completed, response.Status)
		assert.Equal(t, 2, len(response.Rows))
		assert.Equal(t, "John Doe", response.Rows[0]["name"])
		assert.Equal(t, "Jane Smith", response.Rows[1]["name"])
		assert.Empty(t, response.Error)

		httpClient.AssertNumberOfCalls()
	})

	t.Run("WithError", func(t *testing.T) {
		t.Parallel()

		httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/v2/retl-sources/preview/req123", "")
			},
			ResponseStatus: 200,
			ResponseBody: `{
				"status": "failed",
				"error": "Database connection failed"
			}`,
		})

		c, err := client.New("test-token", client.WithHTTPClient(httpClient))
		require.NoError(t, err)

		retlClient := retl.NewRudderRETLStore(c)

		response, err := retlClient.GetSourcePreviewResult(context.Background(), "req123")
		require.NoError(t, err)

		assert.Equal(t, retl.Failed, response.Status)
		assert.Equal(t, "Database connection failed", response.Error)

		httpClient.AssertNumberOfCalls()
	})

	// This test case is for checking the request building
	t.Run("TestRequestBuilding", func(t *testing.T) {
		t.Parallel()

		httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/v2/retl-sources/preview/req123", "")
			},
			ResponseStatus: 200,
			ResponseBody: `{
				"status": "pending"
			}`,
		})
		c, err := client.New("test-token", client.WithHTTPClient(httpClient))
		require.NoError(t, err)

		retlClient := retl.NewRudderRETLStore(c)

		response, err := retlClient.GetSourcePreviewResult(context.Background(), "req123")
		require.NoError(t, err)
		assert.Equal(t, retl.Pending, response.Status)

		httpClient.AssertNumberOfCalls()
	})

	t.Run("EmptyResultID", func(t *testing.T) {
		t.Parallel()

		httpClient := testutils.NewMockHTTPClient(t)
		c, err := client.New("test-token", client.WithHTTPClient(httpClient))
		require.NoError(t, err)

		retlClient := retl.NewRudderRETLStore(c)

		response, err := retlClient.GetSourcePreviewResult(context.Background(), "")
		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "result ID cannot be empty")
	})

	t.Run("HTTPError", func(t *testing.T) {
		t.Parallel()

		httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/v2/retl-sources/preview/req123", "")
			},
			ResponseError: assert.AnError,
		})

		c, err := client.New("test-token", client.WithHTTPClient(httpClient))
		require.NoError(t, err)

		retlClient := retl.NewRudderRETLStore(c)

		response, err := retlClient.GetSourcePreviewResult(context.Background(), "req123")
		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "getting source preview result")

		httpClient.AssertNumberOfCalls()
	})

	t.Run("InvalidJSONResponse", func(t *testing.T) {
		t.Parallel()

		httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/v2/retl-sources/preview/req123", "")
			},
			ResponseStatus: 200,
			ResponseBody:   `{"invalid": json}`,
		})

		c, err := client.New("test-token", client.WithHTTPClient(httpClient))
		require.NoError(t, err)

		retlClient := retl.NewRudderRETLStore(c)

		response, err := retlClient.GetSourcePreviewResult(context.Background(), "req123")
		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "unmarshalling response")

		httpClient.AssertNumberOfCalls()
	})

	t.Run("EmptyData", func(t *testing.T) {
		t.Parallel()

		httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/v2/retl-sources/preview/req123", "")
			},
			ResponseStatus: 200,
			ResponseBody: `{
				"status": "completed",
				"rows": []
			}`,
		})

		c, err := client.New("test-token", client.WithHTTPClient(httpClient))
		require.NoError(t, err)

		retlClient := retl.NewRudderRETLStore(c)

		response, err := retlClient.GetSourcePreviewResult(context.Background(), "req123")
		require.NoError(t, err)

		assert.Equal(t, retl.Completed, response.Status)
		assert.Equal(t, 0, len(response.Rows))
		assert.Empty(t, response.Error)

		httpClient.AssertNumberOfCalls()
	})
}
