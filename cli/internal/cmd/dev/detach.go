package dev

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/cmderrors"
	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten"
)

const detachReadyTimeout = 10 * time.Second

// spawnFunc builds the child process; extra are the arguments the parent
// adds for the child.
type spawnFunc func(extra []string) *exec.Cmd

// selfSpawner re-runs this binary with the parent's arguments, so the
// child inherits -c and the environment.
func selfSpawner() spawnFunc {
	return func(extra []string) *exec.Cmd {
		exe, err := os.Executable()
		if err != nil {
			exe = os.Args[0]
		}
		return exec.Command(exe, append(withoutDetach(os.Args[1:]), extra...)...)
	}
}

func childExtra(idleExitSet bool) []string {
	extra := []string{"--detached-child", "--quiet"}
	if !idleExitSet {
		extra = append(extra, "--idle-exit", "30m")
	}
	return extra
}

func childArgs(parent []string, idleExitSet bool) []string {
	return append(withoutDetach(parent), childExtra(idleExitSet)...)
}

func withoutDetach(args []string) []string {
	out := make([]string, 0, len(args))
	for _, a := range args {
		if a == "--detach" || strings.HasPrefix(a, "--detach=") {
			continue
		}
		out = append(out, a)
	}
	return out
}

// runDetach starts the child, waits for its ready line, prints it and
// returns. The child's stdio are pipes to this process only, so no agent
// shell pipe stays open after the parent exits.
func runDetach(stdout, stderr io.Writer, opts listenOptions, spawn spawnFunc) error {
	child, err := startChild(spawn(childExtra(opts.idleExitSet)))
	if err != nil {
		return startupError(stderr, &cliError{Code: "listen_failed", Message: err.Error(), Next: "rudder-cli dev listen"})
	}
	defer child.close()

	select {
	case line := <-child.line:
		if line != "" {
			return printDetached(stdout, stderr, opts, line)
		}
		<-child.exited
	case <-child.exited:
	case <-time.After(detachReadyTimeout):
		_ = child.cmd.Process.Kill()
		return startupError(stderr, &cliError{Code: "detach_timeout",
			Message: "the background server printed no ready line within 10s", Next: "rudder-cli dev listen"})
	}
	return child.failure(stderr)
}

// detachedChild is a started child with its first stdout line, its stderr
// and its exit as channels.
type detachedChild struct {
	cmd    *exec.Cmd
	outR   *os.File
	errR   *os.File
	line   chan string
	stderr *lockedBuffer
	copied chan struct{}
	exited chan struct{}
}

func startChild(cmd *exec.Cmd) (*detachedChild, error) {
	outR, outW, err := os.Pipe()
	if err != nil {
		return nil, fmt.Errorf("creating stdout pipe: %w", err)
	}
	errR, errW, err := os.Pipe()
	if err != nil {
		outR.Close()
		outW.Close()
		return nil, fmt.Errorf("creating stderr pipe: %w", err)
	}
	cmd.Stdin, cmd.Stdout, cmd.Stderr = nil, outW, errW
	setDetachAttrs(cmd)
	startErr := cmd.Start()
	outW.Close()
	errW.Close()
	c := &detachedChild{cmd: cmd, outR: outR, errR: errR, line: make(chan string, 1), stderr: &lockedBuffer{},
		copied: make(chan struct{}), exited: make(chan struct{})}
	if startErr != nil {
		c.close()
		return nil, fmt.Errorf("starting background server: %w", startErr)
	}
	go func() {
		line, _ := bufio.NewReader(outR).ReadString('\n')
		c.line <- line
	}()
	go func() {
		_, _ = io.Copy(c.stderr, errR)
		close(c.copied)
	}()
	go func() {
		_ = cmd.Wait()
		close(c.exited)
	}()
	return c, nil
}

func (c *detachedChild) close() {
	c.outR.Close()
	c.errR.Close()
}

// failure repeats the child's stderr, whose last line is its JSON error.
func (c *detachedChild) failure(stderr io.Writer) error {
	select {
	case <-c.copied:
	case <-time.After(time.Second):
	}
	if msg := strings.TrimSpace(c.stderr.String()); msg != "" {
		fmt.Fprintln(stderr, msg)
		return &cmderrors.SilentError{Err: fmt.Errorf("background server failed to start")}
	}
	return startupError(stderr, &cliError{Code: "listen_failed",
		Message: "the background server exited before it was ready", Next: "rudder-cli dev listen"})
}

func printDetached(stdout, stderr io.Writer, opts listenOptions, line string) error {
	fmt.Fprint(stdout, line)
	var ready devlisten.Ready
	if err := json.Unmarshal([]byte(line), &ready); err != nil {
		return fmt.Errorf("decoding ready line: %w", err)
	}
	if !isLoopback(ready.Bind) {
		fmt.Fprintln(stderr, bindWarning(ready.Bind, ready.Port))
	}
	if opts.progress {
		idle := detachIdleExit
		if opts.idleExitSet {
			idle = opts.idleExit
		}
		fmt.Fprintf(stderr, "dev listen: running in background at %s; idle exit %s.\n", ready.URL, idle)
		fmt.Fprintf(stderr, "Stop: rudder-cli dev stop --url %s\n", ready.URL)
	}
	return nil
}

type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}
