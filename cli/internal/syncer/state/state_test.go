package state

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/stretchr/testify/assert"
)

func TestDereference(t *testing.T) {
	// Setup test state
	state := EmptyState()

	// Add some resources to the state
	state.AddResource(&StateResource{
		ID:   "source1",
		Type: "source",
		Output: map[string]interface{}{
			"name": "test source",
			"id":   "src1",
			"config": map[string]interface{}{
				"enabled": true,
				"tags":    []interface{}{"tag1", "tag2"},
			},
		},
	})

	state.AddResource(&StateResource{
		ID:   "dest1",
		Type: "destination",
		Output: map[string]interface{}{
			"name":     "test destination",
			"sourceId": "src1",
		},
	})

	tests := []struct {
		name     string
		input    resources.ResourceData
		expected resources.ResourceData
	}{
		{
			name: "simple property reference",
			input: resources.ResourceData{
				"sourceName": resources.PropertyRef{
					URN:      resources.URN("source1", "source"),
					Property: "name",
				},
			},
			expected: resources.ResourceData{
				"sourceName": "test source",
			},
		},
		{
			name: "nested property reference in map",
			input: resources.ResourceData{
				"sourceName": map[string]interface{}{
					"nested": resources.PropertyRef{
						URN:      resources.URN("source1", "source"),
						Property: "name",
					},
				},
			},
			expected: resources.ResourceData{
				"sourceName": map[string]interface{}{
					"nested": "test source",
				},
			},
		},
		{
			name: "nested property reference in array",
			input: resources.ResourceData{
				"sourceName": []interface{}{
					resources.PropertyRef{
						URN:      resources.URN("source1", "source"),
						Property: "name",
					},
				},
			},
			expected: resources.ResourceData{
				"sourceName": []interface{}{
					"test source",
				},
			},
		},
		{
			name: "nested property reference in array of maps",
			input: resources.ResourceData{
				"sourceName": []interface{}{
					map[string]interface{}{
						"nested": resources.PropertyRef{
							URN:      resources.URN("source1", "source"),
							Property: "name",
						},
					},
				},
			},
			expected: resources.ResourceData{
				"sourceName": []interface{}{
					map[string]interface{}{
						"nested": "test source",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Dereference(tt.input, state)
			assert.Nil(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
