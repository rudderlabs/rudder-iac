package tests

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

// CLIBinary builds a rudder-cli binary once and exposes its location for test runs.
// The build is executed lazily when Setup is invoked for the first time.
// Subsequent Setup calls return the already built binary path.
//
// A caller should call Clean to remove the generated binary when it is no longer needed.
//
//	bin := tests.NewCLIBinary(t.TempDir())
//	path, err := bin.Setup() // builds on first call
//	defer bin.Clean()
//
// This helper honours the cli/logging guidance – prints only essential info to stdout.
type CLIBinary struct {
	dir      string    // directory where binary is built
	once     sync.Once // ensures single build
	binPath  string    // absolute path of built binary
	buildErr error     // build error captured inside once.Do
	exec     Executor  // executor used to run commands
}

// NewCLIBinary initialises the helper. The dir must exist and be writable.
func NewCLIBinary(dir string, exec Executor) (*CLIBinary, error) {
	if dir == "" {
		dir = os.TempDir()
	}
	if stat, err := os.Stat(dir); err != nil || !stat.IsDir() {
		return nil, fmt.Errorf("invalid dir %q: %w", dir, err)
	}

	filename := "rudder-cli"

	if runtime.GOOS == "windows" {
		filename += ".exe"
	}

	filepath := filepath.Join(dir, filename)
	return &CLIBinary{exec: exec, binPath: filepath, dir: dir}, nil
}

// Setup builds the rudder-cli binary once and returns its absolute path.
// If the build has already been executed, the cached path (or error) is returned.
func (c *CLIBinary) Setup() (string, error) {
	c.once.Do(func() {
		buildPath := filepath.Join("..", "cmd", "rudder-cli")

		out, err := c.exec.Execute("go", "build", "-o", c.binPath, buildPath)
		if err != nil {
			c.buildErr = fmt.Errorf("building rudder-cli: %w\n%s", err, out)
			return
		}

	})

	return c.binPath, c.buildErr
}

// Clean removes the built binary if it exists.
// It is safe to call multiple times.
func (c *CLIBinary) Clean() error {
	if c == nil || c.binPath == "" {
		return nil
	}
	if err := os.Remove(c.binPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	c.binPath = ""
	return nil
}
