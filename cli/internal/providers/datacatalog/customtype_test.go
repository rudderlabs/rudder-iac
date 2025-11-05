package datacatalog_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
)

type MockCustomTypeCatalog struct {
	datacatalog.EmptyCatalog
	mockCustomType      *catalog.CustomType
	err                 error
	updateCalled        bool
	setExternalIdCalled bool
}

func (m *MockCustomTypeCatalog) CreateCustomType(ctx context.Context, ctCreate catalog.CustomTypeCreate) (*catalog.CustomType, error) {
	return m.mockCustomType, m.err
}

func (m *MockCustomTypeCatalog) UpdateCustomType(ctx context.Context, id string, ctUpdate *catalog.CustomTypeUpdate) (*catalog.CustomType, error) {
	m.updateCalled = true
	if m.mockCustomType == nil {
		return nil, m.err
	}
	m.mockCustomType.Name = ctUpdate.Name
	m.mockCustomType.Description = ctUpdate.Description
	m.mockCustomType.Type = ctUpdate.Type
	m.mockCustomType.Config = ctUpdate.Config
	m.mockCustomType.Properties = ctUpdate.Properties

	return m.mockCustomType, m.err
}

func (m *MockCustomTypeCatalog) DeleteCustomType(ctx context.Context, id string) error {
	return m.err
}

func (m *MockCustomTypeCatalog) GetCustomType(ctx context.Context, id string) (*catalog.CustomType, error) {
	return m.mockCustomType, m.err
}

func (m *MockCustomTypeCatalog) SetCustomTypeExternalId(ctx context.Context, id string, externalId string) error {
	m.setExternalIdCalled = true
	if m.mockCustomType != nil {
		m.mockCustomType.ExternalId = externalId
	}
	return m.err
}

func (m *MockCustomTypeCatalog) ResetSpies() {
	m.updateCalled = false
	m.setExternalIdCalled = false
}

func (m *MockCustomTypeCatalog) SetCustomType(ct *catalog.CustomType) {
	m.mockCustomType = ct
}

func (m *MockCustomTypeCatalog) SetError(err error) {
	m.err = err
}

func TestCustomTypeProviderOperations(t *testing.T) {
	var (
		ctx                = context.Background()
		mockCatalog        = &MockCustomTypeCatalog{}
		customTypeProvider = datacatalog.NewCustomTypeProvider(mockCatalog, "test-import-dir")
		created, _         = time.Parse(time.RFC3339, "2021-09-01T00:00:00Z")
		updated, _         = time.Parse(time.RFC3339, "2021-09-02T00:00:00Z")
	)

	t.Run("Import", func(t *testing.T) {
		tests := []struct {
			name             string
			localArgs        state.CustomTypeArgs
			remoteCustomType *catalog.CustomType
			mockErr          error
			expectErr        bool
			expectUpdate     bool
			expectSetExtId   bool
			expectResource   *resources.ResourceData
		}{
			{
				name: "successful import no differences",
				localArgs: state.CustomTypeArgs{
					LocalID:     "local-id",
					Name:        "TestType",
					Description: "desc",
					Type:        "object",
					Config:      map[string]any{"key": "val"},
					Properties:  []*state.CustomTypeProperty{{ID: "prop1", Required: true, RefToID: "prop1"}},
				},
				remoteCustomType: &catalog.CustomType{
					ID:          "remote-id",
					Name:        "TestType",
					Description: "desc",
					Type:        "object",
					Config:      map[string]any{"key": "val"},
					Properties:  []catalog.CustomTypeProperty{{ID: "prop1", Required: true}},
					WorkspaceId: "ws-id",
					CreatedAt:   created,
					UpdatedAt:   updated,
				},
				mockErr:        nil,
				expectErr:      false,
				expectUpdate:   false,
				expectSetExtId: true,
				expectResource: &resources.ResourceData{
					"id":              "remote-id",
					"localId":         "local-id",
					"name":            "TestType",
					"description":     "desc",
					"type":            "object",
					"config":          map[string]any{"key": "val"},
					"version":         0,
					"itemDefinitions": []any(nil),
					"rules":           map[string]any(nil),
					"workspaceId":     "ws-id",
					"createdAt":       created.String(),
					"updatedAt":       updated.String(),
					"customTypeArgs": map[string]any{
						"localId":     "local-id",
						"name":        "TestType",
						"description": "desc",
						"type":        "object",
						"config":      map[string]any{"key": "val"},
						"properties":  []map[string]any{{"id": "prop1", "required": true, "refToId": "prop1"}},
						"variants":    []map[string]any{},
					},
				},
			},
			{
				name: "successful import with differences",
				localArgs: state.CustomTypeArgs{
					LocalID:     "local-id",
					Name:        "NewType",
					Description: "new desc",
					Type:        "object",
					Config:      map[string]any{"newkey": "newval"},
					Properties:  []*state.CustomTypeProperty{{ID: "prop1", Required: true, RefToID: "prop1"}, {ID: "prop2", Required: false, RefToID: "prop2"}},
				},
				remoteCustomType: &catalog.CustomType{
					ID:          "remote-id",
					Name:        "TestType",
					Description: "desc",
					Type:        "object",
					Config:      map[string]any{"key": "val"},
					Properties:  []catalog.CustomTypeProperty{{ID: "prop1", Required: true}},
					WorkspaceId: "ws-id",
					CreatedAt:   created,
					UpdatedAt:   updated,
				},
				mockErr:        nil,
				expectErr:      false,
				expectUpdate:   true,
				expectSetExtId: true,
				expectResource: &resources.ResourceData{
					"id":              "remote-id",
					"localId":         "local-id",
					"name":            "NewType",
					"description":     "new desc",
					"type":            "object",
					"config":          map[string]any{"newkey": "newval"},
					"version":         0,
					"itemDefinitions": []any(nil),
					"rules":           map[string]any(nil),
					"workspaceId":     "ws-id",
					"createdAt":       created.String(),
					"updatedAt":       updated.String(),
					"customTypeArgs": map[string]any{
						"localId":     "local-id",
						"name":        "NewType",
						"description": "new desc",
						"type":        "object",
						"config":      map[string]any{"newkey": "newval"},
						"properties":  []map[string]any{{"id": "prop1", "required": true, "refToId": "prop1"}, {"id": "prop2", "required": false, "refToId": "prop2"}},
						"variants":    []map[string]any{},
					},
				},
			},
			{
				name:             "error on get custom type",
				localArgs:        state.CustomTypeArgs{LocalID: "local-id", Name: "TestType", Type: "object", Description: ""},
				remoteCustomType: nil,
				mockErr:          errors.New("error getting custom type"),
				expectErr:        true,
			},
			{
				name:             "error on update",
				localArgs:        state.CustomTypeArgs{LocalID: "local-id", Name: "NewType", Type: "object", Description: ""},
				remoteCustomType: &catalog.CustomType{ID: "remote-id", Name: "TestType", Type: "object"},
				mockErr:          errors.New("error updating custom type"),
				expectErr:        true,
				expectUpdate:     true,
			},
			{
				name:             "error on set external ID",
				localArgs:        state.CustomTypeArgs{LocalID: "local-id", Name: "TestType", Type: "object", Description: ""},
				remoteCustomType: &catalog.CustomType{ID: "remote-id", Name: "TestType", Type: "object"},
				mockErr:          errors.New("error setting external ID"),
				expectErr:        true,
				expectSetExtId:   true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				mockCatalog.ResetSpies()
				mockCatalog.SetCustomType(tt.remoteCustomType)
				mockCatalog.SetError(tt.mockErr)

				res, err := customTypeProvider.Import(ctx, "local-id", tt.localArgs.ToResourceData(), "remote-id")

				if tt.expectErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				var (
					expected = *tt.expectResource
					actual   = *res
				)
				assert.Equal(t, expected, actual)
				assert.Equal(t, tt.expectUpdate, mockCatalog.updateCalled)
				assert.Equal(t, tt.expectSetExtId, mockCatalog.setExternalIdCalled)
			})
		}
	})
}
