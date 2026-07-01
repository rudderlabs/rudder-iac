package typescript

import (
	"fmt"
	"sort"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/typer/generator/core"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer/plan"
)

func processCustomTypeVariant(ct *plan.CustomType, ctx *TSContext, nr *core.NameRegistry) error {
	typeName, err := getOrRegisterCustomTypeName(ct, nr)
	if err != nil {
		return err
	}

	if len(ct.Variants) != 1 {
		return fmt.Errorf("expected exactly one variant for custom type %q, got %d", ct.Name, len(ct.Variants))
	}

	group, err := buildVariantGroup(typeName, ct.Description, ct.Schema, &ct.Variants[0], nr)
	if err != nil {
		return err
	}
	ctx.VariantTypes = append(ctx.VariantTypes, *group)
	return nil
}

func processEventRuleVariant(rule *plan.EventRule, ctx *TSContext, nr *core.NameRegistry) error {
	interfaceName, err := getOrRegisterEventInterfaceName(rule, nr)
	if err != nil {
		return err
	}

	if len(rule.Variants) != 1 {
		return fmt.Errorf("expected exactly one variant for event %q, got %d", rule.Event.Name, len(rule.Variants))
	}

	group, err := buildVariantGroup(interfaceName, rule.Event.Description, &rule.Schema, &rule.Variants[0], nr)
	if err != nil {
		return err
	}
	ctx.VariantTypes = append(ctx.VariantTypes, *group)

	method, err := buildAnalyticsMethod(rule, ctx, nr)
	if err != nil {
		return err
	}
	if method != nil {
		ctx.AnalyticsMethods = append(ctx.AnalyticsMethods, *method)
	}
	return nil
}

func buildVariantGroup(
	parentName string,
	comment string,
	baseSchema *plan.ObjectSchema,
	variant *plan.Variant,
	nr *core.NameRegistry,
) (*TSVariantGroup, error) {
	var caseInterfaces []TSInterface
	var unionParts []string
	var coveredValues []any

	for _, vc := range variant.Cases {
		for _, matchValue := range vc.Match {
			coveredValues = append(coveredValues, matchValue)

			caseName := formatVariantCaseName(parentName, matchValue)
			registered, err := nr.RegisterName(
				"variant:"+parentName+":case:"+fmt.Sprintf("%v", matchValue),
				globalTypeScope, caseName,
			)
			if err != nil {
				return nil, err
			}

			iface, err := buildVariantInterface(
				registered, vc.Description, baseSchema, &vc.Schema,
				variant.Discriminator, matchValue, nil, nr,
			)
			if err != nil {
				return nil, err
			}
			caseInterfaces = append(caseInterfaces, *iface)
			unionParts = append(unionParts, registered)
		}
	}

	defaultName := FormatTypeName(parentName, "Default")
	registeredDefault, err := nr.RegisterName(
		"variant:"+parentName+":default", globalTypeScope, defaultName,
	)
	if err != nil {
		return nil, err
	}

	defaultSchema := variant.DefaultSchema
	if defaultSchema == nil {
		defaultSchema = &plan.ObjectSchema{
			Properties:           map[string]plan.PropertySchema{},
			AdditionalProperties: false,
		}
	}

	defaultIface, err := buildVariantInterface(
		registeredDefault, "Default case", baseSchema, defaultSchema,
		variant.Discriminator, nil, coveredValues, nr,
	)
	if err != nil {
		return nil, err
	}
	caseInterfaces = append(caseInterfaces, *defaultIface)
	unionParts = append(unionParts, registeredDefault)

	sort.Slice(caseInterfaces, func(i, j int) bool {
		return caseInterfaces[i].Name < caseInterfaces[j].Name
	})
	sort.Strings(unionParts)

	return &TSVariantGroup{
		CaseInterfaces: caseInterfaces,
		UnionAlias: TSTypeAlias{
			Alias:   parentName,
			Type:    strings.Join(unionParts, " | "),
			Comment: comment,
		},
	}, nil
}

// buildVariantInterface builds one case (or default) interface by merging the
// base schema with the case/default schema. When discriminatorValue is non-nil
// (a named case) the discriminator property is pinned to that TS literal. When
// nil (the default case) the discriminator is narrowed to
// `Exclude<T, ...covered literals>` so the Default member stays structurally
// disjoint from every named case — without this, narrowing on the
// discriminator (in an `if` or a `switch`) cannot isolate a named case. For an
// exhaustively-covered discriminator the Exclude collapses to `never`,
// correctly marking Default unreachable. coveredValues is every named case's
// match value and is consulted only for the default case.
func buildVariantInterface(
	name string,
	comment string,
	baseSchema *plan.ObjectSchema,
	caseSchema *plan.ObjectSchema,
	discriminator string,
	discriminatorValue any,
	coveredValues []any,
	nr *core.NameRegistry,
) (*TSInterface, error) {
	merged := mergeSchemaProperties(baseSchema, caseSchema)

	sortedNames := make([]string, 0, len(merged))
	for propName := range merged {
		sortedNames = append(sortedNames, propName)
	}
	sort.Strings(sortedNames)

	properties := make([]TSInterfaceProperty, 0, len(sortedNames))
	for _, propName := range sortedNames {
		propSchema := merged[propName]

		fieldName, quoted, err := getOrRegisterInterfacePropertyName(name, propName, nr)
		if err != nil {
			return nil, err
		}

		isDiscriminator := propName == discriminator

		var tsType string
		if isDiscriminator && discriminatorValue != nil {
			tsType = FormatTSLiteral(discriminatorValue)
		} else {
			var enumName string
			if hasEnumConfig(propSchema.Property.Config) {
				enumName, err = getOrRegisterPropertyEnumName(&propSchema.Property, nr)
				if err != nil {
					return nil, err
				}
			}
			tsType, err = resolvePropertyType(&propSchema.Property, enumName, "", nr)
			if err != nil {
				return nil, fmt.Errorf("resolving type for property %q in variant interface %q: %w", propName, name, err)
			}
			// Default-case discriminator: subtract every value a named case
			// already covers so this member stays disjoint and narrows cleanly.
			if isDiscriminator && len(coveredValues) > 0 {
				tsType = formatExcludeType(tsType, coveredValues)
			}
		}

		// A discriminator must be required for narrowing to work — in both
		// named and default cases.
		optional := !propSchema.Required
		if isDiscriminator {
			optional = false
		}

		properties = append(properties, TSInterfaceProperty{
			Name:       fieldName,
			Type:       tsType,
			Comment:    propSchema.Property.Description,
			Optional:   optional,
			QuotedName: quoted,
			SerialName: propName,
		})
	}

	return &TSInterface{
		Name:       name,
		Comment:    comment,
		Properties: properties,
	}, nil
}

func mergeSchemaProperties(baseSchema, caseSchema *plan.ObjectSchema) map[string]plan.PropertySchema {
	merged := make(map[string]plan.PropertySchema)
	if baseSchema != nil {
		for name, ps := range baseSchema.Properties {
			merged[name] = ps
		}
	}
	if caseSchema != nil {
		for name, ps := range caseSchema.Properties {
			if existing, exists := merged[name]; exists {
				existing.Required = existing.Required || ps.Required
				merged[name] = existing
			} else {
				merged[name] = ps
			}
		}
	}
	return merged
}

// formatExcludeType wraps base in `Exclude<base, v1 | v2 | ...>`, formatting
// each value as a TS literal and dropping duplicates. Used to narrow a
// variant's default-case discriminator to the values no named case covers.
func formatExcludeType(base string, values []any) string {
	seen := make(map[string]bool, len(values))
	lits := make([]string, 0, len(values))
	for _, v := range values {
		lit := FormatTSLiteral(v)
		if seen[lit] {
			continue
		}
		seen[lit] = true
		lits = append(lits, lit)
	}
	return fmt.Sprintf("Exclude<%s, %s>", base, strings.Join(lits, " | "))
}

func formatVariantCaseName(parentName string, matchValue any) string {
	matchStr := formatMatchValueForName(matchValue)
	return FormatTypeName(parentName, "Case "+matchStr)
}

func formatMatchValueForName(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case bool:
		if v {
			return "True"
		}
		return "False"
	default:
		return fmt.Sprintf("%v", v)
	}
}
