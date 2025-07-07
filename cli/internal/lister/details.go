package lister

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
)

var priorityFields = []string{"id", "name", "createdAt", "updatedAt"}
var notSetLiteral = "- not set -"

// normalize converts any value into a structure of map[string]any and []any.
func normalize(value any) any {
	if value == nil {
		return nil
	}
	// Use JSON marshaling and unmarshaling to convert structs and other complex types
	// into a standard map[string]any format. This is more robust than reflection.
	data, err := json.Marshal(value)
	if err != nil {
		return value // Fallback for types that can't be marshaled
	}
	var result any
	if err := json.Unmarshal(data, &result); err != nil {
		return value // Fallback if unmarshaling fails
	}
	return result
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

func printListWithDetails(rs []resources.ResourceData) error {
	// Preprocess all resources first
	processedResources := make([]map[string]any, len(rs))
	for i, r := range rs {
		processedResources[i] = preprocessData(r)
	}

	// Calculate the max key width across all processed resources
	globalMaxKeyWidth := 0
	for _, p := range processedResources {
		if width := calculateMaxKeyWidth(p, 0); width > globalMaxKeyWidth {
			globalMaxKeyWidth = width
		}
	}

	// Now print each resource using the new Ruler function
	for i, p := range processedResources {
		if i > 0 {
			ui.Ruler()
			fmt.Println()
		}
		printResourceDetails(p, globalMaxKeyWidth)
		fmt.Println()
	}
	return nil
}

func printResourceDetails(data map[string]any, maxKeyWidth int) {
	printedKeys := make(map[string]bool)

	// Add a check for nil values in priority fields and print "not set" if they are missing
	for _, key := range priorityFields {
		if value, ok := data[key]; ok {
			printField(key, value, 0, maxKeyWidth)
		} else {
			// The key is a priority field but is missing from the data, so it's not set.
			printField(key, nil, 0, maxKeyWidth)
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
		printField(key, data[key], 0, maxKeyWidth)
	}
}

func calculateMaxKeyWidth(data map[string]any, indentLevel int) int {
	maxLen := 0
	indent := strings.Repeat("  ", indentLevel)

	// Also consider priority fields that might be nil and not in the data map
	for _, key := range priorityFields {
		currentLen := len(indent) + len(key)
		if currentLen > maxLen {
			maxLen = currentLen
		}
	}

	for key, value := range data {
		// Only consider fields that print a value on the same line for width calculation
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

func printField(key string, value any, indentLevel int, maxKeyWidth int) {
	indent := strings.Repeat("  ", indentLevel)
	keyStr := ui.Bold(key)

	if v, ok := value.(map[string]any); ok {
		fmt.Printf("%s%s:\n", indent, keyStr)
		var subKeys []string
		for subKey := range v {
			subKeys = append(subKeys, subKey)
		}
		sort.Strings(subKeys)
		for _, subKey := range subKeys {
			printField(subKey, v[subKey], indentLevel+1, maxKeyWidth)
		}
	} else {
		currentKeyWidth := len(indent) + len(key)
		padding := maxKeyWidth - currentKeyWidth
		if padding < 0 {
			padding = 0
		}

		var valueStr string
		if value == nil {
			valueStr = ui.GreyedOut(notSetLiteral)
		} else {
			valueStr = fmt.Sprintf("%v", value)
		}

		if key == "id" {
			valueStr = ui.Color(valueStr, ui.Yellow)
		}

		fmt.Printf("%s%s: %s%s\n", indent, keyStr, strings.Repeat(" ", padding), valueStr)
	}
}
