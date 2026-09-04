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

// allowUnverifiedDestinationResidue lets remote state loading decode unverified
// managed destinations that a previous TestDestinationsApply run left behind.
//
// Destinations are GA, so the provider loads remote destination state on every
// apply, destroy and dry-run — including in tests that touch no destinations.
// These live tests share one workspace, and TestDestinationsApply creates
// managed attentive_tag/http/rs/salesforce destinations there; if its cleanup
// destroy fails, only s3 is registered on the next run and the residue fails
// the whole load. Keep this with the tests that need it rather than as a CI
// repository variable, so the requirement travels with the code.
func allowUnverifiedDestinationResidue(t *testing.T) {
	t.Helper()

	t.Setenv("RUDDERSTACK_CLI_EXPERIMENTAL", "true")
	t.Setenv("RUDDERSTACK_X_UNVERIFIED_DESTINATIONS", "true")
}
