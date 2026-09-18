package cmderrors

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	"github.com/stretchr/testify/assert"
)

func TestExitCode(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected int
	}{
		{
			name:     "no error",
			err:      nil,
			expected: ExitOK,
		},
		{
			// The commands wrap before returning, so classification has to survive
			// an arbitrary depth of %w.
			name:     "validation failure wrapped by the command",
			err:      fmt.Errorf("validating project: %w", fmt.Errorf("%w: syntax validation failed", project.ErrValidationFailed)),
			expected: ExitValidation,
		},
		{
			name:     "expired or rejected credentials",
			err:      fmt.Errorf("fetching workspace: %w", &client.APIError{HTTPStatusCode: http.StatusUnauthorized}),
			expected: ExitAuth,
		},
		{
			name:     "token lacks permission for the resource",
			err:      &client.APIError{HTTPStatusCode: http.StatusForbidden},
			expected: ExitAuth,
		},
		{
			name:     "no token configured at all",
			err:      fmt.Errorf("initialising dependencies: %w", app.ErrNotAuthenticated),
			expected: ExitAuth,
		},
		{
			name:     "upstream rejected the request",
			err:      &client.APIError{HTTPStatusCode: http.StatusBadRequest},
			expected: ExitUpstream,
		},
		{
			name:     "upstream unavailable",
			err:      &client.APIError{HTTPStatusCode: http.StatusBadGateway},
			expected: ExitUpstream,
		},
		{
			// Anything unclassified keeps the historical code so existing scripts
			// that check for exactly 1 do not regress.
			name:     "unclassified failure",
			err:      errors.New("something else went wrong"),
			expected: ExitError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, ExitCode(tt.err))
		})
	}
}
