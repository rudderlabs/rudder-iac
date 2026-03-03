package transformation_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	transformations "github.com/rudderlabs/rudder-iac/api/client/transformations"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/handlers/transformation"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

// mockTransformationStore implements the TransformationStore interface for testing
type mockTransformationStore struct {
	createCalled                      bool
	updateCalled                      bool
	deleteCalled                      bool
	listTransformationsCalled         bool
	getTransformationCalled           bool
	setTransformationExternalIDCalled bool

	createFunc                      func(ctx context.Context, req *transformations.CreateTransformationRequest, publish bool) (*transformations.Transformation, error)
	updateFunc                      func(ctx context.Context, id string, req *transformations.UpdateTransformationRequest, publish bool) (*transformations.Transformation, error)
	deleteFunc                      func(ctx context.Context, id string) error
	listTransformationsFunc         func(ctx context.Context) ([]*transformations.Transformation, error)
	getTransformationFunc           func(ctx context.Context, id string) (*transformations.Transformation, error)
	setTransformationExternalIDFunc func(ctx context.Context, id string, externalID string) error
}

func newMockTransformationStore() *mockTransformationStore {
	return &mockTransformationStore{}
}

func (m *mockTransformationStore) CreateTransformation(ctx context.Context, req *transformations.CreateTransformationRequest, publish bool) (*transformations.Transformation, error) {
	m.createCalled = true
	if m.createFunc != nil {
		return m.createFunc(ctx, req, publish)
	}
	return &transformations.Transformation{
		ID:          "trans-123",
		VersionID:   "ver-456",
		Name:        req.Name,
		Description: req.Description,
		Code:        req.Code,
		Language:    req.Language,
		ExternalID:  req.ExternalID,
		WorkspaceID: "ws-789",
	}, nil
}

func (m *mockTransformationStore) UpdateTransformation(ctx context.Context, id string, req *transformations.UpdateTransformationRequest, publish bool) (*transformations.Transformation, error) {
	m.updateCalled = true
	if m.updateFunc != nil {
		return m.updateFunc(ctx, id, req, publish)
	}
	return &transformations.Transformation{
		ID:          id,
		VersionID:   "ver-updated",
		Name:        req.Name,
		Description: req.Description,
		Code:        req.Code,
		Language:    req.Language,
		WorkspaceID: "ws-789",
	}, nil
}

func (m *mockTransformationStore) DeleteTransformation(ctx context.Context, id string) error {
	m.deleteCalled = true
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

func (m *mockTransformationStore) ListTransformations(ctx context.Context) ([]*transformations.Transformation, error) {
	m.listTransformationsCalled = true
	if m.listTransformationsFunc != nil {
		return m.listTransformationsFunc(ctx)
	}
	return []*transformations.Transformation{}, nil
}

func (m *mockTransformationStore) GetTransformation(ctx context.Context, id string) (*transformations.Transformation, error) {
	m.getTransformationCalled = true
	if m.getTransformationFunc != nil {
		return m.getTransformationFunc(ctx, id)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *mockTransformationStore) SetTransformationExternalID(ctx context.Context, id string, externalID string) error {
	m.setTransformationExternalIDCalled = true
	if m.setTransformationExternalIDFunc != nil {
		return m.setTransformationExternalIDFunc(ctx, id, externalID)
	}
	return nil
}

func (m *mockTransformationStore) CreateLibrary(ctx context.Context, req *transformations.CreateLibraryRequest, publish bool) (*transformations.TransformationLibrary, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockTransformationStore) UpdateLibrary(ctx context.Context, id string, req *transformations.UpdateLibraryRequest, publish bool) (*transformations.TransformationLibrary, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockTransformationStore) GetLibrary(ctx context.Context, id string) (*transformations.TransformationLibrary, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockTransformationStore) ListLibraries(ctx context.Context) ([]*transformations.TransformationLibrary, error) {
	return []*transformations.TransformationLibrary{}, nil
}

func (m *mockTransformationStore) DeleteLibrary(ctx context.Context, id string) error {
	return fmt.Errorf("not implemented")
}

func (m *mockTransformationStore) SetLibraryExternalID(ctx context.Context, id string, externalID string) error {
	return fmt.Errorf("not implemented")
}

func (m *mockTransformationStore) BatchPublish(ctx context.Context, req *transformations.BatchPublishRequest) (*transformations.BatchPublishResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockTransformationStore) BatchTest(ctx context.Context, req *transformations.BatchTestRequest) (*transformations.BatchTestResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func TestHandlerMetadata(t *testing.T) {
	t.Parallel()

	mockStore := newMockTransformationStore()
	handler := transformation.NewHandler(mockStore)

	metadata := handler.Impl.Metadata()

	assert.Equal(t, "transformation", metadata.ResourceType)
	assert.Equal(t, "transformation", metadata.SpecKind)
	assert.Equal(t, "transformations", metadata.SpecMetadataName)
}

func TestNewSpec(t *testing.T) {
	t.Parallel()

	mockStore := newMockTransformationStore()
	handler := transformation.NewHandler(mockStore)

	spec := handler.Impl.NewSpec()

	require.NotNil(t, spec)
	assert.IsType(t, &model.TransformationSpec{}, spec)
}

func TestValidateSpec(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		spec          *model.TransformationSpec
		expectedError bool
		errorContains string
	}{
		{
			name: "valid spec with inline code",
			spec: &model.TransformationSpec{
				ID:          "test-trans",
				Name:        "Test Transformation",
				Description: "Test description",
				Language:    "javascript",
				Code:        "export function transformEvent(event, metadata) { return event; }",
			},
			expectedError: false,
		},
		{
			name: "valid spec with file reference",
			spec: &model.TransformationSpec{
				ID:          "test-trans",
				Name:        "Test Transformation",
				Description: "Test description",
				Language:    "python",
				File:        "transformation.py",
			},
			expectedError: false,
		},
		{
			name: "valid spec with python language",
			spec: &model.TransformationSpec{
				ID:       "test-trans",
				Name:     "Test Transformation",
				Language: "python",
				Code:     "def transform(event):\n    return event",
			},
			expectedError: false,
		},
		{
			name: "valid spec with empty language - validation deferred to resource",
			spec: &model.TransformationSpec{
				ID:       "test-trans",
				Name:     "Test Transformation",
				Language: "",
				Code:     "export function transformEvent(event, metadata) { return event; }",
			},
			expectedError: true,
			errorContains: "language is required",
		},
		{
			name: "missing id",
			spec: &model.TransformationSpec{
				Name:     "Test Transformation",
				Language: "javascript",
				Code:     "export function transformEvent(event, metadata) { return event; }",
			},
			expectedError: true,
			errorContains: "id is required",
		},
		{
			name: "missing name",
			spec: &model.TransformationSpec{
				ID:       "test-trans",
				Language: "javascript",
				Code:     "export function transformEvent(event, metadata) { return event; }",
			},
			expectedError: true,
			errorContains: "name is required",
		},
		{
			name: "both code and file specified",
			spec: &model.TransformationSpec{
				ID:       "test-trans",
				Name:     "Test Transformation",
				Language: "javascript",
				Code:     "export function transformEvent(event, metadata) { return event; }",
				File:     "transform.js",
			},
			expectedError: true,
			errorContains: "code and file are mutually exclusive",
		},
		{
			name: "neither code nor file specified",
			spec: &model.TransformationSpec{
				ID:       "test-trans",
				Name:     "Test Transformation",
				Language: "javascript",
			},
			expectedError: true,
			errorContains: "either code or file must be specified",
		},
		{
			name: "valid spec with tests containing names",
			spec: &model.TransformationSpec{
				ID:       "test-trans",
				Name:     "Test Transformation",
				Language: "javascript",
				Code:     "export function transformEvent(event, metadata) { return event; }",
				Tests: []specs.TransformationTest{
					{Name: "Suite 1"},
					{Name: "Suite 2"},
				},
			},
			expectedError: false,
		},
		{
			name: "multiple tests with one missing name",
			spec: &model.TransformationSpec{
				ID:       "test-trans",
				Name:     "Test Transformation",
				Language: "javascript",
				Code:     "export function transformEvent(event, metadata) { return event; }",
				Tests: []specs.TransformationTest{
					{Name: "Suite 1"},
					{Name: ""},
					{Name: "Suite 3"},
				},
			},
			expectedError: true,
			errorContains: "name is required for each spec tests block",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockStore := newMockTransformationStore()
			handler := transformation.NewHandler(mockStore)

			err := handler.Impl.ValidateSpec(tc.spec)

			if tc.expectedError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestExtractResourcesFromSpec(t *testing.T) {
	t.Parallel()

	t.Run("with inline code", func(t *testing.T) {
		t.Parallel()

		mockStore := newMockTransformationStore()
		handler := transformation.NewHandler(mockStore)

		spec := &model.TransformationSpec{
			ID:          "test-trans",
			Name:        "Test Transformation",
			Description: "Test description",
			Language:    "javascript",
			Code:        "export function transformEvent(event, metadata) { return event; }",
		}

		resources, err := handler.Impl.ExtractResourcesFromSpec("/path/to/spec.yaml", spec)

		require.NoError(t, err)
		require.NotNil(t, resources)
		require.Len(t, resources, 1)

		resource := resources["test-trans"]
		require.NotNil(t, resource)
		assert.Equal(t, "test-trans", resource.ID)
		assert.Equal(t, "Test Transformation", resource.Name)
		assert.Equal(t, "Test description", resource.Description)
		assert.Equal(t, "javascript", resource.Language)
		assert.Equal(t, "export function transformEvent(event, metadata) { return event; }", resource.Code)
	})

	t.Run("with file reference - absolute path", func(t *testing.T) {
		t.Parallel()

		// Create a temporary file with code
		tmpDir := t.TempDir()
		codeFile := filepath.Join(tmpDir, "transform.js")
		codeContent := "export function transformEvent(event, metadata) { return event; }"
		err := os.WriteFile(codeFile, []byte(codeContent), 0644)
		require.NoError(t, err)

		mockStore := newMockTransformationStore()
		handler := transformation.NewHandler(mockStore)

		spec := &model.TransformationSpec{
			ID:          "test-trans",
			Name:        "Test Transformation",
			Description: "Test description",
			Language:    "javascript",
			File:        codeFile,
		}

		resources, err := handler.Impl.ExtractResourcesFromSpec("/path/to/spec.yaml", spec)

		require.NoError(t, err)
		require.NotNil(t, resources)
		require.Len(t, resources, 1)

		resource := resources["test-trans"]
		require.NotNil(t, resource)
		assert.Equal(t, codeContent, resource.Code)
	})

	t.Run("with file reference - relative path", func(t *testing.T) {
		t.Parallel()

		// Create a temporary directory structure
		tmpDir := t.TempDir()
		specDir := filepath.Join(tmpDir, "specs")
		err := os.MkdirAll(specDir, 0755)
		require.NoError(t, err)

		codeFile := filepath.Join(specDir, "transform.js")
		codeContent := "export function transformEvent(event, metadata) { return event; }"
		err = os.WriteFile(codeFile, []byte(codeContent), 0644)
		require.NoError(t, err)

		mockStore := newMockTransformationStore()
		handler := transformation.NewHandler(mockStore)

		spec := &model.TransformationSpec{
			ID:       "test-trans",
			Name:     "Test Transformation",
			Language: "javascript",
			File:     "transform.js",
		}

		specPath := filepath.Join(specDir, "spec.yaml")
		resources, err := handler.Impl.ExtractResourcesFromSpec(specPath, spec)

		require.NoError(t, err)
		require.NotNil(t, resources)
		require.Len(t, resources, 1)

		resource := resources["test-trans"]
		require.NotNil(t, resource)
		assert.Equal(t, codeContent, resource.Code)
	})

	t.Run("with file reference - file not found", func(t *testing.T) {
		t.Parallel()

		mockStore := newMockTransformationStore()
		handler := transformation.NewHandler(mockStore)

		spec := &model.TransformationSpec{
			ID:       "test-trans",
			Name:     "Test Transformation",
			Language: "javascript",
			File:     "/nonexistent/file.js",
		}

		resources, err := handler.Impl.ExtractResourcesFromSpec("/path/to/spec.yaml", spec)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "reading code file")
		assert.Nil(t, resources)
	})

	t.Run("with tests - SpecDir enrichment and defaults", func(t *testing.T) {
		t.Parallel()

		mockStore := newMockTransformationStore()
		handler := transformation.NewHandler(mockStore)

		spec := &model.TransformationSpec{
			ID:       "test-trans",
			Name:     "Test Transformation",
			Language: "javascript",
			Code:     "export function transformEvent(event, metadata) { return event; }",
			Tests: []specs.TransformationTest{
				{
					Name: "Test with defaults",
					// Input and Output not specified - should get defaults
				},
				{
					Name:   "Test with custom paths",
					Input:  "./custom-input",
					Output: "./custom-output",
				},
			},
		}

		specPath := "/path/to/transformations/spec.yaml"
		resources, err := handler.Impl.ExtractResourcesFromSpec(specPath, spec)

		require.NoError(t, err)
		require.NotNil(t, resources)
		require.Len(t, resources, 1)

		resource := resources["test-trans"]
		require.NotNil(t, resource)
		require.Len(t, resource.Tests, 2)

		// Verify SpecDir is populated correctly
		expectedSpecDir := "/path/to/transformations"
		assert.Equal(t, expectedSpecDir, resource.Tests[0].SpecDir)
		assert.Equal(t, expectedSpecDir, resource.Tests[1].SpecDir)

		// Verify defaults are applied for first test
		assert.Equal(t, transformation.DefaultInputPath, resource.Tests[0].Input)
		assert.Equal(t, transformation.DefaultOutputPath, resource.Tests[0].Output)

		// Verify custom values are preserved for second test
		assert.Equal(t, "./custom-input", resource.Tests[1].Input)
		assert.Equal(t, "./custom-output", resource.Tests[1].Output)
	})
}

func TestValidateResource(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		resource      *model.TransformationResource
		expectedError bool
		errorContains string
	}{
		{
			name: "valid javascript resource",
			resource: &model.TransformationResource{
				ID:          "test-trans",
				Name:        "Test Transformation",
				Description: "Test description",
				Language:    "javascript",
				Code:        "export function transformEvent(event, metadata) { return event; }",
			},
			expectedError: false,
		},
		{
			name: "valid python resource - no syntax validation",
			resource: &model.TransformationResource{
				ID:       "test-trans",
				Name:     "Test Transformation",
				Language: "python",
				Code:     "def transform(event):\n    return event",
			},
			expectedError: false,
		},
		{
			name: "missing code",
			resource: &model.TransformationResource{
				ID:       "test-trans",
				Name:     "Test Transformation",
				Language: "javascript",
				Code:     "",
			},
			expectedError: true,
			errorContains: "code is required",
		},
		{
			name: "invalid javascript syntax",
			resource: &model.TransformationResource{
				ID:       "test-trans",
				Name:     "Test Transformation",
				Language: "javascript",
				Code:     "export function transformEvent(event, metadata) { return event;",
			},
			expectedError: true,
			errorContains: "validating code syntax",
		},
		{
			name: "invalid language",
			resource: &model.TransformationResource{
				ID:       "test-trans",
				Name:     "Test Transformation",
				Language: "golang",
				Code:     "package main",
			},
			expectedError: true,
			errorContains: "language must be javascript or python",
		},
		{
			name: "valid test names with allowed characters",
			resource: &model.TransformationResource{
				ID:       "test-trans",
				Name:     "Test Transformation",
				Language: "javascript",
				Code:     "export function transformEvent(event, metadata) { return event; }",
				Tests: []specs.TransformationTest{
					{Name: "Suite 1"},
					{Name: "Suite-1"},
					{Name: "Suite_1"},
					{Name: "Suite/1"},
				},
			},
			expectedError: false,
		},
		{
			name: "empty test name",
			resource: &model.TransformationResource{
				ID:       "test-trans",
				Name:     "Test Transformation",
				Language: "javascript",
				Code:     "export function transformEvent(event, metadata) { return event; }",
				Tests: []specs.TransformationTest{
					{Name: ""},
				},
			},
			expectedError: true,
			errorContains: "test name is required",
		},
		{
			name: "whitespace-only test name",
			resource: &model.TransformationResource{
				ID:       "test-trans",
				Name:     "Test Transformation",
				Language: "javascript",
				Code:     "export function transformEvent(event, metadata) { return event; }",
				Tests: []specs.TransformationTest{
					{Name: "    "},
				},
			},
			expectedError: true,
			errorContains: "test name is required",
		},
		{
			name: "invalid test name character",
			resource: &model.TransformationResource{
				ID:       "test-trans",
				Name:     "Test Transformation",
				Language: "javascript",
				Code:     "export function transformEvent(event, metadata) { return event; }",
				Tests: []specs.TransformationTest{
					{Name: "Suite@1"},
				},
			},
			expectedError: true,
			errorContains: "invalid test name",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockStore := newMockTransformationStore()
			handler := transformation.NewHandler(mockStore)

			graph := resources.NewGraph()

			err := handler.Impl.ValidateResource(tc.resource, graph)

			if tc.expectedError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLoadRemoteResources(t *testing.T) {
	t.Parallel()

	t.Run("success with external IDs", func(t *testing.T) {
		t.Parallel()

		mockStore := newMockTransformationStore()
		mockStore.listTransformationsFunc = func(ctx context.Context) ([]*transformations.Transformation, error) {
			return []*transformations.Transformation{
				{
					ID:          "trans-1",
					VersionID:   "ver-1",
					Name:        "Transformation 1",
					Description: "Description 1",
					Code:        "export function transformEvent(event, metadata) { return event; }",
					Language:    "javascript",
					ExternalID:  "ext-1",
					WorkspaceID: "ws-1",
				},
				{
					ID:          "trans-2",
					VersionID:   "ver-2",
					Name:        "Transformation 2",
					Description: "Description 2",
					Code:        "def transform(event):\n    return event",
					Language:    "python",
					ExternalID:  "ext-2",
					WorkspaceID: "ws-1",
				},
			}, nil
		}

		handler := transformation.NewHandler(mockStore)

		remotes, err := handler.Impl.LoadRemoteResources(context.Background())

		require.NoError(t, err)
		require.Len(t, remotes, 2)
		assert.True(t, mockStore.listTransformationsCalled)

		assert.Equal(t, "trans-1", remotes[0].ID)
		assert.Equal(t, "ext-1", remotes[0].ExternalID)
		assert.Equal(t, "Transformation 1", remotes[0].Name)

		assert.Equal(t, "trans-2", remotes[1].ID)
		assert.Equal(t, "ext-2", remotes[1].ExternalID)
		assert.Equal(t, "Transformation 2", remotes[1].Name)
	})

	t.Run("filters out resources without external IDs", func(t *testing.T) {
		t.Parallel()

		mockStore := newMockTransformationStore()
		mockStore.listTransformationsFunc = func(ctx context.Context) ([]*transformations.Transformation, error) {
			return []*transformations.Transformation{
				{
					ID:          "trans-1",
					VersionID:   "ver-1",
					Name:        "Transformation 1",
					Code:        "export function transformEvent(event, metadata) { return event; }",
					Language:    "javascript",
					ExternalID:  "ext-1",
					WorkspaceID: "ws-1",
				},
				{
					ID:          "trans-2",
					VersionID:   "ver-2",
					Name:        "Transformation 2",
					Code:        "export function transformEvent(event, metadata) { return event; }",
					Language:    "javascript",
					ExternalID:  "", // No external ID
					WorkspaceID: "ws-1",
				},
			}, nil
		}

		handler := transformation.NewHandler(mockStore)

		remotes, err := handler.Impl.LoadRemoteResources(context.Background())

		require.NoError(t, err)
		require.Len(t, remotes, 1)
		assert.Equal(t, "ext-1", remotes[0].ExternalID)
	})

	t.Run("empty list", func(t *testing.T) {
		t.Parallel()

		mockStore := newMockTransformationStore()
		mockStore.listTransformationsFunc = func(ctx context.Context) ([]*transformations.Transformation, error) {
			return []*transformations.Transformation{}, nil
		}

		handler := transformation.NewHandler(mockStore)

		remotes, err := handler.Impl.LoadRemoteResources(context.Background())

		require.NoError(t, err)
		require.Len(t, remotes, 0)
	})

	t.Run("API error", func(t *testing.T) {
		t.Parallel()

		mockStore := newMockTransformationStore()
		mockStore.listTransformationsFunc = func(ctx context.Context) ([]*transformations.Transformation, error) {
			return nil, fmt.Errorf("API error")
		}

		handler := transformation.NewHandler(mockStore)

		remotes, err := handler.Impl.LoadRemoteResources(context.Background())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "listing transformations")
		assert.Nil(t, remotes)
	})
}

func TestLoadImportableResources(t *testing.T) {
	t.Parallel()

	t.Run("returns only resources without external IDs", func(t *testing.T) {
		t.Parallel()

		mockStore := newMockTransformationStore()
		mockStore.listTransformationsFunc = func(ctx context.Context) ([]*transformations.Transformation, error) {
			return []*transformations.Transformation{
				{
					ID:          "trans-1",
					VersionID:   "ver-1",
					Name:        "Transformation 1",
					Code:        "export function transformEvent(event, metadata) { return event; }",
					Language:    "javascript",
					ExternalID:  "", // No external ID - importable
					WorkspaceID: "ws-1",
				},
				{
					ID:          "trans-2",
					VersionID:   "ver-2",
					Name:        "Transformation 2",
					Code:        "export function transformEvent(event, metadata) { return event; }",
					Language:    "javascript",
					ExternalID:  "ext-2", // Has external ID - managed
					WorkspaceID: "ws-1",
				},
			}, nil
		}

		handler := transformation.NewHandler(mockStore)

		importables, err := handler.Impl.LoadImportableResources(context.Background())

		require.NoError(t, err)
		require.Len(t, importables, 1)
		assert.Equal(t, "trans-1", importables[0].ID)
		assert.Equal(t, "", importables[0].ExternalID)
	})

	t.Run("empty list", func(t *testing.T) {
		t.Parallel()

		mockStore := newMockTransformationStore()
		mockStore.listTransformationsFunc = func(ctx context.Context) ([]*transformations.Transformation, error) {
			return []*transformations.Transformation{}, nil
		}

		handler := transformation.NewHandler(mockStore)

		importables, err := handler.Impl.LoadImportableResources(context.Background())

		require.NoError(t, err)
		require.Len(t, importables, 0)
	})

	t.Run("all resources have external IDs", func(t *testing.T) {
		t.Parallel()

		mockStore := newMockTransformationStore()
		mockStore.listTransformationsFunc = func(ctx context.Context) ([]*transformations.Transformation, error) {
			return []*transformations.Transformation{
				{
					ID:          "trans-1",
					VersionID:   "ver-1",
					Name:        "Transformation 1",
					Code:        "export function transformEvent(event, metadata) { return event; }",
					Language:    "javascript",
					ExternalID:  "ext-1",
					WorkspaceID: "ws-1",
				},
			}, nil
		}

		handler := transformation.NewHandler(mockStore)

		importables, err := handler.Impl.LoadImportableResources(context.Background())

		require.NoError(t, err)
		require.Len(t, importables, 0)
	})

	t.Run("API error", func(t *testing.T) {
		t.Parallel()

		mockStore := newMockTransformationStore()
		mockStore.listTransformationsFunc = func(ctx context.Context) ([]*transformations.Transformation, error) {
			return nil, fmt.Errorf("API error")
		}

		handler := transformation.NewHandler(mockStore)

		importables, err := handler.Impl.LoadImportableResources(context.Background())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "listing transformations")
		assert.Nil(t, importables)
	})
}

func TestMapRemoteToState(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		remote := &model.RemoteTransformation{
			Transformation: &transformations.Transformation{
				ID:          "trans-123",
				VersionID:   "ver-456",
				Name:        "Test Transformation",
				Description: "Test description",
				Code:        "export function transformEvent(event, metadata) { return event; }",
				Language:    "javascript",
				ExternalID:  "ext-123",
				WorkspaceID: "ws-789",
			},
		}

		mockStore := newMockTransformationStore()
		handler := transformation.NewHandler(mockStore)

		resource, state, err := handler.Impl.MapRemoteToState(remote, nil)

		require.NoError(t, err)
		require.NotNil(t, resource)
		require.NotNil(t, state)

		assert.Equal(t, "ext-123", resource.ID)
		assert.Equal(t, "Test Transformation", resource.Name)
		assert.Equal(t, "Test description", resource.Description)
		assert.Equal(t, "javascript", resource.Language)
		assert.Equal(t, "export function transformEvent(event, metadata) { return event; }", resource.Code)

		assert.Equal(t, "trans-123", state.ID)
		assert.Equal(t, "ver-456", state.VersionID)
		assert.False(t, state.Modified)
	})
}

func TestCreate(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		mockStore := newMockTransformationStore()
		handler := transformation.NewHandler(mockStore)

		resource := &model.TransformationResource{
			ID:          "ext-123",
			Name:        "Test Transformation",
			Description: "Test description",
			Language:    "javascript",
			Code:        "export function transformEvent(event, metadata) { return event; }",
		}

		state, err := handler.Impl.Create(context.Background(), resource)

		require.NoError(t, err)
		require.NotNil(t, state)
		assert.True(t, mockStore.createCalled)

		assert.Equal(t, "trans-123", state.ID)
		assert.Equal(t, "ver-456", state.VersionID)
		assert.True(t, state.Modified)
	})

	t.Run("API error", func(t *testing.T) {
		t.Parallel()

		mockStore := newMockTransformationStore()
		mockStore.createFunc = func(ctx context.Context, req *transformations.CreateTransformationRequest, publish bool) (*transformations.Transformation, error) {
			return nil, fmt.Errorf("API error")
		}

		handler := transformation.NewHandler(mockStore)

		resource := &model.TransformationResource{
			ID:       "ext-123",
			Name:     "Test Transformation",
			Language: "javascript",
			Code:     "export function transformEvent(event, metadata) { return event; }",
		}

		state, err := handler.Impl.Create(context.Background(), resource)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "creating transformation")
		assert.Nil(t, state)
	})

	t.Run("passes correct parameters", func(t *testing.T) {
		t.Parallel()

		var capturedReq *transformations.CreateTransformationRequest
		var capturedPublish bool

		mockStore := newMockTransformationStore()
		mockStore.createFunc = func(ctx context.Context, req *transformations.CreateTransformationRequest, publish bool) (*transformations.Transformation, error) {
			capturedReq = req
			capturedPublish = publish
			return &transformations.Transformation{
				ID:        "trans-123",
				VersionID: "ver-456",
			}, nil
		}

		handler := transformation.NewHandler(mockStore)

		resource := &model.TransformationResource{
			ID:          "ext-123",
			Name:        "Test Transformation",
			Description: "Test description",
			Language:    "javascript",
			Code:        "export function transformEvent(event, metadata) { return event; }",
		}

		_, err := handler.Impl.Create(context.Background(), resource)

		require.NoError(t, err)
		require.NotNil(t, capturedReq)

		assert.Equal(t, "Test Transformation", capturedReq.Name)
		assert.Equal(t, "Test description", capturedReq.Description)
		assert.Equal(t, "export function transformEvent(event, metadata) { return event; }", capturedReq.Code)
		assert.Equal(t, "javascript", capturedReq.Language)
		assert.Equal(t, "ext-123", capturedReq.ExternalID)
		assert.False(t, capturedPublish)
	})
}

func TestUpdate(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		mockStore := newMockTransformationStore()
		handler := transformation.NewHandler(mockStore)

		newData := &model.TransformationResource{
			ID:          "ext-123",
			Name:        "Updated Transformation",
			Description: "Updated description",
			Language:    "javascript",
			Code:        "export function transformEvent(event, metadata) { return event; }",
		}

		oldData := &model.TransformationResource{
			ID:          "ext-123",
			Name:        "Old Transformation",
			Description: "Old description",
			Language:    "javascript",
			Code:        "export function transformEvent(event, metadata) { return event; }",
		}

		oldState := &model.TransformationState{
			ID:        "trans-123",
			VersionID: "ver-456",
		}

		state, err := handler.Impl.Update(context.Background(), newData, oldData, oldState)

		require.NoError(t, err)
		require.NotNil(t, state)
		assert.True(t, mockStore.updateCalled)

		assert.Equal(t, "trans-123", state.ID)
		assert.Equal(t, "ver-updated", state.VersionID)
		assert.True(t, state.Modified)
	})

	t.Run("API error", func(t *testing.T) {
		t.Parallel()

		mockStore := newMockTransformationStore()
		mockStore.updateFunc = func(ctx context.Context, id string, req *transformations.UpdateTransformationRequest, publish bool) (*transformations.Transformation, error) {
			return nil, fmt.Errorf("API error")
		}

		handler := transformation.NewHandler(mockStore)

		newData := &model.TransformationResource{
			ID:       "ext-123",
			Name:     "Updated Transformation",
			Language: "javascript",
			Code:     "export function transformEvent(event, metadata) { return event; }",
		}

		oldData := &model.TransformationResource{
			ID:       "ext-123",
			Name:     "Old Transformation",
			Language: "javascript",
			Code:     "export function transformEvent(event, metadata) { return event; }",
		}

		oldState := &model.TransformationState{
			ID:        "trans-123",
			VersionID: "ver-456",
		}

		state, err := handler.Impl.Update(context.Background(), newData, oldData, oldState)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "updating transformation")
		assert.Nil(t, state)
	})

	t.Run("passes correct parameters", func(t *testing.T) {
		t.Parallel()

		var capturedID string
		var capturedReq *transformations.UpdateTransformationRequest
		var capturedPublish bool

		mockStore := newMockTransformationStore()
		mockStore.updateFunc = func(ctx context.Context, id string, req *transformations.UpdateTransformationRequest, publish bool) (*transformations.Transformation, error) {
			capturedID = id
			capturedReq = req
			capturedPublish = publish
			return &transformations.Transformation{
				ID:        id,
				VersionID: "ver-updated",
			}, nil
		}

		handler := transformation.NewHandler(mockStore)

		newData := &model.TransformationResource{
			ID:          "ext-123",
			Name:        "Updated Transformation",
			Description: "Updated description",
			Language:    "javascript",
			Code:        "export function transformEvent(event, metadata) { return event; }",
		}

		oldData := &model.TransformationResource{
			ID:       "ext-123",
			Name:     "Old Transformation",
			Language: "javascript",
			Code:     "old code",
		}

		oldState := &model.TransformationState{
			ID:        "trans-123",
			VersionID: "ver-456",
		}

		_, err := handler.Impl.Update(context.Background(), newData, oldData, oldState)

		require.NoError(t, err)
		require.NotNil(t, capturedReq)

		assert.Equal(t, "trans-123", capturedID)
		assert.Equal(t, "Updated Transformation", capturedReq.Name)
		assert.Equal(t, "Updated description", capturedReq.Description)
		assert.Equal(t, "export function transformEvent(event, metadata) { return event; }", capturedReq.Code)
		assert.Equal(t, "javascript", capturedReq.Language)
		assert.False(t, capturedPublish)
	})
}

func TestDelete(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		mockStore := newMockTransformationStore()
		handler := transformation.NewHandler(mockStore)

		oldData := &model.TransformationResource{
			ID:       "ext-123",
			Name:     "Test Transformation",
			Language: "javascript",
			Code:     "export function transformEvent(event, metadata) { return event; }",
		}

		oldState := &model.TransformationState{
			ID:        "trans-123",
			VersionID: "ver-456",
		}

		err := handler.Impl.Delete(context.Background(), "ext-123", oldData, oldState)

		require.NoError(t, err)
		assert.True(t, mockStore.deleteCalled)
	})

	t.Run("API error", func(t *testing.T) {
		t.Parallel()

		mockStore := newMockTransformationStore()
		mockStore.deleteFunc = func(ctx context.Context, id string) error {
			return fmt.Errorf("API error")
		}

		handler := transformation.NewHandler(mockStore)

		oldData := &model.TransformationResource{
			ID:   "ext-123",
			Name: "Test Transformation",
		}

		oldState := &model.TransformationState{
			ID:        "trans-123",
			VersionID: "ver-456",
		}

		err := handler.Impl.Delete(context.Background(), "ext-123", oldData, oldState)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "deleting transformation")
	})

	t.Run("uses state ID not external ID", func(t *testing.T) {
		t.Parallel()

		var capturedID string

		mockStore := newMockTransformationStore()
		mockStore.deleteFunc = func(ctx context.Context, id string) error {
			capturedID = id
			return nil
		}

		handler := transformation.NewHandler(mockStore)

		oldData := &model.TransformationResource{
			ID:   "ext-123",
			Name: "Test Transformation",
		}

		oldState := &model.TransformationState{
			ID:        "trans-123",
			VersionID: "ver-456",
		}

		err := handler.Impl.Delete(context.Background(), "ext-123", oldData, oldState)

		require.NoError(t, err)
		assert.Equal(t, "trans-123", capturedID)
	})
}

func TestImport(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		mockStore := newMockTransformationStore()
		mockStore.getTransformationFunc = func(ctx context.Context, id string) (*transformations.Transformation, error) {
			return &transformations.Transformation{
				ID:          "trans-123",
				VersionID:   "ver-456",
				Name:        "Remote Transformation",
				Description: "Remote description",
				Code:        "export function transformEvent(event, metadata) { return event; }",
				Language:    "javascript",
				WorkspaceID: "ws-789",
			}, nil
		}

		handler := transformation.NewHandler(mockStore)

		resource := &model.TransformationResource{
			ID:          "ext-new-123",
			Name:        "Local Transformation",
			Description: "Local description",
			Code:        "export function transformEvent(event, metadata) { return event; }",
			Language:    "javascript",
		}

		state, err := handler.Impl.Import(context.Background(), resource, "trans-123")

		require.NoError(t, err)
		require.NotNil(t, state)
		assert.Equal(t, "trans-123", state.ID)
		assert.Equal(t, "ver-updated", state.VersionID)
		assert.True(t, state.Modified)
		assert.True(t, mockStore.getTransformationCalled)
		assert.True(t, mockStore.setTransformationExternalIDCalled)
		assert.True(t, mockStore.updateCalled)
	})

	t.Run("GetTransformation API error", func(t *testing.T) {
		t.Parallel()

		mockStore := newMockTransformationStore()
		mockStore.getTransformationFunc = func(ctx context.Context, id string) (*transformations.Transformation, error) {
			return nil, fmt.Errorf("API error")
		}

		handler := transformation.NewHandler(mockStore)

		resource := &model.TransformationResource{
			ID:   "ext-new-123",
			Name: "Local Transformation",
		}

		state, err := handler.Impl.Import(context.Background(), resource, "trans-123")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "getting transformation trans-123")
		assert.Nil(t, state)
	})

	t.Run("SetTransformationExternalID API error", func(t *testing.T) {
		t.Parallel()

		mockStore := newMockTransformationStore()
		mockStore.getTransformationFunc = func(ctx context.Context, id string) (*transformations.Transformation, error) {
			return &transformations.Transformation{
				ID:        "trans-123",
				VersionID: "ver-456",
			}, nil
		}
		mockStore.setTransformationExternalIDFunc = func(ctx context.Context, id string, externalID string) error {
			return fmt.Errorf("API error")
		}

		handler := transformation.NewHandler(mockStore)

		resource := &model.TransformationResource{
			ID:   "ext-new-123",
			Name: "Local Transformation",
		}

		state, err := handler.Impl.Import(context.Background(), resource, "trans-123")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "setting transformation external ID")
		assert.Nil(t, state)
	})
}

// Mock implementations for namer and resolver
type mockNamer struct {
	nameFunc func(scope namer.ScopeName) (string, error)
}

func (m *mockNamer) Name(scope namer.ScopeName) (string, error) {
	if m.nameFunc != nil {
		return m.nameFunc(scope)
	}
	return scope.Name, nil
}

func (m *mockNamer) Load(names []namer.ScopeName) error {
	return nil
}

type mockResolver struct{}

func (m *mockResolver) ResolveReference(urn string) (any, error) {
	return nil, nil
}

func (m *mockResolver) ResolveToReference(entityType string, remoteID string) (string, error) {
	return "", nil
}

func TestFormatForExport(t *testing.T) {
	t.Parallel()

	t.Run("empty remotes returns nil", func(t *testing.T) {
		t.Parallel()

		mockStore := newMockTransformationStore()
		handler := transformation.NewHandler(mockStore)

		namer := &mockNamer{}
		resolver := &mockResolver{}

		result, err := handler.Impl.FormatForExport(map[string]*model.RemoteTransformation{}, namer, resolver)

		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("javascript transformation exports spec and code file", func(t *testing.T) {
		t.Parallel()

		mockStore := newMockTransformationStore()
		handler := transformation.NewHandler(mockStore)

		remotes := map[string]*model.RemoteTransformation{
			"test-trans": {
				Transformation: &transformations.Transformation{
					ID:          "trans-123",
					VersionID:   "ver-456",
					Name:        "Test Transformation",
					Description: "Test description",
					Code:        "export function transformEvent(event, metadata) { return event; }",
					Language:    "javascript",
					WorkspaceID: "ws-789",
				},
			},
		}

		namer := &mockNamer{}
		resolver := &mockResolver{}

		result, err := handler.Impl.FormatForExport(remotes, namer, resolver)

		require.NoError(t, err)
		require.Len(t, result, 2)

		// Check YAML spec file
		assert.Equal(t, "transformations/test-trans.yaml", result[0].RelativePath)

		// Check code file
		assert.Equal(t, "transformations/javascript/test-trans.js", result[1].RelativePath)
		assert.Equal(t, "export function transformEvent(event, metadata) { return event; }", result[1].Content)
	})

	t.Run("python transformation exports to python folder", func(t *testing.T) {
		t.Parallel()

		mockStore := newMockTransformationStore()
		handler := transformation.NewHandler(mockStore)

		remotes := map[string]*model.RemoteTransformation{
			"test-trans": {
				Transformation: &transformations.Transformation{
					ID:          "trans-123",
					VersionID:   "ver-456",
					Name:        "Test Transformation",
					Description: "Test description",
					Code:        "def transform(event):\n    return event",
					Language:    "python",
					WorkspaceID: "ws-789",
				},
			},
		}

		namer := &mockNamer{}
		resolver := &mockResolver{}

		result, err := handler.Impl.FormatForExport(remotes, namer, resolver)

		require.NoError(t, err)
		require.Len(t, result, 2)

		// Check code file path
		assert.Equal(t, "transformations/python/test-trans.py", result[1].RelativePath)
		assert.Equal(t, "def transform(event):\n    return event", result[1].Content)
	})

	t.Run("unsupported language returns error", func(t *testing.T) {
		t.Parallel()

		mockStore := newMockTransformationStore()
		handler := transformation.NewHandler(mockStore)

		remotes := map[string]*model.RemoteTransformation{
			"test-trans": {
				Transformation: &transformations.Transformation{
					ID:          "trans-123",
					VersionID:   "ver-456",
					Name:        "Test Transformation",
					Code:        "package main",
					Language:    "golang",
					WorkspaceID: "ws-789",
				},
			},
		}

		namer := &mockNamer{}
		resolver := &mockResolver{}

		result, err := handler.Impl.FormatForExport(remotes, namer, resolver)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported language 'golang'")
		assert.Nil(t, result)
	})

	t.Run("namer error", func(t *testing.T) {
		t.Parallel()

		mockStore := newMockTransformationStore()
		handler := transformation.NewHandler(mockStore)

		remotes := map[string]*model.RemoteTransformation{
			"test-trans": {
				Transformation: &transformations.Transformation{
					ID:          "trans-123",
					VersionID:   "ver-456",
					Name:        "Test Transformation",
					Code:        "export function transformEvent(event, metadata) { return event; }",
					Language:    "javascript",
					WorkspaceID: "ws-789",
				},
			},
		}

		namer := &mockNamer{
			nameFunc: func(scope namer.ScopeName) (string, error) {
				return "", fmt.Errorf("namer error")
			},
		}
		resolver := &mockResolver{}

		result, err := handler.Impl.FormatForExport(remotes, namer, resolver)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "generating file name for transformation")
		assert.Nil(t, result)
	})

	t.Run("multiple transformations", func(t *testing.T) {
		t.Parallel()

		mockStore := newMockTransformationStore()
		handler := transformation.NewHandler(mockStore)

		remotes := map[string]*model.RemoteTransformation{
			"test-trans-1": {
				Transformation: &transformations.Transformation{
					ID:          "trans-123",
					VersionID:   "ver-456",
					Name:        "Test Transformation 1",
					Code:        "export function transformEvent(event, metadata) { return event; }",
					Language:    "javascript",
					WorkspaceID: "ws-789",
				},
			},
			"test-trans-2": {
				Transformation: &transformations.Transformation{
					ID:          "trans-456",
					VersionID:   "ver-789",
					Name:        "Test Transformation 2",
					Code:        "def transform(event):\n    return event",
					Language:    "python",
					WorkspaceID: "ws-789",
				},
			},
		}

		namer := &mockNamer{}
		resolver := &mockResolver{}

		result, err := handler.Impl.FormatForExport(remotes, namer, resolver)

		require.NoError(t, err)
		require.Len(t, result, 4) // 2 specs + 2 code files
	})
}
