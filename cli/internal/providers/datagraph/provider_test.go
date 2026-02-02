package datagraph_test

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph"
	dgHandler "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/handlers/datagraph"
	modelHandler "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/handlers/model"
	dgModel "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/testutils"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadSpec_ComprehensiveDataGraphWithInlineModels(t *testing.T) {
	mockClient := &testutils.MockDataGraphClient{}
	provider := datagraph.NewProvider(mockClient)

	// Comprehensive spec with data graph and multiple inline models
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

	// Validate total resources (1 data graph + 4 models)
	allResources := graph.Resources()
	assert.Len(t, allResources, 5)
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
				"type":       "entity",
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
	assert.Len(t, types, 2)
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

	parsed, err := provider.ParseSpec("test.yaml", spec)
	require.NoError(t, err)
	require.NotNil(t, parsed)

	// Should return data graph ID plus all inline model IDs
	expectedIDs := []string{"my-data-graph", "user", "purchase"}
	assert.ElementsMatch(t, expectedIDs, parsed.ExternalIDs)
}
