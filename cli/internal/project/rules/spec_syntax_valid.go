package rules

import (
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

type SpecSyntaxValidRule struct {
}

func NewSpecSyntaxValidRule() rules.Rule {
	return &SpecSyntaxValidRule{}
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

func (r *SpecSyntaxValidRule) AppliesTo() []string {
	return []string{"*"}
}

func (r *SpecSyntaxValidRule) Validate(ctx *rules.ValidationContext) []rules.ValidationResult {
	results := []rules.ValidationResult{}

	if ctx.Kind == "" {
		results = append(results, rules.ValidationResult{
			Reference: "/kind",
			Message:   "kind is required",
		})
	}

	if ctx.Version == "" {
		results = append(results, rules.ValidationResult{
			Reference: "/version",
			Message:   "version is required",
		})
	}

	if ctx.Metadata == nil {
		results = append(results, rules.ValidationResult{
			Reference: "/metadata",
			Message:   "metadata is required",
		})
	}

	if ctx.Spec == nil {
		results = append(results, rules.ValidationResult{
			Reference: "/spec",
			Message:   "spec is required",
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
metadata: # missing metadata
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
`),
		},
	}

}
