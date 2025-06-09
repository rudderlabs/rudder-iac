package properties

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/pkg/logger"
	schemaErrors "github.com/rudderlabs/rudder-iac/cli/pkg/schema/errors"
)

// PropertyInfo holds analyzed property information
type PropertyInfo struct {
	Name        string                   `json:"name"`
	Type        string                   `json:"type"`
	Description string                   `json:"description,omitempty"`
	Required    bool                     `json:"required"`
	Example     interface{}              `json:"example,omitempty"`
	Properties  map[string]*PropertyInfo `json:"properties,omitempty"` // For object types
	Items       *PropertyInfo            `json:"items,omitempty"`      // For array types
	Enum        []interface{}            `json:"enum,omitempty"`       // For enum types
}

// TypeDetector detects property types from schema values
type TypeDetector interface {
	DetectType(value interface{}) string
	InferTypeFromValue(value interface{}) string
}

// NameResolver resolves property names from paths
type NameResolver interface {
	ResolveName(path string) string
	ValidateName(name string) error
}

// PropertyAnalyzer analyzes property structures from schemas
type PropertyAnalyzer struct {
	typeDetector TypeDetector
	nameResolver NameResolver
	logger       *logger.Logger
}

// defaultTypeDetector provides basic type detection
type defaultTypeDetector struct{}

func (d *defaultTypeDetector) DetectType(value interface{}) string {
	if value == nil {
		return "null"
	}

	switch v := value.(type) {
	case bool:
		return "boolean"
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return "integer"
	case float32, float64:
		return "number"
	case string:
		return "string"
	case []interface{}:
		if len(v) > 0 {
			// Detect array item type
			itemType := d.DetectType(v[0])
			return fmt.Sprintf("array<%s>", itemType)
		}
		return "array"
	case map[string]interface{}:
		return "object"
	default:
		// Use reflection as fallback
		return d.InferTypeFromValue(value)
	}
}

func (d *defaultTypeDetector) InferTypeFromValue(value interface{}) string {
	if value == nil {
		return "null"
	}

	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Bool:
		return "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "integer"
	case reflect.Float32, reflect.Float64:
		return "number"
	case reflect.String:
		return "string"
	case reflect.Slice, reflect.Array:
		return "array"
	case reflect.Map, reflect.Struct:
		return "object"
	default:
		return "unknown"
	}
}

// defaultNameResolver provides basic name resolution
type defaultNameResolver struct{}

func (r *defaultNameResolver) ResolveName(path string) string {
	// Clean up the path to create a proper property name
	name := strings.TrimPrefix(path, "properties.")
	name = strings.ReplaceAll(name, ".", "_")
	name = strings.ReplaceAll(name, "-", "_")

	// Convert to camelCase
	parts := strings.Split(name, "_")
	if len(parts) == 0 {
		return "unknown_property"
	}

	result := strings.ToLower(parts[0])
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) > 0 {
			result += strings.ToUpper(parts[i][:1])
			if len(parts[i]) > 1 {
				result += strings.ToLower(parts[i][1:])
			}
		}
	}

	if result == "" {
		return "unknown_property"
	}

	return result
}

func (r *defaultNameResolver) ValidateName(name string) error {
	if name == "" {
		return schemaErrors.NewProcessError(
			schemaErrors.ErrorTypeProcessValidation,
			"Property name cannot be empty",
			schemaErrors.AsUserError(),
		)
	}

	if len(name) > 100 {
		return schemaErrors.NewProcessError(
			schemaErrors.ErrorTypeProcessValidation,
			"Property name too long (max 100 characters)",
			schemaErrors.AsUserError(),
		)
	}

	// Check for valid identifier
	if !isValidIdentifier(name) {
		return schemaErrors.NewProcessError(
			schemaErrors.ErrorTypeProcessValidation,
			"Property name must be a valid identifier",
			schemaErrors.AsUserError(),
		)
	}

	return nil
}

// isValidIdentifier checks if a string is a valid identifier
func isValidIdentifier(s string) bool {
	if len(s) == 0 {
		return false
	}

	// Must start with letter or underscore
	first := s[0]
	if !((first >= 'a' && first <= 'z') || (first >= 'A' && first <= 'Z') || first == '_') {
		return false
	}

	// Rest can be letters, digits, or underscores
	for i := 1; i < len(s); i++ {
		c := s[i]
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
			return false
		}
	}

	return true
}

// NewPropertyAnalyzer creates a new property analyzer
func NewPropertyAnalyzer(log *logger.Logger) *PropertyAnalyzer {
	return &PropertyAnalyzer{
		typeDetector: &defaultTypeDetector{},
		nameResolver: &defaultNameResolver{},
		logger:       log,
	}
}

// NewPropertyAnalyzerWithDeps creates a property analyzer with custom dependencies
func NewPropertyAnalyzerWithDeps(typeDetector TypeDetector, nameResolver NameResolver, log *logger.Logger) *PropertyAnalyzer {
	return &PropertyAnalyzer{
		typeDetector: typeDetector,
		nameResolver: nameResolver,
		logger:       log,
	}
}

// AnalyzeProperties analyzes properties from a schema
func (p *PropertyAnalyzer) AnalyzeProperties(schema map[string]interface{}) (map[string]*PropertyInfo, error) {
	properties := make(map[string]*PropertyInfo)

	err := p.analyzePropertiesRecursive(schema, "", properties)
	if err != nil {
		return nil, schemaErrors.NewProcessError(
			schemaErrors.ErrorTypeProcessTransform,
			"Failed to analyze properties",
			schemaErrors.WithCause(err),
		)
	}

	p.logger.Info(fmt.Sprintf("Analyzed %d properties from schema", len(properties)))

	return properties, nil
}

// analyzePropertiesRecursive recursively analyzes nested properties
func (p *PropertyAnalyzer) analyzePropertiesRecursive(obj interface{}, path string, properties map[string]*PropertyInfo) error {
	switch v := obj.(type) {
	case map[string]interface{}:
		for key, value := range v {
			currentPath := key
			if path != "" {
				currentPath = path + "." + key
			}

			// Resolve property name
			propName := p.nameResolver.ResolveName(currentPath)
			if err := p.nameResolver.ValidateName(propName); err != nil {
				p.logger.Warn(fmt.Sprintf("Skipping invalid property name %s: %v", propName, err))
				continue
			}

			// Create property info
			propInfo := &PropertyInfo{
				Name:        propName,
				Type:        p.typeDetector.DetectType(value),
				Description: p.generateDescription(propName, value),
				Required:    false, // Default to false, could be enhanced with schema analysis
				Example:     p.generateExample(value),
			}

			// Handle nested objects
			if propInfo.Type == "object" {
				if objValue, ok := value.(map[string]interface{}); ok {
					nestedProps := make(map[string]*PropertyInfo)
					if err := p.analyzePropertiesRecursive(objValue, currentPath, nestedProps); err != nil {
						return err
					}
					propInfo.Properties = nestedProps
				}
			}

			// Handle arrays
			if strings.HasPrefix(propInfo.Type, "array") {
				if arrayValue, ok := value.([]interface{}); ok && len(arrayValue) > 0 {
					itemInfo := &PropertyInfo{
						Type: p.typeDetector.DetectType(arrayValue[0]),
					}

					// Analyze array item if it's an object
					if itemInfo.Type == "object" {
						if objValue, ok := arrayValue[0].(map[string]interface{}); ok {
							nestedProps := make(map[string]*PropertyInfo)
							if err := p.analyzePropertiesRecursive(objValue, currentPath+"[0]", nestedProps); err != nil {
								return err
							}
							itemInfo.Properties = nestedProps
						}
					}

					propInfo.Items = itemInfo
				}
			}

			properties[propName] = propInfo
		}

	case []interface{}:
		// Analyze array items
		for i, item := range v {
			itemPath := fmt.Sprintf("%s[%d]", path, i)
			if err := p.analyzePropertiesRecursive(item, itemPath, properties); err != nil {
				return err
			}
		}

	default:
		// For primitive values, create a simple property
		if path != "" {
			propName := p.nameResolver.ResolveName(path)
			if err := p.nameResolver.ValidateName(propName); err == nil {
				properties[propName] = &PropertyInfo{
					Name:        propName,
					Type:        p.typeDetector.DetectType(v),
					Description: p.generateDescription(propName, v),
					Example:     p.generateExample(v),
				}
			}
		}
	}

	return nil
}

// generateDescription creates a description for a property
func (p *PropertyAnalyzer) generateDescription(name string, value interface{}) string {
	typeName := p.typeDetector.DetectType(value)

	// Create basic description based on name and type
	description := fmt.Sprintf("A %s property", typeName)

	// Add more context based on name patterns
	lowerName := strings.ToLower(name)
	if strings.Contains(lowerName, "id") {
		description = "Unique identifier"
	} else if strings.Contains(lowerName, "name") {
		description = "Name or title"
	} else if strings.Contains(lowerName, "email") {
		description = "Email address"
	} else if strings.Contains(lowerName, "timestamp") || strings.Contains(lowerName, "time") {
		description = "Timestamp value"
	} else if strings.Contains(lowerName, "count") || strings.Contains(lowerName, "number") {
		description = "Numeric count or quantity"
	} else if strings.Contains(lowerName, "url") || strings.Contains(lowerName, "link") {
		description = "URL or link"
	}

	return description
}

// generateExample creates an example value for a property
func (p *PropertyAnalyzer) generateExample(value interface{}) interface{} {
	// For now, use the actual value as example
	// In a more sophisticated implementation, we might anonymize or generate typical examples
	switch v := value.(type) {
	case string:
		if len(v) > 50 {
			return v[:47] + "..."
		}
		return v
	case []interface{}:
		if len(v) > 0 {
			return []interface{}{p.generateExample(v[0])}
		}
		return []interface{}{}
	case map[string]interface{}:
		example := make(map[string]interface{})
		count := 0
		for key, val := range v {
			if count >= 3 { // Limit example object size
				break
			}
			example[key] = p.generateExample(val)
			count++
		}
		return example
	default:
		return v
	}
}

// MergeProperties merges multiple property maps
func (p *PropertyAnalyzer) MergeProperties(propMaps ...map[string]*PropertyInfo) map[string]*PropertyInfo {
	merged := make(map[string]*PropertyInfo)

	for _, propMap := range propMaps {
		for name, prop := range propMap {
			if existing, exists := merged[name]; exists {
				// Merge properties with the same name
				p.mergePropertyInfo(existing, prop)
			} else {
				merged[name] = prop
			}
		}
	}

	return merged
}

// mergePropertyInfo merges two PropertyInfo instances
func (p *PropertyAnalyzer) mergePropertyInfo(existing, new *PropertyInfo) {
	// If types differ, use 'any' type
	if existing.Type != new.Type {
		existing.Type = "any"
	}

	// Keep the more descriptive description
	if len(new.Description) > len(existing.Description) {
		existing.Description = new.Description
	}

	// Merge required flag (true if either is required)
	existing.Required = existing.Required || new.Required

	// Merge nested properties for objects
	if existing.Properties != nil && new.Properties != nil {
		for name, prop := range new.Properties {
			if existingProp, exists := existing.Properties[name]; exists {
				p.mergePropertyInfo(existingProp, prop)
			} else {
				existing.Properties[name] = prop
			}
		}
	} else if new.Properties != nil {
		existing.Properties = new.Properties
	}
}

// GetPropertyStats returns statistics about analyzed properties
func (p *PropertyAnalyzer) GetPropertyStats(properties map[string]*PropertyInfo) map[string]interface{} {
	stats := make(map[string]interface{})

	totalProps := len(properties)
	typeDistribution := make(map[string]int)
	requiredCount := 0
	objectCount := 0
	arrayCount := 0

	for _, prop := range properties {
		typeDistribution[prop.Type]++

		if prop.Required {
			requiredCount++
		}

		if prop.Type == "object" {
			objectCount++
		}

		if strings.HasPrefix(prop.Type, "array") {
			arrayCount++
		}
	}

	stats["total_properties"] = totalProps
	stats["type_distribution"] = typeDistribution
	stats["required_count"] = requiredCount
	stats["object_count"] = objectCount
	stats["array_count"] = arrayCount

	return stats
}
