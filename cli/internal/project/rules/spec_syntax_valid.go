package rules

import (
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

type SpecSyntaxValidRule struct {
	appliesToVersions []string
}

func NewSpecSyntaxValidRule(appliesToVersions []string) rules.Rule {
	return &SpecSyntaxValidRule{
		appliesToVersions: appliesToVersions,
	}
}

func (r *SpecSyntaxValidRule) ID() string {
	return "project/spec-syntax-valid"
}

func (r *SpecSyntaxValidRule) Severity() rules.Severity {
	return rules.Error
}

func (r *SpecSyntaxValidRule) Description() string {
	return "spec syntax must be valid"
}

func (r *SpecSyntaxValidRule) AppliesToKinds() []string {
	return []string{"*"}
}

func (r *SpecSyntaxValidRule) AppliesToVersions() []string {
	return r.appliesToVersions
}

func (r *SpecSyntaxValidRule) Validate(ctx *rules.ValidationContext) []rules.ValidationResult {
	results := []rules.ValidationResult{}

	if ctx.Kind == "" {
		results = append(results, rules.ValidationResult{
			Reference: "/kind",
			Message:   "'kind' is required",
		})
	}

	if ctx.Version == "" {
		results = append(results, rules.ValidationResult{
			Reference: "/version",
			Message:   "'version' is required",
		})
	}

	if len(ctx.Metadata) == 0 {
		results = append(results, rules.ValidationResult{
			Reference: "/metadata",
			Message:   "'metadata' is required",
		})
	}

	if len(ctx.Spec) == 0 {
		results = append(results, rules.ValidationResult{
			Reference: "/spec",
			Message:   "'spec' is required",
		})
	}

	return results
}

func (r *SpecSyntaxValidRule) Examples() rules.Examples {
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
		},
		Invalid: []string{
			heredoc.Doc(`
version: rudder/v1
kind: # missing kind
metadata:
  name: my-properties
spec:
  properties:
    - name: MyTestProperty
      type: string
`),
			heredoc.Doc(`
version: # missing version
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
kind: properties
metadata: # missing metadata name
spec:
  properties:
    - name: MyTestProperty
      type: string
`),
			heredoc.Doc(`
version: rudder/v1
kind: properties
metadata:
  name: my-properties
# missing spec
`),
		},
	}

}
