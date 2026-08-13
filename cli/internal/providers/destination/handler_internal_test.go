package destination

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsEmptyConfigValue_TypedContainers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value any
		empty bool
	}{
		{
			name:  "typed slice with only empty strings is empty",
			value: []string{""},
			empty: true,
		},
		{
			name:  "typed slice with a populated string is not empty",
			value: []string{"", "Order Completed"},
			empty: false,
		},
		{
			name:  "typed map with only empty values is empty",
			value: map[string]string{"provider": ""},
			empty: true,
		},
		{
			name:  "false is not empty",
			value: false,
			empty: false,
		},
		{
			name:  "zero is not empty",
			value: 0,
			empty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.empty, isEmptyConfigValue(tt.value))
		})
	}
}
