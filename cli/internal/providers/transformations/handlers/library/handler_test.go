package library_test

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
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/handlers/library"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

// mockTransformationStore implements the TransformationStore interface for testing
type mockTransformationStore struct {
	createLibraryCalled        bool
	updateLibraryCalled        bool
	deleteLibraryCalled        bool
	listLibrariesCalled        bool
	getLibraryCalled           bool
	setLibraryExternalIDCalled bool

	createLibraryFunc        func(ctx context.Context, req *transformations.CreateLibraryRequest, publish bool) (*transformations.TransformationLibrary, error)
	updateLibraryFunc        func(ctx context.Context, id string, req *transformations.UpdateLibraryRequest, publish bool) (*transformations.TransformationLibrary, error)
	deleteLibraryFunc        func(ctx context.Context, id string) error
	listLibrariesFunc        func(ctx context.Context) ([]*transformations.TransformationLibrary, error)
	getLibraryFunc           func(ctx context.Context, id string) (*transformations.TransformationLibrary, error)
	setLibraryExternalIDFunc func(ctx context.Context, id string, externalID string) error
}

func newMockTransformationStore() *mockTransformationStore {
	return &mockTransformationStore{}
}

func (m *mockTransformationStore) CreateLibrary(ctx context.Context, req *transformations.CreateLibraryRequest, publish bool) (*transformations.TransformationLibrary, error) {
	m.createLibraryCalled = true
	if m.createLibraryFunc != nil {
		return m.createLibraryFunc(ctx, req, publish)
	}
	return &transformations.TransformationLibrary{
		ID:          "lib-123",
		VersionID:   "ver-456",
		Name:        req.Name,
		Description: req.Description,
		Code:        req.Code,
		Language:    req.Language,
		ImportName:  "testLibrary",
		ExternalID:  req.ExternalID,
		WorkspaceID: "ws-789",
	}, nil
}

func (m *mockTransformationStore) UpdateLibrary(ctx context.Context, id string, req *transformations.UpdateLibraryRequest, publish bool) (*transformations.TransformationLibrary, error) {
	m.updateLibraryCalled = true
	if m.updateLibraryFunc != nil {
		return m.updateLibraryFunc(ctx, id, req, publish)
	}
	return &transformations.TransformationLibrary{
		ID:          id,
		VersionID:   "ver-updated",
		Name:        req.Name,
		Description: req.Description,
		Code:        req.Code,
		Language:    req.Language,
		ImportName:  "updatedLibrary",
		WorkspaceID: "ws-789",
	}, nil
}

func (m *mockTransformationStore) DeleteLibrary(ctx context.Context, id string) error {
	m.deleteLibraryCalled = true
	if m.deleteLibraryFunc != nil {
		return m.deleteLibraryFunc(ctx, id)
	}
	return nil
}

func (m *mockTransformationStore) ListLibraries(ctx context.Context) ([]*transformations.TransformationLibrary, error) {
	m.listLibrariesCalled = true
	if m.listLibrariesFunc != nil {
		return m.listLibrariesFunc(ctx)
	}
	return []*transformations.TransformationLibrary{}, nil
}

func (m *mockTransformationStore) GetLibrary(ctx context.Context, id string) (*transformations.TransformationLibrary, error) {
	m.getLibraryCalled = true
	if m.getLibraryFunc != nil {
		return m.getLibraryFunc(ctx, id)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *mockTransformationStore) SetLibraryExternalID(ctx context.Context, id string, externalID string) error {
	m.setLibraryExternalIDCalled = true
	if m.setLibraryExternalIDFunc != nil {
		return m.setLibraryExternalIDFunc(ctx, id, externalID)
	}
	return nil
}

func (m *mockTransformationStore) CreateTransformation(ctx context.Context, req *transformations.CreateTransformationRequest, publish bool) (*transformations.Transformation, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockTransformationStore) UpdateTransformation(ctx context.Context, id string, req *transformations.UpdateTransformationRequest, publish bool) (*transformations.Transformation, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockTransformationStore) GetTransformation(ctx context.Context, id string) (*transformations.Transformation, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockTransformationStore) ListTransformations(ctx context.Context) ([]*transformations.Transformation, error) {
	return []*transformations.Transformation{}, nil
}

func (m *mockTransformationStore) DeleteTransformation(ctx context.Context, id string) error {
	return fmt.Errorf("not implemented")
}

func (m *mockTransformationStore) SetTransformationExternalID(ctx context.Context, id string, externalID string) error {
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
	handler := library.NewHandler(mockStore)

	metadata := handler.Impl.Metadata()

	assert.Equal(t, "transformation-library", metadata.ResourceType)
	assert.Equal(t, "transformation-library", metadata.SpecKind)
	assert.Equal(t, "transformation-libraries", metadata.SpecMetadataName)
}

func TestNewSpec(t *testing.T) {
	t.Parallel()

	mockStore := newMockTransformationStore()
	handler := library.NewHandler(mockStore)

	spec := handler.Impl.NewSpec()

	require.NotNil(t, spec)
	assert.IsType(t, &model.LibrarySpec{}, spec)
}

func TestValidateSpec(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		spec          *model.LibrarySpec
		expectedError bool
		errorContains string
	}{
		{
			name: "valid spec with inline code",
			spec: &model.LibrarySpec{
				ID:          "test-lib",
				Name:        "Test Library",
				Description: "Test description",
				Language:    "javascript",
				Code:        "export function helper() { return true; }",
				ImportName:  "testLibrary",
			},
			expectedError: false,
		},
		{
			name: "valid spec with file reference",
			spec: &model.LibrarySpec{
				ID:          "test-lib",
				Name:        "Test Library",
				Description: "Test description",
				Language:    "python",
				File:        "library.py",
				ImportName:  "testLibrary",
			},
			expectedError: false,
		},
		{
			name: "valid spec with python language",
			spec: &model.LibrarySpec{
				ID:         "test-lib",
				Name:       "Test Library",
				Language:   "python",
				Code:       "def helper():\n    return True",
				ImportName: "testLibrary",
			},
			expectedError: false,
		},
		{
			name: "missing id",
			spec: &model.LibrarySpec{
				Name:       "Test Library",
				Language:   "javascript",
				Code:       "export function helper() { return true; }",
				ImportName: "testLibrary",
			},
			expectedError: true,
			errorContains: "id is required",
		},
		{
			name: "missing name",
			spec: &model.LibrarySpec{
				ID:         "test-lib",
				Language:   "javascript",
				Code:       "export function helper() { return true; }",
				ImportName: "testLibrary",
			},
			expectedError: true,
			errorContains: "name is required",
		},
		{
			name: "missing import_name",
			spec: &model.LibrarySpec{
				ID:       "test-lib",
				Name:     "Test Library",
				Language: "javascript",
				Code:     "export function helper() { return true; }",
			},
			expectedError: true,
			errorContains: "import_name is required",
		},
		{
			name: "both code and file specified",
			spec: &model.LibrarySpec{
				ID:         "test-lib",
				Name:       "Test Library",
				Language:   "javascript",
				Code:       "export function helper() { return true; }",
				File:       "library.js",
				ImportName: "testLibrary",
			},
			expectedError: true,
			errorContains: "code and file are mutually exclusive",
		},
		{
			name: "neither code nor file specified",
			spec: &model.LibrarySpec{
				ID:         "test-lib",
				Name:       "Test Library",
				Language:   "javascript",
				ImportName: "testLibrary",
			},
			expectedError: true,
			errorContains: "either code or file must be specified",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockStore := newMockTransformationStore()
			handler := library.NewHandler(mockStore)

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
		handler := library.NewHandler(mockStore)

		spec := &model.LibrarySpec{
			ID:          "test-lib",
			Name:        "Test Library",
			Description: "Test description",
			Language:    "javascript",
			Code:        "export function helper() { return true; }",
			ImportName:  "testLibrary",
		}

		resources, err := handler.Impl.ExtractResourcesFromSpec("/path/to/spec.yaml", spec)

		require.NoError(t, err)
		require.NotNil(t, resources)
		require.Len(t, resources, 1)

		resource := resources["test-lib"]
		require.NotNil(t, resource)
		assert.Equal(t, "test-lib", resource.ID)
		assert.Equal(t, "Test Library", resource.Name)
		assert.Equal(t, "Test description", resource.Description)
		assert.Equal(t, "javascript", resource.Language)
		assert.Equal(t, "export function helper() { return true; }", resource.Code)
		assert.Equal(t, "testLibrary", resource.ImportName)
	})

	t.Run("with file reference - absolute path", func(t *testing.T) {
		t.Parallel()

		// Create a temporary file with code
		tmpDir := t.TempDir()
		codeFile := filepath.Join(tmpDir, "library.js")
		codeContent := "export function helper() { return true; }"
		err := os.WriteFile(codeFile, []byte(codeContent), 0644)
		require.NoError(t, err)

		mockStore := newMockTransformationStore()
		handler := library.NewHandler(mockStore)

		spec := &model.LibrarySpec{
			ID:          "test-lib",
			Name:        "Test Library",
			Description: "Test description",
			Language:    "javascript",
			File:        codeFile,
			ImportName:  "testLibrary",
		}

		resources, err := handler.Impl.ExtractResourcesFromSpec("/path/to/spec.yaml", spec)

		require.NoError(t, err)
		require.NotNil(t, resources)
		require.Len(t, resources, 1)

		resource := resources["test-lib"]
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

		codeFile := filepath.Join(specDir, "library.js")
		codeContent := "export function helper() { return true; }"
		err = os.WriteFile(codeFile, []byte(codeContent), 0644)
		require.NoError(t, err)

		mockStore := newMockTransformationStore()
		handler := library.NewHandler(mockStore)

		spec := &model.LibrarySpec{
			ID:         "test-lib",
			Name:       "Test Library",
			Language:   "javascript",
			File:       "library.js",
			ImportName: "testLibrary",
		}

		specPath := filepath.Join(specDir, "spec.yaml")
		resources, err := handler.Impl.ExtractResourcesFromSpec(specPath, spec)

		require.NoError(t, err)
		require.NotNil(t, resources)
		require.Len(t, resources, 1)

		resource := resources["test-lib"]
		require.NotNil(t, resource)
		assert.Equal(t, codeContent, resource.Code)
	})

	t.Run("with file reference - file not found", func(t *testing.T) {
		t.Parallel()

		mockStore := newMockTransformationStore()
		handler := library.NewHandler(mockStore)

		spec := &model.LibrarySpec{
			ID:         "test-lib",
			Name:       "Test Library",
			Language:   "javascript",
			File:       "/nonexistent/file.js",
			ImportName: "testLibrary",
		}

		resources, err := handler.Impl.ExtractResourcesFromSpec("/path/to/spec.yaml", spec)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "reading code file")
		assert.Nil(t, resources)
	})
}

func TestValidateResource(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		resource      *model.LibraryResource
		expectedError bool
		errorContains string
	}{
		{
			name: "valid javascript resource",
			resource: &model.LibraryResource{
				ID:          "test-lib",
				Name:        "Test Library",
				Description: "Test description",
				Language:    "javascript",
				Code:        "export function helper() { return true; }",
				ImportName:  "testLibrary",
			},
			expectedError: false,
		},
		{
			name: "valid python resource - no syntax validation",
			resource: &model.LibraryResource{
				ID:         "test-lib",
				Name:       "Test Library",
				Language:   "python",
				Code:       "def helper():\n    return True",
				ImportName: "testLibrary",
			},
			expectedError: false,
		},
		{
			name: "missing code",
			resource: &model.LibraryResource{
				ID:         "test-lib",
				Name:       "Test Library",
				Language:   "javascript",
				Code:       "",
				ImportName: "testLibrary",
			},
			expectedError: true,
			errorContains: "code is required",
		},
		{
			name: "invalid javascript syntax",
			resource: &model.LibraryResource{
				ID:         "test-lib",
				Name:       "Test Library",
				Language:   "javascript",
				Code:       "export function helper() { return true;",
				ImportName: "testLibrary",
			},
			expectedError: true,
			errorContains: "validating code syntax",
		},
		{
			name: "import_name not camelCase of name",
			resource: &model.LibraryResource{
				ID:         "test-lib",
				Name:       "Test Library",
				Language:   "javascript",
				Code:       "export function helper() { return true; }",
				ImportName: "wrongName",
			},
			expectedError: true,
			errorContains: "import_name must be camelCase of name",
		},
		{
			name: "invalid language",
			resource: &model.LibraryResource{
				ID:         "test-lib",
				Name:       "Test Library",
				Language:   "rust",
				Code:       "fn helper() {}",
				ImportName: "testLibrary",
			},
			expectedError: true,
			errorContains: "language must be javascript or python",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockStore := newMockTransformationStore()
			handler := library.NewHandler(mockStore)

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
		mockStore.listLibrariesFunc = func(ctx context.Context) ([]*transformations.TransformationLibrary, error) {
			return []*transformations.TransformationLibrary{
				{
					ID:          "lib-1",
					VersionID:   "ver-1",
					Name:        "Library 1",
					Description: "Description 1",
					Code:        "export function helper1() { return true; }",
					Language:    "javascript",
					ImportName:  "library1",
					ExternalID:  "ext-1",
					WorkspaceID: "ws-1",
				},
				{
					ID:          "lib-2",
					VersionID:   "ver-2",
					Name:        "Library 2",
					Description: "Description 2",
					Code:        "def helper2():\n    return True",
					Language:    "python",
					ImportName:  "library2",
					ExternalID:  "ext-2",
					WorkspaceID: "ws-1",
				},
			}, nil
		}

		handler := library.NewHandler(mockStore)

		remotes, err := handler.Impl.LoadRemoteResources(context.Background())

		require.NoError(t, err)
		require.Len(t, remotes, 2)
		assert.True(t, mockStore.listLibrariesCalled)

		assert.Equal(t, "lib-1", remotes[0].ID)
		assert.Equal(t, "ext-1", remotes[0].ExternalID)
		assert.Equal(t, "Library 1", remotes[0].Name)
		assert.Equal(t, "library1", remotes[0].ImportName)

		assert.Equal(t, "lib-2", remotes[1].ID)
		assert.Equal(t, "ext-2", remotes[1].ExternalID)
		assert.Equal(t, "Library 2", remotes[1].Name)
		assert.Equal(t, "library2", remotes[1].ImportName)
	})

	t.Run("filters out resources without external IDs", func(t *testing.T) {
		t.Parallel()

		mockStore := newMockTransformationStore()
		mockStore.listLibrariesFunc = func(ctx context.Context) ([]*transformations.TransformationLibrary, error) {
			return []*transformations.TransformationLibrary{
				{
					ID:          "lib-1",
					VersionID:   "ver-1",
					Name:        "Library 1",
					Code:        "export function helper() { return true; }",
					Language:    "javascript",
					ImportName:  "library1",
					ExternalID:  "ext-1",
					WorkspaceID: "ws-1",
				},
				{
					ID:          "lib-2",
					VersionID:   "ver-2",
					Name:        "Library 2",
					Code:        "export function helper() { return true; }",
					Language:    "javascript",
					ImportName:  "library2",
					ExternalID:  "", // No external ID
					WorkspaceID: "ws-1",
				},
			}, nil
		}

		handler := library.NewHandler(mockStore)

		remotes, err := handler.Impl.LoadRemoteResources(context.Background())

		require.NoError(t, err)
		require.Len(t, remotes, 1)
		assert.Equal(t, "ext-1", remotes[0].ExternalID)
	})

	t.Run("empty list", func(t *testing.T) {
		t.Parallel()

		mockStore := newMockTransformationStore()
		mockStore.listLibrariesFunc = func(ctx context.Context) ([]*transformations.TransformationLibrary, error) {
			return []*transformations.TransformationLibrary{}, nil
		}

		handler := library.NewHandler(mockStore)

		remotes, err := handler.Impl.LoadRemoteResources(context.Background())

		require.NoError(t, err)
		require.Len(t, remotes, 0)
	})

	t.Run("API error", func(t *testing.T) {
		t.Parallel()

		mockStore := newMockTransformationStore()
		mockStore.listLibrariesFunc = func(ctx context.Context) ([]*transformations.TransformationLibrary, error) {
			return nil, fmt.Errorf("API error")
		}

		handler := library.NewHandler(mockStore)

		remotes, err := handler.Impl.LoadRemoteResources(context.Background())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "listing libraries")
		assert.Nil(t, remotes)
	})
}

func TestLoadImportableResources(t *testing.T) {
	t.Parallel()

	t.Run("returns only resources without external IDs", func(t *testing.T) {
		t.Parallel()

		mockStore := newMockTransformationStore()
		mockStore.listLibrariesFunc = func(ctx context.Context) ([]*transformations.TransformationLibrary, error) {
			return []*transformations.TransformationLibrary{
				{
					ID:          "lib-1",
					VersionID:   "ver-1",
					Name:        "Library 1",
					Description: "Test library 1",
					Code:        "export function helper1() { return true; }",
					Language:    "javascript",
					ImportName:  "library1",
					WorkspaceID: "ws-1",
					ExternalID:  "", // No external ID - should be included
				},
				{
					ID:          "lib-2",
					VersionID:   "ver-2",
					Name:        "Library 2",
					Description: "Test library 2",
					Code:        "export function helper2() { return true; }",
					Language:    "javascript",
					ImportName:  "library2",
					WorkspaceID: "ws-1",
					ExternalID:  "ext-lib-2", // Has external ID - should be filtered out
				},
				{
					ID:          "lib-3",
					VersionID:   "ver-3",
					Name:        "Library 3",
					Description: "Test library 3",
					Code:        "export function helper3() { return true; }",
					Language:    "javascript",
					ImportName:  "library3",
					WorkspaceID: "ws-1",
					ExternalID:  "", // No external ID - should be included
				},
			}, nil
		}

		handler := library.NewHandler(mockStore)
		importables, err := handler.Impl.LoadImportableResources(context.Background())

		require.NoError(t, err)
		require.NotNil(t, importables)
		assert.Len(t, importables, 2) // Only lib-1 and lib-3 (no external IDs)
		assert.Equal(t, "lib-1", importables[0].ID)
		assert.Equal(t, "Library 1", importables[0].Name)
		assert.Equal(t, "lib-3", importables[1].ID)
		assert.Equal(t, "Library 3", importables[1].Name)
	})

	t.Run("empty list", func(t *testing.T) {
		t.Parallel()

		mockStore := newMockTransformationStore()
		mockStore.listLibrariesFunc = func(ctx context.Context) ([]*transformations.TransformationLibrary, error) {
			return []*transformations.TransformationLibrary{}, nil
		}

		handler := library.NewHandler(mockStore)
		importables, err := handler.Impl.LoadImportableResources(context.Background())

		require.NoError(t, err)
		require.NotNil(t, importables)
		assert.Len(t, importables, 0)
	})

	t.Run("all have external IDs", func(t *testing.T) {
		t.Parallel()

		mockStore := newMockTransformationStore()
		mockStore.listLibrariesFunc = func(ctx context.Context) ([]*transformations.TransformationLibrary, error) {
			return []*transformations.TransformationLibrary{
				{
					ID:         "lib-1",
					VersionID:  "ver-1",
					Name:       "Library 1",
					ExternalID: "ext-lib-1",
				},
				{
					ID:         "lib-2",
					VersionID:  "ver-2",
					Name:       "Library 2",
					ExternalID: "ext-lib-2",
				},
			}, nil
		}

		handler := library.NewHandler(mockStore)
		importables, err := handler.Impl.LoadImportableResources(context.Background())

		require.NoError(t, err)
		require.NotNil(t, importables)
		assert.Len(t, importables, 0) // All filtered out
	})

	t.Run("API error", func(t *testing.T) {
		t.Parallel()

		mockStore := newMockTransformationStore()
		mockStore.listLibrariesFunc = func(ctx context.Context) ([]*transformations.TransformationLibrary, error) {
			return nil, fmt.Errorf("API error")
		}

		handler := library.NewHandler(mockStore)
		importables, err := handler.Impl.LoadImportableResources(context.Background())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "listing libraries")
		assert.Nil(t, importables)
	})
}

func TestMapRemoteToState(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		remote := &model.RemoteLibrary{
			TransformationLibrary: &transformations.TransformationLibrary{
				ID:          "lib-123",
				VersionID:   "ver-456",
				Name:        "Test Library",
				Description: "Test description",
				Code:        "export function helper() { return true; }",
				Language:    "javascript",
				ImportName:  "testLibrary",
				ExternalID:  "ext-123",
				WorkspaceID: "ws-789",
			},
		}

		mockStore := newMockTransformationStore()
		handler := library.NewHandler(mockStore)

		resource, state, err := handler.Impl.MapRemoteToState(remote, nil)

		require.NoError(t, err)
		require.NotNil(t, resource)
		require.NotNil(t, state)

		assert.Equal(t, "ext-123", resource.ID)
		assert.Equal(t, "Test Library", resource.Name)
		assert.Equal(t, "Test description", resource.Description)
		assert.Equal(t, "javascript", resource.Language)
		assert.Equal(t, "export function helper() { return true; }", resource.Code)
		assert.Equal(t, "testLibrary", resource.ImportName)

		assert.Equal(t, "lib-123", state.ID)
		assert.Equal(t, "ver-456", state.VersionID)
	})
}

func TestCreate(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		mockStore := newMockTransformationStore()
		handler := library.NewHandler(mockStore)

		resource := &model.LibraryResource{
			ID:          "ext-123",
			Name:        "Test Library",
			Description: "Test description",
			Language:    "javascript",
			Code:        "export function helper() { return true; }",
			ImportName:  "testLibrary",
		}

		state, err := handler.Impl.Create(context.Background(), resource)

		require.NoError(t, err)
		require.NotNil(t, state)
		assert.True(t, mockStore.createLibraryCalled)

		assert.Equal(t, "lib-123", state.ID)
		assert.Equal(t, "ver-456", state.VersionID)
	})

	t.Run("API error", func(t *testing.T) {
		t.Parallel()

		mockStore := newMockTransformationStore()
		mockStore.createLibraryFunc = func(ctx context.Context, req *transformations.CreateLibraryRequest, publish bool) (*transformations.TransformationLibrary, error) {
			return nil, fmt.Errorf("API error")
		}

		handler := library.NewHandler(mockStore)

		resource := &model.LibraryResource{
			ID:         "ext-123",
			Name:       "Test Library",
			Language:   "javascript",
			Code:       "export function helper() { return true; }",
			ImportName: "testLibrary",
		}

		state, err := handler.Impl.Create(context.Background(), resource)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "creating library")
		assert.Nil(t, state)
	})

	t.Run("passes correct parameters", func(t *testing.T) {
		t.Parallel()

		var capturedReq *transformations.CreateLibraryRequest
		var capturedPublish bool

		mockStore := newMockTransformationStore()
		mockStore.createLibraryFunc = func(ctx context.Context, req *transformations.CreateLibraryRequest, publish bool) (*transformations.TransformationLibrary, error) {
			capturedReq = req
			capturedPublish = publish
			return &transformations.TransformationLibrary{
				ID:        "lib-123",
				VersionID: "ver-456",
			}, nil
		}

		handler := library.NewHandler(mockStore)

		resource := &model.LibraryResource{
			ID:          "ext-123",
			Name:        "Test Library",
			Description: "Test description",
			Language:    "javascript",
			Code:        "export function helper() { return true; }",
			ImportName:  "testLibrary",
		}

		_, err := handler.Impl.Create(context.Background(), resource)

		require.NoError(t, err)
		require.NotNil(t, capturedReq)

		assert.Equal(t, "Test Library", capturedReq.Name)
		assert.Equal(t, "Test description", capturedReq.Description)
		assert.Equal(t, "export function helper() { return true; }", capturedReq.Code)
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
		handler := library.NewHandler(mockStore)

		newData := &model.LibraryResource{
			ID:          "ext-123",
			Name:        "Updated Library",
			Description: "Updated description",
			Language:    "javascript",
			Code:        "export function helper() { return true; }",
			ImportName:  "updatedLibrary",
		}

		oldData := &model.LibraryResource{
			ID:          "ext-123",
			Name:        "Old Library",
			Description: "Old description",
			Language:    "javascript",
			Code:        "export function helper() { return false; }",
			ImportName:  "oldLibrary",
		}

		oldState := &model.LibraryState{
			ID:        "lib-123",
			VersionID: "ver-456",
		}

		state, err := handler.Impl.Update(context.Background(), newData, oldData, oldState)

		require.NoError(t, err)
		require.NotNil(t, state)
		assert.True(t, mockStore.updateLibraryCalled)

		assert.Equal(t, "lib-123", state.ID)
		assert.Equal(t, "ver-updated", state.VersionID)
	})

	t.Run("API error", func(t *testing.T) {
		t.Parallel()

		mockStore := newMockTransformationStore()
		mockStore.updateLibraryFunc = func(ctx context.Context, id string, req *transformations.UpdateLibraryRequest, publish bool) (*transformations.TransformationLibrary, error) {
			return nil, fmt.Errorf("API error")
		}

		handler := library.NewHandler(mockStore)

		newData := &model.LibraryResource{
			ID:         "ext-123",
			Name:       "Updated Library",
			Language:   "javascript",
			Code:       "export function helper() { return true; }",
			ImportName: "updatedLibrary",
		}

		oldData := &model.LibraryResource{
			ID:         "ext-123",
			Name:       "Old Library",
			Language:   "javascript",
			Code:       "export function helper() { return false; }",
			ImportName: "oldLibrary",
		}

		oldState := &model.LibraryState{
			ID:        "lib-123",
			VersionID: "ver-456",
		}

		state, err := handler.Impl.Update(context.Background(), newData, oldData, oldState)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "updating library")
		assert.Nil(t, state)
	})

	t.Run("passes correct parameters", func(t *testing.T) {
		t.Parallel()

		var capturedID string
		var capturedReq *transformations.UpdateLibraryRequest
		var capturedPublish bool

		mockStore := newMockTransformationStore()
		mockStore.updateLibraryFunc = func(ctx context.Context, id string, req *transformations.UpdateLibraryRequest, publish bool) (*transformations.TransformationLibrary, error) {
			capturedID = id
			capturedReq = req
			capturedPublish = publish
			return &transformations.TransformationLibrary{
				ID:        id,
				VersionID: "ver-updated",
			}, nil
		}

		handler := library.NewHandler(mockStore)

		newData := &model.LibraryResource{
			ID:          "ext-123",
			Name:        "Updated Library",
			Description: "Updated description",
			Language:    "javascript",
			Code:        "export function helper() { return true; }",
			ImportName:  "updatedLibrary",
		}

		oldData := &model.LibraryResource{
			ID:         "ext-123",
			Name:       "Old Library",
			Language:   "javascript",
			Code:       "old code",
			ImportName: "oldLibrary",
		}

		oldState := &model.LibraryState{
			ID:        "lib-123",
			VersionID: "ver-456",
		}

		_, err := handler.Impl.Update(context.Background(), newData, oldData, oldState)

		require.NoError(t, err)
		require.NotNil(t, capturedReq)

		assert.Equal(t, "lib-123", capturedID)
		assert.Equal(t, "Updated Library", capturedReq.Name)
		assert.Equal(t, "Updated description", capturedReq.Description)
		assert.Equal(t, "export function helper() { return true; }", capturedReq.Code)
		assert.Equal(t, "javascript", capturedReq.Language)
		assert.False(t, capturedPublish)
	})
}

func TestDelete(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		mockStore := newMockTransformationStore()
		handler := library.NewHandler(mockStore)

		oldData := &model.LibraryResource{
			ID:         "ext-123",
			Name:       "Test Library",
			Language:   "javascript",
			Code:       "export function helper() { return true; }",
			ImportName: "testLibrary",
		}

		oldState := &model.LibraryState{
			ID:        "lib-123",
			VersionID: "ver-456",
		}

		err := handler.Impl.Delete(context.Background(), "ext-123", oldData, oldState)

		require.NoError(t, err)
		assert.True(t, mockStore.deleteLibraryCalled)
	})

	t.Run("API error", func(t *testing.T) {
		t.Parallel()

		mockStore := newMockTransformationStore()
		mockStore.deleteLibraryFunc = func(ctx context.Context, id string) error {
			return fmt.Errorf("API error")
		}

		handler := library.NewHandler(mockStore)

		oldData := &model.LibraryResource{
			ID:   "ext-123",
			Name: "Test Library",
		}

		oldState := &model.LibraryState{
			ID:        "lib-123",
			VersionID: "ver-456",
		}

		err := handler.Impl.Delete(context.Background(), "ext-123", oldData, oldState)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "deleting library")
	})

	t.Run("uses state ID not external ID", func(t *testing.T) {
		t.Parallel()

		var capturedID string

		mockStore := newMockTransformationStore()
		mockStore.deleteLibraryFunc = func(ctx context.Context, id string) error {
			capturedID = id
			return nil
		}

		handler := library.NewHandler(mockStore)

		oldData := &model.LibraryResource{
			ID:   "ext-123",
			Name: "Test Library",
		}

		oldState := &model.LibraryState{
			ID:        "lib-123",
			VersionID: "ver-456",
		}

		err := handler.Impl.Delete(context.Background(), "ext-123", oldData, oldState)

		require.NoError(t, err)
		assert.Equal(t, "lib-123", capturedID)
	})
}

func TestImport(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		var capturedExternalID string
		mockStore := newMockTransformationStore()
		mockStore.getLibraryFunc = func(ctx context.Context, id string) (*transformations.TransformationLibrary, error) {
			return &transformations.TransformationLibrary{
				ID:          "lib-remote-123",
				VersionID:   "ver-remote-456",
				Name:        "Remote Library",
				Description: "Remote library description",
				Code:        "export function helper() { return true; }",
				Language:    "javascript",
				ImportName:  "remoteLibrary",
				WorkspaceID: "ws-789",
				ExternalID:  "", // No external ID initially
			}, nil
		}
		mockStore.setLibraryExternalIDFunc = func(ctx context.Context, id string, externalID string) error {
			capturedExternalID = externalID
			return nil
		}

		handler := library.NewHandler(mockStore)

		data := &model.LibraryResource{
			ID:         "ext-new-123",
			Name:       "Imported Library",
			Language:   "javascript",
			Code:       "export function helper() { return true; }",
			ImportName: "importedLibrary",
		}

		state, err := handler.Impl.Import(context.Background(), data, "lib-remote-123")

		require.NoError(t, err)
		require.NotNil(t, state)
		assert.Equal(t, "lib-remote-123", state.ID)
		assert.Equal(t, "ver-updated", state.VersionID)
		assert.Equal(t, "ext-new-123", capturedExternalID)
		assert.True(t, mockStore.getLibraryCalled)
		assert.True(t, mockStore.setLibraryExternalIDCalled)
		assert.True(t, mockStore.updateLibraryCalled)
	})

	t.Run("GetLibrary API error", func(t *testing.T) {
		t.Parallel()

		mockStore := newMockTransformationStore()
		mockStore.getLibraryFunc = func(ctx context.Context, id string) (*transformations.TransformationLibrary, error) {
			return nil, fmt.Errorf("library not found")
		}

		handler := library.NewHandler(mockStore)

		data := &model.LibraryResource{
			ID:         "ext-new-123",
			Name:       "Imported Library",
			Language:   "javascript",
			Code:       "export function helper() { return true; }",
			ImportName: "importedLibrary",
		}

		state, err := handler.Impl.Import(context.Background(), data, "lib-remote-123")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "getting library")
		assert.Nil(t, state)
	})

	t.Run("SetLibraryExternalID API error", func(t *testing.T) {
		t.Parallel()

		mockStore := newMockTransformationStore()
		mockStore.getLibraryFunc = func(ctx context.Context, id string) (*transformations.TransformationLibrary, error) {
			return &transformations.TransformationLibrary{
				ID:          "lib-remote-123",
				VersionID:   "ver-remote-456",
				Name:        "Remote Library",
				Description: "Remote library description",
				Code:        "export function helper() { return true; }",
				Language:    "javascript",
				ImportName:  "remoteLibrary",
				WorkspaceID: "ws-789",
			}, nil
		}
		mockStore.setLibraryExternalIDFunc = func(ctx context.Context, id string, externalID string) error {
			return fmt.Errorf("API error setting external ID")
		}

		handler := library.NewHandler(mockStore)

		data := &model.LibraryResource{
			ID:         "ext-new-123",
			Name:       "Imported Library",
			Language:   "javascript",
			Code:       "export function helper() { return true; }",
			ImportName: "importedLibrary",
		}

		state, err := handler.Impl.Import(context.Background(), data, "lib-remote-123")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "setting library external ID")
		assert.Nil(t, state)
	})
}

func TestFormatForExport(t *testing.T) {
	t.Parallel()

	t.Run("empty remotes returns nil", func(t *testing.T) {
		t.Parallel()

		mockStore := newMockTransformationStore()
		handler := library.NewHandler(mockStore)

		remotes := map[string]*model.RemoteLibrary{}
		idNamer := &mockNamer{}
		resolver := &mockResolver{}

		result, err := handler.Impl.FormatForExport(remotes, idNamer, resolver)

		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("javascript library exports spec and code file", func(t *testing.T) {
		t.Parallel()

		mockStore := newMockTransformationStore()
		handler := library.NewHandler(mockStore)

		remotes := map[string]*model.RemoteLibrary{
			"my-library": {
				TransformationLibrary: &transformations.TransformationLibrary{
					ID:          "remote-lib-123",
					VersionID:   "ver-456",
					Name:        "My Library",
					Description: "Test library",
					Code:        "export function helper() { return true; }",
					Language:    "javascript",
					ImportName:  "myLibrary",
					WorkspaceID: "ws-789",
					ExternalID:  "",
				},
			},
		}
		idNamer := &mockNamer{nameFunc: func(scope namer.ScopeName) (string, error) {
			return "my-library", nil
		}}
		resolver := &mockResolver{}

		result, err := handler.Impl.FormatForExport(remotes, idNamer, resolver)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result, 2) // Spec YAML + Code file

		// Check YAML spec
		assert.Equal(t, "transformations/my-library.yaml", result[0].RelativePath)
		assert.IsType(t, &specs.Spec{}, result[0].Content)

		spec := result[0].Content.(*specs.Spec)
		assert.Equal(t, "transformation-library", spec.Kind)
		assert.Equal(t, "my-library", spec.Spec["id"])
		assert.Equal(t, "My Library", spec.Spec["name"])
		assert.Equal(t, "Test library", spec.Spec["description"])
		assert.Equal(t, "javascript", spec.Spec["language"])
		assert.Equal(t, "myLibrary", spec.Spec["import_name"])
		assert.Equal(t, "javascript/my-library.js", spec.Spec["file"])

		// Check code file
		assert.Equal(t, "transformations/javascript/my-library.js", result[1].RelativePath)
		assert.Equal(t, "export function helper() { return true; }", result[1].Content)
	})

	t.Run("python library exports to python folder", func(t *testing.T) {
		t.Parallel()

		mockStore := newMockTransformationStore()
		handler := library.NewHandler(mockStore)

		remotes := map[string]*model.RemoteLibrary{
			"my-py-lib": {
				TransformationLibrary: &transformations.TransformationLibrary{
					ID:          "remote-lib-123",
					VersionID:   "ver-456",
					Name:        "My Python Library",
					Description: "Test python library",
					Code:        "def helper():\n    return True",
					Language:    "python",
					ImportName:  "myPyLib",
					WorkspaceID: "ws-789",
				},
			},
		}
		idNamer := &mockNamer{nameFunc: func(scope namer.ScopeName) (string, error) {
			return "my-py-lib", nil
		}}
		resolver := &mockResolver{}

		result, err := handler.Impl.FormatForExport(remotes, idNamer, resolver)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result, 2)

		// Check YAML spec has .py extension
		spec := result[0].Content.(*specs.Spec)
		assert.Equal(t, "python", spec.Spec["language"])
		assert.Equal(t, "python/my-py-lib.py", spec.Spec["file"])

		// Check code file path
		assert.Equal(t, "transformations/python/my-py-lib.py", result[1].RelativePath)
	})

	t.Run("unsupported language returns error", func(t *testing.T) {
		t.Parallel()

		mockStore := newMockTransformationStore()
		handler := library.NewHandler(mockStore)

		remotes := map[string]*model.RemoteLibrary{
			"bad-lib": {
				TransformationLibrary: &transformations.TransformationLibrary{
					ID:       "remote-lib-123",
					Name:     "Bad Library",
					Language: "ruby",
				},
			},
		}
		idNamer := &mockNamer{}
		resolver := &mockResolver{}

		result, err := handler.Impl.FormatForExport(remotes, idNamer, resolver)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported language 'ruby'")
		assert.Nil(t, result)
	})

	t.Run("namer error returns error", func(t *testing.T) {
		t.Parallel()

		mockStore := newMockTransformationStore()
		handler := library.NewHandler(mockStore)

		remotes := map[string]*model.RemoteLibrary{
			"my-lib": {
				TransformationLibrary: &transformations.TransformationLibrary{
					ID:       "remote-lib-123",
					Name:     "My Library",
					Language: "javascript",
				},
			},
		}
		idNamer := &mockNamer{nameFunc: func(scope namer.ScopeName) (string, error) {
			return "", fmt.Errorf("namer error")
		}}
		resolver := &mockResolver{}

		result, err := handler.Impl.FormatForExport(remotes, idNamer, resolver)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "generating file name")
		assert.Nil(t, result)
	})

	t.Run("multiple libraries", func(t *testing.T) {
		t.Parallel()

		mockStore := newMockTransformationStore()
		handler := library.NewHandler(mockStore)

		remotes := map[string]*model.RemoteLibrary{
			"lib1": {
				TransformationLibrary: &transformations.TransformationLibrary{
					ID:         "remote-lib-1",
					Name:       "Library 1",
					Code:       "export function lib1() { return 1; }",
					Language:   "javascript",
					ImportName: "lib1",
				},
			},
			"lib2": {
				TransformationLibrary: &transformations.TransformationLibrary{
					ID:         "remote-lib-2",
					Name:       "Library 2",
					Code:       "def lib2():\n    return 2",
					Language:   "python",
					ImportName: "lib2",
				},
			},
		}
		idNamer := &mockNamer{nameFunc: func(scope namer.ScopeName) (string, error) {
			return scope.Name, nil
		}}
		resolver := &mockResolver{}

		result, err := handler.Impl.FormatForExport(remotes, idNamer, resolver)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result, 4) // 2 libraries × (spec + code)

		// Verify we have both libraries
		relativePaths := make([]string, len(result))
		for i, r := range result {
			relativePaths[i] = r.RelativePath
		}
		assert.Contains(t, relativePaths, "transformations/lib1.yaml")
		assert.Contains(t, relativePaths, "transformations/javascript/lib1.js")
		assert.Contains(t, relativePaths, "transformations/lib2.yaml")
		assert.Contains(t, relativePaths, "transformations/python/lib2.py")
	})
}

// Mock implementations for testing FormatForExport

type mockNamer struct {
	nameFunc func(scope namer.ScopeName) (string, error)
}

func (m *mockNamer) Name(scope namer.ScopeName) (string, error) {
	if m.nameFunc != nil {
		return m.nameFunc(scope)
	}
	return "default-name", nil
}

type mockResolver struct {
}

func (m *mockResolver) ResolveReference(urn string) (any, error) {
	return nil, nil
}

func (m *mockNamer) Load(names []namer.ScopeName) error {
	return nil
}

func (m *mockResolver) ResolveToReference(entityType string, remoteID string) (string, error) {
	return "", nil
}
