package datagraph_test

import (
	"testing"

	dgClient "github.com/rudderlabs/rudder-iac/api/client/datagraph"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/importmanifest"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph"
	dgModel "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/testutils"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A matched data graph adopts the existing local composite YAML as the source
// of truth: manifest entries only (graph + matched children), no spec file.
// Upstream-only children are dropped — never linked, they stay unmanaged
// upstream and untouched by apply.
func TestFormatForExport_MatchedGraphEmitsEntriesOnly(t *testing.T) {
	t.Parallel()

	provider := datagraph.NewProvider(&testutils.MockDataGraphClient{}, nil)

	matchedGraph := &resources.RemoteResource{
		ID:         "dg-remote-1",
		ExternalID: "main-graph", // adopted local identity
		Data: &dgModel.RemoteDataGraph{
			DataGraph: &dgClient.DataGraph{ID: "dg-remote-1", WorkspaceID: "ws-123", AccountID: "acc-1"},
		},
		MatchedWith: localDataGraph("main-graph", "acc-1"),
	}

	matchedModel := &resources.RemoteResource{
		ID:         "model-remote-1",
		ExternalID: "users", // adopted local identity
		Data: &dgModel.RemoteModel{
			Model: &dgClient.Model{ID: "model-remote-1", Name: "Users", DataGraphID: "dg-remote-1"},
		},
		MatchedWith: localModel("users", "Users", "main-graph"),
	}

	// Upstream-only model: no local counterpart, stays unlinked.
	upstreamOnlyModel := &resources.RemoteResource{
		ID:         "model-remote-2",
		ExternalID: "orders",
		Data: &dgModel.RemoteModel{
			Model: &dgClient.Model{ID: "model-remote-2", Name: "Orders", DataGraphID: "dg-remote-1"},
		},
	}

	matchedRel := &resources.RemoteResource{
		ID:         "rel-remote-1",
		ExternalID: "user-orders", // adopted local identity
		Data: &dgModel.RemoteRelationship{
			Relationship: &dgClient.Relationship{
				ID: "rel-remote-1", Name: "User Orders", DataGraphID: "dg-remote-1", SourceModelID: "model-remote-1",
			},
		},
		MatchedWith: localRelationship("user-orders", "User Orders", "main-graph"),
	}

	collection := buildRemoteResources(
		map[string]*resources.RemoteResource{"dg-remote-1": matchedGraph},
		map[string]*resources.RemoteResource{"model-remote-1": matchedModel, "model-remote-2": upstreamOnlyModel},
		map[string]*resources.RemoteResource{"rel-remote-1": matchedRel},
	)

	entities, entries, err := provider.FormatForExport(collection, nil, nil)
	require.NoError(t, err)

	assert.Empty(t, entities, "matched graph must not write a spec — the local composite YAML is authoritative")
	assert.ElementsMatch(t, []importmanifest.ImportEntry{
		{WorkspaceID: "ws-123", URN: "data-graph:main-graph", RemoteID: "dg-remote-1"},
		{WorkspaceID: "ws-123", URN: "data-graph-model:users", RemoteID: "model-remote-1"},
		{WorkspaceID: "ws-123", URN: "data-graph-relationship:user-orders", RemoteID: "rel-remote-1"},
	}, entries, "entries for graph + matched children only; upstream-only children dropped")
}

func TestFormatForExport_MatchedAndUnmatchedGraphsCoexist(t *testing.T) {
	t.Parallel()

	provider := datagraph.NewProvider(&testutils.MockDataGraphClient{}, nil)

	matchedGraph := &resources.RemoteResource{
		ID:         "dg-remote-1",
		ExternalID: "main-graph",
		Data: &dgModel.RemoteDataGraph{
			DataGraph: &dgClient.DataGraph{ID: "dg-remote-1", WorkspaceID: "ws-123", AccountID: "acc-1"},
		},
		MatchedWith: localDataGraph("main-graph", "acc-1"),
	}

	newGraph := &resources.RemoteResource{
		ID:         "dg-remote-2",
		ExternalID: "staging-graph",
		Data: &dgModel.RemoteDataGraph{
			DataGraph: &dgClient.DataGraph{ID: "dg-remote-2", WorkspaceID: "ws-123", AccountID: "acc-2"},
		},
	}

	collection := buildRemoteResources(
		map[string]*resources.RemoteResource{"dg-remote-1": matchedGraph, "dg-remote-2": newGraph},
		nil,
		nil,
	)

	entities, entries, err := provider.FormatForExport(collection, nil, nil)
	require.NoError(t, err)

	// Only the unmatched graph writes a spec.
	require.Len(t, entities, 1)
	assert.Contains(t, entities[0].RelativePath, "staging-graph.yaml")

	assert.ElementsMatch(t, []importmanifest.ImportEntry{
		{WorkspaceID: "ws-123", URN: "data-graph:main-graph", RemoteID: "dg-remote-1"},
		{WorkspaceID: "ws-123", URN: "data-graph:staging-graph", RemoteID: "dg-remote-2"},
	}, entries)
}
