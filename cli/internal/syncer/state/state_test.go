package state_test

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	s "github.com/rudderlabs/rudder-iac/cli/internal/syncer/state"
	"github.com/stretchr/testify/assert"
)

func TestDereference(t *testing.T) {
	// Setup test state
	state := s.EmptyState()

	// Add some resources to the state
	state.AddResource(&s.ResourceState{
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

	state.AddResource(&s.ResourceState{
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
			result, err := s.Dereference(tt.input, state)
			assert.Nil(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMerge(t *testing.T) {
	state1 := s.EmptyState()
	state1.Version = "1.0.0"
	resource1 := &s.ResourceState{
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

	state2 := s.EmptyState()
	state2.Version = "1.0.0"
	resource2 := &s.ResourceState{
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

	mergedState, err := state1.Merge(state2)
	assert.Nil(t, err)

	assert.Equal(t, state1.Version, mergedState.Version)
	assert.Equal(t, 2, len(mergedState.Resources))
	assert.NotNil(t, mergedState.GetResource(resources.URN(resource1.ID, resource1.Type)))
	assert.NotNil(t, mergedState.GetResource(resources.URN(resource2.ID, resource2.Type)))
	assert.Equal(t, resource1.Data(), mergedState.GetResource(resources.URN(resource1.ID, resource1.Type)).Data())
	assert.Equal(t, resource2.Data(), mergedState.GetResource(resources.URN(resource2.ID, resource2.Type)).Data())

	// ensure original states are unchanged
	assert.Equal(t, 1, len(state1.Resources))
	assert.Equal(t, 1, len(state2.Resources))
	assert.Equal(t, state1.Resources[resources.URN(resource1.ID, resource1.Type)].Data(), resource1.Data())
	assert.Equal(t, state2.Resources[resources.URN(resource2.ID, resource2.Type)].Data(), resource2.Data())
}

func TestMergeIncompatibleVersion(t *testing.T) {
	state1 := s.EmptyState()
	state2 := s.EmptyState()
	state2.Version = "incompatible_version"

	mergedState, err := state1.Merge(state2)
	assert.Nil(t, mergedState)
	assert.NotNil(t, err)
	assert.IsType(t, &s.ErrIncompatibleVersion{}, err)
	assert.Equal(t, err.(*s.ErrIncompatibleVersion).Version, "incompatible_version")
	assert.Equal(t, "incompatible state version: incompatible_version", err.Error())
}

func TestMergeURNAlreadyExists(t *testing.T) {
	state1 := s.EmptyState()
	resource1 := &s.ResourceState{
		ID:   "source1",
		Type: "source",
		Output: map[string]any{
			"name": "test source",
			"id":   "src1",
		},
	}
	state1.AddResource(resource1)

	state2 := s.EmptyState()
	resource2 := &s.ResourceState{
		ID:   "source1", // Same ID as resource1
		Type: "source",
		Output: map[string]any{
			"name": "another test source",
			"id":   "src2",
		},
	}
	state2.AddResource(resource2)

	mergedState, err := state1.Merge(state2)
	assert.Nil(t, mergedState)
	assert.NotNil(t, err)
	assert.IsType(t, &s.ErrURNAlreadyExists{}, err)
	assert.Equal(t, err.(*s.ErrURNAlreadyExists).URN, resources.URN("source1", "source"))
	assert.Equal(t, "URN already exists: source:source1", err.Error())
}
