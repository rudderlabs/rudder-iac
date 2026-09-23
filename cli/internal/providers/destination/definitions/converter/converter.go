package converter

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
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

	localJSON, err = applySelectors(localJSON, api, props)
	if err != nil {
		return nil, err
	}

	return unmarshalConfigMap(localJSON)
}

// applySelectors makes each selector's local keys agree with the API
// discriminator: keys it does not point at are removed, and the key it does
// point at is materialized when the API carried no value for it — naming a list
// while storing none means that list is in force and empty. See Discriminator.
//
// It runs after the whole pipeline so it does not depend on where the
// selector's properties sit in the list.
func applySelectors(localJSON string, apiConfig map[string]any, props []ConfigProperty) (string, error) {
	for _, p := range props {
		if p.Selector == nil {
			continue
		}
		selected, present := p.Selector.LocalKeyFor(apiConfig)
		if !present {
			continue
		}

		for localKey := range p.Selector.LocalKeys {
			if localKey == selected || !gjson.Get(localJSON, localKey).Exists() {
				continue
			}

			pruned, err := sjson.Delete(localJSON, localKey)
			if err != nil {
				return localJSON, fmt.Errorf("dropping unselected config key %q: %w", localKey, err)
			}
			localJSON = pruned

			// A group emptied out entirely would otherwise linger as a bare
			// `event_filtering: {}`, reading as configuration nobody wrote.
			lastDot := strings.LastIndex(localKey, ".")
			if lastDot < 0 {
				continue
			}
			parentKey := localKey[:lastDot]
			if parent := gjson.Get(localJSON, parentKey); parent.IsObject() && len(parent.Map()) == 0 {
				pruned, err := sjson.Delete(localJSON, parentKey)
				if err != nil {
					return localJSON, fmt.Errorf("dropping emptied config block %q: %w", parentKey, err)
				}
				localJSON = pruned
			}
		}

		// Every selector in use governs lists, so an in-force key the API left
		// out is an empty one.
		if selected == "" || gjson.Get(localJSON, selected).Exists() {
			continue
		}
		materialized, err := sjson.SetRaw(localJSON, selected, "[]")
		if err != nil {
			return localJSON, fmt.Errorf("materializing selected config key %q: %w", selected, err)
		}
		localJSON = materialized
	}

	return localJSON, nil
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
