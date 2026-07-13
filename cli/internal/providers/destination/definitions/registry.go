package definitions

import (
	"fmt"
	"maps"
	"slices"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
)

type typeVersion struct {
	Type    string
	Version int64
}

type apiTypeVersion struct {
	APIType string
	Version int64
}

// Registry stores registered destination definitions keyed by (Type, Version)
// with a reverse index by (APIType, Version) for import/remote lookups.
type Registry struct {
	byTypeVersion    map[typeVersion]*RegisteredDefinition
	byAPITypeVersion map[apiTypeVersion]*RegisteredDefinition
}

func NewRegistry() *Registry {
	return &Registry{
		byTypeVersion:    make(map[typeVersion]*RegisteredDefinition),
		byAPITypeVersion: make(map[apiTypeVersion]*RegisteredDefinition),
	}
}

func (r *Registry) Register(def *DestinationDefinition) error {
	if def == nil {
		return fmt.Errorf("destination definition is nil")
	}

	if def.APIType == "" {
		def.APIType = def.Type
	}

	if err := validateDefinitionSourceTypes(def); err != nil {
		return err
	}

	key := typeVersion{Type: def.Type, Version: def.Version}
	if _, exists := r.byTypeVersion[key]; exists {
		return fmt.Errorf("destination definition %s version %d already registered", def.Type, def.Version)
	}

	apiKey := apiTypeVersion{APIType: def.APIType, Version: def.Version}
	if existing, exists := r.byAPITypeVersion[apiKey]; exists {
		return fmt.Errorf(
			"apiType %q version %d is already registered as %q",
			def.APIType,
			def.Version,
			existing.Type,
		)
	}

	registered, err := newRegisteredDefinition(def)
	if err != nil {
		return fmt.Errorf("registering destination definition %s version %d: %w", def.Type, def.Version, err)
	}

	r.byTypeVersion[key] = registered
	r.byAPITypeVersion[apiKey] = registered
	return nil
}

func validateDefinitionSourceTypes(def *DestinationDefinition) error {
	if err := common.ValidateSourceTypes(def.SourceTypes); err != nil {
		return fmt.Errorf("validating destination definition source types: %w", err)
	}
	if err := validateConnectionModeSourceTypes(def); err != nil {
		return err
	}
	return validateConsentValidationOverrides(def)
}

func validateConnectionModeSourceTypes(def *DestinationDefinition) error {
	for _, sourceType := range slices.Sorted(maps.Keys(def.ConnectionModes)) {
		if !slices.Contains(def.SourceTypes, sourceType) {
			return fmt.Errorf("connection modes configured for unsupported source type %q", sourceType)
		}
	}
	for _, sourceType := range def.SourceTypes {
		if _, ok := def.ConnectionModes[sourceType]; !ok {
			return fmt.Errorf("source type %q has no connection modes", sourceType)
		}
	}
	return nil
}

func validateConsentValidationOverrides(def *DestinationDefinition) error {
	for _, sourceType := range slices.Sorted(maps.Keys(def.ConsentValidationOverrides)) {
		if !slices.Contains(def.SourceTypes, sourceType) {
			return fmt.Errorf("consent validation override configured for unsupported source type %q", sourceType)
		}
		if def.ConsentValidationOverrides[sourceType] == nil {
			return fmt.Errorf("consent validation override for source type %q is nil", sourceType)
		}
	}
	return nil
}

func (r *Registry) Get(destType string, version int64) (*RegisteredDefinition, error) {
	registered, ok := r.byTypeVersion[typeVersion{Type: destType, Version: version}]
	if !ok {
		return nil, fmt.Errorf("destination definition %s version %d not found", destType, version)
	}
	return registered, nil
}

// GetByAPIType resolves a definition by the upstream API type and version.
func (r *Registry) GetByAPIType(apiType string, version int64) (*RegisteredDefinition, error) {
	registered, ok := r.byAPITypeVersion[apiTypeVersion{APIType: apiType, Version: version}]
	if !ok {
		return nil, fmt.Errorf("destination definition for apiType %s version %d not found", apiType, version)
	}
	return registered, nil
}

func (r *Registry) SupportedTypes() []string {
	types := make([]string, 0, len(r.byTypeVersion))
	seen := make(map[string]struct{}, len(r.byTypeVersion))
	for key := range r.byTypeVersion {
		if _, ok := seen[key.Type]; ok {
			continue
		}
		seen[key.Type] = struct{}{}
		types = append(types, key.Type)
	}
	slices.Sort(types)
	return types
}

func (r *Registry) Versions(destType string) ([]int64, error) {
	versions := make([]int64, 0)
	for key, registered := range r.byTypeVersion {
		if key.Type != destType {
			continue
		}
		versions = append(versions, registered.Version)
	}
	if len(versions) == 0 {
		return nil, fmt.Errorf("destination type %s not found", destType)
	}
	slices.Sort(versions)
	return versions, nil
}

func (r *Registry) IsSupported(destType string) bool {
	for key := range r.byTypeVersion {
		if key.Type == destType {
			return true
		}
	}
	return false
}
