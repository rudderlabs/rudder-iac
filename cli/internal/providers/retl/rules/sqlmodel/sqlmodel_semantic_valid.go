package sqlmodel

import (
	"fmt"

	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/accounts"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/sqlmodel"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

var validateSQLModelSemantic = func(
	_ string,
	_ string,
	_ map[string]any,
	spec sqlmodel.SQLModelSpec,
	graph *resources.Graph,
) []rules.ValidationResult {
	results := validateDisplayNameUniqueness(spec, graph)
	return append(results, ValidateAccountReference(spec.Account, string(spec.SourceDefinition), graph)...)
}

// ValidateAccountReference checks a RETL source's account reference against the
// project: the account must exist, and its type must be the source's
// source_definition. The control plane rejects a mismatch on create ("Source
// definition name and account role do not match"), so this reports it at
// validate time rather than midway through an apply. A raw account_id is opaque
// locally and is not checked. Both RETL source kinds use it.
func ValidateAccountReference(ref, sourceDefinition string, graph *resources.Graph) []rules.ValidationResult {
	if ref == "" {
		return nil
	}
	// A malformed reference is the syntax rule's, or the loader's, to report.
	id, err := sqlmodel.ParseAccountRef(ref)
	if err != nil {
		return nil
	}

	reference := "/" + sqlmodel.AccountKey
	account, ok := graph.GetResource(resources.URN(id, accounts.AccountResourceType))
	if !ok {
		return []rules.ValidationResult{{
			Reference: reference,
			Message:   fmt.Sprintf("account '%s' not found in the project; reference an account spec in the project, or set account_id instead", id),
		}}
	}

	// An account spec with an unregistered definition fails to load, so an
	// account in the graph always has a known type; the guards only keep a
	// corrupt graph from panicking.
	data, ok := account.RawData().(*accounts.AccountResource)
	if !ok {
		return nil
	}
	accountType, ok := accounts.DefinitionType(data.AccountDefinitionName)
	if !ok || accountType == sourceDefinition {
		return nil
	}
	return []rules.ValidationResult{{
		Reference: reference,
		Message: fmt.Sprintf(
			"account '%s' is a '%s' account (%s) and cannot back source_definition '%s'",
			id, accountType, data.AccountDefinitionName, sourceDefinition,
		),
	}}
}

// validateDisplayNameUniqueness checks the model's display_name against the
// other SQL models. A clash with a table source is retl/table/semantic-valid's
// to report, on the table source, so it is reported once.
func validateDisplayNameUniqueness(spec sqlmodel.SQLModelSpec, graph *resources.Graph) []rules.ValidationResult {
	clash, ok := prules.NameClash(sqlmodel.DisplayNameKey, spec.DisplayName, DisplayNames(graph, sqlmodel.ResourceType))
	if !ok {
		return nil
	}
	return []rules.ValidationResult{{
		Reference: "/" + sqlmodel.DisplayNameKey,
		Message:   fmt.Sprintf("%s within kind '%s'", clash, sqlmodel.ResourceKind),
	}}
}

// DisplayNames returns the display_name of every RETL source of the given
// resource types in the graph.
func DisplayNames(graph *resources.Graph, resourceTypes ...string) []string {
	var names []string
	for _, resourceType := range resourceTypes {
		for _, resource := range graph.ResourcesByType(resourceType) {
			name, _ := resource.Data()[sqlmodel.DisplayNameKey].(string)
			names = append(names, name)
		}
	}
	return names
}

func NewSQLModelSemanticValidRule() rules.Rule {
	return prules.NewTypedRule(
		"retl/sqlmodel/semantic-valid",
		rules.Error,
		"retl sql model semantic constraints must be satisfied",
		rules.Examples{},
		prules.NewSemanticPatternValidator(
			prules.LegacyVersionPatterns(sqlmodel.ResourceKind),
			validateSQLModelSemantic,
		),
		prules.NewSemanticPatternValidator(
			prules.V1VersionPatterns(sqlmodel.ResourceKind),
			validateSQLModelSemantic,
		),
	)
}
