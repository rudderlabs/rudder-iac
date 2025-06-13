package converter

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/schema/models"
)

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

	eventName := eventIdentifier // Use the original eventIdentifier as the event name

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

// Backward compatibility wrapper methods

// generateUniqueCustomTypeName provides backward compatibility wrapper
func (sa *SchemaAnalyzer) generateUniqueCustomTypeName(path, baseType string) string {
	customTypeFactory := NewCustomTypeFactory()
	return customTypeFactory.generateUniqueCustomTypeName(sa, path, baseType)
}

// generateUniquePropertyID provides backward compatibility wrapper
func (sa *SchemaAnalyzer) generateUniquePropertyID(name, propType string) string {
	propertyFactory := NewPropertyFactory()
	return propertyFactory.generateUniquePropertyID(sa, name, propType)
}
