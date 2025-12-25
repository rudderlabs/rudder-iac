package datacatalog

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/location"
)

// categoryNameRegex validates category name format
// Must start with a letter (upper/lower) or underscore, followed by 2-64 characters
// including spaces, word characters, commas, periods, and hyphens
var categoryNameRegex = regexp.MustCompile(`^[A-Z_a-z][\s\w,.-]{2,64}$`)

// CategoryNameRule validates that category names follow the required format
type CategoryNameRule struct{}

// ID returns the rule identifier
func (r *CategoryNameRule) ID() string {
	return "datacatalog/categories/name-format"
}

// Severity returns the severity of this rule
func (r *CategoryNameRule) Severity() validation.Severity {
	return validation.SeverityError
}

// Description returns a description of the rule
func (r *CategoryNameRule) Description() string {
	return "Ensures that category names start with a letter or underscore, followed by 2-64 characters including spaces, word characters, commas, periods, and hyphens"
}

// Examples returns examples for the rule
func (r *CategoryNameRule) Examples() [][]byte {
	return [][]byte{
		[]byte(`
version: rudder/v0.1
kind: categories
metadata:
  name: example
spec:
  categories:
    - id: "user_actions"
      name: "User Actions"
`),
	}
}

// AppliesTo returns the kinds this rule applies to
func (r *CategoryNameRule) AppliesTo() []string {
	return []string{"categories"}
}

// Validate executes the validation logic
func (r *CategoryNameRule) Validate(ctx *validation.ValidationContext, graph *resources.Graph) []validation.ValidationError {
	var errors []validation.ValidationError

	specMap, ok := ctx.Spec.(map[string]any)
	if !ok {
		return errors
	}

	categories, ok := specMap["categories"].([]any)
	if !ok {
		return errors
	}

	for i, cat := range categories {
		category, ok := cat.(map[string]any)
		if !ok {
			continue
		}

		name, ok := category["name"].(string)
		if !ok || name == "" {
			// Skip - required fields rule handles this case
			continue
		}

		log.Printf("[category_name] Validating category name: %s", name)

		// Check for leading/trailing whitespace
		if name != strings.TrimSpace(name) {
			path := fmt.Sprintf("/spec/categories/%d/name", i)
			errors = append(errors, validation.ValidationError{
				Msg:      "category name cannot have leading or trailing whitespace characters",
				Fragment: "name",
				Pos:      r.getPosition(ctx, path),
			})
			continue
		}

		log.Printf("[category_name] Category name matches regex: %v", categoryNameRegex.MatchString(name))
		// Check regex format
		if !categoryNameRegex.MatchString(name) {
			path := fmt.Sprintf("/spec/categories/%d/name", i)
			errors = append(errors, validation.ValidationError{
				Msg:      "category name must start with a letter or underscore, followed by 2-64 characters including spaces, word characters, commas, periods, and hyphens",
				Fragment: "name",
				Pos:      r.getPosition(ctx, path),
			})
		}
	}

	return errors
}

func (r *CategoryNameRule) getPosition(ctx *validation.ValidationContext, path string) location.Position {
	if ctx.PathIndex != nil {
		if pos := ctx.PathIndex.Lookup(path); pos != nil {
			return *pos
		}
	}
	return location.Position{}
}
