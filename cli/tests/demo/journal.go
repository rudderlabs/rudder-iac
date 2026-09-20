// Package demo records what the e2e suite actually ran, so a demo can be
// generated from a real test run rather than written by hand.
//
// Recording is off unless RUDDER_DEMO_JOURNAL names an output file, so an
// ordinary `make test-e2e` behaves exactly as it did before this package
// existed.
package demo

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	// EnvJournal names the file records are appended to. Unset disables recording.
	EnvJournal = "RUDDER_DEMO_JOURNAL"

	KindExec = "exec"
	KindSay  = "say"

	// Redacted replaces any value that may carry a secret.
	Redacted = "<redacted>"

	// MaxOutputBytes caps recorded command output. A demo needs enough output to
	// be legible, not a complete transcript; the cap keeps one runaway command
	// from producing a journal nobody can read or review.
	MaxOutputBytes = 16 << 10

	truncationMarker = "\n… [output truncated]"

	envPrefix = "RUDDERSTACK_"
)

// secretKeyPattern matches env names whose values must never reach disk.
// It is deliberately broad: a false positive costs a demo one visible value,
// a false negative writes a credential into a committed artifact.
var secretKeyPattern = regexp.MustCompile(`(?i)token|secret|password|key`)

// Record is one entry in the journal: a command the suite ran, or a line of
// narration a test supplied.
type Record struct {
	Kind     string            `json:"kind"`
	Start    time.Time         `json:"start"`
	End      time.Time         `json:"end"`
	Dir      string            `json:"dir,omitempty"`
	Argv     []string          `json:"argv,omitempty"`
	Env      map[string]string `json:"env,omitempty"`
	ExitCode int               `json:"exitCode"`
	Output   string            `json:"output,omitempty"`
	Text     string            `json:"text,omitempty"` // narration, for KindSay
}

var (
	mu         sync.Mutex
	file       *os.File
	openedPath string
	literals   []string
)

// reset drops cached writer state so a test can rebind EnvJournal.
func reset() {
	mu.Lock()
	defer mu.Unlock()

	if file != nil {
		_ = file.Close()
	}
	file, openedPath, literals = nil, "", nil
}

// Enabled reports whether records are being written.
func Enabled() bool {
	return os.Getenv(EnvJournal) != ""
}

// RedactLiterals registers values replaced with Redacted in every subsequent
// record. Tests that hold fixture credentials register them here so the value
// is masked before it is written, never after.
func RedactLiterals(values ...string) {
	mu.Lock()
	defer mu.Unlock()

	literals = append(literals, values...)
}

// Append writes one record. It is a no-op when recording is off, and it never
// fails a test: a demo is worth less than the run it observes, so a journal
// that cannot be written is dropped silently rather than breaking the suite.
func Append(r Record) {
	if !Enabled() {
		return
	}

	mu.Lock()
	defer mu.Unlock()

	w := writer()
	if w == nil {
		return
	}

	r.Env = redactEnv(r.Env)
	r.Output = capOutput(redactLiteralsIn(r.Output))
	r.Text = redactLiteralsIn(r.Text)

	line, err := json.Marshal(r)
	if err != nil {
		return
	}

	_, _ = w.Write(append(line, '\n'))
}

// writer opens the journal for the path RUDDER_DEMO_JOURNAL currently names,
// reopening whenever that path differs from the one already open. Callers
// hold mu.
//
// A one-shot cache would keep writing to whatever file was open first: this
// package's own tests rebind the env var across many subtests in the same
// process, and cli/tests (a different package, so it cannot reach reset) does
// the same across e2e tests. Comparing against the current env value is what
// lets a later test's records land in its own file instead of the first
// test's.
func writer() *os.File {
	path := os.Getenv(EnvJournal)
	if path == openedPath && file != nil {
		return file
	}

	if file != nil {
		_ = file.Close()
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		file, openedPath = nil, ""
		return nil
	}

	file, openedPath = f, path

	return file
}

// CollectEnv returns the RUDDERSTACK_* environment as it stands right now.
// Tests set their flags with t.Setenv, so reading at call time is what lets a
// generated script reproduce the flags a step actually ran under.
//
// Exported because the journal hook that calls it lives in package tests.
func CollectEnv() map[string]string {
	out := map[string]string{}

	for _, kv := range os.Environ() {
		k, v, found := strings.Cut(kv, "=")
		if !found || !strings.HasPrefix(k, envPrefix) {
			continue
		}
		out[k] = v
	}

	return out
}

func redactEnv(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}

	out := make(map[string]string, len(in))
	for k, v := range in {
		if secretKeyPattern.MatchString(k) {
			out[k] = Redacted
			continue
		}
		out[k] = redactLiteralsIn(v)
	}

	return out
}

// redactLiteralsIn masks registered fixture credentials. Callers hold mu.
func redactLiteralsIn(s string) string {
	for _, lit := range literals {
		if lit == "" {
			continue
		}
		s = strings.ReplaceAll(s, lit, Redacted)
	}

	return s
}

func capOutput(s string) string {
	if len(s) <= MaxOutputBytes {
		return s
	}

	return s[:MaxOutputBytes] + truncationMarker
}
