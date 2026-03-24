package rules

import (
	"fmt"
	"slices"

	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

type resourceKindVersionValidRule struct {
	supportedKinds    []string
	supportedVersions []string
	supportedPatterns []rules.MatchPattern
}

func NewResourceKindVersionValidRule(supportedKinds []string, supportedVersions []string, supportedPatterns []rules.MatchPattern) rules.Rule {
	return &resourceKindVersionValidRule{
		supportedKinds:    supportedKinds,
		supportedVersions: supportedVersions,
		supportedPatterns: supportedPatterns,
	}
}

func (r *resourceKindVersionValidRule) ID() string               { return "project/resource-kind-version-valid" }
func (r *resourceKindVersionValidRule) Severity() rules.Severity { return rules.Error }
func (r *resourceKindVersionValidRule) Description() string {
	return "resource kind must be supported with the specified version"
}
func (r *resourceKindVersionValidRule) AppliesTo() []rules.MatchPattern {
	return []rules.MatchPattern{rules.MatchAll()}
}
func (r *resourceKindVersionValidRule) Examples() rules.Examples { return rules.Examples{} }

func (r *resourceKindVersionValidRule) Validate(ctx *rules.ValidationContext) []rules.ValidationResult {
	if !slices.Contains(r.supportedKinds, ctx.Kind) {
		return nil
	}
	if !slices.Contains(r.supportedVersions, ctx.Version) {
		return nil
	}

	for _, p := range r.supportedPatterns {
		if p.Kind == ctx.Kind && p.Version == ctx.Version {
			return nil
		}
	}

	return []rules.ValidationResult{{
		Reference: "/kind",
		Message:   fmt.Sprintf("kind '%s' is not supported with version '%s'", ctx.Kind, ctx.Version),
	}}
}
