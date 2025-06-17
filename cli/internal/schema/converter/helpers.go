package converter

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
)

// buildPath constructs a hierarchical path from base path and key
func buildPath(basePath, key string) string {
	if basePath == "" {
		return key
	}
	return basePath + "." + key
}

// extractPropertyName extracts the last component from a dot-separated path
func extractPropertyName(path string) string {
	parts := strings.Split(path, ".")
	return parts[len(parts)-1]
}

// getJSONType determines the JSON type of a value
func getJSONType(value interface{}) string {
	if value == nil {
		return "null"
	}

	switch v := value.(type) {
	case string:
		// Check for type hints first
		switch v {
		case "float64":
			return "number"
		case "bool":
			return "boolean"
		default:
			return "string"
		}
	case float64:
		return "number"
	case int, int64:
		return "integer"
	case bool:
		return "boolean"
	case map[string]interface{}:
		return "object"
	case []interface{}:
		return "array"
	default:
		return "string"
	}
}

// mapJSONTypeToYAML maps JSON types to their YAML equivalents
func mapJSONTypeToYAML(jsonType string) string {
	switch jsonType {
	case "number":
		return "number"
	case "integer":
		return "integer"
	case "boolean":
		return "boolean"
	case "object":
		return "object"
	case "array":
		return "array"
	case "null":
		return "null"
	default:
		return "string"
	}
}

// generateEventName converts snake_case identifier to Title Case
func generateEventName(eventIdentifier string) string {
	// Convert snake_case to Title Case
	parts := strings.Split(eventIdentifier, "_")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, " ")
}

// generateRandomID creates a random alphanumeric identifier
func generateRandomID() string {
	const length = 10
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}
	return string(result)
}

// sanitizeID sanitizes an identifier using basic mode
func sanitizeID(input string) string {
	stringUtils := &StringUtils{}
	return stringUtils.sanitize(input, SanitizationModeBasic)
}

// sanitizeEventID sanitizes an event identifier using event mode
func sanitizeEventID(input string) string {
	stringUtils := &StringUtils{}
	return stringUtils.sanitize(input, SanitizationModeEvent)
}

// generateStructureHash creates a hash for an object structure
func generateStructureHash(structure map[string]string) string {
	stringUtils := &StringUtils{}
	keys := make([]string, 0, len(structure))
	for key := range structure {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var parts []string
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s:%s", key, structure[key]))
	}

	content := strings.Join(parts, "|")
	return stringUtils.generateHash(content, "")
}

// generateArrayHash creates a hash for an array type
func generateArrayHash(itemType string) string {
	stringUtils := &StringUtils{}
	return stringUtils.generateHash(itemType, "array")
}
