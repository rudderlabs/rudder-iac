package validation

import (
	"context"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/pathindex"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRule is a configurable Rule implementation for testing
type mockRule struct {
	id          string
	severity    rules.Severity
	description string
	appliesTo   []rules.MatchPattern
	examples    rules.Examples
	validateFn  func(ctx *rules.ValidationContext) []rules.ValidationResult
}

func (m *mockRule) ID() string                      { return m.id }
func (m *mockRule) Severity() rules.Severity        { return m.severity }
func (m *mockRule) Description() string             { return m.description }
func (m *mockRule) AppliesTo() []rules.MatchPattern { return m.appliesTo }
func (m *mockRule) Examples() rules.Examples        { return m.examples }
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

// engineTestSupportedPatterns matches kinds/versions used in engine_test raw specs (rudder/v0.1).
var engineTestSupportedPatterns = []rules.MatchPattern{
	rules.MatchKindVersion("properties", "rudder/v0.1"),
	rules.MatchKindVersion("events", "rudder/v0.1"),
}

func TestValidationEngine_ValidateSyntax(t *testing.T) {
	t.Parallel()

	t.Run("empty input", func(t *testing.T) {
		t.Parallel()

		t.Run("empty specs map returns empty diagnostics", func(t *testing.T) {
			t.Parallel()

			registry := rules.NewRegistry(nil)
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
			registry := rules.NewRegistry(nil)
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
				appliesTo: []rules.MatchPattern{rules.MatchKind("properties")},
				validateFn: func(ctx *rules.ValidationContext) []rules.ValidationResult {
					return nil
				},
			}

			registry := rules.NewRegistry(engineTestSupportedPatterns)
			require.NoError(t, registry.RegisterSyntactic(passingRule))

			engine, err := NewValidationEngine(registry, nil)
			require.NoError(t, err)

			diagnostics, err := engine.ValidateSyntax(context.Background(), rawSpecs)
			require.NoError(t, err)

			assert.Empty(t, diagnostics)
		})

		t.Run("per-spec rules receive file path metadata", func(t *testing.T) {
			t.Parallel()

			rawSpecs := map[string]*specs.RawSpec{
				"/path/to/test.yaml": newRawSpec(t, validPropertiesYAML),
			}

			var (
				receivedFilePath string
				receivedFileName string
			)

			rule := &mockRule{
				id:        "file-path-rule",
				severity:  rules.Error,
				appliesTo: []rules.MatchPattern{rules.MatchKind("properties")},
				validateFn: func(ctx *rules.ValidationContext) []rules.ValidationResult {
					receivedFilePath = ctx.FilePath
					receivedFileName = ctx.FileName
					return nil
				},
			}

			registry := rules.NewRegistry(engineTestSupportedPatterns)
			require.NoError(t, registry.RegisterSyntactic(rule))

			engine, err := NewValidationEngine(registry, nil)
			require.NoError(t, err)

			diagnostics, err := engine.ValidateSyntax(context.Background(), rawSpecs)
			require.NoError(t, err)

			assert.Empty(t, diagnostics)
			assert.Equal(t, "/path/to/test.yaml", receivedFilePath)
			assert.Equal(t, "test.yaml", receivedFileName)
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
				appliesTo: []rules.MatchPattern{rules.MatchKind("properties")},
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
				appliesTo: []rules.MatchPattern{rules.MatchKind("events")},
				validateFn: func(ctx *rules.ValidationContext) []rules.ValidationResult {
					return []rules.ValidationResult{
						{Reference: "/spec/group", Message: "events error 1"},
						{Reference: "/metadata/name", Message: "events error 2"},
					}
				},
			}

			registry := rules.NewRegistry(engineTestSupportedPatterns)
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
				appliesTo: []rules.MatchPattern{rules.MatchKind("properties")},
				validateFn: func(ctx *rules.ValidationContext) []rules.ValidationResult {
					return []rules.ValidationResult{
						{Reference: "/spec/group", Message: "test error"},
					}
				},
			}

			registry := rules.NewRegistry(engineTestSupportedPatterns)
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
				appliesTo: []rules.MatchPattern{rules.MatchKind("properties"), rules.MatchKind("events")},
				validateFn: func(ctx *rules.ValidationContext) []rules.ValidationResult {
					return []rules.ValidationResult{
						{Reference: "/metadata/name", Message: "error"},
					}
				},
			}

			registry := rules.NewRegistry(engineTestSupportedPatterns)
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
				appliesTo: []rules.MatchPattern{rules.MatchKind("properties")},
				validateFn: func(ctx *rules.ValidationContext) []rules.ValidationResult {
					// Reference a path that doesn't exist exactly - should fall back to nearest
					return []rules.ValidationResult{
						{Reference: "/spec/properties/0/nonexistent", Message: "error at nonexistent path"},
					}
				},
			}

			registry := rules.NewRegistry(engineTestSupportedPatterns)
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

			registry := rules.NewRegistry(nil)
			engine, err := NewValidationEngine(registry, nil)
			require.NoError(t, err)

			graph := resources.NewGraph()
			diagnostics, err := engine.ValidateSemantic(context.Background(), map[string]*specs.RawSpec{}, graph, "")

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
				appliesTo: []rules.MatchPattern{rules.MatchKind("properties")},
				validateFn: func(ctx *rules.ValidationContext) []rules.ValidationResult {
					return nil
				},
			}

			registry := rules.NewRegistry(engineTestSupportedPatterns)
			require.NoError(t, registry.RegisterSemantic(passingRule))

			engine, err := NewValidationEngine(registry, nil)
			require.NoError(t, err)

			graph := resources.NewGraph()
			diagnostics, err := engine.ValidateSemantic(context.Background(), rawSpecs, graph, "")

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
				appliesTo: []rules.MatchPattern{rules.MatchKind("properties")},
				validateFn: func(ctx *rules.ValidationContext) []rules.ValidationResult {
					return []rules.ValidationResult{
						{Reference: "/spec/group", Message: "semantic error"},
					}
				},
			}

			registry := rules.NewRegistry(engineTestSupportedPatterns)
			require.NoError(t, registry.RegisterSemantic(failingRule))

			engine, err := NewValidationEngine(registry, nil)
			require.NoError(t, err)

			graph := resources.NewGraph()
			diagnostics, err := engine.ValidateSemantic(context.Background(), rawSpecs, graph, "")

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
				appliesTo: []rules.MatchPattern{rules.MatchKind("properties")},
				validateFn: func(ctx *rules.ValidationContext) []rules.ValidationResult {
					receivedGraph = ctx.Graph
					return nil
				},
			}

			registry := rules.NewRegistry(engineTestSupportedPatterns)
			require.NoError(t, registry.RegisterSemantic(graphCheckRule))

			engine, err := NewValidationEngine(registry, nil)
			require.NoError(t, err)

			testGraph := resources.NewGraph()
			_, err = engine.ValidateSemantic(context.Background(), rawSpecs, testGraph, "")

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
				appliesTo: []rules.MatchPattern{rules.MatchKind("properties")},
				validateFn: func(ctx *rules.ValidationContext) []rules.ValidationResult {
					return []rules.ValidationResult{
						{Reference: "/spec/group", Message: "properties semantic error"},
					}
				},
			}

			eventsRule := &mockRule{
				id:        "events-semantic-rule",
				severity:  rules.Error,
				appliesTo: []rules.MatchPattern{rules.MatchKind("events")},
				validateFn: func(ctx *rules.ValidationContext) []rules.ValidationResult {
					return []rules.ValidationResult{
						{Reference: "/spec/group", Message: "events semantic error"},
					}
				},
			}

			registry := rules.NewRegistry(engineTestSupportedPatterns)
			require.NoError(t, registry.RegisterSemantic(propsRule))
			require.NoError(t, registry.RegisterSemantic(eventsRule))

			engine, err := NewValidationEngine(registry, nil)
			require.NoError(t, err)

			graph := resources.NewGraph()
			diagnostics, err := engine.ValidateSemantic(context.Background(), rawSpecs, graph, "")

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

// mockMultiSpecRule implements both Rule and ProjectRule
type mockMultiSpecRule struct {
	mockRule
	validateProjectFn func(specs map[string]*rules.ValidationContext) map[string][]rules.ValidationResult
}

func (m *mockMultiSpecRule) ValidateSpecs(specs map[string]*rules.ValidationContext) map[string][]rules.ValidationResult {
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

		pr := &mockMultiSpecRule{
			mockRule: mockRule{
				id:        "project-rule",
				severity:  rules.Error,
				appliesTo: []rules.MatchPattern{rules.MatchAll()},
			},
			validateProjectFn: func(specs map[string]*rules.ValidationContext) map[string][]rules.ValidationResult {
				return map[string][]rules.ValidationResult{
					"/path/to/test.yaml": {
						{Reference: "/spec/group", Message: "project-wide error"},
					},
				}
			},
		}

		registry := rules.NewRegistry(nil)
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
			appliesTo: []rules.MatchPattern{rules.MatchKind("properties")},
			validateFn: func(ctx *rules.ValidationContext) []rules.ValidationResult {
				return []rules.ValidationResult{
					{Reference: "/spec/group", Message: "syntax error"},
				}
			},
		}

		pr := &mockMultiSpecRule{
			mockRule: mockRule{
				id:        "project-rule",
				severity:  rules.Error,
				appliesTo: []rules.MatchPattern{rules.MatchAll()},
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

		registry := rules.NewRegistry(engineTestSupportedPatterns)
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

	t.Run("project rule with exact patterns is discovered via AllSyntacticRules", func(t *testing.T) {
		t.Parallel()

		rawSpecs := map[string]*specs.RawSpec{
			"/path/to/test.yaml": newRawSpec(t, validPropertiesYAML),
		}

		pr := &mockMultiSpecRule{
			mockRule: mockRule{
				id:       "exact-pattern-project-rule",
				severity: rules.Error,
				// Exact pattern, NOT MatchAll — old SyntacticRulesFor("","") would miss this
				appliesTo: []rules.MatchPattern{rules.MatchKindVersion("properties", "rudder/v0.1")},
			},
			validateProjectFn: func(specs map[string]*rules.ValidationContext) map[string][]rules.ValidationResult {
				return map[string][]rules.ValidationResult{
					"/path/to/test.yaml": {
						{Reference: "/spec/group", Message: "exact-pattern project error"},
					},
				}
			},
		}

		registry := rules.NewRegistry(engineTestSupportedPatterns)
		require.NoError(t, registry.RegisterSyntactic(pr))

		engine, err := NewValidationEngine(registry, nil)
		require.NoError(t, err)

		diagnostics, err := engine.ValidateSyntax(context.Background(), rawSpecs)
		require.NoError(t, err)
		require.Len(t, diagnostics, 1)
		assert.Equal(t, "exact-pattern-project-rule", diagnostics[0].RuleID)
		assert.Equal(t, "exact-pattern project error", diagnostics[0].Message)
	})

	t.Run("project rule receives all specs", func(t *testing.T) {
		t.Parallel()

		rawSpecs := map[string]*specs.RawSpec{
			"/path/to/props.yaml":  newRawSpec(t, validPropertiesYAML),
			"/path/to/events.yaml": newRawSpec(t, validEventsYAML),
		}

		var receivedPaths []string

		pr := &mockMultiSpecRule{
			mockRule: mockRule{
				id:        "capture-rule",
				severity:  rules.Error,
				appliesTo: []rules.MatchPattern{rules.MatchAll()},
			},
			validateProjectFn: func(specs map[string]*rules.ValidationContext) map[string][]rules.ValidationResult {
				for path := range specs {
					receivedPaths = append(receivedPaths, path)
				}
				return nil
			},
		}

		registry := rules.NewRegistry(nil)
		require.NoError(t, registry.RegisterSyntactic(pr))

		engine, err := NewValidationEngine(registry, nil)
		require.NoError(t, err)

		_, err = engine.ValidateSyntax(context.Background(), rawSpecs)
		require.NoError(t, err)

		assert.Len(t, receivedPaths, 2)
		assert.Contains(t, receivedPaths, "/path/to/props.yaml")
		assert.Contains(t, receivedPaths, "/path/to/events.yaml")
	})

	t.Run("multiple-resource rule receives only specs matching its AppliesTo patterns", func(t *testing.T) {
		t.Parallel()

		rawSpecs := map[string]*specs.RawSpec{
			"/path/to/props.yaml":  newRawSpec(t, validPropertiesYAML),
			"/path/to/events.yaml": newRawSpec(t, validEventsYAML),
		}

		var receivedPaths []string

		// Scoped to "properties" only — must NOT receive the events spec.
		pr := &mockMultiSpecRule{
			mockRule: mockRule{
				id:        "properties-only-rule",
				severity:  rules.Error,
				appliesTo: []rules.MatchPattern{rules.MatchKind("properties")},
			},
			validateProjectFn: func(specs map[string]*rules.ValidationContext) map[string][]rules.ValidationResult {
				for path := range specs {
					receivedPaths = append(receivedPaths, path)
				}
				return nil
			},
		}

		registry := rules.NewRegistry(engineTestSupportedPatterns)
		require.NoError(t, registry.RegisterSyntactic(pr))

		engine, err := NewValidationEngine(registry, nil)
		require.NoError(t, err)

		_, err = engine.ValidateSyntax(context.Background(), rawSpecs)
		require.NoError(t, err)

		assert.Equal(t, []string{"/path/to/props.yaml"}, receivedPaths)
	})

}

// mockSourceRule implements both Rule and SourceRule
type mockSourceRule struct {
	mockRule
	validateSourcesFn func(specs map[string]*rules.ValidationContext) map[string][]rules.ValidationResult
}

func (m *mockSourceRule) ValidateSources(specs map[string]*rules.ValidationContext) map[string][]rules.ValidationResult {
	return m.validateSourcesFn(specs)
}

// sourceWithReferences is a properties spec as written. The unquoted
// reference makes the group a flow mapping, so the spec does not decode until
// substitution runs.
const sourceWithReferences = `version: rudder/v0.1
kind: properties
metadata:
  name: test-properties
spec:
  group: {{ .GROUP }}
  token: '{{ .TOKEN }}'
`

func TestValidationEngine_ValidateSources(t *testing.T) {
	t.Parallel()

	t.Run("source rule reads matching specs as written", func(t *testing.T) {
		t.Parallel()

		var got map[string]*rules.ValidationContext
		sr := &mockSourceRule{
			mockRule: mockRule{
				id:        "source-rule",
				severity:  rules.Warning,
				appliesTo: []rules.MatchPattern{rules.MatchKindVersion("properties", "rudder/v0.1")},
			},
			validateSourcesFn: func(contexts map[string]*rules.ValidationContext) map[string][]rules.ValidationResult {
				got = contexts
				return map[string][]rules.ValidationResult{
					"/path/to/props.yaml": {{Reference: "/spec/token", Message: "token as written"}},
				}
			},
		}

		registry := rules.NewRegistry(engineTestSupportedPatterns)
		require.NoError(t, registry.RegisterSyntactic(sr))
		engine, err := NewValidationEngine(registry, nil)
		require.NoError(t, err)

		sources := map[string]*specs.RawSpec{
			"/path/to/props.yaml":  {Data: []byte(sourceWithReferences)},
			"/path/to/events.yaml": {Data: []byte(validEventsYAML)},
		}
		diagnostics, err := engine.ValidateSources(context.Background(), sources)
		require.NoError(t, err)

		assert.Equal(t, map[string]*rules.ValidationContext{
			"/path/to/props.yaml": rules.NewSourceContext("/path/to/props.yaml", &specs.RawSpec{Data: []byte(sourceWithReferences)}),
		}, got)
		assert.Equal(t, Diagnostics{{
			RuleID:   "source-rule",
			Severity: rules.Warning,
			Message:  "token as written",
			File:     "/path/to/props.yaml",
			Position: pathindex.Position{Line: 7, Column: 3, LineText: "token: {{ .TOKEN }}"},
			Examples: rules.Examples{},
		}}, diagnostics)
	})

	t.Run("other rules do not run", func(t *testing.T) {
		t.Parallel()

		perSpec := &mockRule{
			id:        "per-spec-rule",
			severity:  rules.Error,
			appliesTo: []rules.MatchPattern{rules.MatchAll()},
			validateFn: func(*rules.ValidationContext) []rules.ValidationResult {
				return []rules.ValidationResult{{Message: "should not appear"}}
			},
		}
		registry := rules.NewRegistry(nil)
		require.NoError(t, registry.RegisterSyntactic(perSpec))
		engine, err := NewValidationEngine(registry, nil)
		require.NoError(t, err)

		diagnostics, err := engine.ValidateSources(context.Background(), map[string]*specs.RawSpec{
			"/path/to/props.yaml": {Data: []byte(sourceWithReferences)},
		})
		require.NoError(t, err)
		assert.Empty(t, diagnostics)
	})

	t.Run("source that is not YAML as written is indexed through placeholders", func(t *testing.T) {
		t.Parallel()

		// "/hook" after the unquoted reference is not YAML until substitution.
		// A masked line shows no value, and an unmasked one shows each
		// reference as written, never its placeholder. A key after a reference
		// on its line keeps its column as written, though the placeholder
		// before it is longer than the reference.
		source := sourceWithReferences + "  url: {{ .HOST }}/hook\n  pair: {from: '{{ .FROM }}', to: x}\n"
		sr := &mockSourceRule{
			mockRule: mockRule{id: "source-rule", severity: rules.Warning, appliesTo: []rules.MatchPattern{rules.MatchKind("properties")}},
			validateSourcesFn: func(map[string]*rules.ValidationContext) map[string][]rules.ValidationResult {
				return map[string][]rules.ValidationResult{
					"/path/to/props.yaml": {
						{Reference: "/spec/group", Message: "group as written"},
						{Reference: "/spec/token", Message: "token as written", Sensitive: true},
						{Reference: "/spec/url", Message: "url as written"},
						{Reference: "/spec/pair/to", Message: "to as written"},
					},
				}
			},
		}
		registry := rules.NewRegistry(engineTestSupportedPatterns)
		require.NoError(t, registry.RegisterSyntactic(sr))
		engine, err := NewValidationEngine(registry, nil)
		require.NoError(t, err)

		diagnostics, err := engine.ValidateSources(context.Background(), map[string]*specs.RawSpec{
			"/path/to/props.yaml": {Data: []byte(source)},
		})
		require.NoError(t, err)

		diagnostic := func(message string, position pathindex.Position, sensitive bool) Diagnostic {
			return Diagnostic{
				RuleID:    "source-rule",
				Severity:  rules.Warning,
				Message:   message,
				File:      "/path/to/props.yaml",
				Position:  position,
				Sensitive: sensitive,
				Examples:  rules.Examples{},
			}
		}
		assert.Equal(t, Diagnostics{
			diagnostic("group as written", pathindex.Position{Line: 6, Column: 3, LineText: "group: {{ .GROUP }}"}, false),
			diagnostic("token as written", pathindex.Position{Line: 7, Column: 3, LineText: "token: ***"}, true),
			diagnostic("url as written", pathindex.Position{Line: 8, Column: 3, LineText: "url: {{ .HOST }}/hook"}, false),
			diagnostic("to as written", pathindex.Position{Line: 9, Column: 31, LineText: "to: x"}, false),
		}, diagnostics)
	})

	t.Run("source that does not parse at all reports at its start", func(t *testing.T) {
		t.Parallel()

		sr := &mockSourceRule{
			mockRule: mockRule{id: "source-rule", severity: rules.Warning, appliesTo: []rules.MatchPattern{rules.MatchAll()}},
			validateSourcesFn: func(map[string]*rules.ValidationContext) map[string][]rules.ValidationResult {
				return map[string][]rules.ValidationResult{
					"/path/to/broken.yaml": {{Reference: "/spec/token", Message: "token as written"}},
				}
			},
		}
		registry := rules.NewRegistry(nil)
		require.NoError(t, registry.RegisterSyntactic(sr))
		engine, err := NewValidationEngine(registry, nil)
		require.NoError(t, err)

		diagnostics, err := engine.ValidateSources(context.Background(), map[string]*specs.RawSpec{
			"/path/to/broken.yaml": {Data: []byte("spec: [unclosed\n")},
		})
		require.NoError(t, err)
		assert.Equal(t, Diagnostics{{
			RuleID:   "source-rule",
			Severity: rules.Warning,
			Message:  "token as written",
			File:     "/path/to/broken.yaml",
			Position: pathindex.StartingPosition,
			Examples: rules.Examples{},
		}}, diagnostics)
	})

	t.Run("results for an unknown file fail", func(t *testing.T) {
		t.Parallel()

		sr := &mockSourceRule{
			mockRule: mockRule{id: "source-rule", severity: rules.Warning, appliesTo: []rules.MatchPattern{rules.MatchAll()}},
			validateSourcesFn: func(map[string]*rules.ValidationContext) map[string][]rules.ValidationResult {
				return map[string][]rules.ValidationResult{"/elsewhere.yaml": {{Message: "x"}}}
			},
		}
		registry := rules.NewRegistry(nil)
		require.NoError(t, registry.RegisterSyntactic(sr))
		engine, err := NewValidationEngine(registry, nil)
		require.NoError(t, err)

		_, err = engine.ValidateSources(context.Background(), map[string]*specs.RawSpec{
			"/path/to/props.yaml": {Data: []byte(sourceWithReferences)},
		})
		require.ErrorContains(t, err, "unknown file: /elsewhere.yaml")
	})
}

const secretLinesYAML = `version: rudder/v0.1
kind: properties
metadata:
  name: test-properties
spec:
  group: test-group
  password: fake-password-value
  headers:
    - from: fake-header-value
      to: X-Header
  tokens:
    - 'fake-token: value'
`

// A sensitive result's line keeps its key and loses its value, whether the
// reference resolves exactly or falls back to its nearest ancestor.
func TestValidationEngine_MasksSensitiveResults(t *testing.T) {
	t.Parallel()

	rule := &mockRule{
		id:        "secret-rule",
		severity:  rules.Error,
		appliesTo: []rules.MatchPattern{rules.MatchAll()},
		validateFn: func(*rules.ValidationContext) []rules.ValidationResult {
			return []rules.ValidationResult{
				{Reference: "/spec/group", Message: "plain"},
				{Reference: "/spec/password", Message: "secret value", Sensitive: true},
				{Reference: "/spec/headers/0/missing", Message: "secret list item", Sensitive: true},
				{Reference: "/spec/tokens/0", Message: "secret scalar list item", Sensitive: true},
			}
		},
	}
	registry := rules.NewRegistry(nil)
	require.NoError(t, registry.RegisterSyntactic(rule))
	engine, err := NewValidationEngine(registry, nil)
	require.NoError(t, err)

	diagnostics, err := engine.ValidateSyntax(context.Background(), map[string]*specs.RawSpec{
		"/path/to/props.yaml": newRawSpec(t, secretLinesYAML),
	})
	require.NoError(t, err)

	lines := make(map[string]string, len(diagnostics))
	for _, d := range diagnostics {
		lines[d.Message] = d.Position.LineText
	}
	assert.Equal(t, map[string]string{
		"plain":                   "group: test-group",
		"secret value":            "password: ***",
		"secret list item":        "- from: ***",
		"secret scalar list item": "***",
	}, lines)
}
