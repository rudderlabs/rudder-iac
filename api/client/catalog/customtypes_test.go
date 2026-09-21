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

func TestCreateCustomType(t *testing.T) {
	dc, httpClient := newDataCatalog(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return assertCall(t, req, "POST", catalogURL("v2/catalog/custom-types"), `{"name":"ct","description":"desc","type":"object","config":{"x":1},"externalId":"ext-1"}`)
		},
		ResponseStatus: 200,
		ResponseBody:   `{"id":"ct-1","name":"ct","type":"object","workspaceId":"ws-1"}`,
	})

	customType, err := dc.CreateCustomType(context.Background(), catalog.CustomTypeCreate{
		Name:        "ct",
		Description: "desc",
		Type:        "object",
		Config:      map[string]any{"x": float64(1)},
		ExternalId:  "ext-1",
	})
	require.NoError(t, err)
	assert.Equal(t, &catalog.CustomType{ID: "ct-1", Name: "ct", Type: "object", WorkspaceId: "ws-1"}, customType)
	httpClient.AssertNumberOfCalls()
}

func TestUpdateCustomType(t *testing.T) {
	dc, httpClient := newDataCatalog(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return assertCall(t, req, "PUT", catalogURL("v2/catalog/custom-types/ct-1"), `{"name":"ct2","description":"desc2","type":"array","config":{"y":2}}`)
		},
		ResponseStatus: 200,
		ResponseBody:   `{"id":"ct-1","name":"ct2","type":"array","workspaceId":"ws-1"}`,
	})

	customType, err := dc.UpdateCustomType(context.Background(), "ct-1", &catalog.CustomTypeUpdate{
		Name:        "ct2",
		Description: "desc2",
		Type:        "array",
		Config:      map[string]any{"y": float64(2)},
	})
	require.NoError(t, err)
	assert.Equal(t, &catalog.CustomType{ID: "ct-1", Name: "ct2", Type: "array", WorkspaceId: "ws-1"}, customType)
	httpClient.AssertNumberOfCalls()
}

func TestGetCustomType(t *testing.T) {
	dc, httpClient := newDataCatalog(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return assertCall(t, req, "GET", catalogURL("v2/catalog/custom-types/ct-1"), "")
		},
		ResponseStatus: 200,
		ResponseBody:   `{"id":"ct-1","name":"ct2","type":"array","workspaceId":"ws-1"}`,
	})

	customType, err := dc.GetCustomType(context.Background(), "ct-1")
	require.NoError(t, err)
	assert.Equal(t, &catalog.CustomType{ID: "ct-1", Name: "ct2", Type: "array", WorkspaceId: "ws-1"}, customType)
	httpClient.AssertNumberOfCalls()
}

func TestSetCustomTypeExternalID(t *testing.T) {
	dc, httpClient := newDataCatalog(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return assertCall(t, req, "PUT", catalogURL("v2/catalog/custom-types/ct-1/external-id"), `{"externalId":"ext-2"}`)
		},
		ResponseStatus: 200,
		ResponseBody:   `{}`,
	})

	err := dc.SetCustomTypeExternalId(context.Background(), "ct-1", "ext-2")
	require.NoError(t, err)
	httpClient.AssertNumberOfCalls()
}

func TestDeleteCustomType(t *testing.T) {
	dc, httpClient := newDataCatalog(t, testutils.Call{
		Validate: func(req *http.Request) bool {
			return assertCall(t, req, "DELETE", catalogURL("v2/catalog/custom-types/ct-1"), "")
		},
		ResponseStatus: 200,
		ResponseBody:   `{}`,
	})

	err := dc.DeleteCustomType(context.Background(), "ct-1")
	require.NoError(t, err)
	httpClient.AssertNumberOfCalls()
}

func TestGetCustomTypes(t *testing.T) {
	dc, httpClient := newDataCatalog(t,
		testutils.Call{ResponseStatus: 200, ResponseBody: `{"data":[{"id":"ct-1"},{"id":"ct-2"}],"total":3,"currentPage":1,"pageSize":2}`}, // This request will be thrown away as it's used only to calculate concurrency
		testutils.Call{ResponseStatus: 200, ResponseBody: `{"data":[{"id":"ct-1"},{"id":"ct-2"}],"total":3,"currentPage":1,"pageSize":2}`},
		testutils.Call{ResponseStatus: 200, ResponseBody: `{"data":[{"id":"ct-3"}],"total":3,"currentPage":2,"pageSize":2}`},
	)

	customTypes, err := dc.GetCustomTypes(context.Background(), catalog.ListOptions{})
	require.NoError(t, err)
	require.Len(t, customTypes, 3)

	ids := []string{customTypes[0].ID, customTypes[1].ID, customTypes[2].ID}
	sort.Strings(ids)
	assert.Equal(t, []string{"ct-1", "ct-2", "ct-3"}, ids)
	httpClient.AssertNumberOfCalls()
}

func TestCustomTypeErrors(t *testing.T) {
	tests := []struct {
		name          string
		calls         []testutils.Call
		operation     func(dc catalog.DataCatalog) error
		expectedError string
	}{
		{
			name:  "GetCustomType request error",
			calls: []testutils.Call{{ResponseError: errors.New("net")}},
			operation: func(dc catalog.DataCatalog) error {
				_, err := dc.GetCustomType(context.Background(), "id")
				return err
			},
			expectedError: "sending get request",
		},
		{
			name:  "CreateCustomType decode error",
			calls: []testutils.Call{{ResponseStatus: 200, ResponseBody: "{"}},
			operation: func(dc catalog.DataCatalog) error {
				_, err := dc.CreateCustomType(context.Background(), catalog.CustomTypeCreate{Name: "ct"})
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
