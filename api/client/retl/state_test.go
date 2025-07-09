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

func TestReadState(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/v2/cli/retl/state", "")
		},
		ResponseStatus: 200,
		ResponseBody: `{
			"version": "1.0",
			"resources": {
				"source1": {
					"id": "source1",
					"type": "retl_source",
					"input": {"name": "Source 1"},
					"output": {"id": "src1"},
					"dependencies": []
				},
				"source2": {
					"id": "source2",
					"type": "retl_source",
					"input": {"name": "Source 2"},
					"output": {"id": "src2"},
					"dependencies": ["source1"]
				}
			}
		}`,
	})

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	retlClient := retl.NewRudderRETLStore(c)
	state, err := retlClient.ReadState(context.Background())
	require.NoError(t, err)

	assert.Equal(t, "1.0", state.Version)
	assert.Len(t, state.Resources, 2)

	source1, exists := state.Resources["source1"]
	assert.True(t, exists)
	assert.Equal(t, "source1", source1.ID)
	assert.Equal(t, "retl_source", source1.Type)
	assert.Equal(t, "Source 1", source1.Input["name"])
	assert.Equal(t, "src1", source1.Output["id"])
	assert.Empty(t, source1.Dependencies)

	source2, exists := state.Resources["source2"]
	assert.True(t, exists)
	assert.Equal(t, "source2", source2.ID)
	assert.Equal(t, []string{"source1"}, source2.Dependencies)

	httpClient.AssertNumberOfCalls()
}

func TestReadStateAPIError(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/v2/cli/retl/state", "")
		},
		ResponseStatus: 500,
		ResponseBody:   `{"error":"Internal Server Error"}`,
	})

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	retlClient := retl.NewRudderRETLStore(c)
	_, err = retlClient.ReadState(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "sending read state request")

	httpClient.AssertNumberOfCalls()
}

func TestReadStateMalformedResponse(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/v2/cli/retl/state", "")
		},
		ResponseStatus: 200,
		ResponseBody:   `{invalid_json`,
	})

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	retlClient := retl.NewRudderRETLStore(c)
	_, err = retlClient.ReadState(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshalling response")

	httpClient.AssertNumberOfCalls()
}

func TestPutResourceState(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			expected := `{"urn":"retl:source:source1","state":{"id":"source1","type":"retl_source","input":{"name":"Source 1"},"output":{"id":"src1"},"dependencies":[]}}`
			return testutils.ValidateRequest(t, req, "PUT", "https://api.rudderstack.com/v2/cli/retl/state/source1", expected)
		},
		ResponseStatus: 200,
		ResponseBody:   `{}`,
	})

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	retlClient := retl.NewRudderRETLStore(c)

	req := retl.PutStateRequest{
		ID:  "source1",
		URN: "retl:source:source1",
		State: retl.ResourceState{
			ID:   "source1",
			Type: "retl_source",
			Input: map[string]interface{}{
				"name": "Source 1",
			},
			Output: map[string]interface{}{
				"id": "src1",
			},
			Dependencies: []string{},
		},
	}

	err = retlClient.PutResourceState(context.Background(), req)
	require.NoError(t, err)

	httpClient.AssertNumberOfCalls()
}

func TestPutResourceStateAPIError(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return testutils.ValidateRequest(t, req, "PUT", "https://api.rudderstack.com/v2/cli/retl/state/source1", "")
		},
		ResponseStatus: 500,
		ResponseBody:   `{"error":"Internal Server Error"}`,
	})

	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	retlClient := retl.NewRudderRETLStore(c)

	req := retl.PutStateRequest{
		ID:  "source1",
		URN: "retl:source:source1",
		State: retl.ResourceState{
			ID:   "source1",
			Type: "retl_source",
		},
	}

	err = retlClient.PutResourceState(context.Background(), req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "sending put state request")

	httpClient.AssertNumberOfCalls()
}

func TestPutResourceStateInvalidRequest(t *testing.T) {
	c, err := client.New("test-token")
	require.NoError(t, err)

	retlClient := retl.NewRudderRETLStore(c)

	req := retl.PutStateRequest{
		ID:  "source1",
		URN: "retl:source:source1",
		State: retl.ResourceState{
			Input: map[string]interface{}{
				"invalid": make(chan int), // This will cause json.Marshal to fail
			},
		},
	}

	err = retlClient.PutResourceState(context.Background(), req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "marshalling PUT request")
}
