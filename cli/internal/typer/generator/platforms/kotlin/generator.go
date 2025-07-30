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
			Optional:   !propSchema.Required,
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

// getPropertyKotlinType returns the Kotlin type name for a property, using appropriate type aliases
func getPropertyKotlinType(property plan.Property, nameRegistry *core.NameRegistry) (string, error) {
	return getOrRegisterPropertyAliasName(&property, nameRegistry)
}

// processEventRules processes event rules and generates data classes for event properties/traits
func processEventRules(p *plan.TrackingPlan, ctx *KotlinContext, nameRegistry *core.NameRegistry) error {
	// Map rules by a unique composite key for deterministic processing
	ruleMap := make(map[string]plan.EventRule)
	for _, rule := range p.Rules {
		// Create a unique key using event type, name, and section
		key := string(rule.Event.EventType) + ":" + rule.Event.Name + ":" + string(rule.Section)
		ruleMap[key] = rule
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
		dataClass, err := createEventDataClass(rule, nameRegistry)
		if err != nil {
			return err
		}
		ctx.DataClasses = append(ctx.DataClasses, *dataClass)
	}

	return nil
}

// createEventDataClass creates a KotlinDataClass from an event rule
func createEventDataClass(rule plan.EventRule, nameRegistry *core.NameRegistry) (*KotlinDataClass, error) {
	// Generate class name based on event type, name, and section
	var prefix string
	var baseName string

	if rule.Event.EventType == plan.EventTypeTrack {
		// For track events: add "Track" prefix and use the event name
		prefix = "Track"
		baseName = rule.Event.Name
	} else {
		// For other events: use the type as prefix (capitalized) and skip the name
		switch rule.Event.EventType {
		case plan.EventTypeIdentify:
			prefix = "Identify"
		case plan.EventTypePage:
			prefix = "Page"
		case plan.EventTypeScreen:
			prefix = "Screen"
		case plan.EventTypeGroup:
			prefix = "Group"
		default:
			return nil, fmt.Errorf("unsupported event type: %s", rule.Event.EventType)
		}
		baseName = ""
	}

	var suffix string
	switch rule.Section {
	case plan.EventRuleSectionProperties:
		suffix = "Properties"
	case plan.EventRuleSectionTraits:
		suffix = "Traits"
	default:
		return nil, fmt.Errorf("unsupported event rule section: %s", rule.Section)
	}

	className := FormatClassName("", prefix+baseName+suffix)

	// Register the class name with a unique key
	registrationKey := "event:" + string(rule.Event.EventType) + ":" + rule.Event.Name
	finalName, err := nameRegistry.RegisterName(
		registrationKey,
		"dataclass",
		className,
	)
	if err != nil {
		return nil, err
	}

	// Use the shared helper to create properties from the rule's schema
	properties, err := createKotlinPropertiesFromSchema(&rule.Schema, nameRegistry)
	if err != nil {
		return nil, err
	}

	return &KotlinDataClass{
		Name:       finalName,
		Comment:    rule.Event.Description,
		Properties: properties,
	}, nil
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
