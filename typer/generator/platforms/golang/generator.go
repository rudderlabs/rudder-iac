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

// trackRule is a track rule kept for generation, with the wire keys of the
// fields that pass the skip check, in byte order.
type trackRule struct {
	key    string
	rule   *plan.EventRule
	fields []string
}

// propertyKey identifies one Property{Name} type: the catalog allows two
// properties with the same name and different types.
type propertyKey struct{ name, signature string }

func keyOf(p plan.Property) propertyKey {
	return propertyKey{p.Name, typeSignature(p)}
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

	rules, err := trackRules(p)
	if err != nil {
		return nil, err
	}

	// Property types register before payloads and methods, so the order in
	// which names take collision suffixes matches the other generators.
	propertyTypes, err := addPropertyTypes(ctx, rules, registry)
	if err != nil {
		return nil, err
	}

	for _, r := range rules {
		if err := addTrackRule(ctx, r, propertyTypes, registry); err != nil {
			return nil, err
		}
	}

	for _, payload := range ctx.Payloads {
		ctx.UsesWithAdditional = ctx.UsesWithAdditional || payload.Open
		ctx.UsesPtr = ctx.UsesPtr || slices.ContainsFunc(payload.Fields, func(f GoField) bool { return f.Pointer })
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

// trackRules returns the plan's track rules in rule-key order. Rules, variants
// and fields that need constructs the generator does not support yet are left
// out with a warning, never silently.
func trackRules(p *plan.TrackingPlan) ([]trackRule, error) {
	all := make([]trackRule, len(p.Rules))
	for i := range p.Rules {
		rule := &p.Rules[i]
		all[i] = trackRule{
			key:  string(rule.Event.EventType) + ":" + rule.Event.Name + ":" + string(rule.Section),
			rule: rule,
		}
	}
	slices.SortStableFunc(all, func(a, b trackRule) int { return strings.Compare(a.key, b.key) })

	var rules []trackRule
	for _, r := range all {
		// Names are checked before the skip check, so a rule or field left out
		// for now still fails generation on a name no Go identifier can carry.
		if err := validateNames(r.rule); err != nil {
			return nil, err
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

		for _, wireKey := range slices.Sorted(maps.Keys(r.rule.Schema.Properties)) {
			ps := r.rule.Schema.Properties[wireKey]
			if reason := unsupportedReason(ps); reason != "" {
				core.Warn(fmt.Sprintf("skipping property %q (%s) of track event %q: Go generation does not support %s yet", wireKey, declaredTypes(ps.Property), event.Name, reason))
				continue
			}
			r.fields = append(r.fields, wireKey)
		}
		rules = append(rules, r)
	}
	return rules, nil
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
	return validateSchemaNames(rule.Schema, event)
}

// validateSchemaNames checks the properties of schema and of its nested object
// schemas, and the custom types they reference.
func validateSchemaNames(schema plan.ObjectSchema, event plan.Event) error {
	for _, wireKey := range slices.Sorted(maps.Keys(schema.Properties)) {
		ps := schema.Properties[wireKey]
		for _, name := range []string{wireKey, ps.Property.Name} {
			if _, err := pascalCase(name); err != nil {
				return fmt.Errorf("naming property %q of %s event %q: %w", name, event.EventType, event.Name, err)
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
		if err := validateSchemaNames(*ps.Schema, event); err != nil {
			return err
		}
	}
	return nil
}

// unsupportedReason names the construct a field needs that a later version of
// the generator adds, or returns "" when the field can be generated.
func unsupportedReason(ps plan.PropertySchema) string {
	p := ps.Property
	switch {
	case ps.Schema != nil:
		return "nested object schemas"
	case p.Config != nil && len(p.Config.Enum) > 0:
		return "enums"
	case len(p.Types) > 1:
		return "multi-type properties"
	case len(p.ItemTypes) > 0:
		return "array item types"
	case len(p.Types) == 1 && plan.IsCustomType(p.Types[0]):
		return "custom types"
	case len(p.Types) == 1 && p.Types[0] == plan.PrimitiveTypeNull:
		return "the null type"
	}
	return ""
}

func declaredTypes(p plan.Property) string {
	types := cmp.Or(typeList(p.Types), "any")
	if len(p.ItemTypes) > 0 {
		types += " of " + typeList(p.ItemTypes)
	}
	return types
}

// typeSignature is a property's canonical type signature: its sorted types,
// its sorted item types and its enum values, always empty while the skip check
// drops enum fields.
func typeSignature(p plan.Property) string {
	return typeList(p.Types) + ";items:" + typeList(p.ItemTypes) + ";enum:"
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

// goType maps a type the skip check lets through to Go, and reports whether
// that Go type can hold nil (slice, map or interface).
func goType(p plan.Property) (string, bool, error) {
	if len(p.Types) == 0 {
		return "any", true, nil
	}
	switch p.Types[0] {
	case plan.PrimitiveTypeString:
		return "string", false, nil
	case plan.PrimitiveTypeInteger:
		return "int64", false, nil
	case plan.PrimitiveTypeNumber:
		return "float64", false, nil
	case plan.PrimitiveTypeBoolean:
		return "bool", false, nil
	case plan.PrimitiveTypeArray:
		return "[]any", true, nil
	case plan.PrimitiveTypeObject:
		return "map[string]any", true, nil
	}
	return "", false, fmt.Errorf("unsupported type %v of property %q", p.Types[0], p.Name)
}

// addPropertyTypes adds one Property{Name} alias per (name, type signature)
// that an emitted field references, ordered and registered by name, then
// signature. Descriptions come from the first field that references them.
func addPropertyTypes(ctx *GoContext, rules []trackRule, registry *core.NameRegistry) (map[propertyKey]propertyType, error) {
	first := map[propertyKey]plan.Property{}
	for _, r := range rules {
		for _, wireKey := range r.fields {
			prop := r.rule.Schema.Properties[wireKey].Property
			k := keyOf(prop)
			if _, seen := first[k]; !seen {
				first[k] = prop
			}
		}
	}

	keys := slices.SortedFunc(maps.Keys(first), func(a, b propertyKey) int {
		return cmp.Or(strings.Compare(a.name, b.name), strings.Compare(a.signature, b.signature))
	})

	types := make(map[propertyKey]propertyType, len(keys))
	for _, k := range keys {
		prop := first[k]
		base, err := pascalCase(prop.Name)
		if err != nil {
			return nil, fmt.Errorf("naming property %q: %w", prop.Name, err)
		}
		name, err := registry.RegisterName("property:"+k.name+":"+k.signature, packageScope, "Property"+base)
		if err != nil {
			return nil, fmt.Errorf("registering property %q: %w", prop.Name, err)
		}
		typ, nilable, err := goType(prop)
		if err != nil {
			return nil, err
		}

		ctx.PropertyTypes = append(ctx.PropertyTypes, GoTypeAlias{
			Name: name,
			Doc:  withDescription(fmt.Sprintf("%s represents the property %s.", name, strconv.Quote(prop.Name)), prop.Description),
			Type: typ,
		})
		types[k] = propertyType{name: name, nilable: nilable}
	}
	return types, nil
}

// addTrackRule adds the payload type and method of one track rule.
func addTrackRule(ctx *GoContext, r trackRule, propertyTypes map[propertyKey]propertyType, registry *core.NameRegistry) error {
	event := r.rule.Event
	base, err := pascalCase(event.Name)
	if err != nil {
		return fmt.Errorf("naming track event %q: %w", event.Name, err)
	}

	payload, err := newPayload(r, base, propertyTypes, registry)
	if err != nil {
		return err
	}
	if payload != nil {
		ctx.Payloads = append(ctx.Payloads, payload)
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

// newPayload registers the payload type of a track rule whose event name
// pascal-cases to base, or returns nil when the rule's schema is empty and
// closed.
func newPayload(r trackRule, base string, propertyTypes map[propertyKey]propertyType, registry *core.NameRegistry) (*GoPayload, error) {
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
	if len(schema.Properties) == 0 {
		return &GoPayload{
			Name:     name,
			Doc:      fmt.Sprintf("%s holds the properties of %s. The tracking plan declares no properties and allows any.", name, strconv.Quote(event.Name)),
			MapAlias: true,
		}, nil
	}

	payload := &GoPayload{
		Name:         name,
		Doc:          fmt.Sprintf("%s holds the properties of %s.", name, strconv.Quote(event.Name)),
		Open:         schema.AdditionalProperties,
		DeclaredKeys: slices.Sorted(maps.Keys(schema.Properties)),
	}
	if payload.Fields, err = structFields(r, payload, propertyTypes, registry); err != nil {
		return nil, err
	}
	return payload, nil
}

// structFields names the fields of a payload struct in byte order of their
// wire keys, after the names the struct's own methods and its
// AdditionalProperties field already hold.
func structFields(r trackRule, payload *GoPayload, propertyTypes map[propertyKey]propertyType, registry *core.NameRegistry) ([]GoField, error) {
	var (
		scope    = "struct:" + payload.Name + ":fields"
		reserved = []string{"MarshalJSON", "ToProperties"}
	)
	if payload.Open {
		reserved = append(reserved, "AdditionalProperties")
	}
	for _, name := range reserved {
		if _, err := registry.RegisterName("reserved:"+name, scope, name); err != nil {
			return nil, fmt.Errorf("reserving %s in %s: %w", name, payload.Name, err)
		}
	}

	fields := make([]GoField, 0, len(r.fields))
	for _, wireKey := range r.fields {
		ps := r.rule.Schema.Properties[wireKey]
		base, err := fieldName(wireKey)
		if err != nil {
			return nil, fmt.Errorf("naming property %q of track event %q: %w", wireKey, r.rule.Event.Name, err)
		}
		name, err := registry.RegisterName("key:"+wireKey, scope, base)
		if err != nil {
			return nil, fmt.Errorf("registering property %q of track event %q: %w", wireKey, r.rule.Event.Name, err)
		}

		pt := propertyTypes[keyOf(ps.Property)]
		field := GoField{Name: name, Key: wireKey, Type: pt.name, Required: ps.Required}
		// Optional fields need an absent state; slices, maps and interfaces
		// already have one in nil.
		if !ps.Required && !pt.nilable {
			field.Pointer = true
			field.Type = "*" + pt.name
		}
		fields = append(fields, field)
	}
	return fields, nil
}

func withDescription(summary, description string) string {
	if strings.TrimSpace(description) == "" {
		return summary
	}
	return summary + "\n\n" + description
}
