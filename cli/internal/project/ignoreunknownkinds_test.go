package project_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/testutils"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/renderer"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// A consumer registering only the Source provider still has to cope with
// projects whose directories hold specs owned by other providers (rudder-data-gov
// keeps data-graphs and transformations next to its data catalog). Callers that
// only read a subset of the project opt into skipping those via
// WithIgnoreUnknownKinds instead of failing the whole load.
func TestProject_Load_UnownedKind(t *testing.T) {
	t.Parallel()

	const (
		sourceYAML    = "kind: Source\nversion: rudder/v1\nmetadata:\n  name: my_source\nspec:\n  k: v"
		dataGraphYAML = "kind: data-graph\nversion: rudder/v1\nmetadata:\n  name: my_graph\nspec:\n  k: v"
	)

	newProject := func(opts ...project.ProjectOption) project.Project {
		mockProvider := testutils.NewMockProvider(nil, nil)
		mockProvider.MatchPatterns = []rules.MatchPattern{
			rules.MatchKindVersion("Source", specs.SpecVersionV1),
		}
		mockLoader := &MockLoader{LoadFunc: func(string) (map[string]*specs.RawSpec, error) {
			return map[string]*specs.RawSpec{
				"source.yaml":     {Data: []byte(sourceYAML)},
				"data-graph.yaml": {Data: []byte(dataGraphYAML)},
			}, nil
		}}

		opts = append([]project.ProjectOption{
			project.WithLoader(mockLoader),
			project.WithRenderer(renderer.NewTextRenderer(&bytes.Buffer{})),
		}, opts...)

		return project.New(mockProvider, opts...)
	}

	t.Run("fails syntax validation by default", func(t *testing.T) {
		t.Parallel()

		err := newProject().Load("test_dir")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "syntax validation failed")
	})

	t.Run("is skipped with WithIgnoreUnknownKinds, and owned kinds still load", func(t *testing.T) {
		t.Parallel()

		proj := newProject(project.WithIgnoreUnknownKinds())

		require.NoError(t, proj.Load("test_dir"))
		assert.Equal(t, []string{"source.yaml"}, specPaths(proj))
	})
}

// import-manifest is owned by the always-present import-manifest provider, but
// BuildRegistry treats it as a known kind only when the ImportMerge experimental
// flag is on. WithIgnoreUnknownKinds must mirror that exactly: skip a stray
// import-manifest when the flag is off (a real project can carry one), keep it
// when the flag is on. Regression guard for the #671<->#673 interaction — before
// isKnownKind became flag-aware it treated import-manifest as always-known, so
// the flag-off case failed syntax validation instead of being skipped.
func TestProject_Load_ImportManifestKind_RespectsImportMergeFlag(t *testing.T) {
	const (
		sourceYAML   = "kind: Source\nversion: rudder/v1\nmetadata:\n  name: my_source\nspec:\n  k: v"
		manifestYAML = `version: rudder/v1
kind: import-manifest
metadata:
  name: import-manifest
spec:
  workspaces:
    - workspace_id: ws-1
      resources:
        - urn: "event-stream-source:my-src"
          remote_id: rid-1
`
	)

	newProject := func(opts ...project.ProjectOption) project.Project {
		mockProvider := testutils.NewMockProvider(nil, nil)
		mockProvider.MatchPatterns = []rules.MatchPattern{
			rules.MatchKindVersion("Source", specs.SpecVersionV1),
		}
		mockLoader := &MockLoader{LoadFunc: func(string) (map[string]*specs.RawSpec, error) {
			return map[string]*specs.RawSpec{
				"source.yaml":          {Data: []byte(sourceYAML)},
				"import-manifest.yaml": {Data: []byte(manifestYAML)},
			}, nil
		}}

		opts = append([]project.ProjectOption{
			project.WithLoader(mockLoader),
			project.WithRenderer(renderer.NewTextRenderer(&bytes.Buffer{})),
		}, opts...)

		return project.New(mockProvider, opts...)
	}

	// Toggles global viper state, so this test cannot run in parallel.
	t.Run("flag off: stray import-manifest is skipped, not rejected", func(t *testing.T) {
		proj := newProject(project.WithIgnoreUnknownKinds())

		require.NoError(t, proj.Load("test_dir"))
		assert.Equal(t, []string{"source.yaml"}, specPaths(proj))
	})

	t.Run("flag on: import-manifest is recognized, not skipped", func(t *testing.T) {
		enableImportMerge(t)

		proj := newProject(project.WithIgnoreUnknownKinds())

		require.NoError(t, proj.Load("test_dir"))
		assert.ElementsMatch(t, []string{"source.yaml", "import-manifest.yaml"}, specPaths(proj))
	})
}

func specPaths(p project.Project) []string {
	paths := make([]string, 0, len(p.Specs()))
	for path := range p.Specs() {
		paths = append(paths, path)
	}
	return paths
}
