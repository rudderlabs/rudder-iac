package provider

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
)

func deepcopyresource(input resources.ResourceData) resources.ResourceData {
	deepcopy := make(resources.ResourceData)

	for k, v := range input {
		switch value := v.(type) {
		case map[string]interface{}:
			deepcopy[k] = deepcopymap(value)
		case []interface{}:
			deepcopy[k] = deepcopyslice(value)
		default:
			deepcopy[k] = v
		}
	}

	return deepcopy
}

func deepcopymap(input map[string]interface{}) map[string]interface{} {
	output := make(map[string]interface{})
	for k, v := range input {
		switch value := v.(type) {
		case map[string]interface{}:
			output[k] = deepcopymap(value)
		case []interface{}:
			output[k] = deepcopyslice(value)
		default:
			output[k] = v
		}
	}
	return output
}

func deepcopyslice(input []interface{}) []interface{} {
	output := make([]interface{}, len(input))
	for i, v := range input {
		switch value := v.(type) {
		case map[string]interface{}:
			output[i] = deepcopymap(value)
		case []interface{}:
			output[i] = deepcopyslice(value)
		default:
			output[i] = v
		}
	}
	return output
}
