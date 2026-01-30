package localcatalog

import (
	"encoding/json"
	"fmt"
	"strings"

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
	p.Config = convertConfigKeysToSnakeCase(v0.Config)

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
			if itemTypesArray, ok := itemTypes.([]interface{}); ok {
				// Check if single or multiple item types
				if len(itemTypesArray) == 1 {
					// Single item type
					if itemTypeStr, ok := itemTypesArray[0].(string); ok {
						p.ItemType = itemTypeStr
					}
				} else if len(itemTypesArray) > 1 {
					// Multiple item types
					p.ItemTypes = make([]string, len(itemTypesArray))
					for i, item := range itemTypesArray {
						if itemStr, ok := item.(string); ok {
							p.ItemTypes[i] = itemStr
						}
					}
				}
				// Remove item_types from config after migration
				delete(p.Config, "item_types")
			}
		}
	}

	return nil
}

// convertConfigKeysToSnakeCase converts camelCase config keys to snake_case
// for v1 spec format. This handles the migration from v0 to v1 config format.
func convertConfigKeysToSnakeCase(config map[string]interface{}) map[string]interface{} {
	if config == nil {
		return nil
	}

	converted := make(map[string]interface{})
	for key, value := range config {
		snakeKey := utils.ToSnakeCase(key)
		converted[snakeKey] = value
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

func extractEntititesV1(s *specs.Spec, dc *DataCatalog) error {
	switch s.Kind {
	case KindProperties:
		pSpec := PropertySpecV1{}
		if err := extractSpec(s.Spec, &pSpec); err != nil {
			return fmt.Errorf("extracting the property spec: %w", err)
		}
		for _, prop := range pSpec.Properties {
			dc.Properties[prop.LocalID] = prop
		}

	default:
		return fmt.Errorf("unknown kind: %s", s.Kind)

	}

	return nil
}
