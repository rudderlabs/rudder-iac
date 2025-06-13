package converter

import (
	"fmt"
	"sort"
	"strings"
)

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
	// Create a temporary custom type to generate the uniqueness signature
	tempCustomType := &CustomTypeInfo{
		Type:          baseType,
		Structure:     structure,
		ArrayItemType: arrayItemType,
	}

	// Generate uniqueness signature based on YAML output structure
	uniquenessSignature := ctf.generateUniquenessSignature(analyzer, tempCustomType)

	// Check if custom type with same uniqueness signature already exists
	for _, customType := range analyzer.CustomTypes {
		existingSignature := ctf.generateUniquenessSignature(analyzer, customType)
		if existingSignature == uniquenessSignature {
			return customType, nil
		}
	}

	// Generate custom type ID based on uniqueness signature for consistent deduplication
	customTypeID := ctf.generateCustomTypeIDFromSignature(uniquenessSignature)
	customTypeName := ctf.generateUniqueCustomTypeName(analyzer, path, baseType)

	customType := &CustomTypeInfo{
		ID:            customTypeID,
		Name:          customTypeName,
		Type:          baseType,
		Description:   fmt.Sprintf("Custom %s type for %s", baseType, path),
		Structure:     structure,
		ArrayItemType: arrayItemType,
		Hash:          customTypeID, // Use the ID as the hash for backward compatibility
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

// generateCustomTypeIDFromSignature creates consistent custom type IDs based on uniqueness signature
func (ctf *CustomTypeFactory) generateCustomTypeIDFromSignature(signature string) string {
	// Generate a hash from the signature to create a deterministic ID
	signatureHash := ctf.stringUtils.generateHash(signature, "")

	// Extract type from signature
	parts := strings.SplitN(signature, ":", 2)
	if len(parts) < 2 {
		return fmt.Sprintf("unknown_type_%s", signatureHash)
	}

	baseType := parts[0]
	return fmt.Sprintf("%s_type_%s", baseType, signatureHash)
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

// generateUniquenessSignature creates a signature for uniqueness based on structure content
func (ctf *CustomTypeFactory) generateUniquenessSignature(analyzer *SchemaAnalyzer, customType *CustomTypeInfo) string {
	if customType.Type == "array" {
		// For array types, uniqueness is based on itemType
		return fmt.Sprintf("array:%s", customType.ArrayItemType)
	} else if customType.Type == "object" {
		// For object types, uniqueness is based on structure (property references)
		if len(customType.Structure) == 0 {
			return "object:empty"
		}

		// Sort structure keys for consistent output
		var structKeys []string
		for structKey := range customType.Structure {
			structKeys = append(structKeys, structKey)
		}
		sort.Strings(structKeys)

		// Create signature based on property references
		var propertyRefs []string
		for _, structKey := range structKeys {
			typeInfo := customType.Structure[structKey]
			// Find the property ID for this field
			propertyID := ctf.findPropertyForStructField(analyzer, structKey, customType)
			if propertyID != "" {
				propertyRef := fmt.Sprintf("#/properties/extracted_properties/%s", propertyID)
				propertyRefs = append(propertyRefs, propertyRef)
			} else {
				// Fallback to field:type format if property not found
				propertyRefs = append(propertyRefs, fmt.Sprintf("%s:%s", structKey, typeInfo))
			}
		}

		// Sort the refs for deterministic ordering
		sort.Strings(propertyRefs)
		return fmt.Sprintf("object:%s", strings.Join(propertyRefs, "|"))
	}

	// Fallback for other types
	return fmt.Sprintf("%s:unknown", customType.Type)
}

// findPropertyForStructField finds a property ID for a given struct field
func (ctf *CustomTypeFactory) findPropertyForStructField(analyzer *SchemaAnalyzer, fieldName string, typeInfo *CustomTypeInfo) string {
	// Use the same logic as in generators.go for consistency
	for _, property := range analyzer.Properties {
		// Check if the property path ends with the field name
		// This handles nested paths like "properties.user.name" matching field "name"
		pathParts := strings.Split(property.Path, ".")
		if len(pathParts) > 0 && pathParts[len(pathParts)-1] == fieldName {
			return property.ID
		}

		// Also check if the property name matches
		if property.Name == fieldName {
			return property.ID
		}
	}
	return ""
}
