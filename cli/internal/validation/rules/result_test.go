package rules

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidationResult_SeverityChecks(t *testing.T) {
	tests := []struct {
		name      string
		severity  Severity
		isError   bool
		isWarning bool
		isInfo    bool
	}{
		{
			name:      "error severity",
			severity:  Error,
			isError:   true,
			isWarning: false,
			isInfo:    false,
		},
		{
			name:      "warning severity",
			severity:  Warning,
			isError:   false,
			isWarning: true,
			isInfo:    false,
		},
		{
			name:      "info severity",
			severity:  Info,
			isError:   false,
			isWarning: false,
			isInfo:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidationResult{Severity: tt.severity}

			assert.Equal(t, tt.isError, result.IsError())
			assert.Equal(t, tt.isWarning, result.IsWarning())
			assert.Equal(t, tt.isInfo, result.IsInfo())
		})
	}
}