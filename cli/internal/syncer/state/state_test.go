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
	state.AddResource(&ResourceState{
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

	state.AddResource(&ResourceState{
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
		{
			name: "nested property reference in array of maps defined through map[string]interface{}",
			input: resources.ResourceData{
				"sourceName": []map[string]interface{}{
					{
						"nested": resources.PropertyRef{
							URN:      resources.URN("source1", "source"),
							Property: "name",
						},
					},
				},
			},
			expected: resources.ResourceData{
				"sourceName": []map[string]interface{}{
					{
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

func TestMerge(t *testing.T) {
	state1 := EmptyState()
	resource1 := &ResourceState{
		ID:   "source1",
		Type: "source",
		Input: map[string]any{
			"name": "input test source",
		},
		Output: map[string]any{
			"name": "test source",
			"id":   "src1",
		},
	}
	state1.AddResource(resource1)

	state2 := EmptyState()
	resource2 := &ResourceState{
		ID:   "dest1",
		Type: "destination",
		Input: map[string]any{
			"name": "input test destination",
		},
		Output: map[string]any{
			"name":     "test destination",
			"sourceId": "src1",
		},
	}
	state2.AddResource(resource2)

	state1.Merge(state2)

	assert.Equal(t, 2, len(state1.Resources))
	assert.NotNil(t, state1.GetResource(resources.URN(resource1.ID, resource1.Type)))
	assert.NotNil(t, state1.GetResource(resources.URN(resource2.ID, resource2.Type)))
	assert.Equal(t, resource1.Data(), state1.GetResource(resources.URN(resource1.ID, resource1.Type)).Data())
	assert.Equal(t, resource2.Data(), state1.GetResource(resources.URN(resource2.ID, resource2.Type)).Data())

	state3 := EmptyState()
	state3.Version = "incompatible_version"

	err := state1.Merge(state3)
	assert.NotNil(t, err)
	assert.ErrorIs(t, err, ErrIncompatibleState)
}
