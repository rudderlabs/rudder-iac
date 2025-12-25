package completion

import (
	"testing"

	protocol "github.com/tliron/glsp/protocol_3_16"

	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

// TestExtractContext tests the context extraction logic
func TestExtractContext(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		line     int
		char     int
		wantType CompletionType
		wantKind string
		wantGroup string
	}{
		{
			name:     "hash slash typed - reference start",
			content:  "  $ref: \"#/",
			line:     0,
			char:     11,
			wantType: CompletionTypeReferenceStart,
			wantKind: "",
			wantGroup: "",
		},
		{
			name:     "kind completion - properties",
			content:  "  $ref: \"#/properties/",
			line:     0,
			char:     22,
			wantType: CompletionTypeReferenceKind,
			wantKind: "properties",
			wantGroup: "",
		},
		{
			name:     "kind completion - events",
			content:  "  $ref: \"#/events/",
			line:     0,
			char:     18,
			wantType: CompletionTypeReferenceKind,
			wantKind: "events",
			wantGroup: "",
		},
		{
			name:     "resource completion - properties general",
			content:  "  $ref: \"#/properties/general/",
			line:     0,
			char:     30,
			wantType: CompletionTypeReferenceResource,
			wantKind: "properties",
			wantGroup: "general",
		},
		{
			name:     "resource completion - events tracking",
			content:  "  $ref: \"#/events/tracking/",
			line:     0,
			char:     27,
			wantType: CompletionTypeReferenceResource,
			wantKind: "events",
			wantGroup: "tracking",
		},
		{
			name:     "no reference pattern",
			content:  "  name: some_value",
			line:     0,
			char:     10,
			wantType: CompletionTypeNone,
			wantKind: "",
			wantGroup: "",
		},
		{
			name:     "partial kind",
			content:  "  $ref: \"#/prop",
			line:     0,
			char:     15,
			wantType: CompletionTypeReferenceStart,
			wantKind: "",
			wantGroup: "",
		},
		{
			name:     "multiline yaml with reference",
			content:  "metadata:\n  name: test\ndata:\n  $ref: \"#/custom-types/",
			line:     3,
			char:     24,
			wantType: CompletionTypeReferenceKind,
			wantKind: "custom-types",
			wantGroup: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := extractContext([]byte(tt.content), tt.line, tt.char)

			if ctx.completionType != tt.wantType {
				t.Errorf("completionType = %v, want %v", ctx.completionType, tt.wantType)
			}

			if ctx.kind != tt.wantKind {
				t.Errorf("kind = %v, want %v", ctx.kind, tt.wantKind)
			}

			if ctx.group != tt.wantGroup {
				t.Errorf("group = %v, want %v", ctx.group, tt.wantGroup)
			}
		})
	}
}

// TestGetKindCompletions tests kind completion generation
func TestGetKindCompletions(t *testing.T) {
	provider := NewProvider()
	items := provider.getKindCompletions()

	if len(items) != 4 {
		t.Errorf("expected 4 kind completions, got %d", len(items))
	}

	expectedKinds := map[string]bool{
		"properties":   false,
		"events":       false,
		"custom-types": false,
		"categories":   false,
	}

	for _, item := range items {
		if _, exists := expectedKinds[item.Label]; !exists {
			t.Errorf("unexpected kind: %s", item.Label)
		}
		expectedKinds[item.Label] = true

		// Verify structure
		if item.Kind == nil || *item.Kind != protocol.CompletionItemKindModule {
			t.Errorf("kind %s has incorrect CompletionItemKind", item.Label)
		}

		if item.InsertText == nil || *item.InsertText == "" {
			t.Errorf("kind %s has no InsertText", item.Label)
		}
	}

	// Verify all expected kinds are present
	for kind, found := range expectedKinds {
		if !found {
			t.Errorf("missing expected kind: %s", kind)
		}
	}
}

// TestGetCompletionsWithEmptyGraph tests handling of empty resource graph
func TestGetCompletionsWithEmptyGraph(t *testing.T) {
	provider := NewProvider()

	tests := []struct {
		name    string
		content string
		line    int
		char    int
	}{
		{
			name:    "reference start with nil graph",
			content: "  $ref: \"#/",
			line:    0,
			char:    11,
		},
		{
			name:    "kind completion with nil graph",
			content: "  $ref: \"#/properties/",
			line:    0,
			char:    22,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items, err := provider.GetCompletions(
				[]byte(tt.content),
				tt.line,
				tt.char,
				nil, // nil graph
			)

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if len(items) != 0 {
				t.Errorf("expected empty completions for nil graph, got %d items", len(items))
			}
		})
	}
}

// TestGetCompletionsReferenceStart tests completions when user types "#/"
func TestGetCompletionsReferenceStart(t *testing.T) {
	provider := NewProvider()
	graph := createMockResourceGraph()
	provider.RebuildCache(graph)

	content := []byte(`$ref: "#/`)
	items, err := provider.GetCompletions(content, 0, 10, graph)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(items) != 4 {
		t.Errorf("expected 4 kind completions, got %d", len(items))
	}

	// Verify we get the expected kinds
	kindLabels := make(map[string]bool)
	for _, item := range items {
		kindLabels[item.Label] = true
	}

	expectedKinds := []string{"properties", "events", "custom-types", "categories"}
	for _, kind := range expectedKinds {
		if !kindLabels[kind] {
			t.Errorf("missing expected kind: %s", kind)
		}
	}
}

// TestGetCompletionsReferenceKind tests completions when user types "#/properties/"
func TestGetCompletionsReferenceKind(t *testing.T) {
	provider := NewProvider()
	graph := createMockResourceGraph()
	provider.RebuildCache(graph)

	content := []byte(`$ref: "#/properties/`)
	items, err := provider.GetCompletions(content, 0, 20, graph)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(items) == 0 {
		t.Error("expected group completions, got none")
	}

	// Verify items have correct structure
	for _, item := range items {
		if item.Kind == nil || *item.Kind != protocol.CompletionItemKindFolder {
			t.Errorf("group %s has incorrect CompletionItemKind", item.Label)
		}

		if item.InsertText == nil || *item.InsertText == "" {
			t.Errorf("group %s has no InsertText", item.Label)
		}
	}
}

// TestGetCompletionsReferenceResource tests completions when user types full path
func TestGetCompletionsReferenceResource(t *testing.T) {
	provider := NewProvider()
	graph := createMockResourceGraph()
	provider.RebuildCache(graph)

	content := []byte(`$ref: "#/properties/general/`)
	items, err := provider.GetCompletions(content, 0, 28, graph)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(items) == 0 {
		t.Error("expected resource completions, got none")
	}

	// Verify items have correct structure
	for _, item := range items {
		if item.Kind == nil || *item.Kind != protocol.CompletionItemKindField {
			t.Errorf("resource %s has incorrect CompletionItemKind", item.Label)
		}

		if item.InsertText == nil || *item.InsertText == "" {
			t.Errorf("resource %s has no InsertText", item.Label)
		}
	}
}

// TestCacheRebuild tests cache rebuild functionality
func TestCacheRebuild(t *testing.T) {
	provider := NewProvider()
	graph := createMockResourceGraph()

	// Initially cache should be empty
	items := provider.cache.GetKindCompletions()
	if len(items) != 0 {
		t.Error("expected empty cache initially")
	}

	// Rebuild cache
	provider.RebuildCache(graph)

	// Now cache should have kind completions
	items = provider.cache.GetKindCompletions()
	if len(items) != 4 {
		t.Errorf("expected 4 kind completions after rebuild, got %d", len(items))
	}

	// Test group completions cache
	groupItems := provider.cache.GetGroupsForKind("properties")
	if groupItems == nil {
		t.Error("expected cached group completions for properties")
	}
}

// TestCacheConcurrency tests thread-safe cache access
func TestCacheConcurrency(t *testing.T) {
	provider := NewProvider()
	graph := createMockResourceGraph()
	provider.RebuildCache(graph)

	// Run concurrent reads
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			items := provider.cache.GetKindCompletions()
			if len(items) != 4 {
				t.Errorf("expected 4 items, got %d", len(items))
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestExtractGroupsForKind tests group extraction from resource graph
func TestExtractGroupsForKind(t *testing.T) {
	provider := NewProvider()
	graph := createMockResourceGraph()

	groups := provider.extractGroupsForKind("properties", graph)

	if len(groups) == 0 {
		t.Error("expected groups for properties, got none")
	}

	// Verify groups have counts
	for groupName, count := range groups {
		if count <= 0 {
			t.Errorf("group %s has invalid count: %d", groupName, count)
		}
	}
}

// TestGroupsFilteredByKind verifies that groups are correctly filtered per kind
// This tests the scenario where user types "#/properties/" and should ONLY see
// property groups, not category groups
func TestGroupsFilteredByKind(t *testing.T) {
	provider := NewProvider()

	// Create a graph mimicking user's setup:
	// - properties with group "api_tracking"
	// - categories with group "app_categories"
	graph := resources.NewGraph()

	// Add property resource
	propResource := resources.NewResource(
		"api_method",
		"property",
		map[string]interface{}{"name": "API Method"},
		[]string{},
		resources.WithResourceFileMetadata("#/properties/api_tracking/api_method"),
	)
	graph.AddResource(propResource)

	// Add category resource with different group
	catResource := resources.NewResource(
		"user_actions",
		"category",
		map[string]interface{}{"name": "User Actions"},
		[]string{},
		resources.WithResourceFileMetadata("#/categories/app_categories/user_actions"),
	)
	graph.AddResource(catResource)

	// Rebuild cache
	provider.RebuildCache(graph)

	// Test: When user types "#/properties/", they should see ONLY "api_tracking"
	content := []byte(`category: "#/properties/`)
	items, err := provider.GetCompletions(content, 0, 24, graph)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have exactly 1 group (api_tracking), NOT app_categories
	if len(items) != 1 {
		t.Errorf("expected exactly 1 group for properties, got %d", len(items))
		for _, item := range items {
			t.Logf("  - got item: %s", item.Label)
		}
	}

	// Verify the correct group is returned
	if len(items) > 0 && items[0].Label != "api_tracking" {
		t.Errorf("expected group 'api_tracking', got '%s'", items[0].Label)
	}

	// Also verify categories shows its own groups
	contentCat := []byte(`ref: "#/categories/`)
	itemsCat, err := provider.GetCompletions(contentCat, 0, 19, graph)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(itemsCat) != 1 {
		t.Errorf("expected exactly 1 group for categories, got %d", len(itemsCat))
	}

	if len(itemsCat) > 0 && itemsCat[0].Label != "app_categories" {
		t.Errorf("expected group 'app_categories', got '%s'", itemsCat[0].Label)
	}
}

// TestExtractResourcesForGroup tests resource extraction
func TestExtractResourcesForGroup(t *testing.T) {
	provider := NewProvider()
	graph := createMockResourceGraph()

	// Extract groups first to know which groups exist
	groups := provider.extractGroupsForKind("properties", graph)
	if len(groups) == 0 {
		t.Skip("no groups available for testing")
	}

	// Get first group name
	var groupName string
	for name := range groups {
		groupName = name
		break
	}

	resources := provider.extractResourcesForGroup("properties", groupName, graph)

	if len(resources) == 0 {
		t.Errorf("expected resources for group %s, got none", groupName)
	}

	// Verify resources have metadata
	for _, res := range resources {
		metadata := res.FileMetadata()
		if metadata == nil || metadata.MetadataRef == "" {
			t.Error("resource missing metadata or MetadataRef")
		}
	}
}

// TestExtractDescription tests description extraction from resources
func TestExtractDescription(t *testing.T) {
	provider := NewProvider()

	tests := []struct {
		name     string
		resource *resources.Resource
		wantDesc string
	}{
		{
			name: "description field present",
			resource: resources.NewResource(
				"test_id",
				"test_type",
				map[string]interface{}{
					"description": "Test description",
					"name":        "test_name",
				},
				[]string{},
			),
			wantDesc: "Test description",
		},
		{
			name: "only name field present",
			resource: resources.NewResource(
				"test_id",
				"test_type",
				map[string]interface{}{
					"name": "test_name",
				},
				[]string{},
			),
			wantDesc: "test_name",
		},
		{
			name: "no description or name",
			resource: resources.NewResource(
				"test_id",
				"test_type",
				map[string]interface{}{},
				[]string{},
			),
			wantDesc: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			desc := provider.extractDescription(tt.resource)

			if desc != tt.wantDesc {
				t.Errorf("extractDescription() = %v, want %v", desc, tt.wantDesc)
			}
		})
	}
}

// createMockResourceGraph creates a mock resource graph for testing
func createMockResourceGraph() *resources.Graph {
	graph := resources.NewGraph()

	// Add mock resources with different kinds and groups
	mockResources := []struct {
		kind         string
		group        string
		id           string
		desc         string
		resourceType string
	}{
		{"properties", "general", "globalId", "Global identifier for the user", "property"},
		{"properties", "general", "userId", "User unique identifier", "property"},
		{"properties", "api_tracking", "api_method", "HTTP method used for API call", "property"},
		{"properties", "api_tracking", "api_endpoint", "API endpoint path", "property"},
		{"events", "tracking", "page_viewed", "User viewed a page", "event"},
		{"events", "tracking", "button_clicked", "User clicked a button", "event"},
		{"custom-types", "common", "address", "Address type definition", "custom-type"},
		{"categories", "user", "identification", "User identification category", "category"},
	}

	for _, mock := range mockResources {
		ref := "#/" + mock.kind + "/" + mock.group + "/" + mock.id

		resource := resources.NewResource(
			mock.id,
			mock.resourceType,
			map[string]interface{}{
				"description": mock.desc,
				"name":        mock.id,
			},
			[]string{}, // no dependencies
			resources.WithResourceFileMetadata(ref),
		)

		graph.AddResource(resource)
	}

	return graph
}
