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

func TestCreateProperty(t *testing.T) {
	dc, httpClient := newDataCatalog(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return assertCall(t, req, "POST", catalogURL("v2/catalog/properties"), `{"name":"prop","description":"desc","type":"string","propConfig":{"maxLength":10},"externalId":"ext-1"}`)
		},
		ResponseStatus: 200,
		ResponseBody:   `{"id":"p-1","name":"prop","type":"string","workspaceId":"ws-1"}`,
	})

	prop, err := dc.CreateProperty(context.Background(), catalog.PropertyCreate{
		Name:        "prop",
		Description: "desc",
		Type:        "string",
		Config:      map[string]any{"maxLength": float64(10)},
		ExternalId:  "ext-1",
	})
	require.NoError(t, err)
	assert.Equal(t, &catalog.Property{ID: "p-1", Name: "prop", Type: "string", WorkspaceId: "ws-1"}, prop)
	httpClient.AssertNumberOfCalls()
}

func TestUpdateProperty(t *testing.T) {
	dc, httpClient := newDataCatalog(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return assertCall(t, req, "PUT", catalogURL("v2/catalog/properties/p-1"), `{"name":"prop2","description":"desc2","type":"number","propConfig":{"minimum":1}}`)
		},
		ResponseStatus: 200,
		ResponseBody:   `{"id":"p-1","name":"prop2","type":"number","workspaceId":"ws-1"}`,
	})

	prop, err := dc.UpdateProperty(context.Background(), "p-1", &catalog.PropertyUpdate{
		Name:        "prop2",
		Description: "desc2",
		Type:        "number",
		Config:      map[string]any{"minimum": float64(1)},
	})
	require.NoError(t, err)
	assert.Equal(t, &catalog.Property{ID: "p-1", Name: "prop2", Type: "number", WorkspaceId: "ws-1"}, prop)
	httpClient.AssertNumberOfCalls()
}

func TestGetProperty(t *testing.T) {
	dc, httpClient := newDataCatalog(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return assertCall(t, req, "GET", catalogURL("v2/catalog/properties/p-1"), "")
		},
		ResponseStatus: 200,
		ResponseBody:   `{"id":"p-1","name":"prop2","type":"number","workspaceId":"ws-1"}`,
	})

	prop, err := dc.GetProperty(context.Background(), "p-1")
	require.NoError(t, err)
	assert.Equal(t, &catalog.Property{ID: "p-1", Name: "prop2", Type: "number", WorkspaceId: "ws-1"}, prop)
	httpClient.AssertNumberOfCalls()
}

func TestSetPropertyExternalID(t *testing.T) {
	dc, httpClient := newDataCatalog(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return assertCall(t, req, "PUT", catalogURL("v2/catalog/properties/p-1/external-id"), `{"externalId":"ext-2"}`)
		},
		ResponseStatus: 200,
		ResponseBody:   `{}`,
	})

	err := dc.SetPropertyExternalId(context.Background(), "p-1", "ext-2")
	require.NoError(t, err)
	httpClient.AssertNumberOfCalls()
}

func TestDeleteProperty(t *testing.T) {
	dc, httpClient := newDataCatalog(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return assertCall(t, req, "DELETE", catalogURL("v2/catalog/properties/p-1"), "")
		},
		ResponseStatus: 200,
		ResponseBody:   `{}`,
	})

	err := dc.DeleteProperty(context.Background(), "p-1")
	require.NoError(t, err)
	httpClient.AssertNumberOfCalls()
}

func TestGetProperties(t *testing.T) {
	dc, httpClient := newDataCatalog(t,
		testutils.Call{ResponseStatus: 200, ResponseBody: `{"data":[{"id":"p-1"},{"id":"p-2"}],"total":3,"currentPage":1,"pageSize":2}`}, // This request will be thrown away as it's used only to calculate concurrency
		testutils.Call{ResponseStatus: 200, ResponseBody: `{"data":[{"id":"p-1"},{"id":"p-2"}],"total":3,"currentPage":1,"pageSize":2}`},
		testutils.Call{ResponseStatus: 200, ResponseBody: `{"data":[{"id":"p-3"}],"total":3,"currentPage":2,"pageSize":2}`},
	)

	props, err := dc.GetProperties(context.Background(), catalog.ListOptions{HasExternalID: boolPtr(true)})
	require.NoError(t, err)
	require.Len(t, props, 3)

	ids := []string{props[0].ID, props[1].ID, props[2].ID}
	sort.Strings(ids)
	assert.Equal(t, []string{"p-1", "p-2", "p-3"}, ids)
	httpClient.AssertNumberOfCalls()
}

func TestPropertyErrors(t *testing.T) {
	tests := []struct {
		name          string
		calls         []testutils.Call
		operation     func(dc catalog.DataCatalog) error
		expectedError string
	}{
		{
			name:  "CreateProperty request error",
			calls: []testutils.Call{{ResponseError: errors.New("network")}},
			operation: func(dc catalog.DataCatalog) error {
				_, err := dc.CreateProperty(context.Background(), catalog.PropertyCreate{Name: "prop"})
				return err
			},
			expectedError: "executing http request",
		},
		{
			name:  "GetProperty decode error",
			calls: []testutils.Call{{ResponseStatus: 200, ResponseBody: "{"}},
			operation: func(dc catalog.DataCatalog) error {
				_, err := dc.GetProperty(context.Background(), "p-1")
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
