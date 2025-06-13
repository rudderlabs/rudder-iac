package converter

import (
	"fmt"
	"strings"
)

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
