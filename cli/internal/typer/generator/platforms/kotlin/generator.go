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

// processPropertiesIntoContext processes individual properties and creates type aliases for all properties
func processPropertiesIntoContext(allProperties map[string]*plan.Property, ctx *KotlinContext, nameRegistry *core.NameRegistry) error {
	// Sort property names for deterministic output
	var sortedNames []string
	for name := range allProperties {
		sortedNames = append(sortedNames, name)
	}
	sort.Strings(sortedNames)

	// Generate type aliases for all properties
	for _, name := range sortedNames {
		property := allProperties[name]
		alias, err := createPropertyTypeAlias(property, nameRegistry)
		if err != nil {
			return err
		}
		ctx.TypeAliases = append(ctx.TypeAliases, *alias)
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
	var kotlinType string
	if property.IsPrimitive() {
		kotlinType = mapPrimitiveToKotlinType(property.PrimitiveType())
	} else if property.IsCustomType() {
		customType := property.CustomType()
		if customType.IsPrimitive() {
			// Reference the custom type alias
			kotlinType, err = getOrRegisterCustomTypeAliasName(customType, nameRegistry)
			if err != nil {
				return nil, err
			}
		} else {
			// Reference the custom type data class
			kotlinType, err = getOrRegisterCustomTypeClassName(customType, nameRegistry)
			if err != nil {
				return nil, err
			}
		}
	} else {
		return nil, fmt.Errorf("unsupported property type: %s", property.Type)
	}

	return &KotlinTypeAlias{
		Alias:   finalName,
		Comment: property.Description,
		Type:    kotlinType,
	}, nil
}

// createCustomTypeDataClass creates a KotlinDataClass from an object custom type
func createCustomTypeDataClass(customType *plan.CustomType, nameRegistry *core.NameRegistry) (*KotlinDataClass, error) {
	finalName, err := getOrRegisterCustomTypeClassName(customType, nameRegistry)
	if err != nil {
		return nil, err
	}

	// Process properties from the custom type's schema in sorted order
	var sortedPropNames []string
	for propName := range customType.Schema.Properties {
		sortedPropNames = append(sortedPropNames, propName)
	}
	sort.Strings(sortedPropNames)

	var properties []KotlinProperty
	for _, propName := range sortedPropNames {
		propSchema := customType.Schema.Properties[propName]
		kotlinType, err := getPropertyKotlinType(propSchema.Property, nameRegistry)
		if err != nil {
			return nil, err
		}

		property := KotlinProperty{
			Name:         formatPropertyName(propName),
			OriginalName: propName,
			Type:         kotlinType,
			Comment:      propSchema.Property.Description,
			Optional:     !propSchema.Required,
		}
		properties = append(properties, property)
	}

	return &KotlinDataClass{
		Name:       finalName,
		Comment:    customType.Description,
		Properties: properties,
	}, nil
}

// getPropertyKotlinType returns the Kotlin type name for a property, using appropriate type aliases
func getPropertyKotlinType(property plan.Property, nameRegistry *core.NameRegistry) (string, error) {
	return getOrRegisterPropertyAliasName(&property, nameRegistry)
}

// mapPrimitiveToKotlinType maps plan primitive types to Kotlin types
func mapPrimitiveToKotlinType(primitiveType plan.PrimitiveType) string {
	switch primitiveType {
	case plan.PrimitiveTypeString:
		return "String"
	case plan.PrimitiveTypeNumber:
		return "Double"
	case plan.PrimitiveTypeBoolean:
		return "Boolean"
	case plan.PrimitiveTypeDate:
		return "String" // For now, represent dates as strings
	case plan.PrimitiveTypeArray:
		return "List<Any>" // Generic array type for now
	default:
		return "Any" // Fallback for unknown types
	}
}
