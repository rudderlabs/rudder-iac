package state

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/stretchr/testify/assert"
)

func TestCustomTypeArgsToResourceData(t *testing.T) {
	// Create a CustomTypeArgs instance
	args := CustomTypeArgs{
		LocalID:     "EmailType",
		Name:        "Email Type",
		Description: "Custom type for email validation",
		Type:        "string",
		Config: map[string]any{
			"format":    "email",
			"minLength": 5,
			"maxLength": 255,
		},
		Properties: []*CustomTypeProperty{
			{
				ID:       "property1",
				Required: true,
			},
			{
				ID:       "property2",
				Required: false,
			},
		},
	}

	// Convert to resource data
	data := args.ToResourceData()

	// Verify the conversion
	assert.Equal(t, "EmailType", data["localId"])
	assert.Equal(t, "Email Type", data["name"])
	assert.Equal(t, "Custom type for email validation", data["description"])
	assert.Equal(t, "string", data["type"])

	config := data["config"].(map[string]any)
	assert.Equal(t, "email", config["format"])
	assert.Equal(t, 5, config["minLength"])
	assert.Equal(t, 255, config["maxLength"])

	properties := data["properties"].([]map[string]any)
	assert.Len(t, properties, 2)
	assert.Equal(t, "property1", properties[0]["id"])
	assert.Equal(t, true, properties[0]["required"])
	assert.Equal(t, "property2", properties[1]["id"])
	assert.Equal(t, false, properties[1]["required"])
}

func TestCustomTypeArgsFromResourceData(t *testing.T) {
	// Create resource data
	data := resources.ResourceData{
		"localId":     "EmailType",
		"name":        "Email Type",
		"description": "Custom type for email validation",
		"type":        "string",
		"config": map[string]any{
			"format":    "email",
			"minLength": 5,
			"maxLength": 255,
		},
		"properties": []map[string]any{
			{
				"refToId":  "prop_val_1",
				"id":       "",
				"required": true,
			},
			{
				"refToId":  "prop_val_2",
				"id":       "",
				"required": false,
			},
		},
	}

	// Create an empty args instance and populate it
	args := CustomTypeArgs{}
	args.FromResourceData(data)

	// Verify the conversion
	assert.Equal(t, "EmailType", args.LocalID)
	assert.Equal(t, "Email Type", args.Name)
	assert.Equal(t, "Custom type for email validation", args.Description)
	assert.Equal(t, "string", args.Type)

	assert.Equal(t, "email", args.Config["format"])
	assert.Equal(t, args.Config["minLength"], 5)
	assert.Equal(t, args.Config["maxLength"], 255)

	assert.Len(t, args.Properties, 2)
	assert.Equal(t, "prop_val_1", args.Properties[0].ID)
	assert.Equal(t, true, args.Properties[0].Required)
	assert.Equal(t, "prop_val_2", args.Properties[1].ID)
	assert.Equal(t, false, args.Properties[1].Required)
}

func TestFromCatalogCustomType(t *testing.T) {
	// Create a catalog custom type
	customType := &localcatalog.CustomType{
		LocalID:     "ObjectType",
		Name:        "Object Type",
		Description: "Object type with properties",
		Type:        "object",
		Config:      map[string]any{},
		Properties: []localcatalog.CustomTypeProperty{
			{
				Ref:      "#/properties/group/prop1",
				Required: true,
			},
			{
				Ref:      "#/properties/group/prop2",
				Required: false,
			},
		},
	}

	// Create a mock function to return URNs
	getURN := func(ref string) string {
		if ref == "#/properties/group/prop1" {
			return "property:prop1"
		} else if ref == "#/properties/group/prop2" {
			return "property:prop2"
		}
		return ""
	}

	// Create an empty args instance and populate it
	args := CustomTypeArgs{}
	args.FromCatalogCustomType(customType, getURN)

	// Verify basic fields
	assert.Equal(t, "ObjectType", args.LocalID)
	assert.Equal(t, "Object Type", args.Name)
	assert.Equal(t, "Object type with properties", args.Description)
	assert.Equal(t, "object", args.Type)

	// Verify properties with PropertyRef
	assert.Len(t, args.Properties, 2)

	// First property reference
	propRef1, ok := args.Properties[0].RefToID.(resources.PropertyRef)
	assert.True(t, ok)
	assert.Equal(t, "property:prop1", propRef1.URN)
	assert.Equal(t, "id", propRef1.Property)
	assert.True(t, args.Properties[0].Required)

	// Second property reference
	propRef2, ok := args.Properties[1].RefToID.(resources.PropertyRef)
	assert.True(t, ok)
	assert.Equal(t, "property:prop2", propRef2.URN)
	assert.Equal(t, "id", propRef2.Property)
	assert.False(t, args.Properties[1].Required)
}
