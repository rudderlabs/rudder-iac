package tests

import (
	"fmt"
	"os"
	"testing"
)

var (
	cliBinPath string
)

// TestMain builds the rudder-cli binary once, exposes it via PATH and then
// runs the package tests. It honours the cli/logging and cli/testing rules by
// printing only essential information and performing cleanup after execution.
func TestMain(m *testing.M) {
	exec, err := NewCmdExecutor("")
	if err != nil {
		fmt.Println("failed to init executor:", err)
		os.Exit(1)
	}

	bin, err := NewCLIBinary(exec)
	if err != nil {
		fmt.Println("failed to init cli binary:", err)
		os.Exit(1)
	}

	path, err := bin.Setup()
	if err != nil {
		fmt.Println("failed to setup cli binary:", err)
		os.Exit(1)
	}

	cliBinPath = path // Set global cli binary path
	exitCode := m.Run()

	if err := bin.Clean(); err != nil {
		fmt.Println("failed to clean cli binary: ", err)
		os.Exit(1)
	}

	os.Exit(exitCode)
}

// allowManagedResidue lets remote state loading see every managed kind a
// previous run could have left behind in the shared workspace, so the destroy
// each live test opens with can remove it.
//
// Unverified destinations: destinations are GA, so remote destination state
// loads on every apply, destroy and dry-run, even in tests that touch no
// destinations. TestDestinationsApply creates managed
// attentive_tag/http/rs/salesforce destinations; if its cleanup destroy fails,
// only s3 is registered on the next run and the residue fails the whole load.
//
// rETL kinds: retl-source-table and retl-connections load only behind their
// own flags, and a destroy cannot see a kind it has not loaded. PR runs are
// cancelled in-progress by design, so a run killed mid TestRETLConnectionsApply
// would otherwise leave rows the next run's opening destroy cannot reach —
// the source then cannot be deleted either while a connection still uses it.
//
// Keep this with the tests rather than as CI repository variables, so the
// requirement travels with the code. Callers still set
// RUDDERSTACK_CLI_EXPERIMENTAL themselves for their own flags; it is set here
// too because the rETL flags take effect only under it.
func allowManagedResidue(t *testing.T) {
	t.Helper()

	t.Setenv("RUDDERSTACK_CLI_EXPERIMENTAL", "true")
	t.Setenv("RUDDERSTACK_X_UNVERIFIED_DESTINATIONS", "true")
	t.Setenv("RUDDERSTACK_X_RETL_TABLE_SUPPORT", "true")
	t.Setenv("RUDDERSTACK_X_RETL_CONNECTION_SUPPORT", "true")
}
