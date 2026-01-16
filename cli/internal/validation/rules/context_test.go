package rules

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
)

func TestValidationContext_HasGraph(t *testing.T) {
	tests := []struct {
		name     string
		ctx      ValidationContext
		expected bool
	}{
		{
			name:     "no graph",
			ctx:      ValidationContext{},
			expected: false,
		},
		{
			name: "with graph",
			ctx: ValidationContext{
				Graph: resources.NewGraph(),
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.ctx.HasGraph())
		})
	}
}
