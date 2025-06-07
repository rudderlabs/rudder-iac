package converter

import (
	"crypto/md5"
	"fmt"
	"math/rand"
	"sort"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/schema/models"
)

// SchemaAnalyzer analyzes schemas to extract events, properties, and custom types
type SchemaAnalyzer struct {
	Events      map[string]*EventInfo
	Properties  map[string]*PropertyInfo
	CustomTypes map[string]*CustomTypeInfo

	// Uniqueness tracking
	UsedCustomTypeNames map[string]bool
	UsedPropertyIDs     map[string]bool
	UsedEventIDs        map[string]bool
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
		Events:              make(map[string]*EventInfo),
		Properties:          make(map[string]*PropertyInfo),
		CustomTypes:         make(map[string]*CustomTypeInfo),
		UsedCustomTypeNames: make(map[string]bool),
		UsedPropertyIDs:     make(map[string]bool),
		UsedEventIDs:        make(map[string]bool),
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
	eventIdentifier := schema.EventIdentifier

	// Handle empty or invalid event identifiers
	if eventIdentifier == "" {
		eventIdentifier = "unknown_event"
	}

	// Sanitize the event identifier to ensure it's valid
	eventID := sanitizeEventID(eventIdentifier)

	// If sanitization results in empty or very short ID, use fallback
	if len(eventID) < 3 {
		eventID = fmt.Sprintf("event_%s", generateRandomID())
	}

	// Check for uniqueness
	if _, exists := sa.Events[eventID]; exists {
		return // Event already processed
	}

	eventName := generateEventName(eventIdentifier)

	event := &EventInfo{
		ID:          eventID,
		Name:        eventName,
		EventType:   "track", // Default type
		Description: fmt.Sprintf("Extracted from eventIdentifier: %s", eventIdentifier),
		Original:    schema,
	}

	sa.Events[eventID] = event
	sa.UsedEventIDs[eventID] = true
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

	// Create new custom type with unique name
	customTypeID := generateCustomTypeID(path, "object", hash)
	customTypeName := sa.generateUniqueCustomTypeName(path, "object")

	customType := &CustomTypeInfo{
		ID:          customTypeID,
		Name:        customTypeName,
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
		customTypeName := sa.generateUniqueCustomTypeName(path, "array")

		customType := &CustomTypeInfo{
			ID:            customTypeID,
			Name:          customTypeName,
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

	// For object arrays, we need to create a custom type for the object
	// and then reference it properly
	if itemType == "object" {
		if objMap, ok := firstItem.(map[string]interface{}); ok {
			// Create custom type for the object
			objectCustomType, err := sa.createCustomTypeForObject(objMap, path+"_item")
			if err != nil {
				return nil, err
			}
			// Set itemType to reference the custom object type
			itemType = fmt.Sprintf("#/custom-types/extracted_custom_types/%s", objectCustomType.ID)
		} else {
			// Fallback to generic object type
			itemType = "object"
		}
	} else {
		// For primitive types, use the YAML type mapping
		itemType = mapJSONTypeToYAML(itemType)
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
	customTypeName := sa.generateUniqueCustomTypeName(path, "array")

	customType := &CustomTypeInfo{
		ID:            customTypeID,
		Name:          customTypeName,
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

	// Extract property name from path
	propertyName := extractPropertyName(path)

	// Skip if property name is empty (e.g., from malformed paths like "context.traits.")
	if propertyName == "" {
		return nil
	}

	// Create property key based on name + type for deduplication
	propertyKey := fmt.Sprintf("%s_%s", sanitizeID(propertyName), sanitizeID(yamlType))

	// Skip if property with same name+type already exists
	if _, exists := sa.Properties[propertyKey]; exists {
		return nil
	}

	// Generate unique property ID based on name and type combination
	propertyID := sa.generateUniquePropertyID(propertyName, yamlType)

	property := &PropertyInfo{
		ID:          propertyID,
		Name:        propertyName,
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

	// Extract property name from path
	propertyName := extractPropertyName(path)

	// Skip if property name is empty (e.g., from malformed paths like "context.traits.")
	if propertyName == "" {
		return nil
	}

	// Create property key based on name + type for deduplication (same as primitive properties)
	propertyKey := fmt.Sprintf("%s_%s", sanitizeID(propertyName), sanitizeID(typeRef))

	// Skip if property with same name+type already exists
	if _, exists := sa.Properties[propertyKey]; exists {
		return nil
	}

	// Generate unique property ID based on name and type combination
	propertyID := sa.generateUniquePropertyID(propertyName, typeRef)

	property := &PropertyInfo{
		ID:          propertyID,
		Name:        propertyName,
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
	// Clean the path to only contain letters
	cleanPath := strings.ReplaceAll(path, ".", "")
	cleanPath = strings.ReplaceAll(cleanPath, "_", "")
	cleanPath = strings.ReplaceAll(cleanPath, "-", "")
	cleanPath = strings.ReplaceAll(cleanPath, " ", "")

	// Remove any non-letter characters
	var letterOnly strings.Builder
	for _, char := range cleanPath {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') {
			letterOnly.WriteRune(char)
		}
	}

	cleanPath = letterOnly.String()

	// Ensure we have at least some letters to work with
	if cleanPath == "" {
		cleanPath = "Generated"
	}

	// Capitalize first letter
	if len(cleanPath) > 0 {
		cleanPath = strings.ToUpper(cleanPath[:1]) + strings.ToLower(cleanPath[1:])
	}

	// Add type suffix
	var typeName string
	if baseType == "array" {
		typeName = cleanPath + "Array"
	} else {
		typeName = cleanPath + "Type"
	}

	// Ensure the name is between 3 and 65 characters
	if len(typeName) < 3 {
		// Pad with "Type" if too short
		typeName = typeName + "Type"
		if len(typeName) < 3 {
			typeName = "GeneratedType"
		}
	}

	if len(typeName) > 65 {
		// Truncate if too long, but ensure it ends properly
		if baseType == "array" {
			maxBase := 65 - 5 // Reserve 5 chars for "Array"
			if maxBase > 0 {
				typeName = typeName[:maxBase] + "Array"
			} else {
				typeName = "GenArray"
			}
		} else {
			maxBase := 65 - 4 // Reserve 4 chars for "Type"
			if maxBase > 0 {
				typeName = typeName[:maxBase] + "Type"
			} else {
				typeName = "GenType"
			}
		}
	}

	return typeName
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

func generateUniquePropertyID(name, propType string) string {
	cleanName := sanitizeID(name)

	// If propType is a custom type reference, extract a clean identifier
	if strings.HasPrefix(propType, "#/custom-types/") {
		// Extract the custom type ID from the reference
		parts := strings.Split(propType, "/")
		if len(parts) >= 3 {
			cleanType := sanitizeID(parts[len(parts)-1])
			return fmt.Sprintf("%s_%s", cleanName, cleanType)
		}
		// Fallback if parsing fails
		return fmt.Sprintf("%s_customtype", cleanName)
	}

	// For primitive types, sanitize normally
	cleanType := sanitizeID(propType)
	return fmt.Sprintf("%s_%s", cleanName, cleanType)
}

func (sa *SchemaAnalyzer) generateUniqueCustomTypeName(path, baseType string) string {
	baseName := generateCustomTypeName(path, baseType)

	// If the name is not used, return it
	if !sa.UsedCustomTypeNames[baseName] {
		sa.UsedCustomTypeNames[baseName] = true
		return baseName
	}

	// If it's already used, add letter suffixes to maintain letter-only requirement
	letterSuffixes := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M", "N", "O", "P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z"}

	for _, suffix := range letterSuffixes {
		candidateName := baseName + suffix

		// Ensure it doesn't exceed 65 characters
		if len(candidateName) > 65 {
			// Truncate base name to make room for suffix
			maxBaseLen := 65 - len(suffix)
			if maxBaseLen > 3 { // Ensure minimum length
				candidateName = baseName[:maxBaseLen] + suffix
			} else {
				candidateName = "GenType" + suffix
			}
		}

		if !sa.UsedCustomTypeNames[candidateName] {
			sa.UsedCustomTypeNames[candidateName] = true
			return candidateName
		}
	}

	// If all single letters are used, try double letters
	for _, suffix1 := range letterSuffixes {
		for _, suffix2 := range letterSuffixes {
			suffix := suffix1 + suffix2
			candidateName := baseName + suffix

			// Ensure it doesn't exceed 65 characters
			if len(candidateName) > 65 {
				// Truncate base name to make room for suffix
				maxBaseLen := 65 - len(suffix)
				if maxBaseLen > 3 { // Ensure minimum length
					candidateName = baseName[:maxBaseLen] + suffix
				} else {
					candidateName = "GenType" + suffix
				}
			}

			if !sa.UsedCustomTypeNames[candidateName] {
				sa.UsedCustomTypeNames[candidateName] = true
				return candidateName
			}
		}
	}

	// Fallback - this should rarely happen
	return "UniqueGenType"
}

func (sa *SchemaAnalyzer) generateUniquePropertyID(name, propType string) string {
	baseID := generateUniquePropertyID(name, propType)

	// If the ID is not used, return it
	if !sa.UsedPropertyIDs[baseID] {
		sa.UsedPropertyIDs[baseID] = true
		return baseID
	}

	// If it's already used, add a counter
	counter := 1
	for {
		candidateID := fmt.Sprintf("%s_%d", baseID, counter)
		if !sa.UsedPropertyIDs[candidateID] {
			sa.UsedPropertyIDs[candidateID] = true
			return candidateID
		}
		counter++
	}
}

func generateRandomID() string {
	const length = 10
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}
	return string(result)
}

func sanitizeEventID(input string) string {
	// Remove or replace problematic characters
	result := input

	// Replace common problematic characters
	replacements := map[string]string{
		"/":  "_",
		"\\": "_",
		"?":  "_",
		"<":  "_",
		">":  "_",
		"\"": "_",
		"'":  "_",
		"(":  "_",
		")":  "_",
		"[":  "_",
		"]":  "_",
		"{":  "_",
		"}":  "_",
		"|":  "_",
		"#":  "_",
		"%":  "_",
		"&":  "_",
		"*":  "_",
		"+":  "_",
		"=":  "_",
		"@":  "_",
		"!":  "_",
		" ":  "_",
		"\t": "_",
		"\n": "_",
		"\r": "_",
	}

	for old, new := range replacements {
		result = strings.ReplaceAll(result, old, new)
	}

	// Remove any remaining non-alphanumeric characters except underscores and hyphens
	var clean strings.Builder
	for _, char := range result {
		if (char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '_' || char == '-' {
			clean.WriteRune(char)
		}
	}

	result = clean.String()

	// Remove consecutive underscores
	for strings.Contains(result, "__") {
		result = strings.ReplaceAll(result, "__", "_")
	}

	// Trim underscores from start and end
	result = strings.Trim(result, "_")

	// Ensure lowercase for consistency
	result = strings.ToLower(result)

	return result
}
