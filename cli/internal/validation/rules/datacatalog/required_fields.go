package datacatalog

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/location"
)

// RequiredFieldsRule validates that mandatory fields are present in data catalog resources
type RequiredFieldsRule struct{}

// ID returns the rule identifier
func (r *RequiredFieldsRule) ID() string {
	return "datacatalog/required-fields"
}

// Severity returns the severity of this rule
func (r *RequiredFieldsRule) Severity() validation.Severity {
	return validation.SeverityError
}

// Description returns a description of the rule
func (r *RequiredFieldsRule) Description() string {
	return "Ensures that mandatory fields like 'id' and 'name' are present in data catalog resources"
}

// Examples returns examples for the rule
func (r *RequiredFieldsRule) Examples() [][]byte {
	return [][]byte{
		[]byte(`
version: rudder/v0.1
kind: properties
metadata:
  name: example
spec:
  properties:
    - id: "prop1"
      name: "Property 1"
`),
	}
}

// AppliesTo returns the kinds this rule applies to
func (r *RequiredFieldsRule) AppliesTo() []string {
	return []string{"properties", "events"}
}

// Validate executes the validation logic
func (r *RequiredFieldsRule) Validate(ctx *validation.ValidationContext, graph *resources.Graph) []validation.ValidationError {
	var errors []validation.ValidationError

	specMap, ok := ctx.Spec.(map[string]any)
	if !ok {
		return errors
	}

	switch ctx.Kind {
	case "properties":
		if props, ok := specMap["properties"].([]any); ok {
			for i, p := range props {
				prop, ok := p.(map[string]any)
				if !ok {
					continue
				}

				// Check ID
				if id, ok := prop["id"].(string); !ok || id == "" {
					path := fmt.Sprintf("/spec/properties/%d/id", i)
					errors = append(errors, validation.ValidationError{
						Msg:      "property 'id' is mandatory",
						Fragment: "id",
						Pos:      r.getPosition(ctx, path, i),
					})
				}

				// Check Name
				if name, ok := prop["name"].(string); !ok || name == "" {
					path := fmt.Sprintf("/spec/properties/%d/name", i)
					errors = append(errors, validation.ValidationError{
						Msg:      "property 'name' is mandatory",
						Fragment: "name",
						Pos:      r.getPosition(ctx, path, i),
					})
				}
			}
		}
	case "events":
		if events, ok := specMap["events"].([]any); ok {
			for i, e := range events {
				event, ok := e.(map[string]any)
				if !ok {
					continue
				}

				// Check ID
				if id, ok := event["id"].(string); !ok || id == "" {
					path := fmt.Sprintf("/spec/events/%d/id", i)
					errors = append(errors, validation.ValidationError{
						Msg:      "event 'id' is mandatory",
						Fragment: "id",
						Pos:      r.getPosition(ctx, path, i),
					})
				}

				// Check Event Type
				if eventType, ok := event["event_type"].(string); !ok || eventType == "" {
					path := fmt.Sprintf("/spec/events/%d/event_type", i)
					errors = append(errors, validation.ValidationError{
						Msg:      "event 'event_type' is mandatory",
						Fragment: "event_type",
						Pos:      r.getPosition(ctx, path, i),
					})
				}
			}
		}
	}

	return errors
}

func (r *RequiredFieldsRule) getPosition(ctx *validation.ValidationContext, path string, index int) location.Position {
	if ctx.PathIndex != nil {
		if pos := ctx.PathIndex.Lookup(path); pos != nil {
			return *pos
		}
	}
	return location.Position{}
}
