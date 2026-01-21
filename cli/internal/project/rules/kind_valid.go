package rules

import (
	"fmt"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/samber/lo"
)

type KindValidRule struct {
	id         string
	validKinds []string
}

func NewKindValidRule(kinds []string) rules.Rule {
	return &KindValidRule{
		id:         "project/kind-valid",
		validKinds: kinds,
	}

}

func (r *KindValidRule) Validate(ctx *rules.ValidationContext) []rules.ValidationResult {

	if ctx.Kind != "" && !lo.Contains(r.validKinds, ctx.Kind) {
		return []rules.ValidationResult{
			{
				Reference: "/kind",
				Message:   "kind must be one of the supported kinds",
			},
		}
	}
	return nil
}

func (r *KindValidRule) ID() string {
	return r.id
}

func (r *KindValidRule) Severity() rules.Severity {
	return rules.Error
}

func (r *KindValidRule) Description() string {
	return fmt.Sprintf("kind needs to be one of the supported kinds: [%s]", strings.Join(r.validKinds, ", "))
}

func (r *KindValidRule) AppliesTo() []string {
	return []string{"*"}
}

func (r *KindValidRule) Examples() rules.Examples {
	return rules.Examples{
		Valid:   r.validKinds,
		Invalid: []string{"invalid-kind"},
	}
}
