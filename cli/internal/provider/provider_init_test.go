package provider_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/importer"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/testutils/example"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/testutils/example/backend"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/differ"
	"github.com/rudderlabs/rudder-iac/cli/internal/varsubst"
	varresolver "github.com/rudderlabs/rudder-iac/cli/internal/varsubst/resolver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// exampleWorkspaceID is the workspace the example backend stamps onto the
// import metadata it exports.
const exampleWorkspaceID = "test-workspace-id"

// TestExampleInit_ClonesManagedAndUnmanaged is the acceptance test for `init`:
// a workspace holding both managed and unmanaged resources is written into an
// empty directory, and applying the project that comes out changes nothing that
// is already managed — the managed resources keep their upstream externalIds,
// so they land in the diff as unmodified rather than as renames or creates.
func TestExampleInit_ClonesManagedAndUnmanaged(t *testing.T) {
	t.Parallel()

	b := backend.NewBackend()

	// Managed upstream: these carry an externalId and must keep it verbatim.
	managedWriter, err := b.CreateWriter("Ursula Le Guin", "le-guin")
	require.NoError(t, err)
	_, err = b.CreateBook("A Wizard of Earthsea", managedWriter.ID, "earthsea", "earthsea-key")
	require.NoError(t, err)

	// Unmanaged upstream: no externalId, so init generates one from the name.
	unmanagedWriter, err := b.CreateWriter("Terry Pratchett", "")
	require.NoError(t, err)
	_, err = b.CreateBook("Small Gods", unmanagedWriter.ID, "", "small-gods-key")
	require.NoError(t, err)

	dir := t.TempDir()
	require.NoError(t, importer.WorkspaceInit(context.Background(), dir, example.NewProvider(b)))

	// The managed writer kept its identifier; the unmanaged one was named from
	// its display name.
	assert.FileExists(t, filepath.Join(dir, "writers", "le-guin.yaml"))
	assert.FileExists(t, filepath.Join(dir, "writers", "terry-pratchett.yaml"))

	// Books reference their author through the same clone, not through a local
	// project that does not exist yet.
	books := readFile(t, filepath.Join(dir, "books", "books.yaml"))
	assert.Contains(t, books, `author: "#writer:le-guin"`)
	assert.Contains(t, books, `author: "#writer:terry-pratchett"`)

	// Nothing is wrapped in imported/ — the clone is the project.
	_, err = os.Stat(filepath.Join(dir, importer.ImportedDir))
	assert.True(t, os.IsNotExist(err))

	// Secrets are not readable from remote, so they come out as variable
	// references plus a var file to fill in — same contract as import.
	varFile := filepath.Join(dir, importer.SecretsVarFileName)
	require.FileExists(t, varFile)
	writeProjectFile(t, varFile, "BOOK_EARTHSEA_ACCESS_KEY: earthsea-key\nBOOK_SMALL_GODS_ACCESS_KEY: small-gods-key\n")

	// The clone loads as a project, and diffing it against the very workspace
	// it came from shows no drift on the managed resources.
	fileResolver, err := varresolver.NewFileResolver(varFile)
	require.NoError(t, err)

	p := example.NewProvider(b)
	proj := project.New(p, project.WithSubstitutor(varsubst.NewSubstitutor(fileResolver)))
	require.NoError(t, proj.Load(dir))

	targetGraph, err := proj.ResourceGraph()
	require.NoError(t, err)

	remote, err := p.LoadResourcesFromRemote(context.Background())
	require.NoError(t, err)
	st, err := p.MapRemoteToState(remote)
	require.NoError(t, err)

	// The workspace ID is what lets the differ honour the specs' import
	// metadata — without it every adopted resource reads as a create.
	diff := differ.ComputeDiff(syncer.StateToGraph(st), targetGraph, differ.DiffOptions{
		WorkspaceID: exampleWorkspaceID,
	})

	assert.Empty(t, diff.RemovedResources, "a clone must never look like a deletion")
	assert.Empty(t, diff.NewResources, "every resource in a clone exists upstream, so none is a create")

	// The previously-unmanaged resources carry import metadata, so they are
	// adopted rather than duplicated.
	assert.ElementsMatch(t,
		[]string{"example-writer:terry-pratchett", "example-book:small-gods"},
		diff.ImportableResources,
	)

	// The managed writer is byte-identical: no rename, no update.
	assert.Contains(t, diff.UnmodifiedResources, "example-writer:le-guin")
	assert.NotContains(t, diff.UpdatedResources, "example-writer:le-guin")
}

func TestExampleInit_RefusesNonEmptyProject(t *testing.T) {
	t.Parallel()

	b := backend.NewBackend()
	_, err := b.CreateWriter("Terry Pratchett", "")
	require.NoError(t, err)

	dir := t.TempDir()
	writeProjectFile(t, filepath.Join(dir, "writers.yaml"), "version: rudder/v1\nkind: writers\n")

	err = importer.WorkspaceInit(context.Background(), dir, example.NewProvider(b))
	require.ErrorIs(t, err, importer.ErrProjectNotEmpty)
}

func readFile(t *testing.T, path string) string {
	t.Helper()

	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(raw)
}
