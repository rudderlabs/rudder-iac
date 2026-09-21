package provider_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
)

func TestExplainBlockingConnections(t *testing.T) {
	t.Parallel()

	t.Run("names both env vars and keeps the backend error", func(t *testing.T) {
		t.Parallel()

		for _, msg := range []string{
			"The destination has active connections, please delete those first",
			"The source has active connections, please delete those first",
		} {
			backend := &client.APIError{HTTPStatusCode: 400, Message: msg}

			err := provider.ExplainBlockingConnections(fmt.Errorf("deleting RETL source: %w", backend))

			var apiErr *client.APIError
			require.ErrorAs(t, err, &apiErr)
			assert.Equal(t, backend, apiErr)
			assert.Contains(t, err.Error(), "RUDDERSTACK_CLI_EXPERIMENTAL=true RUDDERSTACK_X_RETL_CONNECTION_SUPPORT=true")
		}
	})

	// An unrelated failure must not pick up advice about a flag that cannot fix it.
	t.Run("passes other failures through untouched", func(t *testing.T) {
		t.Parallel()

		for _, err := range []error{
			&client.APIError{HTTPStatusCode: 400, Message: "destination is referenced by a running job"},
			&client.APIError{HTTPStatusCode: 500, Message: "active connections"},
			errors.New("active connections"),
		} {
			assert.Same(t, err, provider.ExplainBlockingConnections(err))
		}
	})
}
