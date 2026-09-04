package cmd

import (
	"fmt"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/cmderrors"
	"github.com/stretchr/testify/assert"
)

const wantPermissionDeniedMessage = "Access denied: Your configured token does not have sufficient permissions to perform this action."

func TestFormatExecutionError(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		wantMessage string
		wantPrint   bool
	}{
		{
			name: "formats permission denied API error",
			err: fmt.Errorf("syncing resources: %w", &client.APIError{
				HTTPStatusCode: 403,
				Message:        "Insufficient permissions",
			}),
			wantMessage: wantPermissionDeniedMessage,
			wantPrint:   true,
		},
		{
			name: "preserves silent errors",
			err:  &cmderrors.SilentError{Err: fmt.Errorf("validation failed")},
		},
		{
			name: "skips nil errors",
		},
		{
			name:        "falls through for other errors",
			err:         fmt.Errorf("loading project: invalid spec"),
			wantMessage: "loading project: invalid spec",
			wantPrint:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			message, ok := formatExecutionError(tt.err)
			assert.Equal(t, tt.wantPrint, ok)
			assert.Equal(t, tt.wantMessage, message)
		})
	}
}
