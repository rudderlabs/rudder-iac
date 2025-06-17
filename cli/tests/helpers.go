package tests

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"
)

// Executor allows callers to run external commands and capture their combined output.
// Implementations should be safe for concurrent use by multiple goroutines.
// All helpers in tests package use Executor to abstract command execution.
//
// The returned []byte contains stdout and stderr combined, identical to exec.Cmd.CombinedOutput().
// The error will be non-nil if the command exits with non-zero status or fails to start.
// Callers may use errors.As(err, *exec.ExitError) to inspect exit code if required.
//
// Example:
//
//	exec := tests.NewCmdExecutor("")
//	out, err := exec.Execute("echo", "hello")
//	fmt.Printf("%s", out) // prints "hello\n"
type Executor interface {
	Execute(cmd string, args ...string) ([]byte, error)
}

// CmdExecutor is a concrete implementation of Executor backed by os/exec.
//
// Fields:
//
//	WorkDir: optional working directory for executed commands.
//	Timeout: optional timeout; zero means no timeout.
//
// CmdExecutor logs high-level errors to console (fmt.Println) in accordance with cli/logging rule,
// but suppresses verbose output unless debugging is enabled elsewhere.
type CmdExecutor struct {
	WorkDir string
	Timeout time.Duration
}

// NewCmdExecutor returns a new CmdExecutor.
// If workDir is non-empty, it must exist; otherwise an error is returned.
func NewCmdExecutor(workDir string) (*CmdExecutor, error) {
	if workDir != "" {
		if stat, err := os.Stat(workDir); err != nil || !stat.IsDir() {
			return nil, fmt.Errorf("invalid workDir %q: %w", workDir, err)
		}
	}
	return &CmdExecutor{WorkDir: workDir, Timeout: 2 * time.Minute}, nil
}

// Execute runs the given command with arguments, capturing combined stdout/stderr.
// If Timeout is set and command exceeds it, context deadline exceeded error is returned.
func (c *CmdExecutor) Execute(cmd string, args ...string) ([]byte, error) {
	ctx := context.Background()

	var cancel context.CancelFunc
	if c.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, c.Timeout)
	} else {
		ctx, cancel = context.WithCancel(ctx)
	}
	defer cancel()

	command := exec.CommandContext(ctx, cmd, args...)
	if c.WorkDir != "" {
		command.Dir = c.WorkDir
	}

	output, err := command.CombinedOutput()
	return output, err
}
