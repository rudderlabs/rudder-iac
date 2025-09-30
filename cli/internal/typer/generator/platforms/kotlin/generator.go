package kotlin

import (
	_ "embed"
	"fmt"
	"sort"

	"github.com/rudderlabs/rudder-iac/cli/internal/typer/generator/core"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer/plan"
)

const (
	Platform = "kotlin"
)

func Generate(plan *plan.TrackingPlan, options core.GenerationOptions) ([]*core.File, error) {
	ctx := NewKotlinContext()
	ctx.EventContext = formatEventContext(plan.Metadata, options.RudderCLIVersion)
	nameRegistry := core.NewNameRegistry(KotlinCollisionHandler)

	err := processPropertiesAndCustomTypes(plan, ctx, nameRegistry)
	if err != nil {
		return nil, err
	}

	err = processEventRules(plan, ctx, nameRegistry)
	if err != nil {
		return nil, err
	}

	mainFile, err := GenerateFile("Main.kt", ctx)
	if err != nil {
		return nil, err
	}

	return []*core.File{
		mainFile,
	}, nil
}

func formatEventContext(ec plan.PlanMetadata, rudderCLIVersion string) map[string]string {
	return map[string]string{
		"platform":            fmt.Sprintf("%q", Platform),
		"rudderCLIVersion":    fmt.Sprintf("%q", rudderCLIVersion),
		"trackingPlanId":      fmt.Sprintf("%q", ec.TrackingPlanID),
		"trackingPlanVersion": fmt.Sprintf("%d", ec.TrackingPlanVersion),
	}
}

// processPropertiesAndCustomTypes extracts custom types and properties from the tracking plan and generates corresponding Kotlin types
func processPropertiesAndCustomTypes(p *plan.TrackingPlan, ctx *KotlinContext, nameRegistry *core.NameRegistry) error {
	// Use the plan helper methods to extract types and properties
	customTypes := p.ExtractAllCustomTypes()
	properties := p.ExtractAllProperties()

	// Process custom types first (both primitive and object)
	err := processCustomTypesIntoContext(customTypes, ctx, nameRegistry)
	if err != nil {
		return err
	}

	// Process individual properties to create type aliases
	err = processPropertiesIntoContext(properties, ctx, nameRegistry)
	if err != nil {
		return err
	}

	return nil
}

// processCustomTypesIntoContext processes custom types and adds them to the context
func processCustomTypesIntoContext(customTypes map[string]*plan.CustomType, ctx *KotlinContext, nameRegistry *core.NameRegistry) error {
	// Sort custom type names for deterministic output
	var sortedNames []string
	for name := range customTypes {
		sortedNames = append(sortedNames, name)
	}
	sort.Strings(sortedNames)

	// Generate type aliases for primitive custom types, enums for custom types with enum configs, and data classes for object custom types
	for _, name := range sortedNames {
		customType := customTypes[name]
		if customType.IsPrimitive() {
			// Check if this custom type has enum constraints
			if hasEnumConfig(customType.Config) {
				enum, err := createCustomTypeEnum(customType, nameRegistry)
				if err != nil {
					return err
				}
				ctx.Enums = append(ctx.Enums, *enum)
			} else {
				alias, err := createCustomTypeTypeAlias(customType, nameRegistry)
				if err != nil {
					return err
				}
				ctx.TypeAliases = append(ctx.TypeAliases, *alias)
			}
		} else if customType.Schema != nil && len(customType.Schema.Properties) == 0 && customType.Schema.AdditionalProperties {
			// If this is an empty object with additionalProperties: true
			// create a type alias to JsonObject instead of an empty data class
			finalName, err := getOrRegisterCustomTypeClassName(customType, nameRegistry)
			if err != nil {
				return err
			}
			alias := &KotlinTypeAlias{
				Alias:   finalName,
				Comment: customType.Description,
				Type:    "JsonObject",
			}
			ctx.TypeAliases = append(ctx.TypeAliases, *alias)
		} else {
			dataClass, err := createCustomTypeDataClass(customType, nameRegistry)
			if err != nil {
				return err
			}
			ctx.DataClasses = append(ctx.DataClasses, *dataClass)
		}
	}
	return nil
}

// processPropertiesIntoContext processes individual properties and creates type aliases or enums for all properties
func processPropertiesIntoContext(allProperties map[string]*plan.Property, ctx *KotlinContext, nameRegistry *core.NameRegistry) error {
	// Sort property names for deterministic output
	var sortedNames []string
	for name := range allProperties {
		sortedNames = append(sortedNames, name)
	}
	sort.Strings(sortedNames)

	// Generate type aliases or enums for all properties
	for _, name := range sortedNames {
		property := allProperties[name]

		// Check if this property has enum constraints
		if hasEnumConfig(property.Config) {
			enum, err := createPropertyEnum(property, nameRegistry)
			if err != nil {
				return err
			}
			ctx.Enums = append(ctx.Enums, *enum)
		} else {
			alias, err := createPropertyTypeAlias(property, nameRegistry)
			if err != nil {
				return err
			}
			ctx.TypeAliases = append(ctx.TypeAliases, *alias)
		}
	}
	return nil
}

// createCustomTypeTypeAlias creates a KotlinTypeAlias from a primitive custom type
func createCustomTypeTypeAlias(customType *plan.CustomType, nameRegistry *core.NameRegistry) (*KotlinTypeAlias, error) {
	finalName, err := getOrRegisterCustomTypeAliasName(customType, nameRegistry)
	if err != nil {
		return nil, err
	}

	var kotlinType string
	// Handle array custom types with ItemType
	if customType.Type == plan.PrimitiveTypeArray {
		if customType.ItemType != nil {
			innerKotlinType, err := resolveTypeToKotlinType(customType.ItemType, nameRegistry)
			if err != nil {
				return nil, err
			}
			kotlinType = fmt.Sprintf("List<%s>", innerKotlinType)
		} else {
			// Array with no item type means array of any
			kotlinType = "List<JsonElement>"
		}
	} else {
		// Map primitive type to Kotlin type
		kotlinType = mapPrimitiveToKotlinType(customType.Type)
	}

	return &KotlinTypeAlias{
		Alias:   finalName,
		Comment: customType.Description,
		Type:    kotlinType,
	}, nil
}

// createPropertyTypeAlias creates a KotlinTypeAlias from any property
func createPropertyTypeAlias(property *plan.Property, nameRegistry *core.NameRegistry) (*KotlinTypeAlias, error) {
	finalName, err := getOrRegisterPropertyAliasName(property, nameRegistry)
	if err != nil {
		return nil, err
	}

	// Get the appropriate Kotlin type for this property
	kotlinType, err := resolvePropertyKotlinType(property, nameRegistry)
	if err != nil {
		return nil, err
	}

	return &KotlinTypeAlias{
		Alias:   finalName,
		Comment: property.Description,
		Type:    kotlinType,
	}, nil
}

// resolvePropertyKotlinType resolves the Kotlin type for a property, handling arrays properly
func resolvePropertyKotlinType(property *plan.Property, nameRegistry *core.NameRegistry) (string, error) {
	// TODO: handle multiple types (union types) if needed
	// For now, we only handle single-type properties

	var propertyType plan.PropertyType
	if len(property.Types) == 0 {
		return "JsonElement", nil
	} else if len(property.Types) == 1 {
		propertyType = property.Types[0]
	} else {
		// TODO: Future enhancement for union types
		return "", fmt.Errorf("union types not yet supported: %v", property.Types)
	}

	if plan.IsPrimitiveType(propertyType) {
		primitiveType := *plan.AsPrimitiveType(propertyType)

		// Handle array types by using ItemType
		if primitiveType == plan.PrimitiveTypeArray {
			if len(property.ItemTypes) == 0 {
				// No item type specified means array can contain any type
				return "List<JsonElement>", nil
			} else if len(property.ItemTypes) == 1 {
				itemType := property.ItemTypes[0]
				innerKotlinType, err := resolveTypeToKotlinType(itemType, nameRegistry)
				if err != nil {
					return "", err
				}
				return fmt.Sprintf("List<%s>", innerKotlinType), nil
			} else {
				return "", fmt.Errorf("array properties must have exactly one item type: %v", property.ItemTypes)
			}
		}

		return mapPrimitiveToKotlinType(primitiveType), nil
	} else if plan.IsCustomType(propertyType) {
		return resolveTypeToKotlinType(propertyType, nameRegistry)
	} else {
		return "", fmt.Errorf("unsupported property type: %s", property.Types)
	}
}

// resolveTypeToKotlinType resolves a PropertyType to its Kotlin type representation
func resolveTypeToKotlinType(propertyType plan.PropertyType, nameRegistry *core.NameRegistry) (string, error) {
	if plan.IsPrimitiveType(propertyType) {
		return mapPrimitiveToKotlinType(*plan.AsPrimitiveType(propertyType)), nil
	} else if plan.IsCustomType(propertyType) {
		customType := plan.AsCustomType(propertyType)
		if customType.IsPrimitive() {
			// Check if this custom type has enum constraints
			if hasEnumConfig(customType.Config) {
				// Reference the custom type enum
				return getOrRegisterCustomTypeEnumName(customType, nameRegistry)
			}
			// Reference the custom type alias
			return getOrRegisterCustomTypeAliasName(customType, nameRegistry)
		} else {
			// Reference the custom type data class
			return getOrRegisterCustomTypeClassName(customType, nameRegistry)
		}
	} else {
		return "", fmt.Errorf("unsupported property type: %T", propertyType)
	}
}

func createDataClass(className string, comment string, schema *plan.ObjectSchema, nameRegistry *core.NameRegistry) (*KotlinDataClass, error) {
	// Sort property names for deterministic output
	var sortedPropNames []string
	for propName := range schema.Properties {
		sortedPropNames = append(sortedPropNames, propName)
	}
	sort.Strings(sortedPropNames)

	var properties []KotlinProperty
	var nestedClasses []KotlinDataClass

	for _, propName := range sortedPropNames {
		var property *KotlinProperty
		propSchema := schema.Properties[propName]

		// Check if this property has nested object schema
		if propSchema.Schema != nil {
			// Check if this is an empty object with additionalProperties: true
			if len(propSchema.Schema.Properties) == 0 && propSchema.Schema.AdditionalProperties {
				// Use the property's type alias instead of creating an empty data class
				kotlinType, err := getPropertyKotlinType(propSchema.Property, nameRegistry)
				if err != nil {
					return nil, err
				}
				property = &KotlinProperty{
					Name:       formatPropertyName(propName),
					SerialName: propName,
					Type:       kotlinType,
					Comment:    propSchema.Property.Description,
					Nullable:   !propSchema.Required,
				}
			} else {
				// Generate nested data class
				nestedClass, err := createNestedDataClass(&propSchema, propName, className, nameRegistry)
				if err != nil {
					return nil, err
				}
				nestedClasses = append(nestedClasses, *nestedClass)

				// Property type references the nested class
				nestedClassName := fmt.Sprintf("%s.%s", className, nestedClass.Name)
				property = &KotlinProperty{
					Name:       formatPropertyName(propName),
					SerialName: propName,
					Type:       nestedClassName,
					Comment:    propSchema.Property.Description,
					Nullable:   !propSchema.Required,
				}

			}
		} else {
			kotlinType, err := getPropertyKotlinType(propSchema.Property, nameRegistry)
			if err != nil {
				return nil, err
			}

			property = &KotlinProperty{
				Name:       formatPropertyName(propName),
				SerialName: propName,
				Type:       kotlinType,
				Comment:    propSchema.Property.Description,
				Nullable:   !propSchema.Required,
			}
		}

		if !propSchema.Required {
			property.Default = "null"
		}

		properties = append(properties, *property)

	}

	return &KotlinDataClass{
		Name:          className,
		Comment:       comment,
		Properties:    properties,
		NestedClasses: nestedClasses,
	}, nil
}

// createCustomTypeDataClass creates a KotlinDataClass from an object custom type
func createCustomTypeDataClass(customType *plan.CustomType, nameRegistry *core.NameRegistry) (*KotlinDataClass, error) {
	finalName, err := getOrRegisterCustomTypeClassName(customType, nameRegistry)
	if err != nil {
		return nil, err
	}

	return createDataClass(finalName, customType.Description, customType.Schema, nameRegistry)
}

// getPropertyKotlinType returns the Kotlin type name for a property, using appropriate type aliases or enum classes
func getPropertyKotlinType(property plan.Property, nameRegistry *core.NameRegistry) (string, error) {
	// Check if this property has enum constraints
	if hasEnumConfig(property.Config) {
		return getOrRegisterPropertyEnumName(&property, nameRegistry)
	}
	return getOrRegisterPropertyAliasName(&property, nameRegistry)
}

// processEventRules processes event rules and generates data classes for event properties/traits
func processEventRules(p *plan.TrackingPlan, ctx *KotlinContext, nameRegistry *core.NameRegistry) error {
	// Map rules by a unique composite key for deterministic processing
	ruleMap := make(map[string]*plan.EventRule)
	for _, rule := range p.Rules {
		// Create a unique key using event type, name, and section
		key := string(rule.Event.EventType) + ":" + rule.Event.Name + ":" + string(rule.Section)
		ruleMap[key] = &rule
	}

	// Sort keys for deterministic output
	var sortedKeys []string
	for key := range ruleMap {
		sortedKeys = append(sortedKeys, key)
	}
	sort.Strings(sortedKeys)

	// Process each rule to create event data classes
	for _, key := range sortedKeys {
		rule := ruleMap[key]

		// create data class and method for the event rule
		dataClass, err := createEventDataClass(rule, nameRegistry)
		if err != nil {
			return err
		}
		ctx.DataClasses = append(ctx.DataClasses, *dataClass)

		// create RudderAnalyticsMethod for the event rule
		method, err := createRudderAnalyticsMethod(rule, nameRegistry)
		if err != nil {
			return err
		}

		if method != nil {
			ctx.RudderAnalyticsMethods = append(ctx.RudderAnalyticsMethods, *method)
		}
	}

	return nil
}

// createEventDataClass creates a KotlinDataClass from an event rule
func createEventDataClass(rule *plan.EventRule, nameRegistry *core.NameRegistry) (*KotlinDataClass, error) {
	className, err := getOrRegisterEventDataClassName(rule, nameRegistry)
	if err != nil {
		return nil, err
	}

	return createDataClass(className, rule.Event.Description, &rule.Schema, nameRegistry)
}

// hasEnumConfig checks if a PropertyConfig has enum constraints defined
func hasEnumConfig(config *plan.PropertyConfig) bool {
	return config != nil && config.Enum != nil && len(config.Enum) > 0
}

// createPropertyEnum creates a KotlinEnum from a property with enum constraints
func createPropertyEnum(property *plan.Property, nameRegistry *core.NameRegistry) (*KotlinEnum, error) {
	enumName, err := getOrRegisterPropertyEnumName(property, nameRegistry)
	if err != nil {
		return nil, err
	}

	// Convert enum values to KotlinEnumValue structs
	var enumValues []KotlinEnumValue
	for _, value := range property.Config.Enum {
		enumValues = append(enumValues, KotlinEnumValue{
			Name:       FormatEnumValue(value),
			SerialName: FormatEnumSerialName(value),
		})
	}

	return &KotlinEnum{
		Name:    enumName,
		Comment: property.Description,
		Values:  enumValues,
	}, nil
}

// createCustomTypeEnum creates a KotlinEnum from a custom type with enum constraints
func createCustomTypeEnum(customType *plan.CustomType, nameRegistry *core.NameRegistry) (*KotlinEnum, error) {
	enumName, err := getOrRegisterCustomTypeEnumName(customType, nameRegistry)
	if err != nil {
		return nil, err
	}

	// Convert enum values to KotlinEnumValue structs
	var enumValues []KotlinEnumValue
	for _, value := range customType.Config.Enum {
		enumValues = append(enumValues, KotlinEnumValue{
			Name:       FormatEnumValue(value),
			SerialName: FormatEnumSerialName(value),
		})
	}

	return &KotlinEnum{
		Name:    enumName,
		Comment: customType.Description,
		Values:  enumValues,
	}, nil
}

// hasEnumConstraints checks if a property has enum constraints defined
func hasEnumConstraints(property *plan.Property) bool {
	return property.Config != nil && property.Config.Enum != nil && len(property.Config.Enum) > 0
}

// createPropertyEnum creates a KotlinEnum from a property with enum constraints
func createPropertyEnum(property *plan.Property, nameRegistry *core.NameRegistry) (*KotlinEnum, error) {
	enumName, err := getOrRegisterPropertyEnumName(property, nameRegistry)
	if err != nil {
		return nil, err
	}

	// Convert enum values to KotlinEnumValue structs
	var enumValues []KotlinEnumValue
	for _, value := range property.Config.Enum {
		enumValues = append(enumValues, KotlinEnumValue{
			Name:       formatEnumConstantName(value),
			SerialName: value,
		})
	}

	return &KotlinEnum{
		Name:    enumName,
		Comment: property.Description,
		Values:  enumValues,
	}, nil
}

// mapPrimitiveToKotlinType maps plan primitive types to Kotlin types
func mapPrimitiveToKotlinType(primitiveType plan.PrimitiveType) string {
	switch primitiveType {
	case plan.PrimitiveTypeString:
		return "String"
	case plan.PrimitiveTypeInteger:
		return "Long"
	case plan.PrimitiveTypeNumber:
		return "Double"
	case plan.PrimitiveTypeBoolean:
		return "Boolean"
	case plan.PrimitiveTypeAny:
		return "JsonElement"
	case plan.PrimitiveTypeObject:
		return "JsonObject"
	default:
		return "JsonElement" // Fallback for unknown types
	}
}

// createNestedDataClass creates a nested KotlinDataClass from a property schema
func createNestedDataClass(propSchema *plan.PropertySchema, propName string, parentClassName string, nameRegistry *core.NameRegistry) (*KotlinDataClass, error) {
	// Generate class name for the nested class
	nestedClassName := FormatClassName("", propName)
	mergedName := fmt.Sprintf("%s.%s", parentClassName, nestedClassName)

	dataClass, err := createDataClass(mergedName, propSchema.Property.Description, propSchema.Schema, nameRegistry)
	if err != nil {
		return nil, err
	}

	dataClass.Name = nestedClassName

	return dataClass, nil
}
