package syncer

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/differ"
	"github.com/stretchr/testify/assert"
)

func TestBuildNameIndex(t *testing.T) {
	t.Run("builds index from unmanaged resources", func(t *testing.T) {
		unmanaged := resources.NewRemoteResources()
		unmanaged.Set("category", map[string]*resources.RemoteResource{
			"cat_123": {
				ID:   "cat_123",
				Data: &mockNamedResource{Name: "Canvas"},
			},
			"cat_456": {
				ID:   "cat_456",
				Data: &mockNamedResource{Name: "Analytics"},
			},
		})
		unmanaged.Set("event", map[string]*resources.RemoteResource{
			"evt_789": {
				ID:   "evt_789",
				Data: &mockNamedResource{Name: "Page Viewed"},
			},
		})

		index := buildNameIndex(unmanaged)

		assert.Equal(t, map[string]map[string]differ.UnmanagedResource{
			"category": {
				"Canvas":    {RemoteID: "cat_123", Name: "Canvas"},
				"Analytics": {RemoteID: "cat_456", Name: "Analytics"},
			},
			"event": {
				"Page Viewed": {RemoteID: "evt_789", Name: "Page Viewed"},
			},
		}, index)
	})

	t.Run("skips resources without name", func(t *testing.T) {
		unmanaged := resources.NewRemoteResources()
		unmanaged.Set("category", map[string]*resources.RemoteResource{
			"cat_123": {
				ID:   "cat_123",
				Data: &mockNamedResource{Name: "Canvas"},
			},
			"cat_456": {
				ID:   "cat_456",
				Data: &mockNamedResource{Name: ""}, // Empty name
			},
			"cat_789": {
				ID:   "cat_789",
				Data: nil, // Nil data
			},
		})

		index := buildNameIndex(unmanaged)

		assert.Equal(t, map[string]map[string]differ.UnmanagedResource{
			"category": {
				"Canvas": {RemoteID: "cat_123", Name: "Canvas"},
			},
		}, index)
	})

	t.Run("handles empty collection", func(t *testing.T) {
		unmanaged := resources.NewRemoteResources()

		index := buildNameIndex(unmanaged)

		assert.Empty(t, index)
	})

	t.Run("handles map data type", func(t *testing.T) {
		unmanaged := resources.NewRemoteResources()
		unmanaged.Set("category", map[string]*resources.RemoteResource{
			"cat_123": {
				ID: "cat_123",
				Data: map[string]interface{}{
					"name":        "Canvas",
					"description": "Canvas events",
				},
			},
		})

		index := buildNameIndex(unmanaged)

		assert.Equal(t, map[string]map[string]differ.UnmanagedResource{
			"category": {
				"Canvas": {RemoteID: "cat_123", Name: "Canvas"},
			},
		}, index)
	})

	t.Run("skips duplicate names deterministically by sorting IDs", func(t *testing.T) {
		unmanaged := resources.NewRemoteResources()
		unmanaged.Set("category", map[string]*resources.RemoteResource{
			"cat_456": {
				ID:   "cat_456",
				Data: &mockNamedResource{Name: "Canvas"},
			},
			"cat_123": {
				ID:   "cat_123",
				Data: &mockNamedResource{Name: "Canvas"}, // Same name
			},
		})

		index := buildNameIndex(unmanaged)

		// Should only have one entry for "Canvas" - the one with lower ID (cat_123)
		assert.Len(t, index["category"], 1)
		assert.Contains(t, index["category"], "Canvas")
		// Verify deterministic behavior: cat_123 comes before cat_456 when sorted
		assert.Equal(t, "cat_123", index["category"]["Canvas"].RemoteID)
	})
}

func TestExtractName(t *testing.T) {
	t.Run("extracts name from struct with Name field", func(t *testing.T) {
		data := &mockNamedResource{Name: "Test Name"}
		assert.Equal(t, "Test Name", extractName(data))
	})

	t.Run("extracts name from map", func(t *testing.T) {
		data := map[string]interface{}{"name": "Map Name"}
		assert.Equal(t, "Map Name", extractName(data))
	})

	t.Run("returns empty for nil", func(t *testing.T) {
		assert.Equal(t, "", extractName(nil))
	})

	t.Run("returns empty for struct without Name field", func(t *testing.T) {
		data := &mockNoNameResource{ID: "123"}
		assert.Equal(t, "", extractName(data))
	})

	t.Run("returns empty for map without name key", func(t *testing.T) {
		data := map[string]interface{}{"title": "Title"}
		assert.Equal(t, "", extractName(data))
	})

	t.Run("handles pointer to struct", func(t *testing.T) {
		data := &mockNamedResource{Name: "Pointer Name"}
		assert.Equal(t, "Pointer Name", extractName(data))
	})

	t.Run("handles nil pointer", func(t *testing.T) {
		var data *mockNamedResource
		assert.Equal(t, "", extractName(data))
	})
}

func TestInjectImportMetadata(t *testing.T) {
	t.Run("injects metadata for confirmed matches", func(t *testing.T) {
		target := resources.NewGraph()
		target.AddResource(resources.NewResource(
			"canvas",
			"category",
			resources.ResourceData{"name": "Canvas"},
			[]string{},
		))

		confirmed := []differ.NameMatchCandidate{
			{
				LocalURN:     "category:canvas",
				RemoteID:     "cat_remote_123",
				RemoteName:   "Canvas",
				ResourceType: "category",
			},
		}

		result := injectImportMetadata(target, confirmed, "workspace-123")

		// Get the resource and verify import metadata was added
		res, found := result.GetResource("category:canvas")
		assert.True(t, found)
		assert.NotNil(t, res.ImportMetadata())
		assert.Equal(t, "cat_remote_123", res.ImportMetadata().RemoteId)
		assert.Equal(t, "workspace-123", res.ImportMetadata().WorkspaceId)
	})

	t.Run("handles empty confirmed list", func(t *testing.T) {
		target := resources.NewGraph()
		target.AddResource(resources.NewResource(
			"canvas",
			"category",
			resources.ResourceData{"name": "Canvas"},
			[]string{},
		))

		result := injectImportMetadata(target, []differ.NameMatchCandidate{}, "workspace-123")

		// Graph should be unchanged
		res, found := result.GetResource("category:canvas")
		assert.True(t, found)
		assert.Nil(t, res.ImportMetadata())
	})

	t.Run("handles multiple confirmed matches", func(t *testing.T) {
		target := resources.NewGraph()
		target.AddResource(resources.NewResource(
			"canvas",
			"category",
			resources.ResourceData{"name": "Canvas"},
			[]string{},
		))
		target.AddResource(resources.NewResource(
			"analytics",
			"category",
			resources.ResourceData{"name": "Analytics"},
			[]string{},
		))

		confirmed := []differ.NameMatchCandidate{
			{LocalURN: "category:canvas", RemoteID: "cat_1", RemoteName: "Canvas", ResourceType: "category"},
			{LocalURN: "category:analytics", RemoteID: "cat_2", RemoteName: "Analytics", ResourceType: "category"},
		}

		result := injectImportMetadata(target, confirmed, "workspace-123")

		res1, _ := result.GetResource("category:canvas")
		res2, _ := result.GetResource("category:analytics")

		assert.Equal(t, "cat_1", res1.ImportMetadata().RemoteId)
		assert.Equal(t, "cat_2", res2.ImportMetadata().RemoteId)
	})

	t.Run("preserves dependencies", func(t *testing.T) {
		target := resources.NewGraph()
		target.AddResource(resources.NewResource(
			"canvas",
			"category",
			resources.ResourceData{"name": "Canvas"},
			[]string{},
		))
		target.AddResource(resources.NewResource(
			"page-viewed",
			"event",
			resources.ResourceData{"name": "Page Viewed"},
			[]string{},
		))
		target.AddDependencies("event:page-viewed", []string{"category:canvas"})

		confirmed := []differ.NameMatchCandidate{
			{LocalURN: "category:canvas", RemoteID: "cat_1", RemoteName: "Canvas", ResourceType: "category"},
		}

		result := injectImportMetadata(target, confirmed, "workspace-123")

		// Verify dependencies are preserved
		deps := result.GetDependencies("event:page-viewed")
		assert.Contains(t, deps, "category:canvas")
	})
}

// Mock types for testing
type mockNamedResource struct {
	Name        string
	Description string
}

type mockNoNameResource struct {
	ID    string
	Title string
}
