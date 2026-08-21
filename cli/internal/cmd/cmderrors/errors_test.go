package cmderrors_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/cmderrors"
	"github.com/stretchr/testify/assert"
)

const (
	wantFeatureDisabledMessage  = "This feature is not enabled for your workspace. Please contact support to enable it."
	wantPermissionDeniedMessage = "Access denied: Your configured token does not have sufficient permissions to perform this action."
)

func TestFormatUserFacingError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "formats wrapped permission denied API error",
			err: fmt.Errorf("syncing resources: %w", &client.APIError{
				HTTPStatusCode: 403,
				Message:        "Insufficient permissions",
			}),
			want: wantPermissionDeniedMessage,
		},
		{
			name: "formats wrapped feature disabled API error",
			err: fmt.Errorf("listing data graphs: %w", &client.APIError{
				HTTPStatusCode: 403,
				Message:        "Feature is not enabled for your account: DATA_GRAPH",
			}),
			want: wantFeatureDisabledMessage,
		},
		{
			name: "falls through for non API errors",
			err:  errors.New("loading config: missing token"),
			want: "loading config: missing token",
		},
		{
			name: "falls through for non forbidden API errors",
			err: &client.APIError{
				HTTPStatusCode: 500,
				Message:        "server unavailable",
			},
			want: "http status code: 500, error code: '', error: 'server unavailable'",
		},
		{
			name: "handles nil error",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, cmderrors.FormatUserFacingError(tt.err))
		})
	}
}
