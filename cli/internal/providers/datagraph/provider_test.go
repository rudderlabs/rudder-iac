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

func TestLoadSpec_ComprehensiveDataGraphWithInlineModels(t *testing.T) {
	mockClient := &testutils.MockDataGraphClient{}
	provider := datagraph.NewProvider(mockClient, nil)

	// Comprehensive spec with data graph and multiple inline models with relationships
	spec := &specs.Spec{
		Version: "rudder/v0.1",
		Kind:    "data-graph",
		Spec: map[string]interface{}{
			"id":         "my-data-graph",
			"account_id": "wh-account-123",
			"models": []map[string]interface{}{
				{
					"id":           "user",
					"display_name": "User",
					"type":         "entity",
					"table":        "db.schema.users",
					"description":  "User entity model",
					"primary_id":   "user_id",
					"root":         true,
					"relationships": []map[string]interface{}{
						{
							"id":              "user-account",
							"display_name":    "User Account",
							"cardinality":     "one-to-one",
							"target":          "#data-graph-model:account",
							"source_join_key": "account_id",
							"target_join_key": "account_id",
						},
					},
				},
				{
					"id":           "account",
					"display_name": "Account",
					"type":         "entity",
					"table":        "db.schema.accounts",
					"description":  "Account entity model",
					"primary_id":   "account_id",
					"root":         false,
				},
				{
					"id":           "page_view",
					"display_name": "Page View",
					"type":         "event",
					"table":        "db.schema.page_views",
					"description":  "Page view event",
					"timestamp":    "viewed_at",
					"relationships": []map[string]interface{}{
						{
							"id":              "pageview-user",
							"cardinality":     "many-to-one",
							"display_name":    "PageView User",
							"target":          "#data-graph-model:user",
							"source_join_key": "user_id",
							"target_join_key": "user_id",
						},
					},
				},
				{
					"id":           "purchase",
					"display_name": "Purchase",
					"type":         "event",
					"table":        "db.schema.purchases",
					"timestamp":    "purchased_at",
				},
			},
		},
	}

	// Load the spec
	err := provider.LoadSpec("test.yaml", spec)
	require.NoError(t, err)

	// Get the resource graph to validate extracted resources
	graph, err := provider.ResourceGraph()
	require.NoError(t, err)

	// Validate data graph resource
	dgResource, exists := graph.GetResource(resources.URN("my-data-graph", dgHandler.HandlerMetadata.ResourceType))
	require.True(t, exists, "data graph resource should exist")
	require.NotNil(t, dgResource)

	dgData, ok := dgResource.RawData().(*dgModel.DataGraphResource)
	require.True(t, ok, "resource data should be DataGraphResource")
	assert.Equal(t, &dgModel.DataGraphResource{
		ID:        "my-data-graph",
		AccountID: "wh-account-123",
	}, dgData)

	// Validate entity model resources
	userResource, exists := graph.GetResource(resources.URN("user", modelHandler.HandlerMetadata.ResourceType))
	require.True(t, exists, "user model resource should exist")
	userData, ok := userResource.RawData().(*dgModel.ModelResource)
	require.True(t, ok)
	assert.Equal(t, "user", userData.ID)
	assert.Equal(t, "User", userData.DisplayName)
	assert.Equal(t, "entity", userData.Type)
	assert.Equal(t, "db.schema.users", userData.Table)
	assert.Equal(t, "User entity model", userData.Description)
	assert.Equal(t, "user_id", userData.PrimaryID)
	assert.True(t, userData.Root)
	assert.NotNil(t, userData.DataGraphRef)
	assert.Equal(t, resources.URN("my-data-graph", dgHandler.HandlerMetadata.ResourceType), userData.DataGraphRef.URN)

	accountResource, exists := graph.GetResource(resources.URN("account", modelHandler.HandlerMetadata.ResourceType))
	require.True(t, exists, "account model resource should exist")
	accountData, ok := accountResource.RawData().(*dgModel.ModelResource)
	require.True(t, ok)
	assert.Equal(t, "account", accountData.ID)
	assert.Equal(t, "Account", accountData.DisplayName)
	assert.Equal(t, "entity", accountData.Type)
	assert.Equal(t, "db.schema.accounts", accountData.Table)
	assert.Equal(t, "Account entity model", accountData.Description)
	assert.Equal(t, "account_id", accountData.PrimaryID)
	assert.False(t, accountData.Root)

	// Validate event model resources
	pageViewResource, exists := graph.GetResource(resources.URN("page_view", modelHandler.HandlerMetadata.ResourceType))
	require.True(t, exists, "page_view model resource should exist")
	pageViewData, ok := pageViewResource.RawData().(*dgModel.ModelResource)
	require.True(t, ok)
	assert.Equal(t, "page_view", pageViewData.ID)
	assert.Equal(t, "Page View", pageViewData.DisplayName)
	assert.Equal(t, "event", pageViewData.Type)
	assert.Equal(t, "db.schema.page_views", pageViewData.Table)
	assert.Equal(t, "Page view event", pageViewData.Description)
	assert.Equal(t, "viewed_at", pageViewData.Timestamp)

	purchaseResource, exists := graph.GetResource(resources.URN("purchase", modelHandler.HandlerMetadata.ResourceType))
	require.True(t, exists, "purchase model resource should exist")
	purchaseData, ok := purchaseResource.RawData().(*dgModel.ModelResource)
	require.True(t, ok)
	assert.Equal(t, "purchase", purchaseData.ID)
	assert.Equal(t, "Purchase", purchaseData.DisplayName)
	assert.Equal(t, "event", purchaseData.Type)
	assert.Equal(t, "db.schema.purchases", purchaseData.Table)
	assert.Empty(t, purchaseData.Description) // No description provided
	assert.Equal(t, "purchased_at", purchaseData.Timestamp)

	// Validate that models have PropertyRef to the data graph
	assert.NotNil(t, userData.DataGraphRef)
	assert.Equal(t, resources.URN("my-data-graph", dgHandler.HandlerMetadata.ResourceType), userData.DataGraphRef.URN)
	assert.NotNil(t, accountData.DataGraphRef)
	assert.Equal(t, resources.URN("my-data-graph", dgHandler.HandlerMetadata.ResourceType), accountData.DataGraphRef.URN)
	assert.NotNil(t, pageViewData.DataGraphRef)
	assert.Equal(t, resources.URN("my-data-graph", dgHandler.HandlerMetadata.ResourceType), pageViewData.DataGraphRef.URN)
	assert.NotNil(t, purchaseData.DataGraphRef)
	assert.Equal(t, resources.URN("my-data-graph", dgHandler.HandlerMetadata.ResourceType), purchaseData.DataGraphRef.URN)

	// Validate entity relationship: user-account
	userAccountResource, exists := graph.GetResource(resources.URN("user-account", relationshipHandler.HandlerMetadata.ResourceType))
	require.True(t, exists, "user-account relationship should exist")
	userAccountData, ok := userAccountResource.RawData().(*dgModel.RelationshipResource)
	require.True(t, ok)
	assert.Equal(t, "user-account", userAccountData.ID)
	assert.Equal(t, "User Account", userAccountData.DisplayName)
	assert.Equal(t, "one-to-one", userAccountData.Cardinality)
	assert.Equal(t, "account_id", userAccountData.SourceJoinKey)
	assert.Equal(t, "account_id", userAccountData.TargetJoinKey)
	assert.NotNil(t, userAccountData.DataGraphRef)
	assert.Equal(t, resources.URN("my-data-graph", dgHandler.HandlerMetadata.ResourceType), userAccountData.DataGraphRef.URN)
	assert.NotNil(t, userAccountData.SourceModelRef)
	assert.Equal(t, resources.URN("user", modelHandler.HandlerMetadata.ResourceType), userAccountData.SourceModelRef.URN)
	assert.NotNil(t, userAccountData.TargetModelRef)
	assert.Equal(t, resources.URN("account", modelHandler.HandlerMetadata.ResourceType), userAccountData.TargetModelRef.URN)

	// Validate event relationship: pageview-user
	pageViewUserResource, exists := graph.GetResource(resources.URN("pageview-user", relationshipHandler.HandlerMetadata.ResourceType))
	require.True(t, exists, "pageview-user relationship should exist")
	pageViewUserData, ok := pageViewUserResource.RawData().(*dgModel.RelationshipResource)
	require.True(t, ok)
	assert.Equal(t, "pageview-user", pageViewUserData.ID)
	assert.Equal(t, "PageView User", pageViewUserData.DisplayName)
	assert.Equal(t, "many-to-one", pageViewUserData.Cardinality) // Event relationships must have many-to-one cardinality
	assert.Equal(t, "user_id", pageViewUserData.SourceJoinKey)
	assert.Equal(t, "user_id", pageViewUserData.TargetJoinKey)
	assert.NotNil(t, pageViewUserData.DataGraphRef)
	assert.Equal(t, resources.URN("my-data-graph", dgHandler.HandlerMetadata.ResourceType), pageViewUserData.DataGraphRef.URN)
	assert.NotNil(t, pageViewUserData.SourceModelRef)
	assert.Equal(t, resources.URN("page_view", modelHandler.HandlerMetadata.ResourceType), pageViewUserData.SourceModelRef.URN)
	assert.NotNil(t, pageViewUserData.TargetModelRef)
	assert.Equal(t, resources.URN("user", modelHandler.HandlerMetadata.ResourceType), pageViewUserData.TargetModelRef.URN)

	// Validate total resources (1 data graph + 4 models + 2 relationships)
	allResources := graph.Resources()
	assert.Len(t, allResources, 7)
}

func TestLoadSpec_DataGraphWithoutModels(t *testing.T) {
	mockClient := &testutils.MockDataGraphClient{}
	provider := datagraph.NewProvider(mockClient, nil)

	spec := &specs.Spec{
		Version: "rudder/v0.1",
		Kind:    "data-graph",
		Spec: map[string]interface{}{
			"id":         "simple-dg",
			"account_id": "wh-123",
		},
	}

	err := provider.LoadSpec("test.yaml", spec)
	require.NoError(t, err)

	// Validate only data graph resource exists
	graph, err := provider.ResourceGraph()
	require.NoError(t, err)

	allResources := graph.Resources()
	assert.Len(t, allResources, 1)

	dgResource, exists := graph.GetResource(resources.URN("simple-dg", dgHandler.HandlerMetadata.ResourceType))
	require.True(t, exists)
	dgData, ok := dgResource.RawData().(*dgModel.DataGraphResource)
	require.True(t, ok)
	assert.Equal(t, &dgModel.DataGraphResource{
		ID:        "simple-dg",
		AccountID: "wh-123",
	}, dgData)
}

func TestLoadSpec_DataGraphValidationErrors(t *testing.T) {
	tests := []struct {
		name     string
		spec     map[string]interface{}
		errorMsg string
	}{
		{
			name: "missing id",
			spec: map[string]interface{}{
				"account_id": "wh-123",
			},
			errorMsg: "id is required",
		},
		{
			name: "missing account_id",
			spec: map[string]interface{}{
				"id": "test-dg",
			},
			errorMsg: "account_id is required",
		},
		{
			name: "empty id",
			spec: map[string]interface{}{
				"id":         "",
				"account_id": "wh-123",
			},
			errorMsg: "id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &testutils.MockDataGraphClient{}
			provider := datagraph.NewProvider(mockClient, nil)

			spec := &specs.Spec{
				Version: "rudder/v0.1",
				Kind:    "data-graph",
				Spec:    tt.spec,
			}

			err := provider.LoadSpec("test.yaml", spec)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errorMsg)
		})
	}
}

func TestLoadSpec_ModelValidationErrors(t *testing.T) {
	tests := []struct {
		name      string
		modelSpec map[string]interface{}
		errorMsg  string
	}{
		{
			name: "missing id",
			modelSpec: map[string]interface{}{
				"display_name": "User",
				"type":         "entity",
				"table":        "db.schema.users",
				"primary_id":   "id",
			},
			errorMsg: "id is required",
		},
		{
			name: "missing display_name",
			modelSpec: map[string]interface{}{
				"id":         "user",
				"table":      "db.schema.users",
				"primary_id": "id",
			},
			errorMsg: "display_name is required",
		},
		{
			name: "missing type",
			modelSpec: map[string]interface{}{
				"id":           "user",
				"display_name": "User",
				"table":        "db.schema.users",
				"primary_id":   "id",
			},
			errorMsg: "type must be 'entity' or 'event'",
		},
		{
			name: "invalid type",
			modelSpec: map[string]interface{}{
				"id":           "user",
				"display_name": "User",
				"type":         "invalid",
				"table":        "db.schema.users",
				"primary_id":   "id",
			},
			errorMsg: "type must be 'entity' or 'event'",
		},
		{
			name: "missing table",
			modelSpec: map[string]interface{}{
				"id":           "user",
				"display_name": "User",
				"type":         "entity",
				"primary_id":   "id",
			},
			errorMsg: "3-part reference",
		},
		{
			name: "invalid 1-part table ref",
			modelSpec: map[string]interface{}{
				"id":           "user",
				"display_name": "User",
				"type":         "entity",
				"table":        "users",
				"primary_id":   "id",
			},
			errorMsg: "3-part reference",
		},
		{
			name: "entity model missing primary_id",
			modelSpec: map[string]interface{}{
				"id":           "user",
				"display_name": "User",
				"type":         "entity",
				"table":        "db.schema.users",
			},
			errorMsg: "primary_id is required for entity models",
		},
		{
			name: "event model missing timestamp",
			modelSpec: map[string]interface{}{
				"id":           "purchase",
				"display_name": "Purchase",
				"type":         "event",
				"table":        "db.schema.purchases",
			},
			errorMsg: "timestamp is required for event models",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &testutils.MockDataGraphClient{}
			provider := datagraph.NewProvider(mockClient, nil)

			spec := &specs.Spec{
				Version: "rudder/v0.1",
				Kind:    "data-graph",
				Spec: map[string]interface{}{
					"id":         "test-dg",
					"account_id": "wh-123",
					"models": []map[string]interface{}{
						tt.modelSpec,
					},
				},
			}

			err := provider.LoadSpec("test.yaml", spec)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errorMsg)
		})
	}
}

func TestLoadSpec_DuplicateResourceIDs(t *testing.T) {
	mockClient := &testutils.MockDataGraphClient{}
	provider := datagraph.NewProvider(mockClient, nil)

	spec := &specs.Spec{
		Version: "rudder/v0.1",
		Kind:    "data-graph",
		Spec: map[string]interface{}{
			"id":         "test-dg",
			"account_id": "wh-123",
			"models": []map[string]interface{}{
				{
					"id":           "user",
					"display_name": "User",
					"type":         "entity",
					"table":        "db.schema.users",
					"primary_id":   "id",
				},
				{
					"id":           "user", // Duplicate ID
					"display_name": "User Duplicate",
					"type":         "entity",
					"table":        "db.schema.users2",
					"primary_id":   "id",
				},
			},
		},
	}

	err := provider.LoadSpec("test.yaml", spec)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestProvider_SupportedKinds(t *testing.T) {
	mockClient := &testutils.MockDataGraphClient{}
	provider := datagraph.NewProvider(mockClient, nil)

	kinds := provider.SupportedKinds()
	assert.Contains(t, kinds, "data-graph")
}

func TestProvider_SupportedTypes(t *testing.T) {
	mockClient := &testutils.MockDataGraphClient{}
	provider := datagraph.NewProvider(mockClient, nil)

	types := provider.SupportedTypes()
	assert.Contains(t, types, dgHandler.HandlerMetadata.ResourceType)
	assert.Contains(t, types, modelHandler.HandlerMetadata.ResourceType)
	assert.Contains(t, types, relationshipHandler.HandlerMetadata.ResourceType)
	assert.Len(t, types, 3)
}

func TestParseSpec_DataGraphWithInlineModels(t *testing.T) {
	mockClient := &testutils.MockDataGraphClient{}
	provider := datagraph.NewProvider(mockClient, nil)

	spec := &specs.Spec{
		Version: "rudder/v0.1",
		Kind:    "data-graph",
		Spec: map[string]interface{}{
			"id":         "my-data-graph",
			"name":       "My Data Graph",
			"account_id": "wh-account-123",
			"models": []map[string]interface{}{
				{
					"id":           "user",
					"display_name": "User",
					"type":         "entity",
					"table":        "db.schema.users",
					"primary_id":   "user_id",
					"relationships": []map[string]interface{}{
						{
							"id":              "user-orders",
							"display_name":    "User Orders",
							"cardinality":     "one-to-many",
							"target":          "#data-graph-model:order",
							"source_join_key": "user_id",
							"target_join_key": "user_id",
						},
					},
				},
				{
					"id":           "order",
					"display_name": "Order",
					"table":        "db.schema.orders",
					"primary_id":   "order_id",
				},
				{
					"id":           "purchase",
					"display_name": "Purchase",
					"type":         "event",
					"table":        "db.schema.purchases",
					"timestamp":    "purchased_at",
					"relationships": []map[string]interface{}{
						{
							"id":              "purchase-user",
							"display_name":    "Purchase User",
							"target":          "#data-graph-model:user",
							"source_join_key": "user_id",
							"target_join_key": "user_id",
						},
					},
				},
			},
		},
	}

	parsed, err := provider.ParseSpec("test.yaml", spec)
	require.NoError(t, err)
	require.NotNil(t, parsed)

	// Should return data graph ID plus all inline model IDs and relationship IDs as URNs
	expectedURNs := []specs.URNEntry{
		{URN: resources.URN("my-data-graph", dgHandler.HandlerMetadata.ResourceType), JSONPointerPath: "/spec/id"},
		{URN: resources.URN("user", modelHandler.HandlerMetadata.ResourceType), JSONPointerPath: "/spec/models/0/id"},
		{URN: resources.URN("order", modelHandler.HandlerMetadata.ResourceType), JSONPointerPath: "/spec/models/1/id"},
		{URN: resources.URN("purchase", modelHandler.HandlerMetadata.ResourceType), JSONPointerPath: "/spec/models/2/id"},
		{URN: resources.URN("user-orders", relationshipHandler.HandlerMetadata.ResourceType), JSONPointerPath: "/spec/models/0/relationships/0/id"},
		{URN: resources.URN("purchase-user", relationshipHandler.HandlerMetadata.ResourceType), JSONPointerPath: "/spec/models/2/relationships/0/id"},
	}
	assert.ElementsMatch(t, expectedURNs, parsed.URNs)
}

func TestLoadSpec_RelationshipCardinalityConstraints(t *testing.T) {
	tests := []struct {
		name          string
		sourceType    string
		targetType    string
		cardinality   string
		shouldSucceed bool
		errorContains string
	}{
		{
			name:          "event to event - rejected",
			sourceType:    "event",
			targetType:    "event",
			cardinality:   "many-to-one",
			shouldSucceed: false,
			errorContains: "event models cannot be connected to other event models",
		},
		{
			name:          "event to entity - valid many-to-one",
			sourceType:    "event",
			targetType:    "entity",
			cardinality:   "many-to-one",
			shouldSucceed: true,
		},
		{
			name:          "event to entity - invalid one-to-many",
			sourceType:    "event",
			targetType:    "entity",
			cardinality:   "one-to-many",
			shouldSucceed: false,
			errorContains: "must have cardinality 'many-to-one'",
		},
		{
			name:          "event to entity - invalid one-to-one",
			sourceType:    "event",
			targetType:    "entity",
			cardinality:   "one-to-one",
			shouldSucceed: false,
			errorContains: "must have cardinality 'many-to-one'",
		},
		{
			name:          "entity to event - valid one-to-many",
			sourceType:    "entity",
			targetType:    "event",
			cardinality:   "one-to-many",
			shouldSucceed: true,
		},
		{
			name:          "entity to event - invalid many-to-one",
			sourceType:    "entity",
			targetType:    "event",
			cardinality:   "many-to-one",
			shouldSucceed: false,
			errorContains: "must have cardinality 'one-to-many'",
		},
		{
			name:          "entity to event - invalid one-to-one",
			sourceType:    "entity",
			targetType:    "event",
			cardinality:   "one-to-one",
			shouldSucceed: false,
			errorContains: "must have cardinality 'one-to-many'",
		},
		{
			name:          "entity to entity - valid one-to-one",
			sourceType:    "entity",
			targetType:    "entity",
			cardinality:   "one-to-one",
			shouldSucceed: true,
		},
		{
			name:          "entity to entity - valid one-to-many",
			sourceType:    "entity",
			targetType:    "entity",
			cardinality:   "one-to-many",
			shouldSucceed: true,
		},
		{
			name:          "entity to entity - valid many-to-one",
			sourceType:    "entity",
			targetType:    "entity",
			cardinality:   "many-to-one",
			shouldSucceed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &testutils.MockDataGraphClient{}
			provider := datagraph.NewProvider(mockClient, nil)

			// Build spec with two models and a relationship
			spec := buildRelationshipTestSpec(tt.sourceType, tt.targetType, tt.cardinality)

			// Load spec
			err := provider.LoadSpec("test.yaml", spec)
			require.NoError(t, err, "LoadSpec should not fail")

			// Validate the graph - this triggers ValidateResource
			graph, graphErr := provider.ResourceGraph()
			require.NoError(t, graphErr)

			validateErr := provider.Validate(graph)

			if tt.shouldSucceed {
				require.NoError(t, validateErr, "Validation should pass")

				// Verify relationship exists
				relResource, exists := graph.GetResource(resources.URN("test-rel", relationshipHandler.HandlerMetadata.ResourceType))
				require.True(t, exists, "relationship resource should exist")
				require.NotNil(t, relResource)
			} else {
				require.Error(t, validateErr, "Validation should fail")
				assert.Contains(t, validateErr.Error(), tt.errorContains)
			}
		})
	}
}

// buildRelationshipTestSpec creates a test spec with a data graph, two models, and a relationship
func buildRelationshipTestSpec(sourceType, targetType, cardinality string) *specs.Spec {
	// Build source model
	sourceModel := map[string]interface{}{
		"id":           "source-model",
		"display_name": "Source Model",
		"type":         sourceType,
		"table":        "db.schema.source_table",
		"relationships": []map[string]interface{}{
			{
				"id":              "test-rel",
				"display_name":    "Test Relationship",
				"cardinality":     cardinality,
				"target":          "#data-graph-model:target-model",
				"source_join_key": "join_key",
				"target_join_key": "join_key",
			},
		},
	}

	// Add type-specific fields for source model
	if sourceType == "entity" {
		sourceModel["primary_id"] = "id"
		sourceModel["root"] = true
	} else {
		sourceModel["timestamp"] = "created_at"
	}

	// Build target model
	targetModel := map[string]interface{}{
		"id":           "target-model",
		"display_name": "Target Model",
		"type":         targetType,
		"table":        "db.schema.target_table",
	}

	// Add type-specific fields for target model
	if targetType == "entity" {
		targetModel["primary_id"] = "id"
		targetModel["root"] = false
	} else {
		targetModel["timestamp"] = "created_at"
	}

	return &specs.Spec{
		Version: "rudder/v0.1",
		Kind:    "data-graph",
		Spec: map[string]interface{}{
			"id":         "test-dg",
			"account_id": "wh-123",
			"models": []map[string]interface{}{
				sourceModel,
				targetModel,
			},
		},
	}
}

// Helper to build a RemoteResources collection for FormatForExport tests
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

	result, err := provider.FormatForExport(collection, nil, nil)
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

	// Verify metadata has import block with URNs for all resources
	metadata, err := spec.CommonMetadata()
	require.NoError(t, err)
	require.NotNil(t, metadata.Import)
	require.Len(t, metadata.Import.Workspaces, 1)

	ws := metadata.Import.Workspaces[0]
	assert.Equal(t, "ws-123", ws.WorkspaceID)

	// Should have 4 resources: 1 DG + 2 models + 1 valid relationship
	// (the unresolvable-target relationship is excluded)
	require.Len(t, ws.Resources, 4)

	// Collect URNs for verification
	urns := make(map[string]string) // URN -> RemoteID
	for _, r := range ws.Resources {
		urns[r.URN] = r.RemoteID
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

	result, err := provider.FormatForExport(collection, nil, nil)
	require.NoError(t, err)
	require.Len(t, result, 1)

	spec := result[0].Content.(*specs.Spec)
	assert.Equal(t, "simple-dg", spec.Spec["id"])
	assert.Nil(t, spec.Spec["models"])
	assert.Equal(t, "data-graphs/simple-dg.yaml", result[0].RelativePath)

	// Verify import metadata has only the data graph
	metadata, err := spec.CommonMetadata()
	require.NoError(t, err)
	require.Len(t, metadata.Import.Workspaces[0].Resources, 1)
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

	result, err := provider.FormatForExport(collection, nil, nil)
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

	result, err := provider.FormatForExport(collection, nil, nil)
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

	result, err := provider.FormatForExport(collection, nil, nil)
	require.NoError(t, err)
	require.Len(t, result, 1)

	spec := result[0].Content.(*specs.Spec)
	models, ok := spec.Spec["models"].([]dgModel.ModelSpec)
	require.True(t, ok)

	// Only model-remote-1 should be included (belongs to importable dg-remote-1)
	require.Len(t, models, 1)
	assert.Equal(t, "user", models[0].ID)

	// Verify import metadata only contains resources from the importable DG
	metadata, err := spec.CommonMetadata()
	require.NoError(t, err)
	ws := metadata.Import.Workspaces[0]

	urnSet := make(map[string]bool)
	for _, r := range ws.Resources {
		urnSet[r.URN] = true
	}

	assert.True(t, urnSet[resources.URN("importable-dg", dgHandler.HandlerMetadata.ResourceType)])
	assert.True(t, urnSet[resources.URN("user", modelHandler.HandlerMetadata.ResourceType)])
	// Orphan model should not appear
	assert.False(t, urnSet[resources.URN("orphan-model", modelHandler.HandlerMetadata.ResourceType)])
	assert.False(t, urnSet[fmt.Sprintf("%s:%s", modelHandler.HandlerMetadata.ResourceType, "orphan-model")])
}
