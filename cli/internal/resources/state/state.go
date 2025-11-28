package state

import (
	"fmt"

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
