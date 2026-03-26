package helpers

import (
	"context"
	"fmt"
	"testing"

	transformations "github.com/rudderlabs/rudder-iac/api/client/transformations"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/testorchestrator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockTransformationStore implements transformations.TransformationStore for testing,
// returning preset Transformation and Library values keyed by "id:versionId".
type mockTransformationStore struct {
	transformations.TransformationStore
	transformationVersions map[string]*transformations.Transformation
	libraryVersions        map[string]*transformations.TransformationLibrary
}

func (m *mockTransformationStore) GetTransformationVersion(_ context.Context, id, versionID string) (*transformations.Transformation, error) {
	key := id + ":" + versionID
	if v, ok := m.transformationVersions[key]; ok {
		return v, nil
	}
	return nil, fmt.Errorf("transformation version %s not found", key)
}

func (m *mockTransformationStore) GetLibraryVersion(_ context.Context, id, versionID string) (*transformations.TransformationLibrary, error) {
	key := id + ":" + versionID
	if v, ok := m.libraryVersions[key]; ok {
		return v, nil
	}
	return nil, fmt.Errorf("library version %s not found", key)
}

func TestTransformationSnapshotTester(t *testing.T) {
	t.Parallel()

	t.Run("passes when actual matches expected", func(t *testing.T) {
		t.Parallel()

		store := &mockTransformationStore{
			transformationVersions: map[string]*transformations.Transformation{
				"tr-1:v-1": {
					ID: "tr-1", VersionID: "v-1", WorkspaceID: "ws-123",
					Name: "My Transform", Description: "A test transformation",
					Code:     "export function transformEvent(event) { return event; }",
					Language: "javascript", Imports: []string{},
				},
			},
		}

		refs := map[string]VersionedResourceRef{
			"transformation:my_transform": {ResourceID: "tr-1", VersionID: "v-1"},
		}

		fileManager, err := NewSnapshotFileManager("testdata/snapshot/transformations")
		require.NoError(t, err)

		tester := NewTransformationSnapshotTester(store, refs, fileManager,
			[]string{"id", "versionId", "workspaceId", "externalId"})

		err = tester.SnapshotTest(context.Background())
		assert.NoError(t, err)
	})

	t.Run("detects mismatch", func(t *testing.T) {
		t.Parallel()

		store := &mockTransformationStore{
			transformationVersions: map[string]*transformations.Transformation{
				"tr-1:v-1": {
					ID: "tr-1", VersionID: "v-1", WorkspaceID: "ws-123",
					Name: "My Transform", Description: "WRONG description",
					Code:     "export function transformEvent(event) { return event; }",
					Language: "javascript", Imports: []string{},
				},
			},
		}

		refs := map[string]VersionedResourceRef{
			"transformation:my_transform": {ResourceID: "tr-1", VersionID: "v-1"},
		}

		fileManager, err := NewSnapshotFileManager("testdata/snapshot/transformations")
		require.NoError(t, err)

		tester := NewTransformationSnapshotTester(store, refs, fileManager,
			[]string{"id", "versionId", "workspaceId", "externalId"})

		err = tester.SnapshotTest(context.Background())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "description")
		assert.Contains(t, err.Error(), "WRONG description")
	})

	t.Run("ignores dynamic fields", func(t *testing.T) {
		t.Parallel()

		store := &mockTransformationStore{
			libraryVersions: map[string]*transformations.TransformationLibrary{
				"lib-1:v-1": {
					ID: "different-id", VersionID: "different-version", WorkspaceID: "different-ws",
					Name: "My Library", Description: "A test library",
					Code:     "export function helper() { return true; }",
					Language: "javascript", ImportName: "myLibrary",
				},
			},
		}

		refs := map[string]VersionedResourceRef{
			"transformation-library:my_library": {ResourceID: "lib-1", VersionID: "v-1"},
		}

		fileManager, err := NewSnapshotFileManager("testdata/snapshot/transformations")
		require.NoError(t, err)

		tester := NewTransformationSnapshotTester(store, refs, fileManager,
			[]string{"id", "versionId", "workspaceId", "externalId"})

		err = tester.SnapshotTest(context.Background())
		assert.NoError(t, err)
	})
}

func TestVersionRefs(t *testing.T) {
	t.Parallel()

	results := &testorchestrator.TestResults{
		Transformations: []*testorchestrator.TransformationTestWithDefinitions{
			{Result: &transformations.TransformationTestResult{
				ID: "tr-1", ExternalID: "simple_transform", Name: "Simple Transform", VersionID: "v-1",
			}},
			{Result: &transformations.TransformationTestResult{
				ID: "tr-2", ExternalID: "greeting_transform", Name: "Greeting Transform", VersionID: "v-2",
			}},
		},
		Libraries: []transformations.LibraryTestResult{
			{ID: "lib-1", ExternalID: "utils_library", HandleName: "utilsLibrary", VersionID: "v-1"},
		},
	}

	refs := VersionRefs(results)

	assert.Equal(t, map[string]VersionedResourceRef{
		"transformation:simple_transform":      {ResourceID: "tr-1", VersionID: "v-1"},
		"transformation:greeting_transform":    {ResourceID: "tr-2", VersionID: "v-2"},
		"transformation-library:utils_library": {ResourceID: "lib-1", VersionID: "v-1"},
	}, refs)
}
