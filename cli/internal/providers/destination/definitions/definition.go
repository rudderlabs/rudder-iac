package definitions

import (
	"fmt"
	"maps"
	"reflect"
	"regexp"
	"slices"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

var sourceTypeConfigKeys = []string{
	"connection_mode",
}

// DestinationDefinition is the input to Registry.Register().
type DestinationDefinition struct {
	// Type is the local YAML / registry key (e.g. "s3").
	Type string
	// DisplayName is the destination's product name (e.g. "Amazon S3"), which
	// `rudder-cli docs destination` prints next to Type. Registration requires it.
	DisplayName string
	// APIType is the upstream API destination type (e.g. "S3").
	// When empty at registration, it defaults to Type.
	APIType    string
	Version    int64
	Properties []converter.ConfigProperty
	SecretKeys []string
	// JSONPathSecretKeys lists the secret keys whose value the destination
	// reads from each event at delivery when it is a JSONPath, such as http's
	// headers.from set to $.messageId. Such a value holds no secret, so
	// validate does not ask for a variable there. A JSONPath in any other
	// secret key is a literal value. Every entry must be in SecretKeys.
	JSONPathSecretKeys []string
	NewConfig          func() any
	SourceTypes        []string
	ConnectionModes    map[string][]string
	// ConnectionRequiredKeys lists, per local source type and connection mode,
	// the local config keys a source of that type must carry to connect in that
	// mode. Both dimensions matter: Braze needs rest_api_key from a cloud-mode
	// source and app_key from a device-mode one. Derived from schema.json's
	// connectionMode-conditioned configSchema.allOf branches; the backend calls
	// its equivalent check supportedSourcesValidation.
	ConnectionRequiredKeys map[string]map[string][]string
	// SyncBehaviours lists the rETL sync behaviours the destination accepts
	// (integrations-config db-config.json config.syncBehaviours). Nil means the
	// destination declares none upstream and the backend fallback applies; an
	// explicitly empty list accepts none, mirroring the backend's `??` fallback
	// on absence only.
	SyncBehaviours []string
	// SupportsVisualMapper mirrors db-config.json config.supportsVisualMapper.
	// With it, a rETL connection that names an object runs the object-mapping
	// flow; without it every connection is a JSON-mapper one.
	SupportsVisualMapper bool
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
	// fields and patterns are the described config model, built once at
	// registration so a definition Describe cannot read fails to register.
	fields   []Field
	patterns map[string]funcs.PatternInfo
}

// ValidateConfig validates config with its defaults applied, since that is what
// apply sends: a default can satisfy a conditional requirement or trigger one.
func (d *RegisteredDefinition) ValidateConfig(config map[string]any) []ConfigError {
	// A null under an unknown key isn't reported: removing the key fixes both.
	nullErrors := errorsOutside(findNullValues(config, ""), findUnknownKeys(config, d.configType, ""))
	rawErrors := slices.Concat(nullErrors, findEmptyEnumValues(config, d.fields))
	config = d.ApplyDefaults(config)
	// The source-type checks own the paths they report: they word a wrong shape
	// more specifically than model decoding does.
	sourceTypeErrors := d.validateConsentManagement(config)
	sourceTypeErrors = append(sourceTypeErrors, d.validateConnectionMode(config)...)
	errors := errorsOutside(validateConfigModel(config, d.configType, "", d.ConfigValidateFuncs...), sourceTypeErrors)
	errors = append(errors, sourceTypeErrors...)
	return append(rawErrors, errorsOutside(errors, rawErrors)...)
}

func (d *RegisteredDefinition) LocalToAPI(local map[string]any) (map[string]any, error) {
	return converter.LocalToAPI(d.Properties, local)
}

func (d *RegisteredDefinition) APIToLocal(api map[string]any) (map[string]any, error) {
	return converter.APIToLocal(d.Properties, api)
}

// SelectedKeyRoots returns the top-level config keys holding the discriminator-
// selected member of an exclusive group. Callers that prune empty values use it
// to leave those blocks alone: an empty selected member is a real setting, since
// whitelisting with no event named discards every event. A member surviving only
// because no discriminator was present carries no such meaning and is not
// protected — keeping it would emit a spec whose next apply supplies the missing
// discriminator and turns filtering on.
func (d *RegisteredDefinition) SelectedKeyRoots(apiConfig map[string]any) []string {
	var roots []string
	for _, prop := range d.Properties {
		if prop.Selector == nil {
			continue
		}
		// An absent discriminator also yields "", so this covers both.
		selected, _ := prop.Selector.LocalKeyFor(apiConfig)
		if selected == "" {
			continue
		}
		if root, _, _ := strings.Cut(selected, "."); !slices.Contains(roots, root) {
			roots = append(roots, root)
		}
	}
	return roots
}

func (d *RegisteredDefinition) SecretKeys() []string {
	if d.DestinationDefinition == nil || d.DestinationDefinition.SecretKeys == nil {
		return []string{}
	}
	return append([]string(nil), d.DestinationDefinition.SecretKeys...)
}

// JSONPathSecretKeys returns the secret keys whose value may be a JSONPath
// read from each event.
func (d *RegisteredDefinition) JSONPathSecretKeys() []string {
	if d.DestinationDefinition == nil {
		return nil
	}
	return slices.Clone(d.DestinationDefinition.JSONPathSecretKeys)
}

func (d *RegisteredDefinition) SupportedSourceTypes() []string {
	if d.DestinationDefinition == nil || len(d.SourceTypes) == 0 {
		return nil
	}
	return append([]string(nil), d.SourceTypes...)
}

// defaultSyncBehaviours is the fallback the backend applies to a definition
// that declares no syncBehaviours (config-backend
// retl/api-gateway/connection-config/constants.ts DEFAULT_SYNC_BEHAVIOURS). It
// doubles as the enum registration validates declared behaviours against; split
// the two if the backend default ever narrows below the accepted set.
var defaultSyncBehaviours = []string{"upsert", "mirror", "full"}

// SyncBehaviours returns the accepted rETL sync behaviours, falling back to the
// backend default only when the definition declares none at all.
//
// This is the destination-level list, not the set a connection may actually
// use: the backend intersects it with the source's own list and rejects
// "mirror" outright for the JSON-mapper flow (FLOWS_DISALLOWING_MIRROR in
// config-backend retl/api-gateway/connection-config/assembler.ts). So HTTP
// reads back [upsert mirror full] yet can never use "mirror", and a
// mirror-only destination like bingads_offline_conversions has no valid
// behaviour at all unless the connection carries an object.
func (d *RegisteredDefinition) SyncBehaviours() []string {
	behaviours := d.DestinationDefinition.SyncBehaviours
	if behaviours == nil {
		behaviours = defaultSyncBehaviours
	}
	// Copied by length rather than appended onto a nil slice, which would hand
	// back nil for a declared-empty list and read as absence to the caller.
	copied := make([]string, len(behaviours))
	copy(copied, behaviours)
	return copied
}

func (d *RegisteredDefinition) SupportsVisualMapper() bool {
	return d.DestinationDefinition.SupportsVisualMapper
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

// ConnectionRequiredKeys returns the keys the given source type must carry to
// connect in the given mode. A nil result means the pair requires none.
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
// for sourceType under the source-type-scoped block key. connection_mode is an
// open map, so every source type fits.
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

	fields, patterns, err := describeConfig(def, configType, keyPathSourceTypes, configDefaults)
	if err != nil {
		return nil, fmt.Errorf("describing config model: %w", err)
	}

	if err := validateSecretKeys(def.SecretKeys, fields); err != nil {
		return nil, fmt.Errorf("validating secret keys: %w", err)
	}
	for _, key := range def.JSONPathSecretKeys {
		if !slices.Contains(def.SecretKeys, key) {
			return nil, fmt.Errorf("validating secret keys: JSONPath secret key %q is not a secret key", key)
		}
	}

	return &RegisteredDefinition{
		DestinationDefinition: def,
		configType:            configType,
		keyPathSourceTypes:    keyPathSourceTypes,
		configDefaults:        configDefaults,
		fields:                fields,
		patterns:              patterns,
	}, nil
}

// snakeCaseKey is a dotted path of snake_case segments. secret.VariableName
// upper-cases each segment into the secret's variable name, so such a key
// always adds valid characters and only the spec ID can make the name invalid.
var snakeCaseKey = regexp.MustCompile(`^[a-z0-9_]+(\.[a-z0-9_]+)*$`)

// validateSecretKeys requires every secret key to name a string field of the
// described config, through nested objects and lists of objects. Apply, import
// and the docs command all act on these paths, and a path the config model
// lacks would leave a secret unmasked with nothing to flag it.
func validateSecretKeys(secretKeys []string, fields []Field) error {
	types := make(map[string]string, len(fields))
	for _, f := range fields {
		types[strings.ReplaceAll(f.Key, "[]", "")] = f.Type
	}
	for _, key := range secretKeys {
		fieldType, ok := types[key]
		if !ok {
			return fmt.Errorf("secret key %q does not name a string field of the config model", key)
		}
		if fieldType != "string" {
			return fmt.Errorf("secret key %q names a field of type %s, not string", key, fieldType)
		}
		if !snakeCaseKey.MatchString(key) {
			return fmt.Errorf("secret key %q is not snake_case, so its secret's variable name could be invalid", key)
		}
	}
	return nil
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
