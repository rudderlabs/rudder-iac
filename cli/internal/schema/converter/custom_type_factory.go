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
