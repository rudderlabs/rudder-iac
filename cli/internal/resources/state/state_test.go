package state_test

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	s "github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
	"github.com/stretchr/testify/assert"
)

func TestDereference(t *testing.T) {
	// Setup test state
	state := s.EmptyState()

	// Add some resources to the state
	state.AddResource(&s.ResourceState{
		ID:   "source1",
		Type: "source",
		Output: map[string]any{
			"name": "test source",
			"id":   "src1",
			"config": map[string]any{
				"enabled": true,
				"tags":    []any{"tag1", "tag2"},
			},
		},
	})

	state.AddResource(&s.ResourceState{
		ID:   "dest1",
		Type: "destination",
		Output: map[string]any{
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
				"sourceName": map[string]any{
					"nested": resources.PropertyRef{
						URN:      resources.URN("source1", "source"),
						Property: "name",
					},
				},
			},
			expected: resources.ResourceData{
				"sourceName": map[string]any{
					"nested": "test source",
				},
			},
		},
		{
			name: "nested property reference in array",
			input: resources.ResourceData{
				"sourceName": []any{
					resources.PropertyRef{
						URN:      resources.URN("source1", "source"),
						Property: "name",
					},
				},
			},
			expected: resources.ResourceData{
				"sourceName": []any{
					"test source",
				},
			},
		},
		{
			name: "nested property reference in array of maps",
			input: resources.ResourceData{
				"sourceName": []any{
					map[string]any{
						"nested": resources.PropertyRef{
							URN:      resources.URN("source1", "source"),
							Property: "name",
						},
					},
				},
			},
			expected: resources.ResourceData{
				"sourceName": []any{
					map[string]any{
						"nested": "test source",
					},
				},
			},
		},
		{
			name: "nested property reference in array of maps defined through map[string]any",
			input: resources.ResourceData{
				"sourceName": []map[string]any{
					{
						"nested": resources.PropertyRef{
							URN:      resources.URN("source1", "source"),
							Property: "name",
						},
					},
				},
			},
			expected: resources.ResourceData{
				"sourceName": []map[string]any{
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

func TestDereferenceByReflection(t *testing.T) {
	// Setup test state
	state := s.EmptyState()

	// Add some resources to the state
	state.AddResource(&s.ResourceState{
		ID:   "source1",
		Type: "source",
		Output: map[string]any{
			"name": "test source",
			"id":   "src1",
		},
	})

	state.AddResource(&s.ResourceState{
		ID:   "dest1",
		Type: "destination",
		Output: map[string]any{
			"name":     "test destination",
			"sourceId": "src1",
		},
	})

	type ExampleStruct struct {
		Name      string
		Enabled   bool
		Connected *resources.PropertyRef // Changed to pointer
	}

	input := ExampleStruct{
		Name:    "example",
		Enabled: true,
		Connected: &resources.PropertyRef{ // Changed to pointer
			URN:      resources.URN("dest1", "destination"),
			Property: "name",
		},
	}

	expected := ExampleStruct{
		Name:    "example",
		Enabled: true,
		Connected: &resources.PropertyRef{ // Changed to pointer
			URN:        resources.URN("dest1", "destination"),
			Property:   "name",
			IsResolved: true,
			Value:      "test destination",
		},
	}

	err := s.DereferenceByReflection(&input, state) // Pass pointer and check error
	assert.Nil(t, err)
	assert.Equal(t, expected, input) // Compare input after modification
}

func TestDereferenceByReflectionNested(t *testing.T) {
	// Setup test state
	state := s.EmptyState()

	state.AddResource(&s.ResourceState{
		ID:   "source1",
		Type: "source",
		Output: map[string]any{
			"name": "test source",
			"id":   "src1",
		},
	})

	state.AddResource(&s.ResourceState{
		ID:   "dest1",
		Type: "destination",
		Output: map[string]any{
			"name":     "test destination",
			"sourceId": "src1",
		},
	})

	type NestedStruct struct {
		DestinationRef *resources.PropertyRef
		Tags           []string
	}

	type ComplexStruct struct {
		Name      string
		SourceRef *resources.PropertyRef
		Nested    NestedStruct
		NestedPtr *NestedStruct
		RefSlice  []*resources.PropertyRef
		StringMap map[string]*resources.PropertyRef
	}

	input := ComplexStruct{
		Name: "complex example",
		SourceRef: &resources.PropertyRef{
			URN:      resources.URN("source1", "source"),
			Property: "name",
		},
		Nested: NestedStruct{
			DestinationRef: &resources.PropertyRef{
				URN:      resources.URN("dest1", "destination"),
				Property: "name",
			},
			Tags: []string{"tag1", "tag2"},
		},
		NestedPtr: &NestedStruct{
			DestinationRef: &resources.PropertyRef{
				URN:      resources.URN("dest1", "destination"),
				Property: "sourceId",
			},
			Tags: []string{"tag3"},
		},
		RefSlice: []*resources.PropertyRef{
			{
				URN:      resources.URN("source1", "source"),
				Property: "id",
			},
		},
		StringMap: map[string]*resources.PropertyRef{
			"key1": {
				URN:      resources.URN("dest1", "destination"),
				Property: "name",
			},
		},
	}

	err := s.DereferenceByReflection(&input, state)
	assert.Nil(t, err)

	// Verify all PropertyRefs have been resolved
	assert.True(t, input.SourceRef.IsResolved)
	assert.Equal(t, "test source", input.SourceRef.Value)

	assert.True(t, input.Nested.DestinationRef.IsResolved)
	assert.Equal(t, "test destination", input.Nested.DestinationRef.Value)

	assert.True(t, input.NestedPtr.DestinationRef.IsResolved)
	assert.Equal(t, "src1", input.NestedPtr.DestinationRef.Value)

	assert.True(t, input.RefSlice[0].IsResolved)
	assert.Equal(t, "src1", input.RefSlice[0].Value)

	assert.True(t, input.StringMap["key1"].IsResolved)
	assert.Equal(t, "test destination", input.StringMap["key1"].Value)
}

func TestMerge(t *testing.T) {
	state1 := s.EmptyState()
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

func TestDereferenceByReflectionWithResolveFunction(t *testing.T) {
	// Setup test state with typed OutputRaw
	state := s.EmptyState()

	type SourceStateRemote struct {
		ID             string
		TrackingPlanID string
	}

	// Add resource with typed OutputRaw
	state.AddResource(&s.ResourceState{
		ID:   "source1",
		Type: "event-stream-source",
		OutputRaw: &SourceStateRemote{
			ID:             "remote-id-123",
			TrackingPlanID: "tp-456",
		},
	})

	type TestStruct struct {
		Name            string
		SourceIDRef     *resources.PropertyRef
		TrackingPlanRef *resources.PropertyRef
	}

	// Create PropertyRefs with Resolve functions
	input := TestStruct{
		Name: "test",
		SourceIDRef: &resources.PropertyRef{
			URN: resources.URN("source1", "event-stream-source"),
			Resolve: func(outputRaw any) (string, error) {
				typed, ok := outputRaw.(*SourceStateRemote)
				if !ok {
					return "", assert.AnError
				}
				return typed.ID, nil
			},
		},
		TrackingPlanRef: &resources.PropertyRef{
			URN: resources.URN("source1", "event-stream-source"),
			Resolve: func(outputRaw any) (string, error) {
				typed, ok := outputRaw.(*SourceStateRemote)
				if !ok {
					return "", assert.AnError
				}
				return typed.TrackingPlanID, nil
			},
		},
	}

	err := s.DereferenceByReflection(&input, state)
	assert.Nil(t, err)

	// Verify PropertyRefs were resolved
	assert.True(t, input.SourceIDRef.IsResolved)
	assert.Equal(t, "remote-id-123", input.SourceIDRef.Value)

	assert.True(t, input.TrackingPlanRef.IsResolved)
	assert.Equal(t, "tp-456", input.TrackingPlanRef.Value)
}

func TestDereferenceByReflectionTypeSafetyError(t *testing.T) {
	// Setup test state with typed OutputRaw
	state := s.EmptyState()

	type SourceStateRemote struct {
		ID string
	}

	type WrongStateType struct {
		DifferentField string
	}

	// Add resource with one type
	state.AddResource(&s.ResourceState{
		ID:   "source1",
		Type: "event-stream-source",
		OutputRaw: &WrongStateType{
			DifferentField: "value",
		},
	})

	type TestStruct struct {
		SourceIDRef *resources.PropertyRef
	}

	// Create PropertyRef expecting different type
	input := TestStruct{
		SourceIDRef: &resources.PropertyRef{
			URN: resources.URN("source1", "event-stream-source"),
			Resolve: func(outputRaw any) (string, error) {
				// This type assertion should fail
				typed, ok := outputRaw.(*SourceStateRemote)
				if !ok {
					return "", assert.AnError
				}
				return typed.ID, nil
			},
		},
	}

	err := s.DereferenceByReflection(&input, state)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "resolving property ref")
}

func TestDereferenceByReflectionBackwardCompatibility(t *testing.T) {
	// Setup test state with both Output (old style) and OutputRaw (new style)
	state := s.EmptyState()

	type NewStateType struct {
		RemoteID string
	}

	// Old-style resource with Output map
	state.AddResource(&s.ResourceState{
		ID:   "old-resource",
		Type: "old-type",
		Output: map[string]any{
			"name": "old resource",
			"id":   "old-123",
		},
	})

	// New-style resource with OutputRaw
	state.AddResource(&s.ResourceState{
		ID:   "new-resource",
		Type: "new-type",
		OutputRaw: &NewStateType{
			RemoteID: "new-456",
		},
	})

	type TestStruct struct {
		OldRef *resources.PropertyRef
		NewRef *resources.PropertyRef
	}

	// Mix of old Property-based ref and new Resolve-based ref
	input := TestStruct{
		OldRef: &resources.PropertyRef{
			URN:      resources.URN("old-resource", "old-type"),
			Property: "id", // Old style - string property lookup
		},
		NewRef: &resources.PropertyRef{
			URN: resources.URN("new-resource", "new-type"),
			Resolve: func(outputRaw any) (string, error) {
				typed, ok := outputRaw.(*NewStateType)
				if !ok {
					return "", assert.AnError
				}
				return typed.RemoteID, nil
			},
		},
	}

	err := s.DereferenceByReflection(&input, state)
	assert.Nil(t, err)

	// Both refs should be resolved
	assert.True(t, input.OldRef.IsResolved)
	assert.Equal(t, "old-123", input.OldRef.Value)

	assert.True(t, input.NewRef.IsResolved)
	assert.Equal(t, "new-456", input.NewRef.Value)
}
