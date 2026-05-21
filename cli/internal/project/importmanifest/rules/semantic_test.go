package rules

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	vrules "github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

func TestURNResolves_AllPresent(t *testing.T) {
	t.Parallel()

	rule := findRule(t, Semantic(), "import-manifest/urn-resolves")

	graph := resources.NewGraph()
	graph.AddResource(resources.NewResource("my-src", "source", resources.ResourceData{}, nil))

	ctx := &vrules.ValidationContext{
		FilePath: "manifest.yaml",
		FileName: "manifest.yaml",
		Kind:     specs.KindImportManifest,
		Version:  specs.SpecVersionV1,
		Spec:     validManifestSpec("ws-1", "source:my-src", "r-1"),
		Metadata: map[string]any{},
		Graph:    graph,
	}

	results := rule.Validate(ctx)

	assert.Empty(t, results)
}

func TestURNResolves_OrphanedURN(t *testing.T) {
	t.Parallel()

	rule := findRule(t, Semantic(), "import-manifest/urn-resolves")

	graph := resources.NewGraph()
	// Intentionally do not add "source:missing-src" to the graph.

	ctx := &vrules.ValidationContext{
		FilePath: "manifest.yaml",
		FileName: "manifest.yaml",
		Kind:     specs.KindImportManifest,
		Version:  specs.SpecVersionV1,
		Spec:     validManifestSpec("ws-1", "source:missing-src", "r-1"),
		Metadata: map[string]any{},
		Graph:    graph,
	}

	results := rule.Validate(ctx)

	require.Len(t, results, 1)
	assert.Equal(t, "/spec/workspaces/0/resources/0/urn", results[0].Reference)
	assert.Contains(t, results[0].Message, "source:missing-src")
}

func TestURNResolves_NilGraph(t *testing.T) {
	t.Parallel()

	rule := findRule(t, Semantic(), "import-manifest/urn-resolves")

	ctx := &vrules.ValidationContext{
		FilePath: "manifest.yaml",
		FileName: "manifest.yaml",
		Kind:     specs.KindImportManifest,
		Version:  specs.SpecVersionV1,
		Spec:     validManifestSpec("ws-1", "source:my-src", "r-1"),
		Metadata: map[string]any{},
		Graph:    nil,
	}

	results := rule.Validate(ctx)

	assert.Nil(t, results)
}
