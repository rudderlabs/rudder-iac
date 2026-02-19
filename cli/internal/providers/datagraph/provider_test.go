package datagraph_test

import (
	"testing"

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
	provider := datagraph.NewProvider(mockClient)

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
					"table":        "users",
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
					"table":        "accounts",
					"description":  "Account entity model",
					"primary_id":   "account_id",
					"root":         false,
				},
				{
					"id":           "page_view",
					"display_name": "Page View",
					"type":         "event",
					"table":        "page_views",
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
					"table":        "purchases",
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
	assert.Equal(t, "users", userData.Table)
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
	assert.Equal(t, "accounts", accountData.Table)
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
	assert.Equal(t, "page_views", pageViewData.Table)
	assert.Equal(t, "Page view event", pageViewData.Description)
	assert.Equal(t, "viewed_at", pageViewData.Timestamp)

	purchaseResource, exists := graph.GetResource(resources.URN("purchase", modelHandler.HandlerMetadata.ResourceType))
	require.True(t, exists, "purchase model resource should exist")
	purchaseData, ok := purchaseResource.RawData().(*dgModel.ModelResource)
	require.True(t, ok)
	assert.Equal(t, "purchase", purchaseData.ID)
	assert.Equal(t, "Purchase", purchaseData.DisplayName)
	assert.Equal(t, "event", purchaseData.Type)
	assert.Equal(t, "purchases", purchaseData.Table)
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
	provider := datagraph.NewProvider(mockClient)

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
			provider := datagraph.NewProvider(mockClient)

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
				"table":        "users",
				"primary_id":   "id",
			},
			errorMsg: "id is required",
		},
		{
			name: "missing display_name",
			modelSpec: map[string]interface{}{
				"id":         "user",
				"table":      "users",
				"primary_id": "id",
			},
			errorMsg: "display_name is required",
		},
		{
			name: "missing type",
			modelSpec: map[string]interface{}{
				"id":           "user",
				"display_name": "User",
				"table":        "users",
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
				"table":        "users",
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
			errorMsg: "table is required",
		},
		{
			name: "entity model missing primary_id",
			modelSpec: map[string]interface{}{
				"id":           "user",
				"display_name": "User",
				"type":         "entity",
				"table":        "users",
			},
			errorMsg: "primary_id is required for entity models",
		},
		{
			name: "event model missing timestamp",
			modelSpec: map[string]interface{}{
				"id":           "purchase",
				"display_name": "Purchase",
				"type":         "event",
				"table":        "purchases",
			},
			errorMsg: "timestamp is required for event models",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &testutils.MockDataGraphClient{}
			provider := datagraph.NewProvider(mockClient)

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
	provider := datagraph.NewProvider(mockClient)

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
					"table":        "users",
					"primary_id":   "id",
				},
				{
					"id":           "user", // Duplicate ID
					"display_name": "User Duplicate",
					"type":         "entity",
					"table":        "users2",
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
	provider := datagraph.NewProvider(mockClient)

	kinds := provider.SupportedKinds()
	assert.Contains(t, kinds, "data-graph")
}

func TestProvider_SupportedTypes(t *testing.T) {
	mockClient := &testutils.MockDataGraphClient{}
	provider := datagraph.NewProvider(mockClient)

	types := provider.SupportedTypes()
	assert.Contains(t, types, dgHandler.HandlerMetadata.ResourceType)
	assert.Contains(t, types, modelHandler.HandlerMetadata.ResourceType)
	assert.Contains(t, types, relationshipHandler.HandlerMetadata.ResourceType)
	assert.Len(t, types, 3)
}

func TestParseSpec_DataGraphWithInlineModels(t *testing.T) {
	mockClient := &testutils.MockDataGraphClient{}
	provider := datagraph.NewProvider(mockClient)

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
					"table":        "users",
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
					"table":        "orders",
					"primary_id":   "order_id",
				},
				{
					"id":           "purchase",
					"display_name": "Purchase",
					"type":         "event",
					"table":        "purchases",
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

	// Should return data graph ID plus all inline model IDs and relationship IDs
	expectedLocalIDs := []specs.LocalID{
		{ID: "my-data-graph", JSONPointerPath: "/spec/id"},
		{ID: "user", JSONPointerPath: "/spec/models/0/id"},
		{ID: "order", JSONPointerPath: "/spec/models/1/id"},
		{ID: "purchase", JSONPointerPath: "/spec/models/2/id"},
		{ID: "user-orders", JSONPointerPath: "/spec/models/0/relationships/0/id"},
		{ID: "purchase-user", JSONPointerPath: "/spec/models/2/relationships/0/id"},
	}
	assert.ElementsMatch(t, expectedLocalIDs, parsed.LocalIDs)
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
			provider := datagraph.NewProvider(mockClient)

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
		"table":        "source_table",
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
		"table":        "target_table",
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
