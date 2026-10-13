package golang

import (
	"cmp"
	"fmt"
	"maps"
	"math/big"
	"slices"
	"strconv"
	"strings"

	"github.com/rudderlabs/rudder-iac/typer/generator/core"
	"github.com/rudderlabs/rudder-iac/typer/plan"
)

// variantSchema is the discriminated union of a rule or custom type: its base
// schema, and each branch merged with it.
type variantSchema struct {
	// key is the discriminator's wire key.
	key      string
	optional bool
	domain   matchDomain
	base     *objectSchema
	// absent is set when the discriminator is optional.
	absent *objectSchema
	cases  []variantCase
	// dflt is nil when the cases cover every value the discriminator takes.
	dflt *objectSchema
}

// variantCase is one match value of a case: Go generates a struct per value,
// so that each struct's discriminator is fixed.
type variantCase struct {
	vc *plan.VariantCase
	// value is the normalized match value, text its Go literal and name the
	// suffix of its struct's name.
	value      any
	text, name string
	schema     *objectSchema
}

// schemas are the base schema and the merged branches, whose fields need
// property types.
func (v *variantSchema) schemas() []*objectSchema {
	out := []*objectSchema{v.base}
	for _, c := range v.cases {
		out = append(out, c.schema)
	}
	if v.dflt != nil {
		out = append(out, v.dflt)
	}
	return out
}

// matchDomain is the values a discriminator takes: its effective enum, or
// else any value of its non-null types.
type matchDomain struct {
	types    []plan.PropertyType
	enum     *effectiveEnum
	nullable bool
}

// domainOf returns the match domain of a discriminator of property p, with a
// custom type resolved to the type it stands for.
func domainOf(p plan.Property) matchDomain {
	if e, _ := resolveEnum(p); e != nil { // the resolver reports an invalid enum
		return matchDomain{types: nonNull(p.Types), enum: e, nullable: e.null}
	}
	for _, t := range p.Types {
		ct := plan.AsCustomType(t)
		if ct == nil {
			continue
		}
		// A custom type is never a union member, so it is the only non-null type.
		d := domainOf(plan.Property{Types: []plan.PropertyType{ct.Type}, Config: ct.Config})
		d.nullable = d.nullable || declares(p.Types, plan.PrimitiveTypeNull)
		return d
	}
	return matchDomain{types: nonNull(p.Types), nullable: declares(p.Types, plan.PrimitiveTypeNull)}
}

func (d matchDomain) allows(value any, text string) bool {
	if d.enum == nil {
		return allows(d.types, value)
	}
	return slices.ContainsFunc(d.enum.values, func(v GoEnumValue) bool { return v.Text == text })
}

// exhausts reports whether matched, the Go literals of the match values,
// holds every non-null value: both booleans, or every value of an enum.
func (d matchDomain) exhausts(matched map[string]bool) bool {
	if d.enum != nil {
		return !slices.ContainsFunc(d.enum.values, func(v GoEnumValue) bool { return !matched[v.Text] })
	}
	return slices.Equal(d.types, []plan.PropertyType{plan.PrimitiveTypeBoolean}) && matched["true"] && matched["false"]
}

// variant resolves the variants of a rule or custom type, where, whose base
// schema s resolves to base; nil when it has none.
func (res *resolver) variant(s *plan.ObjectSchema, base *objectSchema, variants []plan.Variant, where string) (*variantSchema, error) {
	if len(variants) == 0 {
		return nil, nil
	}
	v, err := res.mergeVariant(s, base, variants, where)
	if err != nil {
		return nil, fmt.Errorf("mapping the variants of %s: %w", where, err)
	}
	return v, nil
}

// mergeVariant checks every match value against the discriminator's domain
// and merges each branch into the base schema. Default is left out when the
// cases cover every value, since it could then hold only invalid ones.
func (res *resolver) mergeVariant(s *plan.ObjectSchema, base *objectSchema, variants []plan.Variant, where string) (*variantSchema, error) {
	if len(variants) > 1 {
		return nil, fmt.Errorf("found %d variants entries; a rule or custom type may have one", len(variants))
	}
	vr := variants[0]
	disc, ok := s.Properties[vr.Discriminator]
	if !ok {
		return nil, fmt.Errorf("discriminator %q is not a property of the base schema", vr.Discriminator)
	}
	d := domainOf(disc.Property)
	if !declares(d.types, plan.PrimitiveTypeString) && !declares(d.types, plan.PrimitiveTypeInteger) && !declares(d.types, plan.PrimitiveTypeBoolean) {
		return nil, fmt.Errorf("discriminator %q (%s) allows none of string, integer and boolean", vr.Discriminator, declaredTypes(disc.Property))
	}

	var (
		v       = &variantSchema{key: vr.Discriminator, optional: !disc.Required, domain: d, base: base}
		matched = map[string]bool{}
	)
	for i := range vr.Cases {
		var (
			vc    = &vr.Cases[i]
			label = "case " + jsonText(vc.Match)
		)
		schema, err := res.branch(s, &vc.Schema, label, where)
		if err != nil {
			return nil, err
		}
		// toMap sends the discriminator a case fixes.
		schema.fields = withoutField(schema.fields, v.key)
		for _, m := range vc.Match {
			value, text, name, err := matchValue(m, label)
			switch {
			case err != nil:
				return nil, err
			case !d.allows(value, text):
				return nil, fmt.Errorf("%s: match value %s is not a value of discriminator %q", label, text, v.key)
			case matched[text]:
				return nil, fmt.Errorf("%s: match value %s appears more than once", label, text)
			}
			matched[text] = true
			v.cases = append(v.cases, variantCase{vc: vc, value: value, text: text, name: name, schema: schema})
		}
	}

	// No case can match null.
	if d.nullable || !d.exhausts(matched) {
		var err error
		if v.dflt, err = res.branch(s, cmp.Or(vr.DefaultSchema, &plan.ObjectSchema{}), "the default schema", where); err != nil {
			return nil, err
		}
		// Default holds a value no case matches, which only its field can set.
		i := slices.IndexFunc(v.dflt.fields, func(f ruleField) bool { return f.wireKey == v.key })
		v.dflt.fields[i].ps.Required = true
	}
	if v.optional {
		v.absent = &objectSchema{open: base.open, declared: base.declared, fields: withoutField(base.fields, v.key)}
	}
	return v, nil
}

// branch merges b, a case or the default schema, into the base schema s. A
// property in both is required if either requires it, and must otherwise
// agree. The branch is open only when both are, since an extra key must
// satisfy both.
func (res *resolver) branch(s, b *plan.ObjectSchema, label, where string) (*objectSchema, error) {
	if b.AdditionalProperties && !s.AdditionalProperties {
		// The plan does not record whether the base's closedness reaches the branch.
		core.Warn(fmt.Sprintf("generating %s of the variants of %s closed: its base schema does not allow additional properties", label, where))
	}
	merged := plan.ObjectSchema{Properties: maps.Clone(s.Properties), AdditionalProperties: s.AdditionalProperties && b.AdditionalProperties}
	for _, key := range slices.Sorted(maps.Keys(b.Properties)) {
		ps := b.Properties[key]
		if bp, ok := merged.Properties[key]; ok {
			same, err := sameDefinition(bp, ps)
			if err != nil {
				return nil, fmt.Errorf("%s: mapping property %q: %w", label, key, err)
			}
			if !same {
				return nil, fmt.Errorf("%s: property %q conflicts with its definition in the base schema", label, key)
			}
			bp.Required = bp.Required || ps.Required
			ps = bp
		}
		merged.Properties[key] = ps
	}
	return res.schema(&merged, where, "")
}

// sameDefinition reports whether a, the base's property, and b, the branch's,
// agree in everything but required: types, item types, enum values and nested
// schema, whose properties are compared the same way. A branch that declares
// no nested schema keeps the base's: the local provider gives a case property
// an empty open one, since only rules carry nesting. An invalid enum is an
// error, because the merge keeps only one definition and would drop it.
func sameDefinition(a, b plan.PropertySchema) (bool, error) {
	ea, err := resolveEnum(a.Property)
	if err != nil {
		return false, err
	}
	eb, err := resolveEnum(b.Property)
	if err != nil {
		return false, err
	}
	switch {
	case typeSignature(a.Property, ea) != typeSignature(b.Property, eb):
		return false, nil
	case !isStruct(b.Schema):
		return true, nil
	case !isStruct(a.Schema) || a.Schema.AdditionalProperties != b.Schema.AdditionalProperties || len(a.Schema.Properties) != len(b.Schema.Properties):
		return false, nil
	}
	for _, key := range slices.Sorted(maps.Keys(a.Schema.Properties)) {
		pb, ok := b.Schema.Properties[key]
		if !ok || pb.Required != a.Schema.Properties[key].Required {
			return false, nil
		}
		if same, err := sameDefinition(a.Schema.Properties[key], pb); err != nil || !same {
			return false, err
		}
	}
	return true, nil
}

// validateVariantNames checks the names in the case and default schemas of
// variants, which belong to where, as validateSchemaNames does in a base
// schema.
func validateVariantNames(variants []plan.Variant, where string) error {
	for _, vr := range variants {
		for _, vc := range vr.Cases {
			if err := validateSchemaNames(vc.Schema, fmt.Sprintf("case %s of the variants of %s", jsonText(vc.Match), where)); err != nil {
				return err
			}
		}
		if vr.DefaultSchema == nil {
			continue
		}
		if err := validateSchemaNames(*vr.DefaultSchema, "the default schema of the variants of "+where); err != nil {
			return err
		}
	}
	return nil
}

func withoutField(fields []ruleField, key string) []ruleField {
	return slices.DeleteFunc(slices.Clone(fields), func(f ruleField) bool { return f.wireKey == key })
}

// matchValue normalizes a match value of the case label as enum values are,
// so 1 and 1.0 match alike, and returns it with its Go literal and its name,
// which Enum Constant Names would give it as an enum value.
func matchValue(m any, label string) (value any, text, name string, err error) {
	n, _ := normalizeEnumValue(m) // nil, which no case matches, when m is no enum value either
	switch n := n.(type) {
	case string:
		return n, strconv.Quote(n), valueName(n), nil
	case bool:
		text = strconv.FormatBool(n)
		return n, text, valueName(text), nil
	case *big.Rat:
		if !n.IsInt() {
			break
		}
		if !n.Num().IsInt64() {
			return nil, "", "", fmt.Errorf("%s: match value %s does not fit in int64", label, jsonText(m))
		}
		text = n.Num().String()
		return n, text, strings.Replace(text, "-", "Neg", 1), nil
	}
	return nil, "", "", fmt.Errorf("%s: match value %s is not a string, integer or boolean", label, jsonText(m))
}

// isVariant reports whether a property's type is a custom type with variants,
// which Go generates as an interface.
func isVariant(p plan.Property) bool {
	if len(p.Types) != 1 {
		return false
	}
	ct := plan.AsCustomType(p.Types[0])
	return ct != nil && len(ct.Variants) > 0
}

// addVariant adds the interface name of v, then the structs implementing it,
// each followed by the types it hoists: the structs the base schema hoists,
// Absent, the cases in match order, and Default. lead starts the interface's
// doc comment; a payload interface also lists ToProperties.
func addVariant(ctx *GoContext, registry *core.NameRegistry, propertyTypes map[propertyKey]propertyType, name, lead, description string, v *variantSchema, payload bool) (*GoVariant, error) {
	iface := &GoVariant{Name: name, Key: v.key, Payload: payload}
	ctx.add(GoDecl{Variant: iface})

	// Every branch holds the base's fields, so all share what the base hoists.
	hoisted := map[string]string{}
	for _, f := range v.base.fields {
		if f.nested == nil {
			continue
		}
		field, err := fieldName(f.wireKey)
		if err != nil {
			return nil, fmt.Errorf("naming property %q of %s: %w", f.wireKey, name, err)
		}
		first := len(*ctx.out)
		if hoisted[f.wireKey], err = addNested(ctx, registry, propertyTypes, name, field, f); err != nil {
			return nil, err
		}
		// addNested documents the type as the one in name.Field, a field the
		// interface does not have.
		in := strings.NewReplacer(" in "+name+"."+field, " in the "+field+" field of each struct implementing "+name)
		for _, d := range (*ctx.out)[first:] {
			if d.Union != nil {
				d.Union.Doc = in.Replace(d.Union.Doc)
			}
			if d.Struct != nil {
				d.Struct.Doc = in.Replace(d.Struct.Doc)
			}
		}
	}
	var (
		i   = slices.IndexFunc(v.base.fields, func(f ruleField) bool { return f.wireKey == v.key })
		typ = cmp.Or(hoisted[v.key], propertyTypes[keyOf(v.base.fields[i])].name)
		// okSuffix follows the value a case or Default returns, and qualifier
		// the discriminator's key in the interface's doc.
		okSuffix  string
		qualifier string
	)
	iface.Result = typ
	iface.DiscriminatorDoc = fmt.Sprintf("Discriminator returns the value of %s.", strconv.Quote(v.key))
	if v.optional {
		okSuffix, qualifier = ", true", "the optional "
		iface.Result = "(value " + typ + ", ok bool)"
		iface.DiscriminatorDoc = fmt.Sprintf("Discriminator returns the value of %s; ok is false when it is absent.", strconv.Quote(v.key))
	}

	addImplementation := func(id, suffix string, s *objectSchema) (*GoStruct, error) {
		structName, err := registry.RegisterName(id, packageScope, name+suffix)
		if err != nil {
			return nil, fmt.Errorf("registering %s%s: %w", name, suffix, err)
		}
		// A property named discriminator would clash with the method.
		if _, err := registry.RegisterName("reserved:Discriminator", "struct:"+structName+":fields", "Discriminator"); err != nil {
			return nil, fmt.Errorf("reserving Discriminator in %s: %w", structName, err)
		}
		for i := range s.fields {
			s.fields[i].hoisted = hoisted[s.fields[i].wireKey]
		}
		st, err := addStruct(ctx, registry, propertyTypes, structName, "", s, payload)
		if err != nil {
			return nil, err
		}
		st.Variant = iface
		iface.Implementations = append(iface.Implementations, structName)
		return st, nil
	}

	if v.absent != nil {
		st, err := addImplementation("absent:"+name, "Absent", v.absent)
		if err != nil {
			return nil, err
		}
		st.Doc = fmt.Sprintf("%s is the case where %s is absent. It holds the base properties only.", st.Name, strconv.Quote(v.key))
		st.DiscriminatorDoc = fmt.Sprintf("Discriminator reports that %s is absent.", strconv.Quote(v.key))
		st.Returns = "value, false"
	}

	for _, c := range v.cases {
		st, err := addImplementation("case:"+name+":"+c.text, "Case"+c.name, c.schema)
		if err != nil {
			return nil, err
		}
		doc := fmt.Sprintf("%s is the case where %s is %s.", st.Name, v.key, c.text)
		if c.vc.DisplayName != "" {
			doc = fmt.Sprintf("%s is the %s case of %s, where %s is %s.", st.Name, strconv.Quote(c.vc.DisplayName), name, v.key, c.text)
		}
		st.Doc = withDescription(doc, c.vc.Description)
		st.DiscriminatorDoc = "Discriminator returns " + c.text + "."
		ctx.afterValueNames = append(ctx.afterValueNames, func(decls map[string]GoDecl) {
			value, inner := spell(decls, typ, c.value, c.text)
			switch {
			case inner != "":
				// T must be spelled out: NewNullable(1) would be a Nullable[int].
				value = "NewNullable[" + inner + "](" + value + ")"
			case v.domain.enum != nil:
				// The doc names the constant or value function, as Default's
				// note does; a Nullable's keeps the literal, which reads better
				// than the call.
				st.DiscriminatorDoc = "Discriminator returns " + value + "."
			}
			st.Returns, st.Sends = value+okSuffix, "v.Discriminator()"
			if v.optional {
				// Discriminator also returns ok, so toMap writes the value itself.
				st.Sends = value
			}
		})
	}

	if v.dflt != nil {
		st, err := addImplementation("default:"+name, "Default", v.dflt)
		if err != nil {
			return nil, err
		}
		field := &st.Fields[slices.IndexFunc(st.Fields, func(f GoField) bool { return f.Key == v.key })]
		st.DiscriminatorDoc = "Discriminator returns " + field.Name + "."
		st.Returns, st.ReadsReceiver = "v."+field.Name+okSuffix, true
		ctx.afterValueNames = append(ctx.afterValueNames, func(decls map[string]GoDecl) {
			st.Doc, field.Note = defaultDocs(decls, typ, v, name, st.Name)
		})
	}

	implementations := slices.Sorted(slices.Values(iface.Implementations))
	iface.Doc = withDescription(fmt.Sprintf("%s, a discriminated union on %s%s. It is implemented by %s.",
		lead, qualifier, strconv.Quote(v.key), joinList(implementations, "and")), description)
	return iface, nil
}

// defaultDocs returns the doc comment of the Default struct defaultName and
// the note on its discriminator field, naming the values it stands for: the
// enum values no case matches, or any value but the matches, and null when
// the discriminator is nullable.
func defaultDocs(decls map[string]GoDecl, typ string, v *variantSchema, name, defaultName string) (doc, note string) {
	matched := map[string]bool{}
	for _, c := range v.cases {
		matched[c.text] = true
	}
	if v.domain.enum == nil && !v.domain.exhausts(matched) {
		texts := slices.Sorted(maps.Keys(matched))
		doc = fmt.Sprintf("%s is the default case of %s, for any %s other than %s", defaultName, name, v.key, joinList(texts, "and"))
		note = "not " + joinList(texts, "or")
		if v.domain.nullable {
			doc, note = doc+", or null", note+", or null"
		}
		return doc + ".", note
	}

	// Without an enum, the cases match both booleans, so only null is left.
	var rest []string
	if v.domain.enum != nil {
		for _, e := range v.domain.enum.values {
			if !matched[e.Text] {
				value, _ := spell(decls, typ, nil, e.Text)
				rest = append(rest, value)
			}
		}
	}
	if v.domain.nullable {
		rest = append(rest, "null")
	}
	what := "the values no case covers"
	if len(rest) == 1 {
		what = "the only value no case covers"
	}
	return fmt.Sprintf("%s is the default case of %s: %s is %s, %s.", defaultName, name, v.key, joinList(rest, "or"), what),
		"only " + joinList(rest, "or")
}

// finishVariants runs what the variants left for after registerValueNames,
// with every alias, enum and union by name.
func finishVariants(ctx *GoContext) {
	decls := map[string]GoDecl{}
	for _, d := range slices.Concat(ctx.CustomTypes, ctx.PropertyTypes, ctx.Payloads) {
		switch {
		case d.Alias != nil:
			decls[d.Alias.Name] = d
		case d.Enum != nil:
			decls[d.Enum.Name] = d
		case d.Union != nil:
			decls[d.Union.Name] = d
		}
	}
	for _, f := range ctx.afterValueNames {
		f(decls)
	}
}

// spell writes a match value, whose Go literal is text, as a value of typ: the
// literal for a primitive, the constant or value function of an enum, or the
// member constructor of a union. For a Nullable[T] it spells the value for T
// and returns T as inner.
func spell(decls map[string]GoDecl, typ string, value any, text string) (spelled, inner string) {
	if t, ok := strings.CutPrefix(typ, "Nullable["); ok {
		t = strings.TrimSuffix(t, "]")
		spelled, _ = spell(decls, t, value, text)
		return spelled, t
	}
	d := decls[typ]
	switch {
	case d.Alias != nil:
		return spell(decls, d.Alias.Type, value, text)
	case d.Enum != nil:
		v := d.Enum.Values[slices.IndexFunc(d.Enum.Values, func(v GoEnumValue) bool { return v.Text == text })]
		if d.Enum.Base == "" {
			return v.Name + "()", ""
		}
		return v.Name, ""
	case d.Union != nil:
		// An integral match takes the Integer member when there is one.
		params := []string{"int64", "float64"}
		switch value.(type) {
		case string:
			params = []string{"string"}
		case bool:
			params = []string{"bool"}
		}
		for _, p := range params {
			if i := slices.IndexFunc(d.Union.Members, func(m GoUnionMember) bool { return m.Type == p }); i >= 0 {
				return d.Union.Members[i].Name + "(" + text + ")", ""
			}
		}
	}
	return text, ""
}

// joinList joins items for a sentence: "a", "a and b", "a, b and c".
func joinList(items []string, conjunction string) string {
	if len(items) < 2 {
		return strings.Join(items, "")
	}
	last := len(items) - 1
	return strings.Join(items[:last], ", ") + " " + conjunction + " " + items[last]
}
