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

func TestCreateCategory(t *testing.T) {
	dc, httpClient := newDataCatalog(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return assertCall(t, req, "POST", catalogURL("v2/catalog/categories"), `{"name":"category","externalId":"ext-1"}`)
		},
		ResponseStatus: 200,
		ResponseBody:   `{"id":"c-1","name":"category","workspaceId":"ws-1"}`,
	})

	cat, err := dc.CreateCategory(context.Background(), catalog.CategoryCreate{Name: "category", ExternalId: "ext-1"})
	require.NoError(t, err)
	assert.Equal(t, &catalog.Category{ID: "c-1", Name: "category", WorkspaceID: "ws-1"}, cat)
	httpClient.AssertNumberOfCalls()
}

func TestUpdateCategory(t *testing.T) {
	dc, httpClient := newDataCatalog(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return assertCall(t, req, "PUT", catalogURL("v2/catalog/categories/c-1"), `{"name":"category-2"}`)
		},
		ResponseStatus: 200,
		ResponseBody:   `{"id":"c-1","name":"category-2","workspaceId":"ws-1"}`,
	})

	cat, err := dc.UpdateCategory(context.Background(), "c-1", catalog.CategoryUpdate{Name: "category-2"})
	require.NoError(t, err)
	assert.Equal(t, &catalog.Category{ID: "c-1", Name: "category-2", WorkspaceID: "ws-1"}, cat)
	httpClient.AssertNumberOfCalls()
}

func TestGetCategory(t *testing.T) {
	dc, httpClient := newDataCatalog(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return assertCall(t, req, "GET", catalogURL("v2/catalog/categories/c-1"), "")
		},
		ResponseStatus: 200,
		ResponseBody:   `{"id":"c-1","name":"category","workspaceId":"ws-1"}`,
	})

	cat, err := dc.GetCategory(context.Background(), "c-1")
	require.NoError(t, err)
	assert.Equal(t, &catalog.Category{ID: "c-1", Name: "category", WorkspaceID: "ws-1"}, cat)
	httpClient.AssertNumberOfCalls()
}

func TestSetCategoryExternalID(t *testing.T) {
	dc, httpClient := newDataCatalog(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return assertCall(t, req, "PUT", catalogURL("v2/catalog/categories/c-1/external-id"), `{"externalId":"ext-2"}`)
		},
		ResponseStatus: 200,
		ResponseBody:   `{}`,
	})

	err := dc.SetCategoryExternalId(context.Background(), "c-1", "ext-2")
	require.NoError(t, err)
	httpClient.AssertNumberOfCalls()
}

func TestDeleteCategory(t *testing.T) {
	dc, httpClient := newDataCatalog(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return assertCall(t, req, "DELETE", catalogURL("v2/catalog/categories/c-1"), "")
		},
		ResponseStatus: 200,
		ResponseBody:   `{}`,
	})

	err := dc.DeleteCategory(context.Background(), "c-1")
	require.NoError(t, err)
	httpClient.AssertNumberOfCalls()
}

func TestGetCategories(t *testing.T) {
	dc, httpClient := newDataCatalog(t,
		testutils.Call{ResponseStatus: 200, ResponseBody: `{"data":[{"id":"c-1","workspaceId":"ws-1"}],"total":2,"currentPage":1,"pageSize":1}`}, // This request will be thrown away as it's used only to calculate concurrency
		testutils.Call{ResponseStatus: 200, ResponseBody: `{"data":[{"id":"c-1","workspaceId":"ws-1"}],"total":2,"currentPage":1,"pageSize":1}`},
		testutils.Call{ResponseStatus: 200, ResponseBody: `{"data":[{"id":"c-2","workspaceId":"ws-2"},{"id":"c-x"}],"total":2,"currentPage":2,"pageSize":1}`},
	)

	cats, err := dc.GetCategories(context.Background(), catalog.ListOptions{})
	require.NoError(t, err)
	require.Len(t, cats, 2)

	ids := []string{cats[0].ID, cats[1].ID}
	sort.Strings(ids)
	assert.Equal(t, []string{"c-1", "c-2"}, ids)
	httpClient.AssertNumberOfCalls()
}

func TestCategoryErrors(t *testing.T) {
	tests := []struct {
		name          string
		calls         []testutils.Call
		operation     func(dc catalog.DataCatalog) error
		expectedError string
	}{
		{
			name:  "GetCategory request error",
			calls: []testutils.Call{{ResponseError: errors.New("net")}},
			operation: func(dc catalog.DataCatalog) error {
				_, err := dc.GetCategory(context.Background(), "id")
				return err
			},
			expectedError: "sending get request",
		},
		{
			name:  "CreateCategory decode error",
			calls: []testutils.Call{{ResponseStatus: 200, ResponseBody: "{"}},
			operation: func(dc catalog.DataCatalog) error {
				_, err := dc.CreateCategory(context.Background(), catalog.CategoryCreate{Name: "x"})
				return err
			},
			expectedError: "decoding response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dc, httpClient := newDataCatalog(t, tt.calls...)
			err := tt.operation(dc)
			require.Error(t, err)
			assert.ErrorContains(t, err, tt.expectedError)
			httpClient.AssertNumberOfCalls()
		})
	}
}
