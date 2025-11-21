package resources_test

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
)

func TestCollectReferences(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected []*resources.PropertyRef
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
			input: &resources.PropertyRef{
				URN:      "test:urn",
				Property: "prop",
			},
			expected: []*resources.PropertyRef{
				{
					URN:      "test:urn",
					Property: "prop",
				},
			},
		},
		{
			name: "nested map with refs",
			input: map[string]interface{}{
				"ref1": &resources.PropertyRef{
					URN:      "test:urn1",
					Property: "prop1",
				},
				"nested": map[string]interface{}{
					"ref2": &resources.PropertyRef{
						URN:      "test:urn2",
						Property: "prop2",
					},
				},
			},
			expected: []*resources.PropertyRef{
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
				&resources.PropertyRef{
					URN:      "test:urn1",
					Property: "prop1",
				},
				&resources.PropertyRef{
					URN:      "test:urn2",
					Property: "prop2",
				},
			},
			expected: []*resources.PropertyRef{
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
			input: resources.ResourceData{
				"prop1": &resources.PropertyRef{
					URN:      "test:urn1",
					Property: "prop1",
				},
				"prop2": &resources.PropertyRef{
					URN:      "test:urn2",
					Property: "prop2",
				},
			},
			expected: []*resources.PropertyRef{
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
					"ref1": &resources.PropertyRef{
						URN:      "test:urn1",
						Property: "prop1",
					},
				},
				{
					"ref2": &resources.PropertyRef{
						URN:      "test:urn2",
						Property: "prop2",
					},
				},
			},
			expected: []*resources.PropertyRef{
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resources.CollectReferences(tt.input)
			assert.ElementsMatch(t, tt.expected, result)
		})
	}
}
