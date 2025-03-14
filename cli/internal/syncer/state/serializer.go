package state

import (
	"encoding/json"

	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
)

const LatestVersion = "1.0.0"

func ToJSON(state *State) (json.RawMessage, error) {
	// Create a copy of state to avoid modifying the original
	stateCopy := &State{
		Version:   state.Version,
		Resources: make(map[string]*ResourceState),
	}

	if stateCopy.Version == "" {
		stateCopy.Version = LatestVersion
	}

	for urn, res := range state.Resources {
		stateCopy.Resources[urn] = EncodeResourceState(res)
	}

	return json.Marshal(stateCopy)
}

func EncodeResourceState(state *ResourceState) *ResourceState {
	return &ResourceState{
		ID:           state.ID,
		Type:         state.Type,
		Input:        encodeReferences(state.Input),
		Output:       encodeReferences(state.Output),
		Dependencies: state.Dependencies,
	}
}

func encodeReferences(data map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	for k, v := range data {
		switch val := v.(type) {
		case resources.PropertyRef:
			result[k] = map[string]interface{}{
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

func DecodeResourceState(res *ResourceState) *ResourceState {
	return &ResourceState{
		ID:           res.ID,
		Type:         res.Type,
		Input:        decodeReferences(res.Input),
		Output:       decodeReferences(res.Output),
		Dependencies: res.Dependencies,
	}
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
