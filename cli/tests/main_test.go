package tests

import (
	"bytes"
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
		fmt.Println("failed to init CLIBinary helper:", err)
		os.Exit(1)
	}

	path, err := bin.Setup()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	defer func() {
		_ = bin.Clean()
	}()

	cliBinPath = path // Set global cli binary path
	exitCode := m.Run()

	os.Exit(exitCode)
}

func TestE2ESetup(t *testing.T) {
	t.Parallel()

	exec, err := NewCmdExecutor("")
	if err != nil {
		t.Fatalf("init executor: %v", err)
	}

	out, err := exec.Execute(cliBinPath, "-v")
	if err != nil {
		t.Fatalf("rudder-cli -v failed: %v\n%s", err, out)
	}

	if !bytes.Contains(out, []byte("rudder-cli")) {
		t.Errorf("unexpected CLI output: %s", out)
	}
}
