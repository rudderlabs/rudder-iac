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

func TestNonJSONAPIErrorPreservesHTTPStatus(t *testing.T) {
	httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
		ResponseStatus: http.StatusGatewayTimeout,
		ResponseBody:   "upstream request timed out",
	})
	c, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)
	_, err = c.Do(context.Background(), http.MethodPost, "/path", nil)
	require.Error(t, err)
	var apiErr *client.APIError
	require.True(t, errors.As(err, &apiErr), "the status must survive response parsing")
	assert.Equal(t, http.StatusGatewayTimeout, apiErr.HTTPStatusCode)
	assert.ErrorContains(t, err, "upstream request timed out")
}
