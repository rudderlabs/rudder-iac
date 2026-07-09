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

// SDK types are imported under aliased names so the strict cast at the SDK
// boundary can target the SDK's own types without colliding with our generated
// interface names (e.g. our generated `IdentifyTraits` shadows the SDK type of
// the same name).
const (
	sdkApiObjectAlias      = "SDKApiObject"
	sdkIdentifyTraitsAlias = "SDKIdentifyTraits"
	// openObjectType is the permissive shape for `additionalProperties: true`
	// or unknown structure; closedObjectType encodes the stricter
	// `additionalProperties: false` (no extra fields allowed). Using a literal
	// `{}` would be too permissive — TS treats it as "any non-null value", not
	// "empty object" — so we emit `Record<string, never>` to forbid keys.
	openObjectType   = "Record<string, unknown>"
	closedObjectType = "Record<string, never>"
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

	// Order matters: custom types and property enums are registered first so
	// downstream resolvers see their final names. Event rules are processed
	// last because their interfaces refer to those names.
	if err := processCustomTypesIntoContext(p, ctx, nr); err != nil {
		return nil, err
	}
	if err := processPropertyEnumsIntoContext(p, ctx, nr); err != nil {
		return nil, err
	}
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

func getOrRegisterCustomTypeName(ct *plan.CustomType, nr *core.NameRegistry) (string, error) {
	name := FormatTypeName("CustomType", ct.Name)
	return nr.RegisterName("customtype:"+ct.Name, globalTypeScope, name)
}

func getOrRegisterPropertyEnumName(p *plan.Property, nr *core.NameRegistry) (string, error) {
	name := FormatTypeName("Property", p.Name)
	return nr.RegisterName("propertyenum:"+p.Name, globalTypeScope, name)
}

func getOrRegisterEventInterfaceName(rule *plan.EventRule, nr *core.NameRegistry) (string, error) {
	key := "event:" + string(rule.Event.EventType) + ":" + rule.Event.Name + ":" + string(rule.Section)
	switch rule.Event.EventType {
	case plan.EventTypeIdentify:
		// Identify is a singleton — interface is always `IdentifyTraits`. The
		// rule's section (traits vs context.traits) doesn't change the
		// generated interface name; the routing difference is handled in
		// the dispatcher body (AddDataToContext).
		name := FormatTypeName("Identify", "Traits")
		return nr.RegisterName(key, globalTypeScope, name)
	case plan.EventTypeGroup:
		// Group mirrors identify: singleton in analytics-js, one `group()`
		// method, one `GroupTraits` interface. The section (traits vs
		// context.traits) affects routing, not the interface name.
		name := FormatTypeName("Group", "Traits")
		return nr.RegisterName(key, globalTypeScope, name)
	case plan.EventTypeTrack:
		// Track interfaces follow the spec convention: PascalCase event name
		// only, no prefix or suffix. Per-event distinct names come from the
		// event name itself.
		name := FormatTypeName("", rule.Event.Name)
		if name == "" {
			return "", fmt.Errorf("track event has empty name")
		}
		return nr.RegisterName(key, globalTypeScope, name)
	case plan.EventTypePage:
		// Page is a singleton in analytics-js: the SDK exposes one `page()`
		// method with the page name passed at call time, so the typed
		// properties interface is `PageProperties` regardless of the plan
		// rule's name. Mirrors identify/group.
		name := FormatTypeName("Page", "Properties")
		return nr.RegisterName(key, globalTypeScope, name)
	}
	return "", fmt.Errorf("unsupported event type: %s", rule.Event.EventType)
}

func getOrRegisterEventMethodName(rule *plan.EventRule, nr *core.NameRegistry) (string, error) {
	var methodName string

	switch rule.Event.EventType {
	case plan.EventTypeIdentify:
		methodName = "identify"
	case plan.EventTypeGroup:
		// Group is a singleton call in analytics-js — the SDK exposes a single
		// `group(groupId, traits, ...)` method, so the generated wrapper is
		// named `group` regardless of the plan rule name.
		methodName = "group"
	case plan.EventTypeTrack:
		// Always prefix track methods with `track` so the call site reads
		// `analytics.trackUserSignedUp(...)` rather than collapsing into the
		// camelCased event name (which can collide with property accessors).
		methodName = FormatMethodName("track", rule.Event.Name)
	case plan.EventTypePage:
		// Singleton page() — name lives in the call argument, not the method
		// name. Mirrors identify/group.
		methodName = "page"
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
// Emit the plan's property key verbatim so the generated field name matches the
// key sent on the wire; the props object is passed to analytics.track()
// unchanged, so camelCasing the field name would ship a key that violates the
// tracking plan. Quote only when the key is not a valid TS identifier
// (e.g. "用户名", "user-id", "first name"). JSON object keys are unique within a
// single schema, so no NameRegistry collision registration is needed here.
func getOrRegisterInterfacePropertyName(interfaceName, propName string, nr *core.NameRegistry) (name string, quoted bool, err error) {
	if !isValidTSIdentifier(propName) {
		return propName, true, nil
	}
	return propName, false, nil
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
		return openObjectType, nil
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

// resolvePropertyType returns the TS type expression for a property in an
// interface. The optional enumName is the name of the registered enum alias
// (when the property has its own enum config); when set, it short-circuits the
// rest of the resolution — multi-type / array logic is irrelevant when the
// schema constrains the value to a finite set.
//
// nestedTypeName is the name of a hoisted nested-object interface (when the
// surrounding schema defined an inline `Schema` for this property); when set,
// the property's resolved type is the nested interface name and the underlying
// `object` primitive is overridden.
func resolvePropertyType(prop *plan.Property, enumName, nestedTypeName string, nr *core.NameRegistry) (string, error) {
	if enumName != "" {
		return enumName, nil
	}
	if nestedTypeName != "" {
		return nestedTypeName, nil
	}
	if len(prop.Types) == 0 {
		return "unknown", nil
	}
	if len(prop.Types) > 1 {
		return resolveMultiType(prop.Types, prop.ItemTypes, nr)
	}

	pt := prop.Types[0]
	if plan.IsCustomType(pt) {
		ct := plan.AsCustomType(pt)
		if ct == nil {
			return "unknown", nil
		}
		return resolveCustomTypeReference(ct, nr)
	}

	if plan.IsPrimitiveType(pt) {
		primitive := *plan.AsPrimitiveType(pt)
		if primitive == plan.PrimitiveTypeArray {
			return resolveArrayType(prop.ItemTypes, nr)
		}
		return mapPrimitiveToTSType(primitive)
	}

	return "unknown", nil
}

// resolveMultiType emits `A | B | C` for a multi-type property. An array
// member is narrowed with itemTypes — `Types: [Array, Null]` with
// `ItemTypes: [String]` renders `string[] | null` — so the union does not
// lose item-type information; without item types the array stays open
// (`unknown[]`).
func resolveMultiType(types []plan.PropertyType, itemTypes []plan.PropertyType, nr *core.NameRegistry) (string, error) {
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
				part, err = resolveCustomTypeReference(ct, nr)
			}
		case plan.IsPrimitiveType(t):
			primitive := *plan.AsPrimitiveType(t)
			if primitive == plan.PrimitiveTypeArray {
				part, err = resolveArrayType(itemTypes, nr)
			} else {
				part, err = mapPrimitiveToTSType(primitive)
			}
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

// resolveCustomTypeReference returns the TS type expression to use when a
// property references a custom type. Object and enum custom types resolve to
// the registered name; primitive and array custom types also resolve to a
// registered alias (so the IR's intent — "this is the email type" — survives
// to the generated output rather than collapsing to an anonymous `string`).
func resolveCustomTypeReference(ct *plan.CustomType, nr *core.NameRegistry) (string, error) {
	return resolveCustomTypeName(ct, nr)
}

func resolveCustomTypeName(ct *plan.CustomType, nr *core.NameRegistry) (string, error) {
	return getOrRegisterCustomTypeName(ct, nr)
}

func resolveArrayType(itemTypes []plan.PropertyType, nr *core.NameRegistry) (string, error) {
	switch len(itemTypes) {
	case 0:
		return "unknown[]", nil
	case 1:
		return resolveArrayItem(itemTypes[0], nr)
	default:
		// Mixed item types → Array<A | B>. Wrap with Array<...> rather than
		// `(A | B)[]` so the union grouping is unambiguous.
		parts := make([]string, 0, len(itemTypes))
		for i, t := range itemTypes {
			inner, err := resolveSingleItem(t, nr)
			if err != nil {
				return "", fmt.Errorf("resolving array item type at index %d: %w", i, err)
			}
			parts = append(parts, inner)
		}
		return "Array<" + joinUnion(parts) + ">", nil
	}
}

func resolveArrayItem(t plan.PropertyType, nr *core.NameRegistry) (string, error) {
	inner, err := resolveSingleItem(t, nr)
	if err != nil {
		return "", err
	}
	if strings.Contains(inner, "|") {
		return "Array<" + inner + ">", nil
	}
	return inner + "[]", nil
}

func resolveSingleItem(t plan.PropertyType, nr *core.NameRegistry) (string, error) {
	if plan.IsCustomType(t) {
		ct := plan.AsCustomType(t)
		if ct == nil {
			return "unknown", nil
		}
		return resolveCustomTypeReference(ct, nr)
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

// formatEnumLiteral formats a single enum value as a TS literal expression
// (string, number, boolean, or `null`). Strings are quoted and escaped; other
// scalars use their Go representation. Mixed-type enums are deduplicated by
// the caller via joinUnion.
func formatEnumLiteral(value any) string {
	return FormatTSLiteral(value)
}

func buildEnumUnion(values []any) string {
	parts := make([]string, 0, len(values))
	for _, v := range values {
		parts = append(parts, formatEnumLiteral(v))
	}
	if len(parts) == 0 {
		return "unknown"
	}
	return joinUnion(parts)
}

// ========== Custom Type Processing ==========

// processCustomTypesIntoContext walks the transitive closure of custom types
// in the plan and emits one declaration per type:
//   - object → `export interface CustomTypeFoo { ... }`
//   - object with empty schema + additionalProperties → `export type CustomTypeFoo = Record<string, unknown>`
//   - enum-constrained primitive → `export type CustomTypeFoo = "a" | "b"`
//   - array → `export type CustomTypeFoo = Item[]`
//   - bare primitive → `export type CustomTypeFoo = string` (preserved as a
//     named alias so the IR's intent — e.g. "email" — survives to the output)
//
// Variant custom types are emitted as discriminated unions via
// processCustomTypeVariant (case interfaces + union type alias).
func processCustomTypesIntoContext(p *plan.TrackingPlan, ctx *TSContext, nr *core.NameRegistry) error {
	customTypes := p.ExtractAllCustomTypes()

	sortedNames := make([]string, 0, len(customTypes))
	for name := range customTypes {
		sortedNames = append(sortedNames, name)
	}
	sort.Strings(sortedNames)

	for _, name := range sortedNames {
		ct := customTypes[name]

		if len(ct.Variants) > 0 {
			if err := processCustomTypeVariant(ct, ctx, nr); err != nil {
				return err
			}
			continue
		}

		if err := emitCustomType(ct, ctx, nr); err != nil {
			return err
		}
	}

	return nil
}

// emitCustomType emits a single custom type declaration based on its kind.
func emitCustomType(ct *plan.CustomType, ctx *TSContext, nr *core.NameRegistry) error {
	typeName, err := getOrRegisterCustomTypeName(ct, nr)
	if err != nil {
		return err
	}

	switch {
	case ct.Type == plan.PrimitiveTypeObject:
		return emitCustomObjectType(ct, typeName, ctx, nr)
	case hasEnumConfig(ct.Config):
		ctx.CustomTypeAliases = append(ctx.CustomTypeAliases, TSTypeAlias{
			Alias:   typeName,
			Type:    buildEnumUnion(ct.Config.Enum),
			Comment: ct.Description,
		})
		return nil
	case ct.Type == plan.PrimitiveTypeArray:
		itemType, err := resolveCustomTypeArrayItem(ct.ItemType, nr)
		if err != nil {
			return err
		}
		ctx.CustomTypeAliases = append(ctx.CustomTypeAliases, TSTypeAlias{
			Alias:   typeName,
			Type:    itemType,
			Comment: ct.Description,
		})
		return nil
	default:
		// Bare primitive custom type — emitted as a named alias.
		tsType, err := mapPrimitiveToTSType(ct.Type)
		if err != nil {
			return err
		}
		ctx.CustomTypeAliases = append(ctx.CustomTypeAliases, TSTypeAlias{
			Alias:   typeName,
			Type:    tsType,
			Comment: ct.Description,
		})
		return nil
	}
}


func emitCustomObjectType(ct *plan.CustomType, typeName string, ctx *TSContext, nr *core.NameRegistry) error {
	if isEmptySchema(ct.Schema) {
		// `additionalProperties: true` → permissive `Record<string, unknown>`;
		// `additionalProperties: false` → strict `Record<string, never>`.
		// Mirrors Kotlin (`JsonObject` vs `Unit`) and Swift (`[String: Any]`
		// vs empty struct). Collapsing both to one shape — as an earlier draft
		// did — silently lets callers pass extra keys into a closed schema.
		ctx.CustomTypeAliases = append(ctx.CustomTypeAliases, TSTypeAlias{
			Alias:   typeName,
			Type:    emptyObjectType(ct.Schema),
			Comment: ct.Description,
		})
		return nil
	}
	iface, err := buildInterfaceFromSchema(typeName, ct.Description, ct.Schema, nr)
	if err != nil {
		return err
	}
	ctx.CustomInterfaces = append(ctx.CustomInterfaces, *iface)
	return nil
}

// emptyObjectType returns the TS type expression for an empty object schema,
// honouring `additionalProperties` so closed-empty schemas refuse extra keys
// at the type level instead of silently accepting them.
func emptyObjectType(schema *plan.ObjectSchema) string {
	if schema != nil && schema.AdditionalProperties {
		return openObjectType
	}
	return closedObjectType
}

func resolveCustomTypeArrayItem(item plan.PropertyType, nr *core.NameRegistry) (string, error) {
	if item == nil {
		return "unknown[]", nil
	}
	inner, err := resolveSingleItem(item, nr)
	if err != nil {
		return "", err
	}
	if strings.Contains(inner, "|") {
		return "Array<" + inner + ">", nil
	}
	return inner + "[]", nil
}

// ========== Property Enum Processing ==========

// processPropertyEnumsIntoContext walks all properties in the plan and emits
// a top-level type alias for each one with an enum config. Property-level
// enums emit as TS literal unions (`"a" | "b"`) rather than `enum`
// declarations so callers can pass plain string/number values directly.
//
// The registry scope is global (not per-event), so a property defined once and
// referenced from multiple event rules — or referenced from a custom-type
// schema and an event schema — resolves to a single shared alias. Callers
// rely on this dedup: every interface property that points at "device_type"
// renders as `PropertyDeviceType` regardless of where it appears.
func processPropertyEnumsIntoContext(p *plan.TrackingPlan, ctx *TSContext, nr *core.NameRegistry) error {
	properties := p.ExtractAllProperties()

	sortedNames := make([]string, 0, len(properties))
	for name := range properties {
		sortedNames = append(sortedNames, name)
	}
	sort.Strings(sortedNames)

	for _, name := range sortedNames {
		prop := properties[name]
		if !hasEnumConfig(prop.Config) {
			continue
		}

		typeName, err := getOrRegisterPropertyEnumName(prop, nr)
		if err != nil {
			return err
		}
		ctx.PropertyEnums = append(ctx.PropertyEnums, TSTypeAlias{
			Alias:   typeName,
			Type:    buildEnumUnion(prop.Config.Enum),
			Comment: prop.Description,
		})
	}

	return nil
}

// ========== Event Rule Processing ==========

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

		if len(rule.Variants) > 0 {
			if err := processEventRuleVariant(rule, ctx, nr); err != nil {
				return err
			}
			continue
		}

		if err := processOneEventRule(rule, ctx, nr); err != nil {
			return err
		}
	}

	return nil
}

func processOneEventRule(rule *plan.EventRule, ctx *TSContext, nr *core.NameRegistry) error {
	if !isEmptySchema(&rule.Schema) {
		interfaceName, err := getOrRegisterEventInterfaceName(rule, nr)
		if err != nil {
			return err
		}
		iface, err := buildEventInterface(interfaceName, rule.Event.Description, &rule.Schema, ctx, nr)
		if err != nil {
			return err
		}
		ctx.Interfaces = append(ctx.Interfaces, *iface)
	}

	method, err := buildAnalyticsMethod(rule, ctx, nr)
	if err != nil {
		return err
	}
	if method != nil {
		ctx.AnalyticsMethods = append(ctx.AnalyticsMethods, *method)
	}
	return nil
}

// isSupportedEventType lists the event types analytics-js exposes as direct
// SDK methods: track, identify, group, page. Screen is intentionally excluded —
// analytics-js has no `screen()` API (mobile-only concept), so screen rules
// emit a warning and produce no output rather than generating a method that
// can't dispatch.
func isSupportedEventType(t plan.EventType) bool {
	switch t {
	case plan.EventTypeTrack, plan.EventTypeIdentify, plan.EventTypeGroup, plan.EventTypePage:
		return true
	}
	return false
}

func validateEventSection(rule *plan.EventRule) bool {
	switch rule.Event.EventType {
	case plan.EventTypeIdentify, plan.EventTypeGroup:
		return rule.Section == plan.IdentitySectionTraits || rule.Section == plan.IdentitySectionContextTraits
	case plan.EventTypeTrack, plan.EventTypePage:
		return rule.Section == plan.IdentitySectionProperties
	}
	return false
}

// ========== Interface Builders ==========

// buildEventInterface builds the top-level interface for an event rule's
// schema. Inline nested-object schemas are hoisted into separate
// NestedInterfaces (deepest-first), so this interface only references those by
// name rather than embedding object literals.
func buildEventInterface(name, comment string, schema *plan.ObjectSchema, ctx *TSContext, nr *core.NameRegistry) (*TSInterface, error) {
	return buildInterfaceWithNested(name, comment, schema, ctx, nr)
}

// buildInterfaceWithNested builds an interface whose properties may reference
// inline nested-object schemas. Each inline schema produces a hoisted
// top-level interface named `{ParentName}{PropertyPathPascalCase}`.
func buildInterfaceWithNested(name, comment string, schema *plan.ObjectSchema, ctx *TSContext, nr *core.NameRegistry) (*TSInterface, error) {
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

		nestedTypeName, err := buildNestedInterfaceIfPresent(name, propName, &propSchema, ctx, nr)
		if err != nil {
			return nil, err
		}

		var enumName string
		if hasEnumConfig(propSchema.Property.Config) {
			enumName, err = getOrRegisterPropertyEnumName(&propSchema.Property, nr)
			if err != nil {
				return nil, err
			}
		}

		tsType, err := resolvePropertyType(&propSchema.Property, enumName, nestedTypeName, nr)
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

// buildNestedInterfaceIfPresent hoists an inline nested-object schema to a
// top-level interface and returns its name. The hoisted interface is appended
// to ctx.NestedInterfaces *after* its own children, so the slice ends up
// deepest-first.
//
// Empty inline schemas don't hoist an interface; they short-circuit to a
// `Record<string, unknown>` or `Record<string, never>` literal returned via
// the second result, which the caller passes through as the property type.
// This keeps `additionalProperties` semantics intact even when no fields are
// declared (closed-empty schemas refuse extra keys; open-empty accept any).
func buildNestedInterfaceIfPresent(parentName, propName string, propSchema *plan.PropertySchema, ctx *TSContext, nr *core.NameRegistry) (string, error) {
	if propSchema.Schema == nil {
		return "", nil
	}
	if isEmptySchema(propSchema.Schema) {
		return emptyObjectType(propSchema.Schema), nil
	}

	nestedName := FormatTypeName(parentName, propName)
	registeredName, err := nr.RegisterName("nested:"+parentName+":"+propName, globalTypeScope, nestedName)
	if err != nil {
		return "", err
	}

	iface, err := buildInterfaceWithNested(registeredName, propSchema.Property.Description, propSchema.Schema, ctx, nr)
	if err != nil {
		return "", err
	}
	ctx.NestedInterfaces = append(ctx.NestedInterfaces, *iface)
	return registeredName, nil
}

// buildInterfaceFromSchema builds an interface from a custom-type schema. It
// does not hoist nested objects — custom-type schemas declare their own
// reusable shape, so any nested objects inside them are inlined.
func buildInterfaceFromSchema(name, comment string, schema *plan.ObjectSchema, nr *core.NameRegistry) (*TSInterface, error) {
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

		var enumName string
		if hasEnumConfig(propSchema.Property.Config) {
			enumName, err = getOrRegisterPropertyEnumName(&propSchema.Property, nr)
			if err != nil {
				return nil, err
			}
		}

		tsType, err := resolvePropertyType(&propSchema.Property, enumName, "", nr)
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

func buildAnalyticsMethod(rule *plan.EventRule, ctx *TSContext, nr *core.NameRegistry) (*TSAnalyticsMethod, error) {
	switch rule.Event.EventType {
	case plan.EventTypeIdentify:
		return buildIdentifyMethod(rule, ctx, nr)
	case plan.EventTypeGroup:
		return buildGroupMethod(rule, ctx, nr)
	case plan.EventTypeTrack:
		return buildTrackMethod(rule, ctx, nr)
	case plan.EventTypePage:
		return buildPageMethod(rule, ctx, nr)
	}
	return nil, fmt.Errorf("unsupported event type: %s", rule.Event.EventType)
}

// buildIdentifyMethod emits a class method wrapping analytics-js
// `identify()`. The wrapper carries two spec-aligned overloads:
//
//	identify(userId, traits?, options?, callback?)
//	identify(traits?, options?, callback?)  // anonymous
//
// The implementation accepts union-typed positional args and dispatches by
// `typeof arg0 === "string"`. Schemas with no traits collapse to a simpler
// shape (no traits arg in either overload).
func buildIdentifyMethod(rule *plan.EventRule, ctx *TSContext, nr *core.NameRegistry) (*TSAnalyticsMethod, error) {
	methodName, err := getOrRegisterEventMethodName(rule, nr)
	if err != nil {
		return nil, err
	}

	ctx.UsesApiCallback = true
	addDataToContext := rule.Section == plan.IdentitySectionContextTraits

	traitsType := ""
	if !isEmptySchema(&rule.Schema) {
		interfaceName, err := getOrRegisterEventInterfaceName(rule, nr)
		if err != nil {
			return nil, err
		}
		traitsType = interfaceName
		if !addDataToContext {
			ctx.UsesSDKIdentifyTraits = true
		}
	}

	return buildIdentityCallMethod(identityCallSpec{
		MethodName:       methodName,
		Comment:          rule.Event.Description,
		EventName:        rule.Event.Name,
		SDKMethodName:    "identify",
		IDParamName:      "userId",
		IDArgName:        "userIdOrTraits",
		TraitsType:       traitsType,
		SDKTraitsType:    sdkIdentifyTraitsAlias,
		AllowAnonymous:   true,
		AddDataToContext: addDataToContext,
	}), nil
}

func buildTrackMethod(rule *plan.EventRule, ctx *TSContext, nr *core.NameRegistry) (*TSAnalyticsMethod, error) {
	methodName, err := getOrRegisterEventMethodName(rule, nr)
	if err != nil {
		return nil, err
	}

	// Track always carries a callback param per spec; mark the context so the
	// SDK import pulls in ApiCallback. Non-track methods set the same flag.
	ctx.UsesApiCallback = true

	eventNameLiteral := fmt.Sprintf("%q", rule.Event.Name)

	method := &TSAnalyticsMethod{
		Name:          methodName,
		Comment:       rule.Event.Description,
		EventName:     rule.Event.Name,
		SDKMethodName: "track",
		SDKArguments: []TSSDKArgument{
			{Value: eventNameLiteral},
		},
	}

	emptySchema := isEmptySchema(&rule.Schema)
	switch {
	case !emptySchema:
		interfaceName, err := getOrRegisterEventInterfaceName(rule, nr)
		if err != nil {
			return nil, err
		}
		method.MethodArguments = append(method.MethodArguments, TSMethodArgument{
			Name:    "props",
			Type:    interfaceName,
			Comment: "The properties to include with this event",
		})
		method.SDKArguments = append(method.SDKArguments, TSSDKArgument{Value: "props as unknown as " + sdkApiObjectAlias})
		ctx.UsesSDKApiObject = true

	case rule.Schema.AdditionalProperties:
		// Empty schema with allow_unplanned: true → optional generic props arg.
		method.MethodArguments = append(method.MethodArguments, TSMethodArgument{
			Name:     "props",
			Type:     openObjectType,
			Comment:  "Additional properties to include with this event",
			Optional: true,
		})
		method.SDKArguments = append(method.SDKArguments, TSSDKArgument{Value: "props as unknown as " + sdkApiObjectAlias})
		ctx.UsesSDKApiObject = true

	default:
		// Empty schema, additionalProperties: false → no props arg at all.
		// Emit an empty object literal so the wire payload is `properties: {}`
		// regardless of how the SDK serializes an `undefined` properties arg.
		method.SDKArguments = append(method.SDKArguments, TSSDKArgument{Value: "{}"})
	}

	return method, nil
}

// buildGroupMethod emits a wrapper around analytics-js
// `group(groupId, traits?, options?, callback?)`. Same shape as identify —
// see [buildIdentifyMethod].
func buildGroupMethod(rule *plan.EventRule, ctx *TSContext, nr *core.NameRegistry) (*TSAnalyticsMethod, error) {
	methodName, err := getOrRegisterEventMethodName(rule, nr)
	if err != nil {
		return nil, err
	}

	ctx.UsesApiCallback = true
	addDataToContext := rule.Section == plan.IdentitySectionContextTraits

	traitsType := ""
	if !isEmptySchema(&rule.Schema) {
		interfaceName, err := getOrRegisterEventInterfaceName(rule, nr)
		if err != nil {
			return nil, err
		}
		traitsType = interfaceName
		if !addDataToContext {
			ctx.UsesSDKIdentifyTraits = true
		}
	}

	return buildIdentityCallMethod(identityCallSpec{
		MethodName:       methodName,
		Comment:          rule.Event.Description,
		EventName:        rule.Event.Name,
		SDKMethodName:    "group",
		IDParamName:      "groupId",
		IDArgName:        "groupIdOrTraits",
		TraitsType:       traitsType,
		SDKTraitsType:    sdkIdentifyTraitsAlias,
		AllowAnonymous:   true,
		AddDataToContext: addDataToContext,
	}), nil
}

// identityCallSpec parametrises the shared overload-emitting code used by
// identify and group. The two methods have identical shape: an optional
// string identifier, optional typed traits, options, and callback.
type identityCallSpec struct {
	MethodName       string
	Comment          string
	EventName        string
	SDKMethodName    string // "identify" or "group"
	IDParamName      string // "userId" or "groupId" — name shown in the typed overloads
	IDArgName        string // "userIdOrTraits" — name of the union-typed impl param
	TraitsType       string // generated interface name; empty if the rule has no traits
	SDKTraitsType    string // SDK type alias to cast to (sdkIdentifyTraitsAlias)
	AllowAnonymous   bool   // emit the second (no-ID) overload
	AddDataToContext bool   // traits routed into options.context.traits instead of SDK traits param
}

// buildIdentityCallMethod constructs a TSAnalyticsMethod for identify/group.
// When the rule defines traits, two spec-aligned overloads are emitted
// with a two-branch dispatcher. When the rule has no traits, only the
// with-id overload is emitted with a single unconditional branch.
func buildIdentityCallMethod(spec identityCallSpec) *TSAnalyticsMethod {
	hasTraits := spec.TraitsType != ""

	if !hasTraits {
		return buildIdentityCallMethodNoTraits(spec)
	}

	// ===== Overload 1: (id, traits?, options?, callback?) =====
	o1 := []TSMethodArgument{
		{Name: spec.IDParamName, Type: "string"},
		{Name: "traits", Type: spec.TraitsType, Optional: true},
		{Name: "options", Type: "ApiOptions", Optional: true},
		{Name: "callback", Type: "ApiCallback", Optional: true},
	}

	overloads := []TSOverloadSignature{{Arguments: o1}}

	// ===== Overload 2 (anonymous): (traits?, options?, callback?) =====
	if spec.AllowAnonymous {
		o2 := []TSMethodArgument{
			{Name: "traits", Type: spec.TraitsType, Optional: true},
			{Name: "options", Type: "ApiOptions", Optional: true},
			{Name: "callback", Type: "ApiCallback", Optional: true},
		}
		overloads = append(overloads, TSOverloadSignature{Arguments: o2})
	}

	// ===== Implementation signature with union types =====
	// Names like `userIdOrTraits` make the dispatcher body readable while
	// the union types reflect both overload shapes.
	impl := []TSMethodArgument{
		{Name: spec.IDArgName, Type: joinUnion([]string{"string", spec.TraitsType}), Optional: true},
		{Name: "traitsOrOptions", Type: joinUnion([]string{spec.TraitsType, "ApiOptions"}), Optional: true},
		{Name: "optionsOrCallback", Type: joinUnion([]string{"ApiOptions", "ApiCallback"}), Optional: true},
		{Name: "callback", Type: "ApiCallback", Optional: true},
	}

	return &TSAnalyticsMethod{
		Name:               spec.MethodName,
		Comment:            spec.Comment,
		EventName:          spec.EventName,
		SDKMethodName:      spec.SDKMethodName,
		Overloads:          overloads,
		MethodArguments:    impl,
		DispatcherBranches: buildIdentityCallBranches(spec),
	}
}

// buildIdentityCallMethodNoTraits emits the simpler shape used when the plan
// rule has no traits schema. One overload, no anonymous variant, single
// unconditional dispatcher branch.
func buildIdentityCallMethodNoTraits(spec identityCallSpec) *TSAnalyticsMethod {
	overloads := []TSOverloadSignature{{
		Arguments: []TSMethodArgument{
			{Name: spec.IDParamName, Type: "string"},
			{Name: "options", Type: "ApiOptions", Optional: true},
			{Name: "callback", Type: "ApiCallback", Optional: true},
		},
	}}

	impl := []TSMethodArgument{
		{Name: spec.IDParamName, Type: "string"},
		{Name: "options", Type: "ApiOptions", Optional: true},
		{Name: "callback", Type: "ApiCallback", Optional: true},
	}

	return &TSAnalyticsMethod{
		Name:            spec.MethodName,
		Comment:         spec.Comment,
		EventName:       spec.EventName,
		SDKMethodName:   spec.SDKMethodName,
		Overloads:       overloads,
		MethodArguments: impl,
		DispatcherBranches: []TSDispatcherBranch{{
			SDKArguments: []TSSDKArgument{
				{Value: spec.IDParamName},
				{Value: "undefined"},
				{Value: "this.withRudderTyperContext(options)"},
				{Value: "callback"},
			},
		}},
	}
}

// renderIdentityCallBody emits the dispatcher body for the with-traits shape
// of identify/group. The body branches on `typeof <IDArgName> === "string"`
// to forward to either the with-ID overload or the anonymous overload at the
// SDK level.
func buildIdentityCallBranches(spec identityCallSpec) []TSDispatcherBranch {
	if spec.AddDataToContext {
		return buildIdentityCallBranchesContextTraits(spec)
	}

	traitsCast := func(argName string) string {
		return argName + " as unknown as " + spec.SDKTraitsType
	}

	withID := TSDispatcherBranch{
		Condition: fmt.Sprintf(`typeof %s === "string"`, spec.IDArgName),
		SDKArguments: []TSSDKArgument{
			{Value: spec.IDArgName},
			{Value: traitsCast("traitsOrOptions")},
			{Value: "this.withRudderTyperContext(optionsOrCallback as ApiOptions | undefined)"},
			{Value: "callback"},
		},
	}

	branches := []TSDispatcherBranch{withID}
	if spec.AllowAnonymous {
		branches = append(branches, TSDispatcherBranch{
			SDKArguments: []TSSDKArgument{
				{Value: traitsCast(spec.IDArgName)},
				{Value: "this.withRudderTyperContext(traitsOrOptions as ApiOptions | undefined)"},
				{Value: "optionsOrCallback as ApiCallback | undefined"},
			},
		})
	}
	return branches
}

func buildIdentityCallBranchesContextTraits(spec identityCallSpec) []TSDispatcherBranch {
	contextTraitsCast := func(argName string) string {
		return argName + " as unknown as SDKApiObject"
	}

	withID := TSDispatcherBranch{
		Condition: fmt.Sprintf(`typeof %s === "string"`, spec.IDArgName),
		SDKArguments: []TSSDKArgument{
			{Value: spec.IDArgName},
			{Value: "undefined"},
			{Value: "this.withRudderTyperContext(optionsOrCallback as ApiOptions | undefined, " + contextTraitsCast("traitsOrOptions") + ")"},
			{Value: "callback"},
		},
	}

	branches := []TSDispatcherBranch{withID}
	if spec.AllowAnonymous {
		branches = append(branches, TSDispatcherBranch{
			SDKArguments: []TSSDKArgument{
				{Value: "null"},
				{Value: "this.withRudderTyperContext(traitsOrOptions as ApiOptions | undefined, " + contextTraitsCast(spec.IDArgName) + ")"},
				{Value: "optionsOrCallback as ApiCallback | undefined"},
			},
		})
	}
	return branches
}

// buildPageMethod emits a singleton `page()` wrapper around analytics-js
// `page()`. Per the spec, page is a singleton — the page name lives in the
// runtime call, not the method name. Three overloads cover the SDK's call
// shapes:
//
//	page(category, name, properties?, options?, callback?)
//	page(name, properties?, options?, callback?)
//	page(properties?, options?, callback?)
//
// Only one page rule per plan is supported in this iteration — the singleton
// PageProperties interface can only represent one schema. Multiple rules
// trigger a name-registry collision (renamed `page1`, `PageProperties1`).
func buildPageMethod(rule *plan.EventRule, ctx *TSContext, nr *core.NameRegistry) (*TSAnalyticsMethod, error) {
	methodName, err := getOrRegisterEventMethodName(rule, nr)
	if err != nil {
		return nil, err
	}

	ctx.UsesApiCallback = true

	// Determine the properties type. Three plan shapes:
	//   non-empty schema   → typed interface (PageProperties)
	//   empty + additional → open Record<string, unknown>
	//   closed empty       → no properties typing (properties arg is optional & untyped)
	var (
		propsType    string
		propsSDKCast string
	)
	emptySchema := isEmptySchema(&rule.Schema)
	switch {
	case !emptySchema:
		interfaceName, err := getOrRegisterEventInterfaceName(rule, nr)
		if err != nil {
			return nil, err
		}
		propsType = interfaceName
		propsSDKCast = "as unknown as " + sdkApiObjectAlias
		ctx.UsesSDKApiObject = true
	case rule.Schema.AdditionalProperties:
		propsType = openObjectType
		propsSDKCast = "as unknown as " + sdkApiObjectAlias
		ctx.UsesSDKApiObject = true
	default:
		// Closed empty: no properties param; SDK call passes undefined.
		propsType = ""
		propsSDKCast = ""
	}

	return buildPageCallMethod(pageCallSpec{
		MethodName:    methodName,
		Comment:       rule.Event.Description,
		EventName:     rule.Event.Name,
		SDKMethodName: "page",
		PropertiesTy:  propsType,
		PropsSDKCast:  propsSDKCast,
	}), nil
}

// pageCallSpec parametrises the page overload emitter.
type pageCallSpec struct {
	MethodName    string
	Comment       string
	EventName     string
	SDKMethodName string
	// PropertiesTy is the typed properties interface name (or
	// Record<string, unknown> for open empty). Empty for closed empty
	// (the properties arg is omitted from overloads in that case).
	PropertiesTy string
	// PropsSDKCast is the cast applied when forwarding properties to the SDK
	// (e.g., "as unknown as SDKApiObject | undefined"). Empty when
	// PropertiesTy is empty.
	PropsSDKCast string
}

// buildPageCallMethod emits page() with its three spec-aligned overloads and
// a dispatcher body keyed on `typeof arg0 === "string"` and
// `typeof arg1 === "string"`. The closed-empty case (no PropertiesTy) collapses
// the properties slot out of every overload and the impl signature, so the
// dispatcher reads options/callback from the correct positions instead of
// shifting them by one. The wire-level properties arg in that case is `{}` —
// mirrors track's closed-empty behaviour for a deterministic payload.
func buildPageCallMethod(spec pageCallSpec) *TSAnalyticsMethod {
	if spec.PropertiesTy == "" {
		return buildPageCallMethodNoProps(spec)
	}

	// ===== Overloads (with properties) =====
	o1 := []TSMethodArgument{
		{Name: "category", Type: "string"},
		{Name: "name", Type: "string"},
		{Name: "properties", Type: spec.PropertiesTy, Optional: true},
		{Name: "options", Type: "ApiOptions", Optional: true},
		{Name: "callback", Type: "ApiCallback", Optional: true},
	}
	o2 := []TSMethodArgument{
		{Name: "name", Type: "string"},
		{Name: "properties", Type: spec.PropertiesTy, Optional: true},
		{Name: "options", Type: "ApiOptions", Optional: true},
		{Name: "callback", Type: "ApiCallback", Optional: true},
	}
	o3 := []TSMethodArgument{
		{Name: "properties", Type: spec.PropertiesTy, Optional: true},
		{Name: "options", Type: "ApiOptions", Optional: true},
		{Name: "callback", Type: "ApiCallback", Optional: true},
	}

	// 5 positional args cover the longest overload (category, name, props, options, callback).
	impl := []TSMethodArgument{
		{Name: "arg0", Type: joinUnion([]string{"string", spec.PropertiesTy}), Optional: true},
		{Name: "arg1", Type: joinUnion([]string{"string", spec.PropertiesTy, "ApiOptions"}), Optional: true},
		{Name: "arg2", Type: joinUnion([]string{spec.PropertiesTy, "ApiOptions", "ApiCallback"}), Optional: true},
		{Name: "arg3", Type: joinUnion([]string{"ApiOptions", "ApiCallback"}), Optional: true},
		{Name: "arg4", Type: "ApiCallback", Optional: true},
	}

	return &TSAnalyticsMethod{
		Name:               spec.MethodName,
		Comment:            spec.Comment,
		EventName:          spec.EventName,
		SDKMethodName:      spec.SDKMethodName,
		Overloads:          []TSOverloadSignature{{Arguments: o1}, {Arguments: o2}, {Arguments: o3}},
		MethodArguments:    impl,
		DispatcherBranches: buildPageCallBranches(spec),
	}
}

// buildPageCallMethodNoProps emits page() for closed-empty schemas. Every
// overload drops the properties slot, the implementation is 4-slot, and each
// branch forwards `{}` as the SDK's properties argument (matching the way
// closed-empty track sends `{}` for deterministic wire shape).
func buildPageCallMethodNoProps(spec pageCallSpec) *TSAnalyticsMethod {
	o1 := []TSMethodArgument{
		{Name: "category", Type: "string"},
		{Name: "name", Type: "string"},
		{Name: "options", Type: "ApiOptions", Optional: true},
		{Name: "callback", Type: "ApiCallback", Optional: true},
	}
	o2 := []TSMethodArgument{
		{Name: "name", Type: "string"},
		{Name: "options", Type: "ApiOptions", Optional: true},
		{Name: "callback", Type: "ApiCallback", Optional: true},
	}
	o3 := []TSMethodArgument{
		{Name: "options", Type: "ApiOptions", Optional: true},
		{Name: "callback", Type: "ApiCallback", Optional: true},
	}

	impl := []TSMethodArgument{
		{Name: "arg0", Type: joinUnion([]string{"string", "ApiOptions"}), Optional: true},
		{Name: "arg1", Type: joinUnion([]string{"string", "ApiOptions", "ApiCallback"}), Optional: true},
		{Name: "arg2", Type: joinUnion([]string{"ApiOptions", "ApiCallback"}), Optional: true},
		{Name: "arg3", Type: "ApiCallback", Optional: true},
	}

	return &TSAnalyticsMethod{
		Name:            spec.MethodName,
		Comment:         spec.Comment,
		EventName:       spec.EventName,
		SDKMethodName:   spec.SDKMethodName,
		Overloads:       []TSOverloadSignature{{Arguments: o1}, {Arguments: o2}, {Arguments: o3}},
		MethodArguments: impl,
		DispatcherBranches: []TSDispatcherBranch{
			{
				Condition: `typeof arg0 === "string" && typeof arg1 === "string"`,
				SDKArguments: []TSSDKArgument{
					{Value: "arg0"}, {Value: "arg1"}, {Value: "{}"},
					{Value: "this.withRudderTyperContext(arg2 as ApiOptions | undefined)"},
					{Value: "arg3"},
				},
			},
			{
				Condition: `typeof arg0 === "string"`,
				SDKArguments: []TSSDKArgument{
					{Value: "arg0"}, {Value: "{}"},
					{Value: "this.withRudderTyperContext(arg1 as ApiOptions | undefined)"},
					{Value: "arg2 as ApiCallback | undefined"},
				},
			},
			{
				SDKArguments: []TSSDKArgument{
					{Value: "{}"},
					{Value: "this.withRudderTyperContext(arg0 as ApiOptions | undefined)"},
					{Value: "arg1 as ApiCallback | undefined"},
				},
			},
		},
	}
}

func buildPageCallBranches(spec pageCallSpec) []TSDispatcherBranch {
	cast := func(argName string) string {
		return argName + " " + spec.PropsSDKCast
	}

	return []TSDispatcherBranch{
		{
			Condition: `typeof arg0 === "string" && typeof arg1 === "string"`,
			SDKArguments: []TSSDKArgument{
				{Value: "arg0"}, {Value: "arg1"}, {Value: cast("arg2")},
				{Value: "this.withRudderTyperContext(arg3 as ApiOptions | undefined)"},
				{Value: "arg4"},
			},
		},
		{
			Condition: `typeof arg0 === "string"`,
			SDKArguments: []TSSDKArgument{
				{Value: "arg0"}, {Value: cast("arg1")},
				{Value: "this.withRudderTyperContext(arg2 as ApiOptions | undefined)"},
				{Value: "arg3 as ApiCallback | undefined"},
			},
		},
		{
			SDKArguments: []TSSDKArgument{
				{Value: cast("arg0")},
				{Value: "this.withRudderTyperContext(arg1 as ApiOptions | undefined)"},
				{Value: "arg2 as ApiCallback | undefined"},
			},
		},
	}
}
