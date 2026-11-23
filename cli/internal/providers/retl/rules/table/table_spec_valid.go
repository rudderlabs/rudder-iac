package table

import (
	"fmt"
	"reflect"

	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/sqlmodel"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/table"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

var validateTableSpec = func(
	_ string,
	_ string,
	_ map[string]any,
	spec table.TableSpec,
) []rules.ValidationResult {
	validationErrors, err := rules.ValidateStruct(spec, "")
	if err != nil {
		return []rules.ValidationResult{{
			Message: err.Error(),
		}}
	}

	results := funcs.ParseValidationErrors(validationErrors, reflect.TypeOf(spec))
	return append(results, maxComputeNameResults(spec)...)
}

// maxComputeNameResults refuses a MaxCompute schema or table name that the
// warehouse could never resolve, so validate catches it instead of the first
// preview or sync. An empty name is left to the required rule above.
func maxComputeNameResults(spec table.TableSpec) []rules.ValidationResult {
	if spec.SourceDefinition != string(sqlmodel.SourceDefinitionMaxCompute) {
		return nil
	}
	var results []rules.ValidationResult
	for _, field := range []struct{ key, value string }{{"schema", spec.Schema}, {"table", spec.Table}} {
		if field.value == "" {
			continue
		}
		if reason := table.MaxComputeNameError(field.value); reason != "" {
			results = append(results, rules.ValidationResult{
				Reference: "/" + field.key,
				Message:   fmt.Sprintf("'%s' %s for source_definition maxcompute", field.key, reason),
			})
		}
	}
	return results
}

// NewTableSpecSyntaxValidRule validates retl-source-table specs. The kind is
// v1-only, so it matches no legacy spec versions.
func NewTableSpecSyntaxValidRule() rules.Rule {
	return prules.NewTypedRule(
		"retl/table/spec-syntax-valid",
		rules.Error,
		"retl table source spec syntax must be valid (experimental kind)",
		rules.Examples{},
		prules.NewPatternValidator(
			prules.V1VersionPatterns(table.ResourceKind),
			validateTableSpec,
		),
	)
}
