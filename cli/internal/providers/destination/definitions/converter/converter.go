package converter

import (
	"encoding/json"
	"fmt"
)

// LocalToAPI converts a snake_case local config map to camelCase API config
// by running each ConfigProperty's FromLocalFunc in order.
func LocalToAPI(props []ConfigProperty, local map[string]any) (map[string]any, error) {
	localJSON, err := json.Marshal(local)
	if err != nil {
		return nil, fmt.Errorf("marshaling local config: %w", err)
	}

	apiJSON := "{}"
	for _, p := range props {
		r, err := p.FromLocalFunc(apiJSON, string(localJSON))
		if err != nil {
			return nil, err
		}
		apiJSON = r
	}

	return unmarshalConfigMap(apiJSON)
}

// APIToLocal converts a camelCase API config map to snake_case local config
// by running each ConfigProperty's ToLocalFunc in order.
func APIToLocal(props []ConfigProperty, api map[string]any) (map[string]any, error) {
	apiJSON, err := json.Marshal(api)
	if err != nil {
		return nil, fmt.Errorf("marshaling api config: %w", err)
	}

	localJSON := "{}"
	for _, p := range props {
		r, err := p.ToLocalFunc(localJSON, string(apiJSON))
		if err != nil {
			return nil, err
		}
		localJSON = r
	}

	return unmarshalConfigMap(localJSON)
}

func unmarshalConfigMap(jsonStr string) (map[string]any, error) {
	var result map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("unmarshaling config: %w", err)
	}
	if result == nil {
		result = map[string]any{}
	}
	return result, nil
}
