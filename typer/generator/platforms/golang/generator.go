package golang

import (
	"cmp"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/rudderlabs/rudder-iac/typer/generator/core"
	"github.com/rudderlabs/rudder-iac/typer/plan"
)

// Generator generates a Go package for the RudderStack Go SDK,
// github.com/rudderlabs/analytics-go/v4.
type Generator struct{}

// Generate produces one Go file from a tracking plan.
func (g *Generator) Generate(p *plan.TrackingPlan, options core.GenerateOptions, platformOptions any) ([]*core.File, error) {
	opts, err := resolveOptions(g.DefaultOptions().(GoOptions), platformOptions)
	if err != nil {
		return nil, err
	}

	ctx, err := newContext(p, options.RudderCLIVersion, opts.PackageName)
	if err != nil {
		return nil, err
	}

	content, err := render(ctx)
	if err != nil {
		return nil, err
	}
	return []*core.File{{Path: opts.OutputFileName, Content: content}}, nil
}

// resolveOptions validates the given options as they are: the CLI decodes the
// user's options onto the defaults, so an empty value was set explicitly and
// is invalid. Without options the defaults apply.
func resolveOptions(defaults GoOptions, platformOptions any) (GoOptions, error) {
	if platformOptions == nil {
		return defaults, nil
	}
	opts, ok := platformOptions.(GoOptions)
	if !ok {
		return GoOptions{}, fmt.Errorf("unexpected platform options type %T", platformOptions)
	}
	if err := opts.Validate(); err != nil {
		return GoOptions{}, err
	}
	return opts, nil
}

// trackRule is a track rule kept for generation.
type trackRule struct {
	key    string
	rule   *plan.EventRule
	schema *objectSchema
}

// objectSchema is an object schema with the fields that pass the skip check,
// in byte order of their wire keys.
type objectSchema struct {
	open bool
	// declared holds every wire key of the schema, including those of skipped
	// fields, which AdditionalProperties can never supply either.
	declared []string
	fields   []ruleField
}

// ruleField is a field kept for generation, with the effective enum of its
// property, resolved once by the resolver, which rejects an invalid one.
type ruleField struct {
	wireKey string
	ps      plan.PropertySchema
	enum    *effectiveEnum
	// nested is the field's object schema when it is hoisted into a struct of
	// its own; nil when the field takes its Property{Name} type.
	nested *objectSchema
}

// customType is a custom type that a kept field references.
type customType struct {
	ct   *plan.CustomType
	name string
	// schema is set for an object custom type generated as a struct.
	schema *objectSchema
	// reason names the construct the custom type needs that the generator
	// does not support yet; empty when it can be generated.
	reason string
}

// propertyKey identifies one Property{Name} type: the catalog allows two
// properties with the same name and different types.
type propertyKey struct{ name, signature string }

func keyOf(f ruleField) propertyKey {
	return propertyKey{f.ps.Property.Name, typeSignature(f.ps.Property, f.enum)}
}

type propertyType struct {
	name    string
	nilable bool
}

func newContext(p *plan.TrackingPlan, version, packageName string) (*GoContext, error) {
	ctx := &GoContext{
		RudderCLIVersion: version,
		PackageName:      packageName,
		PlanName:         p.Name,
		PlanURL:          p.Metadata.URL,
		PlanID:           p.Metadata.TrackingPlanID,
		PlanVersion:      p.Metadata.TrackingPlanVersion,
	}

	registry := core.NewNameRegistry(core.DefaultCollisionHandler)
	for _, name := range runtimeNames {
		if _, err := registry.RegisterName("runtime:"+name, packageScope, name); err != nil {
			return nil, fmt.Errorf("reserving %s: %w", name, err)
		}
	}
	// No event may take Alias, so a future alias rule needs no renames.
	if _, err := registry.RegisterName("reserved:Alias", methodScope, "Alias"); err != nil {
		return nil, fmt.Errorf("reserving Alias: %w", err)
	}

	rules, customTypes, err := trackRules(p)
	if err != nil {
		return nil, err
	}

	// Custom types register before property types, and those before payloads
	// and methods, so the order in which names take collision suffixes matches
	// the other generators. A custom type's declaration needs the property
	// types of its fields, so it is added after them, and so are the structs
	// it hoists, whose CustomType-prefixed names no property name can take.
	schemas := make([]*objectSchema, 0, len(customTypes)+len(rules))
	for _, c := range customTypes {
		if c.name, err = customTypeName(registry, c.ct.Name); err != nil {
			return nil, err
		}
		if c.schema != nil {
			schemas = append(schemas, c.schema)
		}
	}
	for _, r := range rules {
		schemas = append(schemas, r.schema)
	}

	propertyTypes, err := addPropertyTypes(ctx, schemas, registry)
	if err != nil {
		return nil, err
	}
	if err := addCustomTypes(ctx, customTypes, propertyTypes, registry); err != nil {
		return nil, err
	}

	ctx.out = &ctx.Payloads
	for _, r := range rules {
		if err := addTrackRule(ctx, r, propertyTypes, registry); err != nil {
			return nil, err
		}
	}

	if err := registerValueNames(ctx, registry); err != nil {
		return nil, err
	}

	ctx.QuickStart = quickStart(ctx.Methods)
	return ctx, nil
}

// quickStart picks the method the package doc calls: the first whose payload
// declares a property, else the first method.
func quickStart(methods []GoMethod) *GoMethod {
	if len(methods) == 0 {
		return nil
	}
	if i := slices.IndexFunc(methods, func(m GoMethod) bool { return m.Payload != nil && !m.Payload.MapAlias }); i >= 0 {
		return &methods[i]
	}
	return &methods[0]
}

// trackRules returns the plan's track rules in rule-key order, and the custom
// types their kept fields reference, sorted by name. Rules, variants and
// fields that need constructs the generator does not support yet are left out
// with a warning, never silently.
func trackRules(p *plan.TrackingPlan) ([]trackRule, []*customType, error) {
	all := make([]trackRule, len(p.Rules))
	for i := range p.Rules {
		rule := &p.Rules[i]
		all[i] = trackRule{
			key:  string(rule.Event.EventType) + ":" + rule.Event.Name + ":" + string(rule.Section),
			rule: rule,
		}
	}
	slices.SortStableFunc(all, func(a, b trackRule) int { return strings.Compare(a.key, b.key) })

	var (
		res    = &resolver{customTypes: map[string]*customType{}}
		cycles = &cycleCheck{visited: map[string]bool{}}
		rules  []trackRule
	)
	for _, r := range all {
		// Names and cycles are checked before the skip check, so a rule or
		// field left out for now still fails generation on a name no Go
		// identifier can carry or on a custom type that refers to itself.
		if err := validateNames(r.rule); err != nil {
			return nil, nil, err
		}
		if err := cycles.rule(r.rule); err != nil {
			return nil, nil, err
		}

		event := r.rule.Event
		if event.EventType != plan.EventTypeTrack {
			core.Warn(fmt.Sprintf("skipping the %s rule (section %q): Go generation does not support %s events yet", event.EventType, r.rule.Section, event.EventType))
			continue
		}
		if r.rule.Section != plan.IdentitySectionProperties {
			core.Warn(fmt.Sprintf("skipping track event %q: section %q is not valid for track events", event.Name, r.rule.Section))
			continue
		}
		if len(r.rule.Variants) > 0 {
			core.Warn(fmt.Sprintf("ignoring the variants of track event %q: Go generation does not support variants yet", event.Name))
		}

		var err error
		if r.schema, err = res.schema(&r.rule.Schema, fmt.Sprintf("track event %q", event.Name), ""); err != nil {
			return nil, nil, err
		}
		rules = append(rules, r)
	}

	// Like property types, only the custom types an emitted field needs are
	// generated.
	var customTypes []*customType
	for _, name := range slices.Sorted(maps.Keys(res.customTypes)) {
		if c := res.customTypes[name]; c.reason == "" {
			customTypes = append(customTypes, c)
		}
	}
	return rules, customTypes, nil
}

// resolver applies the skip check to the fields of track rules and of the
// custom types they reference, resolving each custom type once per name.
type resolver struct {
	customTypes map[string]*customType
}

// schema returns the fields of s that pass the skip check, resolving the
// custom types and nested schemas they need. where names the event or custom
// type that s belongs to, and path is the wire-key path to s within it.
func (res *resolver) schema(s *plan.ObjectSchema, where, path string) (*objectSchema, error) {
	out := &objectSchema{open: s.AdditionalProperties, declared: slices.Sorted(maps.Keys(s.Properties))}
	for _, wireKey := range out.declared {
		var (
			ps   = s.Properties[wireKey]
			p    = ps.Property
			name = path + wireKey
		)
		if ct := cmp.Or(unionCustomType(p.Types), unionCustomType(p.ItemTypes)); ct != nil {
			return nil, fmt.Errorf("mapping property %q of %s: custom type %q cannot be a member of a multi-type union", name, where, ct.Name)
		}
		reason, err := res.unsupportedReason(p)
		if err != nil {
			return nil, err
		}
		if reason != "" {
			core.Warn(fmt.Sprintf("skipping property %q (%s) of %s: Go generation does not support %s yet", name, declaredTypes(p), where, reason))
			continue
		}

		f := ruleField{wireKey: wireKey, ps: ps}
		if f.enum, err = resolveEnum(p); err != nil {
			return nil, fmt.Errorf("mapping property %q of %s: %w", name, where, err)
		}
		// The enum beats the declared types, object included.
		if f.enum == nil && declares(p.Types, plan.PrimitiveTypeObject) && isStruct(ps.Schema) {
			if f.nested, err = res.schema(ps.Schema, where, name+"."); err != nil {
				return nil, err
			}
		}
		out.fields = append(out.fields, f)
	}
	return out, nil
}

// unsupportedReason names the construct p needs that the generator does not
// support yet, or returns "" when p can be generated.
func (res *resolver) unsupportedReason(p plan.Property) (string, error) {
	var reason string
	for _, t := range slices.Concat(p.Types, p.ItemTypes) {
		ct := plan.AsCustomType(t)
		if ct == nil {
			continue
		}
		c, err := res.customType(ct)
		if err != nil {
			return "", err
		}
		reason = cmp.Or(reason, c.reason)
	}
	return reason, nil
}

// customType resolves ct, or returns the custom type of that name already
// resolved. The cycle check has rejected every custom type that refers to
// itself, so the recursion ends.
func (res *resolver) customType(ct *plan.CustomType) (*customType, error) {
	if c, ok := res.customTypes[ct.Name]; ok {
		return c, nil
	}

	var (
		c   = &customType{ct: ct}
		err error
	)
	switch {
	case len(ct.Variants) > 0:
		c.reason = "variants"
	case ct.Type == plan.PrimitiveTypeArray:
		c.reason, err = res.unsupportedReason(plan.Property{ItemTypes: []plan.PropertyType{ct.ItemType}})
	case ct.Type == plan.PrimitiveTypeObject && isStruct(ct.Schema):
		where := fmt.Sprintf("custom type %q", ct.Name)
		// Names are checked before the skip check, as in rules.
		if err = validateSchemaNames(*ct.Schema, where); err != nil {
			break
		}
		c.schema, err = res.schema(ct.Schema, where, "")
	}
	if err != nil {
		return nil, err
	}
	res.customTypes[ct.Name] = c
	return c, nil
}

// validateNames rejects a rule whose supplied event, property or custom-type
// names produce no words. Only a track event must have a name; the other event
// types are named by their type.
func validateNames(rule *plan.EventRule) error {
	event := rule.Event
	if event.EventType == plan.EventTypeTrack && event.Name == "" {
		return errors.New("a track event has an empty name")
	}
	if event.Name != "" {
		if _, err := pascalCase(event.Name); err != nil {
			return fmt.Errorf("naming %s event %q: %w", event.EventType, event.Name, err)
		}
	}
	return validateSchemaNames(rule.Schema, fmt.Sprintf("%s event %q", event.EventType, event.Name))
}

// validateSchemaNames checks the properties of schema, which belongs to where,
// and of its nested object schemas, and the custom types they reference.
func validateSchemaNames(schema plan.ObjectSchema, where string) error {
	for _, wireKey := range slices.Sorted(maps.Keys(schema.Properties)) {
		ps := schema.Properties[wireKey]
		for _, name := range []string{wireKey, ps.Property.Name} {
			if _, err := pascalCase(name); err != nil {
				return fmt.Errorf("naming property %q of %s: %w", name, where, err)
			}
		}
		for _, t := range slices.Concat(ps.Property.Types, ps.Property.ItemTypes) {
			ct := plan.AsCustomType(t)
			if ct == nil {
				continue
			}
			if _, err := pascalCase(ct.Name); err != nil {
				return fmt.Errorf("naming custom type %q of property %q: %w", ct.Name, wireKey, err)
			}
		}
		if ps.Schema == nil {
			continue
		}
		if err := validateSchemaNames(*ps.Schema, where); err != nil {
			return err
		}
	}
	return nil
}

// cycleCheck rejects a custom type that refers to itself, reached through
// any rule, variant, nested schema or array item, and fails with the first
// cycle in rule-key and wire-key order. Every rule shares one check, so each
// custom type is visited once.
type cycleCheck struct {
	// path holds the custom types being visited, outermost first.
	path    []string
	visited map[string]bool
}

func (c *cycleCheck) rule(rule *plan.EventRule) error {
	if err := c.schema(&rule.Schema); err != nil {
		return err
	}
	return c.variants(rule.Variants)
}

func (c *cycleCheck) schema(s *plan.ObjectSchema) error {
	if s == nil {
		return nil
	}
	for _, wireKey := range slices.Sorted(maps.Keys(s.Properties)) {
		ps := s.Properties[wireKey]
		for _, t := range slices.Concat(ps.Property.Types, ps.Property.ItemTypes) {
			if err := c.customType(plan.AsCustomType(t)); err != nil {
				return err
			}
		}
		if err := c.schema(ps.Schema); err != nil {
			return err
		}
	}
	return nil
}

func (c *cycleCheck) variants(variants []plan.Variant) error {
	for _, v := range variants {
		for i := range v.Cases {
			if err := c.schema(&v.Cases[i].Schema); err != nil {
				return err
			}
		}
		if err := c.schema(v.DefaultSchema); err != nil {
			return err
		}
	}
	return nil
}

// customType checks ct, when it is set, and the custom types it refers to.
func (c *cycleCheck) customType(ct *plan.CustomType) error {
	if ct == nil || c.visited[ct.Name] {
		return nil
	}
	if i := slices.Index(c.path, ct.Name); i >= 0 {
		return fmt.Errorf("custom type %q refers to itself: %s → %s", ct.Name, strings.Join(c.path[i:], " → "), ct.Name)
	}
	c.path = append(c.path, ct.Name)
	if err := c.schema(ct.Schema); err != nil {
		return err
	}
	if err := c.variants(ct.Variants); err != nil {
		return err
	}
	if err := c.customType(plan.AsCustomType(ct.ItemType)); err != nil {
		return err
	}
	c.path = c.path[:len(c.path)-1]
	// Only a custom type checked to the end is skipped later: one still on
	// the path is the cycle.
	c.visited[ct.Name] = true
	return nil
}

// isUnion reports whether types has two or more non-null types, which Go
// generates as a union struct.
func isUnion(types []plan.PropertyType) bool {
	return len(nonNull(types)) > 1
}

// unionCustomType returns the first custom type of types when they form a
// union: a custom type cannot be a union member, as in Kotlin and Swift, even
// when an enum beats the union, since the enum would drop the values only the
// custom type allows.
func unionCustomType(types []plan.PropertyType) *plan.CustomType {
	if !isUnion(types) {
		return nil
	}
	for _, t := range types {
		if ct := plan.AsCustomType(t); ct != nil {
			return ct
		}
	}
	return nil
}

// isStruct reports whether an object with schema s is generated as a struct:
// one that declares properties or is closed. Without a schema, or with an open
// one that declares none, an object is a map[string]any.
func isStruct(s *plan.ObjectSchema) bool {
	return s != nil && (len(s.Properties) > 0 || !s.AdditionalProperties)
}

func nonNull(types []plan.PropertyType) []plan.PropertyType {
	return slices.DeleteFunc(slices.Clone(types), func(t plan.PropertyType) bool { return t == plan.PrimitiveTypeNull })
}

func declaredTypes(p plan.Property) string {
	types := cmp.Or(typeList(p.Types), "any")
	if len(p.ItemTypes) > 0 {
		types += " of " + typeList(p.ItemTypes)
	}
	return types
}

// typeSignature is a property's canonical type signature: its sorted types,
// its sorted item types and the values of e, its effective enum, in tracking
// plan order.
func typeSignature(p plan.Property, e *effectiveEnum) string {
	signature := typeList(p.Types) + ";items:" + typeList(p.ItemTypes) + ";enum:"
	if e != nil {
		signature += e.signature
	}
	return signature
}

func typeList(types []plan.PropertyType) string {
	names := make([]string, 0, len(types))
	for _, t := range types {
		if ct := plan.AsCustomType(t); ct != nil {
			names = append(names, "custom:"+ct.Name)
			continue
		}
		names = append(names, fmt.Sprint(t))
	}
	slices.Sort(names)
	return strings.Join(names, "|")
}

// addPropertyType adds the type of prop under its registered name, with lead
// starting its doc comment: its effective enum e when it has one, which beats
// its declared types, else a union for two or more non-null types, else an
// alias. A union or alias follows the union its array items need, so the file
// declares an item type before the type that holds it. It reports whether the
// type can hold nil (slice, map or interface).
func addPropertyType(ctx *GoContext, registry *core.NameRegistry, name, lead string, prop plan.Property, e *effectiveEnum) (bool, error) {
	if e != nil {
		return false, addEnum(ctx, registry, name, lead, prop, e)
	}
	if isUnion(prop.Types) {
		return false, addUnion(ctx, registry, name, lead, prop.Description, prop.Types, prop.ItemTypes, "")
	}
	if len(prop.Types) == 0 {
		// Callers send an explicit null in an any as Null{}.
		ctx.UsesNull = true
	}
	typ, nilable, err := goType(ctx, registry, name, prop.Types, prop.ItemTypes)
	if err != nil {
		return false, err
	}
	ctx.add(GoDecl{Alias: &GoTypeAlias{
		Name: name,
		Doc:  withDescription(lead+".", prop.Description),
		Type: typ,
	}})
	return nilable, nil
}

// goType maps a type list with at most one non-null type to Go, and reports
// whether the Go type can hold nil. owner names the union that array items of
// several types need.
func goType(ctx *GoContext, registry *core.NameRegistry, owner string, types, items []plan.PropertyType) (string, bool, error) {
	values := nonNull(types)
	switch {
	case len(types) == 0:
		return "any", true, nil
	case len(values) == 0:
		ctx.UsesNull = true
		return "Null", false, nil
	}
	typ, nilable, err := memberType(ctx, registry, owner, values[0], items)
	if err != nil {
		return "", false, err
	}
	if len(values) == len(types) {
		return typ, nilable, nil
	}
	ctx.UsesNullable = true
	return "Nullable[" + typ + "]", false, nil
}

// memberType maps one non-null JSON type or a custom type to Go, and reports
// whether the Go type can hold nil.
func memberType(ctx *GoContext, registry *core.NameRegistry, owner string, t plan.PropertyType, items []plan.PropertyType) (string, bool, error) {
	if ct := plan.AsCustomType(t); ct != nil {
		name, err := customTypeName(registry, ct.Name)
		// Array custom types are slices, and object ones without a struct maps.
		nilable := ct.Type == plan.PrimitiveTypeArray || ct.Type == plan.PrimitiveTypeObject && !isStruct(ct.Schema)
		return name, nilable, err
	}
	switch t {
	case plan.PrimitiveTypeString:
		return "string", false, nil
	case plan.PrimitiveTypeInteger:
		return "int64", false, nil
	case plan.PrimitiveTypeNumber:
		return "float64", false, nil
	case plan.PrimitiveTypeBoolean:
		return "bool", false, nil
	case plan.PrimitiveTypeObject:
		return "map[string]any", true, nil
	case plan.PrimitiveTypeArray:
		typ, err := arrayType(ctx, registry, owner, items)
		return typ, true, err
	}
	return "", false, fmt.Errorf("unsupported type %v", t)
}

// arrayType maps an array with the given item types to Go. Items of two or
// more non-null types get the union {owner}Item, registered right after its
// owner, as a payload's hoisted structs follow the payload.
func arrayType(ctx *GoContext, registry *core.NameRegistry, owner string, items []plan.PropertyType) (string, error) {
	if !isUnion(items) {
		typ, _, err := goType(ctx, registry, owner, items, nil)
		return "[]" + typ, err
	}
	name, err := registry.RegisterName("item:"+owner, packageScope, owner+"Item")
	if err != nil {
		return "", fmt.Errorf("registering the item union of %s: %w", owner, err)
	}
	if err := addUnion(ctx, registry, name, fmt.Sprintf("%s is one item of %s", name, owner), "", items, nil, ""); err != nil {
		return "", fmt.Errorf("mapping the item types of %s: %w", owner, err)
	}
	return "[]" + name, nil
}

// unionMembers are the JSON types a union can hold, in constructor order.
var unionMembers = []struct {
	typ     plan.PrimitiveType
	name    string
	article string
	nilNote string
}{
	{plan.PrimitiveTypeString, "String", "a", ""},
	{plan.PrimitiveTypeInteger, "Integer", "an", ""},
	{plan.PrimitiveTypeNumber, "Number", "a", ""},
	{plan.PrimitiveTypeBoolean, "Boolean", "a", ""},
	{plan.PrimitiveTypeObject, "Object", "an", "; nil is sent as {}"},
	{plan.PrimitiveTypeArray, "Array", "an", "; nil is sent as []"},
	{plan.PrimitiveTypeNull, "Null", "the", ""},
}

// addUnion adds the union name of two or more non-null types, after any union
// its array member's items need. lead starts its doc comment, and object, when
// set, is the hoisted struct its Object member takes instead of a map. Its
// constructor names are registered later, by registerValueNames.
func addUnion(ctx *GoContext, registry *core.NameRegistry, name, lead, description string, types, items []plan.PropertyType, object string) error {
	var (
		union = &GoUnion{Name: name, HasNull: slices.Contains(types, plan.PropertyType(plan.PrimitiveTypeNull))}
		// The doc lists the member types: "a string, integer or null".
		article string
		words   []string
	)
	for _, m := range unionMembers {
		if !slices.Contains(types, plan.PropertyType(m.typ)) {
			continue
		}
		article = cmp.Or(article, m.article)
		words = append(words, string(m.typ))

		var (
			member  = GoUnionMember{Name: "New" + name + m.name}
			nilNote = m.nilNote
			err     error
		)
		switch {
		case m.typ == plan.PrimitiveTypeNull:
		case m.typ == plan.PrimitiveTypeObject && object != "":
			// A struct cannot be nil.
			member.Type, nilNote = object, ""
		default:
			if member.Type, _, err = memberType(ctx, registry, name, m.typ, items); err != nil {
				return err
			}
		}
		member.Doc = fmt.Sprintf("returns %s %s value%s.", m.article, m.typ, nilNote)
		union.Members = append(union.Members, member)
	}

	var (
		last = len(words) - 1
		zero = "Null is not a member, so the zero value cannot be sent."
	)
	if union.HasNull {
		zero = "Null is a member, so the zero value is null."
	}
	union.Doc = withDescription(fmt.Sprintf("%s: %s %s or %s.\nBuild it with one of the New%s* functions.\n%s",
		lead, article, strings.Join(words[:last], ", "), words[last], name, zero), description)
	ctx.add(GoDecl{Union: union})
	return nil
}

// registerValueNames is the second registration pass, for the functions and
// constants named from plan data: every enum's {Type}Values function, then the
// union constructors and enum constants or value functions. It runs after
// every type name is registered, so none of them can take a type's name, and
// a constant cannot take another enum's {Type}Values.
func registerValueNames(ctx *GoContext, registry *core.NameRegistry) error {
	register := func(id string, name *string) error {
		registered, err := registry.RegisterName(id, packageScope, *name)
		if err != nil {
			return fmt.Errorf("registering %s: %w", *name, err)
		}
		*name = registered
		return nil
	}
	decls := slices.Concat(ctx.CustomTypes, ctx.PropertyTypes, ctx.Payloads)
	for _, d := range decls {
		if e := d.Enum; e != nil {
			if err := register("values:"+e.Name, &e.ValuesFunc); err != nil {
				return err
			}
		}
	}
	for _, d := range decls {
		if u := d.Union; u != nil {
			for i := range u.Members {
				if err := register("constructor:"+u.Members[i].Name, &u.Members[i].Name); err != nil {
					return err
				}
			}
		}
		if e := d.Enum; e != nil {
			for i := range e.Values {
				if err := register(fmt.Sprintf("value:%s:%d", e.Name, i), &e.Values[i].Name); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// addPropertyTypes adds one Property{Name} type per (name, type signature)
// that an emitted field of schemas references, ordered and registered by name,
// then signature. A field whose nested schema is hoisted references none.
// Descriptions come from the first field that references them, walking
// schemas in order and each depth-first.
func addPropertyTypes(ctx *GoContext, schemas []*objectSchema, registry *core.NameRegistry) (map[propertyKey]propertyType, error) {
	ctx.out = &ctx.PropertyTypes
	first := map[propertyKey]ruleField{}
	for _, s := range schemas {
		firstFields(s, first)
	}

	keys := slices.SortedFunc(maps.Keys(first), func(a, b propertyKey) int {
		return cmp.Or(strings.Compare(a.name, b.name), strings.Compare(a.signature, b.signature))
	})

	types := make(map[propertyKey]propertyType, len(keys))
	for _, k := range keys {
		f := first[k]
		prop := f.ps.Property
		base, err := pascalCase(prop.Name)
		if err != nil {
			return nil, fmt.Errorf("naming property %q: %w", prop.Name, err)
		}
		name, err := registry.RegisterName("property:"+k.name+":"+k.signature, packageScope, "Property"+base)
		if err != nil {
			return nil, fmt.Errorf("registering property %q: %w", prop.Name, err)
		}
		lead := fmt.Sprintf("%s represents the property %s", name, strconv.Quote(prop.Name))
		nilable, err := addPropertyType(ctx, registry, name, lead, prop, f.enum)
		if err != nil {
			return nil, fmt.Errorf("mapping property %q: %w", prop.Name, err)
		}
		types[k] = propertyType{name: name, nilable: nilable}
	}
	return types, nil
}

// firstFields records in first, per property key, the first field of s or of
// a schema it hoists that takes a Property{Name} type.
func firstFields(s *objectSchema, first map[propertyKey]ruleField) {
	for _, f := range s.fields {
		if f.nested != nil {
			firstFields(f.nested, first)
			continue
		}
		k := keyOf(f)
		if _, seen := first[k]; !seen {
			first[k] = f
		}
	}
}

// customTypeName registers the type name of the custom type name, or returns
// the one registered: every reference resolves to one CustomType{Name}.
func customTypeName(registry *core.NameRegistry, name string) (string, error) {
	base, err := pascalCase(name)
	if err != nil {
		return "", fmt.Errorf("naming custom type %q: %w", name, err)
	}
	registered, err := registry.RegisterName("customtype:"+name, packageScope, "CustomType"+base)
	if err != nil {
		return "", fmt.Errorf("registering custom type %q: %w", name, err)
	}
	return registered, nil
}

// addCustomTypes adds the custom types, sorted by name, each followed by the
// types its hoisted fields need: a struct for an object that declares
// properties or is closed, else, as for a property, an enum or an alias.
func addCustomTypes(ctx *GoContext, customTypes []*customType, propertyTypes map[propertyKey]propertyType, registry *core.NameRegistry) error {
	ctx.out = &ctx.CustomTypes
	for _, c := range customTypes {
		ct := c.ct
		lead := fmt.Sprintf("%s represents the custom type %s", c.name, strconv.Quote(ct.Name))
		if c.schema != nil {
			if _, err := addStruct(ctx, registry, propertyTypes, c.name, withDescription(lead+".", ct.Description), c.schema, false); err != nil {
				return err
			}
			continue
		}

		prop := plan.Property{Name: ct.Name, Description: ct.Description, Types: []plan.PropertyType{ct.Type}, Config: ct.Config}
		if ct.ItemType != nil {
			prop.ItemTypes = []plan.PropertyType{ct.ItemType}
		}
		e, err := resolveEnum(prop)
		if err != nil {
			return fmt.Errorf("mapping custom type %q: %w", ct.Name, err)
		}
		if _, err := addPropertyType(ctx, registry, c.name, lead, prop, e); err != nil {
			return fmt.Errorf("mapping custom type %q: %w", ct.Name, err)
		}
	}
	return nil
}

// addTrackRule adds the payload type and method of one track rule.
func addTrackRule(ctx *GoContext, r trackRule, propertyTypes map[propertyKey]propertyType, registry *core.NameRegistry) error {
	event := r.rule.Event
	base, err := pascalCase(event.Name)
	if err != nil {
		return fmt.Errorf("naming track event %q: %w", event.Name, err)
	}

	payload, err := addPayload(ctx, r, base, propertyTypes, registry)
	if err != nil {
		return err
	}

	method, err := registry.RegisterName("method:"+r.key, methodScope, "Track"+base)
	if err != nil {
		return fmt.Errorf("registering the method of track event %q: %w", event.Name, err)
	}
	summary := fmt.Sprintf("%s sends the track event %s.", method, strconv.Quote(event.Name))
	switch {
	case payload == nil:
		summary = fmt.Sprintf("%s sends the track event %s, which has no properties.", method, strconv.Quote(event.Name))
	case payload.MapAlias:
		summary += " props may be nil; it is copied, so it can be reused after the call."
	}
	ctx.Methods = append(ctx.Methods, GoMethod{
		Name:    method,
		Doc:     withDescription(summary, event.Description),
		Event:   event.Name,
		Payload: payload,
	})
	return nil
}

// addPayload adds the payload type of a track rule whose event name
// pascal-cases to base, or returns nil when the rule's schema is empty and
// closed.
func addPayload(ctx *GoContext, r trackRule, base string, propertyTypes map[propertyKey]propertyType, registry *core.NameRegistry) (*GoStruct, error) {
	var (
		event  = r.rule.Event
		schema = r.rule.Schema
	)
	if len(schema.Properties) == 0 && !schema.AdditionalProperties {
		return nil, nil
	}

	name, err := registry.RegisterName("payload:"+r.key, packageScope, "Track"+base+"Properties")
	if err != nil {
		return nil, fmt.Errorf("registering the payload of track event %q: %w", event.Name, err)
	}
	if len(schema.Properties) > 0 {
		return addStruct(ctx, registry, propertyTypes, name, fmt.Sprintf("%s holds the properties of %s.", name, strconv.Quote(event.Name)), r.schema, true)
	}
	payload := &GoStruct{
		Name:     name,
		Doc:      fmt.Sprintf("%s holds the properties of %s. The tracking plan declares no properties and allows any.", name, strconv.Quote(event.Name)),
		MapAlias: true,
	}
	ctx.add(GoDecl{Struct: payload})
	return payload, nil
}

// addStruct adds the struct name of the fields of s, then the types its
// hoisted fields need, depth-first in wire-key order, so each follows its
// parent. Fields are named in byte order of their wire keys, after the names
// the struct's own methods and its AdditionalProperties field already hold.
// A payload struct also has ToProperties.
func addStruct(ctx *GoContext, registry *core.NameRegistry, propertyTypes map[propertyKey]propertyType, name, doc string, s *objectSchema, payload bool) (*GoStruct, error) {
	st := &GoStruct{Name: name, Doc: doc, Payload: payload, Open: s.open, DeclaredKeys: s.declared}
	ctx.add(GoDecl{Struct: st})
	ctx.UsesWithAdditional = ctx.UsesWithAdditional || s.open

	var (
		scope    = "struct:" + name + ":fields"
		reserved = []string{"MarshalJSON"}
	)
	if payload {
		reserved = append(reserved, "ToProperties")
	}
	if s.open {
		reserved = append(reserved, "AdditionalProperties")
	}
	for _, r := range reserved {
		if _, err := registry.RegisterName("reserved:"+r, scope, r); err != nil {
			return nil, fmt.Errorf("reserving %s in %s: %w", r, name, err)
		}
	}

	for _, f := range s.fields {
		base, err := fieldName(f.wireKey)
		if err != nil {
			return nil, fmt.Errorf("naming property %q of %s: %w", f.wireKey, name, err)
		}
		field, err := registry.RegisterName("key:"+f.wireKey, scope, base)
		if err != nil {
			return nil, fmt.Errorf("registering property %q of %s: %w", f.wireKey, name, err)
		}

		pt := propertyTypes[keyOf(f)]
		if f.nested != nil {
			typ, err := addNested(ctx, registry, propertyTypes, name, field, f)
			if err != nil {
				return nil, err
			}
			// A hoisted struct, a Nullable of one and a union cannot be nil.
			pt = propertyType{name: typ}
		}
		goField := GoField{Name: field, Key: f.wireKey, Type: pt.name, Required: f.ps.Required}
		// Optional fields need an absent state; slices, maps and interfaces
		// already have one in nil.
		if !f.ps.Required && !pt.nilable {
			goField.Pointer, goField.Type = true, "*"+pt.name
			ctx.UsesPtr = true
		}
		st.Fields = append(st.Fields, goField)
	}
	return st, nil
}

// addNested adds the types that field of parent hoists from f's nested schema
// and returns the field's type: the struct {Parent}{Field}, a Nullable of it
// for object or null, or for a union the union {Parent}{Field}, whose Object
// member takes the struct {Parent}{Field}Object. Each hoisted name is
// registered under its parent and wire key, so names that concatenate alike
// (foo.barBaz and fooBar.baz) take a numeric suffix in registration order.
func addNested(ctx *GoContext, registry *core.NameRegistry, propertyTypes map[propertyKey]propertyType, parent, field string, f ruleField) (string, error) {
	name, err := registry.RegisterName("nested:"+parent+":"+f.wireKey, packageScope, parent+field)
	if err != nil {
		return "", fmt.Errorf("registering the type of %s.%s: %w", parent, field, err)
	}

	var (
		types  = f.ps.Property.Types
		object = name
	)
	if isUnion(types) {
		if object, err = registry.RegisterName("object:"+name, packageScope, name+"Object"); err != nil {
			return "", fmt.Errorf("registering the object member of %s: %w", name, err)
		}
		lead := fmt.Sprintf("%s is the value in %s.%s", name, parent, field)
		if err := addUnion(ctx, registry, name, lead, "", types, f.ps.Property.ItemTypes, object); err != nil {
			return "", fmt.Errorf("mapping %s.%s: %w", parent, field, err)
		}
	}
	if _, err := addStruct(ctx, registry, propertyTypes, object, fmt.Sprintf("%s is the object in %s.%s.", object, parent, field), f.nested, false); err != nil {
		return "", err
	}

	if isUnion(types) || !declares(types, plan.PrimitiveTypeNull) {
		return name, nil
	}
	ctx.UsesNullable = true
	return "Nullable[" + name + "]", nil
}

func withDescription(summary, description string) string {
	if strings.TrimSpace(description) == "" {
		return summary
	}
	return summary + "\n\n" + description
}
