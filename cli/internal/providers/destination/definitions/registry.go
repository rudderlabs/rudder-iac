package definitions

import (
	"fmt"
	"slices"
)

type definitionKey struct {
	Type    string
	Version int64
}

// Registry stores registered destination definitions keyed by (Type, Version).
type Registry struct {
	definitions map[definitionKey]*RegisteredDefinition
}

func NewRegistry() *Registry {
	return &Registry{
		definitions: make(map[definitionKey]*RegisteredDefinition),
	}
}

func (r *Registry) Register(def *DestinationDefinition) error {
	if def == nil {
		return fmt.Errorf("destination definition is nil")
	}

	key := definitionKey{Type: def.Type, Version: def.Version}
	if _, exists := r.definitions[key]; exists {
		return fmt.Errorf("destination definition %s version %d already registered", def.Type, def.Version)
	}

	registered, err := newRegisteredDefinition(def)
	if err != nil {
		return fmt.Errorf("registering destination definition %s version %d: %w", def.Type, def.Version, err)
	}

	r.definitions[key] = registered
	return nil
}

func (r *Registry) Get(destType string, version int64) (*RegisteredDefinition, error) {
	registered, ok := r.definitions[definitionKey{Type: destType, Version: version}]
	if !ok {
		return nil, fmt.Errorf("destination definition %s version %d not found", destType, version)
	}
	return registered, nil
}

func (r *Registry) SupportedTypes() []string {
	types := make([]string, 0, len(r.definitions))
	seen := make(map[string]struct{}, len(r.definitions))
	for key := range r.definitions {
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
	for key, registered := range r.definitions {
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
	for key := range r.definitions {
		if key.Type == destType {
			return true
		}
	}
	return false
}
