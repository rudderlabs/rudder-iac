package rules

import (
	"fmt"
	"slices"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

type SpecSemanticValidRule struct {
	validKinds    []string
	validVersions []string
}

func NewSpecSemanticValidRule(validKinds, validVersions []string) rules.Rule {
	return &SpecSemanticValidRule{
		validKinds:    validKinds,
		validVersions: validVersions,
	}
}

func (r *SpecSemanticValidRule) ID() string {
	return "project/spec-values-valid"
}

func (r *SpecSemanticValidRule) Severity() rules.Severity {
	return rules.Error
}

func (r *SpecSemanticValidRule) Description() string {
	return "spec kind and version must be valid and supported"
}

func (r *SpecSemanticValidRule) AppliesTo() []string {
	return []string{"*"}
}

func (r *SpecSemanticValidRule) Validate(ctx *rules.ValidationContext) []rules.ValidationResult {
	results := []rules.ValidationResult{}

	// Validate version against injected valid versions
	if !slices.Contains(r.validVersions, ctx.Version) {
		results = append(results, rules.ValidationResult{
			Reference: "/version",
			Message:   fmt.Sprintf("version '%s' is not supported, supported versions: %v", ctx.Version, r.validVersions),
		})
	}

	// Validate kind against injected valid kinds
	if !slices.Contains(r.validKinds, ctx.Kind) {
		results = append(results, rules.ValidationResult{
			Reference: "/kind",
			Message:   fmt.Sprintf("kind '%s' is not supported, supported kinds: %v", ctx.Kind, r.validKinds),
		})
	}

	return results
}

func (r *SpecSemanticValidRule) Examples() rules.Examples {
	return rules.Examples{
		Valid: []string{
			heredoc.Doc(`
				version: rudder/v1
				kind: properties
				metadata:
				  name: my-properties
				spec:
				  properties:
				    - name: MyTestProperty
				      type: string
			`),
			heredoc.Doc(`
				version: rudder/0.1
				kind: events
				metadata:
				  name: my-events
				spec:
				  events:
				    - name: MyTestEvent
			`),
		},
		Invalid: []string{
			heredoc.Doc(`
				version: rudder/v2  # unsupported version
				kind: properties
				metadata:
				  name: my-properties
				spec:
				  properties:
				    - name: MyTestProperty
				      type: string
			`),
			heredoc.Doc(`
				version: rudder/v1
				kind: unsupported-kind  # unsupported kind
				metadata:
				  name: my-test
				spec:
				  data: []
			`),
		},
	}
}
