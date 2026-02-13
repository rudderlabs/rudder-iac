package testorchestrator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDefaultEvents(t *testing.T) {
	t.Run("returns non-nil map", func(t *testing.T) {
		events := GetDefaultEvents()
		require.NotNil(t, events)
		assert.NotEmpty(t, events)
	})

	t.Run("contains expected event types", func(t *testing.T) {
		events := GetDefaultEvents()

		expectedTypes := []string{"Track", "Identify", "Page", "Screen"}
		for _, eventType := range expectedTypes {
			assert.Contains(t, events, eventType, "should contain %s event", eventType)
		}
	})

	t.Run("each event is a map", func(t *testing.T) {
		events := GetDefaultEvents()

		for eventType, eventData := range events {
			assert.IsType(t, map[string]any{}, eventData, "%s event should be a map", eventType)
		}
	})

	t.Run("returns fresh copy on each call - mutations don't affect other calls", func(t *testing.T) {
		// Get first copy and mutate it
		events1 := GetDefaultEvents()
		events1["NewEvent"] = map[string]any{"test": "value"}

		// Get second copy
		events2 := GetDefaultEvents()

		// Second copy should not have the mutation
		assert.NotContains(t, events2, "NewEvent", "mutation should not affect new copy")
	})

	t.Run("nested mutations don't affect other calls", func(t *testing.T) {
		// Get first copy and mutate nested data
		events1 := GetDefaultEvents()
		if trackEvent, ok := events1["Track"].(map[string]any); ok {
			trackEvent["customField"] = "customValue"
		}

		// Get second copy
		events2 := GetDefaultEvents()

		// Second copy should not have the nested mutation
		if trackEvent, ok := events2["Track"].(map[string]any); ok {
			assert.NotContains(t, trackEvent, "customField", "nested mutation should not affect new copy")
		}
	})

	t.Run("Track event has expected structure", func(t *testing.T) {
		events := GetDefaultEvents()

		trackEvent, ok := events["Track"].(map[string]any)
		require.True(t, ok, "Track should be a map")

		assert.Contains(t, trackEvent, "type")
		assert.Equal(t, "track", trackEvent["type"])
		assert.Contains(t, trackEvent, "event")
		assert.Contains(t, trackEvent, "properties")
		assert.Contains(t, trackEvent, "userId")
	})

	t.Run("Identify event has expected structure", func(t *testing.T) {
		events := GetDefaultEvents()

		identifyEvent, ok := events["Identify"].(map[string]any)
		require.True(t, ok, "Identify should be a map")

		assert.Contains(t, identifyEvent, "type")
		assert.Equal(t, "identify", identifyEvent["type"])
		assert.Contains(t, identifyEvent, "userId")
		assert.Contains(t, identifyEvent, "context")
	})

	t.Run("Page event has expected structure", func(t *testing.T) {
		events := GetDefaultEvents()

		pageEvent, ok := events["Page"].(map[string]any)
		require.True(t, ok, "Page should be a map")

		assert.Contains(t, pageEvent, "type")
		assert.Equal(t, "page", pageEvent["type"])
		assert.Contains(t, pageEvent, "properties")
	})

	t.Run("Screen event has expected structure", func(t *testing.T) {
		events := GetDefaultEvents()

		screenEvent, ok := events["Screen"].(map[string]any)
		require.True(t, ok, "Screen should be a map")

		assert.Contains(t, screenEvent, "type")
		assert.Equal(t, "screen", screenEvent["type"])
		assert.Contains(t, screenEvent, "properties")
	})
}
