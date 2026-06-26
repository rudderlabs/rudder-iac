package definitions

import (
	"fmt"
	"reflect"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

var sourceTypeConfigKeys = []string{
	"connection_mode",
	"use_native_sdk",
	"consent_management",
}

// DestinationDefinition is the input to Registry.Register().
type DestinationDefinition struct {
	Type            string
	Version         int64
	Properties      []converter.ConfigProperty
	SecretKeys      []string
	NewConfig       func() any
	SourceTypes     []string
	ConnectionModes map[string][]string
}

// ConfigError represents a single validation failure with a JSON-pointer path.
type ConfigError struct {
	Path    string
	Message string
}

// RegisteredDefinition wraps a DestinationDefinition with config model metadata.
type RegisteredDefinition struct {
	*DestinationDefinition
	configType reflect.Type
}

func (d *RegisteredDefinition) ValidateConfig(config map[string]any) []ConfigError {
	return validateConfigModel(config, d.configType, "")
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
	if d.DestinationDefinition == nil || len(d.SourceTypes) == 0 {
		return nil
	}
	return append([]string(nil), d.SourceTypes...)
}

func (d *RegisteredDefinition) ConnectionModes(sourceType string) ([]string, error) {
	modes, ok := d.DestinationDefinition.ConnectionModes[sourceType]
	if !ok {
		return nil, fmt.Errorf("unsupported source type %q for destination %s", sourceType, d.Type)
	}
	return append([]string(nil), modes...), nil
}

func (d *RegisteredDefinition) IsSourceTypeSupported(sourceType string) bool {
	_, ok := d.DestinationDefinition.ConnectionModes[sourceType]
	return ok
}

func (d *RegisteredDefinition) SourceTypeConfigKeys() []string {
	return append([]string(nil), sourceTypeConfigKeys...)
}

func newRegisteredDefinition(def *DestinationDefinition) (*RegisteredDefinition, error) {
	if def.NewConfig == nil {
		return nil, fmt.Errorf("NewConfig is required")
	}

	sample := def.NewConfig()
	configType := reflect.TypeOf(sample)
	if configType == nil || configType.Kind() != reflect.Pointer || configType.Elem().Kind() != reflect.Struct {
		return nil, fmt.Errorf("NewConfig must return a pointer to struct")
	}

	return &RegisteredDefinition{
		DestinationDefinition: def,
		configType:            configType.Elem(),
	}, nil
}
