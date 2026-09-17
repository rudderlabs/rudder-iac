package rules

import (
	"fmt"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

// NameClashMessage describes name's clash with the other names in scope and
// returns "" when there is none. names holds every name in scope, name's own
// included, so a name clashes when it appears twice exactly, or once alongside
// a different spelling of itself.
//
// Names are folded with strings.ToLower, not strings.EqualFold, to mirror the
// server check this anticipates: config-backend's assertSourceNameAvailable
// compares with Postgres `LOWER(name) = LOWER(:name)`, whose per-character
// mapping is the one ToLower implements. The two disagree on characters with a
// multi-character lower-casing — ToLower folds "İstanbul" to "istanbul" and
// EqualFold does not — and folding more narrowly than the server would accept a
// pair at validate that apply then rejects, which is the split this rule
// exists to close.
//
// Names are not trimmed, because stored names were never normalized
// server-side and the server check does not trim either.
//
// The message reads "duplicate <field> '<name>'", followed by
// "(case-insensitive match with '<other>')" when no other name matches
// exactly. Callers say why the clash matters for their resource.
func NameClashMessage(field, name string, names []string) string {
	var (
		key     = strings.ToLower(name)
		exact   int
		variant string
	)
	for _, n := range names {
		if strings.ToLower(n) != key {
			continue
		}
		if n == name {
			exact++
			continue
		}
		// Smallest wins, so the message is stable across the map-ordered graph
		// walk callers build names from. One variant is enough to point at the
		// clash; naming them all does not tell the user more.
		if variant == "" || n < variant {
			variant = n
		}
	}

	if exact > 1 {
		return fmt.Sprintf("duplicate %s '%s'", field, name)
	}
	if variant == "" {
		return ""
	}
	return fmt.Sprintf("duplicate %s '%s' (case-insensitive match with '%s')", field, name, variant)
}

// NamesByKey returns the value of key in the data of every graph resource of
// the given resource types. Both RETL source kinds deliberately share
// sqlmodel.DisplayNameKey, so one call spans them.
func NamesByKey(graph *resources.Graph, key string, resourceTypes ...string) []string {
	var names []string
	for _, resourceType := range resourceTypes {
		for _, resource := range graph.ResourcesByType(resourceType) {
			name, _ := resource.Data()[key].(string)
			names = append(names, name)
		}
	}
	return names
}
