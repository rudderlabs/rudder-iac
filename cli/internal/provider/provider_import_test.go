package provider_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/importer"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/testutils/example"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/testutils/example/backend"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExampleImport(t *testing.T) {
	t.Parallel()

	// Initialize an empty backend
	b := backend.NewBackend()

	// Create an example provider with that backend
	provider := example.NewProvider(b)

	// Create a test directory for storing test data
	testDir := t.TempDir()

	// Load specs from testdata directory
	proj := project.New(testDir, provider)

	wA, err := b.CreateWriter("Writer A", "")
	require.NoError(t, err)
	wB, err := b.CreateWriter("Writer B", "")
	require.NoError(t, err)

	_, err = b.CreateBook("Book A", wA.ID, "")
	require.NoError(t, err)
	_, err = b.CreateBook("Book B", wB.ID, "")
	require.NoError(t, err)

	err = importer.WorkspaceImport(context.Background(), proj, provider)
	require.NoError(t, err, "Failed to import workspace")

	assertDirContents(t, testDir)

	// t.Fatal("Import completed")
}

func assertDirContents(t *testing.T, dir string) {
	t.Helper()

	fmt.Printf("listing directory %s\n", dir)

	// list all files in temporary directory
	files, err := os.ReadDir(dir)
	require.NoError(t, err, "Failed to read directory contents")
	assert.Len(t, files, 1, "Should contain 'imported' directory")

	importedDir := filepath.Join(dir, "imported")
	fmt.Printf("listing imported directory %s\n", importedDir)

	printFileContentsRecursively(t, importedDir)
}

func printFileContentsRecursively(t *testing.T, dir string) {
	t.Helper()

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			relPath, _ := filepath.Rel(dir, path)
			t.Logf("Found file: %s", relPath)
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			t.Logf("File contents:\n%s", string(content))
		}
		return nil
	})
	require.NoError(t, err, "Failed to walk directory")
}
