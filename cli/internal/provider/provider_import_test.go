package provider_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/importer"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/testutils/example"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/testutils/example/backend"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer"
	"github.com/rudderlabs/rudder-iac/cli/internal/varsubst"
	"github.com/rudderlabs/rudder-iac/cli/internal/varsubst/resolver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
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
	proj := project.New(provider)
	err := proj.Load(testDir)
	require.NoError(t, err, "Failed to load project")

	wA, err := b.CreateWriter("Writer A", "")
	require.NoError(t, err)
	wB, err := b.CreateWriter("Writer B", "")
	require.NoError(t, err)

	_, err = b.CreateBook("Book A", wA.ID, "", "")
	require.NoError(t, err)
	_, err = b.CreateBook("Book B", wB.ID, "", "")
	require.NoError(t, err)

	err = importer.WorkspaceImport(context.Background(), proj, provider, importer.ImportOptions{})
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

// End-to-end shape of DEX-410: importing a resource with a secret yields a
// spec with a variable reference (never a masked literal) plus a var
// file with a placeholder; filling the var file and applying round-trips the
// real value to the backend through the secret type.
func TestImportScaffoldsSecretsViaVarSubstitution(t *testing.T) {
	b := backend.NewBackend()
	testDir := t.TempDir()

	// A remote book with a secret the API will never return.
	w, err := b.CreateWriter("Tolkien", "")
	require.NoError(t, err)
	_, err = b.CreateBook("The Hobbit", w.ID, "", "remote-only-access-key")
	require.NoError(t, err)

	// Import into an empty project.
	importProvider := example.NewProvider(b)
	proj := project.New(importProvider)
	require.NoError(t, proj.Load(testDir))
	require.NoError(t, importer.WorkspaceImport(
		context.Background(),
		proj,
		importProvider,
		importer.ImportOptions{},
	))

	// The generated spec carries an unquoted variable reference, not a mask.
	// The handler names the variable from the resource's identity
	// (BOOK_<id>_...), normalized to the substitution grammar.
	const wantVar = "BOOK_THE_HOBBIT_ACCESS_KEY"
	specBytes, err := os.ReadFile(filepath.Join(testDir, "imported", "books", "books.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(specBytes), "accessKey: {{ ."+wantVar+" }}\n")
	assert.NotContains(t, string(specBytes), "(unknown)")

	// The var file scaffolds an unfilled (null) placeholder for exactly that
	// variable — null so that applying without filling it fails loudly.
	varFilePath := filepath.Join(testDir, "imported", importer.SecretsVarFileName)
	varFileBytes, err := os.ReadFile(varFilePath)
	require.NoError(t, err)
	varFileVars := map[string]any{}
	require.NoError(t, yaml.Unmarshal(varFileBytes, &varFileVars))
	assert.Equal(t, map[string]any{wantVar: nil}, varFileVars)

	// Applying with the unfilled placeholder is rejected by the var-file
	// resolver, so a forgotten secret can never silently apply as "".
	_, err = resolver.NewFileResolver(varFilePath)
	require.ErrorContains(t, err, wantVar)

	// The user fills in the real secret.
	require.NoError(t, os.WriteFile(varFilePath, []byte(wantVar+`: "filled-in-access-key"`+"\n"), 0600))

	workspace := &client.Workspace{ID: "test-workspace-id", Name: "Test Workspace"}

	// First apply links the imported resources; the secret is unknown remotely,
	// so the second apply re-sends it (the always-re-apply diff rule).
	for range 2 {
		applyProvider := example.NewProvider(b)
		fileResolver, err := resolver.NewFileResolver(varFilePath)
		require.NoError(t, err)

		proj := project.New(applyProvider, project.WithSubstitutor(varsubst.NewSubstitutor(fileResolver)))
		require.NoError(t, proj.Load(testDir))

		graph, err := proj.ResourceGraph()
		require.NoError(t, err)
		s, err := syncer.New(applyProvider, workspace)
		require.NoError(t, err)
		require.NoError(t, s.Sync(context.Background(), graph))
	}

	// The real value, injected via substitution, reached the backend.
	books := b.AllBooks()
	require.Len(t, books, 1)
	assert.Equal(t, "filled-in-access-key", books[0].AccessKey)
	assert.Equal(t, "the-hobbit", books[0].ExternalID)
}

// A non-secret field carrying a "{{ .VAR }}" reference must not read as drift.
// The import sync guard diffs the local graph against remote state, so import
// has to substitute exactly as apply does: apply sends the resolved value, and
// an unsubstituted token compares as a real (non-secret) change that blocks the
// import. Secrets are exempt from the guard, so only plain fields expose this.
func TestImportSyncGuardNeedsSubstitutedGraph(t *testing.T) {
	b := backend.NewBackend()
	testDir := t.TempDir()
	varFilePath := filepath.Join(testDir, "project.vars.yaml")

	writeProjectFile(t, filepath.Join(testDir, "writer", "tolkien.yaml"), `version: rudder/v1
kind: writer
metadata:
  name: common
spec:
  id: tolkien
  name: J.R.R. Tolkien
`)
	writeProjectFile(t, filepath.Join(testDir, "books", "books.yaml"), `version: rudder/v1
kind: books
metadata:
  name: my_books
spec:
  books:
    - id: "hobbit"
      name: "{{ .BOOK_NAME }}"
      author: "#writer:tolkien"
`)
	writeProjectFile(t, varFilePath, "BOOK_NAME: The Hobbit\n")

	loadProject := func(t *testing.T, withVarFile bool) (*example.Provider, project.Project) {
		t.Helper()

		var opts []project.ProjectOption
		if withVarFile {
			fileResolver, err := resolver.NewFileResolver(varFilePath)
			require.NoError(t, err)
			opts = append(opts, project.WithSubstitutor(varsubst.NewSubstitutor(fileResolver)))
		}

		p := example.NewProvider(b)
		proj := project.New(p, opts...)
		require.NoError(t, proj.Load(testDir))
		return p, proj
	}

	// Apply resolves the variable, so the backend holds the real name.
	applyProvider, applyProject := loadProject(t, true)
	graph, err := applyProject.ResourceGraph()
	require.NoError(t, err)
	s, err := syncer.New(applyProvider, &client.Workspace{ID: "test-workspace-id", Name: "Test Workspace"})
	require.NoError(t, err)
	require.NoError(t, s.Sync(context.Background(), graph))

	// An unmanaged remote resource, so the import has work to do past the guard.
	_, err = b.CreateWriter("George Orwell", "")
	require.NoError(t, err)

	// Without substitution the local graph still holds the literal token.
	noVarsProvider, noVarsProject := loadProject(t, false)
	err = importer.WorkspaceImport(context.Background(), noVarsProject, noVarsProvider, importer.ImportOptions{})
	require.ErrorIs(t, err, importer.ErrProjectNotSynced)

	// With substitution both sides agree and the import proceeds.
	varsProvider, varsProject := loadProject(t, true)
	require.NoError(t, importer.WorkspaceImport(context.Background(), varsProject, varsProvider, importer.ImportOptions{}))
	assert.DirExists(t, filepath.Join(testDir, importer.ImportedDir))
}

func writeProjectFile(t *testing.T, path, content string) {
	t.Helper()

	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0600))
}
