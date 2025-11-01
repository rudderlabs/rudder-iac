package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCollectReferences(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected []*PropertyRef
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name:     "empty map",
			input:    map[string]interface{}{},
			expected: nil,
		},
		{
			name:     "empty slice",
			input:    []interface{}{},
			expected: nil,
		},
		{
			name: "single property ref",
			input: &PropertyRef{
				URN:      "test:urn",
				Property: "prop",
			},
			expected: []*PropertyRef{
				{
					URN:      "test:urn",
					Property: "prop",
				},
			},
		},
		{
			name: "nested map with refs",
			input: map[string]interface{}{
				"ref1": &PropertyRef{
					URN:      "test:urn1",
					Property: "prop1",
				},
				"nested": map[string]interface{}{
					"ref2": &PropertyRef{
						URN:      "test:urn2",
						Property: "prop2",
					},
				},
			},
			expected: []*PropertyRef{
				{
					URN:      "test:urn1",
					Property: "prop1",
				},
				{
					URN:      "test:urn2",
					Property: "prop2",
				},
			},
		},
		{
			name: "slice with refs",
			input: []interface{}{
				&PropertyRef{
					URN:      "test:urn1",
					Property: "prop1",
				},
				&PropertyRef{
					URN:      "test:urn2",
					Property: "prop2",
				},
			},
			expected: []*PropertyRef{
				{
					URN:      "test:urn1",
					Property: "prop1",
				},
				{
					URN:      "test:urn2",
					Property: "prop2",
				},
			},
		},
		{
			name: "resource data with refs",
			input: ResourceData{
				"prop1": &PropertyRef{
					URN:      "test:urn1",
					Property: "prop1",
				},
				"prop2": &PropertyRef{
					URN:      "test:urn2",
					Property: "prop2",
				},
			},
			expected: []*PropertyRef{
				{
					URN:      "test:urn1",
					Property: "prop1",
				},
				{
					URN:      "test:urn2",
					Property: "prop2",
				},
			},
		},
		{
			name: "array of maps with references",
			input: []map[string]interface{}{
				{
					"ref1": &PropertyRef{
						URN:      "test:urn1",
						Property: "prop1",
					},
				},
				{
					"ref2": &PropertyRef{
						URN:      "test:urn2",
						Property: "prop2",
					},
				},
			},
			expected: []*PropertyRef{
				{
					URN:      "test:urn1",
					Property: "prop1",
				},
				{
					URN:      "test:urn2",
					Property: "prop2",
				},
			},
		},
		{
			name: "struct with references",
			input: ExampleStruct{
				RefField: &PropertyRef{
					URN:      "test:urn1",
					Property: "prop1",
				},
				NonRefField: "non-ref",
				Nested: struct {
					RefInNested *PropertyRef
					NonRefField string
				}{
					RefInNested: &PropertyRef{
						URN:      "test:urn2",
						Property: "prop2",
					},
					NonRefField: "non-ref-nested",
				},
				ArrayField: []any{
					ExampleStruct{
						RefField: &PropertyRef{
							URN:      "test:urn3",
							Property: "prop3",
						},
					},
					ExampleStruct{
						RefField: &PropertyRef{
							URN:      "test:urn4",
							Property: "prop4",
						},
					},
				},
			},
			expected: []*PropertyRef{
				{
					URN:      "test:urn1",
					Property: "prop1",
				},
				{
					URN:      "test:urn2",
					Property: "prop2",
				},
				{
					URN:      "test:urn3",
					Property: "prop3",
				},
				{
					URN:      "test:urn4",
					Property: "prop4",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := collectReferencesByReflection(tt.input)
			assert.ElementsMatch(t, tt.expected, result)
		})
	}
}

type ExampleStruct struct {
	RefField    *PropertyRef
	NonRefField string
	Nested      struct {
		RefInNested *PropertyRef
		NonRefField string
	}
	ArrayField []any
}
