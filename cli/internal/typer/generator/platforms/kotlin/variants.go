package kotlin

import (
	"fmt"
	"maps"
	"sort"

	"github.com/rudderlabs/rudder-iac/cli/internal/typer/generator/core"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer/plan"
)

// formatDiscriminatorValue formats a discriminator value based on its runtime type
// For multi-type properties, wraps the value in the appropriate sealed class constructor
func formatDiscriminatorValue(value any, property *plan.Property, kotlinType string) string {
	// Check if this is a multi-type property
	if len(property.Types) > 1 {
		return formatMultiTypeDiscriminatorValue(kotlinType, value)
	}

	// Single-type property - format based on value's runtime type
	switch v := value.(type) {
	case bool:
		// Format as boolean literal
		return fmt.Sprintf("%v", v)
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		// Format as integer literal
		return fmt.Sprintf("%v", v)
	case float32, float64:
		// Format as float literal
		return fmt.Sprintf("%v", v)
	case string:
		// Format as quoted string
		return fmt.Sprintf("%q", v)
	default:
		// For any other type, default to string formatting
		return fmt.Sprintf("%q", v)
	}
}

// formatMultiTypeDiscriminatorValue wraps a discriminator value in the appropriate
// sealed class constructor for multi-type properties
func formatMultiTypeDiscriminatorValue(kotlinType string, value any) string {
	var subclassName string
	var formattedValue string

	switch v := value.(type) {
	case bool:
		subclassName = "BooleanValue"
		formattedValue = fmt.Sprintf("%v", v)
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		subclassName = "IntegerValue"
		formattedValue = fmt.Sprintf("%v", v)
	case float32, float64:
		subclassName = "NumberValue"
		formattedValue = fmt.Sprintf("%v", v)
	case string:
		subclassName = "StringValue"
		formattedValue = fmt.Sprintf("%q", v)
	default:
		// Fallback to StringValue
		subclassName = "StringValue"
		formattedValue = fmt.Sprintf("%q", v)
	}

	return fmt.Sprintf("%s.%s(%s)", kotlinType, subclassName, formattedValue)
}

// createCustomTypeVariantSealedClass creates a sealed class for a custom type with variants
func createCustomTypeVariantSealedClass(
	customType *plan.CustomType,
	nameRegistry *core.NameRegistry,
) (*KotlinSealedClass, error) {
	className, err := getOrRegisterCustomTypeName(customType, nameRegistry)
	if err != nil {
		return nil, err
	}

	return createVariantSealedClass(
		className,
		customType.Description,
		customType.Schema,
		customType.Variants,
		nameRegistry,
	)
}

// createEventRuleVariantSealedClass creates a sealed class for an event rule with variants
func createEventRuleVariantSealedClass(
	rule *plan.EventRule,
	nameRegistry *core.NameRegistry,
) (*KotlinSealedClass, error) {
	className, err := getOrRegisterEventDataClassName(rule, nameRegistry)
	if err != nil {
		return nil, err
	}

	return createVariantSealedClass(
		className,
		rule.Event.Description,
		&rule.Schema,
		rule.Variants,
		nameRegistry,
	)
}

// createVariantSealedClass creates a sealed class from a variant definition
func createVariantSealedClass(
	name string,
	comment string,
	baseSchema *plan.ObjectSchema,
	variants []plan.Variant,
	nameRegistry *core.NameRegistry,
) (*KotlinSealedClass, error) {
	if len(variants) == 0 {
		return nil, fmt.Errorf("no variants provided")
	}

	// We currently support only one variant per type
	if len(variants) > 1 {
		return nil, fmt.Errorf("multiple variants per type are not supported; found %d variants", len(variants))
	}

	variant := variants[0]

	// Create abstract discriminator property
	var abstractProperties []KotlinProperty
	if discriminatorProp, exists := baseSchema.Properties[variant.Discriminator]; exists {
		kotlinType, err := getOrRegisterPropertyTypeTypeName(&discriminatorProp.Property, nameRegistry)
		if err != nil {
			return nil, err
		}

		abstractProperties = append(abstractProperties, KotlinProperty{
			Name:       FormatPropertyName(variant.Discriminator),
			SerialName: variant.Discriminator,
			Type:       kotlinType,
			Comment:    discriminatorProp.Property.Description,
			Nullable:   false,
			Abstract:   true,
		})
	}

	var subclasses []KotlinSealedSubclass

	// Create a subclass for each match value in each case
	for _, variantCase := range variant.Cases {
		for _, matchValue := range variantCase.Match {
			subclass, err := createSealedSubclass(
				matchValue,
				variant.Discriminator,
				baseSchema,
				&variantCase.Schema,
				variantCase.Description,
				nameRegistry,
			)
			if err != nil {
				return nil, err
			}
			subclasses = append(subclasses, *subclass)
		}
	}

	// Create default subclass if default schema exists
	if variant.DefaultSchema != nil {
		defaultSubclass, err := createSealedSubclass(
			nil,
			variant.Discriminator,
			baseSchema,
			variant.DefaultSchema,
			"Default case",
			nameRegistry,
		)
		if err != nil {
			return nil, err
		}
		subclasses = append(subclasses, *defaultSubclass)
	}

	return &KotlinSealedClass{
		Name:       name,
		Comment:    comment,
		Properties: abstractProperties,
		Subclasses: subclasses,
	}, nil
}

// createSealedSubclass creates a sealed subclass for a specific match value
func createSealedSubclass(
	matchValue any,
	discriminator string,
	baseSchema *plan.ObjectSchema,
	caseSchema *plan.ObjectSchema,
	comment string,
	nameRegistry *core.NameRegistry,
) (*KotlinSealedSubclass, error) {
	// Format subclass name from match value
	var subclassName string
	if matchValue == nil {
		subclassName = "Default"
	} else {
		subclassName = formatSealedSubclassName(matchValue)
	}

	// Merge base and case properties
	constructorProps, bodyProps, err := mergeVariantSchemaProperties(
		baseSchema,
		caseSchema,
		discriminator,
		matchValue,
		nameRegistry,
	)
	if err != nil {
		return nil, err
	}

	return &KotlinSealedSubclass{
		Name:           subclassName,
		Comment:        comment,
		Properties:     constructorProps,
		BodyProperties: bodyProps,
	}, nil
}

// mergeVariantSchemaProperties merges properties from base schema and case schema of a variant and a case.
// Depending on the presence of the discriminator property, it returns different sets of properties.
// constructorProperties are the properties that go in the constructor (non-override) of the subclass.
// bodyProperties are the properties that go in the body (override) of the subclass.
func mergeVariantSchemaProperties(
	baseSchema *plan.ObjectSchema,
	caseSchema *plan.ObjectSchema,
	discriminatorProp string,
	discriminatorValue any,
	nameRegistry *core.NameRegistry,
) ([]KotlinProperty, []KotlinProperty, error) {
	// Create merged property map
	merged := make(map[string]plan.PropertySchema)

	// Start with all base properties
	if baseSchema != nil {
		maps.Copy(merged, baseSchema.Properties)
	}

	// Add/override with case properties
	if caseSchema != nil {
		for name, casePropSchema := range caseSchema.Properties {
			if existing, exists := merged[name]; exists {
				// Property exists in both - use case's required status
				existing.Required = existing.Required || casePropSchema.Required
				merged[name] = existing
			} else {
				// New property from case
				merged[name] = casePropSchema
			}
		}
	}

	// Convert to sorted KotlinProperty array
	var sortedNames []string
	for name := range merged {
		sortedNames = append(sortedNames, name)
	}
	sort.Strings(sortedNames)

	var constructorProps []KotlinProperty
	var bodyProps []KotlinProperty

	for _, name := range sortedNames {
		propSchema := merged[name]

		// Special handling for discriminator property
		if name == discriminatorProp {
			kotlinType, err := getOrRegisterPropertyTypeTypeName(&propSchema.Property, nameRegistry)
			if err != nil {
				return nil, nil, err
			}

			var defaultValue string
			if discriminatorValue != nil {
				// Check if the property type is an enum
				if hasEnumConfig(propSchema.Property.Config) {
					// For enums, use the enum constant name (e.g., PropertyDeviceType.MOBILE)
					enumValueName := FormatEnumValue(discriminatorValue)
					defaultValue = fmt.Sprintf("%s.%s", kotlinType, enumValueName)
				} else {
					// For non-enum types, format based on the discriminator value's type
					// (handles both single-type and multi-type properties)
					defaultValue = formatDiscriminatorValue(discriminatorValue, &propSchema.Property, kotlinType)
				}
			}

			// Discriminator goes in the body as an override property (or constructor for Default case)
			discProp := KotlinProperty{
				Name:       FormatPropertyName(name),
				SerialName: name,
				Type:       kotlinType,
				Comment:    propSchema.Property.Description,
				Nullable:   false, // Discriminator is always required
				Default:    defaultValue,
				Override:   true, // Discriminator always overrides the abstract property
			}

			// For non-default cases (when discriminatorValue is set), put in body
			// For default case (when discriminatorValue is nil), put in constructor
			if discriminatorValue != nil {
				bodyProps = append(bodyProps, discProp)
			} else {
				constructorProps = append(constructorProps, discProp)
			}
		} else {
			// Regular property handling - always goes in constructor
			kotlinType, err := getOrRegisterPropertyTypeTypeName(&propSchema.Property, nameRegistry)
			if err != nil {
				return nil, nil, err
			}

			prop := KotlinProperty{
				Name:       FormatPropertyName(name),
				SerialName: name,
				Type:       kotlinType,
				Comment:    propSchema.Property.Description,
				Nullable:   !propSchema.Required,
			}

			if !propSchema.Required {
				prop.Default = "null"
			}

			constructorProps = append(constructorProps, prop)
		}
	}

	return constructorProps, bodyProps, nil
}
