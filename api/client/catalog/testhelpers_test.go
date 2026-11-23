package catalog_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/api/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const catalogAPIBase = "https://api.rudderstack.com/"

type callAsserter interface {
	AssertNumberOfCalls()
}

func catalogURL(path string) string {
	return catalogAPIBase + strings.TrimPrefix(path, "/")
}

// assertCall validates method, URL, and JSON body. testutils.ValidateRequest
// currently ignores its url argument, so we assert the URL here explicitly.
func assertCall(t *testing.T, req *http.Request, method, url, body string) bool {
	t.Helper()
	return assert.Equal(t, url, req.URL.String()) &&
		testutils.ValidateRequest(t, req, method, url, body)
}

func boolPtr(b bool) *bool {
	return &b
}

func newDataCatalog(t *testing.T, calls ...testutils.Call) (catalog.DataCatalog, callAsserter) {
	return newDataCatalogWithOptions(t, nil, calls...)
}

func newDataCatalogWithOptions(t *testing.T, opts []catalog.Opts, calls ...testutils.Call) (catalog.DataCatalog, callAsserter) {
	t.Helper()

	httpClient := testutils.NewMockHTTPClient(t, calls...)
	apiClient, err := client.New("test-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	allOpts := append([]catalog.Opts{catalog.WithConcurrency(1)}, opts...)
	dataCatalog, err := catalog.NewRudderDataCatalog(apiClient, allOpts...)
	require.NoError(t, err)

	return dataCatalog, httpClient
}
