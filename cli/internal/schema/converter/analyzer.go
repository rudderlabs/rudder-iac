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

// StringUtils handles string manipulation utilities
type StringUtils struct{}

// SanitizationMode defines different sanitization strategies
type SanitizationMode int

const (
	SanitizationModeBasic SanitizationMode = iota
	SanitizationModeEvent
)

// sanitize provides unified string sanitization with different modes
func (su *StringUtils) sanitize(input string, mode SanitizationMode) string {
	if mode == SanitizationModeEvent {
		return su.sanitizeEventID(input)
	}
	return su.sanitizeBasic(input)
}

// sanitizeBasic performs basic string sanitization
func (su *StringUtils) sanitizeBasic(input string) string {
	// Replace non-alphanumeric characters with underscores
	result := strings.ReplaceAll(input, ".", "_")
	result = strings.ReplaceAll(result, "-", "_")
	result = strings.ReplaceAll(result, " ", "_")
	result = strings.ToLower(result)
	return result
}

// sanitizeEventID performs event-specific sanitization with more comprehensive character replacement
func (su *StringUtils) sanitizeEventID(input string) string {
	// Remove or replace problematic characters
	result := input

	// Replace common problematic characters
	replacements := map[string]string{
		"/": "_", "\\": "_", "?": "_", "<": "_", ">": "_", "\"": "_",
		"'": "_", "(": "_", ")": "_", "[": "_", "]": "_", "{": "_",
		"}": "_", "|": "_", "#": "_", "%": "_", "&": "_", "*": "_",
		"+": "_", "=": "_", "@": "_", "!": "_", " ": "_", "\t": "_",
		"\n": "_", "\r": "_",
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

// generateHash creates consistent hashes with optional prefix
func (su *StringUtils) generateHash(content, prefix string) string {
	var hashInput string
	if prefix != "" {
		hashInput = prefix + "_" + content
	} else {
		hashInput = content
	}

	hash := md5.Sum([]byte(hashInput))
	return fmt.Sprintf("%x", hash)[:8]
}

// UniquenessStrategy defines how to resolve naming conflicts
type UniquenessStrategy int

const (
	UniquenessStrategyCounter UniquenessStrategy = iota
	UniquenessStrategyLetterSuffix
)

// ensureUnique resolves naming conflicts using the specified strategy
func (su *StringUtils) ensureUnique(baseName string, usedNames map[string]bool, strategy UniquenessStrategy, maxLength int) string {
	if !usedNames[baseName] {
		usedNames[baseName] = true
		return baseName
	}

	switch strategy {
	case UniquenessStrategyLetterSuffix:
		return su.ensureUniqueWithLetterSuffix(baseName, usedNames, maxLength)
	default:
		return su.ensureUniqueWithCounter(baseName, usedNames)
	}
}

// ensureUniqueWithCounter adds numeric suffixes for uniqueness
func (su *StringUtils) ensureUniqueWithCounter(baseName string, usedNames map[string]bool) string {
	counter := 1
	for {
		candidateName := fmt.Sprintf("%s_%d", baseName, counter)
		if !usedNames[candidateName] {
			usedNames[candidateName] = true
			return candidateName
		}
		counter++
	}
}

// ensureUniqueWithLetterSuffix adds letter suffixes for uniqueness
func (su *StringUtils) ensureUniqueWithLetterSuffix(baseName string, usedNames map[string]bool, maxLength int) string {
	letterSuffixes := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M", "N", "O", "P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z"}

	// Try single letters first
	for _, suffix := range letterSuffixes {
		candidateName := baseName + suffix
		if maxLength > 0 && len(candidateName) > maxLength {
			maxBaseLen := maxLength - len(suffix)
			if maxBaseLen > 3 {
				candidateName = baseName[:maxBaseLen] + suffix
			} else {
				candidateName = "GenType" + suffix
			}
		}

		if !usedNames[candidateName] {
			usedNames[candidateName] = true
			return candidateName
		}
	}

	// Try double letters if needed
	for _, suffix1 := range letterSuffixes {
		for _, suffix2 := range letterSuffixes {
			suffix := suffix1 + suffix2
			candidateName := baseName + suffix
			if maxLength > 0 && len(candidateName) > maxLength {
				maxBaseLen := maxLength - len(suffix)
				if maxBaseLen > 3 {
					candidateName = baseName[:maxBaseLen] + suffix
				} else {
					candidateName = "GenType" + suffix
				}
			}

			if !usedNames[candidateName] {
				usedNames[candidateName] = true
				return candidateName
			}
		}
	}

	// Fallback
	return "UniqueGenType"
}

// PropertyFactory handles property creation logic
type PropertyFactory struct {
	stringUtils *StringUtils
}

// NewPropertyFactory creates a new property factory
func NewPropertyFactory() *PropertyFactory {
	return &PropertyFactory{
		stringUtils: &StringUtils{},
	}
}

// createPropertyInternal handles common property creation logic
func (pf *PropertyFactory) createPropertyInternal(analyzer *SchemaAnalyzer, path string, propType string, isCustomType bool, customType *CustomTypeInfo) error {
	if path == "" {
		return nil // Skip root level
	}

	// Extract property name from path
	propertyName := extractPropertyName(path)

	// Skip if property name is empty
	if propertyName == "" {
		return nil
	}

	// Generate the type reference
	var typeRef string
	var jsonType string
	var description string

	if isCustomType && customType != nil {
		typeRef = fmt.Sprintf("#/custom-types/extracted_custom_types/%s", customType.ID)
		jsonType = customType.Type
		description = fmt.Sprintf("Property with custom type from path: %s", path)
	} else {
		yamlType := mapJSONTypeToYAML(propType)
		typeRef = yamlType
		jsonType = propType
		description = fmt.Sprintf("Property extracted from path: %s", path)
	}

	// Create property key based on name + type for deduplication
	propertyKey := fmt.Sprintf("%s_%s", pf.stringUtils.sanitize(propertyName, SanitizationModeBasic), pf.stringUtils.sanitize(typeRef, SanitizationModeBasic))

	// Skip if property with same name+type already exists
	if _, exists := analyzer.Properties[propertyKey]; exists {
		return nil
	}

	// Generate unique property ID
	propertyID := pf.generateUniquePropertyID(analyzer, propertyName, typeRef)

	property := &PropertyInfo{
		ID:          propertyID,
		Name:        propertyName,
		Type:        typeRef,
		Description: description,
		Path:        path,
		JsonType:    jsonType,
	}

	analyzer.Properties[propertyKey] = property
	return nil
}

// generateUniquePropertyID generates unique property IDs
func (pf *PropertyFactory) generateUniquePropertyID(analyzer *SchemaAnalyzer, name, propType string) string {
	cleanName := pf.stringUtils.sanitize(name, SanitizationModeBasic)

	// If propType is a custom type reference, extract a clean identifier
	var cleanType string
	if strings.HasPrefix(propType, "#/custom-types/") {
		parts := strings.Split(propType, "/")
		if len(parts) >= 3 {
			cleanType = pf.stringUtils.sanitize(parts[len(parts)-1], SanitizationModeBasic)
		} else {
			cleanType = "customtype"
		}
	} else {
		cleanType = pf.stringUtils.sanitize(propType, SanitizationModeBasic)
	}

	baseID := fmt.Sprintf("%s_%s", cleanName, cleanType)
	return pf.stringUtils.ensureUnique(baseID, analyzer.UsedPropertyIDs, UniquenessStrategyCounter, 0)
}

// CustomTypeFactory handles custom type creation logic
type CustomTypeFactory struct {
	stringUtils *StringUtils
}

// NewCustomTypeFactory creates a new custom type factory
func NewCustomTypeFactory() *CustomTypeFactory {
	return &CustomTypeFactory{
		stringUtils: &StringUtils{},
	}
}

// createCustomTypeInternal handles common custom type creation logic
func (ctf *CustomTypeFactory) createCustomTypeInternal(analyzer *SchemaAnalyzer, path, baseType string, structure map[string]string, arrayItemType string) (*CustomTypeInfo, error) {
	// Generate hash for uniqueness
	var hash string
	if baseType == "array" {
		hash = ctf.stringUtils.generateHash(arrayItemType, "array")
	} else {
		hash = ctf.generateStructureHash(structure)
	}

	// Check if custom type already exists
	for _, customType := range analyzer.CustomTypes {
		if customType.Hash == hash && customType.Type == baseType {
			return customType, nil
		}
	}

	// Create new custom type with unique name
	customTypeID := ctf.generateCustomTypeID(path, baseType, hash)
	customTypeName := ctf.generateUniqueCustomTypeName(analyzer, path, baseType)

	customType := &CustomTypeInfo{
		ID:            customTypeID,
		Name:          customTypeName,
		Type:          baseType,
		Description:   fmt.Sprintf("Custom %s type for %s", baseType, path),
		Structure:     structure,
		ArrayItemType: arrayItemType,
		Hash:          hash,
	}

	analyzer.CustomTypes[customTypeID] = customType
	return customType, nil
}

// generateStructureHash creates a hash for object structures
func (ctf *CustomTypeFactory) generateStructureHash(structure map[string]string) string {
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
	return ctf.stringUtils.generateHash(content, "")
}

// generateCustomTypeID creates consistent custom type IDs
func (ctf *CustomTypeFactory) generateCustomTypeID(path, baseType, hash string) string {
	cleanPath := ctf.stringUtils.sanitize(path, SanitizationModeBasic)
	return fmt.Sprintf("%s_%s_%s", cleanPath, baseType, hash)
}

// generateUniqueCustomTypeName creates unique custom type names
func (ctf *CustomTypeFactory) generateUniqueCustomTypeName(analyzer *SchemaAnalyzer, path, baseType string) string {
	baseName := ctf.generateCustomTypeName(path, baseType)
	return ctf.stringUtils.ensureUnique(baseName, analyzer.UsedCustomTypeNames, UniquenessStrategyLetterSuffix, 65)
}

// generateCustomTypeName creates base custom type names
func (ctf *CustomTypeFactory) generateCustomTypeName(path, baseType string) string {
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

	// Ensure minimum length
	if len(typeName) < 3 {
		typeName = typeName + "Type"
		if len(typeName) < 3 {
			typeName = "GeneratedType"
		}
	}

	// Handle maximum length
	if len(typeName) > 65 {
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

	customTypeFactory := NewCustomTypeFactory()
	return customTypeFactory.createCustomTypeInternal(sa, path, "object", structure, "")
}

// createCustomTypeForArray creates a custom type for an array
func (sa *SchemaAnalyzer) createCustomTypeForArray(arr []interface{}, path string) (*CustomTypeInfo, error) {
	customTypeFactory := NewCustomTypeFactory()

	if len(arr) == 0 {
		// Empty array - default to string array
		return customTypeFactory.createCustomTypeInternal(sa, path, "array", nil, "string")
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

	return customTypeFactory.createCustomTypeInternal(sa, path, "array", nil, itemType)
}

// createProperty creates a property for a primitive type
func (sa *SchemaAnalyzer) createProperty(path string, value interface{}) error {
	propertyFactory := NewPropertyFactory()
	jsonType := getJSONType(value)
	return propertyFactory.createPropertyInternal(sa, path, jsonType, false, nil)
}

// createPropertyWithCustomType creates a property that references a custom type
func (sa *SchemaAnalyzer) createPropertyWithCustomType(path string, customType *CustomTypeInfo) error {
	propertyFactory := NewPropertyFactory()
	return propertyFactory.createPropertyInternal(sa, path, "", true, customType)
}

// Helper functions

func sanitizeID(input string) string {
	stringUtils := &StringUtils{}
	return stringUtils.sanitize(input, SanitizationModeBasic)
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

func generateArrayHash(itemType string) string {
	stringUtils := &StringUtils{}
	return stringUtils.generateHash(itemType, "array")
}

func (sa *SchemaAnalyzer) generateUniqueCustomTypeName(path, baseType string) string {
	customTypeFactory := NewCustomTypeFactory()
	return customTypeFactory.generateUniqueCustomTypeName(sa, path, baseType)
}

func (sa *SchemaAnalyzer) generateUniquePropertyID(name, propType string) string {
	propertyFactory := NewPropertyFactory()
	return propertyFactory.generateUniquePropertyID(sa, name, propType)
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
	stringUtils := &StringUtils{}
	return stringUtils.sanitize(input, SanitizationModeEvent)
}
