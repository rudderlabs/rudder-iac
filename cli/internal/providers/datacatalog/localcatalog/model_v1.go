package localcatalog

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/utils"
)

type PropertyV1 struct {
	LocalID     string                 `mapstructure:"id" json:"id"`
	Name        string                 `mapstructure:"name" json:"name"`
	Description string                 `mapstructure:"description,omitempty" json:"description"`
	Type        string                 `mapstructure:"type,omitempty" json:"type"`
	Types       []string               `mapstructure:"types,omitempty" json:"types,omitempty"`
	ItemType    string                 `mapstructure:"item_type,omitempty" json:"item_type,omitempty"`
	ItemTypes   []string               `mapstructure:"item_types,omitempty" json:"item_types,omitempty"`
	Config      map[string]interface{} `mapstructure:"config,omitempty" json:"config,omitempty"`
}

type PropertySpecV1 struct {
	Properties []PropertyV1 `mapstructure:"properties" json:"properties"`
}

func (p *PropertyV1) FromV0(v0 Property) error {
	p.LocalID = v0.LocalID
	p.Name = v0.Name
	p.Description = v0.Description
	p.Config = ConvertConfigKeysToSnakeCase(v0.Config)

	// Parse the v0 type field to determine if it's single or multiple types
	if strings.Contains(v0.Type, ",") {
		// Multiple types - split and trim
		p.Types = utils.SplitMultiTypeString(v0.Type)
		p.Type = "" // Clear single type field
	} else {
		// Single type - keep as-is
		p.Type = v0.Type
		p.Types = nil
	}

	// Extract itemTypes from config and move to property-level fields
	if p.Config != nil {
		if itemTypes, ok := p.Config["item_types"]; ok {
			itemTypesArray, ok := itemTypes.([]interface{})
			if !ok {
				return fmt.Errorf("config['item_types'] must be an array of strings")
			}

			// Check if single or multiple item types
			if len(itemTypesArray) == 1 {
				// Single item type
				itemTypeStr, ok := itemTypesArray[0].(string)
				if !ok {
					return fmt.Errorf("config['item_types']: item type must be a string")
				}
				p.ItemType = itemTypeStr
			} else if len(itemTypesArray) > 1 {
				// Multiple item types
				p.ItemTypes = make([]string, len(itemTypesArray))
				for i, item := range itemTypesArray {
					itemStr, ok := item.(string)
					if !ok {
						return fmt.Errorf("config['item_types']: item type must be a string")
					}
					p.ItemTypes[i] = itemStr
				}
			}
			// Remove item_types from config after migration
			delete(p.Config, "item_types")

		}
	}

	return nil
}

var SupportedV0ConfigKeys = []string{
	"enum",
	"exclusiveMaximum",
	"exclusiveMinimum",
	"format",
	"itemTypes",
	"maxItems",
	"maxLength",
	"maximum",
	"minItems",
	"minLength",
	"minimum",
	"multipleOf",
	"pattern",
	"uniqueItems",
}

// ConvertConfigKeysToSnakeCase converts camelCase config keys to snake_case
// for v1 spec format. This handles the migration from v0 to v1 config format.
func ConvertConfigKeysToSnakeCase(config map[string]interface{}) map[string]interface{} {
	if config == nil {
		return nil
	}

	snakeCaseNamer := namer.NewSnakeCase()

	converted := make(map[string]interface{})
	for key, value := range config {
		convertedKey := key
		if slices.Contains(SupportedV0ConfigKeys, key) {
			convertedKey = snakeCaseNamer.Name(key)
		}
		converted[convertedKey] = value
	}

	return converted
}

func (p *PropertySpecV1) FromV0(v0 PropertySpec) error {
	p.Properties = make([]PropertyV1, 0, len(v0.Properties))
	for _, v0Property := range v0.Properties {
		v1Property := PropertyV1{}
		err := v1Property.FromV0(v0Property)
		if err != nil {
			return err
		}
		p.Properties = append(p.Properties, v1Property)
	}
	return nil
}

func extractSpec(spec map[string]any, result any) error {
	jsonByt, err := json.Marshal(spec)
	if err != nil {
		return fmt.Errorf("marshalling the spec: %w", err)
	}

	if err := strictUnmarshal(jsonByt, result); err != nil {
		return fmt.Errorf("extracting the spec: %w", err)
	}

	return nil
}

// CustomTypeSpecV1 represents the spec section of a custom-types resource (V1 spec)
type CustomTypeSpecV1 struct {
	Types []CustomTypeV1 `json:"types"`
}

// CustomTypeV1 represents a user-defined custom type (V1 spec)
type CustomTypeV1 struct {
	LocalID     string                 `mapstructure:"id" json:"id"`
	Name        string                 `mapstructure:"name" json:"name"`
	Description string                 `mapstructure:"description,omitempty" json:"description,omitempty"`
	Type        string                 `mapstructure:"type" json:"type"`
	Config      map[string]any         `mapstructure:"config,omitempty" json:"config,omitempty"`
	Properties  []CustomTypePropertyV1 `mapstructure:"properties,omitempty" json:"properties,omitempty"`
	Variants    VariantsV1             `mapstructure:"variants,omitempty" json:"variants,omitempty"`
}

// CustomTypePropertyV1 represents a property reference within a custom type (V1 spec)
type CustomTypePropertyV1 struct {
	Property string `mapstructure:"property" json:"property"`
	Required bool   `mapstructure:"required" json:"required"`
}

func (c *CustomTypeV1) FromV0(v0 CustomType) error {
	c.LocalID = v0.LocalID
	c.Name = v0.Name
	c.Description = v0.Description
	c.Type = v0.Type
	c.Config = ConvertConfigKeysToSnakeCase(v0.Config)

	// Convert properties from V0 to V1
	if len(v0.Properties) > 0 {
		c.Properties = make([]CustomTypePropertyV1, 0, len(v0.Properties))
		for _, v0Property := range v0.Properties {
			c.Properties = append(c.Properties, CustomTypePropertyV1{
				Property: v0Property.Ref,
				Required: v0Property.Required,
			})
		}
	}

	// Convert variants from V0 to V1
	err := c.Variants.FromV0(v0.Variants)
	if err != nil {
		return fmt.Errorf("converting variants to v1: %w", err)
	}

	return nil
}

func extractEntititesV1(s *specs.Spec, dc *DataCatalog) error {
	switch s.Kind {
	case KindProperties:
		pSpec := PropertySpecV1{}
		if err := extractSpec(s.Spec, &pSpec); err != nil {
			return fmt.Errorf("extracting the property spec: %w", err)
		}
		dc.Properties = append(dc.Properties, pSpec.Properties...)

	case KindEvents:
		eventSpec := EventSpecV1{}
		if err := extractSpec(s.Spec, &eventSpec); err != nil {
			return fmt.Errorf("extracting the event spec: %w", err)
		}
		dc.Events = append(dc.Events, eventSpec.Events...)

	case KindCategories:
		categorySpec := CategorySpecV1{}
		if err := extractSpec(s.Spec, &categorySpec); err != nil {
			return fmt.Errorf("extracting the category spec: %w", err)
		}
		dc.Categories = append(dc.Categories, categorySpec.Categories...)

	case KindTrackingPlansV1:
		tpSpec := TrackingPlanV1{}
		if err := extractSpec(s.Spec, &tpSpec); err != nil {
			return fmt.Errorf("extracting the tracking plan spec: %w", err)
		}

		// Check for duplicates
		for i := range dc.TrackingPlans {
			if dc.TrackingPlans[i].LocalID == tpSpec.LocalID {
				return fmt.Errorf("duplicate tracking plan with id '%s' found", tpSpec.LocalID)
			}
		}
		dc.TrackingPlans = append(dc.TrackingPlans, &tpSpec)

	case KindCustomTypes:
		ctSpec := CustomTypeSpecV1{}
		if err := extractSpec(s.Spec, &ctSpec); err != nil {
			return fmt.Errorf("extracting the custom types spec: %w", err)
		}
		dc.CustomTypes = append(dc.CustomTypes, ctSpec.Types...)

	default:
		return fmt.Errorf("unknown kind: %s", s.Kind)

	}

	return nil
}
