package provider_test

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/importer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/importmanifest"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/testutils/example"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/testutils/example/backend"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/planner"
	"github.com/rudderlabs/rudder-iac/cli/internal/varsubst"
	varresolver "github.com/rudderlabs/rudder-iac/cli/internal/varsubst/resolver"
)

// The acceptance test for reseed: write a whole workspace into an empty
// directory, then ask for the plan apply --dry-run would print against that
// same workspace. Nothing may be created, updated or deleted. Resources the
// workspace did not have under management are adopted instead, which is what
// the import-manifest reseed writes is for, so those are the only entries the
// plan may hold.

// enableReseedFlags turns on the experimental flags reseed needs: importMerge
// makes the import-manifest kind load, without which a reseeded project
// holding one fails to load at all.
func enableReseedFlags(t *testing.T) {
	t.Helper()
	// The composite provider reads its task concurrency at construction, and
	// without the defaults InitConfig registers it is zero, which parks the
	// worker pool forever.
	config.InitConfig(filepath.Join(t.TempDir(), "config.json"))

	prev := map[string]any{
		"experimental":        viper.Get("experimental"),
		"flags.importMerge":   viper.Get("flags.importMerge"),
		"flags.importManaged": viper.Get("flags.importManaged"),
	}
	viper.Set("experimental", true)
	viper.Set("flags.importMerge", true)
	viper.Set("flags.importManaged", true)
	t.Cleanup(func() {
		for k, v := range prev {
			viper.Set(k, v)
		}
	})
}

// compositeOf wraps the example provider the way the app does, so reseed goes
// through the same aggregation (and the same LoadManaged coverage report) as a
// real run.
func compositeOf(t *testing.T, p provider.Provider) importer.ReseedProvider {
	t.Helper()
	composite, err := provider.NewCompositeProvider(map[string]provider.Provider{"example": p})
	require.NoError(t, err)

	reseedable, ok := composite.(importer.ReseedProvider)
	require.True(t, ok, "the composite provider must be able to reseed")
	return reseedable
}

// reseedAndPlan reseeds the backend's workspace into an empty directory, then
// returns the plan apply would make for the result.
func reseedAndPlan(t *testing.T, b *backend.Backend) *planner.Plan {
	t.Helper()
	ctx := context.Background()
	dir := t.TempDir()

	require.NoError(t, importer.Reseed(ctx, dir, compositeOf(t, example.NewProvider(b))))

	// Secrets never come back from the workspace: reseed writes a '{{ .VAR }}'
	// reference and a var file to fill. Apply reads the filled file, so the
	// plan can only be judged with the same values it would use.
	opts := []project.ProjectOption{project.WithWorkspaceID(backend.WorkspaceID)}
	if varFile := fillSecretsVarFile(t, dir); varFile != "" {
		fileResolver, err := varresolver.NewFileResolver(varFile)
		require.NoError(t, err)
		opts = append(opts, project.WithSubstitutor(varsubst.NewSubstitutor(fileResolver)))
	}

	applyProvider := example.NewProvider(b)
	proj := project.New(applyProvider, opts...)
	require.NoError(t, proj.Load(dir), "a reseeded project must load")

	target, err := proj.ResourceGraph()
	require.NoError(t, err)

	remote, err := applyProvider.LoadResourcesFromRemote(ctx)
	require.NoError(t, err)
	state, err := applyProvider.MapRemoteToState(remote)
	require.NoError(t, err)

	return planner.New(backend.WorkspaceID).Plan(syncer.StateToGraph(state), target)
}

// fillSecretsVarFile gives every scaffolded placeholder the value the backend
// already holds, so a secret cannot be the reason the plan is non-empty. It
// returns "" when reseed scaffolded no var file.
func fillSecretsVarFile(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, importer.SecretsVarFileName)
	if _, err := os.Stat(path); err != nil {
		return ""
	}

	// One book, one secret in these fixtures; the name encodes the book id.
	body := ""
	for _, book := range []string{"THE_HOBBIT", "THE_SILMARILLION"} {
		body += "BOOK_" + book + "_ACCESS_KEY: " + bookSecret + "\n"
	}
	require.NoError(t, os.WriteFile(path, []byte(body), 0600))
	return path
}

// The value the backend holds in a book's secret field: a fixture, not a
// credential.
const bookSecret = "remote-only-access-key" //nolint:gosec

// assertAdoptsOnly fails unless the plan creates, updates and deletes nothing,
// and adopts exactly wantAdopted. A secret-only update is exempt: an unknown
// secret re-applies on every run by design, so it is not reseed drift.
func assertAdoptsOnly(t *testing.T, plan *planner.Plan, wantAdopted ...string) {
	t.Helper()
	diff := plan.Diff

	assert.Empty(t, diff.NewResources, "reseed must not leave anything to create")
	assert.Empty(t, diff.RemovedResources, "reseed must not leave anything to delete")
	for urn, rd := range diff.UpdatedResources {
		assert.True(t, rd.IsSecretOnly(), "reseed must not leave %s to update: %+v", urn, rd)
	}

	adopted := append([]string(nil), diff.ImportableResources...)
	sort.Strings(adopted)
	sort.Strings(wantAdopted)
	assert.Equal(t, wantAdopted, adopted)
}

func TestReseedLeavesAnEmptyPlan(t *testing.T) {
	enableReseedFlags(t)

	t.Run("fully managed workspace", func(t *testing.T) {
		b := backend.NewBackend()
		w, err := b.CreateWriter("J.R.R. Tolkien", "tolkien")
		require.NoError(t, err)
		_, err = b.CreateBook("The Hobbit", w.ID, "the-hobbit", bookSecret)
		require.NoError(t, err)

		// Every resource carries an externalId, so every spec reseed writes is
		// keyed by it and matches the resource already there: nothing to adopt.
		assertAdoptsOnly(t, reseedAndPlan(t, b))
	})

	t.Run("fully unmanaged workspace", func(t *testing.T) {
		b := backend.NewBackend()
		w, err := b.CreateWriter("J.R.R. Tolkien", "")
		require.NoError(t, err)
		_, err = b.CreateBook("The Hobbit", w.ID, "", bookSecret)
		require.NoError(t, err)

		// Neither resource is managed, so reseed names both from their remote
		// name and the manifest it writes has apply adopt them.
		assertAdoptsOnly(t, reseedAndPlan(t, b),
			"example-writer:j-r-r-tolkien", "example-book:the-hobbit")
	})

	t.Run("mixed workspace reseeds in one pass", func(t *testing.T) {
		b := backend.NewBackend()
		// A managed writer with an unmanaged book of theirs: the book's spec has
		// to reference the managed writer by the id reseed gave that writer.
		managed, err := b.CreateWriter("J.R.R. Tolkien", "tolkien")
		require.NoError(t, err)
		_, err = b.CreateBook("The Hobbit", managed.ID, "the-hobbit", bookSecret)
		require.NoError(t, err)
		_, err = b.CreateBook("The Silmarillion", managed.ID, "", bookSecret)
		require.NoError(t, err)

		assertAdoptsOnly(t, reseedAndPlan(t, b), "example-book:the-silmarillion")
	})
}

// The manifest reseed writes must name only the resources that still need
// adopting. An entry for a managed resource claims it is unmanaged.
func TestReseedManifestListsOnlyUnmanagedResources(t *testing.T) {
	enableReseedFlags(t)

	b := backend.NewBackend()
	managed, err := b.CreateWriter("J.R.R. Tolkien", "tolkien")
	require.NoError(t, err)
	_, err = b.CreateBook("The Hobbit", managed.ID, "the-hobbit", bookSecret)
	require.NoError(t, err)
	_, err = b.CreateBook("The Silmarillion", managed.ID, "", bookSecret)
	require.NoError(t, err)

	dir := t.TempDir()
	require.NoError(t, importer.Reseed(context.Background(), dir, compositeOf(t, example.NewProvider(b))))

	body, err := os.ReadFile(filepath.Join(dir, importmanifest.FileName))
	require.NoError(t, err)

	assert.Contains(t, string(body), "example-book:the-silmarillion")
	assert.NotContains(t, string(body), "example-book:the-hobbit")
	assert.NotContains(t, string(body), "example-writer:tolkien")
}

// A generated id for an unmanaged resource must not land on a managed
// resource's externalId, or the two specs claim the same resource and one of
// them is wrong. Two writers with the same name force the collision.
func TestReseedNamesUnmanagedAroundManagedIDs(t *testing.T) {
	enableReseedFlags(t)

	b := backend.NewBackend()
	// The managed writer holds exactly the id the namer would give the
	// unmanaged one from its name.
	_, err := b.CreateWriter("Someone Else", "j-r-r-tolkien")
	require.NoError(t, err)
	_, err = b.CreateWriter("J.R.R. Tolkien", "")
	require.NoError(t, err)

	plan := reseedAndPlan(t, b)

	assert.Empty(t, plan.Diff.NewResources)
	assert.Empty(t, plan.Diff.RemovedResources)
	require.Len(t, plan.Diff.ImportableResources, 1)
	assert.NotEqual(t, "example-writer:j-r-r-tolkien", plan.Diff.ImportableResources[0],
		"the unmanaged writer took the managed writer's id")
}

func TestReseedRefusesADirectoryWithAProject(t *testing.T) {
	enableReseedFlags(t)

	b := backend.NewBackend()
	_, err := b.CreateWriter("J.R.R. Tolkien", "tolkien")
	require.NoError(t, err)

	dir := t.TempDir()
	writeProjectFile(t, filepath.Join(dir, "writer", "tolkien.yaml"), `version: rudder/v1
kind: writer
metadata:
  name: common
spec:
  id: tolkien
  name: J.R.R. Tolkien
`)

	err = importer.Reseed(context.Background(), dir, compositeOf(t, example.NewProvider(b)))

	require.ErrorIs(t, err, importer.ErrProjectPresent)
	assert.Contains(t, err.Error(), "tolkien.yaml")
	assert.Contains(t, err.Error(), "empty directory")
}

// A var file is not a project spec, so it must not block a reseed.
func TestReseedAllowsAVarFileInTheDirectory(t *testing.T) {
	enableReseedFlags(t)

	b := backend.NewBackend()
	_, err := b.CreateWriter("J.R.R. Tolkien", "tolkien")
	require.NoError(t, err)

	dir := t.TempDir()
	writeProjectFile(t, filepath.Join(dir, "prod.vars.yaml"), "SOME_KEY: value\n")

	require.NoError(t, importer.Reseed(context.Background(), dir, compositeOf(t, example.NewProvider(b))))
}

// typeWithoutHandler is a provider whose resource type nothing can export.
// Reseeding past it would write a project missing that type, and the next apply
// would read the absence as a deletion.
type typeWithoutHandler struct {
	provider.Provider
}

func (typeWithoutHandler) SupportedTypes() []string { return []string{"profiles"} }
func (typeWithoutHandler) SupportedKinds() []string { return []string{"profile"} }

func TestReseedRefusesATypeNothingCanExport(t *testing.T) {
	enableReseedFlags(t)

	b := backend.NewBackend()
	_, err := b.CreateWriter("J.R.R. Tolkien", "tolkien")
	require.NoError(t, err)

	composite, err := provider.NewCompositeProvider(map[string]provider.Provider{
		"example":  example.NewProvider(b),
		"profiles": typeWithoutHandler{},
	})
	require.NoError(t, err)
	reseedable, ok := composite.(importer.ReseedProvider)
	require.True(t, ok)

	dir := t.TempDir()
	err = importer.Reseed(context.Background(), dir, reseedable)

	require.ErrorIs(t, err, importer.ErrTypesWithoutHandler)
	assert.Contains(t, err.Error(), "profiles", "the error must name the type it cannot write")

	entries, readErr := os.ReadDir(dir)
	require.NoError(t, readErr)
	assert.Empty(t, entries, "a refused reseed must write nothing")
}

func TestReseedOnAnEmptyWorkspaceWritesNothing(t *testing.T) {
	enableReseedFlags(t)

	dir := t.TempDir()
	require.NoError(t, importer.Reseed(context.Background(), dir,
		compositeOf(t, example.NewProvider(backend.NewBackend()))))

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

// Guard against the collection helper reseed relies on silently changing shape.
func TestRemoteResourcesTypes(t *testing.T) {
	c := resources.NewRemoteResources()
	c.Set("example-writer", map[string]*resources.RemoteResource{"rid-1": {ID: "rid-1"}})

	assert.Equal(t, []string{"example-writer"}, c.Types())
}
