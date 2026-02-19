package rules

import (
	"fmt"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDuplicateLocalIDRule_Metadata(t *testing.T) {
	t.Parallel()

	rule := NewDuplicateLocalIDRule(nil)

	assert.Equal(t, "project/duplicate-local-id", rule.ID())
	assert.Equal(t, rules.Error, rule.Severity())
	assert.Equal(t, []string{"*"}, rule.AppliesTo())
	assert.Nil(t, rule.Validate(nil))
}

func TestDuplicateLocalIDRule_ValidateProject(t *testing.T) {
	t.Parallel()

	parseSpec := func(_ string, s *specs.Spec) (*specs.ParsedSpec, error) {
		parsed := &specs.ParsedSpec{}
		switch s.Kind {
		case "properties":
			if props, ok := s.Spec["properties"].([]any); ok {
				for i, p := range props {
					if m, ok := p.(map[string]any); ok {
						if id, ok := m["id"].(string); ok {
							parsed.LocalIDs = append(parsed.LocalIDs, specs.LocalID{
								ID:              id,
								JSONPointerPath: fmt.Sprintf("/spec/properties/%d/id", i),
							})
						}
					}
				}
			}
		case "events":
			if events, ok := s.Spec["events"].([]any); ok {
				for i, e := range events {
					if m, ok := e.(map[string]any); ok {
						if id, ok := m["id"].(string); ok {
							parsed.LocalIDs = append(parsed.LocalIDs, specs.LocalID{
								ID:              id,
								JSONPointerPath: fmt.Sprintf("/spec/events/%d/id", i),
							})
						}
					}
				}
			}
		case "tp":
			if id, ok := s.Spec["id"].(string); ok {
				parsed.LocalIDs = append(parsed.LocalIDs, specs.LocalID{
					ID:              id,
					JSONPointerPath: "/spec/id",
				})
			}
		}
		return parsed, nil
	}

	t.Run("no duplicates", func(t *testing.T) {
		t.Parallel()

		rule := NewDuplicateLocalIDRule(parseSpec)
		pr := rule.(rules.ProjectRule)

		results := pr.ValidateProject(map[string]*rules.ValidationContext{
			"props1.yaml": {Kind: "properties", Spec: map[string]any{
				"properties": []any{
					map[string]any{"id": "email"},
					map[string]any{"id": "name"},
				},
			}},
			"props2.yaml": {Kind: "properties", Spec: map[string]any{
				"properties": []any{
					map[string]any{"id": "age"},
				},
			}},
		})

		assert.Empty(t, results)
	})

	t.Run("duplicate across files", func(t *testing.T) {
		t.Parallel()

		rule := NewDuplicateLocalIDRule(parseSpec)
		pr := rule.(rules.ProjectRule)

		results := pr.ValidateProject(map[string]*rules.ValidationContext{
			"props1.yaml": {Kind: "properties", Spec: map[string]any{
				"properties": []any{
					map[string]any{"id": "email"},
				},
			}},
			"props2.yaml": {Kind: "properties", Spec: map[string]any{
				"properties": []any{
					map[string]any{"id": "email"},
				},
			}},
		})

		require.Len(t, results, 2)
		assert.Len(t, results["props1.yaml"], 1)
		assert.Len(t, results["props2.yaml"], 1)
		assert.Contains(t, results["props1.yaml"][0].Message, "duplicate id 'email'")
		assert.Contains(t, results["props2.yaml"][0].Message, "duplicate id 'email'")
	})

	t.Run("duplicate within same file", func(t *testing.T) {
		t.Parallel()

		rule := NewDuplicateLocalIDRule(parseSpec)
		pr := rule.(rules.ProjectRule)

		results := pr.ValidateProject(map[string]*rules.ValidationContext{
			"props.yaml": {Kind: "properties", Spec: map[string]any{
				"properties": []any{
					map[string]any{"id": "email"},
					map[string]any{"id": "email"},
				},
			}},
		})

		require.Len(t, results, 1)
		assert.Len(t, results["props.yaml"], 2)
		assert.Equal(t, "/spec/properties/0/id", results["props.yaml"][0].Reference)
		assert.Equal(t, "/spec/properties/1/id", results["props.yaml"][1].Reference)
	})

	t.Run("same ID across different kinds is allowed", func(t *testing.T) {
		t.Parallel()

		rule := NewDuplicateLocalIDRule(parseSpec)
		pr := rule.(rules.ProjectRule)

		results := pr.ValidateProject(map[string]*rules.ValidationContext{
			"props.yaml": {Kind: "properties", Spec: map[string]any{
				"properties": []any{
					map[string]any{"id": "user_id"},
				},
			}},
			"events.yaml": {Kind: "events", Spec: map[string]any{
				"events": []any{
					map[string]any{"id": "user_id"},
				},
			}},
		})

		assert.Empty(t, results)
	})

	t.Run("three files with same ID", func(t *testing.T) {
		t.Parallel()

		rule := NewDuplicateLocalIDRule(parseSpec)
		pr := rule.(rules.ProjectRule)

		results := pr.ValidateProject(map[string]*rules.ValidationContext{
			"a.yaml": {Kind: "properties", Spec: map[string]any{
				"properties": []any{map[string]any{"id": "dup"}},
			}},
			"b.yaml": {Kind: "properties", Spec: map[string]any{
				"properties": []any{map[string]any{"id": "dup"}},
			}},
			"c.yaml": {Kind: "properties", Spec: map[string]any{
				"properties": []any{map[string]any{"id": "dup"}},
			}},
		})

		require.Len(t, results, 3)
		assert.Len(t, results["a.yaml"], 1)
		assert.Len(t, results["b.yaml"], 1)
		assert.Len(t, results["c.yaml"], 1)
	})

	t.Run("mixed duplicates and unique", func(t *testing.T) {
		t.Parallel()

		rule := NewDuplicateLocalIDRule(parseSpec)
		pr := rule.(rules.ProjectRule)

		results := pr.ValidateProject(map[string]*rules.ValidationContext{
			"a.yaml": {Kind: "properties", Spec: map[string]any{
				"properties": []any{map[string]any{"id": "dup"}},
			}},
			"b.yaml": {Kind: "properties", Spec: map[string]any{
				"properties": []any{map[string]any{"id": "dup"}},
			}},
			"c.yaml": {Kind: "properties", Spec: map[string]any{
				"properties": []any{map[string]any{"id": "unique1"}},
			}},
			"d.yaml": {Kind: "properties", Spec: map[string]any{
				"properties": []any{map[string]any{"id": "unique2"}},
			}},
		})

		require.Len(t, results, 2)
		assert.Contains(t, results, "a.yaml")
		assert.Contains(t, results, "b.yaml")
	})

	t.Run("tracking plan duplicate IDs", func(t *testing.T) {
		t.Parallel()

		rule := NewDuplicateLocalIDRule(parseSpec)
		pr := rule.(rules.ProjectRule)

		results := pr.ValidateProject(map[string]*rules.ValidationContext{
			"tp1.yaml": {Kind: "tp", Spec: map[string]any{"id": "my_tp"}},
			"tp2.yaml": {Kind: "tp", Spec: map[string]any{"id": "my_tp"}},
		})

		require.Len(t, results, 2)
		assert.Equal(t, "/spec/id", results["tp1.yaml"][0].Reference)
		assert.Equal(t, "/spec/id", results["tp2.yaml"][0].Reference)
	})
}
