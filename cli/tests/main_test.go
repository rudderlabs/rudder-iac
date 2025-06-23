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
