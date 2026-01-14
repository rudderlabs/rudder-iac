package rules

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExamples_HasExamples(t *testing.T) {
	tests := []struct {
		name     string
		examples Examples
		expected bool
	}{
		{
			name:     "no examples",
			examples: Examples{},
			expected: false,
		},
		{
			name: "only valid examples",
			examples: Examples{
				Valid: []string{"example 1"},
			},
			expected: true,
		},
		{
			name: "only invalid examples",
			examples: Examples{
				Invalid: []string{"bad example"},
			},
			expected: true,
		},
		{
			name: "both valid and invalid",
			examples: Examples{
				Valid:   []string{"good"},
				Invalid: []string{"bad"},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.examples.HasExamples())
		})
	}
}
