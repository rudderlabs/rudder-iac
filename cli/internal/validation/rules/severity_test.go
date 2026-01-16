package rules

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSeverity_String(t *testing.T) {
	tests := []struct {
		name     string
		severity Severity
		expected string
	}{
		{
			name:     "info level",
			severity: Info,
			expected: "info",
		},
		{
			name:     "warning level",
			severity: Warning,
			expected: "warning",
		},
		{
			name:     "error level",
			severity: Error,
			expected: "error",
		},
		{
			name:     "unknown level",
			severity: Severity(999),
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.severity.String())
		})
	}
}
