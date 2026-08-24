package tests

import "testing"

// enableLiveWorkspaceCleanupFlags enables every gated resource family that can
// leave managed resources behind in the shared live E2E workspace. Tests that
// start with a workspace-wide destroy need these flags before invoking the CLI,
// otherwise cleanup can see endpoints but miss their active connections.
func enableLiveWorkspaceCleanupFlags(t *testing.T) {
	t.Helper()

	t.Setenv("RUDDERSTACK_CLI_EXPERIMENTAL", "true")
	t.Setenv("RUDDERSTACK_X_DESTINATION_SUPPORT", "true")
	t.Setenv("RUDDERSTACK_X_UNVERIFIED_DESTINATIONS", "true")
	t.Setenv("RUDDERSTACK_X_CONNECTION_SUPPORT", "true")
}
