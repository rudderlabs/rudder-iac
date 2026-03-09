package source

import (
	"regexp"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	dcRules "github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/rules"
	esSource "github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream/source"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

var (
	legacyTrackingPlanRefRegex = regexp.MustCompile(dcRules.TrackingPlanLegacyReferenceRegex)
	trackingPlanRefRegex       = regexp.MustCompile(dcRules.TrackingPlanReferenceRegex)
)

var validateSourceSpec = func(
	_ string,
	version string,
	_ map[string]any,
	spec esSource.SourceSpec,
) []rules.ValidationResult {
	validationErrors, err := rules.ValidateStruct(spec, "")
	if err != nil {
		return []rules.ValidationResult{{
			Message: err.Error(),
		}}
	}

	results := funcs.ParseValidationErrors(validationErrors, nil)

	if spec.Governance != nil && spec.Governance.TrackingPlan != nil && spec.Governance.TrackingPlan.Ref != "" {
		ref := spec.Governance.TrackingPlan.Ref

		if version == specs.SpecVersionV1 {
			if !trackingPlanRefRegex.MatchString(ref) {
				results = append(results, rules.ValidationResult{
					Reference: "/governance/validations/tracking_plan",
					Message:   "'tracking_plan' is invalid: must be of pattern #tracking-plan:<id>",
				})
			}
		} else {
			if !legacyTrackingPlanRefRegex.MatchString(ref) {
				results = append(results, rules.ValidationResult{
					Reference: "/governance/validations/tracking_plan",
					Message:   "'tracking_plan' is invalid: must be of pattern #/tp/<group>/<id>",
				})
			}
		}
	}

	return results
}

func NewSourceSpecSyntaxValidRule() rules.Rule {
	return prules.NewTypedRule(
		"event-stream/source/spec-syntax-valid",
		rules.Error,
		"event stream source spec syntax must be valid",
		rules.Examples{},
		prules.NewPatternValidator(
			prules.LegacyVersionPatterns(esSource.ResourceKind),
			validateSourceSpec,
		),
		prules.NewPatternValidator(
			prules.V1VersionPatterns(esSource.ResourceKind),
			validateSourceSpec,
		),
	)
}
