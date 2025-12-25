package datacatalog

import (
	"fmt"
	"regexp"

	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/location"
)

// customTypeNameRegex validates custom type name format
// Must start with a capital letter, followed by 2-64 alphanumeric characters, underscores, or dashes
var customTypeNameRegex = regexp.MustCompile(`^[A-Z][a-zA-Z0-9_-]{2,64}$`)

// CustomTypeNameRule validates that custom type names follow the required format
type CustomTypeNameRule struct{}

// ID returns the rule identifier
func (r *CustomTypeNameRule) ID() string {
	return "datacatalog/custom-types/name-format"
}

// Severity returns the severity of this rule
func (r *CustomTypeNameRule) Severity() validation.Severity {
	return validation.SeverityError
}

// Description returns a description of the rule
func (r *CustomTypeNameRule) Description() string {
	return "Ensures that custom type names start with a capital letter and contain only letters, numbers, underscores and dashes, with length between 3-65 characters"
}

// Examples returns examples for the rule
func (r *CustomTypeNameRule) Examples() [][]byte {
	return [][]byte{
		[]byte(`
version: rudder/v0.1
kind: custom-types
metadata:
  name: example
spec:
  custom_types:
    - id: "EmailType"
      name: "EmailType"
      type: "string"
      config:
        format: "email"
`),
	}
}

// AppliesTo returns the kinds this rule applies to
func (r *CustomTypeNameRule) AppliesTo() []string {
	return []string{"custom-types"}
}

// Validate executes the validation logic
func (r *CustomTypeNameRule) Validate(ctx *validation.ValidationContext, graph *resources.Graph) []validation.ValidationError {
	var errors []validation.ValidationError

	specMap, ok := ctx.Spec.(map[string]any)
	if !ok {
		return errors
	}

	customTypes, ok := specMap["custom_types"].([]any)
	if !ok {
		return errors
	}

	for i, ct := range customTypes {
		customType, ok := ct.(map[string]any)
		if !ok {
			continue
		}

		name, ok := customType["name"].(string)
		if !ok || name == "" {
			// Skip - required fields rule handles this case
			continue
		}

		if !customTypeNameRegex.MatchString(name) {
			path := fmt.Sprintf("/spec/custom_types/%d/name", i)
			errors = append(errors, validation.ValidationError{
				Msg:      "custom type name must start with a capital letter and contain only letters, numbers, underscores and dashes, with length between 3-65 characters",
				Fragment: "name",
				Pos:      r.getPosition(ctx, path),
			})
		}
	}

	return errors
}

func (r *CustomTypeNameRule) getPosition(ctx *validation.ValidationContext, path string) location.Position {
	if ctx.PathIndex != nil {
		if pos := ctx.PathIndex.Lookup(path); pos != nil {
			return *pos
		}
	}
	return location.Position{}
}
