package catalog_test

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/api/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type callAsserter interface {
	AssertNumberOfCalls()
}

func newDataCatalog(t *testing.T, opts []catalog.Opts, calls ...testutils.Call) (catalog.DataCatalog, callAsserter) {
	t.Helper()

	httpClient := testutils.NewMockHTTPClient(t, calls...)
	apiClient, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	allOpts := append([]catalog.Opts{catalog.WithConcurrency(1)}, opts...)
	dataCatalog, err := catalog.NewRudderDataCatalog(apiClient, allOpts...)
	require.NoError(t, err)

	return dataCatalog, httpClient
}

func validateRequest(t *testing.T, req *http.Request, method, endpoint, body string) bool {
	t.Helper()

	if !assert.Equal(t, method, req.Method) {
		return false
	}

	expectedURL := "https://api.rudderstack.com/" + strings.TrimPrefix(endpoint, "/")
	if !assert.Equal(t, expectedURL, req.URL.String()) {
		return false
	}

	if body == "" {
		return true
	}

	bodyBytes, err := io.ReadAll(req.Body)
	require.NoError(t, err)
	if !assert.JSONEq(t, body, string(bodyBytes)) {
		return false
	}

	return true
}
