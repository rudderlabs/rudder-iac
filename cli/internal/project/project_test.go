package project_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/testutils"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/varsubst"
)

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
	t.Parallel()

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
			wantErr:   "variable substitution failed",
			wantSpecs: map[string]*specs.Spec{},
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
			wantErr:   "variable substitution failed",
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
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tc.wantSpecs, proj.Specs())
		})
	}
}
