package validation

import (
	"context"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRule is a configurable Rule implementation for testing
type mockRule struct {
	id                string
	severity          rules.Severity
	description       string
	appliesTo         []string
	appliesToVersions []string
	examples          rules.Examples
	validateFn        func(ctx *rules.ValidationContext) []rules.ValidationResult
}

func (m *mockRule) ID() string               { return m.id }
func (m *mockRule) Severity() rules.Severity { return m.severity }
func (m *mockRule) Description() string      { return m.description }
func (m *mockRule) AppliesToKinds() []string { return m.appliesTo }
func (m *mockRule) AppliesToVersions() []string {
	if len(m.appliesToVersions) == 0 {
		return []string{"*"}
	}
	return m.appliesToVersions
}
func (m *mockRule) Examples() rules.Examples { return m.examples }
func (m *mockRule) Validate(ctx *rules.ValidationContext) []rules.ValidationResult {
	if m.validateFn != nil {
		return m.validateFn(ctx)
	}
	return nil
}

// newRawSpec creates a RawSpec from YAML string and parses it
func newRawSpec(t *testing.T, yaml string) *specs.RawSpec {
	t.Helper()
	rawSpec := &specs.RawSpec{Data: []byte(yaml)}
	_, err := rawSpec.Parse()
	require.NoError(t, err)
	return rawSpec
}

const validPropertiesYAML = `version: rudder/v0.1
kind: properties
metadata:
  name: test-properties
spec:
  group: test-group
  properties:
    - name: TestProperty
      type: string
`

const validEventsYAML = `version: rudder/v0.1
kind: events
metadata:
  name: test-events
spec:
  group: test-group
  events:
    - name: TestEvent
      description: A test event
`

func TestValidationEngine_ValidateSyntax(t *testing.T) {
	t.Parallel()

	t.Run("empty input", func(t *testing.T) {
		t.Parallel()

		t.Run("empty specs map returns empty diagnostics", func(t *testing.T) {
			t.Parallel()

			registry := rules.NewRegistry()
			engine, err := NewValidationEngine(registry, nil)
			require.NoError(t, err)

			diagnostics, err := engine.ValidateSyntax(
				context.Background(),
				map[string]*specs.RawSpec{},
			)

			require.NoError(t, err)
			assert.Empty(t, diagnostics)
		})
	})

	t.Run("spec validation", func(t *testing.T) {
		t.Parallel()

		t.Run("no matching rules returns empty diagnostics", func(t *testing.T) {
			t.Parallel()

			rawSpecs := map[string]*specs.RawSpec{
				"/path/to/test.yaml": newRawSpec(t, validPropertiesYAML),
			}

			// Registry with no rules for "properties" kind
			registry := rules.NewRegistry()
			engine, err := NewValidationEngine(registry, nil)
			require.NoError(t, err)

			diagnostics, err := engine.ValidateSyntax(context.Background(), rawSpecs)

			require.NoError(t, err)
			assert.Empty(t, diagnostics)
		})

		t.Run("all rules pass returns empty diagnostics", func(t *testing.T) {
			t.Parallel()

			rawSpecs := map[string]*specs.RawSpec{
				"/path/to/test.yaml": newRawSpec(t, validPropertiesYAML),
			}

			passingRule := &mockRule{
				id:        "passing-rule",
				severity:  rules.Error,
				appliesTo: []string{"properties"},
				validateFn: func(ctx *rules.ValidationContext) []rules.ValidationResult {
					return nil
				},
			}

			registry := rules.NewRegistry()
			require.NoError(t, registry.RegisterSyntactic(passingRule))

			engine, err := NewValidationEngine(registry, nil)
			require.NoError(t, err)

			diagnostics, err := engine.ValidateSyntax(context.Background(), rawSpecs)
			require.NoError(t, err)

			assert.Empty(t, diagnostics)
		})

		t.Run("multiple specs with multiple errors each", func(t *testing.T) {
			t.Parallel()

			rawSpecs := map[string]*specs.RawSpec{
				"/path/to/props.yaml":  newRawSpec(t, validPropertiesYAML),
				"/path/to/events.yaml": newRawSpec(t, validEventsYAML),
			}

			propsRule := &mockRule{
				id:        "props-rule",
				severity:  rules.Error,
				appliesTo: []string{"properties"},
				validateFn: func(ctx *rules.ValidationContext) []rules.ValidationResult {
					return []rules.ValidationResult{
						{Reference: "/spec/group", Message: "props error 1"},
						{Reference: "/metadata/name", Message: "props error 2"},
					}
				},
			}

			eventsRule := &mockRule{
				id:        "events-rule",
				severity:  rules.Error,
				appliesTo: []string{"events"},
				validateFn: func(ctx *rules.ValidationContext) []rules.ValidationResult {
					return []rules.ValidationResult{
						{Reference: "/spec/group", Message: "events error 1"},
						{Reference: "/metadata/name", Message: "events error 2"},
					}
				},
			}

			registry := rules.NewRegistry()
			require.NoError(t, registry.RegisterSyntactic(propsRule))
			require.NoError(t, registry.RegisterSyntactic(eventsRule))

			engine, err := NewValidationEngine(registry, nil)
			require.NoError(t, err)

			diagnostics, err := engine.ValidateSyntax(context.Background(), rawSpecs)

			require.NoError(t, err)
			require.Len(t, diagnostics, 4)

			// Verify all files are represented
			var propsCount, eventsCount int
			for _, diag := range diagnostics {
				switch diag.File {
				case "/path/to/props.yaml":
					propsCount++
					assert.Equal(t, "props-rule", diag.RuleID)
				case "/path/to/events.yaml":
					eventsCount++
					assert.Equal(t, "events-rule", diag.RuleID)
				}
			}
			assert.Equal(t, 2, propsCount)
			assert.Equal(t, 2, eventsCount)
		})
	})

	t.Run("diagnostic properties", func(t *testing.T) {
		t.Parallel()

		t.Run("diagnostic has correct file path and position", func(t *testing.T) {
			t.Parallel()

			rawSpecs := map[string]*specs.RawSpec{
				"/path/to/test.yaml": newRawSpec(t, validPropertiesYAML),
			}

			rule := &mockRule{
				id:        "position-rule",
				severity:  rules.Error,
				appliesTo: []string{"properties"},
				validateFn: func(ctx *rules.ValidationContext) []rules.ValidationResult {
					return []rules.ValidationResult{
						{Reference: "/spec/group", Message: "test error"},
					}
				},
			}

			registry := rules.NewRegistry()
			require.NoError(t, registry.RegisterSyntactic(rule))

			engine, err := NewValidationEngine(registry, nil)
			require.NoError(t, err)

			diagnostics, err := engine.ValidateSyntax(context.Background(), rawSpecs)

			require.NoError(t, err)
			require.Len(t, diagnostics, 1)

			diag := diagnostics[0]
			assert.Equal(t, "/path/to/test.yaml", diag.File)
			assert.Greater(t, diag.Position.Line, 0)
			assert.Greater(t, diag.Position.Column, 0)
		})

		t.Run("diagnostics are sorted", func(t *testing.T) {
			t.Parallel()

			rawSpecs := map[string]*specs.RawSpec{
				"/b/test.yaml": newRawSpec(t, validPropertiesYAML),
				"/a/test.yaml": newRawSpec(t, validEventsYAML),
			}

			rule := &mockRule{
				id:        "sort-rule",
				severity:  rules.Error,
				appliesTo: []string{"properties", "events"},
				validateFn: func(ctx *rules.ValidationContext) []rules.ValidationResult {
					return []rules.ValidationResult{
						{Reference: "/metadata/name", Message: "error"},
					}
				},
			}

			registry := rules.NewRegistry()
			require.NoError(t, registry.RegisterSyntactic(rule))

			engine, err := NewValidationEngine(registry, nil)
			require.NoError(t, err)

			diagnostics, err := engine.ValidateSyntax(context.Background(), rawSpecs)

			require.NoError(t, err)
			require.Len(t, diagnostics, 2)

			// Diagnostics should be sorted by file path
			assert.Equal(t, "/a/test.yaml", diagnostics[0].File)
			assert.Equal(t, "/b/test.yaml", diagnostics[1].File)
		})

		t.Run("uses nearest position when exact path not found", func(t *testing.T) {
			t.Parallel()

			rawSpecs := map[string]*specs.RawSpec{
				"/path/to/test.yaml": newRawSpec(t, validPropertiesYAML),
			}

			rule := &mockRule{
				id:        "nearest-lookup-rule",
				severity:  rules.Error,
				appliesTo: []string{"properties"},
				validateFn: func(ctx *rules.ValidationContext) []rules.ValidationResult {
					// Reference a path that doesn't exist exactly - should fall back to nearest
					return []rules.ValidationResult{
						{Reference: "/spec/properties/0/nonexistent", Message: "error at nonexistent path"},
					}
				},
			}

			registry := rules.NewRegistry()
			require.NoError(t, registry.RegisterSyntactic(rule))

			engine, err := NewValidationEngine(registry, nil)
			require.NoError(t, err)

			diagnostics, err := engine.ValidateSyntax(context.Background(), rawSpecs)

			require.NoError(t, err)
			require.Len(t, diagnostics, 1)

			// Position should resolve to nearest available path (/spec/properties/0)
			diag := diagnostics[0]
			assert.Equal(t, 8, diag.Position.Line)
			assert.Equal(t, 7, diag.Position.Column)
			assert.Equal(t, "- name: TestProperty", diag.Position.LineText)
		})
	})
}

func TestValidationEngine_ValidateSemantic(t *testing.T) {
	t.Parallel()

	t.Run("empty input", func(t *testing.T) {
		t.Parallel()

		t.Run("empty specs map returns empty diagnostics", func(t *testing.T) {
			t.Parallel()

			registry := rules.NewRegistry()
			engine, err := NewValidationEngine(registry, nil)
			require.NoError(t, err)

			graph := resources.NewGraph()
			diagnostics, err := engine.ValidateSemantic(context.Background(), map[string]*specs.RawSpec{}, graph)

			require.NoError(t, err)
			assert.Empty(t, diagnostics)
		})
	})

	t.Run("spec validation", func(t *testing.T) {
		t.Parallel()

		t.Run("all rules pass returns empty diagnostics", func(t *testing.T) {
			t.Parallel()

			rawSpecs := map[string]*specs.RawSpec{
				"/path/to/test.yaml": newRawSpec(t, validPropertiesYAML),
			}

			passingRule := &mockRule{
				id:        "passing-semantic-rule",
				severity:  rules.Error,
				appliesTo: []string{"properties"},
				validateFn: func(ctx *rules.ValidationContext) []rules.ValidationResult {
					return nil
				},
			}

			registry := rules.NewRegistry()
			require.NoError(t, registry.RegisterSemantic(passingRule))

			engine, err := NewValidationEngine(registry, nil)
			require.NoError(t, err)

			graph := resources.NewGraph()
			diagnostics, err := engine.ValidateSemantic(context.Background(), rawSpecs, graph)

			require.NoError(t, err)
			assert.Empty(t, diagnostics)
		})

		t.Run("rule fails returns diagnostic", func(t *testing.T) {
			t.Parallel()

			rawSpecs := map[string]*specs.RawSpec{
				"/path/to/test.yaml": newRawSpec(t, validPropertiesYAML),
			}

			failingRule := &mockRule{
				id:        "failing-semantic-rule",
				severity:  rules.Error,
				appliesTo: []string{"properties"},
				validateFn: func(ctx *rules.ValidationContext) []rules.ValidationResult {
					return []rules.ValidationResult{
						{Reference: "/spec/group", Message: "semantic error"},
					}
				},
			}

			registry := rules.NewRegistry()
			require.NoError(t, registry.RegisterSemantic(failingRule))

			engine, err := NewValidationEngine(registry, nil)
			require.NoError(t, err)

			graph := resources.NewGraph()
			diagnostics, err := engine.ValidateSemantic(context.Background(), rawSpecs, graph)

			require.NoError(t, err)
			require.Len(t, diagnostics, 1)

			diag := diagnostics[0]
			assert.Equal(t, "failing-semantic-rule", diag.RuleID)
			assert.Equal(t, rules.Error, diag.Severity)
			assert.Equal(t, "semantic error", diag.Message)
			assert.Equal(t, "/path/to/test.yaml", diag.File)
		})

		t.Run("semantic rules receive populated graph", func(t *testing.T) {
			t.Parallel()

			rawSpecs := map[string]*specs.RawSpec{
				"/path/to/test.yaml": newRawSpec(t, validPropertiesYAML),
			}

			var receivedGraph *resources.Graph

			graphCheckRule := &mockRule{
				id:        "graph-check-rule",
				severity:  rules.Error,
				appliesTo: []string{"properties"},
				validateFn: func(ctx *rules.ValidationContext) []rules.ValidationResult {
					receivedGraph = ctx.Graph
					return nil
				},
			}

			registry := rules.NewRegistry()
			require.NoError(t, registry.RegisterSemantic(graphCheckRule))

			engine, err := NewValidationEngine(registry, nil)
			require.NoError(t, err)

			testGraph := resources.NewGraph()
			_, err = engine.ValidateSemantic(context.Background(), rawSpecs, testGraph)

			require.NoError(t, err)
			assert.NotNil(t, receivedGraph)
			assert.Same(t, testGraph, receivedGraph)
		})

		t.Run("multiple specs with failing rules returns multiple diagnostics", func(t *testing.T) {
			t.Parallel()

			rawSpecs := map[string]*specs.RawSpec{
				"/path/to/props.yaml":  newRawSpec(t, validPropertiesYAML),
				"/path/to/events.yaml": newRawSpec(t, validEventsYAML),
			}

			propsRule := &mockRule{
				id:        "props-semantic-rule",
				severity:  rules.Error,
				appliesTo: []string{"properties"},
				validateFn: func(ctx *rules.ValidationContext) []rules.ValidationResult {
					return []rules.ValidationResult{
						{Reference: "/spec/group", Message: "properties semantic error"},
					}
				},
			}

			eventsRule := &mockRule{
				id:        "events-semantic-rule",
				severity:  rules.Error,
				appliesTo: []string{"events"},
				validateFn: func(ctx *rules.ValidationContext) []rules.ValidationResult {
					return []rules.ValidationResult{
						{Reference: "/spec/group", Message: "events semantic error"},
					}
				},
			}

			registry := rules.NewRegistry()
			require.NoError(t, registry.RegisterSemantic(propsRule))
			require.NoError(t, registry.RegisterSemantic(eventsRule))

			engine, err := NewValidationEngine(registry, nil)
			require.NoError(t, err)

			graph := resources.NewGraph()
			diagnostics, err := engine.ValidateSemantic(context.Background(), rawSpecs, graph)

			require.NoError(t, err)
			require.Len(t, diagnostics, 2)

			files := []string{diagnostics[0].File, diagnostics[1].File}
			assert.Contains(t, files, "/path/to/props.yaml")
			assert.Contains(t, files, "/path/to/events.yaml")

			messages := []string{diagnostics[0].Message, diagnostics[1].Message}
			assert.Contains(t, messages, "properties semantic error")
			assert.Contains(t, messages, "events semantic error")
		})
	})
}

// mockProjectRule implements both Rule and ProjectRule
type mockProjectRule struct {
	mockRule
	validateProjectFn func(specs map[string]*rules.ValidationContext) map[string][]rules.ValidationResult
}

func (m *mockProjectRule) ValidateProject(specs map[string]*rules.ValidationContext) map[string][]rules.ValidationResult {
	if m.validateProjectFn != nil {
		return m.validateProjectFn(specs)
	}
	return nil
}

func TestValidationEngine_ProjectRules(t *testing.T) {
	t.Parallel()

	t.Run("project rule runs after per-spec rules pass", func(t *testing.T) {
		t.Parallel()

		rawSpecs := map[string]*specs.RawSpec{
			"/path/to/test.yaml": newRawSpec(t, validPropertiesYAML),
		}

		pr := &mockProjectRule{
			mockRule: mockRule{
				id:        "project-rule",
				severity:  rules.Error,
				appliesTo: []string{"*"},
			},
			validateProjectFn: func(specs map[string]*rules.ValidationContext) map[string][]rules.ValidationResult {
				return map[string][]rules.ValidationResult{
					"/path/to/test.yaml": {
						{Reference: "/spec/group", Message: "project-wide error"},
					},
				}
			},
		}

		registry := rules.NewRegistry()
		require.NoError(t, registry.RegisterSyntactic(pr))

		engine, err := NewValidationEngine(registry, nil)
		require.NoError(t, err)

		diagnostics, err := engine.ValidateSyntax(context.Background(), rawSpecs)
		require.NoError(t, err)
		require.Len(t, diagnostics, 1)

		assert.Equal(t, "project-rule", diagnostics[0].RuleID)
		assert.Equal(t, "project-wide error", diagnostics[0].Message)
		assert.Greater(t, diagnostics[0].Position.Line, 0)
	})

	t.Run("project rule skipped when per-spec errors exist", func(t *testing.T) {
		t.Parallel()

		rawSpecs := map[string]*specs.RawSpec{
			"/path/to/test.yaml": newRawSpec(t, validPropertiesYAML),
		}

		projectRuleCalled := false

		failingPerSpec := &mockRule{
			id:        "failing-rule",
			severity:  rules.Error,
			appliesTo: []string{"properties"},
			validateFn: func(ctx *rules.ValidationContext) []rules.ValidationResult {
				return []rules.ValidationResult{
					{Reference: "/spec/group", Message: "syntax error"},
				}
			},
		}

		pr := &mockProjectRule{
			mockRule: mockRule{
				id:        "project-rule",
				severity:  rules.Error,
				appliesTo: []string{"*"},
			},
			validateProjectFn: func(specs map[string]*rules.ValidationContext) map[string][]rules.ValidationResult {
				projectRuleCalled = true
				return map[string][]rules.ValidationResult{
					"/path/to/test.yaml": {
						{Reference: "/spec/group", Message: "should not appear"},
					},
				}
			},
		}

		registry := rules.NewRegistry()
		require.NoError(t, registry.RegisterSyntactic(failingPerSpec))
		require.NoError(t, registry.RegisterSyntactic(pr))

		engine, err := NewValidationEngine(registry, nil)
		require.NoError(t, err)

		diagnostics, err := engine.ValidateSyntax(context.Background(), rawSpecs)
		require.NoError(t, err)

		assert.False(t, projectRuleCalled)
		require.Len(t, diagnostics, 1)
		assert.Equal(t, "syntax error", diagnostics[0].Message)
	})

	t.Run("project rule receives all specs", func(t *testing.T) {
		t.Parallel()

		rawSpecs := map[string]*specs.RawSpec{
			"/path/to/props.yaml":  newRawSpec(t, validPropertiesYAML),
			"/path/to/events.yaml": newRawSpec(t, validEventsYAML),
		}

		var receivedPaths []string

		pr := &mockProjectRule{
			mockRule: mockRule{
				id:        "capture-rule",
				severity:  rules.Error,
				appliesTo: []string{"*"},
			},
			validateProjectFn: func(specs map[string]*rules.ValidationContext) map[string][]rules.ValidationResult {
				for path := range specs {
					receivedPaths = append(receivedPaths, path)
				}
				return nil
			},
		}

		registry := rules.NewRegistry()
		require.NoError(t, registry.RegisterSyntactic(pr))

		engine, err := NewValidationEngine(registry, nil)
		require.NoError(t, err)

		_, err = engine.ValidateSyntax(context.Background(), rawSpecs)
		require.NoError(t, err)

		assert.Len(t, receivedPaths, 2)
		assert.Contains(t, receivedPaths, "/path/to/props.yaml")
		assert.Contains(t, receivedPaths, "/path/to/events.yaml")
	})

}
