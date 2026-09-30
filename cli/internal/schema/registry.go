// Package schema generates Draft 2020-12 JSON Schemas from the typed Go
// structures that providers use to decode rudder-cli specifications.
package schema

import (
	"errors"
	"sort"

	"github.com/invopop/jsonschema"
)

// ErrUnknownKind is returned when a requested kind is not in a schema set.
var ErrUnknownKind = errors.New("unknown spec kind")

// Set is a provider-owned collection of schemas keyed by spec kind.
type Set map[string]*jsonschema.Schema

// Kinds returns schema kinds in deterministic order.
func Kinds(schemas Set) []string {
	kinds := make([]string, 0, len(schemas))
	for kind := range schemas {
		kinds = append(kinds, kind)
	}
	sort.Strings(kinds)
	return kinds
}
