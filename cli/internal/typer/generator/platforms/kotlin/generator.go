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

type Generator struct{}

// Generate produces Kotlin code files from a tracking plan
func (k *Generator) Generate(plan *plan.TrackingPlan, options core.GenerateOptions, platformOptions any) ([]*core.File, error) {
	var defaults KotlinOptions = k.DefaultOptions().(KotlinOptions)
	var kotlinOptions KotlinOptions = defaults
	if platformOptions != nil {
		kotlinOptions = platformOptions.(KotlinOptions)
	}

	if err := kotlinOptions.Validate(); err != nil {
		return nil, err
	}

	ctx := NewKotlinContext()

	ctx.PackageName = kotlinOptions.PackageName
	if ctx.PackageName == "" {
		ctx.PackageName = defaults.PackageName
	}

	ctx.RudderCLIVersion = options.RudderCLIVersion
	ctx.EventContext = formatEventContext(plan.Metadata, options.RudderCLIVersion)
	ctx.TrackingPlanName = plan.Name
	ctx.TrackingPlanID = plan.Metadata.TrackingPlanID
	ctx.TrackingPlanVersion = plan.Metadata.TrackingPlanVersion
	ctx.TrackingPlanURL = plan.Metadata.URL
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

		// Check if this custom type has variants
		if len(customType.Variants) > 0 {
			// Generate sealed class for variant custom type
			sealedClass, err := createCustomTypeVariantSealedClass(customType, nameRegistry, ctx.PackageName)
			if err != nil {
				return err
			}
			ctx.SealedClasses = append(ctx.SealedClasses, *sealedClass)
		} else if customType.IsPrimitive() {
			// Check if this custom type has enum constraints
			if hasEnumConfig(customType.Config) {
				enum, err := createCustomTypeEnum(customType, nameRegistry)
				if err != nil {
					return err
				}
				ctx.Enums = append(ctx.Enums, *enum)
			} else {
				alias, err := createCustomTypeTypeAlias(customType, nameRegistry, ctx.PackageName)
				if err != nil {
					return err
				}
				ctx.TypeAliases = append(ctx.TypeAliases, *alias)
			}
		} else if customType.Schema != nil && len(customType.Schema.Properties) == 0 && customType.Schema.AdditionalProperties {
			// If this is an empty object with additionalProperties: true
			// create a type alias to JsonObject instead of an empty data class
			finalName, err := getOrRegisterCustomTypeName(customType, nameRegistry)
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
			dataClass, err := createCustomTypeDataClass(customType, nameRegistry, ctx.PackageName)
			if err != nil {
				return err
			}
			ctx.DataClasses = append(ctx.DataClasses, *dataClass)
		}
	}
	return nil
}

// processPropertiesIntoContext processes individual properties and creates type aliases, enums, or sealed classes for all properties
func processPropertiesIntoContext(allProperties map[string]*plan.Property, ctx *KotlinContext, nameRegistry *core.NameRegistry) error {
	// Sort property names for deterministic output
	var sortedNames []string
	for name := range allProperties {
		sortedNames = append(sortedNames, name)
	}
	sort.Strings(sortedNames)

	// Generate type aliases, enums, or sealed classes for all properties
	for _, name := range sortedNames {
		property := allProperties[name]

		// Check if this property has enum constraints
		if hasEnumConfig(property.Config) {
			enum, err := createPropertyEnum(property, nameRegistry)
			if err != nil {
				return err
			}
			ctx.Enums = append(ctx.Enums, *enum)
		} else if len(property.Types) > 1 {
			// Multi-type property - generate sealed class
			sealedClass, err := createPropertyMultiTypeSealedClass(property, nameRegistry)
			if err != nil {
				return err
			}
			ctx.SealedClasses = append(ctx.SealedClasses, *sealedClass)
		} else if len(property.Types) == 1 && plan.IsPrimitiveType(property.Types[0]) && *plan.AsPrimitiveType(property.Types[0]) == plan.PrimitiveTypeArray && len(property.ItemTypes) > 1 {
			// Array with multiple item types - generate sealed class for array items
			sealedClass, err := createPropertyMultiTypeArrayItemSealedClass(property, nameRegistry)
			if err != nil {
				return err
			}
			ctx.SealedClasses = append(ctx.SealedClasses, *sealedClass)
			// Also create type alias for the array itself
			alias, err := createPropertyTypeAlias(property, nameRegistry, ctx.PackageName)
			if err != nil {
				return err
			}
			ctx.TypeAliases = append(ctx.TypeAliases, *alias)
		} else {
			alias, err := createPropertyTypeAlias(property, nameRegistry, ctx.PackageName)
			if err != nil {
				return err
			}
			ctx.TypeAliases = append(ctx.TypeAliases, *alias)
		}
	}
	return nil
}

// createCustomTypeTypeAlias creates a KotlinTypeAlias from a primitive custom type
func createCustomTypeTypeAlias(customType *plan.CustomType, nameRegistry *core.NameRegistry, packageName string) (*KotlinTypeAlias, error) {
	finalName, err := getOrRegisterCustomTypeName(customType, nameRegistry)
	if err != nil {
		return nil, err
	}

	var kotlinType string
	// Handle array custom types with ItemType
	if customType.Type == plan.PrimitiveTypeArray {
		if customType.ItemType != nil {
			innerKotlinType, err := resolveTypeToKotlinType(customType.ItemType, nameRegistry, packageName)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve item type for custom type %q: %w", customType.Name, err)
			}
			kotlinType = fmt.Sprintf("List<%s>", innerKotlinType)
		} else {
			// Array with no item type means array of any
			kotlinType = "List<JsonElement>"
		}
	} else if customType.Type == plan.PrimitiveTypeObject && isEmptySchema(customType.Schema) {
		if customType.Schema.AdditionalProperties {
			// Empty object schema with additionalProperties means JsonObject
			kotlinType = "JsonObject"
		} else {
			// Empty object schema without additionalProperties means Unit (no data)
			kotlinType = "Unit"
		}
	} else {
		// Map primitive type to Kotlin type
		kotlinType, err = mapPrimitiveToKotlinType(customType.Type)
		if err != nil {
			return nil, fmt.Errorf("failed to map custom type %q to Kotlin type: %w", customType.Name, err)
		}
	}

	return &KotlinTypeAlias{
		Alias:   finalName,
		Comment: customType.Description,
		Type:    kotlinType,
	}, nil
}

// createPropertyTypeAlias creates a KotlinTypeAlias from any property
func createPropertyTypeAlias(property *plan.Property, nameRegistry *core.NameRegistry, packageName string) (*KotlinTypeAlias, error) {
	finalName, err := getOrRegisterPropertyTypeName(property, nameRegistry)
	if err != nil {
		return nil, err
	}

	// Get the appropriate Kotlin type for this property
	kotlinType, err := resolvePropertyKotlinType(property, nameRegistry, packageName)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve Kotlin type for property %q: %w", property.Name, err)
	}

	return &KotlinTypeAlias{
		Alias:   finalName,
		Comment: property.Description,
		Type:    kotlinType,
	}, nil
}

// resolvePropertyKotlinType resolves the Kotlin type for a property, handling arrays and multi-type properties
func resolvePropertyKotlinType(property *plan.Property, nameRegistry *core.NameRegistry, packageName string) (string, error) {
	// no types means any type, which maps to JsonElement for a flexible representation
	if len(property.Types) == 0 {
		return "JsonElement", nil
	} else if len(property.Types) == 1 {
		// Single type property
		propertyType := property.Types[0]

		if plan.IsPrimitiveType(propertyType) {
			primitiveType := *plan.AsPrimitiveType(propertyType)

			// Handle array types by using ItemType
			if primitiveType == plan.PrimitiveTypeArray {
				if len(property.ItemTypes) == 0 {
					// No item type specified means array can contain any type
					return "List<JsonElement>", nil
				} else if len(property.ItemTypes) == 1 {
					itemType := property.ItemTypes[0]
					innerKotlinType, err := resolveTypeToKotlinType(itemType, nameRegistry, packageName)
					if err != nil {
						return "", fmt.Errorf("failed to resolve item type for property %q: %w", property.Name, err)
					}
					return fmt.Sprintf("List<%s>", innerKotlinType), nil
				} else {
					// Multi-type array items - reference the sealed class for array items
					itemClassName, err := getOrRegisterPropertyArrayItemTypeName(property, nameRegistry)
					if err != nil {
						return "", err
					}
					return fmt.Sprintf("List<%s>", fmt.Sprintf("%s.%s", packageName, itemClassName)), nil
				}
			}

			return mapPrimitiveToKotlinType(primitiveType)
		} else if plan.IsCustomType(propertyType) {
			return resolveTypeToKotlinType(propertyType, nameRegistry, packageName)
		} else {
			return "", fmt.Errorf("unsupported property type: %s", property.Types)
		}
	} else {
		// Multi-type property - reference the sealed class
		itemClassName, err := getOrRegisterPropertyArrayItemTypeName(property, nameRegistry)
		if err != nil {
			return "", err
		}

		return fmt.Sprintf("List<%s>", fmt.Sprintf("%s.%s", packageName, itemClassName)), nil
	}
}

// resolveTypeToKotlinType resolves a PropertyType to its Kotlin type representation
func resolveTypeToKotlinType(propertyType plan.PropertyType, nameRegistry *core.NameRegistry, packageName string) (string, error) {
	if plan.IsPrimitiveType(propertyType) {
		return mapPrimitiveToKotlinType(*plan.AsPrimitiveType(propertyType))
	} else if plan.IsCustomType(propertyType) {
		customType := plan.AsCustomType(propertyType)
		simpleName, err := getOrRegisterCustomTypeName(customType, nameRegistry)
		if err != nil {
			return "", err
		}
		// Return fully qualified type name
		return fmt.Sprintf("%s.%s", packageName, simpleName), nil
	} else {
		return "", fmt.Errorf("unsupported property type: %T", propertyType)
	}
}

func createDataClass(className string, comment string, schema *plan.ObjectSchema, nameRegistry *core.NameRegistry, packageName string) (*KotlinDataClass, error) {
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
			// Check if this is an empty object
			if len(propSchema.Schema.Properties) == 0 {
				var kotlinType string
				if propSchema.Schema.AdditionalProperties {
					// Use the property's type alias instead of creating an empty data class
					kt, err := getOrRegisterPropertyTypeName(&propSchema.Property, nameRegistry)
					kotlinType = fmt.Sprintf("%s.%s", packageName, kt)
					if err != nil {
						return nil, err
					}
				} else {
					kotlinType = "Unit"
				}

				property = &KotlinProperty{
					Name:       FormatPropertyName(propName),
					SerialName: propName,
					Type:       kotlinType,
					Comment:    propSchema.Property.Description,
					Nullable:   !propSchema.Required,
				}
			} else {
				// Generate nested data class
				nestedClass, err := createNestedDataClass(&propSchema, propName, className, nameRegistry, packageName)
				if err != nil {
					return nil, err
				}
				nestedClasses = append(nestedClasses, *nestedClass)

				// Property type references the nested class (fully qualified)
				nestedClassName := fmt.Sprintf("%s.%s.%s", packageName, className, nestedClass.Name)
				property = &KotlinProperty{
					Name:       FormatPropertyName(propName),
					SerialName: propName,
					Type:       nestedClassName,
					Comment:    propSchema.Property.Description,
					Nullable:   !propSchema.Required,
				}
			}
		} else {
			kotlinType, err := getOrRegisterPropertyTypeName(&propSchema.Property, nameRegistry)
			if err != nil {
				return nil, err
			}

			property = &KotlinProperty{
				Name:       FormatPropertyName(propName),
				SerialName: propName,
				Type:       fmt.Sprintf("%s.%s", packageName, kotlinType),
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
func createCustomTypeDataClass(customType *plan.CustomType, nameRegistry *core.NameRegistry, packageName string) (*KotlinDataClass, error) {
	finalName, err := getOrRegisterCustomTypeName(customType, nameRegistry)
	if err != nil {
		return nil, err
	}

	return createDataClass(finalName, customType.Description, customType.Schema, nameRegistry, packageName)
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

		// Check if this event rule has variants
		if len(rule.Variants) > 0 {
			// Generate sealed class for variant event
			sealedClass, err := createEventRuleVariantSealedClass(rule, nameRegistry, ctx.PackageName)
			if err != nil {
				return err
			}
			ctx.SealedClasses = append(ctx.SealedClasses, *sealedClass)
		} else {
			// Check if schema is empty
			if isEmptySchema(&rule.Schema) {
				// Empty with additionalProperties - create type alias to JsonObject, otherwise skip
				if rule.Schema.AdditionalProperties {
					typeAlias, err := createEventSchemaTypeAlias(rule, nameRegistry)
					if err != nil {
						return err
					}
					ctx.TypeAliases = append(ctx.TypeAliases, *typeAlias)
				}
			} else {
				// create data class for the event rule
				dataClass, err := createEventDataClass(rule, nameRegistry, ctx.PackageName)
				if err != nil {
					return err
				}
				ctx.DataClasses = append(ctx.DataClasses, *dataClass)
			}
		}

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
func createEventDataClass(rule *plan.EventRule, nameRegistry *core.NameRegistry, packageName string) (*KotlinDataClass, error) {
	className, err := getOrRegisterEventDataClassName(rule, nameRegistry)
	if err != nil {
		return nil, err
	}

	return createDataClass(className, rule.Event.Description, &rule.Schema, nameRegistry, packageName)
}

// hasEnumConfig checks if a PropertyConfig has enum constraints defined
func hasEnumConfig(config *plan.PropertyConfig) bool {
	return config != nil && config.Enum != nil && len(config.Enum) > 0
}

// createPropertyEnum creates a KotlinEnum from a property with enum constraints
func createPropertyEnum(property *plan.Property, nameRegistry *core.NameRegistry) (*KotlinEnum, error) {
	enumName, err := getOrRegisterPropertyTypeName(property, nameRegistry)
	if err != nil {
		return nil, err
	}

	// Convert enum values to KotlinEnumValue structs
	var enumValues []KotlinEnumValue
	for _, value := range property.Config.Enum {
		registeredName, err := getOrRegisterEnumValue(enumName, value, nameRegistry)
		if err != nil {
			return nil, err
		}

		enumValues = append(enumValues, KotlinEnumValue{
			Name:  registeredName,
			Value: value,
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
	enumName, err := getOrRegisterCustomTypeName(customType, nameRegistry)
	if err != nil {
		return nil, err
	}

	// Convert enum values to KotlinEnumValue structs
	var enumValues []KotlinEnumValue
	for _, value := range customType.Config.Enum {
		registeredName, err := getOrRegisterEnumValue(enumName, value, nameRegistry)
		if err != nil {
			return nil, err
		}

		enumValues = append(enumValues, KotlinEnumValue{
			Name:  registeredName,
			Value: value,
		})
	}

	return &KotlinEnum{
		Name:    enumName,
		Comment: customType.Description,
		Values:  enumValues,
	}, nil
}

// mapPrimitiveToKotlinType maps plan primitive types to Kotlin types
func mapPrimitiveToKotlinType(primitiveType plan.PrimitiveType) (string, error) {
	switch primitiveType {
	case plan.PrimitiveTypeString:
		return "String", nil
	case plan.PrimitiveTypeInteger:
		return "Long", nil
	case plan.PrimitiveTypeNumber:
		return "Double", nil
	case plan.PrimitiveTypeBoolean:
		return "Boolean", nil
	case plan.PrimitiveTypeObject:
		return "JsonObject", nil
	case plan.PrimitiveTypeArray:
		return "List<JsonElement>", nil
	case plan.PrimitiveTypeNull:
		return "JsonNull", nil
	default:
		return "", fmt.Errorf("unsupported primitive type: %s", primitiveType)
	}
}

// createNestedDataClass creates a nested KotlinDataClass from a property schema
func createNestedDataClass(propSchema *plan.PropertySchema, propName string, parentClassName string, nameRegistry *core.NameRegistry, packageName string) (*KotlinDataClass, error) {
	// Generate class name for the nested class
	nestedClassName := FormatClassName("", propName)
	mergedName := fmt.Sprintf("%s.%s", parentClassName, nestedClassName)

	dataClass, err := createDataClass(mergedName, propSchema.Property.Description, propSchema.Schema, nameRegistry, packageName)
	if err != nil {
		return nil, err
	}

	dataClass.Name = nestedClassName

	return dataClass, nil
}

// isEmptySchema checks if an ObjectSchema has no defined properties
func isEmptySchema(schema *plan.ObjectSchema) bool {
	return schema != nil && len(schema.Properties) == 0
}

// createEventSchemaTypeAlias creates a type alias to JsonObject for an empty event schema
func createEventSchemaTypeAlias(rule *plan.EventRule, nameRegistry *core.NameRegistry) (*KotlinTypeAlias, error) {
	aliasName, err := getOrRegisterEventDataClassName(rule, nameRegistry)
	if err != nil {
		return nil, err
	}

	return &KotlinTypeAlias{
		Alias:   aliasName,
		Comment: rule.Event.Description,
		Type:    "JsonObject",
	}, nil
}
