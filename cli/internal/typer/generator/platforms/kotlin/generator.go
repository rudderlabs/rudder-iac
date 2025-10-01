package kotlin

import (
	_ "embed"
	"fmt"
	"sort"

	"github.com/rudderlabs/rudder-iac/cli/internal/typer/generator/core"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer/plan"
)

func Generate(plan *plan.TrackingPlan) ([]*core.File, error) {
	ctx := NewKotlinContext()
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

	// Generate type aliases for primitive custom types and data classes for object custom types
	for _, name := range sortedNames {
		customType := customTypes[name]
		if customType.IsPrimitive() {
			alias, err := createCustomTypeTypeAlias(customType, nameRegistry)
			if err != nil {
				return err
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
		if hasEnumConstraints(property) {
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

	// Map primitive type to Kotlin type
	kotlinType := mapPrimitiveToKotlinType(customType.Type)

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
	if len(property.Type) != 1 {
		return "", fmt.Errorf("only properties with a single type are supported: %v", property.Type)
	}

	var propertyType = property.Type[0]

	if plan.IsPrimitiveType(propertyType) {
		primitiveType := *plan.AsPrimitiveType(propertyType)

		// Handle array types by using ItemType
		if primitiveType == plan.PrimitiveTypeArray {
			if len(property.ItemType) != 1 {
				return "", fmt.Errorf("array properties must have exactly one item type: %v", property.ItemType)
			}

			itemType := property.ItemType[0]
			innerKotlinType, err := resolveTypeToKotlinType(itemType, nameRegistry)
			if err != nil {
				return "", err
			}

			return fmt.Sprintf("List<%s>", innerKotlinType), nil
		}

		return mapPrimitiveToKotlinType(primitiveType), nil
	} else if plan.IsCustomType(propertyType) {
		return resolveTypeToKotlinType(propertyType, nameRegistry)
	} else {
		return "", fmt.Errorf("unsupported property type: %s", property.Type)
	}
}

// resolveTypeToKotlinType resolves a PropertyType to its Kotlin type representation
func resolveTypeToKotlinType(propertyType plan.PropertyType, nameRegistry *core.NameRegistry) (string, error) {
	if plan.IsPrimitiveType(propertyType) {
		return mapPrimitiveToKotlinType(*plan.AsPrimitiveType(propertyType)), nil
	} else if plan.IsCustomType(propertyType) {
		customType := plan.AsCustomType(propertyType)
		if customType.IsPrimitive() {
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

// createKotlinPropertiesFromSchema processes properties from an ObjectSchema and returns KotlinProperty objects
func createKotlinPropertiesFromSchema(schema *plan.ObjectSchema, nameRegistry *core.NameRegistry) ([]KotlinProperty, error) {
	// Sort property names for deterministic output
	var sortedPropNames []string
	for propName := range schema.Properties {
		sortedPropNames = append(sortedPropNames, propName)
	}
	sort.Strings(sortedPropNames)

	var properties []KotlinProperty
	for _, propName := range sortedPropNames {
		propSchema := schema.Properties[propName]
		kotlinType, err := getPropertyKotlinType(propSchema.Property, nameRegistry)
		if err != nil {
			return nil, err
		}

		property := KotlinProperty{
			Name:       formatPropertyName(propName),
			SerialName: propName,
			Type:       kotlinType,
			Comment:    propSchema.Property.Description,
			Nullable:   !propSchema.Required,
		}

		if !propSchema.Required {
			property.Default = "null"
		}

		properties = append(properties, property)
	}

	return properties, nil
}

// createCustomTypeDataClass creates a KotlinDataClass from an object custom type
func createCustomTypeDataClass(customType *plan.CustomType, nameRegistry *core.NameRegistry) (*KotlinDataClass, error) {
	finalName, err := getOrRegisterCustomTypeClassName(customType, nameRegistry)
	if err != nil {
		return nil, err
	}

	properties, err := createKotlinPropertiesFromSchema(customType.Schema, nameRegistry)
	if err != nil {
		return nil, err
	}

	return &KotlinDataClass{
		Name:       finalName,
		Comment:    customType.Description,
		Properties: properties,
	}, nil
}

// getPropertyKotlinType returns the Kotlin type name for a property, using appropriate type aliases or enum classes
func getPropertyKotlinType(property plan.Property, nameRegistry *core.NameRegistry) (string, error) {
	// Check if this property has enum constraints
	if hasEnumConstraints(&property) {
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

	// Use the shared helper to create properties from the rule's schema
	properties, err := createKotlinPropertiesFromSchema(&rule.Schema, nameRegistry)
	if err != nil {
		return nil, err
	}

	return &KotlinDataClass{
		Name:       className,
		Comment:    rule.Event.Description,
		Properties: properties,
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
			Name:       FormatEnumValue(value),
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
	default:
		return "Any" // Fallback for unknown types
	}
}
