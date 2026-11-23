package typescript

import (
	"fmt"
	"sort"
	"strings"
)

// keyMapSuffix is appended to a type name to form its generated key-map
// constant, e.g. "UserSignedUp" → "UserSignedUpKeyMap".
const keyMapSuffix = "KeyMap"

// keyMapResolver answers "which key map applies to a value of type T?" for
// every TS type expression the generator can produce. It exists because a
// property's declared type is rarely the interface itself: it may be an array
// (`Foo[]`), a type alias standing for an array (`CustomTypeProfileList`), or a
// discriminated-union alias covering several case interfaces. Each of those
// still needs the nested object's keys rewritten, so resolution has to see
// through aliases and unions rather than matching bare interface names.
type keyMapResolver struct {
	ifaces   map[string]*TSInterface
	aliases  map[string]string   // alias name → underlying type expression
	variants map[string][]string // union alias name → case interface names
	caseOf   map[string]bool     // interface names that are variant cases
	needsMap map[string]bool     // interface name → has at least one remapped key
}

// collectKeyMaps walks every generated interface, works out which ones need a
// camelCase→serial-name map, and attaches the maps to ctx.KeyMaps.
//
// An interface needs a map when at least one of its properties either renames
// its wire key (camelCase Name differs from SerialName) or holds a value whose
// type resolves to something that itself needs remapping.
//
// Variant case interfaces never get a standalone map. A value typed as the
// union alias could be any branch, and the branches cannot disagree — the
// camelCase identifier is a pure function of the plan key — so the branch maps
// merge into one map named after the alias, applied without inspecting the
// discriminator at runtime. Emitting per-case maps on top of that would be
// dead code (`noUnusedLocals` in the validator tsconfig rejects it).
func collectKeyMaps(ctx *TSContext) (*keyMapResolver, error) {
	r := newKeyMapResolver(ctx)

	// Fixed-point iteration: a parent needs a map once a child does, so repeat
	// until nothing flips. Bounded by the interface count — each pass only adds.
	for {
		changed := false
		for name, iface := range r.ifaces {
			if r.needsMap[name] {
				continue
			}
			if r.interfaceNeedsMap(iface) {
				r.needsMap[name] = true
				changed = true
			}
		}
		if !changed {
			break
		}
	}

	var maps []TSKeyMap
	for _, iface := range allInterfaces(ctx) {
		if !r.needsMap[iface.Name] || r.caseOf[iface.Name] {
			continue
		}
		km, err := r.buildKeyMap(iface.Name, iface)
		if err != nil {
			return nil, err
		}
		maps = append(maps, km)
	}

	merged, err := r.buildVariantMaps(ctx)
	if err != nil {
		return nil, err
	}
	maps = append(maps, merged...)

	ctx.KeyMaps = orderKeyMaps(maps)
	return r, nil
}

func newKeyMapResolver(ctx *TSContext) *keyMapResolver {
	r := &keyMapResolver{
		ifaces:   make(map[string]*TSInterface),
		aliases:  make(map[string]string),
		variants: make(map[string][]string),
		caseOf:   make(map[string]bool),
		needsMap: make(map[string]bool),
	}
	for _, iface := range allInterfaces(ctx) {
		r.ifaces[iface.Name] = iface
	}
	for _, a := range ctx.CustomTypeAliases {
		r.aliases[a.Alias] = a.Type
	}
	for _, a := range ctx.PropertyEnums {
		r.aliases[a.Alias] = a.Type
	}
	for gi := range ctx.VariantTypes {
		g := &ctx.VariantTypes[gi]
		names := make([]string, 0, len(g.CaseInterfaces))
		for ci := range g.CaseInterfaces {
			names = append(names, g.CaseInterfaces[ci].Name)
			r.caseOf[g.CaseInterfaces[ci].Name] = true
		}
		r.variants[g.UnionAlias.Alias] = names
	}
	return r
}

// buildVariantMaps emits one merged map per discriminated union whose branches
// need remapping. See collectKeyMaps for why merging is safe.
func (r *keyMapResolver) buildVariantMaps(ctx *TSContext) ([]TSKeyMap, error) {
	var out []TSKeyMap
	for gi := range ctx.VariantTypes {
		g := &ctx.VariantTypes[gi]
		alias := g.UnionAlias.Alias

		merged := TSKeyMap{Name: alias + keyMapSuffix}
		byField := make(map[string]TSKeyMapEntry)
		var fields []string

		for ci := range g.CaseInterfaces {
			c := &g.CaseInterfaces[ci]
			if !r.needsMap[c.Name] {
				continue
			}
			km, err := r.buildKeyMap(c.Name, c)
			if err != nil {
				return nil, err
			}
			for _, e := range km.Entries {
				prev, seen := byField[e.FieldName]
				if !seen {
					byField[e.FieldName] = e
					fields = append(fields, e.FieldName)
					continue
				}
				if prev != e {
					return nil, fmt.Errorf(
						"variant %q: branch %q maps field %q to %q but a sibling branch maps it to %q; "+
							"the union cannot be remapped without runtime dispatch",
						alias, c.Name, e.FieldName, e.SerialName, prev.SerialName)
				}
			}
		}
		if len(fields) == 0 {
			continue
		}
		sort.Strings(fields)
		for _, f := range fields {
			merged.Entries = append(merged.Entries, byField[f])
		}
		out = append(out, merged)
	}
	return out, nil
}

// orderKeyMaps returns the maps ordered so that every map appears after the
// maps it references. TS `const` initializers run at module load, so a nested
// reference to a map defined later would hit the temporal dead zone — emitting
// dependencies first avoids that. Within a dependency tier the order is
// alphabetical for deterministic output.
func orderKeyMaps(maps []TSKeyMap) []TSKeyMap {
	byName := make(map[string]TSKeyMap, len(maps))
	names := make([]string, 0, len(maps))
	for _, m := range maps {
		byName[m.Name] = m
		names = append(names, m.Name)
	}
	sort.Strings(names)

	var (
		ordered  []TSKeyMap
		emitted  = make(map[string]bool)
		visiting = make(map[string]bool)
		visit    func(name string)
	)
	visit = func(name string) {
		if emitted[name] || visiting[name] {
			return // visiting guard also breaks any (unexpected) cycle
		}
		m, ok := byName[name]
		if !ok {
			return
		}
		visiting[name] = true
		for _, e := range m.Entries {
			if e.NestedMapName != "" {
				visit(e.NestedMapName)
			}
		}
		visiting[name] = false
		emitted[name] = true
		ordered = append(ordered, m)
	}
	for _, name := range names {
		visit(name)
	}
	return ordered
}

// wireKeyMaps points every method carrying plan-typed props/traits at the key
// map for that type and rewrites the argument to route through applyKeyMap, so
// the wire payload uses original plan keys. This covers track, identify, group
// and page uniformly — each marks its props argument with PropsExpr, so no
// call site is matched by string inspection.
func wireKeyMaps(ctx *TSContext, r *keyMapResolver) error {
	for mi := range ctx.AnalyticsMethods {
		m := &ctx.AnalyticsMethods[mi]
		if m.PropsTypeName == "" {
			continue
		}
		mapName, err := r.mapNameFor(m.PropsTypeName)
		if err != nil {
			return fmt.Errorf("method %s: %w", m.Name, err)
		}
		if mapName == "" {
			continue
		}
		m.PropsKeyMapName = mapName
		ctx.UsesApplyKeyMap = true

		remapPropsArgs(m.SDKArguments, mapName)
		for bi := range m.DispatcherBranches {
			remapPropsArgs(m.DispatcherBranches[bi].SDKArguments, mapName)
		}
	}
	return nil
}

// propsArg builds an SDK argument that carries the plan-typed props/traits.
// valueFmt is the full argument expression with a single %s where expr sits;
// the key-map wiring re-renders it with expr wrapped in applyKeyMap.
func propsArg(valueFmt, expr string) TSSDKArgument {
	return TSSDKArgument{
		Value:     fmt.Sprintf(valueFmt, expr),
		PropsExpr: expr,
		ValueFmt:  valueFmt,
	}
}

// remapPropsArgs re-renders every props-carrying argument with its expression
// wrapped in applyKeyMap. Arguments that carry no props are left alone.
func remapPropsArgs(args []TSSDKArgument, mapName string) {
	for i := range args {
		if args[i].PropsExpr == "" {
			continue
		}
		args[i].Value = fmt.Sprintf(args[i].ValueFmt, "applyKeyMap("+args[i].PropsExpr+", "+mapName+")")
	}
}

// allInterfaces returns pointers to every interface the generator produced,
// across event, nested, custom, and variant-case buckets.
func allInterfaces(ctx *TSContext) []*TSInterface {
	var out []*TSInterface
	for i := range ctx.CustomInterfaces {
		out = append(out, &ctx.CustomInterfaces[i])
	}
	for i := range ctx.NestedInterfaces {
		out = append(out, &ctx.NestedInterfaces[i])
	}
	for i := range ctx.Interfaces {
		out = append(out, &ctx.Interfaces[i])
	}
	for gi := range ctx.VariantTypes {
		for ci := range ctx.VariantTypes[gi].CaseInterfaces {
			out = append(out, &ctx.VariantTypes[gi].CaseInterfaces[ci])
		}
	}
	return out
}

// interfaceNeedsMap reports whether iface has any property that must be
// remapped, given the current known set of interfaces that need a map.
func (r *keyMapResolver) interfaceNeedsMap(iface *TSInterface) bool {
	for _, p := range iface.Properties {
		if p.Name != p.SerialName {
			return true
		}
		for _, name := range r.resolve(p.Type) {
			if r.needsMap[name] {
				return true
			}
		}
	}
	return false
}

// buildKeyMap constructs the TSKeyMap for a type known to need one. Only
// properties that rename or hold a remappable value produce entries; identity
// keys are omitted so applyKeyMap leaves them untouched.
func (r *keyMapResolver) buildKeyMap(name string, iface *TSInterface) (TSKeyMap, error) {
	km := TSKeyMap{Name: name + keyMapSuffix}
	for _, p := range iface.Properties {
		nested, err := r.mapNameFor(p.Type)
		if err != nil {
			return TSKeyMap{}, fmt.Errorf("interface %s, property %q: %w", iface.Name, p.SerialName, err)
		}
		if p.Name == p.SerialName && nested == "" {
			continue
		}
		km.Entries = append(km.Entries, TSKeyMapEntry{
			FieldName:     p.Name,
			SerialName:    p.SerialName,
			NestedMapName: nested,
		})
	}
	return km, nil
}

// mapNameFor returns the key-map constant to apply to a value of tsType, or ""
// when nothing in that type needs remapping.
//
// A union of several mapped interfaces is only resolvable through its alias,
// where the branch maps have been merged. An inline union of mapped interfaces
// has no such merged map, so it is an error rather than a silent pass-through —
// shipping the wrong keys is worse than failing to generate.
func (r *keyMapResolver) mapNameFor(tsType string) (string, error) {
	names := r.resolve(tsType)
	mapped := make([]string, 0, len(names))
	for _, n := range names {
		if r.needsMap[n] {
			mapped = append(mapped, n)
		}
	}
	if len(mapped) == 0 {
		return "", nil
	}

	if bare := bareTypeName(tsType); bare != "" {
		if _, ok := r.variants[bare]; ok {
			return bare + keyMapSuffix, nil
		}
	}
	if len(mapped) == 1 {
		return mapped[0] + keyMapSuffix, nil
	}
	return "", fmt.Errorf(
		"type %q is an inline union of remappable interfaces (%s); give it a named union alias so the branch maps can be merged",
		tsType, strings.Join(mapped, ", "))
}

// resolve returns the interface names a value of tsType can be, seeing through
// array decorations, nullable unions, type aliases, and union aliases.
func (r *keyMapResolver) resolve(tsType string) []string {
	return r.resolveSeen(tsType, make(map[string]bool))
}

func (r *keyMapResolver) resolveSeen(tsType string, seen map[string]bool) []string {
	t := strings.TrimSpace(tsType)

	if parts := splitTopLevelUnion(t); len(parts) > 1 {
		var out []string
		for _, p := range parts {
			out = append(out, r.resolveSeen(p, seen)...)
		}
		return dedupeStrings(out)
	}

	if inner, ok := unwrapArrayType(t); ok {
		return r.resolveSeen(inner, seen)
	}

	// Anything still carrying generic syntax (Record<>, Exclude<>, ...) holds no
	// interface we can name, so there is nothing to remap.
	if strings.ContainsAny(t, "<>|") {
		return nil
	}
	if t == "" || seen[t] {
		return nil
	}
	seen[t] = true

	if cases, ok := r.variants[t]; ok {
		return cases
	}
	if _, ok := r.ifaces[t]; ok {
		return []string{t}
	}
	if under, ok := r.aliases[t]; ok {
		return r.resolveSeen(under, seen)
	}
	return nil
}

// bareTypeName strips array and nullable decorations and returns the single
// named type underneath, or "" when the expression is not a bare name.
func bareTypeName(tsType string) string {
	t := strings.TrimSpace(tsType)

	if parts := splitTopLevelUnion(t); len(parts) > 1 {
		var named []string
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p == "null" || p == "undefined" {
				continue
			}
			named = append(named, p)
		}
		if len(named) != 1 {
			return ""
		}
		t = named[0]
	}

	if inner, ok := unwrapArrayType(t); ok {
		return bareTypeName(inner)
	}
	if t == "" || strings.ContainsAny(t, "<>|[]() \t") {
		return ""
	}
	return t
}

// unwrapArrayType peels one array decoration, handling both `Foo[]` and
// `Array<Foo>`.
func unwrapArrayType(t string) (string, bool) {
	t = strings.TrimSpace(t)
	if strings.HasSuffix(t, "[]") {
		return strings.TrimSuffix(t, "[]"), true
	}
	if strings.HasPrefix(t, "Array<") && strings.HasSuffix(t, ">") {
		return strings.TrimSuffix(strings.TrimPrefix(t, "Array<"), ">"), true
	}
	return "", false
}

// splitTopLevelUnion splits on `|` that sits outside any generic or paren
// nesting, so `Array<string | number>` stays intact.
func splitTopLevelUnion(t string) []string {
	var (
		parts []string
		depth int
		start int
	)
	for i, c := range t {
		switch c {
		case '<', '(', '[':
			depth++
		case '>', ')', ']':
			depth--
		case '|':
			if depth == 0 {
				parts = append(parts, t[start:i])
				start = i + 1
			}
		}
	}
	parts = append(parts, t[start:])
	return parts
}

func dedupeStrings(in []string) []string {
	seen := make(map[string]bool, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}
