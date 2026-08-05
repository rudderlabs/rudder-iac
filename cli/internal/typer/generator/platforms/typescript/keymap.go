package typescript

import (
	"sort"
	"strings"
)

// keyMapSuffix is appended to an interface name to form its generated key-map
// constant, e.g. "UserSignedUp" → "UserSignedUpKeyMap".
const keyMapSuffix = "KeyMap"

// collectKeyMaps walks every generated interface and emits a per-interface
// camelCase→serial-name key map (TSKeyMap) for those that actually need one.
//
// An interface needs a map when at least one of its properties either:
//   - renames its wire key (camelCase Name differs from SerialName), or
//   - references another interface that itself needs a map (so the nested
//     object's keys get remapped recursively — mirroring Swift's nested
//     toProperties()).
//
// Maps are attached to ctx.KeyMaps (sorted by name for deterministic output)
// and the set of interface names that have a map is returned so callers can
// decide whether a method's props need remapping. This is the single source of
// truth for "does interface X need remapping".
func collectKeyMaps(ctx *TSContext) map[string]bool {
	interfaces := allInterfaces(ctx)

	// byName lets property-type resolution find a referenced interface. Names of
	// interfaces that need a map is computed to a fixed point below, since a
	// parent needs a map if a child does.
	byName := make(map[string]*TSInterface, len(interfaces))
	for i := range interfaces {
		byName[interfaces[i].Name] = interfaces[i]
	}

	needsMap := make(map[string]bool)
	// Fixed-point iteration: repeat until no interface flips to "needs map".
	// Bounded by the number of interfaces (each pass can only add flags).
	for {
		changed := false
		for name, iface := range byName {
			if needsMap[name] {
				continue
			}
			if interfaceNeedsMap(iface, needsMap) {
				needsMap[name] = true
				changed = true
			}
		}
		if !changed {
			break
		}
	}

	maps := make([]TSKeyMap, 0, len(needsMap))
	for _, iface := range interfaces {
		if !needsMap[iface.Name] {
			continue
		}
		maps = append(maps, buildKeyMap(iface, needsMap))
	}

	ctx.KeyMaps = orderKeyMaps(maps)
	return needsMap
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

// wireTrackKeyMaps points each track method at the key map for its typed props
// interface, rewriting the props SDK argument to route through applyKeyMap so
// the wire payload uses original plan keys. Methods whose interface needs no
// remap (or whose props are an open Record, not a typed interface) are left
// forwarding props verbatim.
func wireTrackKeyMaps(ctx *TSContext, needsMap map[string]bool) {
	for mi := range ctx.AnalyticsMethods {
		m := &ctx.AnalyticsMethods[mi]
		if m.SDKMethodName != "track" || len(m.MethodArguments) == 0 {
			continue
		}
		ifaceName := m.MethodArguments[0].Type
		if !needsMap[ifaceName] {
			continue
		}
		mapName := ifaceName + keyMapSuffix
		m.PropsKeyMapName = mapName
		ctx.UsesApplyKeyMap = true
		// The props SDK argument is the one referencing `props` — rewrite it to
		// remap keys first. Other args (event name, options, callback) are
		// untouched.
		for ai := range m.SDKArguments {
			if strings.HasPrefix(m.SDKArguments[ai].Value, "props ") {
				m.SDKArguments[ai].Value = "applyKeyMap(props, " + mapName + ") as unknown as " + sdkApiObjectAlias
			}
		}
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
func interfaceNeedsMap(iface *TSInterface, needsMap map[string]bool) bool {
	for _, p := range iface.Properties {
		if p.Name != p.SerialName {
			return true
		}
		if referencedInterfaceName(p.Type, needsMap) != "" {
			return true
		}
	}
	return false
}

// buildKeyMap constructs the TSKeyMap for an interface known to need one. Only
// properties that rename or reference a nested map produce entries; identity
// keys are omitted so applyKeyMap leaves them untouched.
func buildKeyMap(iface *TSInterface, needsMap map[string]bool) TSKeyMap {
	km := TSKeyMap{Name: iface.Name + keyMapSuffix}
	for _, p := range iface.Properties {
		nested := referencedInterfaceName(p.Type, needsMap)
		if p.Name == p.SerialName && nested == "" {
			continue
		}
		entry := TSKeyMapEntry{FieldName: p.Name, SerialName: p.SerialName}
		if nested != "" {
			entry.NestedMapName = nested + keyMapSuffix
		}
		km.Entries = append(km.Entries, entry)
	}
	return km
}

// referencedInterfaceName returns the name of a single interface referenced by
// a TS type expression when that interface needs a map, else "". It strips
// array/optional decorations (`Foo[]`, `Array<Foo>`, `Foo | null`) so a
// property typed as `CustomTypeUserProfile[]` still resolves to
// `CustomTypeUserProfile`.
//
// Union types with more than one interface member (e.g. discriminated-union
// aliases) are intentionally NOT resolved here — remapping a union requires
// per-branch dispatch on the discriminator, which this draft does not cover.
// See the nested-object limitation in the PR body.
func referencedInterfaceName(tsType string, needsMap map[string]bool) string {
	t := strings.TrimSpace(tsType)
	t = strings.TrimSuffix(t, "[]")
	if strings.HasPrefix(t, "Array<") && strings.HasSuffix(t, ">") {
		t = strings.TrimSuffix(strings.TrimPrefix(t, "Array<"), ">")
	}
	t = strings.TrimSpace(t)
	// Bail on unions/generics — only a bare interface reference is safe to remap.
	if strings.ContainsAny(t, "|<>") {
		return ""
	}
	if needsMap[t] {
		return t
	}
	return ""
}
