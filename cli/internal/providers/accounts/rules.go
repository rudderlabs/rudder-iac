package accounts

import (
	"fmt"
	"reflect"
	"strings"

	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

const SpecSyntaxValidRuleID = "accounts/spec-syntax-valid"

// validateAccountSpec requires the envelope fields (id, name,
// account_definition_name, config) and, for a registered definition, every
// config key its account schema marks required. Both gaps otherwise pass
// validate and apply: a missing id plans an empty "account:" URN (DEX-995) and
// a missing config key only fails at the API with a raw JSON-schema error
// (DEX-994).
func validateAccountSpec(_, _ string, _ map[string]any, spec AccountSpec) []rules.ValidationResult {
	validationErrors, err := rules.ValidateStructWithTagPriority(spec, []string{"mapstructure"})
	if err != nil {
		return []rules.ValidationResult{{Message: err.Error()}}
	}
	if len(validationErrors) > 0 {
		return funcs.ParseMapstructureValidationErrors(validationErrors, reflect.TypeFor[AccountSpec]())
	}

	var results []rules.ValidationResult
	// An OAuth account's credentials are not the CLI's to hold and its
	// definition declares no option fields, so config here is always a mistake.
	// Saying so at validate time beats silently dropping it on apply.
	if isOAuthDefinition(spec.AccountDefinitionName) {
		if len(spec.Config) > 0 {
			results = append(results, rules.ValidationResult{
				Reference: "/config",
				Message: fmt.Sprintf(
					"'config' is not allowed for %s accounts: they carry no options, and their credentials are held by RudderStack's auth service",
					spec.AccountDefinitionName),
			})
		}
		return results
	}
	if key, allowed, unknown := unknownAuthMode(spec.AccountDefinitionName, spec.Config); unknown {
		results = append(results, rules.ValidationResult{
			Reference: "/config/" + key,
			Message:   fmt.Sprintf("'%s' must be one of %s for %s accounts", key, strings.Join(allowed, ", "), spec.AccountDefinitionName),
		})
	}
	for _, key := range missingRequiredConfig(spec.AccountDefinitionName, spec.Config) {
		results = append(results, rules.ValidationResult{
			Reference: "/config/" + key,
			Message:   fmt.Sprintf("'%s' is required for %s accounts", key, spec.AccountDefinitionName),
		})
	}
	return results
}

// NewSpecSyntaxValidRule validates account specs. The kind is v1-only.
func NewSpecSyntaxValidRule() rules.Rule {
	return prules.NewTypedRule(
		SpecSyntaxValidRuleID,
		rules.Error,
		"account spec must have an id, name, definition and the config its definition requires",
		rules.Examples{},
		prules.NewPatternValidator(prules.V1VersionPatterns(AccountSpecKind), validateAccountSpec),
	)
}
