package state_test

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
)

func TestPropertyArgs_ResourceData(t *testing.T) {

	argsWithConfig := state.PropertyArgs{
		Name:        "property-name",
		Description: "property-description",
		Type:        "property-type",
		Config: map[string]interface{}{
			"enum": []string{"value1", "value2"},
		},
	}

	argsWithoutConfig := state.PropertyArgs{
		Name:        "property-name",
		Description: "property-description",
		Type:        "property-type",
		Config:      nil,
	}

	t.Run("to resource data", func(t *testing.T) {
		t.Parallel()

		resourceData := argsWithConfig.ToResourceData()
		assert.Equal(t, resources.ResourceData{
			"name":        "property-name",
			"description": "property-description",
			"type":        "property-type",
			"config": map[string]interface{}{
				"enum": []string{"value1", "value2"},
			},
		}, resourceData)

		resourceData = argsWithoutConfig.ToResourceData()
		assert.Equal(t, resources.ResourceData{
			"name":        "property-name",
			"description": "property-description",
			"type":        "property-type",
			"config":      map[string]interface{}(nil),
		}, resourceData)
	})

	t.Run("from resource data", func(t *testing.T) {
		t.Parallel()

		loopback := state.PropertyArgs{}
		loopback.FromResourceData(argsWithConfig.ToResourceData())
		assert.Equal(t, argsWithConfig, loopback)

		loopback = state.PropertyArgs{}
		loopback.FromResourceData(argsWithoutConfig.ToResourceData())
		assert.Equal(t, argsWithoutConfig, loopback)
	})

}

func TestPropertyArgs_FromCatalogPropertyType(t *testing.T) {
	t.Run("standard property", func(t *testing.T) {
		t.Parallel()

		prop := localcatalog.Property{
			LocalID:     "test-prop",
			Name:        "Test Property",
			Description: "A test property",
			Type:        "string",
			Config: map[string]interface{}{
				"minLength": 5,
				"maxLength": 10,
			},
		}

		urnFromRef := func(string) string {
			return ""
		}

		args := &state.PropertyArgs{}
		err := args.FromCatalogPropertyType(prop, urnFromRef)
		assert.NoError(t, err)
		assert.Equal(t, "Test Property", args.Name)
		assert.Equal(t, "A test property", args.Description)
		assert.Equal(t, "string", args.Type)
		assert.Equal(t, prop.Config, args.Config)
	})

	t.Run("custom type reference in type field", func(t *testing.T) {
		t.Parallel()

		prop := localcatalog.Property{
			LocalID:     "test-email",
			Name:        "Test Email",
			Description: "A test email property",
			Type:        "#/custom-types/email-types/EmailType",
			Config:      map[string]interface{}{},
		}

		urnFromRef := func(ref string) string {
			if ref == "#/custom-types/email-types/EmailType" {
				return "custom-type:EmailType"
			}
			return ""
		}

		args := &state.PropertyArgs{}
		err := args.FromCatalogPropertyType(prop, urnFromRef)
		assert.NoError(t, err)
		assert.Equal(t, "Test Email", args.Name)
		assert.Equal(t, "A test email property", args.Description)

		// Check that the type field is correctly converted to a PropertyRef
		propRef, ok := args.Type.(resources.PropertyRef)
		assert.True(t, ok, "Type should be a PropertyRef")
		assert.Equal(t, "custom-type:EmailType", propRef.URN)
		assert.Equal(t, "name", propRef.Property)
	})

	t.Run("custom type reference in itemTypes", func(t *testing.T) {
		t.Parallel()

		prop := localcatalog.Property{
			LocalID:     "test-email-list",
			Name:        "Test Email List",
			Description: "A list of emails",
			Type:        "array",
			Config: map[string]interface{}{
				"itemTypes": []interface{}{"#/custom-types/email-types/EmailType"},
			},
		}

		urnFromRef := func(ref string) string {
			if ref == "#/custom-types/email-types/EmailType" {
				return "custom-type:EmailType"
			}
			return ""
		}

		args := &state.PropertyArgs{}
		err := args.FromCatalogPropertyType(prop, urnFromRef)
		assert.NoError(t, err)
		assert.Equal(t, "Test Email List", args.Name)
		assert.Equal(t, "A list of emails", args.Description)
		assert.Equal(t, "array", args.Type)

		// Check that the itemTypes field is correctly converted to contain a PropertyRef
		itemTypes, ok := args.Config["itemTypes"].([]interface{})
		assert.True(t, ok, "itemTypes should be an array")
		assert.Len(t, itemTypes, 1)

		propRef, ok := itemTypes[0].(resources.PropertyRef)
		assert.True(t, ok, "itemTypes[0] should be a PropertyRef")
		assert.Equal(t, "custom-type:EmailType", propRef.URN)
		assert.Equal(t, "name", propRef.Property)
	})

	t.Run("itemTypes reference resolution error", func(t *testing.T) {
		t.Parallel()

		prop := localcatalog.Property{
			LocalID:     "test-email-list",
			Name:        "Test Email List",
			Description: "A list of emails",
			Type:        "array",
			Config: map[string]interface{}{
				"itemTypes": []interface{}{"#/custom-types/email-types/NonExistentType"},
			},
		}

		urnFromRef := func(ref string) string {
			return "" // No URN found
		}

		args := &state.PropertyArgs{}
		err := args.FromCatalogPropertyType(prop, urnFromRef)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unable to resolve ref to the custom type urn")
	})
}

func TestPropertyState_ResourceData(t *testing.T) {

	propertyState := state.PropertyState{
		PropertyArgs: state.PropertyArgs{
			Name:        "property-name",
			Description: "property-description",
			Type:        "property-type",
			Config:      nil,
		},
		ID:          "upstream-property-id",
		Name:        "property-name",
		Description: "property-description",
		Type:        "property-type",
		WorkspaceID: "workspace-id",
		CreatedAt:   "2021-09-01T00:00:00Z",
		UpdatedAt:   "2021-09-01T00:00:00Z",
		Config:      nil,
	}

	t.Run("to resource data", func(t *testing.T) {
		t.Parallel()

		resourceData := propertyState.ToResourceData()
		assert.Equal(t, resources.ResourceData{
			"id":          "upstream-property-id",
			"name":        "property-name",
			"description": "property-description",
			"type":        "property-type",
			"config":      map[string]interface{}(nil),
			"workspaceId": "workspace-id",
			"createdAt":   "2021-09-01T00:00:00Z",
			"updatedAt":   "2021-09-01T00:00:00Z",
			"propertyArgs": map[string]interface{}{
				"name":        "property-name",
				"description": "property-description",
				"type":        "property-type",
				"config":      map[string]interface{}(nil),
			},
		}, resourceData)
	})

	t.Run("from resource data", func(t *testing.T) {
		t.Parallel()

		loopback := state.PropertyState{}
		loopback.FromResourceData(propertyState.ToResourceData())
		assert.Equal(t, propertyState, loopback)
	})

}

func TestPropertyArgs_DiffUpstream(t *testing.T) {

	t.Run("no diff", func(t *testing.T) {
		t.Parallel()

		args := &state.PropertyArgs{
			Name:        "test-property",
			Description: "A test property",
			Type:        "string",
			Config: map[string]interface{}{
				"minLength": 5,
				"maxLength": 10,
			},
		}

		upstream := &catalog.Property{
			Name:        "test-property",
			Description: "A test property",
			Type:        "string",
			Config: map[string]interface{}{
				"minLength": 5,
				"maxLength": 10,
			},
		}

		diffed := args.DiffUpstream(upstream)
		assert.False(t, diffed)
	})

	t.Run("diff - name changed", func(t *testing.T) {
		t.Parallel()

		args := &state.PropertyArgs{
			Name:        "test-property",
			Description: "A test property",
			Type:        "string",
			Config:      map[string]interface{}{},
		}

		upstream := &catalog.Property{
			Name:        "test-property-updated",
			Description: "A test property",
			Type:        "string",
			Config:      map[string]interface{}{},
		}

		diffed := args.DiffUpstream(upstream)
		assert.True(t, diffed)
	})

	t.Run("diff - description changed", func(t *testing.T) {
		t.Parallel()

		args := &state.PropertyArgs{
			Name:        "test-property",
			Description: "A test property",
			Type:        "string",
			Config:      map[string]interface{}{},
		}

		upstream := &catalog.Property{
			Name:        "test-property",
			Description: "Updated description",
			Type:        "string",
			Config:      map[string]interface{}{},
		}

		diffed := args.DiffUpstream(upstream)
		assert.True(t, diffed)
	})

	t.Run("diff - type changed", func(t *testing.T) {
		t.Parallel()

		args := &state.PropertyArgs{
			Name:        "test-property",
			Description: "A test property",
			Type:        "string",
			Config:      map[string]interface{}{},
		}

		upstream := &catalog.Property{
			Name:        "test-property",
			Description: "A test property",
			Type:        "number",
			Config:      map[string]interface{}{},
		}

		diffed := args.DiffUpstream(upstream)
		assert.True(t, diffed)
	})

	t.Run("diff - config changed", func(t *testing.T) {
		t.Parallel()

		args := &state.PropertyArgs{
			Name:        "test-property",
			Description: "A test property",
			Type:        "string",
			Config: map[string]interface{}{
				"minLength": 5,
				"maxLength": 10,
			},
		}

		upstream := &catalog.Property{
			Name:        "test-property",
			Description: "A test property",
			Type:        "string",
			Config: map[string]interface{}{
				"minLength": 5,
				"maxLength": 20,
			},
		}

		diffed := args.DiffUpstream(upstream)
		assert.True(t, diffed)
	})
}
