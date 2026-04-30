package typescript

import (
	"fmt"
	"sort"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/typer/generator/core"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer/plan"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
)

const Platform = "typescript"

// Generator implements core.Generator for the TypeScript platform.
type Generator struct{}

const (
	globalTypeScope = "types"
	methodScope     = "methods"
)

// ========== Main Entry Point ==========

func (g *Generator) Generate(p *plan.TrackingPlan, opts core.GenerateOptions, platformOpts any) ([]*core.File, error) {
	defaults := g.DefaultOptions().(TypeScriptOptions)
	tsOpts := defaults
	if platformOpts != nil {
		tsOpts = platformOpts.(TypeScriptOptions)
	}

	outputFileName := tsOpts.OutputFileName
	if outputFileName == "" {
		outputFileName = defaults.OutputFileName
	}

	ctx := &TSContext{
		RudderCLIVersion:    opts.RudderCLIVersion,
		TrackingPlanName:    p.Name,
		TrackingPlanID:      p.Metadata.TrackingPlanID,
		TrackingPlanVersion: p.Metadata.TrackingPlanVersion,
		TrackingPlanURL:     p.Metadata.URL,
		EventContext:        formatEventContext(p.Metadata, opts.RudderCLIVersion),
	}

	nr := core.NewNameRegistry(typescriptCollisionHandler)

	if err := processEventRules(p, ctx, nr); err != nil {
		return nil, err
	}

	file, err := GenerateFile(outputFileName, ctx)
	if err != nil {
		return nil, err
	}

	return []*core.File{file}, nil
}

func formatEventContext(meta plan.PlanMetadata, version string) map[string]string {
	return map[string]string{
		"platform":            fmt.Sprintf("%q", Platform),
		"rudderCLIVersion":    fmt.Sprintf("%q", version),
		"trackingPlanId":      fmt.Sprintf("%q", meta.TrackingPlanID),
		"trackingPlanVersion": fmt.Sprintf("%d", meta.TrackingPlanVersion),
	}
}

func typescriptCollisionHandler(name string, existing []string) string {
	return core.DefaultCollisionHandler(name, existing)
}

// ========== Naming Helpers ==========

func getOrRegisterEventInterfaceName(rule *plan.EventRule, nr *core.NameRegistry) (string, error) {
	var prefix, suffix string

	switch rule.Event.EventType {
	case plan.EventTypeIdentify:
		prefix = "Identify"
	default:
		return "", fmt.Errorf("unsupported event type: %s", rule.Event.EventType)
	}

	switch rule.Section {
	case plan.IdentitySectionTraits, plan.IdentitySectionContextTraits:
		suffix = "Traits"
	default:
		return "", fmt.Errorf("unsupported section: %s", rule.Section)
	}

	name := FormatTypeName(prefix, rule.Event.Name+" "+suffix)
	key := "event:" + string(rule.Event.EventType) + ":" + rule.Event.Name + ":" + string(rule.Section)
	return nr.RegisterName(key, globalTypeScope, name)
}

func getOrRegisterEventMethodName(rule *plan.EventRule, nr *core.NameRegistry) (string, error) {
	var methodName string

	switch rule.Event.EventType {
	case plan.EventTypeIdentify:
		methodName = "identify"
	default:
		return "", fmt.Errorf("unsupported event type: %s", rule.Event.EventType)
	}

	key := "method:" + string(rule.Event.EventType) + ":" + rule.Event.Name
	return nr.RegisterName(key, methodScope, methodName)
}

// getOrRegisterInterfacePropertyName returns the name to emit for a property
// inside an interface, plus a `quoted` flag indicating whether the template
// must wrap it in double quotes.
//
// When the JSON key sanitises to a valid TS identifier, it is registered with
// the NameRegistry under a per-interface scope so duplicates after sanitisation
// trigger the collision handler. When it does not (e.g. "用户名", "user-id",
// "first name"), the original key is preserved verbatim and emitted as a
// quoted property; registration is unnecessary because JSON object keys are
// already unique within a single schema.
func getOrRegisterInterfacePropertyName(interfaceName, propName string, nr *core.NameRegistry) (name string, quoted bool, err error) {
	formatted := FormatPropertyName(propName)
	if !isValidTSIdentifier(formatted) {
		return propName, true, nil
	}
	scope := fmt.Sprintf("interface:%s:fields", interfaceName)
	registered, err := nr.RegisterName(propName, scope, formatted)
	if err != nil {
		return "", false, err
	}
	return registered, false, nil
}

// isValidTSIdentifier reports whether s is a syntactically valid TS identifier.
// Used to decide whether a property must be emitted as a quoted JSON-style key.
func isValidTSIdentifier(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		switch {
		case r == '_' || r == '$':
			continue
		case i == 0 && !isIdentStart(r):
			return false
		case i > 0 && !isIdentPart(r):
			return false
		}
	}
	return true
}

func isIdentStart(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_' || r == '$'
}

func isIdentPart(r rune) bool {
	return isIdentStart(r) || (r >= '0' && r <= '9')
}

// ========== Type Mapping ==========

// mapPrimitiveToTSType maps a plan primitive to its base TS type.
//   - PrimitiveTypeArray and PrimitiveTypeObject return open types here
//     (`unknown[]`, `Record<string, unknown>`); callers that have item-type or
//     schema information narrow further before emitting.
func mapPrimitiveToTSType(t plan.PrimitiveType) (string, error) {
	switch t {
	case plan.PrimitiveTypeString:
		return "string", nil
	case plan.PrimitiveTypeInteger, plan.PrimitiveTypeNumber:
		return "number", nil
	case plan.PrimitiveTypeBoolean:
		return "boolean", nil
	case plan.PrimitiveTypeNull:
		return "null", nil
	case plan.PrimitiveTypeArray:
		return "unknown[]", nil
	case plan.PrimitiveTypeObject:
		return "Record<string, unknown>", nil
	default:
		return "", fmt.Errorf("unsupported primitive type: %s", t)
	}
}

// resolvePropertyType returns the TS type expression for a property in an
// interface. Multi-type unions (including `T | null` for nullable properties)
// are emitted inline so the IR's distinction between optional (`prop?: T`) and
// nullable (`T | null`) survives into the output.
//
// Custom types currently resolve to their underlying primitive — named type
// aliases for custom types are intentionally deferred to follow-up tickets so
// the identify-only landing stays small.
func resolvePropertyType(prop *plan.Property) (string, error) {
	if len(prop.Types) == 0 {
		return "unknown", nil
	}

	if len(prop.Types) > 1 {
		return resolveMultiType(prop.Types)
	}

	pt := prop.Types[0]
	if plan.IsCustomType(pt) {
		ct := plan.AsCustomType(pt)
		if ct == nil {
			return "unknown", nil
		}
		return resolveCustomType(ct)
	}

	if plan.IsPrimitiveType(pt) {
		primitive := *plan.AsPrimitiveType(pt)
		if primitive == plan.PrimitiveTypeArray {
			return resolveArrayType(prop.ItemTypes)
		}
		return mapPrimitiveToTSType(primitive)
	}

	return "unknown", nil
}

// resolveMultiType emits `A | B | C` for a multi-type property. Each element is
// resolved independently; nested arrays/objects use the same open-type fallback
// as resolvePropertyType.
func resolveMultiType(types []plan.PropertyType) (string, error) {
	parts := make([]string, 0, len(types))
	for _, t := range types {
		var (
			part string
			err  error
		)
		switch {
		case plan.IsCustomType(t):
			ct := plan.AsCustomType(t)
			if ct == nil {
				part = "unknown"
			} else {
				part, err = resolveCustomType(ct)
			}
		case plan.IsPrimitiveType(t):
			part, err = mapPrimitiveToTSType(*plan.AsPrimitiveType(t))
		default:
			part = "unknown"
		}
		if err != nil {
			return "", err
		}
		parts = append(parts, part)
	}
	return joinUnion(parts), nil
}

// resolveCustomType returns the underlying TS type for a custom type. Object
// custom types fall back to the open record type because named interfaces for
// custom types are out of scope for the identify-only ticket.
func resolveCustomType(ct *plan.CustomType) (string, error) {
	if ct.Type == plan.PrimitiveTypeArray {
		if ct.ItemType == nil {
			return "unknown[]", nil
		}
		return resolveArrayItem(ct.ItemType)
	}
	if ct.Type == plan.PrimitiveTypeObject {
		// Open object — no per-custom-type interface generated yet.
		return "Record<string, unknown>", nil
	}
	return mapPrimitiveToTSType(ct.Type)
}

func resolveArrayType(itemTypes []plan.PropertyType) (string, error) {
	switch len(itemTypes) {
	case 0:
		return "unknown[]", nil
	case 1:
		inner, err := resolveArrayItem(itemTypes[0])
		if err != nil {
			return "", err
		}
		return inner, nil
	default:
		// Mixed item types → Array<A | B>. Wrap with Array<...> rather than
		// `(A | B)[]` so the union grouping is unambiguous.
		parts := make([]string, 0, len(itemTypes))
		for _, t := range itemTypes {
			inner, err := resolveSingleItem(t)
			if err != nil {
				return "", err
			}
			parts = append(parts, inner)
		}
		return "Array<" + joinUnion(parts) + ">", nil
	}
}

func resolveArrayItem(t plan.PropertyType) (string, error) {
	inner, err := resolveSingleItem(t)
	if err != nil {
		return "", err
	}
	// Wrap unions in Array<...> for clarity; otherwise use the postfix `[]`.
	if strings.Contains(inner, "|") {
		return "Array<" + inner + ">", nil
	}
	return inner + "[]", nil
}

func resolveSingleItem(t plan.PropertyType) (string, error) {
	if plan.IsCustomType(t) {
		ct := plan.AsCustomType(t)
		if ct == nil {
			return "unknown", nil
		}
		return resolveCustomType(ct)
	}
	if plan.IsPrimitiveType(t) {
		return mapPrimitiveToTSType(*plan.AsPrimitiveType(t))
	}
	return "unknown", nil
}

// joinUnion joins parts with `|`, preserving order on first occurrence and
// dropping duplicates. Dedup matters because plan-level distinct types can
// collapse to the same TS type — e.g. `integer` and `number` both become
// `number`, so `integer | number | boolean` should render `number | boolean`
// rather than `number | number | boolean`.
func joinUnion(parts []string) string {
	if len(parts) == 0 {
		return "unknown"
	}
	seen := make(map[string]bool, len(parts))
	deduped := make([]string, 0, len(parts))
	for _, p := range parts {
		if seen[p] {
			continue
		}
		seen[p] = true
		deduped = append(deduped, p)
	}
	return strings.Join(deduped, " | ")
}

// ========== Processing ==========

func processEventRules(p *plan.TrackingPlan, ctx *TSContext, nr *core.NameRegistry) error {
	ruleMap := make(map[string]*plan.EventRule)
	for i := range p.Rules {
		rule := &p.Rules[i]
		key := string(rule.Event.EventType) + ":" + rule.Event.Name + ":" + string(rule.Section)
		ruleMap[key] = rule
	}

	sortedKeys := make([]string, 0, len(ruleMap))
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

		if !isEmptySchema(&rule.Schema) {
			interfaceName, err := getOrRegisterEventInterfaceName(rule, nr)
			if err != nil {
				return err
			}
			iface, err := buildInterface(interfaceName, rule.Event.Description, &rule.Schema, nr)
			if err != nil {
				return err
			}
			ctx.Interfaces = append(ctx.Interfaces, *iface)
		}

		method, err := buildIdentifyMethod(rule, nr)
		if err != nil {
			return err
		}
		if method != nil {
			ctx.AnalyticsMethods = append(ctx.AnalyticsMethods, *method)
		}
	}

	return nil
}

// isSupportedEventType limits the v1 typer to identify only. Other event types
// are accepted in the plan (so existing reference plans keep validating) but
// emit a warning and produce no output.
func isSupportedEventType(t plan.EventType) bool {
	return t == plan.EventTypeIdentify
}

func validateEventSection(rule *plan.EventRule) bool {
	if rule.Event.EventType == plan.EventTypeIdentify {
		return rule.Section == plan.IdentitySectionTraits || rule.Section == plan.IdentitySectionContextTraits
	}
	return false
}

func isEmptySchema(schema *plan.ObjectSchema) bool {
	return schema == nil || len(schema.Properties) == 0
}

// ========== Interface Builder ==========

func buildInterface(name, comment string, schema *plan.ObjectSchema, nr *core.NameRegistry) (*TSInterface, error) {
	sortedNames := make([]string, 0, len(schema.Properties))
	for propName := range schema.Properties {
		sortedNames = append(sortedNames, propName)
	}
	sort.Strings(sortedNames)

	properties := make([]TSInterfaceProperty, 0, len(sortedNames))
	for _, propName := range sortedNames {
		propSchema := schema.Properties[propName]

		fieldName, quoted, err := getOrRegisterInterfacePropertyName(name, propName, nr)
		if err != nil {
			return nil, err
		}

		tsType, err := resolvePropertyType(&propSchema.Property)
		if err != nil {
			return nil, fmt.Errorf("resolving type for property %q in interface %q: %w", propName, name, err)
		}

		properties = append(properties, TSInterfaceProperty{
			Name:       fieldName,
			Type:       tsType,
			Comment:    propSchema.Property.Description,
			Optional:   !propSchema.Required,
			QuotedName: quoted,
		})
	}

	return &TSInterface{
		Name:       name,
		Comment:    comment,
		Properties: properties,
	}, nil
}

// ========== Analytics Method Builder ==========

func buildIdentifyMethod(rule *plan.EventRule, nr *core.NameRegistry) (*TSAnalyticsMethod, error) {
	methodName, err := getOrRegisterEventMethodName(rule, nr)
	if err != nil {
		return nil, err
	}

	method := &TSAnalyticsMethod{
		Name:          methodName,
		Comment:       rule.Event.Description,
		EventName:     rule.Event.Name,
		SDKMethodName: "identify",
		MethodArguments: []TSMethodArgument{
			{
				Name:    "userId",
				Type:    "string",
				Comment: "The user identifier",
			},
		},
		SDKArguments: []TSSDKArgument{
			{Value: "userId"},
		},
	}

	if !isEmptySchema(&rule.Schema) {
		interfaceName, err := getOrRegisterEventInterfaceName(rule, nr)
		if err != nil {
			return nil, err
		}
		method.MethodArguments = append(method.MethodArguments, TSMethodArgument{
			Name:     "traits",
			Type:     interfaceName,
			Comment:  "The traits to include with this event",
			Optional: true,
		})
		// Cast to any at the SDK boundary: the SDK's IdentifyTraits parameter has
		// an index signature, which our strictly-typed generated interface lacks.
		// The public method surface stays type-safe; only this one hop into the
		// SDK is loosened so strict-mode compilation passes.
		method.SDKArguments = append(method.SDKArguments, TSSDKArgument{Value: "traits as any"})
	} else {
		method.SDKArguments = append(method.SDKArguments, TSSDKArgument{Value: "undefined"})
	}

	return method, nil
}
