package converter

import (
	"crypto/md5"
	"fmt"
	"sort"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/experimental/schema/models"
)

// SchemaAnalyzer analyzes schemas to extract events, properties, and custom types
type SchemaAnalyzer struct {
	Events      map[string]*EventInfo
	Properties  map[string]*PropertyInfo
	CustomTypes map[string]*CustomTypeInfo
}

// EventInfo holds information about an extracted event
type EventInfo struct {
	ID          string
	Name        string
	EventType   string
	Description string
	Original    models.Schema
}

// PropertyInfo holds information about an extracted property
type PropertyInfo struct {
	ID          string
	Name        string
	Type        string
	Description string
	Path        string
	JsonType    string
}

// CustomTypeInfo holds information about a custom type
type CustomTypeInfo struct {
	ID            string
	Name          string
	Type          string
	Description   string
	Structure     map[string]string
	ArrayItemType string
	Hash          string
}

// NewSchemaAnalyzer creates a new schema analyzer
func NewSchemaAnalyzer() *SchemaAnalyzer {
	return &SchemaAnalyzer{
		Events:      make(map[string]*EventInfo),
		Properties:  make(map[string]*PropertyInfo),
		CustomTypes: make(map[string]*CustomTypeInfo),
	}
}

// AnalyzeSchemas processes all schemas to extract events, properties, and custom types
func (sa *SchemaAnalyzer) AnalyzeSchemas(schemas []models.Schema) error {
	// First pass: Extract unique events
	for _, schema := range schemas {
		sa.extractEvent(schema)
	}

	// Second pass: Analyze properties and identify custom types
	for _, schema := range schemas {
		err := sa.analyzeProperties(schema.Schema, "")
		if err != nil {
			return fmt.Errorf("failed to analyze properties for schema %s: %w", schema.UID, err)
		}
	}

	return nil
}

// extractEvent extracts event information from a schema
func (sa *SchemaAnalyzer) extractEvent(schema models.Schema) {
	eventID := sanitizeID(schema.EventIdentifier)

	// Skip if already processed
	if _, exists := sa.Events[eventID]; exists {
		return
	}

	sa.Events[eventID] = &EventInfo{
		ID:          eventID,
		Name:        generateEventName(schema.EventIdentifier),
		EventType:   schema.EventType,
		Description: fmt.Sprintf("Extracted from eventIdentifier: %s", schema.EventIdentifier),
		Original:    schema,
	}
}

// analyzeProperties recursively analyzes schema properties
func (sa *SchemaAnalyzer) analyzeProperties(obj interface{}, path string) error {
	switch v := obj.(type) {
	case map[string]interface{}:
		return sa.analyzeObject(v, path)
	case []interface{}:
		return sa.analyzeArray(v, path)
	default:
		// Primitive type - create property
		return sa.createProperty(path, v)
	}
}

// analyzeObject analyzes an object and potentially creates custom types
func (sa *SchemaAnalyzer) analyzeObject(obj map[string]interface{}, path string) error {
	// Create properties for each field
	for key, value := range obj {
		fieldPath := buildPath(path, key)

		switch v := value.(type) {
		case map[string]interface{}:
			// Nested object - create custom type
			customType, err := sa.createCustomTypeForObject(v, fieldPath)
			if err != nil {
				return err
			}

			// Create property referencing the custom type
			err = sa.createPropertyWithCustomType(fieldPath, customType)
			if err != nil {
				return err
			}

			// Recursively analyze the nested object
			err = sa.analyzeObject(v, fieldPath)
			if err != nil {
				return err
			}

		case []interface{}:
			// Array - create custom array type
			customType, err := sa.createCustomTypeForArray(v, fieldPath)
			if err != nil {
				return err
			}

			// Create property referencing the custom type
			err = sa.createPropertyWithCustomType(fieldPath, customType)
			if err != nil {
				return err
			}

			// Recursively analyze array items
			err = sa.analyzeArray(v, fieldPath)
			if err != nil {
				return err
			}

		default:
			// Primitive type
			err := sa.createProperty(fieldPath, v)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// analyzeArray analyzes an array and its items
func (sa *SchemaAnalyzer) analyzeArray(arr []interface{}, path string) error {
	if len(arr) == 0 {
		return nil
	}

	// Analyze first item to determine array type
	firstItem := arr[0]

	switch v := firstItem.(type) {
	case map[string]interface{}:
		// Array of objects - analyze the object structure
		return sa.analyzeObject(v, path+"_item")
	case []interface{}:
		// Nested array - analyze recursively
		return sa.analyzeArray(v, path+"_item")
	default:
		// Array of primitives - no further analysis needed
		return nil
	}
}

// createCustomTypeForObject creates a custom type for an object
func (sa *SchemaAnalyzer) createCustomTypeForObject(obj map[string]interface{}, path string) (*CustomTypeInfo, error) {
	structure := make(map[string]string)

	// Analyze object structure
	for key, value := range obj {
		jsonType := getJSONType(value)
		structure[key] = jsonType
	}

	// Generate hash for uniqueness
	hash := generateStructureHash(structure)

	// Check if custom type already exists
	for _, customType := range sa.CustomTypes {
		if customType.Hash == hash && customType.Type == "object" {
			return customType, nil
		}
	}

	// Create new custom type
	customTypeID := generateCustomTypeID(path, "object", hash)
	customType := &CustomTypeInfo{
		ID:          customTypeID,
		Name:        generateCustomTypeName(path, "object"),
		Type:        "object",
		Description: fmt.Sprintf("Custom object type for %s", path),
		Structure:   structure,
		Hash:        hash,
	}

	sa.CustomTypes[customTypeID] = customType
	return customType, nil
}

// createCustomTypeForArray creates a custom type for an array
func (sa *SchemaAnalyzer) createCustomTypeForArray(arr []interface{}, path string) (*CustomTypeInfo, error) {
	if len(arr) == 0 {
		// Empty array - default to string array
		itemType := "string"
		hash := generateArrayHash(itemType)

		customTypeID := generateCustomTypeID(path, "array", hash)
		customType := &CustomTypeInfo{
			ID:            customTypeID,
			Name:          generateCustomTypeName(path, "array"),
			Type:          "array",
			Description:   fmt.Sprintf("Custom array type for %s", path),
			ArrayItemType: itemType,
			Hash:          hash,
		}

		sa.CustomTypes[customTypeID] = customType
		return customType, nil
	}

	// Determine array item type from first element
	firstItem := arr[0]
	itemType := getJSONType(firstItem)

	// For object arrays, we need to analyze the object structure
	if itemType == "object" {
		if objMap, ok := firstItem.(map[string]interface{}); ok {
			structure := make(map[string]string)
			for key, value := range objMap {
				structure[key] = getJSONType(value)
			}
			itemType = fmt.Sprintf("object_%s", generateStructureHash(structure))
		}
	}

	hash := generateArrayHash(itemType)

	// Check if custom type already exists
	for _, customType := range sa.CustomTypes {
		if customType.Hash == hash && customType.Type == "array" {
			return customType, nil
		}
	}

	// Create new custom type
	customTypeID := generateCustomTypeID(path, "array", hash)
	customType := &CustomTypeInfo{
		ID:            customTypeID,
		Name:          generateCustomTypeName(path, "array"),
		Type:          "array",
		Description:   fmt.Sprintf("Custom array type for %s", path),
		ArrayItemType: itemType,
		Hash:          hash,
	}

	sa.CustomTypes[customTypeID] = customType
	return customType, nil
}

// createProperty creates a property for a primitive type
func (sa *SchemaAnalyzer) createProperty(path string, value interface{}) error {
	if path == "" {
		return nil // Skip root level
	}

	jsonType := getJSONType(value)
	yamlType := mapJSONTypeToYAML(jsonType)

	propertyKey := generatePropertyKey(path, yamlType)

	// Skip if property already exists
	if _, exists := sa.Properties[propertyKey]; exists {
		return nil
	}

	propertyID := sanitizeID(path)
	property := &PropertyInfo{
		ID:          propertyID,
		Name:        extractPropertyName(path),
		Type:        yamlType,
		Description: fmt.Sprintf("Property extracted from path: %s", path),
		Path:        path,
		JsonType:    jsonType,
	}

	sa.Properties[propertyKey] = property
	return nil
}

// createPropertyWithCustomType creates a property that references a custom type
func (sa *SchemaAnalyzer) createPropertyWithCustomType(path string, customType *CustomTypeInfo) error {
	if path == "" {
		return nil
	}

	typeRef := fmt.Sprintf("#/custom-types/extracted_custom_types/%s", customType.ID)
	propertyKey := generatePropertyKey(path, typeRef)

	// Skip if property already exists
	if _, exists := sa.Properties[propertyKey]; exists {
		return nil
	}

	propertyID := sanitizeID(path)
	property := &PropertyInfo{
		ID:          propertyID,
		Name:        extractPropertyName(path),
		Type:        typeRef,
		Description: fmt.Sprintf("Property with custom type from path: %s", path),
		Path:        path,
		JsonType:    customType.Type,
	}

	sa.Properties[propertyKey] = property
	return nil
}

// Helper functions

func sanitizeID(input string) string {
	// Replace non-alphanumeric characters with underscores
	result := strings.ReplaceAll(input, ".", "_")
	result = strings.ReplaceAll(result, "-", "_")
	result = strings.ReplaceAll(result, " ", "_")
	result = strings.ToLower(result)
	return result
}

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

func buildPath(basePath, key string) string {
	if basePath == "" {
		return key
	}
	return basePath + "." + key
}

func extractPropertyName(path string) string {
	parts := strings.Split(path, ".")
	return parts[len(parts)-1]
}

func getJSONType(value interface{}) string {
	if value == nil {
		return "null"
	}

	switch value.(type) {
	case string:
		return "string"
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
		// For type strings like "string", "float64", etc.
		if str, ok := value.(string); ok {
			switch str {
			case "float64":
				return "number"
			case "bool":
				return "boolean"
			default:
				return "string"
			}
		}
		return "string"
	}
}

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

func generatePropertyKey(path, propType string) string {
	return fmt.Sprintf("%s_%s", sanitizeID(path), sanitizeID(propType))
}

func generateCustomTypeID(path, baseType, hash string) string {
	cleanPath := sanitizeID(path)
	return fmt.Sprintf("%s_%s_%s", cleanPath, baseType, hash[:8])
}

func generateCustomTypeName(path, baseType string) string {
	cleanPath := strings.ReplaceAll(path, ".", " ")
	cleanPath = strings.ReplaceAll(cleanPath, "_", " ")
	parts := strings.Fields(cleanPath)

	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}

	typeName := strings.Join(parts, " ")
	if baseType == "array" {
		return typeName + " Array Type"
	}
	return typeName + " Type"
}

func generateStructureHash(structure map[string]string) string {
	// Create a consistent hash based on structure
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
	hash := md5.Sum([]byte(content))
	return fmt.Sprintf("%x", hash)[:8]
}

func generateArrayHash(itemType string) string {
	hash := md5.Sum([]byte("array_" + itemType))
	return fmt.Sprintf("%x", hash)[:8]
}
