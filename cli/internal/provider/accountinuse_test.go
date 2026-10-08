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
	assert.Contains(t, got.Error(), "data graphs the CLI manages, remove them from the project")
	assert.Contains(t, got.Error(), "otherwise delete them in the workspace first")

	var unwrapped *client.APIError
	assert.True(t, errors.As(got, &unwrapped), "the annotation is added, not substituted")
}

// An unrelated failure must not acquire advice that has nothing to do with it.
func TestExplainBlockingAccountUsage_LeavesOtherFailuresAlone(t *testing.T) {
	cases := []struct {
		name string
		err  error
	}{
		{"unrelated conflict", &client.APIError{HTTPStatusCode: http.StatusConflict, Message: "external id already claimed"}},
		{"right words, 400 from a route the CLI does not call", &client.APIError{HTTPStatusCode: http.StatusBadRequest, Message: "can't be removed because it is being used"}},
		{"right words, wrong status", &client.APIError{HTTPStatusCode: http.StatusInternalServerError, Message: "can't be removed because it is being used"}},
		{"not an API error", errors.New("connection refused")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := provider.ExplainBlockingAccountUsage(tc.err)

			assert.Equal(t, tc.err.Error(), got.Error(), "the error must pass through untouched")
			assert.NotContains(t, got.Error(), "RUDDERSTACK_X_RETL")
		})
	}
}
