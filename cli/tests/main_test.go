package tests

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// TestMain builds the rudder-cli binary once, exposes it via PATH and then
// runs the package tests. It honours the cli/logging and cli/testing rules by
// printing only essential information and performing cleanup after execution.
func TestMain(m *testing.M) {
	tmpDir, err := os.MkdirTemp("", "rudder-cli-bin-*")
	if err != nil {
		fmt.Println("failed to create temp dir:", err)
		os.Exit(1)
	}

	defer os.RemoveAll(tmpDir)

	exec, err := NewCmdExecutor("")
	if err != nil {
		fmt.Println("failed to init executor:", err)
		os.Exit(1)
	}

	bin, err := NewCLIBinary(tmpDir, exec)
	if err != nil {
		fmt.Println("failed to init CLIBinary helper:", err)
		os.Exit(1)
	}

	path, err := bin.Setup()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	oldpath := os.Getenv("PATH")
	binDir := filepath.Dir(path)
	os.Setenv("PATH", fmt.Sprintf("%s/%s", binDir, oldpath))

	defer os.Setenv("PATH", oldpath)

	exitCode := m.Run()

	_ = bin.Clean()
	os.Exit(exitCode)
}

func TestE2ESetup(t *testing.T) {
	t.Parallel()

	cfgPath, cleanup, err := NewConfigBuilder("").Build()
	if err != nil {
		t.Fatalf("create config: %v", err)
	}
	defer func() {
		if err := cleanup(); err != nil {
			t.Logf("cleanup config: %v", err)
		}
	}()

	exec, err := NewCmdExecutor("")
	if err != nil {
		t.Fatalf("init executor: %v", err)
	}

	out, err := exec.Execute("rudder-cli", "-c", cfgPath, "-v")
	if err != nil {
		t.Fatalf("rudder-cli -v failed: %v\n%s", err, out)
	}

	if !bytes.Contains(out, []byte("rudder-cli")) {
		t.Errorf("unexpected CLI output: %s", out)
	}
}
