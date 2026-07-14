package common

import (
	"encoding/json"
	"fmt"
	"maps"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

const consentManagementLocalKey = "consent_management"

// Properties returns ConfigProperty entries for the consent_management block scoped to source types.
func Properties(sourceTypes []string) []converter.ConfigProperty {
	if len(sourceTypes) == 0 {
		return nil
	}

	properties := make([]converter.ConfigProperty, 0, len(sourceTypes))
	for _, sourceType := range sourceTypes {
		localSourceType := camelToSnake(sourceType)
		properties = append(properties, converter.ArrayWithObjects(
			fmt.Sprintf("consentManagement.%s", sourceType),
			fmt.Sprintf("%s.%s", consentManagementLocalKey, localSourceType),
			map[string]any{
				"provider":           "provider",
				"resolutionStrategy": "resolution_strategy",
				"consents": converter.APINestedObject{
					LocalKey:  "consents",
					NestedKey: "consent",
				},
			},
		))
	}

	return properties
}

// SchemaFragment returns the JSON Schema fragment for consent_management scoped to source types.
func SchemaFragment(sourceTypes []string) json.RawMessage {
	properties := make(map[string]any, len(sourceTypes))
	for _, sourceType := range sourceTypes {
		localSourceType := camelToSnake(sourceType)
		properties[localSourceType] = map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"provider": map[string]any{
						"type": "string",
						"enum": []string{"oneTrust", "ketch", "iubenda", "custom"},
					},
					"resolution_strategy": map[string]any{
						"type": "string",
						"enum": []string{"and", "or", ""},
					},
					"consents": map[string]any{
						"type":  "array",
						"items": map[string]any{"type": "string"},
					},
				},
				"required":             []string{"provider", "consents"},
				"additionalProperties": false,
			},
		}
	}

	fragment := map[string]any{
		consentManagementLocalKey: map[string]any{
			"type":                 "object",
			"properties":           properties,
			"additionalProperties": false,
		},
	}

	raw, err := json.Marshal(fragment)
	if err != nil {
		panic(fmt.Sprintf("marshaling consent_management schema fragment: %v", err))
	}
	return json.RawMessage(raw)
}

// MergeSchemas appends fragment properties to the base schema's properties object.
func MergeSchemas(base, fragment json.RawMessage) (json.RawMessage, error) {
	var baseSchema map[string]any
	if err := json.Unmarshal(base, &baseSchema); err != nil {
		return nil, fmt.Errorf("unmarshaling base schema: %w", err)
	}

	var fragmentSchema map[string]any
	if err := json.Unmarshal(fragment, &fragmentSchema); err != nil {
		return nil, fmt.Errorf("unmarshaling fragment schema: %w", err)
	}

	baseProperties, ok := baseSchema["properties"].(map[string]any)
	if !ok {
		baseProperties = map[string]any{}
		baseSchema["properties"] = baseProperties
	}

	maps.Copy(baseProperties, fragmentSchema)

	merged, err := json.Marshal(baseSchema)
	if err != nil {
		return nil, fmt.Errorf("marshaling merged schema: %w", err)
	}
	return json.RawMessage(merged), nil
}

// LocalSourceTypeKey converts an upstream camelCase source type to the
// snake_case key used in local YAML config (e.g. reactNative → react_native).
func LocalSourceTypeKey(sourceType string) string {
	return camelToSnake(sourceType)
}

func camelToSnake(s string) string {
	var res strings.Builder
	for i, v := range s {
		if 'A' <= v && v <= 'Z' {
			if i != 0 {
				res.WriteByte('_')
			}
			res.WriteRune(v + 32)
		} else {
			res.WriteRune(v)
		}
	}
	return res.String()
}
