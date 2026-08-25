package definitions

import (
	"fmt"
	"maps"
	"reflect"
	"slices"

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
	// SupportedSourcesValidation lists, per local source type, the local config
	// keys that must be present for a source of that type to connect (mirrors
	// the backend's connect-time supportedSourcesValidation check). Source
	// types without an entry have no connect-time required keys.
	SupportedSourcesValidation map[string][]string
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
	// configDefaults holds the config model's `default` tags keyed by local
	// config key, applied to local specs via ApplyDefaults.
	configDefaults map[string]any
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

// SupportedSourcesValidation returns the local config keys that must be
// present for the given source type to connect. A nil result means the source
// type has no connect-time required keys.
func (d *RegisteredDefinition) SupportedSourcesValidation(sourceType string) []string {
	fields, ok := d.DestinationDefinition.SupportedSourcesValidation[sourceType]
	if !ok {
		return nil
	}
	return append([]string(nil), fields...)
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

	if err := validateSupportedSourcesValidation(def, configType); err != nil {
		return nil, fmt.Errorf("validating supported sources requirements: %w", err)
	}

	keyPathSourceTypes, err := buildGatedKeyPaths(def, configType)
	if err != nil {
		return nil, fmt.Errorf("building gated key paths: %w", err)
	}

	configDefaults, err := buildConfigDefaults(configType)
	if err != nil {
		return nil, fmt.Errorf("building config defaults: %w", err)
	}

	return &RegisteredDefinition{
		DestinationDefinition: def,
		configType:            configType,
		keyPathSourceTypes:    keyPathSourceTypes,
		configDefaults:        configDefaults,
	}, nil
}

// validateSupportedSourcesValidation rejects entries for source types outside
// SourceTypes and required keys outside the local config surface (the config
// struct plus the source-type block keys) — the config model is a closed
// allowlist, so an unknown required key could never be satisfied.
func validateSupportedSourcesValidation(def *DestinationDefinition, configType reflect.Type) error {
	configFields := structFieldsByMapstructureTag(configType)

	for _, sourceType := range slices.Sorted(maps.Keys(def.SupportedSourcesValidation)) {
		if !slices.Contains(def.SourceTypes, sourceType) {
			return fmt.Errorf("supported sources validation configured for unsupported source type %q", sourceType)
		}

		requiredKeys := def.SupportedSourcesValidation[sourceType]
		if len(requiredKeys) == 0 {
			return fmt.Errorf("supported sources validation for source type %q has no required config keys", sourceType)
		}
		for _, key := range requiredKeys {
			if _, ok := configFields[key]; ok || slices.Contains(sourceTypeConfigKeys, key) {
				continue
			}
			return fmt.Errorf("supported sources validation for source type %q references unknown config key %q", sourceType, key)
		}
	}
	return nil
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
