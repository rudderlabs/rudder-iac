package definitions

import (
	"encoding/json"
	"fmt"

	"github.com/kaptinlin/jsonschema"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

var sourceTypeConfigKeys = []string{
	"connection_mode",
	"use_native_sdk",
	"consent_management",
}

// DestinationDefinition is the input to Registry.Register().
type DestinationDefinition struct {
	Type       string
	Version    int64
	Properties []converter.ConfigProperty
	SecretKeys []string
	Schema     json.RawMessage
}

// ConfigError represents a single validation failure with a JSON-pointer path.
type ConfigError struct {
	Path    string
	Message string
}

// RegisteredDefinition wraps a DestinationDefinition with compiled schema and extracted metadata.
type RegisteredDefinition struct {
	*DestinationDefinition
	compiled        *jsonschema.Schema
	sourceTypes     []string
	connectionModes map[string][]string
}

func (d *RegisteredDefinition) ValidateConfig(config map[string]any) []ConfigError {
	return validateConfig(d.compiled, config)
}

func (d *RegisteredDefinition) LocalToAPI(local map[string]any) (map[string]any, error) {
	return converter.LocalToAPI(d.Properties, local)
}

func (d *RegisteredDefinition) APIToLocal(api map[string]any) (map[string]any, error) {
	return converter.APIToLocal(d.Properties, api)
}

func (d *RegisteredDefinition) SecretKeys() []string {
	if d.DestinationDefinition == nil || d.DestinationDefinition.SecretKeys == nil {
		return []string{}
	}
	return append([]string(nil), d.DestinationDefinition.SecretKeys...)
}

func (d *RegisteredDefinition) SupportedSourceTypes() []string {
	return append([]string(nil), d.sourceTypes...)
}

func (d *RegisteredDefinition) ConnectionModes(sourceType string) ([]string, error) {
	modes, ok := d.connectionModes[sourceType]
	if !ok {
		return nil, fmt.Errorf("unsupported source type %q for destination %s", sourceType, d.Type)
	}
	return append([]string(nil), modes...), nil
}

func (d *RegisteredDefinition) IsSourceTypeSupported(sourceType string) bool {
	_, ok := d.connectionModes[sourceType]
	return ok
}

func (d *RegisteredDefinition) SourceTypeConfigKeys() []string {
	return append([]string(nil), sourceTypeConfigKeys...)
}

func newRegisteredDefinition(def *DestinationDefinition) (*RegisteredDefinition, error) {
	compiled, err := compileSchema(def.Schema)
	if err != nil {
		return nil, err
	}

	sourceTypes, connectionModes, err := extractConnectionModeMetadata(def.Schema)
	if err != nil {
		return nil, err
	}

	return &RegisteredDefinition{
		DestinationDefinition: def,
		compiled:                compiled,
		sourceTypes:             sourceTypes,
		connectionModes:         connectionModes,
	}, nil
}

func extractConnectionModeMetadata(schema json.RawMessage) ([]string, map[string][]string, error) {
	var parsed struct {
		Properties map[string]json.RawMessage `json:"properties"`
	}
	if err := json.Unmarshal(schema, &parsed); err != nil {
		return nil, nil, fmt.Errorf("parsing schema: %w", err)
	}

	connectionModeRaw, ok := parsed.Properties["connection_mode"]
	if !ok {
		return nil, map[string][]string{}, nil
	}

	var connectionMode struct {
		Properties map[string]struct {
			Enum []string `json:"enum"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(connectionModeRaw, &connectionMode); err != nil {
		return nil, nil, fmt.Errorf("parsing connection_mode schema: %w", err)
	}

	sourceTypes := make([]string, 0, len(connectionMode.Properties))
	connectionModes := make(map[string][]string, len(connectionMode.Properties))
	for sourceType, property := range connectionMode.Properties {
		sourceTypes = append(sourceTypes, sourceType)
		connectionModes[sourceType] = append([]string(nil), property.Enum...)
	}

	return sourceTypes, connectionModes, nil
}
