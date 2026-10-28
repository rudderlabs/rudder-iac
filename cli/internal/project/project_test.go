package project_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	provrules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/testutils"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/renderer"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/varsubst"
)

func enableImportMerge(t *testing.T) {
	t.Helper()
	prevExp, prevFlag := viper.Get("experimental"), viper.Get("flags.importMerge")
	viper.Set("experimental", true)
	viper.Set("flags.importMerge", true)
	t.Cleanup(func() {
		viper.Set("experimental", prevExp)
		viper.Set("flags.importMerge", prevFlag)
	})
}

// fixtureMatchPatterns declares the test fixture kinds (Source/Destination) so a
// MockProvider reports them as supported. Without this the gatekeeper rule
// SpecSyntaxValidRule rejects them as unknown kinds — these tests exercise
// loading/routing/substitution, not kind validation, so they need a provider
// that treats their fixture kinds as known (as a real provider would).
var fixtureMatchPatterns = []rules.MatchPattern{
	{Kind: "Source", Version: "rudder/0.1"},
	{Kind: "Source", Version: "rudder/v1"},
	{Kind: "Source", Version: "rudder/v2.0"},
	{Kind: "Destination", Version: "rudder/0.1"},
}

// mapResolver is a tiny in-memory varsubst.Resolver used to drive substitution
// from test cases without depending on env vars or files.
type mapResolver map[string]string

func (m mapResolver) Resolve(name string) (string, bool) {
	v, ok := m[name]
	return v, ok
}

// MockLoader is a mock implementation of the project.Loader interface for testing.
type MockLoader struct {
	LoadFunc func(location string) (map[string]*specs.RawSpec, error)
}

// Load calls the mock LoadFunc.
func (m *MockLoader) Load(location string) (map[string]*specs.RawSpec, error) {
	if m.LoadFunc != nil {
		return m.LoadFunc(location)
	}
	return nil, errors.New("MockLoader.LoadFunc is not set")
}

// mockConsumerProvider embeds MockProvider and implements
// provider.ImportManifestLoader, recording the workspace manifest it receives via
// the read-path broadcast.
type mockConsumerProvider struct {
	*testutils.MockProvider
	gotManifest *specs.WorkspaceImportMetadata
}

func (m *mockConsumerProvider) LoadImportManifest(manifest *specs.WorkspaceImportMetadata) error {
	m.gotManifest = manifest
	return nil
}

func TestProject_BroadcastsImportManifest(t *testing.T) {
	enableImportMerge(t)

	consumer := &mockConsumerProvider{MockProvider: testutils.NewMockProvider(nil, nil)}
	// Declare the resource kind so the gatekeeper SpecSyntaxValidRule treats
	// "Source" as known (the manifest pattern alone would reject it otherwise).
	consumer.MatchPatterns = []rules.MatchPattern{
		rules.MatchKindVersion("Source", specs.SpecVersionV1),
	}
	manifestYAML := "version: rudder/v1\n" +
		"kind: import-manifest\n" +
		"metadata:\n  name: import-manifest\n" +
		"spec:\n" +
		"  workspaces:\n" +
		"    - workspace_id: ws-a\n" +
		"      resources:\n" +
		"        - urn: event:login\n" +
		"          remote_id: rem-1\n" +
		"    - workspace_id: ws-b\n" +
		"      resources:\n" +
		"        - urn: event:logout\n" +
		"          remote_id: rem-2\n"
	resourceYAML := "kind: Source\nversion: rudder/v1\nmetadata:\n  name: my_source\nspec:\n  k: v"

	mockLoader := &MockLoader{LoadFunc: func(string) (map[string]*specs.RawSpec, error) {
		return map[string]*specs.RawSpec{
			"import-manifest.yaml": {Data: []byte(manifestYAML)},
			"source.yaml":          {Data: []byte(resourceYAML)},
		}, nil
	}}

	proj := project.New(consumer, project.WithLoader(mockLoader), project.WithWorkspaceID("ws-a"))
	require.NoError(t, proj.Load("test_dir"))

	require.NotNil(t, consumer.gotManifest, "consumer should have received the broadcast manifest")
	// Scoped to the active workspace ws-a only.
	assert.Equal(t, &specs.WorkspaceImportMetadata{
		WorkspaceID: "ws-a",
		Resources:   []specs.ImportIds{{URN: "event:login", RemoteID: "rem-1"}},
	}, consumer.gotManifest)
}

func TestNewProject_Load_Error(t *testing.T) {
	t.Parallel()

	provider := testutils.NewMockProvider(nil, nil)
	mockLoader := &MockLoader{}
	p := project.New(provider, project.WithLoader(mockLoader))

	assert.NotNil(t, p)
	mockLoader.LoadFunc = func(location string) (map[string]*specs.RawSpec, error) {
		assert.Equal(t, "test_location", location)
		return nil, errors.New("custom loader called")
	}
	err := p.Load("test_location")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "custom loader called")
	// The text renderer printed nothing, so the caller still has to.
	assert.NotErrorIs(t, err, project.ErrReported)
}

func TestProject_Load_Success(t *testing.T) {
	t.Parallel()

	mockProvider := testutils.NewMockProvider(nil, nil)
	mockProvider.MatchPatterns = fixtureMatchPatterns
	mockLoader := &MockLoader{}

	proj := project.New(mockProvider, project.WithLoader(mockLoader))

	expectedSpecs := map[string]*specs.RawSpec{
		"path/to/spec1.yaml": {Data: []byte("kind: Source\nversion: rudder/0.1\nmetadata:\n  name: abc\nspec:\n  k: v")},
		"path/to/spec2.yaml": {Data: []byte("kind: Destination\nversion: rudder/0.1\nmetadata:\n name: abc\nspec:\n  k: v")},
	}

	mockLoader.LoadFunc = func(location string) (map[string]*specs.RawSpec, error) {
		return expectedSpecs, nil
	}

	err := proj.Load("test_dir")
	require.NoError(t, err)

	assert.Equal(t, 2, len(mockProvider.LoadLegacySpecCalledWithArgs), "LoadLegacySpec should be called for each spec")
	// Order might not be guaranteed from map iteration, so check presence
	foundSpec1 := false
	foundSpec2 := false
	for _, arg := range mockProvider.LoadLegacySpecCalledWithArgs {
		if arg.Path == "path/to/spec1.yaml" && arg.Spec.Kind == "Source" {
			foundSpec1 = true
		}
		if arg.Path == "path/to/spec2.yaml" && arg.Spec.Kind == "Destination" {
			foundSpec2 = true
		}
	}
	assert.True(t, foundSpec1, "Spec1 should have been loaded")
	assert.True(t, foundSpec2, "Spec2 should have been loaded")
}

// syntaxWarningProvider registers a syntactic rule that only ever warns, so a
// spec can pass syntax validation while still carrying a syntax diagnostic.
type syntaxWarningProvider struct {
	*testutils.MockProvider
}

func (p *syntaxWarningProvider) SyntacticRules() []rules.Rule {
	return []rules.Rule{provrules.NewTypedRule(
		"test/source/syntax-warning",
		rules.Warning,
		"source specs always warn",
		rules.Examples{},
		provrules.NewPatternValidator(
			fixtureMatchPatterns,
			func(_ string, _ string, _ map[string]any, _ map[string]any) []rules.ValidationResult {
				return []rules.ValidationResult{{Reference: "/k", Message: "k could not be checked"}}
			},
		),
	)}
}

func TestProject_Load_RendersSyntaxWarningsWithoutSyntaxErrors(t *testing.T) {
	t.Parallel()

	mockProvider := testutils.NewMockProvider(nil, nil)
	mockProvider.MatchPatterns = fixtureMatchPatterns
	mockLoader := &MockLoader{LoadFunc: func(string) (map[string]*specs.RawSpec, error) {
		return map[string]*specs.RawSpec{
			"spec.yaml": {Data: []byte("kind: Source\nversion: rudder/v1\nmetadata:\n  name: abc\nspec:\n  k: v")},
		}, nil
	}}

	var out bytes.Buffer
	proj := project.New(&syntaxWarningProvider{mockProvider},
		project.WithLoader(mockLoader),
		project.WithRenderer(renderer.NewTextRenderer(&out)),
	)

	require.NoError(t, proj.Load("test_dir"))
	assert.Equal(t, `
warning[test/source/syntax-warning]: k could not be checked
  --> spec.yaml:6:3
     |
   6 | k: v
     | ^^^^

Found 0 error(s), 1 warning(s)
`, out.String())
}

func TestProject_Load_ProviderLoadSpecError(t *testing.T) {
	t.Parallel()

	mockProvider := testutils.NewMockProvider(nil, nil)
	mockProvider.MatchPatterns = fixtureMatchPatterns
	mockLoader := &MockLoader{}

	proj := project.New(mockProvider, project.WithLoader(mockLoader))

	validSpecs := map[string]*specs.RawSpec{
		"path/to/spec.yaml": {Data: []byte("kind: Source\nversion: rudder/0.1\nmetadata:\n  name: my_source\nspec:\n  k: v")},
	}

	mockLoader.LoadFunc = func(location string) (map[string]*specs.RawSpec, error) {
		return validSpecs, nil
	}

	expectedErr := errors.New("provider LoadSpec failed")
	mockProvider.LoadLegacySpecErr = expectedErr

	err := proj.Load("test_dir")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "loading spec path/to/spec.yaml")
	assert.True(t, errors.Is(err, expectedErr))
}

func TestProject_GetResourceGraph_Success(t *testing.T) {
	t.Parallel()

	mockProvider := testutils.NewMockProvider(nil, nil)
	proj := project.New(mockProvider) // Loader doesn't matter for this test

	expectedGraph := &resources.Graph{}
	mockProvider.GetResourceGraphVal = expectedGraph
	mockProvider.GetResourceGraphErr = nil

	graph, err := proj.ResourceGraph()
	require.NoError(t, err)
	assert.Same(t, expectedGraph, graph) // Check if it's the exact same instance
	assert.Equal(t, 1, mockProvider.GetResourceGraphCalledCount)
}

func TestProject_GetResourceGraph_Error(t *testing.T) {
	t.Parallel()

	mockProvider := testutils.NewMockProvider(nil, nil)
	proj := project.New(mockProvider)

	expectedErr := errors.New("GetResourceGraph failed")
	mockProvider.GetResourceGraphVal = nil
	mockProvider.GetResourceGraphErr = expectedErr

	graph, err := proj.ResourceGraph()
	require.Error(t, err)
	assert.Nil(t, graph)
	assert.True(t, errors.Is(err, expectedErr))
	assert.Equal(t, 1, mockProvider.GetResourceGraphCalledCount)
}

func TestProject_LoadSpec_VersionRouting(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name                       string
		specVersion                string
		expectError                bool
		expectLoadSpecCalled       bool
		expectLoadLegacySpecCalled bool
		errorContains              string
	}{
		{
			name:                       "rudder/v1 spec calls LoadSpec",
			specVersion:                "rudder/v1",
			expectError:                false,
			expectLoadSpecCalled:       true,
			expectLoadLegacySpecCalled: false,
		},
		{
			name:                       "rudder/0.1 spec calls LoadLegacySpec",
			specVersion:                "rudder/0.1",
			expectError:                false,
			expectLoadSpecCalled:       false,
			expectLoadLegacySpecCalled: true,
		},
		{
			name:                       "unsupported version returns error",
			specVersion:                "rudder/v2.0",
			expectError:                true,
			expectLoadSpecCalled:       false,
			expectLoadLegacySpecCalled: false,
			errorContains:              "unsupported spec version: rudder/v2.0",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockProvider := testutils.NewMockProvider(nil, nil)
			mockProvider.MatchPatterns = fixtureMatchPatterns
			mockLoader := &MockLoader{}

			opts := []project.ProjectOption{project.WithLoader(mockLoader)}

			proj := project.New(mockProvider, opts...)

			specsMap := map[string]*specs.RawSpec{
				"path/to/spec.yaml": {
					Data: fmt.Appendf(
						nil,
						"kind: Source\nversion: %s\nmetadata:\n  name: my_source\nspec:\n  k: v",
						tc.specVersion,
					)},
			}

			mockLoader.LoadFunc = func(location string) (map[string]*specs.RawSpec, error) {
				return specsMap, nil
			}

			err := proj.Load("test_dir")

			if tc.expectError {
				require.Error(t, err)
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains)
				}
			} else {
				require.NoError(t, err)
			}

			// Verify LoadSpec was called (or not)
			loadSpecCalled := len(mockProvider.LoadSpecCalledWithArgs) > 0
			assert.Equal(
				t,
				tc.expectLoadSpecCalled,
				loadSpecCalled,
				"LoadSpec called mismatch: expected %v, got %v", tc.expectLoadSpecCalled, loadSpecCalled)

			// Verify LoadLegacySpec was called (or not)
			loadLegacySpecCalled := len(mockProvider.LoadLegacySpecCalledWithArgs) > 0
			assert.Equal(
				t,
				tc.expectLoadLegacySpecCalled,
				loadLegacySpecCalled,
				"LoadLegacySpec called mismatch: expected %v, got %v", tc.expectLoadLegacySpecCalled, loadLegacySpecCalled)
		})
	}
}

func TestProject_Load_WithSubstitutor(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		substitutor varsubst.Substitutor
		rawSpecs    map[string][]byte
		wantErr     string // empty when Load should succeed
		wantMatches bool   // the rendered output holds the failure, so ErrReported applies
		wantSpecs   map[string]*specs.Spec
	}{
		{
			name:        "resolves variables in metadata",
			substitutor: varsubst.NewSubstitutor(mapResolver{"NAME": "resolved_name"}),
			rawSpecs: map[string][]byte{
				"path/to/spec.yaml": []byte("kind: Source\nversion: rudder/0.1\nmetadata:\n  name: {{ .NAME }}\nspec:\n  k: v"),
			},
			wantSpecs: map[string]*specs.Spec{
				"path/to/spec.yaml": {
					Kind:     "Source",
					Version:  "rudder/0.1",
					Metadata: map[string]any{"name": "resolved_name"},
					Spec:     map[string]any{"k": "v"},
				},
			},
		},
		{
			name:        "preserves multiline string in imported single-quoted slot",
			substitutor: varsubst.NewSubstitutor(mapResolver{"CERT": "-----BEGIN CERTIFICATE-----\nline one\nline two\n-----END CERTIFICATE-----\n"}),
			rawSpecs: map[string][]byte{
				"path/to/spec.yaml": []byte("kind: Source\nversion: rudder/0.1\nmetadata:\n  name: redis\nspec:\n  ca_certificate: '{{ .CERT }}'"),
			},
			wantSpecs: map[string]*specs.Spec{
				"path/to/spec.yaml": {
					Kind:     "Source",
					Version:  "rudder/0.1",
					Metadata: map[string]any{"name": "redis"},
					Spec: map[string]any{
						"ca_certificate": "-----BEGIN CERTIFICATE-----\nline one\nline two\n-----END CERTIFICATE-----\n",
					},
				},
			},
		},
		{
			// Bare 5432 (no surrounding quotes in the spec) parses as int after substitution.
			name:        "preserves non-string scalar type",
			substitutor: varsubst.NewSubstitutor(mapResolver{"PORT": "5432"}),
			rawSpecs: map[string][]byte{
				"path/to/spec.yaml": []byte("kind: Source\nversion: rudder/0.1\nmetadata:\n  name: db\nspec:\n  port: {{ .PORT }}"),
			},
			wantSpecs: map[string]*specs.Spec{
				"path/to/spec.yaml": {
					Kind:     "Source",
					Version:  "rudder/0.1",
					Metadata: map[string]any{"name": "db"},
					Spec:     map[string]any{"port": 5432},
				},
			},
		},
		{
			name:        "undefined variable aborts load before parsing",
			substitutor: varsubst.NewSubstitutor(mapResolver{}),
			rawSpecs: map[string][]byte{
				"path/to/spec.yaml": []byte("kind: Source\nversion: rudder/0.1\nmetadata:\n  name: {{ .MISSING }}\nspec:\n  k: v"),
			},
			wantErr:   "variable substitution failed: make sure undefined variables are defined in a variable file and passed with --var-file",
			wantSpecs: map[string]*specs.Spec{},
		},
		{
			// A var file cannot fix a malformed token, so no --var-file hint.
			name:        "invalid variable syntax aborts load without var-file hint",
			substitutor: varsubst.NewSubstitutor(mapResolver{}),
			rawSpecs: map[string][]byte{
				"path/to/spec.yaml": []byte("kind: Source\nversion: rudder/0.1\nmetadata:\n  name: \"{{ .1A }}\"\nspec:\n  k: v"),
			},
			wantErr:     "variable substitution failed",
			wantMatches: true,
			wantSpecs:   map[string]*specs.Spec{},
		},
		{
			name:        "nil substitutor leaves spec untouched",
			substitutor: nil,
			rawSpecs: map[string][]byte{
				"path/to/spec.yaml": []byte("kind: Source\nversion: rudder/0.1\nmetadata:\n  name: literal\nspec:\n  k: \"{{ .v }}\""),
			},
			wantSpecs: map[string]*specs.Spec{
				"path/to/spec.yaml": {
					Kind:     "Source",
					Version:  "rudder/0.1",
					Metadata: map[string]any{"name": "literal"},
					Spec:     map[string]any{"k": "{{ .v }}"},
				},
			},
		},
		{
			// Substitution is all-or-nothing: a single failed spec aborts the
			// pass before any spec is parsed, so no specs reach Specs().
			name:        "mixed clean and errored specs short-circuits before parsing",
			substitutor: varsubst.NewSubstitutor(mapResolver{"NAME": "clean_name"}),
			rawSpecs: map[string][]byte{
				"path/to/clean.yaml":   []byte("kind: Source\nversion: rudder/0.1\nmetadata:\n  name: {{ .NAME }}\nspec:\n  k: v"),
				"path/to/errored.yaml": []byte("kind: Source\nversion: rudder/0.1\nmetadata:\n  name: {{ .MISSING }}\nspec:\n  k: v"),
			},
			wantErr:   "variable substitution failed: make sure undefined variables are defined in a variable file and passed with --var-file",
			wantSpecs: map[string]*specs.Spec{},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockProvider := testutils.NewMockProvider(nil, nil)
			mockProvider.MatchPatterns = fixtureMatchPatterns
			mockLoader := &MockLoader{LoadFunc: func(string) (map[string]*specs.RawSpec, error) {
				raw := make(map[string]*specs.RawSpec, len(tc.rawSpecs))
				for path, data := range tc.rawSpecs {
					raw[path] = &specs.RawSpec{Data: data}
				}
				return raw, nil
			}}

			proj := project.New(mockProvider,
				project.WithLoader(mockLoader),
				project.WithSubstitutor(tc.substitutor),
			)

			err := proj.Load("test_dir")
			if tc.wantErr != "" {
				require.EqualError(t, err, tc.wantErr)
				assert.Equal(t, tc.wantMatches, errors.Is(err, project.ErrReported))
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tc.wantSpecs, proj.Specs())
		})
	}
}

// warningProvider adds a syntactic and a semantic rule, each warning on the
// fixture spec named after it, so a test can place the two phases' diagnostics
// in different files.
type warningProvider struct {
	*testutils.MockProvider
}

func warnOnSpecNamed(name string) func(string, string, map[string]any, map[string]any) []rules.ValidationResult {
	return func(_ string, _ string, metadata map[string]any, _ map[string]any) []rules.ValidationResult {
		if metadata["name"] != name {
			return nil
		}
		return []rules.ValidationResult{{Message: "check this spec"}}
	}
}

func (p *warningProvider) SyntacticRules() []rules.Rule {
	return []rules.Rule{provrules.NewTypedRule(
		"test/syntactic-warning",
		rules.Warning,
		"the syntactic fixture spec always warns",
		rules.Examples{},
		provrules.NewPatternValidator(fixtureMatchPatterns, warnOnSpecNamed("syntactic_source")),
	)}
}

func (p *warningProvider) SemanticRules() []rules.Rule {
	return []rules.Rule{provrules.NewTypedRule(
		"test/semantic-warning",
		rules.Warning,
		"the semantic fixture spec always warns",
		rules.Examples{},
		provrules.NewPatternValidator(fixtureMatchPatterns, warnOnSpecNamed("semantic_source")),
	)}
}

func fixtureSpec(name string) *specs.RawSpec {
	return &specs.RawSpec{Data: []byte("kind: Source\nversion: rudder/v1\nmetadata:\n  name: " + name + "\nspec:\n  k: v")}
}

// TestProject_Load_RendersSyntacticWarnings covers what handleValidation does
// with syntactic diagnostics that do not stop the load: they used to be dropped
// on the way to the semantic phase, and on the error paths in between. The
// two-file case also pins the ordering, since the merged set is sorted by file
// rather than by phase.
func TestProject_Load_RendersSyntacticWarnings(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name      string
		rawSpecs  map[string]*specs.RawSpec
		graphErr  error
		wantLines []string
	}{
		{
			name:      "syntactic warning alone",
			rawSpecs:  map[string]*specs.RawSpec{"b.yaml": fixtureSpec("syntactic_source")},
			wantLines: []string{"warning[test/syntactic-warning]: check this spec"},
		},
		{
			name:      "load fails after the syntax gate",
			rawSpecs:  map[string]*specs.RawSpec{"b.yaml": fixtureSpec("syntactic_source")},
			graphErr:  errors.New("graph is broken"),
			wantLines: []string{"warning[test/syntactic-warning]: check this spec"},
		},
		{
			name: "semantic file sorts ahead of the syntactic one",
			rawSpecs: map[string]*specs.RawSpec{
				"b.yaml": fixtureSpec("syntactic_source"),
				"a.yaml": fixtureSpec("semantic_source"),
			},
			wantLines: []string{
				"warning[test/semantic-warning]: check this spec",
				"warning[test/syntactic-warning]: check this spec",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockProvider := &warningProvider{MockProvider: testutils.NewMockProvider(nil, nil)}
			mockProvider.MatchPatterns = fixtureMatchPatterns
			mockProvider.GetResourceGraphErr = tc.graphErr
			mockLoader := &MockLoader{LoadFunc: func(string) (map[string]*specs.RawSpec, error) {
				return tc.rawSpecs, nil
			}}

			var buf bytes.Buffer
			proj := project.New(mockProvider, project.WithLoader(mockLoader), project.WithRenderer(renderer.NewTextRenderer(&buf)))

			err := proj.Load("test_dir")
			assert.Equal(t, tc.graphErr != nil, err != nil, "load error: %v", err)

			var rendered []string
			for _, line := range strings.Split(buf.String(), "\n") {
				if strings.Contains(line, "check this spec") {
					rendered = append(rendered, strings.TrimSpace(line))
				}
			}
			assert.Equal(t, tc.wantLines, rendered)
		})
	}
}

// sourceRuleProvider adds a rule that reads specs as written, before
// substitution, and warns on each spec that holds a variable reference.
type sourceRuleProvider struct {
	*testutils.MockProvider
}

func (p *sourceRuleProvider) SyntacticRules() []rules.Rule {
	return []rules.Rule{referenceWarningRule{}}
}

type referenceWarningRule struct{}

func (referenceWarningRule) ID() string                      { return "test/source-warning" }
func (referenceWarningRule) Severity() rules.Severity        { return rules.Warning }
func (referenceWarningRule) Description() string             { return "a spec holds a variable reference" }
func (referenceWarningRule) AppliesTo() []rules.MatchPattern { return fixtureMatchPatterns }
func (referenceWarningRule) Examples() rules.Examples        { return rules.Examples{} }
func (referenceWarningRule) Validate(*rules.ValidationContext) []rules.ValidationResult {
	return nil
}

func (referenceWarningRule) ValidateSources(specs map[string]*rules.ValidationContext) map[string][]rules.ValidationResult {
	results := make(map[string][]rules.ValidationResult)
	for path, ctx := range specs {
		if bytes.Contains(ctx.Source, []byte("{{ .")) {
			results[path] = []rules.ValidationResult{{Reference: "/metadata/name", Message: "reference as written"}}
		}
	}
	return results
}

// TestProject_Load_RendersSourceDiagnostics covers the rules that read specs
// before substitution: substitution stops the load when a variable is
// undefined, and their diagnostics must be rendered then too. The JSON
// renderer must list the same diagnostics, warnings with severity warning.
func TestProject_Load_RendersSourceDiagnostics(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name      string
		rawSpecs  map[string]string
		wantErr   string
		wantLines []string
	}{
		{
			name: "substitution succeeds",
			rawSpecs: map[string]string{
				"a.yaml": "kind: Source\nversion: rudder/v1\nmetadata:\n  name: '{{ .NAME }}'\nspec:\n  k: v",
			},
			wantLines: []string{"warning[test/source-warning]: reference as written"},
		},
		{
			name: "an undefined variable in another spec stops substitution",
			rawSpecs: map[string]string{
				"a.yaml": "kind: Source\nversion: rudder/v1\nmetadata:\n  name: '{{ .NAME }}'\nspec:\n  k: v",
				"b.yaml": "kind: Source\nversion: rudder/v1\nmetadata:\n  name: '{{ .MISSING }}'\nspec:\n  k: v",
			},
			wantErr: "variable substitution failed: make sure undefined variables are defined in a variable file and passed with --var-file",
			// b.yaml's warning sits on the key, ahead of the token's error.
			wantLines: []string{
				"warning[test/source-warning]: reference as written",
				"warning[test/source-warning]: reference as written",
				"error[project/var-substitution]: undefined variable \"MISSING\"",
			},
		},
		{
			name: "syntax errors after substitution",
			rawSpecs: map[string]string{
				"a.yaml": "kind: Unknown\nversion: rudder/v1\nmetadata:\n  name: '{{ .NAME }}'\nspec:\n  k: v",
				"b.yaml": "kind: Source\nversion: rudder/v1\nmetadata:\n  name: '{{ .NAME }}'\nspec:\n  k: v",
			},
			wantErr: "syntax validation failed",
			wantLines: []string{
				"error[project/spec-syntax-valid]: 'kind' must be one of [Source Destination]",
				"warning[test/source-warning]: reference as written",
			},
		},
	} {
		for _, useJSON := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/json=%v", tc.name, useJSON), func(t *testing.T) {
				t.Parallel()

				mockProvider := &sourceRuleProvider{MockProvider: testutils.NewMockProvider(nil, nil)}
				mockProvider.MatchPatterns = fixtureMatchPatterns
				mockLoader := &MockLoader{LoadFunc: func(string) (map[string]*specs.RawSpec, error) {
					raw := make(map[string]*specs.RawSpec, len(tc.rawSpecs))
					for path, data := range tc.rawSpecs {
						raw[path] = &specs.RawSpec{Data: []byte(data)}
					}
					return raw, nil
				}}

				var buf bytes.Buffer
				r := renderer.NewTextRenderer(&buf)
				if useJSON {
					r = renderer.NewJSONRenderer(&buf)
				}
				proj := project.New(mockProvider,
					project.WithLoader(mockLoader),
					project.WithSubstitutor(varsubst.NewSubstitutor(mapResolver{"NAME": "resolved_name"})),
					project.WithRenderer(r),
				)

				err := proj.Load("test_dir")
				if tc.wantErr != "" {
					require.EqualError(t, err, tc.wantErr)
				} else {
					require.NoError(t, err)
				}

				var rendered []string
				if useJSON {
					var doc struct {
						Diagnostics []struct {
							RuleID   string `json:"ruleId"`
							Severity string `json:"severity"`
							Message  string `json:"message"`
						} `json:"diagnostics"`
						Summary struct {
							Errors   int `json:"errors"`
							Warnings int `json:"warnings"`
						} `json:"summary"`
					}
					require.NoError(t, json.Unmarshal(buf.Bytes(), &doc))
					var errorCount, warningCount int
					for _, d := range doc.Diagnostics {
						// The text renderer's first line, so both renderers
						// share one list of expected diagnostics.
						rendered = append(rendered, fmt.Sprintf("%s[%s]: %s", d.Severity, d.RuleID, d.Message))
						switch d.Severity {
						case "error":
							errorCount++
						case "warning":
							warningCount++
						}
					}
					assert.Equal(t, errorCount, doc.Summary.Errors)
					assert.Equal(t, warningCount, doc.Summary.Warnings)
				} else {
					for _, line := range strings.Split(buf.String(), "\n") {
						if strings.HasPrefix(line, "warning[") || strings.HasPrefix(line, "error[") {
							rendered = append(rendered, line)
						}
					}
				}
				assert.Equal(t, tc.wantLines, rendered)
			})
		}
	}
}

// providerWithErrorRules adds a syntactic and a semantic rule that fail the
// fixture spec named after them, so Load stops on real engine diagnostics.
type providerWithErrorRules struct {
	*testutils.MockProvider
}

func failOnSpecNamed(name string) func(string, string, map[string]any, map[string]any) []rules.ValidationResult {
	return func(_ string, _ string, metadata map[string]any, _ map[string]any) []rules.ValidationResult {
		if metadata["name"] != name {
			return nil
		}
		return []rules.ValidationResult{{Message: "broken spec"}}
	}
}

func (p *providerWithErrorRules) SyntacticRules() []rules.Rule {
	return []rules.Rule{provrules.NewTypedRule(
		"test/syntactic-error", rules.Error, "fails the syntactic fixture spec", rules.Examples{},
		provrules.NewPatternValidator(fixtureMatchPatterns, failOnSpecNamed("syntactic_source")),
	)}
}

func (p *providerWithErrorRules) SemanticRules() []rules.Rule {
	return []rules.Rule{provrules.NewTypedRule(
		"test/semantic-error", rules.Error, "fails the semantic fixture spec", rules.Examples{},
		provrules.NewPatternValidator(fixtureMatchPatterns, failOnSpecNamed("semantic_source")),
	)}
}

// Under the JSON renderer every failed load must leave one document on stdout,
// and the returned error must match ErrReported so the caller does not
// print the failure again. An empty diagnostics array has to mean "validated
// clean", so a failure with no diagnostics of its own gets a load-failed one.
func TestProject_Load_JSONRendererRecordsEveryFailure(t *testing.T) {
	t.Parallel()

	oneSource := map[string]*specs.RawSpec{"spec.yaml": fixtureSpec("abc")}

	for _, tc := range []struct {
		name      string
		rawSpecs  map[string]*specs.RawSpec
		loadErr   error
		graphErr  error
		wantErr   string
		wantRules []string
	}{
		{
			name:      "loader failure before validation",
			loadErr:   errors.New("no such directory"),
			wantErr:   "failed to load specs using specLoader: no such directory",
			wantRules: []string{"project/load-failed"},
		},
		{
			name:      "failure after the syntax gate carries no diagnostics",
			rawSpecs:  oneSource,
			graphErr:  errors.New("event not found"),
			wantErr:   "building resource graph: event not found",
			wantRules: []string{"project/load-failed"},
		},
		{
			name:      "syntax rule error",
			rawSpecs:  map[string]*specs.RawSpec{"spec.yaml": fixtureSpec("syntactic_source")},
			wantErr:   "syntax validation failed",
			wantRules: []string{"test/syntactic-error"},
		},
		{
			name:      "semantic rule error",
			rawSpecs:  map[string]*specs.RawSpec{"spec.yaml": fixtureSpec("semantic_source")},
			wantErr:   "semantic validation failed",
			wantRules: []string{"test/semantic-error"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockProvider := &providerWithErrorRules{MockProvider: testutils.NewMockProvider(nil, nil)}
			mockProvider.MatchPatterns = fixtureMatchPatterns
			mockProvider.GetResourceGraphErr = tc.graphErr
			mockLoader := &MockLoader{LoadFunc: func(string) (map[string]*specs.RawSpec, error) {
				return tc.rawSpecs, tc.loadErr
			}}

			var out bytes.Buffer
			proj := project.New(mockProvider,
				project.WithLoader(mockLoader),
				project.WithRenderer(renderer.NewJSONRenderer(&out)),
			)

			err := proj.Load("test_dir")

			require.EqualError(t, err, tc.wantErr)
			assert.ErrorIs(t, err, project.ErrReported)

			var doc struct {
				Diagnostics []struct {
					RuleID string `json:"ruleId"`
				} `json:"diagnostics"`
			}
			require.NoError(t, json.Unmarshal(out.Bytes(), &doc))
			var got []string
			for _, d := range doc.Diagnostics {
				got = append(got, d.RuleID)
			}
			assert.Equal(t, tc.wantRules, got)
		})
	}
}

// providerWithUnregistrableRule adds a syntactic rule for a kind the provider
// does not support, so building the rule registry fails before validation runs.
type providerWithUnregistrableRule struct {
	*testutils.MockProvider
}

func (p *providerWithUnregistrableRule) SyntacticRules() []rules.Rule {
	return []rules.Rule{provrules.NewTypedRule(
		"test/unregistrable", rules.Error, "applies to a kind the provider does not support", rules.Examples{},
		provrules.NewPatternValidator(
			[]rules.MatchPattern{rules.MatchKindVersion("Unsupported", "rudder/v1")},
			failOnSpecNamed("unused"),
		),
	)}
}

// unknownFileRule is a project-wide rule that reports on a file the load does
// not hold, so syntax validation fails to run after the specs are parsed.
type unknownFileRule struct{}

func (unknownFileRule) ID() string                      { return "test/unknown-file" }
func (unknownFileRule) Severity() rules.Severity        { return rules.Error }
func (unknownFileRule) Description() string             { return "reports on a file the load does not hold" }
func (unknownFileRule) AppliesTo() []rules.MatchPattern { return fixtureMatchPatterns }
func (unknownFileRule) Examples() rules.Examples        { return rules.Examples{} }
func (unknownFileRule) Validate(*rules.ValidationContext) []rules.ValidationResult {
	return nil
}

func (unknownFileRule) ValidateSpecs(map[string]*rules.ValidationContext) map[string][]rules.ValidationResult {
	return map[string][]rules.ValidationResult{"elsewhere.yaml": {{Message: "not in the load"}}}
}

// unknownSourceFileRule is a source rule that reports on a file the load does
// not hold, so source validation fails to run before substitution.
type unknownSourceFileRule struct{ referenceWarningRule }

func (unknownSourceFileRule) ID() string { return "test/unknown-source-file" }
func (unknownSourceFileRule) ValidateSources(map[string]*rules.ValidationContext) map[string][]rules.ValidationResult {
	return map[string][]rules.ValidationResult{"elsewhere.yaml": {{Message: "not in the load"}}}
}

// providerWithFailingSourceRun adds a source rule that makes source validation
// fail to run.
type providerWithFailingSourceRun struct {
	*testutils.MockProvider
}

func (p *providerWithFailingSourceRun) SyntacticRules() []rules.Rule {
	return []rules.Rule{unknownSourceFileRule{}}
}

// providerWithFailingSyntaxRun adds a source rule, whose warning is collected
// before substitution, and a rule that makes syntax validation fail to run.
type providerWithFailingSyntaxRun struct {
	*testutils.MockProvider
}

func (p *providerWithFailingSyntaxRun) SyntacticRules() []rules.Rule {
	return []rules.Rule{referenceWarningRule{}, unknownFileRule{}}
}

// Load sets up the registry and the engine and runs the source rules before it
// parses any spec, so a failure there carries no diagnostics. Syntax validation runs after parsing,
// so when it fails to run Load renders the source and parse diagnostics it
// collected before the error. Under the JSON renderer the document also holds
// the error, and under the text renderer the caller still prints it.
func TestProject_Load_SetupFailureRendersCollectedDiagnostics(t *testing.T) {
	t.Parallel()

	newMock := func() *testutils.MockProvider {
		mock := testutils.NewMockProvider(nil, nil)
		mock.MatchPatterns = fixtureMatchPatterns
		return mock
	}

	for _, tc := range []struct {
		name        string
		newProvider func() provider.Provider
		wantErr     string
		wantRules   []string // ruleIds in the JSON document
		wantMessage string   // part of the project/load-failed message
		wantText    []string // starts of the diagnostic lines the text renderer prints
		wantSummary string   // the text renderer's summary, empty when it prints nothing
	}{
		{
			name:        "registry setup",
			newProvider: func() provider.Provider { return &providerWithUnregistrableRule{MockProvider: newMock()} },
			wantErr:     "setting up registry: registering syntactic rule test/unregistrable",
			wantRules:   []string{"project/load-failed"},
			wantMessage: "setting up registry",
		},
		{
			name:        "source validation run",
			newProvider: func() provider.Provider { return &providerWithFailingSourceRun{MockProvider: newMock()} },
			wantErr:     "source validation: source rule test/unknown-source-file returned results for unknown file: elsewhere.yaml",
			wantRules:   []string{"project/load-failed"},
			wantMessage: "source validation",
		},
		{
			name:        "syntax validation run",
			newProvider: func() provider.Provider { return &providerWithFailingSyntaxRun{MockProvider: newMock()} },
			wantErr:     "syntactic validation: project-wide validation: project rule test/unknown-file returned results for unknown file: elsewhere.yaml",
			wantRules:   []string{"test/source-warning", "project/spec-syntax-parse-valid", "project/load-failed"},
			wantMessage: "syntactic validation",
			wantText: []string{
				"warning[test/source-warning]: reference as written",
				"error[project/spec-syntax-parse-valid]: failed to parse spec from path bad.yaml",
			},
			wantSummary: "Found 1 error(s), 1 warning(s)",
		},
	} {
		for _, useJSON := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s/json=%v", tc.name, useJSON), func(t *testing.T) {
				t.Parallel()

				mockLoader := &MockLoader{LoadFunc: func(string) (map[string]*specs.RawSpec, error) {
					return map[string]*specs.RawSpec{
						"bad.yaml": {Data: []byte("kind: [")},
						"a.yaml":   {Data: []byte("kind: Source\nversion: rudder/v1\nmetadata:\n  name: '{{ .NAME }}'\nspec:\n  k: v")},
					}, nil
				}}

				var out bytes.Buffer
				r := renderer.NewTextRenderer(&out)
				if useJSON {
					r = renderer.NewJSONRenderer(&out)
				}
				// Each run gets its own provider, since the mock records the
				// specs it parses.
				proj := project.New(tc.newProvider(),
					project.WithLoader(mockLoader),
					project.WithSubstitutor(varsubst.NewSubstitutor(mapResolver{"NAME": "resolved_name"})),
					project.WithRenderer(r),
				)

				err := proj.Load("test_dir")

				require.ErrorContains(t, err, tc.wantErr)
				assert.Equal(t, useJSON, errors.Is(err, project.ErrReported))

				if useJSON {
					var doc struct {
						Diagnostics []struct {
							RuleID  string `json:"ruleId"`
							Message string `json:"message"`
						} `json:"diagnostics"`
					}
					require.NoError(t, json.Unmarshal(out.Bytes(), &doc))
					var got []string
					for _, d := range doc.Diagnostics {
						got = append(got, d.RuleID)
					}
					// require, so an empty array fails this run instead of
					// panicking on the index below and stopping every test.
					require.Equal(t, tc.wantRules, got)
					assert.Contains(t, doc.Diagnostics[len(doc.Diagnostics)-1].Message, tc.wantMessage)
					return
				}

				var lines []string
				for _, line := range strings.Split(out.String(), "\n") {
					if strings.HasPrefix(line, "warning[") || strings.HasPrefix(line, "error[") {
						lines = append(lines, line)
					}
				}
				require.Len(t, lines, len(tc.wantText))
				for i, want := range tc.wantText {
					assert.True(t, strings.HasPrefix(lines[i], want), "line %d: %q", i, lines[i])
				}
				if tc.wantSummary == "" {
					assert.Empty(t, out.String())
				} else {
					assert.Contains(t, out.String(), tc.wantSummary)
				}
				assert.NotContains(t, out.String(), tc.wantMessage)
			})
		}
	}
}
