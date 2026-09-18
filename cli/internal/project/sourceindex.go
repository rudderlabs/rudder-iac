package project

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
)

// SourceLocation is the place in the project sources where a resource's URN is
// declared.
type SourceLocation struct {
	File   string
	Line   int
	Column int
}

// SourceLocator exposes the URN to source position map a project builds while
// loading.
//
// It is deliberately separate from [Project]: only read-only consumers that
// render the project back to a human (the editor graph command) need positions,
// so the main interface stays unchanged and every other implementation is
// unaffected. Callers type-assert for it.
type SourceLocator interface {
	SourceLocations() map[string]SourceLocation
}

// SourceLocations returns where each loaded URN is declared. It is empty until
// Load runs, and partial when some specs failed to parse.
func (p *project) SourceLocations() map[string]SourceLocation {
	return p.sourceIndex
}

// buildSourceIndex maps every URN the provider can parse out of the loaded
// specs to the position where it is declared.
//
// Positions come from NearestPosition rather than PositionLookup because a URN
// whose exact JSON Pointer is absent from the index — a generated id, or one
// nested below what the walker records — still lands the reader on its closest
// enclosing node. An approximate position beats none.
func buildSourceIndex(p provider.SpecLoader, parsed map[string]*specs.RawSpec) map[string]SourceLocation {
	index := make(map[string]SourceLocation)

	for path, rawSpec := range parsed {
		spec := rawSpec.Parsed()
		if spec == nil {
			continue
		}

		// A spec the provider cannot parse already surfaces as a diagnostic;
		// here it simply contributes no positions.
		parsedSpec, err := p.ParseSpec(path, spec)
		if err != nil || parsedSpec == nil {
			continue
		}

		indexer, err := rawSpec.PathIndexer()
		if err != nil {
			continue
		}

		for _, entry := range parsedSpec.URNs {
			pos := indexer.NearestPosition(entry.JSONPointerPath)
			index[entry.URN] = SourceLocation{
				File:   path,
				Line:   pos.Line,
				Column: pos.Column,
			}
		}
	}

	return index
}
