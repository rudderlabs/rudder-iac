package catalog_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/api/client/catalog"
)

func TestGetCategoriesPaginatesInPageOrder(t *testing.T) {
	var (
		mu       sync.Mutex
		requests = map[string]int{}
	)

	// Three pages of two categories, with the middle page the slowest, so the
	// requests finish in a different order than the pages.
	pages := map[string][]string{"": {"c0", "c1"}, "2": {"c2", "c3"}, "3": {"c4"}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		page := req.URL.Query().Get("page")

		mu.Lock()
		requests[page]++
		mu.Unlock()

		if page == "2" {
			time.Sleep(50 * time.Millisecond)
		}
		items := ""
		for i, id := range pages[page] {
			if i > 0 {
				items += ","
			}
			items += fmt.Sprintf(`{"id": %q, "workspaceId": "ws"}`, id)
		}
		fmt.Fprintf(w, `{"data": [%s], "total": 5, "currentPage": 1, "pageSize": 2}`, items)
	}))
	defer server.Close()

	apiClient, err := client.New("test-token", client.WithBaseURL(server.URL))
	require.NoError(t, err)
	dataCatalog, err := catalog.NewRudderDataCatalog(apiClient, catalog.WithConcurrency(3))
	require.NoError(t, err)

	categories, err := dataCatalog.GetCategories(context.Background(), catalog.ListOptions{})
	require.NoError(t, err)

	ids := make([]string, len(categories))
	for i, c := range categories {
		ids[i] = c.ID
	}
	assert.Equal(t, []string{"c0", "c1", "c2", "c3", "c4"}, ids)
	assert.Equal(t, map[string]int{"": 1, "2": 1, "3": 1}, requests, "each page is requested once, including the first")
}
