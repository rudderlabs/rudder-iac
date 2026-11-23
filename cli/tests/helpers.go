package tests

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/rudderlabs/rudder-iac/cli/tests/demo"
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

	start := time.Now().UTC()
	output, err := command.CombinedOutput()

	journal(start, command, output, err)

	return output, err
}

// journal records the invocation for demo generation. Every CLI call in this
// package funnels through Execute, which is what lets a whole suite run be
// captured without touching a single test.
//
// The Enabled() guard below is a cost fast path, not a correctness one: it
// skips os.Getwd, errors.As and a full os.Environ() scan on every Execute.
// demo.Append carries the identical guard, so correctness is backstopped
// there even if this one is ever removed by accident. If Append's guard is
// ever removed, this one becomes load-bearing and needs its own test.
func journal(start time.Time, command *exec.Cmd, output []byte, runErr error) {
	if !demo.Enabled() {
		return
	}

	dir := command.Dir
	if dir == "" {
		dir, _ = os.Getwd()
	}

	exitCode := 0
	var exitErr *exec.ExitError
	switch {
	case errors.As(runErr, &exitErr):
		exitCode = exitErr.ExitCode()
	case runErr != nil:
		// The command never produced an exit code — it failed to start, or the
		// context killed it before exec. -1 is what ExitCode() itself reports
		// when no code is available, so the journal keeps one vocabulary.
		exitCode = -1
	}

	demo.Append(demo.Record{
		Kind:     demo.KindExec,
		Start:    start,
		End:      time.Now().UTC(),
		Dir:      dir,
		Argv:     command.Args,
		Env:      demo.CollectEnv(),
		ExitCode: exitCode,
		Output:   string(output),
	})
}
