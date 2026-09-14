package rules

import (
	"fmt"
	"slices"
	"strings"
)

// NameClash describes name's clash with the other names in scope, compared
// case-insensitively: the control plane rejects a source whose name differs
// from another's only in case, so a case-sensitive check would let the clash
// through to a failed apply. names holds every name in scope, name's own
// included. The description reads "duplicate <field> '<name>'", followed by
// "(case-insensitive match with '<other>')" when no other name matches
// exactly; ok is false when name has no clash.
func NameClash(field, name string, names []string) (string, bool) {
	var (
		key      = strings.ToLower(name)
		exact    int
		variants []string
	)
	for _, n := range names {
		if strings.ToLower(n) != key {
			continue
		}
		if n == name {
			exact++
			continue
		}
		variants = append(variants, n)
	}

	if exact > 1 {
		return fmt.Sprintf("duplicate %s '%s'", field, name), true
	}
	if len(variants) == 0 {
		return "", false
	}
	slices.Sort(variants)
	return fmt.Sprintf(
		"duplicate %s '%s' (case-insensitive match with '%s')",
		field, name, strings.Join(slices.Compact(variants), "', '"),
	), true
}
