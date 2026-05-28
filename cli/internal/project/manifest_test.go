package project

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubProjectProvider is a minimal ProjectProvider that does not implement ImportManifestLoader.
type stubProjectProvider struct{}

func (s *stubProjectProvider) LoadSpec(_ string, _ *specs.Spec) error                        { return nil }
func (s *stubProjectProvider) LoadLegacySpec(_ string, _ *specs.Spec) error                  { return nil }
func (s *stubProjectProvider) ParseSpec(_ string, _ *specs.Spec) (*specs.ParsedSpec, error)  { return nil, nil }
func (s *stubProjectProvider) ResourceGraph() (*resources.Graph, error)                      { return nil, nil }
func (s *stubProjectProvider) SupportedKinds() []string                                      { return nil }
func (s *stubProjectProvider) SupportedTypes() []string                                      { return nil }
func (s *stubProjectProvider) SupportedMatchPatterns() []rules.MatchPattern                  { return nil }
func (s *stubProjectProvider) SyntacticRules() []rules.Rule                                  { return nil }
func (s *stubProjectProvider) SemanticRules() []rules.Rule                                   { return nil }

// manifestCapableProvider is a ProjectProvider that also implements ImportManifestLoader.
type manifestCapableProvider struct {
	stubProjectProvider
	manifest *specs.WorkspacesImportMetadata
}

func (m *manifestCapableProvider) LoadImportManifest(manifest *specs.WorkspacesImportMetadata) error {
	m.manifest = manifest
	return nil
}

var _ provider.ImportManifestLoader = (*manifestCapableProvider)(nil)

func TestSeparateManifests(t *testing.T) {
	t.Parallel()

	manifestSpec := &specs.RawSpec{Data: []byte("version: rudder/v1\nkind: import-manifest\nmetadata:\n  name: test\nspec:\n  workspaces: []\n")}
	_, _ = manifestSpec.Parse()

	resourceSpec := &specs.RawSpec{Data: []byte("version: rudder/v1\nkind: properties\nmetadata:\n  name: test\nspec:\n  properties: []\n")}
	_, _ = resourceSpec.Parse()

	rawSpecs := map[string]*specs.RawSpec{
		"manifest.yaml": manifestSpec,
		"props.yaml":    resourceSpec,
	}

	resourceSpecs, manifestSpecs := separateManifests(rawSpecs)

	assert.Len(t, resourceSpecs, 1)
	assert.Len(t, manifestSpecs, 1)
	assert.Contains(t, resourceSpecs, "props.yaml")
	assert.Contains(t, manifestSpecs, "manifest.yaml")
}

func TestDecodeManifestWorkspaces(t *testing.T) {
	t.Parallel()

	t.Run("valid input", func(t *testing.T) {
		t.Parallel()

		specMap := map[string]any{
			"workspaces": []any{
				map[string]any{
					"workspace_id": "ws-123",
					"resources": []any{
						map[string]any{
							"urn":       "source:my-src",
							"remote_id": "remote-456",
						},
					},
				},
			},
		}

		result, err := decodeManifestWorkspaces(specMap)

		require.NoError(t, err)
		assert.Equal(t, []specs.WorkspaceImportMetadata{
			{
				WorkspaceID: "ws-123",
				Resources: []specs.ImportIds{
					{URN: "source:my-src", RemoteID: "remote-456"},
				},
			},
		}, result)
	})

	t.Run("unknown fields rejected", func(t *testing.T) {
		t.Parallel()

		specMap := map[string]any{
			"workspaces": []any{
				map[string]any{
					"workspace_id":  "ws-123",
					"unknown_field": "bad",
				},
			},
		}

		_, err := decodeManifestWorkspaces(specMap)
		assert.Error(t, err)
	})
}

func TestParseManifests(t *testing.T) {
	t.Parallel()

	t.Run("empty manifests returns nil", func(t *testing.T) {
		t.Parallel()

		result, err := parseManifests(map[string]*specs.RawSpec{})

		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("multiple manifest files merged", func(t *testing.T) {
		t.Parallel()

		spec1 := &specs.RawSpec{Data: []byte("version: rudder/v1\nkind: import-manifest\nmetadata:\n  name: m1\nspec:\n  workspaces:\n    - workspace_id: ws-1\n      resources:\n        - urn: \"source:a\"\n          remote_id: r1\n")}
		_, _ = spec1.Parse()

		spec2 := &specs.RawSpec{Data: []byte("version: rudder/v1\nkind: import-manifest\nmetadata:\n  name: m2\nspec:\n  workspaces:\n    - workspace_id: ws-2\n      resources:\n        - urn: \"source:b\"\n          remote_id: r2\n")}
		_, _ = spec2.Parse()

		manifestSpecs := map[string]*specs.RawSpec{
			"m1.yaml": spec1,
			"m2.yaml": spec2,
		}

		result, err := parseManifests(manifestSpecs)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result.Workspaces, 2)
	})
}

func TestBroadcastImportManifest(t *testing.T) {
	t.Parallel()

	t.Run("nil manifest is a no-op", func(t *testing.T) {
		t.Parallel()

		err := broadcastImportManifest(&stubProjectProvider{}, nil)
		assert.NoError(t, err)
	})

	t.Run("provider without ImportManifestLoader is silently skipped", func(t *testing.T) {
		t.Parallel()

		manifest := &specs.WorkspacesImportMetadata{
			Workspaces: []specs.WorkspaceImportMetadata{
				{WorkspaceID: "ws-1"},
			},
		}

		err := broadcastImportManifest(&stubProjectProvider{}, manifest)
		assert.NoError(t, err)
	})

	t.Run("provider implementing ImportManifestLoader receives manifest", func(t *testing.T) {
		t.Parallel()

		manifest := &specs.WorkspacesImportMetadata{
			Workspaces: []specs.WorkspaceImportMetadata{
				{
					WorkspaceID: "ws-1",
					Resources: []specs.ImportIds{
						{URN: "source:a", RemoteID: "r1"},
					},
				},
			},
		}

		pp := &manifestCapableProvider{}
		err := broadcastImportManifest(pp, manifest)

		require.NoError(t, err)
		assert.Equal(t, manifest, pp.manifest)
	})
}

func TestCheckInlineManifestConflicts(t *testing.T) {
	t.Parallel()

	t.Run("no manifests returns nil", func(t *testing.T) {
		t.Parallel()

		diags := checkInlineManifestConflicts(
			map[string]*specs.RawSpec{},
			map[string]*specs.RawSpec{},
		)
		assert.Nil(t, diags)
	})

	t.Run("no conflicts returns nil", func(t *testing.T) {
		t.Parallel()

		manifestSpec := &specs.RawSpec{Data: []byte("version: rudder/v1\nkind: import-manifest\nmetadata:\n  name: m\nspec:\n  workspaces:\n    - workspace_id: ws-1\n      resources:\n        - urn: \"source:a\"\n          remote_id: r1\n")}
		_, _ = manifestSpec.Parse()

		resourceSpec := &specs.RawSpec{Data: []byte("version: rudder/v1\nkind: event-stream-source\nmetadata:\n  name: src\nspec:\n  id: my-source\n")}
		_, _ = resourceSpec.Parse()

		diags := checkInlineManifestConflicts(
			map[string]*specs.RawSpec{"source.yaml": resourceSpec},
			map[string]*specs.RawSpec{"manifest.yaml": manifestSpec},
		)
		assert.Empty(t, diags)
	})

	t.Run("conflict detected between manifest and inline metadata", func(t *testing.T) {
		t.Parallel()

		manifestSpec := &specs.RawSpec{Data: []byte("version: rudder/v1\nkind: import-manifest\nmetadata:\n  name: m\nspec:\n  workspaces:\n    - workspace_id: ws-1\n      resources:\n        - urn: \"source:shared\"\n          remote_id: r1\n")}
		_, _ = manifestSpec.Parse()

		resourceSpec := &specs.RawSpec{Data: []byte("version: rudder/v1\nkind: event-stream-source\nmetadata:\n  name: src\n  import:\n    workspaces:\n      - workspace_id: ws-1\n        resources:\n          - urn: \"source:shared\"\n            remote_id: r2\nspec:\n  id: my-source\n")}
		_, _ = resourceSpec.Parse()

		diags := checkInlineManifestConflicts(
			map[string]*specs.RawSpec{"source.yaml": resourceSpec},
			map[string]*specs.RawSpec{"manifest.yaml": manifestSpec},
		)

		require.Len(t, diags, 1)
		assert.Contains(t, diags[0].Message, "source:shared")
		assert.Equal(t, "project/import-manifest-inline-conflict", diags[0].RuleID)
	})
}
