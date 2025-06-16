package tests

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

var (
	ErrBinaryAlreadyCleaned = errors.New("binary already cleaned")
)

// CLIBinary builds a rudder-cli binary once and exposes its location for test runs.
// The build is executed lazily when Setup is invoked for the first time.
// Subsequent Setup calls return the already built binary path.
//
// A caller should call Clean to remove the generated binary when it is no longer needed.
//
//	bin := tests.NewCLIBinary(executor)
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
func NewCLIBinary(exec Executor) (*CLIBinary, error) {
	dir, err := os.MkdirTemp("", "rudder-cli-bin-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}

	filename := "rudder-cli"

	if runtime.GOOS == "windows" {
		filename += ".exe"
	}

	return &CLIBinary{
		exec:    exec,
		binPath: filepath.Join(dir, filename),
		dir:     dir,
	}, nil
}

// Setup builds the rudder-cli binary once and returns its absolute path.
// If the build has already been executed, the cached path (or error) is returned.
func (c *CLIBinary) Setup() (string, error) {
	c.once.Do(func() {
		buildPath := filepath.Join("..", "cmd", "rudder-cli")

		out, err := c.exec.Execute("go", "build", "-o", c.binPath, buildPath)
		if err != nil {
			c.buildErr = fmt.Errorf("building rudder-cli: %w\n%s\n", err, out)
			return
		}

	})

	return c.binPath, c.buildErr
}

// Clean removes the built binary if it exists.
// It is safe to call multiple times.
func (c *CLIBinary) Clean() error {
	// Best effort to remove the directory
	// which contains the binary
	if err := os.RemoveAll(c.dir); err != nil {
		return err
	}

	c.binPath = ""
	c.buildErr = ErrBinaryAlreadyCleaned

	return nil
}
