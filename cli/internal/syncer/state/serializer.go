package state

import (
	"encoding/json"

	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
)

const RudderRef = "$__rudderRef"

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
				RudderRef:  val.URN,
				"property": val.Property,
			}

		case []map[string]interface{}:
			newArray := make([]map[string]interface{}, len(val))
			for i, item := range val {
				newArray[i] = encodeReferences(item)
			}
			result[k] = newArray

		case map[string]interface{}:
			result[k] = encodeReferences(val)

		case []interface{}:
			newArray := make([]interface{}, len(val))
			for i, item := range val {

				if m, ok := item.(resources.PropertyRef); ok {
					newArray[i] = map[string]interface{}{
						RudderRef:  m.URN,
						"property": m.Property,
					}
					continue
				}
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
					URN:      val[RudderRef].(string),
					Property: val["property"].(string),
				}
			} else {
				result[k] = decodeReferences(val)
			}

		case []map[string]interface{}:
			newMap := make([]map[string]interface{}, len(val))
			for i, item := range val {
				newMap[i] = decodeReferences(item)
			}

			result[k] = newMap

		case []interface{}:
			newArray := make([]interface{}, len(val))
			for i, item := range val {
				if m, ok := item.(map[string]interface{}); ok {
					if isReference(m) {
						newArray[i] = resources.PropertyRef{
							URN:      m[RudderRef].(string),
							Property: m["property"].(string),
						}
					} else {
						newArray[i] = decodeReferences(m)
					}
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
	_, hasRef := m[RudderRef]
	_, hasProperty := m["property"]

	return hasRef && hasProperty
}
