package definitions

import (
	"fmt"
	"reflect"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

var sourceTypeConfigKeys = []string{
	"connection_mode",
	"use_native_sdk",
}

// DestinationDefinition is the input to Registry.Register().
type DestinationDefinition struct {
	// Type is the local YAML / registry key (e.g. "s3").
	Type string
	// APIType is the upstream API destination type (e.g. "S3").
	// When empty at registration, it defaults to Type.
	APIType         string
	Version         int64
	Properties      []converter.ConfigProperty
	SecretKeys      []string
	NewConfig       func() any
	SourceTypes     []string
	ConnectionModes map[string][]string
	// ConsentValidationOverrides replaces canonical consent validation for selected local source types.
	ConsentValidationOverrides map[string]common.ConsentValidator
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
	// keyPathSourceTypes is the reverse index of gated properties:
	// local config keypath (JSON pointer) -> source types entitled to it.
	keyPathSourceTypes map[string][]string
}

func (d *RegisteredDefinition) ValidateConfig(config map[string]any) []ConfigError {
	errors := validateConfigModel(config, d.configType, "")
	return append(errors, d.validateConsentManagement(config)...)
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

// LocalSourceTypeKeys returns keys allowed under source-type-scoped config blocks.
func (d *RegisteredDefinition) LocalSourceTypeKeys() []string {
	return d.SupportedSourceTypes()
}

func (d *RegisteredDefinition) ConnectionModes(sourceType string) ([]string, error) {
	modes, ok := d.DestinationDefinition.ConnectionModes[sourceType]
	if !ok {
		return nil, fmt.Errorf("unsupported source type %q for destination %s", sourceType, d.Type)
	}
	return append([]string(nil), modes...), nil
}

func (d *RegisteredDefinition) SourceTypeConfigKeys() []string {
	return append([]string(nil), sourceTypeConfigKeys...)
}

// GatedKeyPaths returns local config keypaths (JSON pointer, e.g.
// "/event_upload_period_millis") mapped to the source types entitled to use
// them. Keypaths absent from the map are default keys, allowed for every
// connected source type.
func (d *RegisteredDefinition) GatedKeyPaths() map[string][]string {
	out := make(
		map[string][]string,
		len(d.keyPathSourceTypes),
	)

	for keyPath, sourceTypes := range d.keyPathSourceTypes {
		out[keyPath] = append([]string(nil), sourceTypes...)
	}

	return out
}

func newRegisteredDefinition(def *DestinationDefinition) (*RegisteredDefinition, error) {
	if def.NewConfig == nil {
		return nil, fmt.Errorf("NewConfig is required")
	}

	var (
		sample     = def.NewConfig()
		configType = reflect.TypeOf(sample)
	)

	if configType == nil || configType.Kind() != reflect.Pointer || configType.Elem().Kind() != reflect.Struct {
		return nil, fmt.Errorf("NewConfig must return a pointer to struct")
	}
	configType = configType.Elem()

	if err := validateConsentConfigModel(def, configType); err != nil {
		return nil, fmt.Errorf("validating consent config model: %w", err)
	}

	keyPathSourceTypes, err := buildGatedKeyPaths(def, configType)
	if err != nil {
		return nil, fmt.Errorf("building gated key paths: %w", err)
	}

	return &RegisteredDefinition{
		DestinationDefinition: def,
		configType:            configType,
		keyPathSourceTypes:    keyPathSourceTypes,
	}, nil
}

func validateConsentConfigModel(def *DestinationDefinition, configType reflect.Type) error {
	consentField, hasConsentField := structFieldsByMapstructureTag(configType)["consent_management"]
	if hasConsentField && derefType(consentField.Type) != reflect.TypeOf(common.ConsentManagement{}) {
		return fmt.Errorf("consent_management config field must use common.ConsentManagement")
	}
	if len(def.ConsentValidationOverrides) > 0 && !hasConsentField {
		return fmt.Errorf("consent validation overrides require a common.ConsentManagement config field")
	}
	return nil
}
