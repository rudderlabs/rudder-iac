package kotlin

import (
	_ "embed"
	"sort"

	"github.com/rudderlabs/rudder-iac/cli/internal/typer/generator/core"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer/plan"
)

func Generate(plan *plan.TrackingPlan) ([]*core.File, error) {
	ctx := NewKotlinContext()
	nameRegistry := core.NewNameRegistry(KotlinCollisionHandler)

	err := processCustomTypes(plan, ctx, nameRegistry)
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

// processCustomTypes extracts custom types from the tracking plan and enriches context with type aliases for primitive types
func processCustomTypes(p *plan.TrackingPlan, ctx *KotlinContext, nameRegistry *core.NameRegistry) error {
	customTypes := make(map[string]*plan.CustomType)

	// Extract all custom types from event rules
	for _, rule := range p.Rules {
		extractCustomTypesFromSchema(rule.Schema, customTypes)
	}

	// Sort custom type names for deterministic output
	var sortedNames []string
	for name := range customTypes {
		sortedNames = append(sortedNames, name)
	}
	sort.Strings(sortedNames)

	// Generate type aliases for primitive custom types in sorted order
	for _, name := range sortedNames {
		customType := customTypes[name]
		if customType.IsPrimitive() {
			alias, err := createCustomTypeTypeAlias(customType, nameRegistry)
			if err != nil {
				return err
			}
			ctx.TypeAliases = append(ctx.TypeAliases, *alias)
		}
	}

	return nil
}

// extractCustomTypesFromSchema recursively extracts custom types from an ObjectSchema
func extractCustomTypesFromSchema(schema plan.ObjectSchema, customTypes map[string]*plan.CustomType) {
	for _, propSchema := range schema.Properties {
		if propSchema.Property.IsCustomType() {
			customType := propSchema.Property.CustomType()
			if customType != nil {
				customTypes[customType.Name] = customType
			}
		}

		// Recursively process nested schemas
		if propSchema.Schema != nil {
			extractCustomTypesFromSchema(*propSchema.Schema, customTypes)
		}
	}
}

// createCustomTypeTypeAlias creates a KotlinTypeAlias from a primitive custom type
func createCustomTypeTypeAlias(customType *plan.CustomType, nameRegistry *core.NameRegistry) (*KotlinTypeAlias, error) {
	aliasName := FormatClassName("CustomType", customType.Name)

	// Register the name to handle collisions
	finalName, err := nameRegistry.RegisterName("customtype:"+customType.Name, "typealias", aliasName)
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
