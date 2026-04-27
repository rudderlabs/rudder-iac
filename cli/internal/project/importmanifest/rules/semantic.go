package rules

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// Semantic returns the semantic rules contributed by the import-manifest
// provider. They run after resource graph construction.
func Semantic() []rules.Rule {
	return []rules.Rule{
		newURNResolvesRule(),
	}
}

// urnResolvesRule ensures every URN listed in an import-manifest matches a
// node in the resource graph. Orphaned URNs — from typos or stale entries —
// would otherwise silently fail to mark anything Importable.
type urnResolvesRule struct{}

func newURNResolvesRule() rules.Rule { return &urnResolvesRule{} }

func (r *urnResolvesRule) ID() string               { return "import-manifest/urn-resolves" }
func (r *urnResolvesRule) Severity() rules.Severity { return rules.Error }
func (r *urnResolvesRule) Description() string {
	return "every URN in an import-manifest must reference a resource present in the project"
}
func (r *urnResolvesRule) AppliesTo() []rules.MatchPattern {
	return []rules.MatchPattern{
		rules.MatchKindVersion(specs.KindImportManifest, specs.SpecVersionV1),
	}
}
func (r *urnResolvesRule) Examples() rules.Examples { return rules.Examples{} }

func (r *urnResolvesRule) Validate(ctx *rules.ValidationContext) []rules.ValidationResult {
	if ctx.Graph == nil {
		return nil
	}

	var results []rules.ValidationResult
	for _, entry := range collectManifestURNs(ctx.Spec) {
		if _, ok := ctx.Graph.GetResource(entry.value); ok {
			continue
		}
		results = append(results, rules.ValidationResult{
			Reference: fmt.Sprintf(
				"/spec/workspaces/%d/resources/%d/urn",
				entry.workspaceIdx, entry.resourceIdx,
			),
			Message: fmt.Sprintf("URN %q not found in the resource graph", entry.value),
		})
	}
	return results
}
