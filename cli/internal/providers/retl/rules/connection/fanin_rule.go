package connection

import (
	"fmt"

	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	retlConnection "github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/connection"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// NewInheritedFanInRule reports inherited fan-in separately from the error
// rule: the diagnostic engine uses a rule's severity, not a result's severity.
func NewInheritedFanInRule() rules.Rule {
	return prules.NewTypedRule(
		"retl/connection/inherited-fan-in",
		rules.Warning,
		"existing rETL fan-in with a distinct imported connection ID per source in the active workspace can be retained but not expanded",
		rules.Examples{},
		prules.NewContextSemanticPatternValidator(
			prules.V1VersionPatterns(retlConnection.ResourceKind),
			func(spec retlConnection.ConnectionsSpec, ctx *rules.ValidationContext) []rules.ValidationResult {
				var results []rules.ValidationResult
				for index, c := range spec.Connections {
					endpoints := resolveEndpoints(ctx.Graph, c)
					if !endpoints.resolved() {
						continue
					}
					inherited := destinationFanIn(ctx.Graph, endpoints, ctx.WorkspaceID)
					if inherited {
						results = append(results, result(destinationRef(index), fmt.Sprintf("destination '%s' has multiple rETL sources, each importing a distinct connection in the active workspace; this inherited fan-in may be retained, but new connections or replacements that introduce fan-in are not allowed", endpoints.destinationID)))
					}
				}
				return results
			},
		),
	)
}
