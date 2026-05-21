package catalog_test

import (
	"context"
	"errors"
	"net/http"
	"sort"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/api/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCategoryCRUDAndExternalID(t *testing.T) {
	calls := []testutils.Call{
		{
			Validate: func(req *http.Request) bool {
				return validateRequest(t, req, "POST", "v2/catalog/categories", `{"name":"category","externalId":"ext-1"}`)
			},
			ResponseStatus: 200,
			ResponseBody:   `{"id":"c-1","name":"category","workspaceId":"ws-1"}`,
		},
		{
			Validate: func(req *http.Request) bool {
				return validateRequest(t, req, "PUT", "v2/catalog/categories/c-1", `{"name":"category-2"}`)
			},
			ResponseStatus: 200,
			ResponseBody:   `{"id":"c-1","name":"category-2","workspaceId":"ws-1"}`,
		},
		{
			Validate: func(req *http.Request) bool {
				return validateRequest(t, req, "GET", "v2/catalog/categories/c-1", "")
			},
			ResponseStatus: 200,
			ResponseBody:   `{"id":"c-1","name":"category-2","workspaceId":"ws-1"}`,
		},
		{
			Validate: func(req *http.Request) bool {
				return validateRequest(t, req, "PUT", "v2/catalog/categories/c-1/external-id", `{"externalId":"ext-2"}`)
			},
			ResponseStatus: 200,
			ResponseBody:   `{}`,
		},
		{
			Validate: func(req *http.Request) bool {
				return validateRequest(t, req, "DELETE", "v2/catalog/categories/c-1", "")
			},
			ResponseStatus: 200,
			ResponseBody:   `{}`,
		},
	}

	dc, httpClient := newDataCatalog(t, nil, calls...)
	cat, err := dc.CreateCategory(context.Background(), catalog.CategoryCreate{Name: "category", ExternalId: "ext-1"})
	require.NoError(t, err)
	assert.Equal(t, &catalog.Category{ID: "c-1", Name: "category", WorkspaceID: "ws-1"}, cat)

	cat, err = dc.UpdateCategory(context.Background(), "c-1", catalog.CategoryUpdate{Name: "category-2"})
	require.NoError(t, err)
	assert.Equal(t, "category-2", cat.Name)

	cat, err = dc.GetCategory(context.Background(), "c-1")
	require.NoError(t, err)
	assert.Equal(t, "c-1", cat.ID)

	err = dc.SetCategoryExternalId(context.Background(), "c-1", "ext-2")
	require.NoError(t, err)

	err = dc.DeleteCategory(context.Background(), "c-1")
	require.NoError(t, err)

	httpClient.AssertNumberOfCalls()
}

func TestGetCategoriesFilteringAndErrors(t *testing.T) {
	t.Run("filters categories without workspace id", func(t *testing.T) {
		calls := []testutils.Call{
			{ResponseStatus: 200, ResponseBody: `{"data":[{"id":"ignore"}],"total":2,"currentPage":1,"pageSize":1}`},
			{ResponseStatus: 200, ResponseBody: `{"data":[{"id":"c-1","workspaceId":"ws-1"}],"total":2,"currentPage":1,"pageSize":1}`},
			{ResponseStatus: 200, ResponseBody: `{"data":[{"id":"c-2","workspaceId":"ws-2"},{"id":"c-x"}],"total":2,"currentPage":2,"pageSize":1}`},
		}

		dc, httpClient := newDataCatalog(t, nil, calls...)
		cats, err := dc.GetCategories(context.Background(), catalog.ListOptions{})
		require.NoError(t, err)
		require.Len(t, cats, 2)
		ids := []string{cats[0].ID, cats[1].ID}
		sort.Strings(ids)
		assert.Equal(t, []string{"c-1", "c-2"}, ids)
		httpClient.AssertNumberOfCalls()
	})

	t.Run("returns request and decode errors", func(t *testing.T) {
		calls := []testutils.Call{{ResponseError: errors.New("net")}}
		dc, httpClient := newDataCatalog(t, nil, calls...)
		_, err := dc.GetCategory(context.Background(), "id")
		require.Error(t, err)
		assert.ErrorContains(t, err, "sending get request")
		httpClient.AssertNumberOfCalls()

		calls = []testutils.Call{{ResponseStatus: 200, ResponseBody: "{"}}
		dc, httpClient = newDataCatalog(t, nil, calls...)
		_, err = dc.CreateCategory(context.Background(), catalog.CategoryCreate{Name: "x"})
		require.Error(t, err)
		assert.ErrorContains(t, err, "decoding response")
		httpClient.AssertNumberOfCalls()
	})
}
