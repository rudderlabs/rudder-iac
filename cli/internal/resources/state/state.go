package state

import (
	"fmt"
	"reflect"

	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

type State struct {
	Resources map[string]*ResourceState `json:"resources"`
}

func EmptyState() *State {
	return &State{
		Resources: make(map[string]*ResourceState),
	}
}

type ResourceState struct {
	ID           string         `json:"id"`
	Type         string         `json:"type"`
	Input        map[string]any `json:"input"`
	Output       map[string]any `json:"output"`
	InputRaw     any            `json:"-"` // Strongly-typed input (e.g., *SourceResource)
	OutputRaw    any            `json:"-"` // Strongly-typed output (e.g., *SourceStateRemote)
	Dependencies []string       `json:"dependencies"`
}

func (sr *ResourceState) Data() resources.ResourceData {
	data := make(resources.ResourceData)
	for k, v := range sr.Input {
		data[k] = v
	}
	for k, v := range sr.Output {
		data[k] = v
	}
	return data
}

func (s *State) AddResource(r *ResourceState) {
	s.Resources[resources.URN(r.ID, r.Type)] = r
}

func (s *State) RemoveResource(urn string) {
	delete(s.Resources, urn)
}

func (s *State) GetResource(urn string) *ResourceState {
	return s.Resources[urn]
}

func (s *State) String() string {
	json, _ := ToJSON(s)
	return string(json)
}

func Dereference(data resources.ResourceData, state *State) (resources.ResourceData, error) {
	dereferenced, err := dereferenceValue(data, state)
	if err != nil {
		return nil, err
	}

	return dereferenced.(resources.ResourceData), nil
}

func dereferenceValue(v any, state *State) (any, error) {
	switch val := v.(type) {
	case resources.PropertyRef:
		resource := state.GetResource(val.URN)
		if resource == nil {
			return nil, fmt.Errorf("referred resource '%s' does not exist", val.URN)
		}

		resourceData := resource.Data()
		if resourceData == nil {
			return nil, nil
		}
		return dereferenceValue(resourceData[val.Property], state)
	case *resources.PropertyRef:
		if val == nil {
			return nil, nil
		}
		return dereferenceValue(*val, state)
	case resources.ResourceData:
		result := make(resources.ResourceData)
		for k, v := range val {
			dereferenced, err := dereferenceValue(v, state)
			if err != nil {
				return nil, err
			}
			result[k] = dereferenced
		}
		return result, nil
	case map[string]any:
		result := make(map[string]any)
		for k, v := range val {
			dereferenced, err := dereferenceValue(v, state)
			if err != nil {
				return nil, err
			}
			result[k] = dereferenced
		}
		return result, nil
	case []any:
		result := make([]any, len(val))
		for i, v := range val {
			dereferenced, err := dereferenceValue(v, state)
			if err != nil {
				return nil, err
			}
			result[i] = dereferenced
		}
		return result, nil

	case []map[string]any:
		result := make([]map[string]any, len(val))
		for i, v := range val {
			deferenced, err := dereferenceValue(v, state)
			if err != nil {
				return nil, err
			}
			result[i] = deferenced.(map[string]any)
		}
		return result, nil

	default:
		return v, nil
	}
}

func DereferenceByReflection(v any, state *State) error {
	return dereferenceByReflectionValue(reflect.ValueOf(v), state)
}

func dereferenceByReflectionValue(val reflect.Value, state *State) error {
	if !val.IsValid() {
		return nil
	}

	switch val.Kind() {
	case reflect.Ptr:
		if val.IsNil() {
			return nil
		}

		// Special handling for *PropertyRef
		if val.Type() == reflect.TypeOf(&resources.PropertyRef{}) {
			propRef := val.Interface().(*resources.PropertyRef)
			if err := resolvePropertyRef(propRef, state); err != nil {
				return err
			}
			return nil
		}

		// For other pointers, dereference and continue
		return dereferenceByReflectionValue(val.Elem(), state)

	case reflect.Struct:
		// Special handling for PropertyRef by value
		if val.Type() == reflect.TypeOf(resources.PropertyRef{}) {
			// PropertyRef by value cannot be modified, skip it
			// (only *PropertyRef can be modified in place)
			return nil
		}

		// Recursively process all struct fields
		for i := 0; i < val.NumField(); i++ {
			field := val.Field(i)

			// Skip unexported fields
			if !field.CanInterface() {
				continue
			}

			if err := dereferenceByReflectionValue(field, state); err != nil {
				return err
			}
		}

	case reflect.Slice, reflect.Array:
		// Recursively process all slice/array elements
		for i := 0; i < val.Len(); i++ {
			if err := dereferenceByReflectionValue(val.Index(i), state); err != nil {
				return err
			}
		}

	case reflect.Map:
		// Recursively process all map values
		for _, key := range val.MapKeys() {
			mapValue := val.MapIndex(key)
			if err := dereferenceByReflectionValue(mapValue, state); err != nil {
				return err
			}
		}
	}

	return nil
}

func resolvePropertyRef(propRef *resources.PropertyRef, state *State) error {
	if propRef == nil {
		return nil
	}

	if propRef.Resolve != nil {
		resource := state.GetResource(propRef.URN)
		if resource == nil {
			return fmt.Errorf("referred resource '%s' does not exist", propRef.URN)
		}

		value, err := propRef.Resolve(resource.OutputRaw)
		if err != nil {
			return fmt.Errorf("resolving property ref for %s: %w", propRef.URN, err)
		}
		propRef.IsResolved = true
		propRef.Value = value
		return nil
	}

	resource := state.GetResource(propRef.URN)
	if resource == nil {
		return fmt.Errorf("referred resource '%s' does not exist", propRef.URN)
	}

	resourceData := resource.Data()
	if resourceData == nil {
		return nil
	}

	value, exists := resourceData[propRef.Property]
	if !exists {
		return fmt.Errorf("property '%s' does not exist in resource '%s'", propRef.Property, propRef.URN)
	}

	// Convert value to string for ResolvedValue
	var stringValue string
	switch v := value.(type) {
	case string:
		stringValue = v
	case fmt.Stringer:
		stringValue = v.String()
	default:
		stringValue = fmt.Sprintf("%v", v)
	}

	// Populate the IsResolved & Value fields in place
	propRef.IsResolved = true
	propRef.Value = stringValue

	return nil
}

// Merge returns a new State, combining the current state with another state.
// If the versions are incompatible it returns an ErrIncompatibleVersion error.
// If there are URNs that exist in both states it returns an ErrURNAlreadyExists error.
func (s *State) Merge(other *State) (*State, error) {
	if other == nil {
		return s, nil
	}

	newState := EmptyState()

	for k, v := range s.Resources {
		newState.Resources[k] = v
	}

	for k, v := range other.Resources {
		if _, exists := s.Resources[k]; exists {
			return nil, &ErrURNAlreadyExists{URN: k}
		}
		newState.Resources[k] = v
	}

	return newState, nil
}
