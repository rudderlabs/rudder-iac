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

func TestPropertyCRUDAndExternalID(t *testing.T) {
	t.Run("create update get set external id and delete", func(t *testing.T) {
		calls := []testutils.Call{
			{
				Validate: func(req *http.Request) bool {
					return validateRequest(t, req, "POST", "v2/catalog/properties", `{"name":"prop","description":"desc","type":"string","propConfig":{"maxLength":10},"externalId":"ext-1"}`)
				},
				ResponseStatus: 200,
				ResponseBody:   `{"id":"p-1","name":"prop","type":"string","workspaceId":"ws-1"}`,
			},
			{
				Validate: func(req *http.Request) bool {
					return validateRequest(t, req, "PUT", "v2/catalog/properties/p-1", `{"name":"prop2","description":"desc2","type":"number","propConfig":{"minimum":1}}`)
				},
				ResponseStatus: 200,
				ResponseBody:   `{"id":"p-1","name":"prop2","type":"number","workspaceId":"ws-1"}`,
			},
			{
				Validate: func(req *http.Request) bool {
					return validateRequest(t, req, "GET", "v2/catalog/properties/p-1", "")
				},
				ResponseStatus: 200,
				ResponseBody:   `{"id":"p-1","name":"prop2","type":"number","workspaceId":"ws-1"}`,
			},
			{
				Validate: func(req *http.Request) bool {
					return validateRequest(t, req, "PUT", "v2/catalog/properties/p-1/external-id", `{"externalId":"ext-2"}`)
				},
				ResponseStatus: 200,
				ResponseBody:   `{}`,
			},
			{
				Validate: func(req *http.Request) bool {
					return validateRequest(t, req, "DELETE", "v2/catalog/properties/p-1", "")
				},
				ResponseStatus: 200,
				ResponseBody:   `{}`,
			},
		}

		dc, httpClient := newDataCatalog(t, nil, calls...)
		prop, err := dc.CreateProperty(context.Background(), catalog.PropertyCreate{
			Name:        "prop",
			Description: "desc",
			Type:        "string",
			Config:      map[string]any{"maxLength": float64(10)},
			ExternalId:  "ext-1",
		})
		require.NoError(t, err)
		assert.Equal(t, &catalog.Property{ID: "p-1", Name: "prop", Type: "string", WorkspaceId: "ws-1"}, prop)

		prop, err = dc.UpdateProperty(context.Background(), "p-1", &catalog.PropertyUpdate{
			Name:        "prop2",
			Description: "desc2",
			Type:        "number",
			Config:      map[string]any{"minimum": float64(1)},
		})
		require.NoError(t, err)
		assert.Equal(t, "prop2", prop.Name)

		prop, err = dc.GetProperty(context.Background(), "p-1")
		require.NoError(t, err)
		assert.Equal(t, "p-1", prop.ID)

		err = dc.SetPropertyExternalId(context.Background(), "p-1", "ext-2")
		require.NoError(t, err)

		err = dc.DeleteProperty(context.Background(), "p-1")
		require.NoError(t, err)

		httpClient.AssertNumberOfCalls()
	})

	t.Run("request and decode failures", func(t *testing.T) {
		calls := []testutils.Call{{ResponseError: errors.New("network")}}
		dc, httpClient := newDataCatalog(t, nil, calls...)
		_, err := dc.CreateProperty(context.Background(), catalog.PropertyCreate{Name: "prop"})
		require.Error(t, err)
		assert.ErrorContains(t, err, "executing http request")
		httpClient.AssertNumberOfCalls()

		calls = []testutils.Call{{ResponseStatus: 200, ResponseBody: "{"}}
		dc, httpClient = newDataCatalog(t, nil, calls...)
		_, err = dc.GetProperty(context.Background(), "p-1")
		require.Error(t, err)
		assert.ErrorContains(t, err, "decoding response")
		httpClient.AssertNumberOfCalls()
	})
}

func TestGetPropertiesPagination(t *testing.T) {
	hasExternalID := true
	calls := []testutils.Call{
		{ResponseStatus: 200, ResponseBody: `{"data":[{"id":"first"}],"total":3,"currentPage":1,"pageSize":2}`},
		{ResponseStatus: 200, ResponseBody: `{"data":[{"id":"p-1"},{"id":"p-2"}],"total":3,"currentPage":1,"pageSize":2}`},
		{ResponseStatus: 200, ResponseBody: `{"data":[{"id":"p-3"}],"total":3,"currentPage":2,"pageSize":2}`},
	}

	dc, httpClient := newDataCatalog(t, nil, calls...)
	props, err := dc.GetProperties(context.Background(), catalog.ListOptions{HasExternalID: &hasExternalID})
	require.NoError(t, err)
	require.Len(t, props, 3)
	ids := []string{props[0].ID, props[1].ID, props[2].ID}
	sort.Strings(ids)
	assert.Equal(t, []string{"p-1", "p-2", "p-3"}, ids)
	httpClient.AssertNumberOfCalls()
}
