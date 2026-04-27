package datagraph_test

import (
	"fmt"
	"testing"

	dgClient "github.com/rudderlabs/rudder-iac/api/client/datagraph"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph"
	dgHandler "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/handlers/datagraph"
	modelHandler "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/handlers/model"
	relationshipHandler "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/handlers/relationship"
	dgModel "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/testutils"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildRemoteResources is a helper to build a RemoteResources collection for FormatForExport tests
func buildRemoteResources(
	dataGraphs map[string]*resources.RemoteResource,
	models map[string]*resources.RemoteResource,
	relationships map[string]*resources.RemoteResource,
) *resources.RemoteResources {
	collection := resources.NewRemoteResources()
	if len(dataGraphs) > 0 {
		collection.Set(dgHandler.HandlerMetadata.ResourceType, dataGraphs)
	}
	if len(models) > 0 {
		collection.Set(modelHandler.HandlerMetadata.ResourceType, models)
	}
	if len(relationships) > 0 {
		collection.Set(relationshipHandler.HandlerMetadata.ResourceType, relationships)
	}
	return collection
}

func TestFormatForExport_FullCompositeExport(t *testing.T) {
	mockClient := &testutils.MockDataGraphClient{}
	provider := datagraph.NewProvider(mockClient, nil)

	collection := buildRemoteResources(
		map[string]*resources.RemoteResource{
			"dg-remote-1": {
				ID:         "dg-remote-1",
				ExternalID: "my-data-graph",
				Data: &dgModel.RemoteDataGraph{
					DataGraph: &dgClient.DataGraph{
						ID:          "dg-remote-1",
						WorkspaceID: "ws-123",
						AccountID:   "account-1",
					},
					AccountName: "My Warehouse",
				},
			},
		},
		map[string]*resources.RemoteResource{
			"model-remote-1": {
				ID:         "model-remote-1",
				ExternalID: "user",
				Data: &dgModel.RemoteModel{
					Model: &dgClient.Model{
						ID:          "model-remote-1",
						Name:        "User",
						Type:        "entity",
						TableRef:    "db.schema.users",
						DataGraphID: "dg-remote-1",
						PrimaryID:   "user_id",
						Root:        true,
					},
				},
			},
			"model-remote-2": {
				ID:         "model-remote-2",
				ExternalID: "purchase",
				Data: &dgModel.RemoteModel{
					Model: &dgClient.Model{
						ID:          "model-remote-2",
						Name:        "Purchase",
						Type:        "event",
						TableRef:    "db.schema.purchases",
						DataGraphID: "dg-remote-1",
						Timestamp:   "purchased_at",
					},
				},
			},
		},
		map[string]*resources.RemoteResource{
			"rel-remote-1": {
				ID:         "rel-remote-1",
				ExternalID: "purchase-user",
				Data: &dgModel.RemoteRelationship{
					Relationship: &dgClient.Relationship{
						ID:            "rel-remote-1",
						Name:          "Purchase User",
						Cardinality:   "many-to-one",
						SourceModelID: "model-remote-2",
						TargetModelID: "model-remote-1",
						DataGraphID:   "dg-remote-1",
						SourceJoinKey: "user_id",
						TargetJoinKey: "user_id",
					},
				},
			},
			// Relationship whose target model is NOT in the importable collection —
			// should be silently skipped.
			"rel-remote-2": {
				ID:         "rel-remote-2",
				ExternalID: "user-order",
				Data: &dgModel.RemoteRelationship{
					Relationship: &dgClient.Relationship{
						ID:            "rel-remote-2",
						Name:          "User Order",
						Cardinality:   "one-to-many",
						SourceModelID: "model-remote-1",
						TargetModelID: "model-remote-99", // Not in importable collection
						DataGraphID:   "dg-remote-1",
						SourceJoinKey: "user_id",
						TargetJoinKey: "user_id",
					},
				},
			},
		},
	)

	result, entries, err := provider.FormatForExport(collection, nil, nil)
	require.NoError(t, err)
	require.Len(t, result, 1)

	entity := result[0]
	assert.Equal(t, "data-graphs/my-data-graph.yaml", entity.RelativePath)

	spec, ok := entity.Content.(*specs.Spec)
	require.True(t, ok)
	assert.Equal(t, specs.SpecVersionV1, spec.Version)
	assert.Equal(t, "data-graph", spec.Kind)

	// Verify spec body
	assert.Equal(t, "my-data-graph", spec.Spec["id"])
	assert.Equal(t, "account-1", spec.Spec["account_id"])

	// Verify models are present
	models, ok := spec.Spec["models"].([]dgModel.ModelSpec)
	require.True(t, ok)
	require.Len(t, models, 2)

	// Import metadata travels via the ImportEntry slice — no inline metadata.import.
	metadata, err := spec.CommonMetadata()
	require.NoError(t, err)
	assert.Nil(t, metadata.Import, "emitted specs must not carry inline metadata.import")

	// Should have 4 entries: 1 DG + 2 models + 1 valid relationship
	// (the unresolvable-target relationship is excluded)
	require.Len(t, entries, 4)

	urns := make(map[string]string) // URN -> RemoteID
	for _, e := range entries {
		assert.Equal(t, "ws-123", e.WorkspaceID)
		urns[e.URN] = e.RemoteID
	}
	assert.Equal(t, "dg-remote-1", urns[resources.URN("my-data-graph", dgHandler.HandlerMetadata.ResourceType)])
	assert.Equal(t, "model-remote-1", urns[resources.URN("user", modelHandler.HandlerMetadata.ResourceType)])
	assert.Equal(t, "model-remote-2", urns[resources.URN("purchase", modelHandler.HandlerMetadata.ResourceType)])
	assert.Equal(t, "rel-remote-1", urns[resources.URN("purchase-user", relationshipHandler.HandlerMetadata.ResourceType)])

	// Unresolvable-target relationship should NOT appear
	assert.Empty(t, urns[resources.URN("user-order", relationshipHandler.HandlerMetadata.ResourceType)])

	// Verify relationship target reference is resolved correctly
	var purchaseModel, userModel dgModel.ModelSpec
	for _, m := range models {
		switch m.ID {
		case "purchase":
			purchaseModel = m
		case "user":
			userModel = m
		}
	}
	require.Len(t, purchaseModel.Relationships, 1)
	assert.Equal(t, "#data-graph-model:user", purchaseModel.Relationships[0].Target)

	// User model's unresolvable relationship should have been skipped
	assert.Empty(t, userModel.Relationships)
}

func TestFormatForExport_DataGraphWithoutModels(t *testing.T) {
	mockClient := &testutils.MockDataGraphClient{}
	provider := datagraph.NewProvider(mockClient, nil)

	collection := buildRemoteResources(
		map[string]*resources.RemoteResource{
			"dg-remote-1": {
				ID:         "dg-remote-1",
				ExternalID: "simple-dg",
				Data: &dgModel.RemoteDataGraph{
					DataGraph: &dgClient.DataGraph{
						ID:          "dg-remote-1",
						WorkspaceID: "ws-123",
						AccountID:   "account-1",
					},
				},
			},
		},
		nil, nil,
	)

	result, entries, err := provider.FormatForExport(collection, nil, nil)
	require.NoError(t, err)
	require.Len(t, result, 1)

	spec := result[0].Content.(*specs.Spec)
	assert.Equal(t, "simple-dg", spec.Spec["id"])
	assert.Nil(t, spec.Spec["models"])
	assert.Equal(t, "data-graphs/simple-dg.yaml", result[0].RelativePath)

	// Import entries have only the data graph
	require.Len(t, entries, 1)
	assert.Equal(t, "ws-123", entries[0].WorkspaceID)
	assert.Equal(t, resources.URN("simple-dg", dgHandler.HandlerMetadata.ResourceType), entries[0].URN)
}

func TestFormatForExport_MultipleDataGraphs(t *testing.T) {
	mockClient := &testutils.MockDataGraphClient{}
	provider := datagraph.NewProvider(mockClient, nil)

	collection := buildRemoteResources(
		map[string]*resources.RemoteResource{
			"dg-remote-1": {
				ID:         "dg-remote-1",
				ExternalID: "dg-one",
				Data: &dgModel.RemoteDataGraph{
					DataGraph: &dgClient.DataGraph{
						ID:          "dg-remote-1",
						WorkspaceID: "ws-123",
						AccountID:   "account-1",
					},
				},
			},
			"dg-remote-2": {
				ID:         "dg-remote-2",
				ExternalID: "dg-two",
				Data: &dgModel.RemoteDataGraph{
					DataGraph: &dgClient.DataGraph{
						ID:          "dg-remote-2",
						WorkspaceID: "ws-456",
						AccountID:   "account-2",
					},
				},
			},
		},
		nil, nil,
	)

	result, _, err := provider.FormatForExport(collection, nil, nil)
	require.NoError(t, err)
	require.Len(t, result, 2)

	// Verify deterministic ordering — sorted by external ID
	assert.Equal(t, "data-graphs/dg-one.yaml", result[0].RelativePath)
	assert.Equal(t, "data-graphs/dg-two.yaml", result[1].RelativePath)

	for _, entity := range result {
		spec := entity.Content.(*specs.Spec)
		assert.Equal(t, specs.SpecVersionV1, spec.Version)
		assert.Equal(t, "data-graph", spec.Kind)
	}
}

func TestFormatForExport_EmptyCollection(t *testing.T) {
	mockClient := &testutils.MockDataGraphClient{}
	provider := datagraph.NewProvider(mockClient, nil)

	collection := resources.NewRemoteResources()

	result, _, err := provider.FormatForExport(collection, nil, nil)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestFormatForExport_SkipsUnmanagedModelsUnderManagedDataGraphs(t *testing.T) {
	mockClient := &testutils.MockDataGraphClient{}
	provider := datagraph.NewProvider(mockClient, nil)

	// Only importable DG is dg-remote-1. Models under dg-remote-2 (a managed DG
	// not in the importable collection) should be excluded.
	collection := buildRemoteResources(
		map[string]*resources.RemoteResource{
			"dg-remote-1": {
				ID:         "dg-remote-1",
				ExternalID: "importable-dg",
				Data: &dgModel.RemoteDataGraph{
					DataGraph: &dgClient.DataGraph{
						ID:          "dg-remote-1",
						WorkspaceID: "ws-123",
						AccountID:   "account-1",
					},
				},
			},
		},
		map[string]*resources.RemoteResource{
			"model-remote-1": {
				ID:         "model-remote-1",
				ExternalID: "user",
				Data: &dgModel.RemoteModel{
					Model: &dgClient.Model{
						ID:          "model-remote-1",
						Name:        "User",
						Type:        "entity",
						TableRef:    "db.schema.users",
						DataGraphID: "dg-remote-1",
						PrimaryID:   "user_id",
					},
				},
			},
			"model-remote-orphan": {
				ID:         "model-remote-orphan",
				ExternalID: "orphan-model",
				Data: &dgModel.RemoteModel{
					Model: &dgClient.Model{
						ID:          "model-remote-orphan",
						Name:        "Orphan",
						Type:        "entity",
						TableRef:    "db.schema.orphans",
						DataGraphID: "dg-remote-2", // Belongs to a managed DG not in importable collection
						PrimaryID:   "orphan_id",
					},
				},
			},
		},
		nil,
	)

	result, entries, err := provider.FormatForExport(collection, nil, nil)
	require.NoError(t, err)
	require.Len(t, result, 1)

	spec := result[0].Content.(*specs.Spec)
	models, ok := spec.Spec["models"].([]dgModel.ModelSpec)
	require.True(t, ok)

	// Only model-remote-1 should be included (belongs to importable dg-remote-1)
	require.Len(t, models, 1)
	assert.Equal(t, "user", models[0].ID)

	// Import entries only contain resources from the importable DG.
	urnSet := make(map[string]bool, len(entries))
	for _, e := range entries {
		urnSet[e.URN] = true
	}

	assert.True(t, urnSet[resources.URN("importable-dg", dgHandler.HandlerMetadata.ResourceType)])
	assert.True(t, urnSet[resources.URN("user", modelHandler.HandlerMetadata.ResourceType)])
	// Orphan model should not appear
	assert.False(t, urnSet[resources.URN("orphan-model", modelHandler.HandlerMetadata.ResourceType)])
	assert.False(t, urnSet[fmt.Sprintf("%s:%s", modelHandler.HandlerMetadata.ResourceType, "orphan-model")])
}
