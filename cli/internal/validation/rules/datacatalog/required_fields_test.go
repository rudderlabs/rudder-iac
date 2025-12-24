package datacatalog

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/validation"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/location"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequiredFieldsRule(t *testing.T) {
	rule := &RequiredFieldsRule{}

	t.Run("valid properties", func(t *testing.T) {
		ctx := &validation.ValidationContext{
			Kind: "properties",
			Spec: map[string]any{
				"properties": []any{
					map[string]any{
						"id":   "prop1",
						"name": "Property 1",
					},
				},
			},
		}
		errors := rule.Validate(ctx, nil)
		assert.Empty(t, errors)
	})

	t.Run("missing property id and name", func(t *testing.T) {
		yamlData := `
version: rudder/v0.1
kind: properties
spec:
  properties:
    - description: "missing id and name"`

		idx, err := location.YAMLDataIndex([]byte(yamlData))
		require.NoError(t, err)

		ctx := &validation.ValidationContext{
			Kind: "properties",
			Spec: map[string]any{
				"properties": []any{
					map[string]any{
						"description": "missing id and name",
					},
				},
			},
			PathIndex: idx,
		}

		errors := rule.Validate(ctx, nil)
		require.Len(t, errors, 2)

		assert.Equal(t, "property 'id' is mandatory", errors[0].Msg)
		assert.Equal(t, "id", errors[0].Fragment)
		// Since 'id' is missing, it won't be found by PathIndex for /spec/properties/0/id
		// unless we indexed the parent or handle missing keys.
		// My PathIndex currently only indexes what's in the YAML.

		assert.Equal(t, "property 'name' is mandatory", errors[1].Msg)
		assert.Equal(t, "name", errors[1].Fragment)
	})

	t.Run("valid events", func(t *testing.T) {
		ctx := &validation.ValidationContext{
			Kind: "events",
			Spec: map[string]any{
				"events": []any{
					map[string]any{
						"id":         "event1",
						"event_type": "track",
					},
				},
			},
		}
		errors := rule.Validate(ctx, nil)
		assert.Empty(t, errors)
	})

	t.Run("missing event id and type", func(t *testing.T) {
		ctx := &validation.ValidationContext{
			Kind: "events",
			Spec: map[string]any{
				"events": []any{
					map[string]any{
						"name": "Missing ID and Type",
					},
				},
			},
		}
		errors := rule.Validate(ctx, nil)
		require.Len(t, errors, 2)
		assert.Equal(t, "event 'id' is mandatory", errors[0].Msg)
		assert.Equal(t, "event 'event_type' is mandatory", errors[1].Msg)
	})
}
