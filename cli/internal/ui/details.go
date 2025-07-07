package ui

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
)

var priorityFields = []string{"id", "name", "createdAt", "updatedAt"}
var notSetLiteral = "- not set -"

// RenderDetails renders a single resource to a string.
// It calculates the required column width based only on the fields within that single resource.
func RenderDetails(resource resources.ResourceData) string {
	processed := preprocessData(resource)
	maxKeyWidth := calculateMaxKeyWidth(processed, 0)
	return renderSingleResource(processed, maxKeyWidth)
}

// RenderDetailsList renders a list of resources to a single, formatted string.
// It calculates the required column width across all items to ensure alignment.
func RenderDetailsList(rs []resources.ResourceData) string {
	if len(rs) == 0 {
		return "No resources found."
	}

	processedResources := make([]map[string]any, len(rs))
	for i, r := range rs {
		processedResources[i] = preprocessData(r)
	}

	globalMaxKeyWidth := 0
	for _, p := range processedResources {
		if width := calculateMaxKeyWidth(p, 0); width > globalMaxKeyWidth {
			globalMaxKeyWidth = width
		}
	}

	var b strings.Builder
	for i, p := range processedResources {
		if i > 0 {
			b.WriteString("\n")
			b.WriteString(Ruler())
			b.WriteString("\n")
		}
		b.WriteString(renderSingleResource(p, globalMaxKeyWidth))
	}
	return b.String()
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
		if _, isMap := value.(map[string]any); !isMap {
			currentLen := len(indent) + len(key)
			if currentLen > maxLen {
				maxLen = currentLen
			}
		}

		if v, ok := value.(map[string]any); ok {
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
