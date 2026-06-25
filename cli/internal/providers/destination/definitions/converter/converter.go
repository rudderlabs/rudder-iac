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

	result, err := unmarshalConfigMap(apiJSON)
	if err != nil {
		return nil, err
	}

	passthroughUnmappedLocalKeys(result, local, props)
	return result, nil
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

	result, err := unmarshalConfigMap(localJSON)
	if err != nil {
		return nil, err
	}

	passthroughUnmappedAPIKeys(result, api, props)
	return result, nil
}

func passthroughUnmappedLocalKeys(result, local map[string]any, props []ConfigProperty) {
	mapped := mappedTopLevelLocalKeys(props)
	for key, value := range local {
		if _, mappedKey := mapped[key]; mappedKey {
			continue
		}
		if _, exists := result[key]; !exists {
			result[key] = value
		}
	}
}

func passthroughUnmappedAPIKeys(result, api map[string]any, props []ConfigProperty) {
	mapped := mappedTopLevelAPIKeys(props)
	for key, value := range api {
		if _, mappedKey := mapped[key]; mappedKey {
			continue
		}
		if _, exists := result[key]; !exists {
			result[key] = value
		}
	}
}

func mappedTopLevelLocalKeys(props []ConfigProperty) map[string]struct{} {
	keys := make(map[string]struct{})
	for _, property := range props {
		for _, localKey := range property.localKeys {
			keys[topLevelKey(localKey)] = struct{}{}
		}
	}
	return keys
}

func mappedTopLevelAPIKeys(props []ConfigProperty) map[string]struct{} {
	keys := make(map[string]struct{})
	for _, property := range props {
		for _, apiKey := range property.apiKeys {
			keys[topLevelKey(apiKey)] = struct{}{}
		}
	}
	return keys
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
