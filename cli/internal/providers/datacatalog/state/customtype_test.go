package state

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
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
			"format":     "email",
			"min_length": 5,
			"max_length": 255,
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
	assert.Equal(t, 5, config["min_length"])
	assert.Equal(t, 255, config["max_length"])

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
			"format":     "email",
			"min_length": 5,
			"max_length": 255,
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
	assert.Equal(t, args.Config["min_length"], 5)
	assert.Equal(t, args.Config["max_length"], 255)

	assert.Len(t, args.Properties, 2)
	assert.Equal(t, "prop_val_1", args.Properties[0].ID)
	assert.Equal(t, true, args.Properties[0].Required)
	assert.Equal(t, "prop_val_2", args.Properties[1].ID)
	assert.Equal(t, false, args.Properties[1].Required)
}

func TestFromCatalogCustomType(t *testing.T) {
	// Create a catalog custom type
	customType := &localcatalog.CustomTypeV1{
		LocalID:     "ObjectType",
		Name:        "Object Type",
		Description: "Object type with properties",
		Type:        "object",
		Config:      map[string]any{},
		Properties: []localcatalog.CustomTypePropertyV1{
			{
				Property: "#property:prop1",
				Required: true,
			},
			{
				Property: "#property:prop2",
				Required: false,
			},
		},
	}

	// Create a mock function to return URNs
	urnFromRef := func(ref string) string {
		switch ref {
		case "#property:prop1":
			return "property:prop1"
		case "#property:prop2":
			return "property:prop2"
		}
		return ""
	}

	// Create an empty args instance and populate it
	args := CustomTypeArgs{}
	args.FromCatalogCustomType(customType, urnFromRef)

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

func TestCustomTypePropertyDiff(t *testing.T) {
	tests := []struct {
		name     string
		prop1    *CustomTypeProperty
		prop2    *CustomTypeProperty
		expected bool
	}{
		{
			name: "identical properties",
			prop1: &CustomTypeProperty{
				RefToID:  "ref1",
				ID:       "id1",
				Required: true,
			},
			prop2: &CustomTypeProperty{
				RefToID:  "ref1",
				ID:       "id1",
				Required: true,
			},
			expected: false,
		},
		{
			name: "different ID",
			prop1: &CustomTypeProperty{
				RefToID:  "ref1",
				ID:       "id1",
				Required: true,
			},
			prop2: &CustomTypeProperty{
				RefToID:  "ref1",
				ID:       "id2",
				Required: true,
			},
			expected: true,
		},
		{
			name: "different Required",
			prop1: &CustomTypeProperty{
				RefToID:  "ref1",
				ID:       "id1",
				Required: true,
			},
			prop2: &CustomTypeProperty{
				RefToID:  "ref1",
				ID:       "id1",
				Required: false,
			},
			expected: true,
		},
		{
			name: "different RefToID",
			prop1: &CustomTypeProperty{
				RefToID:  "ref1",
				ID:       "id1",
				Required: true,
			},
			prop2: &CustomTypeProperty{
				RefToID:  "ref2",
				ID:       "id1",
				Required: true,
			},
			expected: true,
		},
		{
			name: "complex RefToID objects",
			prop1: &CustomTypeProperty{
				RefToID: resources.PropertyRef{
					URN:      "property:prop1",
					Property: "id",
				},
				ID:       "id1",
				Required: true,
			},
			prop2: &CustomTypeProperty{
				RefToID: resources.PropertyRef{
					URN:      "property:prop2",
					Property: "id",
				},
				ID:       "id1",
				Required: true,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.prop1.Diff(tt.prop2)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCustomTypeArgs_Diff(t *testing.T) {
	baseArgs := &CustomTypeArgs{
		LocalID:     "TestType",
		Name:        "Test Type",
		Description: "Test description",
		Type:        "string",
		Config: map[string]any{
			"format": "email",
			"length": 100,
		},
		Properties: []*CustomTypeProperty{
			{
				RefToID:  "ref1",
				ID:       "prop1",
				Required: true,
			},
		},
		Variants: Variants{
			{
				Type:          "string",
				Discriminator: "type",
				Cases:         []VariantCase{},
				Default:       []PropertyReference{},
			},
		},
	}

	tests := []struct {
		name     string
		args1    *CustomTypeArgs
		args2    *CustomTypeArgs
		expected bool
	}{
		{
			name:     "identical args",
			args1:    baseArgs,
			args2:    baseArgs,
			expected: false,
		},
		{
			name:  "different LocalID",
			args1: baseArgs,
			args2: &CustomTypeArgs{
				LocalID:     "DifferentType",
				Name:        "Test Type",
				Description: "Test description",
				Type:        "string",
				Config: map[string]any{
					"format": "email",
					"length": 100,
				},
				Properties: []*CustomTypeProperty{
					{
						RefToID:  "ref1",
						ID:       "prop1",
						Required: true,
					},
				},
				Variants: Variants{
					{
						Type:          "string",
						Discriminator: "type",
						Cases:         []VariantCase{},
						Default:       []PropertyReference{},
					},
				},
			},
			expected: true,
		},
		{
			name:  "different Name",
			args1: baseArgs,
			args2: &CustomTypeArgs{
				LocalID:     "TestType",
				Name:        "Different Name",
				Description: "Test description",
				Type:        "string",
				Config: map[string]any{
					"format": "email",
					"length": 100,
				},
				Properties: []*CustomTypeProperty{
					{
						RefToID:  "ref1",
						ID:       "prop1",
						Required: true,
					},
				},
				Variants: Variants{
					{
						Type:          "string",
						Discriminator: "type",
						Cases:         []VariantCase{},
						Default:       []PropertyReference{},
					},
				},
			},
			expected: true,
		},
		{
			name:  "different Description",
			args1: baseArgs,
			args2: &CustomTypeArgs{
				LocalID:     "TestType",
				Name:        "Test Type",
				Description: "Different description",
				Type:        "string",
				Config: map[string]any{
					"format": "email",
					"length": 100,
				},
				Properties: []*CustomTypeProperty{
					{
						RefToID:  "ref1",
						ID:       "prop1",
						Required: true,
					},
				},
				Variants: Variants{
					{
						Type:          "string",
						Discriminator: "type",
						Cases:         []VariantCase{},
						Default:       []PropertyReference{},
					},
				},
			},
			expected: true,
		},
		{
			name:  "different Type",
			args1: baseArgs,
			args2: &CustomTypeArgs{
				LocalID:     "TestType",
				Name:        "Test Type",
				Description: "Test description",
				Type:        "number",
				Config: map[string]any{
					"format": "email",
					"length": 100,
				},
				Properties: []*CustomTypeProperty{
					{
						RefToID:  "ref1",
						ID:       "prop1",
						Required: true,
					},
				},
				Variants: Variants{
					{
						Type:          "string",
						Discriminator: "type",
						Cases:         []VariantCase{},
						Default:       []PropertyReference{},
					},
				},
			},
			expected: true,
		},
		{
			name:  "different Config",
			args1: baseArgs,
			args2: &CustomTypeArgs{
				LocalID:     "TestType",
				Name:        "Test Type",
				Description: "Test description",
				Type:        "string",
				Config: map[string]any{
					"format": "url",
					"length": 200,
				},
				Properties: []*CustomTypeProperty{
					{
						RefToID:  "ref1",
						ID:       "prop1",
						Required: true,
					},
				},
				Variants: Variants{
					{
						Type:          "string",
						Discriminator: "type",
						Cases:         []VariantCase{},
						Default:       []PropertyReference{},
					},
				},
			},
			expected: true,
		},
		{
			name:  "different Properties length",
			args1: baseArgs,
			args2: &CustomTypeArgs{
				LocalID:     "TestType",
				Name:        "Test Type",
				Description: "Test description",
				Type:        "string",
				Config: map[string]any{
					"format": "email",
					"length": 100,
				},
				Properties: []*CustomTypeProperty{
					{
						RefToID:  "ref1",
						ID:       "prop1",
						Required: true,
					},
					{
						RefToID:  "ref2",
						ID:       "prop2",
						Required: false,
					},
				},
				Variants: Variants{
					{
						Type:          "string",
						Discriminator: "type",
						Cases:         []VariantCase{},
						Default:       []PropertyReference{},
					},
				},
			},
			expected: true,
		},
		{
			name:  "property not found",
			args1: baseArgs,
			args2: &CustomTypeArgs{
				LocalID:     "TestType",
				Name:        "Test Type",
				Description: "Test description",
				Type:        "string",
				Config: map[string]any{
					"format": "email",
					"length": 100,
				},
				Properties: []*CustomTypeProperty{
					{
						RefToID:  "ref1",
						ID:       "different_prop",
						Required: true,
					},
				},
				Variants: Variants{
					{
						Type:          "string",
						Discriminator: "type",
						Cases:         []VariantCase{},
						Default:       []PropertyReference{},
					},
				},
			},
			expected: true,
		},
		{
			name:  "property differs",
			args1: baseArgs,
			args2: &CustomTypeArgs{
				LocalID:     "TestType",
				Name:        "Test Type",
				Description: "Test description",
				Type:        "string",
				Config: map[string]any{
					"format": "email",
					"length": 100,
				},
				Properties: []*CustomTypeProperty{
					{
						RefToID:  "ref1",
						ID:       "prop1",
						Required: false, // Different from baseArgs
					},
				},
				Variants: Variants{
					{
						Type:          "string",
						Discriminator: "type",
						Cases:         []VariantCase{},
						Default:       []PropertyReference{},
					},
				},
			},
			expected: true,
		},
		{
			name:  "different Variants",
			args1: baseArgs,
			args2: &CustomTypeArgs{
				LocalID:     "TestType",
				Name:        "Test Type",
				Description: "Test description",
				Type:        "string",
				Config: map[string]any{
					"format": "email",
					"length": 100,
				},
				Properties: []*CustomTypeProperty{
					{
						RefToID:  "ref1",
						ID:       "prop1",
						Required: true,
					},
				},
				Variants: Variants{
					{
						Type:          "number",
						Discriminator: "type",
						Cases:         []VariantCase{},
						Default:       []PropertyReference{},
					},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.args1.Diff(tt.args2)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPropertyByID(t *testing.T) {
	args := &CustomTypeArgs{
		Properties: []*CustomTypeProperty{
			{
				ID:       "prop1",
				RefToID:  "ref1",
				Required: true,
			},
			{
				ID:       "prop2",
				RefToID:  "ref2",
				Required: false,
			},
		},
	}

	// Test existing property
	prop := args.PropertyByID("prop1")
	assert.NotNil(t, prop)
	assert.Equal(t, "prop1", prop.ID)
	assert.Equal(t, "ref1", prop.RefToID)
	assert.True(t, prop.Required)

	// Test non-existing property
	prop = args.PropertyByID("nonexistent")
	assert.Nil(t, prop)
}
