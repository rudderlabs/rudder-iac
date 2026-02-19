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
)

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

	assert.Equal(t, 1, mockProvider.ValidateCalledCount, "Validate should be called once")
}

func TestProject_Load_ProviderLoadSpecError(t *testing.T) {
	t.Parallel()

	mockProvider := testutils.NewMockProvider(nil, nil)
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
	assert.Contains(t, err.Error(), "provider failed to load spec from path path/to/spec.yaml")
	assert.True(t, errors.Is(err, expectedErr))
}

func TestProject_Load_ProviderValidateError(t *testing.T) {
	t.Parallel()

	mockProvider := testutils.NewMockProvider(nil, nil)
	mockLoader := &MockLoader{}

	proj := project.New(mockProvider, project.WithLoader(mockLoader))

	validSpecs := map[string]*specs.RawSpec{
		"path/to/spec.yaml": {Data: []byte("kind: Source\nversion: rudder/0.1\nmetadata:\n  name: my_source\nspec:\n  k: v")},
	}

	mockLoader.LoadFunc = func(location string) (map[string]*specs.RawSpec, error) {
		return validSpecs, nil
	}

	mockProvider.LoadLegacySpecErr = nil
	expectedErr := errors.New("provider Validate failed")
	mockProvider.ValidateErr = expectedErr

	err := proj.Load("test_dir")
	require.Error(t, err)
	assert.True(t, errors.Is(err, expectedErr))
	assert.Equal(t, 1, mockProvider.ValidateCalledCount)
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

func TestProject_LoadSpec_WithV1SpecSupport(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name                       string
		specVersion                string
		useV1SpecSupport           bool
		expectError                bool
		expectLoadSpecCalled       bool
		expectLoadLegacySpecCalled bool
		errorContains              string
	}{
		{
			name:                       "rudder/v1 spec without v1 support - returns error",
			specVersion:                "rudder/v1",
			useV1SpecSupport:           false,
			expectError:                true,
			expectLoadSpecCalled:       false,
			expectLoadLegacySpecCalled: false,
			errorContains:              "unsupported spec version: rudder/v1",
		},
		{
			name:                       "rudder/v1 spec with v1 support - calls LoadSpec",
			specVersion:                "rudder/v1",
			useV1SpecSupport:           true,
			expectError:                false,
			expectLoadSpecCalled:       true,
			expectLoadLegacySpecCalled: false,
		},
		{
			name:                       "rudder/0.1 spec without v1 support - calls LoadLegacySpec (backward compatible)",
			specVersion:                "rudder/0.1",
			useV1SpecSupport:           false,
			expectError:                false,
			expectLoadSpecCalled:       false,
			expectLoadLegacySpecCalled: true,
		},
		{
			name:                       "rudder/0.1 spec with v1 support - calls LoadLegacySpec (backward compatible)",
			specVersion:                "rudder/0.1",
			useV1SpecSupport:           true,
			expectError:                false,
			expectLoadSpecCalled:       false,
			expectLoadLegacySpecCalled: true,
		},
		{
			name:                       "unsupported version without v1 support - returns error",
			specVersion:                "rudder/v2.0",
			useV1SpecSupport:           false,
			expectError:                true,
			expectLoadSpecCalled:       false,
			expectLoadLegacySpecCalled: false,
			errorContains:              "unsupported spec version: rudder/v2.0",
		},
		{
			name:                       "unsupported version with v1 support - returns error",
			specVersion:                "rudder/v2.0",
			useV1SpecSupport:           true,
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
			mockLoader := &MockLoader{}

			var opts []project.ProjectOption
			opts = append(opts, project.WithLoader(mockLoader))
			if tc.useV1SpecSupport {
				opts = append(opts, project.WithV1SpecSupport())
			}

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

func TestProject_ValidateSpec(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name          string
		spec          *specs.Spec
		parsedSpec    *specs.ParsedSpec
		expectedError bool
		errorContains string
	}{
		{
			name: "success - all import metadata IDs match external IDs",
			spec: &specs.Spec{
				Kind: "Source",
				Metadata: map[string]any{
					"import": map[string]any{
						"workspaces": []any{
							map[string]any{
								"workspace_id": "ws-123",
								"resources": []any{
									map[string]any{
										"local_id":  "id1",
										"remote_id": "remote1",
									},
									map[string]any{
										"local_id":  "id2",
										"remote_id": "remote2",
									},
								},
							},
						},
					},
				},
			},
			parsedSpec: &specs.ParsedSpec{
				LocalIDs: []specs.LocalID{
					{ID: "id1", JSONPointerPath: "/spec/id"},
					{ID: "id2", JSONPointerPath: "/spec/id"},
				},
			},
			expectedError: false,
		},
		{
			name: "error - extra IDs in import metadata not in spec",
			spec: &specs.Spec{
				Kind: "Source",
				Metadata: map[string]any{
					"import": map[string]any{
						"workspaces": []any{
							map[string]any{
								"workspace_id": "ws-123",
								"resources": []any{
									map[string]any{
										"local_id":  "id1",
										"remote_id": "remote1",
									},
									map[string]any{
										"local_id":  "id2",
										"remote_id": "remote2",
									},
									map[string]any{
										"local_id":  "id3",
										"remote_id": "remote3",
									},
								},
							},
						},
					},
				},
			},
			parsedSpec: &specs.ParsedSpec{
				LocalIDs: []specs.LocalID{
					{ID: "id1", JSONPointerPath: "/spec/id"},
					{ID: "id2", JSONPointerPath: "/spec/id"},
				},
			},
			expectedError: true,
			errorContains: "local_id from import metadata missing in spec: id3",
		},
		{
			name: "success - missing IDs in import metadata (created instead of imported)",
			spec: &specs.Spec{
				Kind: "Source",
				Metadata: map[string]any{
					"import": map[string]any{
						"workspaces": []any{
							map[string]any{
								"workspace_id": "ws-123",
								"resources": []any{
									map[string]any{
										"local_id":  "id1",
										"remote_id": "remote1",
									},
								},
							},
						},
					},
				},
			},
			parsedSpec: &specs.ParsedSpec{
				LocalIDs: []specs.LocalID{
					{ID: "id1", JSONPointerPath: "/spec/id"},
					{ID: "id2", JSONPointerPath: "/spec/id"},
					{ID: "id3", JSONPointerPath: "/spec/id"},
				},
			},
			expectedError: false,
		},
		{
			name: "success - empty both external IDs and import metadata",
			spec: &specs.Spec{
				Kind: "Source",
				Metadata: map[string]any{
					"import": map[string]any{
						"workspaces": []any{},
					},
				},
			},
			parsedSpec: &specs.ParsedSpec{},
			expectedError: false,
		},
		{
			name: "error - invalid import metadata structure",
			spec: &specs.Spec{
				Kind: "Source",
				Metadata: map[string]any{
					"import": "invalid_string_not_object",
				},
			},
			parsedSpec: &specs.ParsedSpec{
				LocalIDs: []specs.LocalID{
					{ID: "id1", JSONPointerPath: "/spec/id"},
				},
			},
			expectedError: true,
			errorContains: "failed to decode metadata",
		},
		{
			name: "success - multiple workspaces with matching IDs",
			spec: &specs.Spec{
				Kind: "Source",
				Metadata: map[string]any{
					"import": map[string]any{
						"workspaces": []any{
							map[string]any{
								"workspace_id": "ws-123",
								"resources": []any{
									map[string]any{
										"local_id":  "id1",
										"remote_id": "remote1",
									},
									map[string]any{
										"local_id":  "id2",
										"remote_id": "remote2",
									},
								},
							},
							map[string]any{
								"workspace_id": "ws-456",
								"resources": []any{
									map[string]any{
										"local_id":  "id3",
										"remote_id": "remote3",
									},
								},
							},
						},
					},
				},
			},
			parsedSpec: &specs.ParsedSpec{
				LocalIDs: []specs.LocalID{
					{ID: "id1", JSONPointerPath: "/spec/id"},
					{ID: "id2", JSONPointerPath: "/spec/id"},
					{ID: "id3", JSONPointerPath: "/spec/id"},
				},
			},
			expectedError: false,
		},
		{
			name: "error - multiple workspaces with extra ID",
			spec: &specs.Spec{
				Kind: "Source",
				Metadata: map[string]any{
					"import": map[string]any{
						"workspaces": []any{
							map[string]any{
								"workspace_id": "ws-123",
								"resources": []any{
									map[string]any{
										"local_id":  "id1",
										"remote_id": "remote1",
									},
								},
							},
							map[string]any{
								"workspace_id": "ws-456",
								"resources": []any{
									map[string]any{
										"local_id":  "id2",
										"remote_id": "remote2",
									},
									map[string]any{
										"local_id":  "id3",
										"remote_id": "remote3",
									},
								},
							},
						},
					},
				},
			},
			parsedSpec: &specs.ParsedSpec{
				LocalIDs: []specs.LocalID{
					{ID: "id1", JSONPointerPath: "/spec/id"},
				},
			},
			expectedError: true,
			errorContains: "local_id from import metadata missing in spec: id2, id3",
		},
		{
			name: "success - no import metadata key",
			spec: &specs.Spec{
				Kind:     "Source",
				Metadata: map[string]any{},
			},
			parsedSpec:    &specs.ParsedSpec{},
			expectedError: false,
		},
	}

	for _, tc := range cases {
		tc := tc // capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := project.ValidateSpec(tc.spec, tc.parsedSpec)

			if tc.expectedError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
