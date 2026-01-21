package rules

import (
	"fmt"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/samber/lo"
)

type VersionValidRule struct {
	validVersions []string
}

func NewVersionValidRule() rules.Rule {
	return &VersionValidRule{
		validVersions: []string{
			specs.SpecVersionV1,
			specs.SpecVersionV0_1Variant,
			specs.SpecVersionV0_1,
		},
	}
}

func (r *VersionValidRule) ID() string {
	return "project/version-valid"
}

func (r *VersionValidRule) Validate(ctx *rules.ValidationContext) []rules.ValidationResult {

	if ctx.Version != "" && !lo.Contains(r.validVersions, ctx.Version) {
		return []rules.ValidationResult{
			{
				Reference: "/version",
				Message:   fmt.Sprintf("version must be one of the supported versions: %s", strings.Join(r.validVersions, ", ")),
			},
		}
	}

	return nil
}

func (r *VersionValidRule) Severity() rules.Severity {
	return rules.Error
}

func (r *VersionValidRule) Description() string {
	return "version must be valid"
}

func (r *VersionValidRule) AppliesTo() []string {
	return []string{"*"}
}

func (r *VersionValidRule) Examples() rules.Examples {
	return rules.Examples{
		Valid:   []string{specs.SpecVersionV1, specs.SpecVersionV0_1},
		Invalid: []string{"rudder/v0", "rudder/v1.1"},
	}
}
