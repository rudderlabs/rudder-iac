package swift

import (
	"fmt"
	"maps"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/typer/generator/core"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer/plan"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
)

const Platform = "swift"

// Generator implements core.Generator for the Swift platform.
type Generator struct{}

const (
	globalTypeScope = "types"
	methodScope     = "methods"
)

// ========== Main Entry Point ==========

func (g *Generator) Generate(p *plan.TrackingPlan, opts core.GenerateOptions, platformOpts any) ([]*core.File, error) {
	defaults := g.DefaultOptions().(SwiftOptions)
	swiftOpts := defaults
	if platformOpts != nil {
		swiftOpts = platformOpts.(SwiftOptions)
	}

	outputFileName := swiftOpts.OutputFileName
	if outputFileName == "" {
		outputFileName = defaults.OutputFileName
	}

	ctx := &SwiftContext{
		RudderCLIVersion:    opts.RudderCLIVersion,
		TrackingPlanName:    p.Name,
		TrackingPlanID:      p.Metadata.TrackingPlanID,
		TrackingPlanVersion: p.Metadata.TrackingPlanVersion,
		TrackingPlanURL:     p.Metadata.URL,
		EventContext:        formatEventContext(p.Metadata, opts.RudderCLIVersion),
	}

	nr := core.NewNameRegistry(swiftCollisionHandler)

	if err := processPropertiesAndCustomTypes(p, ctx, nr); err != nil {
		return nil, err
	}

	if err := processEventRules(p, ctx, nr); err != nil {
		return nil, err
	}

	pruneUnreferencedPropertyAliases(ctx)

	file, err := GenerateFile(outputFileName, ctx)
	if err != nil {
		return nil, err
	}

	return []*core.File{file}, nil
}

func formatEventContext(meta plan.PlanMetadata, version string) map[string]string {
	return map[string]string{
		"sdk":                 `"rudder-sdk-swift"`,
		"language":            `"swift"`,
		"rudderTyperVersion":  fmt.Sprintf("%q", version),
		"trackingPlanId":      fmt.Sprintf("%q", meta.TrackingPlanID),
		"trackingPlanVersion": fmt.Sprintf("%d", meta.TrackingPlanVersion),
	}
}

func swiftCollisionHandler(name string, existing []string) string {
	return core.DefaultCollisionHandler(name, existing)
}

// ========== Naming Helpers ==========

func getOrRegisterCustomTypeName(ct *plan.CustomType, nr *core.NameRegistry) (string, error) {
	name := FormatTypeName("CustomType " + ct.Name)
	return nr.RegisterName("customtype:"+ct.Name, globalTypeScope, name)
}

func getOrRegisterPropertyTypeName(p *plan.Property, nr *core.NameRegistry) (string, error) {
	name := FormatTypeName("Property " + p.Name)
	return nr.RegisterName("property:"+p.Name, globalTypeScope, name)
}

func getOrRegisterPropertyArrayItemTypeName(p *plan.Property, nr *core.NameRegistry) (string, error) {
	name := FormatTypeName("Property " + p.Name + " Item")
	return nr.RegisterName("property:item:"+p.Name, globalTypeScope, name)
}

func getOrRegisterPropertyFieldName(structName, propName string, nr *core.NameRegistry) (string, error) {
	scope := fmt.Sprintf("struct:%s:fields", structName)
	name := formatPropertyName(propName)
	return nr.RegisterName(propName, scope, name)
}

// formatEnumValue converts an enum value to its string representation for case
// name generation. Float values use fixed-point notation to avoid scientific
// notation (e.g. 1e+06) which produces invalid Swift identifiers.
func formatEnumValue(value any) string {
	switch v := value.(type) {
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 64)
	default:
		return fmt.Sprintf("%v", value)
	}
}

func getOrRegisterEnumCaseName(enumName string, value any, nr *core.NameRegistry) (string, error) {
	scope := fmt.Sprintf("enum:%s", enumName)
	formatted := formatEnumCaseName(formatEnumValue(value))
	if formatted == "" {
		formatted = "_"
	}
	// '_' is the Swift wildcard pattern and cannot be used as an enum case identifier.
	// Pre-register a placeholder so the collision handler suffixes the actual value to '_1'.
	if formatted == "_" {
		_, err := nr.RegisterName(fmt.Sprintf("enum:placeholder:%s", formatted), scope, formatted)
		if err != nil {
			return "", err
		}
	}
	valueKey := fmt.Sprintf("%v", value)
	return nr.RegisterName(valueKey, scope, formatted)
}

func getOrRegisterEventStructName(rule *plan.EventRule, nr *core.NameRegistry) (string, error) {
	var prefix, suffix string

	switch rule.Event.EventType {
	case plan.EventTypeTrack:
		prefix = "Track"
	case plan.EventTypeIdentify:
		prefix = "Identify"
	case plan.EventTypeScreen:
		prefix = "Screen"
	case plan.EventTypeGroup:
		prefix = "Group"
	default:
		return "", fmt.Errorf("unsupported event type: %s", rule.Event.EventType)
	}

	switch rule.Section {
	case plan.IdentitySectionProperties:
		suffix = "Properties"
	case plan.IdentitySectionTraits, plan.IdentitySectionContextTraits:
		suffix = "Traits"
	default:
		return "", fmt.Errorf("unsupported section: %s", rule.Section)
	}

	name := FormatTypeName(prefix + " " + rule.Event.Name + " " + suffix)
	key := "event:" + string(rule.Event.EventType) + ":" + rule.Event.Name + ":" + string(rule.Section)
	return nr.RegisterName(key, globalTypeScope, name)
}

func getOrRegisterEventMethodName(rule *plan.EventRule, nr *core.NameRegistry) (string, error) {
	var methodName string

	switch rule.Event.EventType {
	case plan.EventTypeTrack:
		methodName = formatMethodName("track " + rule.Event.Name)
	case plan.EventTypeIdentify:
		methodName = "identify"
	case plan.EventTypeScreen:
		methodName = formatMethodName("screen " + rule.Event.Name)
	case plan.EventTypeGroup:
		methodName = "group"
	default:
		return "", fmt.Errorf("unsupported event type: %s", rule.Event.EventType)
	}

	key := "method:" + string(rule.Event.EventType) + ":" + rule.Event.Name
	return nr.RegisterName(key, methodScope, methodName)
}

// ========== Type Mapping ==========

func mapPrimitiveToSwiftType(t plan.PrimitiveType) (string, error) {
	switch t {
	case plan.PrimitiveTypeString:
		return "String", nil
	case plan.PrimitiveTypeInteger:
		return "Int", nil
	case plan.PrimitiveTypeNumber:
		return "Double", nil
	case plan.PrimitiveTypeBoolean:
		return "Bool", nil
	case plan.PrimitiveTypeObject:
		return "[String: Any]", nil
	case plan.PrimitiveTypeArray:
		return "[Any]", nil
	case plan.PrimitiveTypeNull:
		return "NSNull", nil
	default:
		return "", fmt.Errorf("unsupported primitive type: %s", t)
	}
}

func hasEnumConfig(config *plan.PropertyConfig) bool {
	return config != nil && len(config.Enum) > 0
}

func isEmptySchema(schema *plan.ObjectSchema) bool {
	return schema == nil || len(schema.Properties) == 0
}

// ========== Processing ==========

func processPropertiesAndCustomTypes(p *plan.TrackingPlan, ctx *SwiftContext, nr *core.NameRegistry) error {
	customTypes := p.ExtractAllCustomTypes()
	properties := p.ExtractAllProperties()

	if err := processCustomTypesIntoContext(customTypes, ctx, nr); err != nil {
		return err
	}

	return processPropertiesIntoContext(properties, ctx, nr)
}

func processCustomTypesIntoContext(customTypes map[string]*plan.CustomType, ctx *SwiftContext, nr *core.NameRegistry) error {
	var sortedNames []string
	for name := range customTypes {
		sortedNames = append(sortedNames, name)
	}
	sort.Strings(sortedNames)

	for _, name := range sortedNames {
		ct := customTypes[name]

		if len(ct.Variants) > 0 {
			typeName, err := getOrRegisterCustomTypeName(ct, nr)
			if err != nil {
				return err
			}
			variantEnum, err := createVariantEnum(typeName, ct.Description, ct.Schema, ct.Variants, nr, nil)
			if err != nil {
				return err
			}
			ctx.VariantEnums = append(ctx.VariantEnums, *variantEnum)
		} else if ct.Type == plan.PrimitiveTypeObject && ct.Schema != nil && isEmptySchema(ct.Schema) {
			typeName, err := getOrRegisterCustomTypeName(ct, nr)
			if err != nil {
				return err
			}
			if ct.Schema.AdditionalProperties {
				// Empty object with additionalProperties: true → open [String: Any] alias
				ctx.TypeAliases = append(ctx.TypeAliases, SwiftTypeAlias{
					Alias:   typeName,
					Type:    "[String: Any]",
					Comment: ct.Description,
				})
			} else {
				// Empty object with additionalProperties: false → empty struct so callers
				// cannot pass arbitrary keys (mirrors Kotlin's Unit mapping)
				s, err := createSwiftStruct(typeName, ct.Description, "toProperties", ct.Schema, nr, nil)
				if err != nil {
					return err
				}
				ctx.Structs = append(ctx.Structs, *s)
			}
		} else if ct.IsPrimitive() {
			if hasEnumConfig(ct.Config) {
				enum, err := createCustomTypeEnum(ct, nr)
				if err != nil {
					return err
				}
				ctx.Enums = append(ctx.Enums, *enum)
			} else {
				alias, err := createCustomTypeAlias(ct, nr)
				if err != nil {
					return err
				}
				ctx.TypeAliases = append(ctx.TypeAliases, *alias)
			}
		} else {
			// Object custom type → struct
			typeName, err := getOrRegisterCustomTypeName(ct, nr)
			if err != nil {
				return err
			}
			s, err := createSwiftStruct(typeName, ct.Description, "toProperties", ct.Schema, nr, nil)
			if err != nil {
				return err
			}
			ctx.Structs = append(ctx.Structs, *s)
		}
	}

	return nil
}

func processPropertiesIntoContext(properties map[string]*plan.Property, ctx *SwiftContext, nr *core.NameRegistry) error {
	var sortedNames []string
	for name := range properties {
		sortedNames = append(sortedNames, name)
	}
	sort.Strings(sortedNames)

	for _, name := range sortedNames {
		p := properties[name]

		if hasEnumConfig(p.Config) {
			enum, err := createPropertyEnum(p, nr)
			if err != nil {
				return err
			}
			ctx.Enums = append(ctx.Enums, *enum)
		} else if len(p.Types) > 1 {
			// Multi-type property → SwiftMultiTypeEnum
			typeName, err := getOrRegisterPropertyTypeName(p, nr)
			if err != nil {
				return err
			}
			multiEnum, err := createMultiTypeEnum(typeName, p.Description, p.Types)
			if err != nil {
				return err
			}
			ctx.MultiTypeEnums = append(ctx.MultiTypeEnums, *multiEnum)
		} else if len(p.Types) == 1 {
			pt := p.Types[0]
			isArrayWithMultiItems := plan.IsPrimitiveType(pt) &&
				*plan.AsPrimitiveType(pt) == plan.PrimitiveTypeArray &&
				len(p.ItemTypes) > 1

			if isArrayWithMultiItems {
				// Array with multiple item types → item enum + typealias for the array
				itemTypeName, err := getOrRegisterPropertyArrayItemTypeName(p, nr)
				if err != nil {
					return err
				}
				multiEnum, err := createMultiTypeEnum(itemTypeName, fmt.Sprintf("Item type for %s array", p.Name), p.ItemTypes)
				if err != nil {
					return err
				}
				ctx.MultiTypeEnums = append(ctx.MultiTypeEnums, *multiEnum)

				propTypeName, err := getOrRegisterPropertyTypeName(p, nr)
				if err != nil {
					return err
				}
				ctx.PropertyTypeAliases = append(ctx.PropertyTypeAliases, SwiftTypeAlias{
					Alias:   propTypeName,
					Type:    "[" + itemTypeName + "]",
					Comment: p.Description,
				})
			} else {
				alias, err := createPropertyAlias(p, nr)
				if err != nil {
					return err
				}
				ctx.PropertyTypeAliases = append(ctx.PropertyTypeAliases, *alias)
			}
		} else {
			// No types → RudderValue (unbounded property)
			ctx.UsesRudderValue = true
			alias, err := createPropertyAlias(p, nr)
			if err != nil {
				return err
			}
			ctx.PropertyTypeAliases = append(ctx.PropertyTypeAliases, *alias)
		}

		// Array with no item type is also unbounded → [RudderValue]
		if len(p.Types) == 1 {
			pt := p.Types[0]
			if plan.IsPrimitiveType(pt) && *plan.AsPrimitiveType(pt) == plan.PrimitiveTypeArray && len(p.ItemTypes) == 0 {
				ctx.UsesRudderValue = true
			}
		}
	}

	return nil
}

// pruneUnreferencedPropertyAliases removes PropertyTypeAliases that are not
// referenced by any struct property. Object-typed properties that are rendered
// as inline nested structs register a top-level alias that is never used;
// this removes that dead code from the generated output.
func pruneUnreferencedPropertyAliases(ctx *SwiftContext) {
	referenced := make(map[string]bool)
	for i := range ctx.Structs {
		collectTypesFromStruct(&ctx.Structs[i], referenced)
	}
	for i := range ctx.VariantEnums {
		for j := range ctx.VariantEnums[i].PayloadStructs {
			collectTypesFromStruct(&ctx.VariantEnums[i].PayloadStructs[j], referenced)
		}
	}

	filtered := make([]SwiftTypeAlias, 0, len(ctx.PropertyTypeAliases))
	for _, alias := range ctx.PropertyTypeAliases {
		if referenced[alias.Alias] {
			filtered = append(filtered, alias)
		}
	}
	ctx.PropertyTypeAliases = filtered
}

func collectTypesFromStruct(s *SwiftStruct, out map[string]bool) {
	for _, prop := range s.Properties {
		out[prop.Type] = true
	}
	for i := range s.NestedStructs {
		collectTypesFromStruct(&s.NestedStructs[i], out)
	}
}

func processEventRules(p *plan.TrackingPlan, ctx *SwiftContext, nr *core.NameRegistry) error {
	// Build a map of type name → serialize suffix for all registered enum and
	// multi-type enum types. resolveStructPropertyType consults this instead of
	// re-checking the local property config, so the serialize expression stays
	// consistent with the type declaration even when two events share a property
	// name but differ in their type constraint.
	typeSerializeSuffix := make(map[string]string, len(ctx.Enums)+len(ctx.MultiTypeEnums))
	for _, e := range ctx.Enums {
		typeSerializeSuffix[e.Name] = ".rawValue"
	}
	for _, e := range ctx.MultiTypeEnums {
		typeSerializeSuffix[e.Name] = ".value"
	}

	ruleMap := make(map[string]*plan.EventRule)
	for i := range p.Rules {
		rule := &p.Rules[i]
		key := string(rule.Event.EventType) + ":" + rule.Event.Name + ":" + string(rule.Section)
		ruleMap[key] = rule
	}

	var sortedKeys []string
	for key := range ruleMap {
		sortedKeys = append(sortedKeys, key)
	}
	sort.Strings(sortedKeys)

	for _, key := range sortedKeys {
		rule := ruleMap[key]

		if !isSupportedEventType(rule.Event.EventType) {
			ui.PrintWarning(fmt.Sprintf("unsupported event type %q, skipping", rule.Event.EventType))
			continue
		}

		if !validateEventSection(rule) {
			ui.PrintWarning(fmt.Sprintf("invalid section %q for event type %q, skipping", rule.Section, rule.Event.EventType))
			continue
		}

		if len(rule.Variants) > 0 {
			structName, err := getOrRegisterEventStructName(rule, nr)
			if err != nil {
				return err
			}
			variantEnum, err := createVariantEnum(structName, rule.Event.Description, &rule.Schema, rule.Variants, nr, typeSerializeSuffix)
			if err != nil {
				return err
			}
			ctx.VariantEnums = append(ctx.VariantEnums, *variantEnum)
		} else if isEmptySchema(&rule.Schema) && rule.Schema.AdditionalProperties {
			// allow_unplanned: true → typealias to [String: Any]
			structName, err := getOrRegisterEventStructName(rule, nr)
			if err != nil {
				return err
			}
			ctx.TypeAliases = append(ctx.TypeAliases, SwiftTypeAlias{
				Alias:   structName,
				Type:    "[String: Any]",
				Comment: rule.Event.Description,
			})
		} else if !isEmptySchema(&rule.Schema) {
			structName, err := getOrRegisterEventStructName(rule, nr)
			if err != nil {
				return err
			}
			serializeMethod := sectionToSerializeMethod(rule.Section)
			s, err := createSwiftStruct(structName, rule.Event.Description, serializeMethod, &rule.Schema, nr, typeSerializeSuffix)
			if err != nil {
				return err
			}
			ctx.Structs = append(ctx.Structs, *s)
		}

		method, err := createAnalyticsMethod(rule, nr)
		if err != nil {
			return err
		}
		if method != nil {
			ctx.AnalyticsMethods = append(ctx.AnalyticsMethods, *method)
		}
	}

	return nil
}

// ========== Enum Builders ==========

func createCustomTypeEnum(ct *plan.CustomType, nr *core.NameRegistry) (*SwiftEnum, error) {
	enumName, err := getOrRegisterCustomTypeName(ct, nr)
	if err != nil {
		return nil, err
	}
	return buildSwiftEnum(enumName, ct.Description, ct.Config.Enum, nr)
}

func createPropertyEnum(p *plan.Property, nr *core.NameRegistry) (*SwiftEnum, error) {
	enumName, err := getOrRegisterPropertyTypeName(p, nr)
	if err != nil {
		return nil, err
	}
	return buildSwiftEnum(enumName, p.Description, p.Config.Enum, nr)
}

func buildSwiftEnum(name, comment string, values []any, nr *core.NameRegistry) (*SwiftEnum, error) {
	rawType := inferEnumRawType(values)

	var enumValues []SwiftEnumValue
	for _, v := range values {
		caseName, err := getOrRegisterEnumCaseName(name, v, nr)
		if err != nil {
			return nil, err
		}
		// Coerce values to match the chosen raw type so the template emits valid literals.
		value := v
		switch rawType {
		case "String":
			if _, ok := v.(string); !ok {
				value = fmt.Sprintf("%v", v)
			}
		case "Int":
			// YAML parses integer literals as float64; convert to int so the
			// template emits `case n200 = 200` not `case n200 = 200.0`.
			switch vt := v.(type) {
			case float64:
				value = int(vt)
			case float32:
				value = int(vt)
			}
		}
		enumValues = append(enumValues, SwiftEnumValue{Name: caseName, Value: value})
	}

	return &SwiftEnum{
		Name:    name,
		Comment: comment,
		RawType: rawType,
		Values:  enumValues,
	}, nil
}

// inferEnumRawType chooses the Swift enum raw type based on the value set:
//   - all integers (or whole-number floats, as YAML parsers emit float64 for all numbers) → Int
//   - all numeric (int or float mix with fractional values) → Double
//   - everything else → String (non-string values are coerced by the caller)
func inferEnumRawType(values []any) string {
	if len(values) == 0 {
		return "String"
	}
	var (
		allInt     = true
		allNumeric = true
	)
	for _, v := range values {
		switch vt := v.(type) {
		case int, int32, int64:
			// integer — keeps allInt true
		case float32:
			// YAML/JSON parsers emit float64 for all numbers; treat whole-number
			// floats as integers so status codes like 200 map to Int, not Double.
			if float64(vt) != math.Trunc(float64(vt)) {
				allInt = false
			}
		case float64:
			if vt != math.Trunc(vt) {
				allInt = false
			}
		default:
			allInt = false
			allNumeric = false
		}
	}
	switch {
	case allInt:
		return "Int"
	case allNumeric:
		return "Double"
	default:
		return "String"
	}
}

// ========== Multi-Type Enum Builder ==========

func createMultiTypeEnum(name, comment string, types []plan.PropertyType) (*SwiftMultiTypeEnum, error) {
	var cases []SwiftMultiTypeCase
	for _, t := range types {
		if !plan.IsPrimitiveType(t) {
			return nil, fmt.Errorf("unsupported type in multi-type union: %T", t)
		}
		c, err := primitiveToMultiTypeCase(*plan.AsPrimitiveType(t))
		if err != nil {
			return nil, err
		}
		cases = append(cases, c)
	}
	return &SwiftMultiTypeEnum{Name: name, Comment: comment, Cases: cases}, nil
}

func primitiveToMultiTypeCase(t plan.PrimitiveType) (SwiftMultiTypeCase, error) {
	switch t {
	case plan.PrimitiveTypeString:
		return SwiftMultiTypeCase{Name: "string", AssociatedType: "String"}, nil
	case plan.PrimitiveTypeInteger:
		return SwiftMultiTypeCase{Name: "int", AssociatedType: "Int"}, nil
	case plan.PrimitiveTypeNumber:
		return SwiftMultiTypeCase{Name: "double", AssociatedType: "Double"}, nil
	case plan.PrimitiveTypeBoolean:
		return SwiftMultiTypeCase{Name: "bool", AssociatedType: "Bool"}, nil
	case plan.PrimitiveTypeObject:
		return SwiftMultiTypeCase{Name: "object", AssociatedType: "[String: Any]"}, nil
	case plan.PrimitiveTypeArray:
		return SwiftMultiTypeCase{Name: "array", AssociatedType: "[Any]"}, nil
	case plan.PrimitiveTypeNull:
		return SwiftMultiTypeCase{Name: "null", IsNull: true}, nil
	default:
		return SwiftMultiTypeCase{}, fmt.Errorf("unsupported type in multi-type union: %s", t)
	}
}

// ========== Variant Enum Builder ==========

func createVariantEnum(name, comment string, baseSchema *plan.ObjectSchema, variants []plan.Variant, nr *core.NameRegistry, typeSerializeSuffix map[string]string) (*SwiftVariantEnum, error) {
	if len(variants) == 0 {
		return nil, fmt.Errorf("no variants provided for %q", name)
	}
	if len(variants) > 1 {
		return nil, fmt.Errorf("multiple variants per type not supported; found %d for %q", len(variants), name)
	}

	variant := variants[0]
	var cases []SwiftVariantCase
	var payloadStructs []SwiftStruct

	for _, vc := range variant.Cases {
		// All match values in a case share the same schema, so one struct covers them all.
		// Use DisplayName for the struct; fall back to the first match value if unset.
		structLabel := vc.DisplayName
		if structLabel == "" {
			structLabel = fmt.Sprintf("%v", vc.Match[0])
		}
		payloadName := FormatTypeName("Case " + structLabel)

		merged := mergeSchemas(baseSchema, &vc.Schema)
		s, err := createSwiftStruct(payloadName, vc.Description, "toProperties", merged, nr, typeSerializeSuffix)
		if err != nil {
			return nil, err
		}

		// For single-match cases the discriminator value is fixed by definition,
		// so embed it as a hardcoded constant — callers cannot supply the wrong value.
		if len(vc.Match) == 1 {
			for i := range s.Properties {
				if s.Properties[i].SerialName == variant.Discriminator {
					prop := &s.Properties[i]
					matchStr := fmt.Sprintf("%v", vc.Match[0])
					// Enum-typed discriminators serialize via .rawValue, so the constant
					// must use enum case syntax (.post) rather than a string literal ("POST").
					if strings.HasSuffix(prop.SerializeExpr, ".rawValue") {
						prop.ConstantValue = "." + formatEnumCaseName(matchStr)
					} else {
						prop.ConstantValue = FormatSwiftLiteral(vc.Match[0])
					}
					break
				}
			}
		}
		payloadStructs = append(payloadStructs, *s)

		// One enum case per match value, all pointing to the shared struct.
		for _, matchValue := range vc.Match {
			caseName := formatPropertyName(fmt.Sprintf("case %v", matchValue))
			cases = append(cases, SwiftVariantCase{
				Name:               caseName,
				Comment:            vc.Description,
				PayloadType:        payloadName,
				DiscriminatorValue: fmt.Sprintf("%v", matchValue),
				IsDefault:          false,
			})
		}
	}

	// Default case — uses DefaultSchema if provided, else an empty schema
	defaultSchema := variant.DefaultSchema
	if defaultSchema == nil {
		defaultSchema = &plan.ObjectSchema{
			Properties:           map[string]plan.PropertySchema{},
			AdditionalProperties: false,
		}
	}
	defaultStruct, err := createSwiftStruct("Default", "Default case for unmatched discriminator values", "toProperties", defaultSchema, nr, typeSerializeSuffix)
	if err != nil {
		return nil, err
	}
	payloadStructs = append(payloadStructs, *defaultStruct)
	cases = append(cases, SwiftVariantCase{Name: "other", IsDefault: true})

	return &SwiftVariantEnum{
		Name:           name,
		Comment:        comment,
		Cases:          cases,
		PayloadStructs: payloadStructs,
	}, nil
}

// mergeSchemas merges base and case schema properties.
// Case properties take precedence on required status.
func mergeSchemas(base, caseSchema *plan.ObjectSchema) *plan.ObjectSchema {
	merged := &plan.ObjectSchema{Properties: make(map[string]plan.PropertySchema)}
	if base != nil {
		maps.Copy(merged.Properties, base.Properties)
	}
	if caseSchema != nil {
		for name, ps := range caseSchema.Properties {
			if existing, ok := merged.Properties[name]; ok {
				existing.Required = existing.Required || ps.Required
				merged.Properties[name] = existing
			} else {
				merged.Properties[name] = ps
			}
		}
	}
	return merged
}

// ========== Struct Builder ==========

func createSwiftStruct(name, comment, serializeMethod string, schema *plan.ObjectSchema, nr *core.NameRegistry, typeSerializeSuffix map[string]string) (*SwiftStruct, error) {
	if isEmptySchema(schema) {
		return &SwiftStruct{
			Name:            name,
			Comment:         comment,
			SerializeMethod: serializeMethod,
		}, nil
	}

	var sortedPropNames []string
	for propName := range schema.Properties {
		sortedPropNames = append(sortedPropNames, propName)
	}
	sort.Strings(sortedPropNames)

	var (
		properties    []SwiftStructProperty
		nestedStructs []SwiftStruct
	)

	for _, propName := range sortedPropNames {
		propSchema := schema.Properties[propName]

		fieldName, err := getOrRegisterPropertyFieldName(name, propName, nr)
		if err != nil {
			return nil, err
		}

		if propSchema.Schema != nil && len(propSchema.Schema.Properties) > 0 {
			// Nested object → nested struct (always uses toProperties)
			nestedName := FormatTypeName(propName)
			nestedStruct, err := createSwiftStruct(nestedName, propSchema.Property.Description, "toProperties", propSchema.Schema, nr, typeSerializeSuffix)
			if err != nil {
				return nil, err
			}
			nestedStructs = append(nestedStructs, *nestedStruct)

			properties = append(properties, SwiftStructProperty{
				Name:          fieldName,
				SerialName:    propName,
				Type:          nestedName,
				Comment:       propSchema.Property.Description,
				Optional:      !propSchema.Required,
				SerializeExpr: fieldName + ".toProperties()",
			})
		} else {
			swiftType, serializeExpr, err := resolveStructPropertyType(fieldName, &propSchema, nr, typeSerializeSuffix)
			if err != nil {
				return nil, fmt.Errorf("resolving type for property %q in struct %q: %w", propName, name, err)
			}
			properties = append(properties, SwiftStructProperty{
				Name:          fieldName,
				SerialName:    propName,
				Type:          swiftType,
				Comment:       propSchema.Property.Description,
				Optional:      !propSchema.Required,
				SerializeExpr: serializeExpr,
			})
		}
	}

	return &SwiftStruct{
		Name:            name,
		Comment:         comment,
		SerializeMethod: serializeMethod,
		Properties:      properties,
		NestedStructs:   nestedStructs,
	}, nil
}

// resolveStructPropertyType returns the Swift type string and serialize expression for a struct property.
// SerializeExpr references the unwrapped variable (same as fieldName) so it works for both
// required (direct dict assignment) and optional (if let unwrapping) properties.
func resolveStructPropertyType(fieldName string, propSchema *plan.PropertySchema, nr *core.NameRegistry, typeSerializeSuffix map[string]string) (swiftType string, serializeExpr string, err error) {
	prop := &propSchema.Property

	// For enum-constrained and multi-type properties, the serialize expression
	// must match the type that was actually registered during
	// processPropertiesAndCustomTypes. Two events can share a property name but
	// define different type constraints; the registered type is authoritative —
	// not the local schema. typeSerializeSuffix maps registered type names to
	// their correct suffix (.rawValue for enums, .value for multi-type enums).
	//
	// typeSerializeSuffix is nil in the custom type context (no cross-event
	// deduplication), so fall back to direct type signals there.
	if hasEnumConfig(prop.Config) || len(prop.Types) > 1 {
		typeName, err := getOrRegisterPropertyTypeName(prop, nr)
		if err != nil {
			return "", "", err
		}
		if typeSerializeSuffix != nil {
			if suffix, ok := typeSerializeSuffix[typeName]; ok {
				return typeName, fieldName + suffix, nil
			}
			// Not in map → registered as a plain alias due to deduplication — fall through.
		} else {
			// Custom type context: no cross-event deduplication, use direct signals.
			if hasEnumConfig(prop.Config) {
				return typeName, fieldName + ".rawValue", nil
			}
			return typeName, fieldName + ".value", nil
		}
	}

	if len(prop.Types) == 0 {
		return "RudderValue", fieldName + ".value", nil
	}

	pt := prop.Types[0]

	if plan.IsCustomType(pt) {
		ct := plan.AsCustomType(pt)
		if ct == nil {
			return "Any", fieldName, nil
		}
		typeName, err := getOrRegisterCustomTypeName(ct, nr)
		if err != nil {
			return "", "", err
		}
		// Determine serialize expression based on custom type kind
		itemCT := plan.AsCustomType(ct.ItemType)
		switch {
		case len(ct.Variants) > 0:
			return typeName, fieldName + ".toProperties()", nil
		case hasEnumConfig(ct.Config):
			return typeName, fieldName + ".rawValue", nil
		case ct.Type == plan.PrimitiveTypeObject && ct.Schema != nil &&
			(len(ct.Schema.Properties) > 0 || !ct.Schema.AdditionalProperties):
			return typeName, fieldName + ".toProperties()", nil
		case ct.Type == plan.PrimitiveTypeArray && itemCT != nil &&
			itemCT.Type == plan.PrimitiveTypeObject && itemCT.Schema != nil && len(itemCT.Schema.Properties) > 0:
			return typeName, fieldName + ".map { $0.toProperties() }", nil
		default:
			return typeName, fieldName, nil
		}
	}

	if plan.IsPrimitiveType(pt) {
		// All primitive-typed properties reference the registered type alias (e.g. PropertyFoo = String)
		// so struct fields stay consistent with the generated typealias declarations.
		typeName, err := getOrRegisterPropertyTypeName(prop, nr)
		if err != nil {
			return "", "", err
		}
		if *plan.AsPrimitiveType(pt) == plan.PrimitiveTypeArray {
			switch {
			case len(prop.ItemTypes) == 0:
				// Unbounded array → [RudderValue], unwrap each element for serialization.
				return typeName, fieldName + ".map { $0.value }", nil
			case len(prop.ItemTypes) > 1:
				// Multi-item-type enum: unwrap each element via .value.
				return typeName, fieldName + ".map { $0.value }", nil
			}
		}
		return typeName, fieldName, nil
	}

	return "Any", fieldName, nil
}

// ========== Type Alias Builders ==========

func createCustomTypeAlias(ct *plan.CustomType, nr *core.NameRegistry) (*SwiftTypeAlias, error) {
	typeName, err := getOrRegisterCustomTypeName(ct, nr)
	if err != nil {
		return nil, err
	}

	var swiftType string
	if ct.Type == plan.PrimitiveTypeArray {
		if ct.ItemType != nil {
			if plan.IsPrimitiveType(ct.ItemType) {
				inner, err := mapPrimitiveToSwiftType(*plan.AsPrimitiveType(ct.ItemType))
				if err != nil {
					return nil, err
				}
				swiftType = "[" + inner + "]"
			} else if plan.IsCustomType(ct.ItemType) {
				itemCT := plan.AsCustomType(ct.ItemType)
				if itemCT != nil {
					itemName, err := getOrRegisterCustomTypeName(itemCT, nr)
					if err != nil {
						return nil, err
					}
					swiftType = "[" + itemName + "]"
				} else {
					swiftType = "[Any]"
				}
			} else {
				swiftType = "[Any]"
			}
		} else {
			swiftType = "[Any]"
		}
	} else {
		swiftType, err = mapPrimitiveToSwiftType(ct.Type)
		if err != nil {
			return nil, err
		}
	}

	return &SwiftTypeAlias{Alias: typeName, Type: swiftType, Comment: ct.Description}, nil
}

func createPropertyAlias(p *plan.Property, nr *core.NameRegistry) (*SwiftTypeAlias, error) {
	typeName, err := getOrRegisterPropertyTypeName(p, nr)
	if err != nil {
		return nil, err
	}

	swiftType, err := resolvePropertyAliasType(p, nr)
	if err != nil {
		return nil, err
	}

	return &SwiftTypeAlias{Alias: typeName, Type: swiftType, Comment: p.Description}, nil
}

func resolvePropertyAliasType(p *plan.Property, nr *core.NameRegistry) (string, error) {
	if len(p.Types) == 0 {
		return "RudderValue", nil
	}
	if len(p.Types) > 1 {
		return "Any", nil
	}

	pt := p.Types[0]

	if plan.IsPrimitiveType(pt) {
		primitive := *plan.AsPrimitiveType(pt)
		if primitive == plan.PrimitiveTypeArray {
			if len(p.ItemTypes) == 0 {
				// Unbounded array — no item type constraint.
				return "[RudderValue]", nil
			}
			if len(p.ItemTypes) == 1 {
				itemType := p.ItemTypes[0]
				if plan.IsPrimitiveType(itemType) {
					inner, err := mapPrimitiveToSwiftType(*plan.AsPrimitiveType(itemType))
					if err != nil {
						return "", err
					}
					return "[" + inner + "]", nil
				}
				if plan.IsCustomType(itemType) {
					ct := plan.AsCustomType(itemType)
					if ct != nil {
						itemName, err := getOrRegisterCustomTypeName(ct, nr)
						if err != nil {
							return "", err
						}
						return "[" + itemName + "]", nil
					}
				}
				return "[RudderValue]", nil
			}
		}
		return mapPrimitiveToSwiftType(primitive)
	}

	if plan.IsCustomType(pt) {
		ct := plan.AsCustomType(pt)
		if ct != nil {
			return getOrRegisterCustomTypeName(ct, nr)
		}
	}

	return "Any", nil
}

// ========== Analytics Method Builder ==========

func createAnalyticsMethod(rule *plan.EventRule, nr *core.NameRegistry) (*SwiftAnalyticsMethod, error) {
	switch rule.Event.EventType {
	case plan.EventTypeTrack:
		return buildTrackMethod(rule, nr)
	case plan.EventTypeIdentify:
		return buildIdentifyMethod(rule, nr)
	case plan.EventTypeScreen:
		return buildScreenMethod(rule, nr)
	case plan.EventTypeGroup:
		return buildGroupMethod(rule, nr)
	default:
		ui.PrintWarning(fmt.Sprintf("unsupported event type %q, skipping", rule.Event.EventType))
		return nil, nil
	}
}

func shouldIncludeProperties(rule *plan.EventRule) bool {
	return !isEmptySchema(&rule.Schema) || rule.Schema.AdditionalProperties
}

func sectionToSerializeMethod(section plan.IdentitySection) string {
	switch section {
	case plan.IdentitySectionTraits, plan.IdentitySectionContextTraits:
		return "toTraits"
	default:
		return "toProperties"
	}
}

func isSupportedEventType(eventType plan.EventType) bool {
	switch eventType {
	case plan.EventTypeTrack, plan.EventTypeIdentify, plan.EventTypeScreen, plan.EventTypeGroup:
		return true
	}
	return false
}

func validateEventSection(rule *plan.EventRule) bool {
	switch rule.Event.EventType {
	case plan.EventTypeTrack:
		return rule.Section == plan.IdentitySectionProperties
	case plan.EventTypeIdentify:
		return rule.Section == plan.IdentitySectionTraits || rule.Section == plan.IdentitySectionContextTraits
	case plan.EventTypeScreen:
		return rule.Section == plan.IdentitySectionProperties
	case plan.EventTypeGroup:
		return rule.Section == plan.IdentitySectionTraits || rule.Section == plan.IdentitySectionContextTraits
	}
	return false
}

func buildTrackMethod(rule *plan.EventRule, nr *core.NameRegistry) (*SwiftAnalyticsMethod, error) {
	methodName, err := getOrRegisterEventMethodName(rule, nr)
	if err != nil {
		return nil, err
	}

	method := &SwiftAnalyticsMethod{
		Name:          methodName,
		Comment:       rule.Event.Description,
		EventName:     rule.Event.Name,
		SDKMethodName: "track",
		SDKArguments: []SwiftSDKCallArgument{
			{Label: "name", Value: FormatSwiftLiteral(rule.Event.Name)},
		},
	}

	if isEmptySchema(&rule.Schema) && rule.Schema.AdditionalProperties {
		// allow_unplanned: true → optional [String: Any] parameter passed directly
		className, err := getOrRegisterEventStructName(rule, nr)
		if err != nil {
			return nil, err
		}
		method.MethodArguments = append(method.MethodArguments, SwiftMethodArgument{
			Name:     "properties",
			Type:     className,
			Comment:  "The properties to include with this event",
			Optional: true,
			Default:  "nil",
		})
		method.SDKArguments = append(method.SDKArguments, SwiftSDKCallArgument{
			Label: "properties",
			Value: "properties",
		})
	} else if !isEmptySchema(&rule.Schema) {
		className, err := getOrRegisterEventStructName(rule, nr)
		if err != nil {
			return nil, err
		}
		method.MethodArguments = append(method.MethodArguments, SwiftMethodArgument{
			Name:    "properties",
			Type:    className,
			Comment: "The properties to include with this event",
		})
		method.SDKArguments = append(method.SDKArguments, SwiftSDKCallArgument{
			Label: "properties",
			Value: "properties.toProperties()",
		})
	} else {
		// Empty event — pass nil
		method.SDKArguments = append(method.SDKArguments, SwiftSDKCallArgument{
			Label: "properties",
			Value: "nil",
		})
	}

	return method, nil
}

func buildIdentifyMethod(rule *plan.EventRule, nr *core.NameRegistry) (*SwiftAnalyticsMethod, error) {
	methodName, err := getOrRegisterEventMethodName(rule, nr)
	if err != nil {
		return nil, err
	}

	method := &SwiftAnalyticsMethod{
		Name:          methodName,
		Comment:       rule.Event.Description,
		EventName:     rule.Event.Name,
		SDKMethodName: "identify",
		MethodArguments: []SwiftMethodArgument{
			{Name: "userId", Type: "String", Optional: true, Default: "nil"},
		},
		SDKArguments: []SwiftSDKCallArgument{
			{Label: "userId", Value: "userId"},
		},
	}

	if shouldIncludeProperties(rule) {
		className, err := getOrRegisterEventStructName(rule, nr)
		if err != nil {
			return nil, err
		}
		method.MethodArguments = append(method.MethodArguments, SwiftMethodArgument{
			Name:     "traits",
			Type:     className,
			Comment:  "The traits to include with this event",
			Optional: true,
			Default:  "nil",
		})
		if rule.Section == plan.IdentitySectionContextTraits {
			method.AddDataToContext = true
		} else {
			method.SDKArguments = append(method.SDKArguments, SwiftSDKCallArgument{
				Label: "traits",
				Value: "traits?.toTraits()",
			})
		}
	}

	return method, nil
}

func buildGroupMethod(rule *plan.EventRule, nr *core.NameRegistry) (*SwiftAnalyticsMethod, error) {
	methodName, err := getOrRegisterEventMethodName(rule, nr)
	if err != nil {
		return nil, err
	}

	method := &SwiftAnalyticsMethod{
		Name:          methodName,
		Comment:       rule.Event.Description,
		EventName:     rule.Event.Name,
		SDKMethodName: "group",
		MethodArguments: []SwiftMethodArgument{
			{Name: "groupId", Type: "String"},
		},
		SDKArguments: []SwiftSDKCallArgument{
			{Label: "groupId", Value: "groupId"},
		},
	}

	if shouldIncludeProperties(rule) {
		className, err := getOrRegisterEventStructName(rule, nr)
		if err != nil {
			return nil, err
		}
		method.MethodArguments = append(method.MethodArguments, SwiftMethodArgument{
			Name:     "traits",
			Type:     className,
			Comment:  "The traits to include with this event",
			Optional: true,
			Default:  "nil",
		})
		if rule.Section == plan.IdentitySectionContextTraits {
			method.AddDataToContext = true
		} else {
			method.SDKArguments = append(method.SDKArguments, SwiftSDKCallArgument{
				Label: "traits",
				Value: "traits?.toTraits()",
			})
		}
	}

	return method, nil
}

func buildScreenMethod(rule *plan.EventRule, nr *core.NameRegistry) (*SwiftAnalyticsMethod, error) {
	methodName, err := getOrRegisterEventMethodName(rule, nr)
	if err != nil {
		return nil, err
	}

	method := &SwiftAnalyticsMethod{
		Name:          methodName,
		Comment:       rule.Event.Description,
		EventName:     rule.Event.Name,
		SDKMethodName: "screen",
		MethodArguments: []SwiftMethodArgument{
			{Name: "screenName", Type: "String"},
		},
		SDKArguments: []SwiftSDKCallArgument{
			{Label: "screenName", Value: "screenName"},
			{Label: "category", Value: "category"},
		},
	}

	if isEmptySchema(&rule.Schema) && rule.Schema.AdditionalProperties {
		className, err := getOrRegisterEventStructName(rule, nr)
		if err != nil {
			return nil, err
		}
		method.MethodArguments = append(method.MethodArguments, SwiftMethodArgument{
			Name:     "properties",
			Type:     className,
			Optional: true,
			Default:  "nil",
		})
		method.SDKArguments = append(method.SDKArguments, SwiftSDKCallArgument{
			Label: "properties",
			Value: "properties",
		})
	} else if !isEmptySchema(&rule.Schema) {
		className, err := getOrRegisterEventStructName(rule, nr)
		if err != nil {
			return nil, err
		}
		method.MethodArguments = append(method.MethodArguments, SwiftMethodArgument{
			Name:    "properties",
			Type:    className,
			Comment: "The properties to include with this event",
		})
		method.SDKArguments = append(method.SDKArguments, SwiftSDKCallArgument{
			Label: "properties",
			Value: "properties.toProperties()",
		})
	} else {
		method.SDKArguments = append(method.SDKArguments, SwiftSDKCallArgument{
			Label: "properties",
			Value: "nil",
		})
	}

	// category always comes after properties in the method signature
	method.MethodArguments = append(method.MethodArguments, SwiftMethodArgument{
		Name:     "category",
		Type:     "String",
		Optional: true,
		Default:  "nil",
	})

	return method, nil
}

