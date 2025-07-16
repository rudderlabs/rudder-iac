package ui

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

var priorityFields = []string{"id", "name", "createdAt", "updatedAt"}
var notSetLiteral = "- not set -"

// FormattedMap formats a single map[string]any resource into a structured, human-readable string.
//
// Formatting applied:
// - Priority fields (id, name, createdAt, updatedAt) are displayed first
// - Remaining fields are sorted alphabetically
// - Column alignment with padding for clean layout
// - Color coding: IDs in yellow, unset values in grey
// - Nested objects are indented with proper hierarchy
// - Column width is calculated based on the longest key in the resource
//
// Designed for displaying detailed information of a single resource in CLI output.
func FormattedMap(resource map[string]any) string {
	processed := preprocessData(resource)
	maxKeyWidth := calculateMaxKeyWidth(processed, 0)
	return renderSingleResource(processed, maxKeyWidth)
}

// renderSingleResource renders a pre-processed resource map into a string.
func renderSingleResource(data map[string]any, maxKeyWidth int) string {
	var b strings.Builder
	printedKeys := make(map[string]bool)

	for _, key := range priorityFields {
		if value, ok := data[key]; ok {
			b.WriteString(renderField(key, value, 0, maxKeyWidth))
		} else {
			b.WriteString(renderField(key, nil, 0, maxKeyWidth))
		}
		printedKeys[key] = true
	}

	var otherKeys []string
	for key := range data {
		if !printedKeys[key] {
			otherKeys = append(otherKeys, key)
		}
	}
	sort.Strings(otherKeys)

	for _, key := range otherKeys {
		b.WriteString(renderField(key, data[key], 0, maxKeyWidth))
	}

	return b.String()
}

// renderField renders a single key-value pair to a string.
func renderField(key string, value any, indentLevel int, maxKeyWidth int) string {
	var b strings.Builder
	indent := strings.Repeat("  ", indentLevel)
	keyStr := Bold(key)

	if v, ok := value.(map[string]any); ok {
		b.WriteString(fmt.Sprintf("%s%s:\n", indent, keyStr))
		var subKeys []string
		for subKey := range v {
			subKeys = append(subKeys, subKey)
		}
		sort.Strings(subKeys)
		for _, subKey := range subKeys {
			b.WriteString(renderField(subKey, v[subKey], indentLevel+1, maxKeyWidth))
		}
	} else {
		currentKeyWidth := len(indent) + len(key)
		padding := maxKeyWidth - currentKeyWidth
		if padding < 0 {
			padding = 0
		}

		var valueStr string
		if value == nil {
			valueStr = GreyedOut(notSetLiteral)
		} else {
			valueStr = fmt.Sprintf("%v", value)
		}

		if key == "id" {
			valueStr = Color(valueStr, Yellow)
		}

		b.WriteString(fmt.Sprintf("%s%s: %s%s\n", indent, keyStr, strings.Repeat(" ", padding), valueStr))
	}
	return b.String()
}

// calculateMaxKeyWidth calculates the max key width for a given map.
func calculateMaxKeyWidth(data map[string]any, indentLevel int) int {
	maxLen := 0
	indent := strings.Repeat("  ", indentLevel)

	for _, key := range priorityFields {
		currentLen := len(indent) + len(key)
		if currentLen > maxLen {
			maxLen = currentLen
		}
	}

	for key, value := range data {
		if v, isMap := value.(map[string]any); !isMap {
			currentLen := len(indent) + len(key)
			if currentLen > maxLen {
				maxLen = currentLen
			}
		} else {
			if subLen := calculateMaxKeyWidth(v, indentLevel+1); subLen > maxLen {
				maxLen = subLen
			}
		}
	}
	return maxLen
}

// preprocessData recursively prepares data for printing.
func preprocessData(data map[string]any) map[string]any {
	processed := make(map[string]any, len(data))
	for key, value := range data {
		if value == nil || value == "" {
			processed[key] = nil
			continue
		}
		normalizedValue := normalize(value)
		if asMap, ok := normalizedValue.(map[string]any); ok {
			processed[key] = preprocessData(asMap)
		} else {
			processed[key] = normalizedValue
		}
	}
	return processed
}

// normalize converts any value into a structure of map[string]any and []any.
func normalize(value any) any {
	if value == nil {
		return nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		return value
	}
	var result any
	if err := json.Unmarshal(data, &result); err != nil {
		return value
	}
	return result
}
