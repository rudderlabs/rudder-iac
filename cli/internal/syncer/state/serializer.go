package state

import (
	"encoding/json"

	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
)

func ToJSON(state *State) (json.RawMessage, error) {
	// Create a copy of state to avoid modifying the original
	stateCopy := &State{
		Resources: make(map[string]*StateResource),
	}

	for urn, res := range state.Resources {
		stateCopy.Resources[urn] = &StateResource{
			ID:           res.ID,
			Type:         res.Type,
			Input:        encodeReferences(res.Input),
			Output:       encodeReferences(res.Output),
			Dependencies: res.Dependencies,
		}
	}

	return json.Marshal(stateCopy)
}

func FromJSON(data json.RawMessage) (*State, error) {
	state := &State{}
	if err := json.Unmarshal(data, state); err != nil {
		return nil, err
	}

	// Decode references in a new state copy
	decodedState := &State{
		Resources: make(map[string]*StateResource),
	}

	for urn, res := range state.Resources {
		decodedState.Resources[urn] = &StateResource{
			ID:           res.ID,
			Type:         res.Type,
			Input:        decodeReferences(res.Input),
			Output:       decodeReferences(res.Output),
			Dependencies: res.Dependencies,
		}
	}

	return decodedState, nil
}

func encodeReferences(data map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	for k, v := range data {
		switch val := v.(type) {
		case resources.PropertyRef:
			result[k] = map[string]string{
				"$ref":     val.URN,
				"property": val.Property,
			}
		case map[string]interface{}:
			result[k] = encodeReferences(val)
		case []interface{}:
			newArray := make([]interface{}, len(val))
			for i, item := range val {
				if m, ok := item.(map[string]interface{}); ok {
					newArray[i] = encodeReferences(m)
				} else {
					newArray[i] = item
				}
			}
			result[k] = newArray
		default:
			result[k] = v
		}
	}

	return result
}

func decodeReferences(data map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	for k, v := range data {
		switch val := v.(type) {
		case map[string]interface{}:
			if isReference(val) {
				result[k] = resources.PropertyRef{
					URN:      val["$ref"].(string),
					Property: val["property"].(string),
				}
			} else {
				result[k] = decodeReferences(val)
			}
		case []interface{}:
			newArray := make([]interface{}, len(val))
			for i, item := range val {
				if m, ok := item.(map[string]interface{}); ok {
					newArray[i] = decodeReferences(m)
				} else {
					newArray[i] = item
				}
			}
			result[k] = newArray
		default:
			result[k] = v
		}
	}

	return result
}

func isReference(v interface{}) bool {
	m, ok := v.(map[string]interface{})
	if !ok {
		return false
	}
	_, hasRef := m["$ref"]
	return hasRef
}
