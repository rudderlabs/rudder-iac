package definitions

import (
	"fmt"
	"maps"
	"reflect"
	"slices"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
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
	// ConnectionRequiredKeys lists, per local source type and connection
	// mode, the local config keys that must be present for a source of that
	// type to connect in that mode. It mirrors the backend's connect-time
	// check, which is named supportedSourcesValidation there — a config-backend
	// entity, not a db-config key, so do not go looking for one.
	// Requiredness depends on both source type and mode: Braze
	// needs rest_api_key from a cloud-mode source and app_key from a
	// device-mode one. Derived from schema.json's connectionMode-conditioned
	// configSchema.allOf branches; pairs without an entry have no connect-time
	// required keys.
	ConnectionRequiredKeys map[string]map[string][]string
	// ConsentValidationOverrides replaces canonical consent validation for selected local source types.
	ConsentValidationOverrides map[string]common.ConsentValidator
	// ConfigValidateFuncs registers extra go-playground custom validate tags,
	// scoped to this definition's config struct alone. Use it for
	// destination-specific constraints a built-in cross-field tag cannot
	// express (e.g. a condition keyed on a map entry), never for fleet-wide
	// conventions.
	ConfigValidateFuncs []rules.CustomValidateFunc
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
	errors := validateConfigModel(config, d.configType, "", d.ConfigValidateFuncs...)
	errors = append(errors, d.validateConsentManagement(config)...)
	return append(errors, d.validateConnectionMode(config)...)
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

// ConnectionRequiredKeys returns the local config keys that must be
// present for the given source type to connect in the given mode — one mode
// value, not the connection_mode block that carries it. A nil result means the
// pair has no connect-time required keys.
func (d *RegisteredDefinition) ConnectionRequiredKeys(sourceType, mode string) []string {
	fields, ok := d.DestinationDefinition.ConnectionRequiredKeys[sourceType][mode]
	if !ok {
		return nil
	}
	return append([]string(nil), fields...)
}

func (d *RegisteredDefinition) SourceTypeConfigKeys() []string {
	return append([]string(nil), sourceTypeConfigKeys...)
}

// AcceptsSourceTypeEntry reports whether the config model would accept an entry
// for sourceType under the source-type-scoped block key. The two blocks are
// shaped differently: connection_mode is an open map, so every source type
// fits, while use_native_sdk is a struct naming one field per source type, so
// only those do.
func (d *RegisteredDefinition) AcceptsSourceTypeEntry(key, sourceType string) bool {
	field, ok := structFieldsByMapstructureTag(d.configType)[key]
	if !ok {
		return false
	}
	if derefType(field.Type).Kind() == reflect.Map {
		return true
	}
	return configStructHasKeyPath(d.configType, key+"."+sourceType)
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

	if err := validateConnectionModeConfigModel(configType); err != nil {
		return nil, fmt.Errorf("validating connection mode config model: %w", err)
	}

	if err := validateConnectionRequiredKeys(def, configType); err != nil {
		return nil, fmt.Errorf("validating connection required keys: %w", err)
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

// validateConnectionRequiredKeys rejects entries for source types outside
// SourceTypes, connection modes outside that source type's ConnectionModes,
// and required keys outside the local config surface (the config struct plus
// the source-type block keys) — the config model is a closed allowlist, so an
// unknown required key could never be satisfied.
func validateConnectionRequiredKeys(def *DestinationDefinition, configType reflect.Type) error {
	configFields := structFieldsByMapstructureTag(configType)

	for _, sourceType := range slices.Sorted(maps.Keys(def.ConnectionRequiredKeys)) {
		if !slices.Contains(def.SourceTypes, sourceType) {
			return fmt.Errorf("connection required keys configured for unsupported source type %q", sourceType)
		}

		byMode := def.ConnectionRequiredKeys[sourceType]
		if len(byMode) == 0 {
			return fmt.Errorf("connection required keys for source type %q list no connection modes", sourceType)
		}

		for _, mode := range slices.Sorted(maps.Keys(byMode)) {
			if !slices.Contains(def.ConnectionModes[sourceType], mode) {
				return fmt.Errorf("connection required keys for source type %q reference unsupported connection mode %q", sourceType, mode)
			}

			requiredKeys := byMode[mode]
			if len(requiredKeys) == 0 {
				return fmt.Errorf("connection required keys for source type %q in mode %q are empty", sourceType, mode)
			}
			for _, key := range requiredKeys {
				if _, ok := configFields[key]; ok || slices.Contains(sourceTypeConfigKeys, key) {
					continue
				}
				return fmt.Errorf("connection required keys for source type %q in mode %q reference unknown config key %q", sourceType, mode, key)
			}
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

// validateConnectionModeConfigModel mirrors the consent type check. A bespoke
// type here would silently opt out of validateConnectionMode's per-source-type
// enum check — its type assertion would yield an empty map — so a mistyped
// field surfaces at registration rather than as a decode error at apply time.
func validateConnectionModeConfigModel(configType reflect.Type) error {
	field, ok := structFieldsByMapstructureTag(configType)["connection_mode"]
	if ok && derefType(field.Type) != reflect.TypeOf(common.ConnectionMode{}) {
		return fmt.Errorf("connection_mode config field must use common.ConnectionMode")
	}
	return nil
}
