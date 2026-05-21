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

func TestCustomTypeCRUDAndExternalID(t *testing.T) {
	calls := []testutils.Call{
		{
			Validate: func(req *http.Request) bool {
				return validateRequest(t, req, "POST", "v2/catalog/custom-types", `{"name":"ct","description":"desc","type":"object","config":{"x":1},"externalId":"ext-1"}`)
			},
			ResponseStatus: 200,
			ResponseBody:   `{"id":"ct-1","name":"ct","type":"object","workspaceId":"ws-1"}`,
		},
		{
			Validate: func(req *http.Request) bool {
				return validateRequest(t, req, "PUT", "v2/catalog/custom-types/ct-1", `{"name":"ct2","description":"desc2","type":"array","config":{"y":2}}`)
			},
			ResponseStatus: 200,
			ResponseBody:   `{"id":"ct-1","name":"ct2","type":"array","workspaceId":"ws-1"}`,
		},
		{
			Validate: func(req *http.Request) bool {
				return validateRequest(t, req, "GET", "v2/catalog/custom-types/ct-1", "")
			},
			ResponseStatus: 200,
			ResponseBody:   `{"id":"ct-1","name":"ct2","type":"array","workspaceId":"ws-1"}`,
		},
		{
			Validate: func(req *http.Request) bool {
				return validateRequest(t, req, "PUT", "v2/catalog/custom-types/ct-1/external-id", `{"externalId":"ext-2"}`)
			},
			ResponseStatus: 200,
			ResponseBody:   `{}`,
		},
		{
			Validate: func(req *http.Request) bool {
				return validateRequest(t, req, "DELETE", "v2/catalog/custom-types/ct-1", "")
			},
			ResponseStatus: 200,
			ResponseBody:   `{}`,
		},
	}

	dc, httpClient := newDataCatalog(t, nil, calls...)
	customType, err := dc.CreateCustomType(context.Background(), catalog.CustomTypeCreate{
		Name:        "ct",
		Description: "desc",
		Type:        "object",
		Config:      map[string]any{"x": float64(1)},
		ExternalId:  "ext-1",
	})
	require.NoError(t, err)
	assert.Equal(t, &catalog.CustomType{ID: "ct-1", Name: "ct", Type: "object", WorkspaceId: "ws-1"}, customType)

	customType, err = dc.UpdateCustomType(context.Background(), "ct-1", &catalog.CustomTypeUpdate{
		Name:        "ct2",
		Description: "desc2",
		Type:        "array",
		Config:      map[string]any{"y": float64(2)},
	})
	require.NoError(t, err)
	assert.Equal(t, "ct2", customType.Name)

	customType, err = dc.GetCustomType(context.Background(), "ct-1")
	require.NoError(t, err)
	assert.Equal(t, "ct-1", customType.ID)

	err = dc.SetCustomTypeExternalId(context.Background(), "ct-1", "ext-2")
	require.NoError(t, err)

	err = dc.DeleteCustomType(context.Background(), "ct-1")
	require.NoError(t, err)

	httpClient.AssertNumberOfCalls()
}

func TestGetCustomTypesPaginationAndErrors(t *testing.T) {
	t.Run("aggregates all pages", func(t *testing.T) {
		calls := []testutils.Call{
			{ResponseStatus: 200, ResponseBody: `{"data":[{"id":"ignore"}],"total":3,"currentPage":1,"pageSize":2}`},
			{ResponseStatus: 200, ResponseBody: `{"data":[{"id":"ct-1"},{"id":"ct-2"}],"total":3,"currentPage":1,"pageSize":2}`},
			{ResponseStatus: 200, ResponseBody: `{"data":[{"id":"ct-3"}],"total":3,"currentPage":2,"pageSize":2}`},
		}

		dc, httpClient := newDataCatalog(t, nil, calls...)
		customTypes, err := dc.GetCustomTypes(context.Background(), catalog.ListOptions{})
		require.NoError(t, err)
		require.Len(t, customTypes, 3)
		ids := []string{customTypes[0].ID, customTypes[1].ID, customTypes[2].ID}
		sort.Strings(ids)
		assert.Equal(t, []string{"ct-1", "ct-2", "ct-3"}, ids)
		httpClient.AssertNumberOfCalls()
	})

	t.Run("request and decode errors", func(t *testing.T) {
		calls := []testutils.Call{{ResponseError: errors.New("net")}}
		dc, httpClient := newDataCatalog(t, nil, calls...)
		_, err := dc.GetCustomType(context.Background(), "id")
		require.Error(t, err)
		assert.ErrorContains(t, err, "sending get request")
		httpClient.AssertNumberOfCalls()

		calls = []testutils.Call{{ResponseStatus: 200, ResponseBody: "{"}}
		dc, httpClient = newDataCatalog(t, nil, calls...)
		_, err = dc.CreateCustomType(context.Background(), catalog.CustomTypeCreate{Name: "ct"})
		require.Error(t, err)
		assert.ErrorContains(t, err, "decoding response")
		httpClient.AssertNumberOfCalls()
	})
}
