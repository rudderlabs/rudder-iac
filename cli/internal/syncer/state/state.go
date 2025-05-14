package state

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
)

type State struct {
	Version   string                    `json:"version"`
	Resources map[string]*ResourceState `json:"resources"`
}

var ErrIncompatibleState = fmt.Errorf("incompatible state version")

func EmptyState() *State {
	return &State{
		Resources: make(map[string]*ResourceState),
	}
}

type ResourceState struct {
	ID           string                 `json:"id"`
	Type         string                 `json:"type"`
	Input        map[string]interface{} `json:"input"`
	Output       map[string]interface{} `json:"output"`
	Dependencies []string               `json:"dependencies"`
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

func dereferenceValue(v interface{}, state *State) (interface{}, error) {
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
	case map[string]interface{}:
		result := make(map[string]interface{})
		for k, v := range val {
			dereferenced, err := dereferenceValue(v, state)
			if err != nil {
				return nil, err
			}
			result[k] = dereferenced
		}
		return result, nil
	case []interface{}:
		result := make([]interface{}, len(val))
		for i, v := range val {
			dereferenced, err := dereferenceValue(v, state)
			if err != nil {
				return nil, err
			}
			result[i] = dereferenced
		}
		return result, nil

	case []map[string]interface{}:
		result := make([]map[string]interface{}, len(val))
		for i, v := range val {
			deferenced, err := dereferenceValue(v, state)
			if err != nil {
				return nil, err
			}
			result[i] = deferenced.(map[string]interface{})
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
	newState := EmptyState()
	newState.Version = s.Version

	for k, v := range s.Resources {
		newState.Resources[k] = v
	}

	if s.Version != other.Version {
		return nil, &ErrIncompatibleVersion{Version: other.Version}
	}

	for k, v := range other.Resources {
		if _, exists := s.Resources[k]; exists {
			return nil, &ErrURNAlreadyExists{URN: k}
		}
		newState.Resources[k] = v
	}

	return newState, nil
}
