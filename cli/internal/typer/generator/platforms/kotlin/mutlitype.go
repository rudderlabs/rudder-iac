package kotlin

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/typer/generator/core"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer/plan"
)

// createPropertyMultiTypeSealedClass creates a sealed class for a property with multiple types
func createPropertyMultiTypeSealedClass(property *plan.Property, nameRegistry *core.NameRegistry) (*KotlinSealedClass, error) {
	className, err := getOrRegisterPropertyMultiTypeClassName(property, nameRegistry)
	if err != nil {
		return nil, err
	}

	return createMultiTypeSealedClass(className, property.Description, property.Types, nameRegistry)
}

// createPropertyMultiTypeArrayItemSealedClass creates a sealed class for array items with multiple types
func createPropertyMultiTypeArrayItemSealedClass(property *plan.Property, nameRegistry *core.NameRegistry) (*KotlinSealedClass, error) {
	className, err := getOrRegisterPropertyMultiTypeArrayItemClassName(property, nameRegistry)
	if err != nil {
		return nil, err
	}

	comment := fmt.Sprintf("Item type for %s array", property.Name)
	return createMultiTypeSealedClass(className, comment, property.ItemTypes, nameRegistry)
}

func createMultiTypeSealedClass(className string, comment string, types []plan.PropertyType, nameRegistry *core.NameRegistry) (*KotlinSealedClass, error) {
	var subclasses []KotlinSealedSubclass
	for _, propertyType := range types {
		if plan.IsPrimitiveType(propertyType) {
			primitiveType := *plan.AsPrimitiveType(propertyType)
			subclass, err := createMultiTypeSubclass(primitiveType, nameRegistry)
			if err != nil {
				return nil, err
			}
			subclasses = append(subclasses, *subclass)
		} else {
			return nil, fmt.Errorf("unsupported property type in multi-type union: %T", propertyType)
		}
	}

	return &KotlinSealedClass{
		Name:       className,
		Comment:    comment,
		Properties: []KotlinProperty{}, // No shared properties
		Subclasses: subclasses,
	}, nil
}

// createMultiTypeSubclass creates a sealed class subclass for a specific type in a multi-type union
func createMultiTypeSubclass(primitiveType plan.PrimitiveType, nameRegistry *core.NameRegistry) (*KotlinSealedSubclass, error) {
	var typeName string
	switch primitiveType {
	case plan.PrimitiveTypeString:
		typeName = "StringValue"
	case plan.PrimitiveTypeInteger:
		typeName = "IntegerValue"
	case plan.PrimitiveTypeNumber:
		typeName = "NumberValue"
	case plan.PrimitiveTypeBoolean:
		typeName = "BooleanValue"
	case plan.PrimitiveTypeObject:
		typeName = "ObjectValue"
	case plan.PrimitiveTypeArray:
		typeName = "ArrayValue"
	default:
		return nil, fmt.Errorf("unsupported primitive type in multi-type union: %s", primitiveType)
	}

	var jsonConversion string
	switch primitiveType {
	case plan.PrimitiveTypeString:
		jsonConversion = "JsonPrimitive(value)"
	case plan.PrimitiveTypeInteger:
		jsonConversion = "JsonPrimitive(value)"
	case plan.PrimitiveTypeNumber:
		jsonConversion = "JsonPrimitive(value)"
	case plan.PrimitiveTypeBoolean:
		jsonConversion = "JsonPrimitive(value)"
	case plan.PrimitiveTypeObject:
		jsonConversion = "value" // JsonObject is already a JsonElement
	case plan.PrimitiveTypeArray:
		jsonConversion = "JsonArray(value)" // List needs to be wrapped in JsonArray
	default:
		return nil, fmt.Errorf("unsupported primitive type in multi-type union: %s", primitiveType)
	}

	kotlinType, err := mapPrimitiveToKotlinType(primitiveType)
	if err != nil {
		return nil, err
	}

	return &KotlinSealedSubclass{
		Name:    typeName,
		Comment: fmt.Sprintf("Represents a '%s' value", primitiveType),
		Properties: []KotlinProperty{
			{
				Name:       "value",
				SerialName: "value",
				Type:       kotlinType,
				Nullable:   false,
			},
		},
		BodyProperties: []KotlinProperty{
			{
				Name:       "_jsonElement",
				SerialName: "_jsonElement",
				Type:       "JsonElement",
				Default:    jsonConversion,
				Override:   true,
			},
		},
	}, nil
}
