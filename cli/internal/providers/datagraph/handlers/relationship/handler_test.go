package relationship

import (
	"context"
	"fmt"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	dgClient "github.com/rudderlabs/rudder-iac/api/client/datagraph"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/handlers/datagraph"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/handlers/model"
	dgModel "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/testutils"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Test Fixtures and Helpers
// ============================================================================

// createResolvedDataGraphRef creates a PropertyRef pointing to a data graph with a resolved remote ID
func createResolvedDataGraphRef(id, remoteID string) *resources.PropertyRef {
	urn := resources.URN(id, datagraph.HandlerMetadata.ResourceType)
	ref := CreateDataGraphReference(urn)
	ref.IsResolved = true
	ref.Value = remoteID
	return ref
}

// createResolvedModelRef creates a PropertyRef pointing to a model with a resolved remote ID
func createResolvedModelRef(id, remoteID string) *resources.PropertyRef {
	urn := resources.URN(id, model.HandlerMetadata.ResourceType)
	ref := CreateModelReference(urn)
	ref.IsResolved = true
	ref.Value = remoteID
	return ref
}

// createTestRelationshipResource creates a test relationship resource with all required fields
func createTestRelationshipResource(
	id, displayName, cardinality string,
	dataGraphRef, sourceModelRef, targetModelRef *resources.PropertyRef,
	sourceJoinKey, targetJoinKey string,
) *dgModel.RelationshipResource {
	return &dgModel.RelationshipResource{
		ID:             id,
		DisplayName:    displayName,
		Cardinality:    cardinality,
		DataGraphRef:   dataGraphRef,
		SourceModelRef: sourceModelRef,
		TargetModelRef: targetModelRef,
		SourceJoinKey:  sourceJoinKey,
		TargetJoinKey:  targetJoinKey,
	}
}

// setupCRUDTestFixtures creates common test fixtures for CRUD operations
func setupCRUDTestFixtures() (*resources.PropertyRef, *resources.PropertyRef, *resources.PropertyRef) {
	dataGraphRef := createResolvedDataGraphRef("my-dg", "dg-remote-123")
	sourceModelRef := createResolvedModelRef("user", "em-1")
	targetModelRef := createResolvedModelRef("account", "em-2")
	return dataGraphRef, sourceModelRef, targetModelRef
}

// setupValidationTestGraph creates a resource graph with common test fixtures for validation tests
func setupValidationTestGraph() (*resources.Graph, *resources.PropertyRef, *resources.PropertyRef, *resources.PropertyRef, *resources.PropertyRef) {
	graph := resources.NewGraph()

	// Add data graph
	graph.AddResource(resources.NewResource(
		"my-dg",
		"data-graph",
		resources.ResourceData{},
		[]string{},
	))

	// Add entity models
	graph.AddResource(resources.NewResource(
		"user-model",
		model.HandlerMetadata.ResourceType,
		resources.ResourceData{},
		[]string{},
		resources.WithRawData(&dgModel.ModelResource{
			ID:          "user",
			DisplayName: "User",
			Type:        "entity",
		}),
	))

	graph.AddResource(resources.NewResource(
		"account-model",
		model.HandlerMetadata.ResourceType,
		resources.ResourceData{},
		[]string{},
		resources.WithRawData(&dgModel.ModelResource{
			ID:          "account",
			DisplayName: "Account",
			Type:        "entity",
		}),
	))

	// Add event model
	graph.AddResource(resources.NewResource(
		"purchase-model",
		model.HandlerMetadata.ResourceType,
		resources.ResourceData{},
		[]string{},
		resources.WithRawData(&dgModel.ModelResource{
			ID:          "purchase",
			DisplayName: "Purchase",
			Type:        "event",
		}),
	))

	// Create refs
	dataGraphURN := resources.URN("my-dg", datagraph.HandlerMetadata.ResourceType)
	dataGraphRef := CreateDataGraphReference(dataGraphURN)

	entityModelURN := resources.URN("user-model", model.HandlerMetadata.ResourceType)
	entityModelRef := CreateModelReference(entityModelURN)

	anotherEntityModelURN := resources.URN("account-model", model.HandlerMetadata.ResourceType)
	anotherEntityModelRef := CreateModelReference(anotherEntityModelURN)

	eventModelURN := resources.URN("purchase-model", model.HandlerMetadata.ResourceType)
	eventModelRef := CreateModelReference(eventModelURN)

	return graph, dataGraphRef, entityModelRef, anotherEntityModelRef, eventModelRef
}

// createURNResolver creates a mock URN resolver with common mappings for tests
func createURNResolver() *testutils.MockURNResolver {
	urnResolver := testutils.NewMockURNResolver()
	urnResolver.AddMapping("data-graph", "dg-remote-1", "data-graph:my-dg")
	urnResolver.AddMapping("data-graph-model", "em-1", "data-graph-model:user")
	urnResolver.AddMapping("data-graph-model", "em-2", "data-graph-model:account")
	return urnResolver
}

// ============================================================================
// ValidateResource Tests
// ============================================================================

func TestValidateResource(t *testing.T) {
	mockClient := &testutils.MockDataGraphClient{}
	h := &HandlerImpl{client: mockClient}

	graph, dataGraphRef, entityModelRef, anotherEntityModelRef, eventModelRef := setupValidationTestGraph()

	// Create non-existent refs for error testing
	nonExistentDGURN := resources.URN("non-existent-dg", datagraph.HandlerMetadata.ResourceType)
	nonExistentDGRef := CreateDataGraphReference(nonExistentDGURN)

	nonExistentModelURN := resources.URN("non-existent-model", model.HandlerMetadata.ResourceType)
	nonExistentModelRef := CreateModelReference(nonExistentModelURN)

	tests := []struct {
		name     string
		resource *dgModel.RelationshipResource
		wantErr  bool
		errMsg   string
	}{
		{
			name: "valid entity-to-entity relationship with one-to-many",
			resource: &dgModel.RelationshipResource{
				ID:             "user-accounts",
				DisplayName:    "User Accounts",
				Cardinality:    "one-to-many",
				DataGraphRef:   dataGraphRef,
				SourceModelRef: entityModelRef,
				TargetModelRef: anotherEntityModelRef,
				SourceJoinKey:  "user_id",
				TargetJoinKey:  "id",
			},
			wantErr: false,
		},
		{
			name: "valid entity-to-entity relationship with many-to-one",
			resource: &dgModel.RelationshipResource{
				ID:             "account-user",
				DisplayName:    "Account User",
				Cardinality:    "many-to-one",
				DataGraphRef:   dataGraphRef,
				SourceModelRef: anotherEntityModelRef,
				TargetModelRef: entityModelRef,
				SourceJoinKey:  "user_id",
				TargetJoinKey:  "id",
			},
			wantErr: false,
		},
		{
			name: "valid entity-to-entity relationship with one-to-one",
			resource: &dgModel.RelationshipResource{
				ID:             "user-profile",
				DisplayName:    "User Profile",
				Cardinality:    "one-to-one",
				DataGraphRef:   dataGraphRef,
				SourceModelRef: entityModelRef,
				TargetModelRef: anotherEntityModelRef,
				SourceJoinKey:  "user_id",
				TargetJoinKey:  "id",
			},
			wantErr: false,
		},
		{
			name: "valid event-to-entity relationship with many-to-one",
			resource: &dgModel.RelationshipResource{
				ID:             "purchase-user",
				DisplayName:    "Purchase User",
				Cardinality:    "many-to-one",
				DataGraphRef:   dataGraphRef,
				SourceModelRef: eventModelRef,
				TargetModelRef: entityModelRef,
				SourceJoinKey:  "user_id",
				TargetJoinKey:  "id",
			},
			wantErr: false,
		},
		{
			name: "valid entity-to-event relationship with one-to-many",
			resource: &dgModel.RelationshipResource{
				ID:             "user-purchases",
				DisplayName:    "User Purchases",
				Cardinality:    "one-to-many",
				DataGraphRef:   dataGraphRef,
				SourceModelRef: entityModelRef,
				TargetModelRef: eventModelRef,
				SourceJoinKey:  "id",
				TargetJoinKey:  "user_id",
			},
			wantErr: false,
		},
		{
			name: "missing display_name",
			resource: &dgModel.RelationshipResource{
				ID:             "user-accounts",
				Cardinality:    "one-to-many",
				DataGraphRef:   dataGraphRef,
				SourceModelRef: entityModelRef,
				TargetModelRef: anotherEntityModelRef,
				SourceJoinKey:  "user_id",
				TargetJoinKey:  "id",
			},
			wantErr: true,
			errMsg:  "display_name is required",
		},
		{
			name: "missing cardinality",
			resource: &dgModel.RelationshipResource{
				ID:             "user-accounts",
				DisplayName:    "User Accounts",
				DataGraphRef:   dataGraphRef,
				SourceModelRef: entityModelRef,
				TargetModelRef: anotherEntityModelRef,
				SourceJoinKey:  "user_id",
				TargetJoinKey:  "id",
			},
			wantErr: true,
			errMsg:  "cardinality is required",
		},
		{
			name: "missing data_graph reference",
			resource: &dgModel.RelationshipResource{
				ID:             "user-accounts",
				DisplayName:    "User Accounts",
				Cardinality:    "one-to-many",
				SourceModelRef: entityModelRef,
				TargetModelRef: anotherEntityModelRef,
				SourceJoinKey:  "user_id",
				TargetJoinKey:  "id",
			},
			wantErr: true,
			errMsg:  "data_graph reference is required",
		},
		{
			name: "missing source model reference",
			resource: &dgModel.RelationshipResource{
				ID:             "user-accounts",
				DisplayName:    "User Accounts",
				Cardinality:    "one-to-many",
				DataGraphRef:   dataGraphRef,
				TargetModelRef: anotherEntityModelRef,
				SourceJoinKey:  "user_id",
				TargetJoinKey:  "id",
			},
			wantErr: true,
			errMsg:  "source model reference is required",
		},
		{
			name: "missing target model reference",
			resource: &dgModel.RelationshipResource{
				ID:             "user-accounts",
				DisplayName:    "User Accounts",
				Cardinality:    "one-to-many",
				DataGraphRef:   dataGraphRef,
				SourceModelRef: entityModelRef,
				SourceJoinKey:  "user_id",
				TargetJoinKey:  "id",
			},
			wantErr: true,
			errMsg:  "target model reference is required",
		},
		{
			name: "missing source_join_key",
			resource: &dgModel.RelationshipResource{
				ID:             "user-accounts",
				DisplayName:    "User Accounts",
				Cardinality:    "one-to-many",
				DataGraphRef:   dataGraphRef,
				SourceModelRef: entityModelRef,
				TargetModelRef: anotherEntityModelRef,
				TargetJoinKey:  "id",
			},
			wantErr: true,
			errMsg:  "source_join_key is required",
		},
		{
			name: "missing target_join_key",
			resource: &dgModel.RelationshipResource{
				ID:             "user-accounts",
				DisplayName:    "User Accounts",
				Cardinality:    "one-to-many",
				DataGraphRef:   dataGraphRef,
				SourceModelRef: entityModelRef,
				TargetModelRef: anotherEntityModelRef,
				SourceJoinKey:  "user_id",
			},
			wantErr: true,
			errMsg:  "target_join_key is required",
		},
		{
			name: "referenced data graph does not exist",
			resource: &dgModel.RelationshipResource{
				ID:             "user-accounts",
				DisplayName:    "User Accounts",
				Cardinality:    "one-to-many",
				DataGraphRef:   nonExistentDGRef,
				SourceModelRef: entityModelRef,
				TargetModelRef: anotherEntityModelRef,
				SourceJoinKey:  "user_id",
				TargetJoinKey:  "id",
			},
			wantErr: true,
			errMsg:  "referenced data graph data-graph:non-existent-dg does not exist",
		},
		{
			name: "referenced source model does not exist",
			resource: &dgModel.RelationshipResource{
				ID:             "user-accounts",
				DisplayName:    "User Accounts",
				Cardinality:    "one-to-many",
				DataGraphRef:   dataGraphRef,
				SourceModelRef: nonExistentModelRef,
				TargetModelRef: anotherEntityModelRef,
				SourceJoinKey:  "user_id",
				TargetJoinKey:  "id",
			},
			wantErr: true,
			errMsg:  "referenced source model data-graph-model:non-existent-model does not exist",
		},
		{
			name: "referenced target model does not exist",
			resource: &dgModel.RelationshipResource{
				ID:             "user-accounts",
				DisplayName:    "User Accounts",
				Cardinality:    "one-to-many",
				DataGraphRef:   dataGraphRef,
				SourceModelRef: entityModelRef,
				TargetModelRef: nonExistentModelRef,
				SourceJoinKey:  "user_id",
				TargetJoinKey:  "id",
			},
			wantErr: true,
			errMsg:  "referenced target model data-graph-model:non-existent-model does not exist",
		},
		{
			name: "invalid: event-to-event relationship",
			resource: &dgModel.RelationshipResource{
				ID:             "purchase-pageview",
				DisplayName:    "Purchase PageView",
				Cardinality:    "many-to-one",
				DataGraphRef:   dataGraphRef,
				SourceModelRef: eventModelRef,
				TargetModelRef: eventModelRef,
				SourceJoinKey:  "session_id",
				TargetJoinKey:  "session_id",
			},
			wantErr: true,
			errMsg:  "event models cannot be connected to other event models",
		},
		{
			name: "invalid: event-to-entity with wrong cardinality (one-to-many)",
			resource: &dgModel.RelationshipResource{
				ID:             "purchase-user",
				DisplayName:    "Purchase User",
				Cardinality:    "one-to-many",
				DataGraphRef:   dataGraphRef,
				SourceModelRef: eventModelRef,
				TargetModelRef: entityModelRef,
				SourceJoinKey:  "user_id",
				TargetJoinKey:  "id",
			},
			wantErr: true,
			errMsg:  "relationships from event models must have cardinality 'many-to-one', got \"one-to-many\"",
		},
		{
			name: "invalid: event-to-entity with wrong cardinality (one-to-one)",
			resource: &dgModel.RelationshipResource{
				ID:             "purchase-user",
				DisplayName:    "Purchase User",
				Cardinality:    "one-to-one",
				DataGraphRef:   dataGraphRef,
				SourceModelRef: eventModelRef,
				TargetModelRef: entityModelRef,
				SourceJoinKey:  "user_id",
				TargetJoinKey:  "id",
			},
			wantErr: true,
			errMsg:  "relationships from event models must have cardinality 'many-to-one', got \"one-to-one\"",
		},
		{
			name: "invalid: entity-to-event with wrong cardinality (many-to-one)",
			resource: &dgModel.RelationshipResource{
				ID:             "user-purchases",
				DisplayName:    "User Purchases",
				Cardinality:    "many-to-one",
				DataGraphRef:   dataGraphRef,
				SourceModelRef: entityModelRef,
				TargetModelRef: eventModelRef,
				SourceJoinKey:  "id",
				TargetJoinKey:  "user_id",
			},
			wantErr: true,
			errMsg:  "relationships from entity models to event models must have cardinality 'one-to-many', got \"many-to-one\"",
		},
		{
			name: "invalid: entity-to-event with wrong cardinality (one-to-one)",
			resource: &dgModel.RelationshipResource{
				ID:             "user-purchases",
				DisplayName:    "User Purchases",
				Cardinality:    "one-to-one",
				DataGraphRef:   dataGraphRef,
				SourceModelRef: entityModelRef,
				TargetModelRef: eventModelRef,
				SourceJoinKey:  "id",
				TargetJoinKey:  "user_id",
			},
			wantErr: true,
			errMsg:  "relationships from entity models to event models must have cardinality 'one-to-many', got \"one-to-one\"",
		},
		{
			name: "invalid: entity-to-entity with invalid cardinality",
			resource: &dgModel.RelationshipResource{
				ID:             "user-accounts",
				DisplayName:    "User Accounts",
				Cardinality:    "many-to-many",
				DataGraphRef:   dataGraphRef,
				SourceModelRef: entityModelRef,
				TargetModelRef: anotherEntityModelRef,
				SourceJoinKey:  "user_id",
				TargetJoinKey:  "id",
			},
			wantErr: true,
			errMsg:  "cardinality must be one of: one-to-one, one-to-many, many-to-one",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := h.ValidateResource(tt.resource, graph)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ============================================================================
// MapRemoteToState Tests
// ============================================================================

func TestMapRemoteToState(t *testing.T) {
	mockClient := &testutils.MockDataGraphClient{}
	h := &HandlerImpl{client: mockClient}

	t.Run("relationship with external ID", func(t *testing.T) {
		urnResolver := createURNResolver()

		remote := &dgModel.RemoteRelationship{
			Relationship: &dgClient.Relationship{
				ID:            "rel-1",
				ExternalID:    "user-accounts",
				Name:          "User Accounts",
				Cardinality:   "one-to-many",
				SourceModelID: "em-1",
				TargetModelID: "em-2",
				SourceJoinKey: "id",
				TargetJoinKey: "user_id",
				DataGraphID:   "dg-remote-1",
			},
		}

		resource, state, err := h.MapRemoteToState(remote, urnResolver)
		require.NoError(t, err)

		assert.Equal(t, "user-accounts", resource.ID)
		assert.Equal(t, "User Accounts", resource.DisplayName)
		assert.Equal(t, "one-to-many", resource.Cardinality)
		assert.Equal(t, "id", resource.SourceJoinKey)
		assert.Equal(t, "user_id", resource.TargetJoinKey)

		require.NotNil(t, resource.DataGraphRef)
		assert.Equal(t, "data-graph:my-dg", resource.DataGraphRef.URN)

		require.NotNil(t, resource.SourceModelRef)
		assert.Equal(t, "data-graph-model:user", resource.SourceModelRef.URN)

		require.NotNil(t, resource.TargetModelRef)
		assert.Equal(t, "data-graph-model:account", resource.TargetModelRef.URN)

		expectedState := &dgModel.RelationshipState{
			ID: "rel-1",
		}
		assert.Equal(t, expectedState, state)
	})

	t.Run("relationship without external ID", func(t *testing.T) {
		remote := &dgModel.RemoteRelationship{
			Relationship: &dgClient.Relationship{
				ID:            "rel-2",
				Name:          "Purchase User",
				Cardinality:   "many-to-one",
				SourceModelID: "evm-1",
				TargetModelID: "em-1",
				SourceJoinKey: "user_id",
				TargetJoinKey: "id",
				DataGraphID:   "dg-remote-1",
			},
		}

		resource, state, err := h.MapRemoteToState(remote, nil)
		require.NoError(t, err)
		assert.Nil(t, resource)
		assert.Nil(t, state)
	})

	// Error tests - use table-driven approach
	errorTests := []struct {
		name              string
		setupResolver     func() *testutils.MockURNResolver
		expectedErrSubstr string
	}{
		{
			name: "error resolving data graph URN",
			setupResolver: func() *testutils.MockURNResolver {
				// Don't add data graph mapping
				return testutils.NewMockURNResolver()
			},
			expectedErrSubstr: "resolving data graph URN",
		},
		{
			name: "error resolving source model URN",
			setupResolver: func() *testutils.MockURNResolver {
				resolver := testutils.NewMockURNResolver()
				resolver.AddMapping("data-graph", "dg-remote-1", "data-graph:my-dg")
				// Don't add source model mapping
				return resolver
			},
			expectedErrSubstr: "resolving from model URN",
		},
		{
			name: "error resolving target model URN",
			setupResolver: func() *testutils.MockURNResolver {
				resolver := testutils.NewMockURNResolver()
				resolver.AddMapping("data-graph", "dg-remote-1", "data-graph:my-dg")
				resolver.AddMapping("data-graph-model", "em-1", "data-graph-model:user")
				// Don't add target model mapping
				return resolver
			},
			expectedErrSubstr: "resolving to model URN",
		},
	}

	for _, tt := range errorTests {
		t.Run(tt.name, func(t *testing.T) {
			remote := &dgModel.RemoteRelationship{
				Relationship: &dgClient.Relationship{
					ID:            "rel-1",
					ExternalID:    "user-accounts",
					Name:          "User Accounts",
					Cardinality:   "one-to-many",
					SourceModelID: "em-1",
					TargetModelID: "em-2",
					DataGraphID:   "dg-remote-1",
				},
			}

			resource, state, err := h.MapRemoteToState(remote, tt.setupResolver())
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedErrSubstr)
			assert.Nil(t, resource)
			assert.Nil(t, state)
		})
	}
}

// ============================================================================
// ValidateResource - Edge Cases for RawData Type Assertions
// ============================================================================

func TestValidateResource_InvalidRawDataType(t *testing.T) {
	mockClient := &testutils.MockDataGraphClient{}
	h := &HandlerImpl{client: mockClient}

	dataGraphURN := resources.URN("my-dg", datagraph.HandlerMetadata.ResourceType)
	dataGraphRef := CreateDataGraphReference(dataGraphURN)

	tests := []struct {
		name          string
		setupGraph    func() (*resources.Graph, *resources.PropertyRef, *resources.PropertyRef)
		expectedError string
	}{
		{
			name: "source model has invalid RawData type",
			setupGraph: func() (*resources.Graph, *resources.PropertyRef, *resources.PropertyRef) {
				graph := resources.NewGraph()
				graph.AddResource(resources.NewResource("my-dg", "data-graph", resources.ResourceData{}, []string{}))

				invalidSourceURN := resources.URN("invalid-source", model.HandlerMetadata.ResourceType)
				graph.AddResource(resources.NewResource(
					"invalid-source", model.HandlerMetadata.ResourceType, resources.ResourceData{}, []string{},
					resources.WithRawData("not a model resource"),
				))

				validTargetURN := resources.URN("valid-target", model.HandlerMetadata.ResourceType)
				graph.AddResource(resources.NewResource(
					"valid-target", model.HandlerMetadata.ResourceType, resources.ResourceData{}, []string{},
					resources.WithRawData(&dgModel.ModelResource{ID: "target", DisplayName: "Target", Type: "entity"}),
				))

				return graph, CreateModelReference(invalidSourceURN), CreateModelReference(validTargetURN)
			},
			expectedError: "does not point to a valid model resource",
		},
		{
			name: "target model has invalid RawData type",
			setupGraph: func() (*resources.Graph, *resources.PropertyRef, *resources.PropertyRef) {
				graph := resources.NewGraph()
				graph.AddResource(resources.NewResource("my-dg", "data-graph", resources.ResourceData{}, []string{}))

				validSourceURN := resources.URN("valid-source", model.HandlerMetadata.ResourceType)
				graph.AddResource(resources.NewResource(
					"valid-source", model.HandlerMetadata.ResourceType, resources.ResourceData{}, []string{},
					resources.WithRawData(&dgModel.ModelResource{ID: "source", DisplayName: "Source", Type: "entity"}),
				))

				invalidTargetURN := resources.URN("invalid-target", model.HandlerMetadata.ResourceType)
				graph.AddResource(resources.NewResource(
					"invalid-target", model.HandlerMetadata.ResourceType, resources.ResourceData{}, []string{},
					resources.WithRawData(123),
				))

				return graph, CreateModelReference(validSourceURN), CreateModelReference(invalidTargetURN)
			},
			expectedError: "does not point to a valid model resource",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			graph, sourceRef, targetRef := tt.setupGraph()

			resource := createTestRelationshipResource(
				"test-rel", "Test Relationship", "one-to-one",
				dataGraphRef, sourceRef, targetRef,
				"id", "id",
			)

			err := h.ValidateResource(resource, graph)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

// ============================================================================
// Create Tests
// ============================================================================

func TestCreate(t *testing.T) {
	t.Run("create relationship", func(t *testing.T) {
		mockClient := &testutils.MockDataGraphClient{
			CreateRelationshipFunc: func(ctx context.Context, req *dgClient.CreateRelationshipRequest) (*dgClient.Relationship, error) {
				assert.Equal(t, "dg-remote-123", req.DataGraphID)
				assert.Equal(t, "User Accounts", req.Name)
				assert.Equal(t, "one-to-many", req.Cardinality)
				assert.Equal(t, "em-1", req.SourceModelID)
				assert.Equal(t, "em-2", req.TargetModelID)
				assert.Equal(t, "id", req.SourceJoinKey)
				assert.Equal(t, "user_id", req.TargetJoinKey)
				assert.Equal(t, "user-accounts", req.ExternalID)

				return &dgClient.Relationship{
					ID:            "rel-456",
					Name:          req.Name,
					Cardinality:   req.Cardinality,
					SourceModelID: req.SourceModelID,
					TargetModelID: req.TargetModelID,
					SourceJoinKey: req.SourceJoinKey,
					TargetJoinKey: req.TargetJoinKey,
					ExternalID:    req.ExternalID,
				}, nil
			},
		}

		h := &HandlerImpl{client: mockClient}
		dataGraphRef, sourceModelRef, targetModelRef := setupCRUDTestFixtures()

		data := createTestRelationshipResource(
			"user-accounts", "User Accounts", "one-to-many",
			dataGraphRef, sourceModelRef, targetModelRef,
			"id", "user_id",
		)

		state, err := h.Create(context.Background(), data)
		require.NoError(t, err)

		expectedState := &dgModel.RelationshipState{
			ID: "rel-456",
		}
		assert.Equal(t, expectedState, state)
	})

}

// ============================================================================
// Update Tests
// ============================================================================

func TestUpdate(t *testing.T) {
	t.Run("update relationship", func(t *testing.T) {
		mockClient := &testutils.MockDataGraphClient{
			UpdateRelationshipFunc: func(ctx context.Context, req *dgClient.UpdateRelationshipRequest) (*dgClient.Relationship, error) {
				assert.Equal(t, "dg-remote-123", req.DataGraphID)
				assert.Equal(t, "rel-456", req.RelationshipID)
				assert.Equal(t, "Updated User Accounts", req.Name)
				assert.Equal(t, "many-to-one", req.Cardinality)
				assert.Equal(t, "em-1", req.SourceModelID)
				assert.Equal(t, "em-2", req.TargetModelID)
				assert.Equal(t, "user_id", req.SourceJoinKey)
				assert.Equal(t, "id", req.TargetJoinKey)

				return &dgClient.Relationship{
					ID:            req.RelationshipID,
					Name:          req.Name,
					Cardinality:   req.Cardinality,
					SourceModelID: req.SourceModelID,
					TargetModelID: req.TargetModelID,
					SourceJoinKey: req.SourceJoinKey,
					TargetJoinKey: req.TargetJoinKey,
				}, nil
			},
		}

		h := &HandlerImpl{client: mockClient}
		dataGraphRef, sourceModelRef, targetModelRef := setupCRUDTestFixtures()

		newData := createTestRelationshipResource(
			"user-accounts", "Updated User Accounts", "many-to-one",
			dataGraphRef, sourceModelRef, targetModelRef,
			"user_id", "id",
		)

		oldData := createTestRelationshipResource(
			"user-accounts", "User Accounts", "one-to-many",
			dataGraphRef, sourceModelRef, targetModelRef,
			"id", "user_id",
		)

		oldState := &dgModel.RelationshipState{
			ID: "rel-456",
		}

		state, err := h.Update(context.Background(), newData, oldData, oldState)
		require.NoError(t, err)

		expectedState := &dgModel.RelationshipState{
			ID: "rel-456",
		}
		assert.Equal(t, expectedState, state)
	})

}

// ============================================================================
// Import Tests
// ============================================================================

func TestImport(t *testing.T) {
	t.Run("import relationship", func(t *testing.T) {
		mockClient := &testutils.MockDataGraphClient{
			SetRelationshipExternalIDFunc: func(ctx context.Context, req *dgClient.SetRelationshipExternalIDRequest) (*dgClient.Relationship, error) {
				assert.Equal(t, "dg-123", req.DataGraphID)
				assert.Equal(t, "rel-456", req.RelationshipID)
				assert.Equal(t, "user-accounts", req.ExternalID)
				return &dgClient.Relationship{
					ID:            req.RelationshipID,
					ExternalID:    req.ExternalID,
					DataGraphID:   req.DataGraphID,
					Name:          "User Accounts",
					Cardinality:   "one-to-many",
					SourceModelID: "em-1",
					TargetModelID: "em-2",
				}, nil
			},
			GetRelationshipFunc: func(ctx context.Context, req *dgClient.GetRelationshipRequest) (*dgClient.Relationship, error) {
				assert.Equal(t, "dg-123", req.DataGraphID)
				assert.Equal(t, "rel-456", req.RelationshipID)
				return &dgClient.Relationship{
					ID:            req.RelationshipID,
					ExternalID:    "user-accounts",
					DataGraphID:   req.DataGraphID,
					Name:          "User Accounts",
					Cardinality:   "one-to-many",
					SourceModelID: "em-1",
					TargetModelID: "em-2",
				}, nil
			},
		}

		h := &HandlerImpl{client: mockClient}

		dataGraphRef := createResolvedDataGraphRef("my-dg", "dg-123")
		sourceModelRef := createResolvedModelRef("user", "em-1")
		targetModelRef := createResolvedModelRef("account", "em-2")

		data := createTestRelationshipResource(
			"user-accounts", "User Accounts", "one-to-many",
			dataGraphRef, sourceModelRef, targetModelRef,
			"id", "user_id",
		)

		state, err := h.Import(context.Background(), data, "rel-456")
		require.NoError(t, err)

		expectedState := &dgModel.RelationshipState{
			ID: "rel-456",
		}
		assert.Equal(t, expectedState, state)
	})

}

// ============================================================================
// Delete Tests
// ============================================================================

func TestDelete(t *testing.T) {
	t.Run("delete relationship", func(t *testing.T) {
		mockClient := &testutils.MockDataGraphClient{
			DeleteRelationshipFunc: func(ctx context.Context, req *dgClient.DeleteRelationshipRequest) error {
				assert.Equal(t, "dg-remote-123", req.DataGraphID)
				assert.Equal(t, "rel-456", req.RelationshipID)
				return nil
			},
		}

		h := &HandlerImpl{client: mockClient}
		dataGraphRef, sourceModelRef, targetModelRef := setupCRUDTestFixtures()

		oldData := createTestRelationshipResource(
			"user-accounts", "User Accounts", "one-to-many",
			dataGraphRef, sourceModelRef, targetModelRef,
			"id", "user_id",
		)

		oldState := &dgModel.RelationshipState{
			ID: "rel-456",
		}

		err := h.Delete(context.Background(), "user-accounts", oldData, oldState)
		require.NoError(t, err)
	})

}

// ============================================================================
// CRUD Operations - Error Handling (Table-Driven)
// ============================================================================

func TestCRUDOperations_Errors(t *testing.T) {
	dataGraphRef, sourceModelRef, targetModelRef := setupCRUDTestFixtures()

	relationshipData := createTestRelationshipResource(
		"user-accounts", "User Accounts", "one-to-many",
		dataGraphRef, sourceModelRef, targetModelRef,
		"id", "user_id",
	)

	oldState := &dgModel.RelationshipState{ID: "rel-456"}

	tests := []struct {
		name        string
		operation   string // "create", "update", "delete", "import-set", "import-get"
		setupMock   func() *testutils.MockDataGraphClient
		expectedErr string
	}{
		{
			name:      "create - API error",
			operation: "create",
			setupMock: func() *testutils.MockDataGraphClient {
				return &testutils.MockDataGraphClient{
					CreateRelationshipFunc: func(ctx context.Context, req *dgClient.CreateRelationshipRequest) (*dgClient.Relationship, error) {
						return nil, fmt.Errorf("API error: failed to create")
					},
				}
			},
			expectedErr: "creating relationship",
		},
		{
			name:      "update - API error",
			operation: "update",
			setupMock: func() *testutils.MockDataGraphClient {
				return &testutils.MockDataGraphClient{
					UpdateRelationshipFunc: func(ctx context.Context, req *dgClient.UpdateRelationshipRequest) (*dgClient.Relationship, error) {
						return nil, fmt.Errorf("API error: failed to update")
					},
				}
			},
			expectedErr: "updating relationship",
		},
		{
			name:      "delete - API error",
			operation: "delete",
			setupMock: func() *testutils.MockDataGraphClient {
				return &testutils.MockDataGraphClient{
					DeleteRelationshipFunc: func(ctx context.Context, req *dgClient.DeleteRelationshipRequest) error {
						return fmt.Errorf("API error: failed to delete")
					},
				}
			},
			expectedErr: "deleting relationship",
		},
		{
			name:      "import - SetExternalID error",
			operation: "import-set",
			setupMock: func() *testutils.MockDataGraphClient {
				return &testutils.MockDataGraphClient{
					SetRelationshipExternalIDFunc: func(ctx context.Context, req *dgClient.SetRelationshipExternalIDRequest) (*dgClient.Relationship, error) {
						return nil, fmt.Errorf("API error: failed to set external ID")
					},
				}
			},
			expectedErr: "setting external ID",
		},
		{
			name:      "import - GetRelationship error",
			operation: "import-get",
			setupMock: func() *testutils.MockDataGraphClient {
				return &testutils.MockDataGraphClient{
					SetRelationshipExternalIDFunc: func(ctx context.Context, req *dgClient.SetRelationshipExternalIDRequest) (*dgClient.Relationship, error) {
						return &dgClient.Relationship{ID: req.RelationshipID, ExternalID: req.ExternalID}, nil
					},
					GetRelationshipFunc: func(ctx context.Context, req *dgClient.GetRelationshipRequest) (*dgClient.Relationship, error) {
						return nil, fmt.Errorf("API error: failed to get")
					},
				}
			},
			expectedErr: "getting relationship",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &HandlerImpl{client: tt.setupMock()}

			var err error
			switch tt.operation {
			case "create":
				_, err = h.Create(context.Background(), relationshipData)
			case "update":
				_, err = h.Update(context.Background(), relationshipData, relationshipData, oldState)
			case "delete":
				err = h.Delete(context.Background(), "user-accounts", relationshipData, oldState)
			case "import-set", "import-get":
				_, err = h.Import(context.Background(), relationshipData, "rel-456")
			}

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}

// ============================================================================
// LoadRemoteResources Tests
// ============================================================================

func TestLoadRemoteResources(t *testing.T) {
	mockClient := &testutils.MockDataGraphClient{
		ListDataGraphsFunc: func(ctx context.Context, page, perPage int, hasExternalID *bool) (*dgClient.ListDataGraphsResponse, error) {
			require.NotNil(t, hasExternalID)
			assert.True(t, *hasExternalID)

			if page == 1 {
				return &dgClient.ListDataGraphsResponse{
					Data: []dgClient.DataGraph{
						{
							ID:         "dg-1",
							ExternalID: "my-dg",
						},
					},
					Paging: client.Paging{Next: ""},
				}, nil
			}
			return &dgClient.ListDataGraphsResponse{}, nil
		},
		ListRelationshipsFunc: func(ctx context.Context, req *dgClient.ListRelationshipsRequest) (*dgClient.ListRelationshipsResponse, error) {
			assert.Equal(t, "dg-1", req.DataGraphID)
			require.NotNil(t, req.HasExternalID)
			assert.True(t, *req.HasExternalID)

			if req.Page == 1 {
				return &dgClient.ListRelationshipsResponse{
					Data: []dgClient.Relationship{
						{
							ID:            "rel-1",
							ExternalID:    "user-accounts",
							Name:          "User Accounts",
							Cardinality:   "one-to-many",
							SourceModelID: "em-1",
							TargetModelID: "em-2",
							SourceJoinKey: "id",
							TargetJoinKey: "user_id",
							DataGraphID:   "dg-1",
						},
						{
							ID:            "rel-2",
							ExternalID:    "purchase-user",
							Name:          "Purchase User",
							Cardinality:   "many-to-one",
							SourceModelID: "evm-1",
							TargetModelID: "em-1",
							SourceJoinKey: "user_id",
							TargetJoinKey: "id",
							DataGraphID:   "dg-1",
						},
					},
					Paging: client.Paging{Next: ""},
				}, nil
			}
			return &dgClient.ListRelationshipsResponse{}, nil
		},
	}

	h := &HandlerImpl{client: mockClient}
	remotes, err := h.LoadRemoteResources(context.Background())
	require.NoError(t, err)
	require.Len(t, remotes, 2)

	// Check first relationship
	assert.Equal(t, "rel-1", remotes[0].ID)
	assert.Equal(t, "user-accounts", remotes[0].ExternalID)
	assert.Equal(t, "User Accounts", remotes[0].Name)
	assert.Equal(t, "one-to-many", remotes[0].Cardinality)
	assert.Equal(t, "dg-1", remotes[0].DataGraphID)

	// Check second relationship
	assert.Equal(t, "rel-2", remotes[1].ID)
	assert.Equal(t, "purchase-user", remotes[1].ExternalID)
	assert.Equal(t, "Purchase User", remotes[1].Name)
	assert.Equal(t, "many-to-one", remotes[1].Cardinality)
	assert.Equal(t, "dg-1", remotes[1].DataGraphID)
}

func TestLoadImportableResources(t *testing.T) {
	mockClient := &testutils.MockDataGraphClient{
		ListDataGraphsFunc: func(ctx context.Context, page, perPage int, hasExternalID *bool) (*dgClient.ListDataGraphsResponse, error) {
			if page == 1 {
				return &dgClient.ListDataGraphsResponse{
					Data: []dgClient.DataGraph{
						{
							ID:         "dg-1",
							ExternalID: "",
						},
					},
					Paging: client.Paging{Next: ""},
				}, nil
			}
			return &dgClient.ListDataGraphsResponse{}, nil
		},
		ListRelationshipsFunc: func(ctx context.Context, req *dgClient.ListRelationshipsRequest) (*dgClient.ListRelationshipsResponse, error) {
			assert.Equal(t, "dg-1", req.DataGraphID)
			require.NotNil(t, req.HasExternalID)
			assert.False(t, *req.HasExternalID)

			if req.Page == 1 {
				return &dgClient.ListRelationshipsResponse{
					Data: []dgClient.Relationship{
						{
							ID:            "rel-3",
							Name:          "Account User",
							Cardinality:   "many-to-one",
							SourceModelID: "em-2",
							TargetModelID: "em-1",
							SourceJoinKey: "user_id",
							TargetJoinKey: "id",
							DataGraphID:   "dg-1",
						},
					},
					Paging: client.Paging{Next: ""},
				}, nil
			}
			return &dgClient.ListRelationshipsResponse{}, nil
		},
	}

	h := &HandlerImpl{client: mockClient}
	remotes, err := h.LoadImportableResources(context.Background())
	require.NoError(t, err)
	require.Len(t, remotes, 1)

	// Check relationship
	assert.Equal(t, "rel-3", remotes[0].ID)
	assert.Equal(t, "", remotes[0].ExternalID)
	assert.Equal(t, "Account User", remotes[0].Name)
	assert.Equal(t, "many-to-one", remotes[0].Cardinality)
	assert.Equal(t, "dg-1", remotes[0].DataGraphID)
}

// ============================================================================
// LoadRemote Operations - Error Handling (Table-Driven)
// ============================================================================

func TestLoadRemoteOperations_Errors(t *testing.T) {
	tests := []struct {
		name              string
		loadRemote        bool // true = LoadRemoteResources, false = LoadImportableResources
		setupMock         func() *testutils.MockDataGraphClient
		expectedErrSubstr string
	}{
		{
			name:       "LoadRemoteResources - error listing data graphs",
			loadRemote: true,
			setupMock: func() *testutils.MockDataGraphClient {
				return &testutils.MockDataGraphClient{
					ListDataGraphsFunc: func(ctx context.Context, page, perPage int, hasExternalID *bool) (*dgClient.ListDataGraphsResponse, error) {
						return nil, fmt.Errorf("API error: failed to list data graphs")
					},
				}
			},
			expectedErrSubstr: "listing data graphs",
		},
		{
			name:       "LoadRemoteResources - error listing relationships",
			loadRemote: true,
			setupMock: func() *testutils.MockDataGraphClient {
				return &testutils.MockDataGraphClient{
					ListDataGraphsFunc: func(ctx context.Context, page, perPage int, hasExternalID *bool) (*dgClient.ListDataGraphsResponse, error) {
						if page == 1 {
							return &dgClient.ListDataGraphsResponse{
								Data:   []dgClient.DataGraph{{ID: "dg-1", ExternalID: "my-dg"}},
								Paging: client.Paging{Next: ""},
							}, nil
						}
						return &dgClient.ListDataGraphsResponse{}, nil
					},
					ListRelationshipsFunc: func(ctx context.Context, req *dgClient.ListRelationshipsRequest) (*dgClient.ListRelationshipsResponse, error) {
						return nil, fmt.Errorf("API error: failed to list relationships")
					},
				}
			},
			expectedErrSubstr: "loading relationships for data graph",
		},
		{
			name:       "LoadImportableResources - error listing data graphs",
			loadRemote: false,
			setupMock: func() *testutils.MockDataGraphClient {
				return &testutils.MockDataGraphClient{
					ListDataGraphsFunc: func(ctx context.Context, page, perPage int, hasExternalID *bool) (*dgClient.ListDataGraphsResponse, error) {
						return nil, fmt.Errorf("API error: failed to list data graphs")
					},
				}
			},
			expectedErrSubstr: "listing data graphs",
		},
		{
			name:       "LoadImportableResources - error listing relationships",
			loadRemote: false,
			setupMock: func() *testutils.MockDataGraphClient {
				return &testutils.MockDataGraphClient{
					ListDataGraphsFunc: func(ctx context.Context, page, perPage int, hasExternalID *bool) (*dgClient.ListDataGraphsResponse, error) {
						if page == 1 {
							return &dgClient.ListDataGraphsResponse{
								Data:   []dgClient.DataGraph{{ID: "dg-1", ExternalID: ""}},
								Paging: client.Paging{Next: ""},
							}, nil
						}
						return &dgClient.ListDataGraphsResponse{}, nil
					},
					ListRelationshipsFunc: func(ctx context.Context, req *dgClient.ListRelationshipsRequest) (*dgClient.ListRelationshipsResponse, error) {
						return nil, fmt.Errorf("API error: failed to list relationships")
					},
				}
			},
			expectedErrSubstr: "loading importable relationships for data graph",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &HandlerImpl{client: tt.setupMock()}

			var err error
			var remotes []*dgModel.RemoteRelationship
			if tt.loadRemote {
				remotes, err = h.LoadRemoteResources(context.Background())
			} else {
				remotes, err = h.LoadImportableResources(context.Background())
			}

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedErrSubstr)
			assert.Nil(t, remotes)
		})
	}
}

// ============================================================================
// Pagination Tests
// ============================================================================

func TestLoadRemoteResourcesWithPagination(t *testing.T) {
	mockClient := &testutils.MockDataGraphClient{
		ListDataGraphsFunc: func(ctx context.Context, page, perPage int, hasExternalID *bool) (*dgClient.ListDataGraphsResponse, error) {
			if page == 1 {
				return &dgClient.ListDataGraphsResponse{
					Data: []dgClient.DataGraph{
						{ID: "dg-1", ExternalID: "my-dg-1"},
					},
					Paging: client.Paging{Next: "page2"},
				}, nil
			} else if page == 2 {
				return &dgClient.ListDataGraphsResponse{
					Data: []dgClient.DataGraph{
						{ID: "dg-2", ExternalID: "my-dg-2"},
					},
					Paging: client.Paging{Next: ""},
				}, nil
			}
			return &dgClient.ListDataGraphsResponse{}, nil
		},
		ListRelationshipsFunc: func(ctx context.Context, req *dgClient.ListRelationshipsRequest) (*dgClient.ListRelationshipsResponse, error) {
			if req.DataGraphID == "dg-1" && req.Page == 1 {
				return &dgClient.ListRelationshipsResponse{
					Data: []dgClient.Relationship{
						{ID: "rel-1", ExternalID: "rel-1-ext", DataGraphID: "dg-1", Name: "Rel 1", Cardinality: "one-to-many", SourceModelID: "em-1", TargetModelID: "em-2", SourceJoinKey: "id", TargetJoinKey: "fk"},
					},
					Paging: client.Paging{Next: "page2"},
				}, nil
			} else if req.DataGraphID == "dg-1" && req.Page == 2 {
				return &dgClient.ListRelationshipsResponse{
					Data: []dgClient.Relationship{
						{ID: "rel-2", ExternalID: "rel-2-ext", DataGraphID: "dg-1", Name: "Rel 2", Cardinality: "many-to-one", SourceModelID: "em-1", TargetModelID: "em-2", SourceJoinKey: "fk", TargetJoinKey: "id"},
					},
					Paging: client.Paging{Next: ""},
				}, nil
			} else if req.DataGraphID == "dg-2" && req.Page == 1 {
				return &dgClient.ListRelationshipsResponse{
					Data: []dgClient.Relationship{
						{ID: "rel-3", ExternalID: "rel-3-ext", DataGraphID: "dg-2", Name: "Rel 3", Cardinality: "one-to-one", SourceModelID: "em-3", TargetModelID: "em-4", SourceJoinKey: "id", TargetJoinKey: "id"},
					},
					Paging: client.Paging{Next: ""},
				}, nil
			}
			return &dgClient.ListRelationshipsResponse{}, nil
		},
	}

	h := &HandlerImpl{client: mockClient}
	remotes, err := h.LoadRemoteResources(context.Background())
	require.NoError(t, err)
	require.Len(t, remotes, 3)

	// Verify all relationships were loaded
	assert.Equal(t, "rel-1", remotes[0].ID)
	assert.Equal(t, "rel-2", remotes[1].ID)
	assert.Equal(t, "rel-3", remotes[2].ID)
}

// ============================================================================
// Spec Validation Tests (ValidateSpec should error)
// ============================================================================

func TestValidateSpec(t *testing.T) {
	mockClient := &testutils.MockDataGraphClient{}
	h := &HandlerImpl{client: mockClient}

	spec := &struct{}{}
	err := h.ValidateSpec(spec)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "relationship handler does not support standalone specs")
}

// ============================================================================
// ExtractResourcesFromSpec Tests (should error)
// ============================================================================

func TestExtractResourcesFromSpec(t *testing.T) {
	mockClient := &testutils.MockDataGraphClient{}
	h := &HandlerImpl{client: mockClient}

	spec := &struct{}{}
	resources, err := h.ExtractResourcesFromSpec("test.yaml", spec)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "relationship handler does not support standalone spec extraction")
	assert.Nil(t, resources)
}

// ============================================================================
// FormatForExport Tests
// ============================================================================

func TestFormatForExport(t *testing.T) {
	mockClient := &testutils.MockDataGraphClient{}
	h := &HandlerImpl{client: mockClient}

	collection := map[string]*dgModel.RemoteRelationship{
		"rel-1": {
			Relationship: &dgClient.Relationship{
				ID:         "rel-1",
				ExternalID: "user-accounts",
			},
		},
	}

	result, err := h.FormatForExport(collection, nil, nil)
	require.NoError(t, err)
	assert.Nil(t, result)
}

// ============================================================================
// Metadata and NewSpec Tests
// ============================================================================

func TestMetadata(t *testing.T) {
	mockClient := &testutils.MockDataGraphClient{}
	h := &HandlerImpl{client: mockClient}

	metadata := h.Metadata()
	assert.Equal(t, "data-graph-relationship", metadata.ResourceType)
	assert.Equal(t, "data-graph", metadata.SpecKind)
	assert.Equal(t, "data-graph", metadata.SpecMetadataName)
}

func TestNewSpec(t *testing.T) {
	mockClient := &testutils.MockDataGraphClient{}
	h := &HandlerImpl{client: mockClient}

	spec := h.NewSpec()
	assert.NotNil(t, spec)
}

// ============================================================================
// Helper Function Tests
// ============================================================================

func TestCreateModelReference(t *testing.T) {
	urn := "data-graph-model:user"
	ref := CreateModelReference(urn)

	assert.NotNil(t, ref)
	assert.Equal(t, urn, ref.URN)
}

func TestCreateDataGraphReference(t *testing.T) {
	urn := "data-graph:my-dg"
	ref := CreateDataGraphReference(urn)

	assert.NotNil(t, ref)
	assert.Equal(t, urn, ref.URN)
}

// ============================================================================
// NewHandler Test
// ============================================================================

func TestNewHandler(t *testing.T) {
	mockClient := &testutils.MockDataGraphClient{}
	handler := NewHandler(mockClient)

	assert.NotNil(t, handler)
}
