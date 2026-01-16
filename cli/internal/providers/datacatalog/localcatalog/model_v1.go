package localcatalog

import (
	"encoding/json"
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/utils"
)

type PropertyV1 struct {
	LocalID     string                 `mapstructure:"id" json:"id"`
	Name        string                 `mapstructure:"name" json:"name"`
	Description string                 `mapstructure:"description,omitempty" json:"description"`
	Type        string                 `mapstructure:"type,omitempty" json:"type"`
	Config      map[string]interface{} `mapstructure:"config,omitempty" json:"config,omitempty"`
}

type PropertySpecV1 struct {
	Properties []PropertyV1 `mapstructure:"properties" json:"properties"`
}

func (p *PropertyV1) FromV0(v0 Property) error {
	p.LocalID = v0.LocalID
	p.Name = v0.Name
	p.Description = v0.Description
	p.Type = v0.Type
	p.Config = convertConfigKeysToSnakeCase(v0.Config)
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
	name, ok := s.Metadata["name"].(string)
	if !ok {
		name = ""
	}
	switch s.Kind {
	case KindProperties:
		pSpec := PropertySpecV1{}
		if err := extractSpec(s.Spec, &pSpec); err != nil {
			return fmt.Errorf("extracting the property spec: %w", err)
		}
		dc.Properties[EntityGroup(name)] = pSpec.Properties

	default:
		return fmt.Errorf("unknown kind: %s", s.Kind)

	}

	return nil
}
