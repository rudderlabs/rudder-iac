package provider_test

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
)

func TestExplainBlockingConnections(t *testing.T) {
	t.Parallel()

	// One case per service that raises this refusal, spelled as that service
	// spells it. The three wordings are unrelated strings, so a table built on
	// one of them leaves the other two paths silently unannotated.
	t.Run("names both env vars and keeps the backend error", func(t *testing.T) {
		t.Parallel()

		for name, msg := range map[string]string{
			"destination.service.ts": "The destination has active connections, please delete those first",
			"source.service.ts":      "The source has active connections, please delete those first",
			"retl/service.ts":        "The source is connected to some destinations.",
		} {
			t.Run(name, func(t *testing.T) {
				t.Parallel()

				backend := &client.APIError{HTTPStatusCode: http.StatusBadRequest, Message: msg}

				err := provider.ExplainBlockingConnections(fmt.Errorf("deleting RETL source: %w", backend))

				var apiErr *client.APIError
				require.ErrorAs(t, err, &apiErr)
				assert.Equal(t, backend, apiErr)
				assert.Contains(t, err.Error(), msg, "the backend's own reason must survive")
				assert.Contains(t, err.Error(), "RUDDERSTACK_CLI_EXPERIMENTAL=true RUDDERSTACK_X_RETL_CONNECTION_SUPPORT=true")
			})
		}
	})

	// An unrelated failure must not pick up advice about a flag that cannot fix it.
	t.Run("passes other failures through untouched", func(t *testing.T) {
		t.Parallel()

		for _, err := range []error{
			&client.APIError{HTTPStatusCode: http.StatusBadRequest, Message: "destination is referenced by a running job"},
			&client.APIError{HTTPStatusCode: http.StatusInternalServerError, Message: "active connections"},
			errors.New("active connections"),
		} {
			assert.Same(t, err, provider.ExplainBlockingConnections(err))
		}
	})
}
