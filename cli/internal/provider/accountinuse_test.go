package provider_test

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The dependent is usually invisible to the run that tripped this: a source
// made in the webapp is outside the project, so the backend's reason names
// nothing the user can see in the plan.
func TestExplainBlockingAccountUsage_ExplainsTheRefusal(t *testing.T) {
	apiErr := &client.APIError{
		HTTPStatusCode: http.StatusConflict,
		Message:        "This account can't be removed because it is being used by sources: src-1.",
	}

	got := provider.ExplainBlockingAccountUsage(fmt.Errorf("deleting account: %w", apiErr))

	require.Error(t, got)
	assert.Contains(t, got.Error(), "src-1", "the backend's own reason must survive")
	assert.NotContains(t, got.Error(), "RUDDERSTACK_", "the message must not send the reader to experimental flags (DEX-959)")
	assert.Contains(t, got.Error(), "Point them at another account or remove them from the project")
	assert.NotContains(t, got.Error(), "delete them in the workspace", "deleting a CLI-managed source in the workspace gets it recreated on the next apply")

	var unwrapped *client.APIError
	assert.True(t, errors.As(got, &unwrapped), "the annotation is added, not substituted")
}

// Which API errors count as an in-use refusal is tested in api/client; this only
// checks that anything else is returned as is.
func TestExplainBlockingAccountUsage_LeavesOtherFailuresAlone(t *testing.T) {
	for _, err := range []error{
		&client.APIError{HTTPStatusCode: http.StatusConflict, Message: "external id already claimed"},
		errors.New("connection refused"),
	} {
		assert.Same(t, err, provider.ExplainBlockingAccountUsage(err))
	}
}
