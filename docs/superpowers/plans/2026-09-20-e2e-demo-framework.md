# E2E Demo Framework Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Turn a real `go test` run of the e2e suite into a narrated, provenance-stamped, reproducible demo-magic script, without rewriting the tests.

**Architecture:** `CmdExecutor.Execute` is the single choke point for every CLI invocation in package `tests`; a journal hook there records the whole run to JSONL when `RUDDER_DEMO_JOURNAL` is set, and costs nothing when it is not. A generator joins that journal against `go test -json` events on time — exact, because package `tests` is serial — and emits one demo directory per top-level test. A replay of the generated script produces a second journal, and diffing the two is the drift check that keeps generated demos honest.

**Tech Stack:** Go 1.25.0, stdlib only (`encoding/json`, `os/exec`, `testing`); `testify` for assertions per repo convention; `demo-magic.sh` and `asciinema` as external binaries invoked by Make targets, not Go dependencies.

**Spec:** `docs/superpowers/specs/2026-09-20-e2e-demo-framework-design.md`

## Global Constraints

- Module path is `github.com/rudderlabs/rudder-iac`; Go directive is `go 1.25.0`.
- Assertions use `testify` (`assert`/`require`) exclusively. Prefer whole-struct comparison over field-by-field (CLAUDE.md).
- Errors wrap with verb/action context: `fmt.Errorf("joining journal to events: %w", err)`. Sentinel errors take an `Err` prefix.
- Initialisms are fully capitalised in identifiers: `ID`, `URL`, `API` — never `Id`, `Url`.
- Guard clauses over `else`; a `var (...)` block for 2+ related declarations.
- Comments explain *why*, not *what*. Do not narrate obvious code.
- `make lint` must pass before every commit.
- **No behaviour change to `make test-e2e` when `RUDDER_DEMO_JOURNAL` is unset.** This is the load-bearing safety property of the whole plan: the journal is opt-in, and an ordinary e2e run must be byte-identical to today's.
- Generated artifacts live under `demos/` and `casts/` at the repository root and are committed.
- Env var names, fixed: `RUDDER_DEMO_JOURNAL` (journal output path), `RUDDER_DEMO_AGENT` (force non-interactive annotation mode).

## Deviation from the spec (accept or reject before starting)

The spec's Security section asks for two things: redaction of secrets at journal-write time, **and** a final scan of `demo.sh`, `manifest.json` and the cast against `destinationRawSecrets`.

This plan implements the first and **drops the second**. Reasoning: `destinationRawSecrets` lives in `command_destinations_apply_test.go` as a `_test.go` var, so it is not importable by a generator binary, and the property it guards — *these literals never surface in CLI output* — is already asserted by `TestDestinationsApply` itself. A cast is nothing but CLI output, so scanning it re-checks a property the suite already proves. Instead, Task 1 exposes `demo.RedactLiterals(...)` and Task 2 calls it from that test, so registered literals are replaced **before anything reaches disk** — which is the stronger form of what the spec's own principle asks for.

If you want the belt-and-braces cast scan anyway, say so and it becomes Task 10.

---

## File Structure

| File | Responsibility |
|---|---|
| `cli/tests/demo/journal.go` | `Record` type, the JSONL writer, the env gate, redaction |
| `cli/tests/demo/journal_test.go` | Unit tests for the above |
| `cli/tests/demo/say.go` | `demo.Say` — the only test-facing narration API |
| `cli/tests/demo/say_test.go` | Unit tests for `Say` |
| `cli/tests/helpers.go` | Modified: journal hook inside `CmdExecutor.Execute` |
| `cli/tests/demo/cmd/demogen/events.go` | `go test -json` parsing, the active-test stack, parallel detection |
| `cli/tests/demo/cmd/demogen/join.go` | Attribute journal records to subtests by time |
| `cli/tests/demo/cmd/demogen/derive.go` | Subtest name → prose; verification classification |
| `cli/tests/demo/cmd/demogen/rewrite.go` | Absolute path → portable path; fixture copying |
| `cli/tests/demo/cmd/demogen/emit.go` | `demo.sh`, `manifest.json`, `README.md`, `ANNOTATE.md`, `GAPS.md` |
| `cli/tests/demo/cmd/demogen/main.go` | Flag parsing, wiring |
| `Makefile` | `demo-record`, `demo-generate`, `demo-cast`, `demo-check`; widen `make test` |
| `demos/profiles/mini.env`, `demos/profiles/cloud.env` | Backend selection — a URL and a token *reference* |
| `scripts/demo-cast.sh` | PTY recording + provenance stamping |
| `scripts/demo-check.sh` | Replay + journal diff |

---

## Task 1: The journal package

**Files:**
- Create: `cli/tests/demo/journal.go`
- Create: `cli/tests/demo/journal_test.go`
- Create: `cli/tests/demo/say.go`
- Create: `cli/tests/demo/say_test.go`
- Modify: `Makefile:47-49` (widen `make test` so this package's unit tests actually run)

**Interfaces:**
- Consumes: nothing.
- Produces:
  - `demo.Enabled() bool`
  - `demo.Append(r demo.Record)`
  - `demo.Say(t *testing.T, text string)`
  - `demo.RedactLiterals(values ...string)`
  - `demo.Record{Kind, Start, End, Dir, Argv, Env, ExitCode, Output string/…}` — full definition in Step 3.
  - `demo.KindExec = "exec"`, `demo.KindSay = "say"`

- [ ] **Step 1: Write the failing tests for the writer**

Create `cli/tests/demo/journal_test.go`:

```go
package demo

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// readRecords decodes every JSONL line the writer produced.
func readRecords(t *testing.T, path string) []Record {
	t.Helper()

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var (
		dec  = json.NewDecoder(strings.NewReader(string(data)))
		recs []Record
	)
	for dec.More() {
		var r Record
		require.NoError(t, dec.Decode(&r))
		recs = append(recs, r)
	}
	return recs
}

func TestAppendWritesRecordWhenEnabled(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal.jsonl")
	t.Setenv(EnvJournal, path)
	reset()

	start := time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC)
	Append(Record{
		Kind:     KindExec,
		Start:    start,
		End:      start.Add(2 * time.Second),
		Dir:      "/work",
		Argv:     []string{"rudder-cli", "apply", "-l", "project"},
		Env:      map[string]string{"RUDDERSTACK_X_RETL_TABLE_SUPPORT": "true"},
		ExitCode: 0,
		Output:   "applied",
	})

	assert.Equal(t, []Record{{
		Kind:     KindExec,
		Start:    start,
		End:      start.Add(2 * time.Second),
		Dir:      "/work",
		Argv:     []string{"rudder-cli", "apply", "-l", "project"},
		Env:      map[string]string{"RUDDERSTACK_X_RETL_TABLE_SUPPORT": "true"},
		ExitCode: 0,
		Output:   "applied",
	}}, readRecords(t, path))
}

func TestAppendIsNoOpWhenDisabled(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(EnvJournal, "")
	reset()

	assert.False(t, Enabled())
	Append(Record{Kind: KindExec, Argv: []string{"rudder-cli", "apply"}})

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	assert.Empty(t, entries, "nothing may be written when journaling is off")
}

func TestAppendRedactsSecretEnvByPattern(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal.jsonl")
	t.Setenv(EnvJournal, path)
	reset()

	Append(Record{
		Kind: KindExec,
		Argv: []string{"rudder-cli", "apply"},
		Env: map[string]string{
			"RUDDERSTACK_ACCESS_TOKEN":            "pat_live_abc123",
			"RUDDERSTACK_CLI_TELEMETRY_WRITE_KEY": "wk_xyz",
			"RUDDERSTACK_X_RETL_TABLE_SUPPORT":    "true",
			"RUDDERSTACK_API_URL":                 "http://localhost:15580",
		},
	})

	recs := readRecords(t, path)
	require.Len(t, recs, 1)
	assert.Equal(t, map[string]string{
		"RUDDERSTACK_ACCESS_TOKEN":            Redacted,
		"RUDDERSTACK_CLI_TELEMETRY_WRITE_KEY": Redacted,
		"RUDDERSTACK_X_RETL_TABLE_SUPPORT":    "true",
		"RUDDERSTACK_API_URL":                 "http://localhost:15580",
	}, recs[0].Env)
}

func TestAppendRedactsRegisteredLiteralsFromOutput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal.jsonl")
	t.Setenv(EnvJournal, path)
	reset()

	// Stand-ins for the fixture credentials TestDestinationsApply registers.
	// Deliberately low-entropy so secret scanners do not flag this file.
	RedactLiterals("not-a-real-api-key", "not-a-real-access-key-id")
	Append(Record{
		Kind:   KindExec,
		Argv:   []string{"rudder-cli", "apply"},
		Output: "accessKeyId=not-a-real-access-key-id apiKey=not-a-real-api-key ok",
	})

	recs := readRecords(t, path)
	require.Len(t, recs, 1)
	assert.Equal(t, "accessKeyId=<redacted> apiKey=<redacted> ok", recs[0].Output)
}

func TestAppendCapsOutput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal.jsonl")
	t.Setenv(EnvJournal, path)
	reset()

	Append(Record{Kind: KindExec, Argv: []string{"x"}, Output: strings.Repeat("a", MaxOutputBytes+500)})

	recs := readRecords(t, path)
	require.Len(t, recs, 1)
	assert.Len(t, recs[0].Output, MaxOutputBytes+len(truncationMarker))
	assert.True(t, strings.HasSuffix(recs[0].Output, truncationMarker))
}

func TestCollectEnvKeepsOnlyRudderstackPrefix(t *testing.T) {
	t.Setenv("RUDDERSTACK_X_RETL_TABLE_SUPPORT", "true")
	t.Setenv("HOME", "/home/someone")
	t.Setenv("PATH", "/usr/bin")

	got := CollectEnv()

	assert.Equal(t, "true", got["RUDDERSTACK_X_RETL_TABLE_SUPPORT"])
	assert.NotContains(t, got, "HOME")
	assert.NotContains(t, got, "PATH")
}
```

- [ ] **Step 2: Run the tests to verify they fail**

```bash
export GVM_ROOT="$HOME/.gvm"
go test ./cli/tests/demo/... -run 'TestAppend|TestCollectEnv' -v
```

Expected: FAIL to build — `undefined: Record`, `undefined: Append`, `undefined: EnvJournal`.

- [ ] **Step 3: Write the journal implementation**

Create `cli/tests/demo/journal.go`:

```go
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
	mu       sync.Mutex
	file     *os.File
	opened   bool
	literals []string
)

// reset drops cached writer state so a test can rebind EnvJournal.
func reset() {
	mu.Lock()
	defer mu.Unlock()

	if file != nil {
		_ = file.Close()
	}
	file, opened, literals = nil, false, nil
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

// writer opens the journal on first use. Callers hold mu.
func writer() *os.File {
	if opened {
		return file
	}
	opened = true

	f, err := os.OpenFile(os.Getenv(EnvJournal), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil
	}
	file = f

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
```

- [ ] **Step 4: Run the writer tests to verify they pass**

```bash
export GVM_ROOT="$HOME/.gvm"
go test ./cli/tests/demo/... -run 'TestAppend|TestCollectEnv' -v
```

Expected: PASS, six tests.

- [ ] **Step 5: Write the failing test for `Say`**

Create `cli/tests/demo/say_test.go`:

```go
package demo

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSayRecordsNarration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal.jsonl")
	t.Setenv(EnvJournal, path)
	reset()

	Say(t, "The typed rule catches bad table specs")

	recs := readRecords(t, path)
	require.Len(t, recs, 1)
	assert.Equal(t, KindSay, recs[0].Kind)
	assert.Equal(t, "The typed rule catches bad table specs", recs[0].Text)
	assert.False(t, recs[0].Start.IsZero(), "narration needs a timestamp to be placed in the stream")
}

func TestSayIsNoOpWhenDisabled(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(EnvJournal, "")
	reset()

	Say(t, "no journal configured")

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	assert.Empty(t, entries)
}
```

- [ ] **Step 6: Run it to verify it fails**

```bash
go test ./cli/tests/demo/... -run TestSay -v
```

Expected: FAIL — `undefined: Say`.

- [ ] **Step 7: Implement `Say`**

Create `cli/tests/demo/say.go`:

```go
package demo

import (
	"testing"
	"time"
)

// Say records a line of narration at this point in the run.
//
// Demo prose is otherwise derived from subtest names, which carry the step but
// never the reason. Call this where the reason is the point — a regression the
// step exists to prove, a behaviour that looks wrong until it is explained.
//
//	demo.Say(t, "Before DEX-917 the connection was created but never read back.")
//
// It costs one comparison when recording is off.
func Say(t *testing.T, text string) {
	t.Helper()

	if !Enabled() {
		return
	}

	now := time.Now().UTC()
	Append(Record{Kind: KindSay, Start: now, End: now, Text: text})
}
```

- [ ] **Step 8: Run the `Say` tests to verify they pass**

```bash
go test ./cli/tests/demo/... -run TestSay -v
```

Expected: PASS, two tests.

- [ ] **Step 9: Widen `make test` so this package's unit tests run**

`make test` currently excludes every package under `cli/tests`:

```makefile
	@go test --race --covermode=atomic --coverprofile=coverage-unit.out $(shell go list ./... | grep -v /cli/tests)
```

That pattern also excludes `cli/tests/helpers` and now `cli/tests/demo`, both of which hold hermetic unit tests that need no backend. Exclude only the e2e package itself by anchoring the pattern:

```makefile
	@go test --race --covermode=atomic --coverprofile=coverage-unit.out $(shell go list ./... | grep -v '/cli/tests$$')
```

The `$$` is a literal `$` for Make, anchoring the grep to end-of-line.

- [ ] **Step 10: Verify `make test` still passes and now includes the new package**

```bash
export GVM_ROOT="$HOME/.gvm"
make test 2>&1 | grep -E 'cli/tests|FAIL|ok ' | head -20
```

Expected: `ok  github.com/rudderlabs/rudder-iac/cli/tests/demo` and `ok  …/cli/tests/helpers` both present, no `FAIL`, and `…/cli/tests` itself absent.

- [ ] **Step 11: Lint and commit**

```bash
export GVM_ROOT="$HOME/.gvm"
make lint
git add cli/tests/demo Makefile
git commit -m "test(demo): journal the e2e suite's CLI invocations

Record type, JSONL writer and demo.Say, all gated on RUDDER_DEMO_JOURNAL so
an ordinary e2e run is unchanged. Secrets are masked before the record
reaches disk: env keys by pattern, fixture literals by registration.

Anchor make test's exclusion to the e2e package so cli/tests/helpers and
cli/tests/demo unit tests run with the rest of the unit suite.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

## Task 2: Hook the journal into the executor

**Files:**
- Modify: `cli/tests/helpers.go:60-80` (`CmdExecutor.Execute`)
- Create: `cli/tests/executor_journal_test.go`
- Modify: `cli/tests/command_destinations_apply_test.go` (one `RedactLiterals` call)

**Interfaces:**
- Consumes: `demo.Enabled`, `demo.Append`, `demo.Record`, `demo.KindExec`, `demo.RedactLiterals`, `demo.EnvJournal`, `demo.Redacted` from Task 1.
- Produces: a journal file written by any real e2e run started with `RUDDER_DEMO_JOURNAL` set. No new Go API.

- [ ] **Step 1: Write the failing test**

Create `cli/tests/executor_journal_test.go`. It exercises the hook with `echo`, so it needs no backend and no built CLI:

```go
package tests

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/tests/demo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func readJournal(t *testing.T, path string) []demo.Record {
	t.Helper()

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var recs []demo.Record
	dec := json.NewDecoder(strings.NewReader(string(data)))
	for dec.More() {
		var r demo.Record
		require.NoError(t, dec.Decode(&r))
		recs = append(recs, r)
	}

	return recs
}

func TestExecuteJournalsCommand(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal.jsonl")
	t.Setenv(demo.EnvJournal, path)
	t.Setenv("RUDDERSTACK_X_RETL_TABLE_SUPPORT", "true")

	executor, err := NewCmdExecutor("")
	require.NoError(t, err)

	out, err := executor.Execute("echo", "hello")
	require.NoError(t, err)
	assert.Equal(t, "hello\n", string(out))

	recs := readJournal(t, path)
	require.Len(t, recs, 1)
	assert.Equal(t, demo.KindExec, recs[0].Kind)
	assert.Equal(t, []string{"echo", "hello"}, recs[0].Argv)
	assert.Equal(t, 0, recs[0].ExitCode)
	assert.Equal(t, "hello\n", recs[0].Output)
	assert.Equal(t, "true", recs[0].Env["RUDDERSTACK_X_RETL_TABLE_SUPPORT"])
	assert.False(t, recs[0].Start.IsZero())
	assert.False(t, recs[0].End.Before(recs[0].Start))
}

func TestExecuteRecordsNonZeroExit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal.jsonl")
	t.Setenv(demo.EnvJournal, path)

	executor, err := NewCmdExecutor("")
	require.NoError(t, err)

	_, err = executor.Execute("sh", "-c", "exit 7")
	require.Error(t, err)

	recs := readJournal(t, path)
	require.Len(t, recs, 1)
	assert.Equal(t, 7, recs[0].ExitCode)
}

func TestExecuteWritesNothingWhenJournalDisabled(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(demo.EnvJournal, "")

	executor, err := NewCmdExecutor("")
	require.NoError(t, err)

	out, err := executor.Execute("echo", "hello")
	require.NoError(t, err)
	assert.Equal(t, "hello\n", string(out), "output must be identical whether or not journaling is on")

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	assert.Empty(t, entries)
}
```

- [ ] **Step 2: Run it to verify it fails**

```bash
export GVM_ROOT="$HOME/.gvm"
go test ./cli/tests -run TestExecute -v
```

Expected: FAIL — the journal file does not exist, so `os.ReadFile` errors in `readJournal`.

Note: `TestMain` in this package builds the CLI binary first. That build must succeed for the test to run, but these tests do not use the binary.

- [ ] **Step 3: Add the hook to `Execute`**

In `cli/tests/helpers.go`, add the import:

```go
	"github.com/rudderlabs/rudder-iac/cli/tests/demo"
```

Replace the tail of `Execute` — currently:

```go
	output, err := command.CombinedOutput()
	return output, err
}
```

with:

```go
	start := time.Now().UTC()
	output, err := command.CombinedOutput()

	journal(start, command, output, err)

	return output, err
}

// journal records the invocation for demo generation. Every CLI call in this
// package funnels through Execute, which is what lets a whole suite run be
// captured without touching a single test.
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
	if errors.As(runErr, &exitErr) {
		exitCode = exitErr.ExitCode()
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
```

Add `"errors"` to the import block (`os`, `os/exec` and `time` are already imported).

- [ ] **Step 4: Run the hook tests to verify they pass**

```bash
export GVM_ROOT="$HOME/.gvm"
go test ./cli/tests -run TestExecute -v
go test ./cli/tests/demo/... -v
```

Expected: PASS for all three `TestExecute*` and all eight demo package tests.

- [ ] **Step 5: Register the destination fixture secrets**

In `cli/tests/command_destinations_apply_test.go`, at the top of `TestDestinationsApply` (line ~182), immediately after the existing `t.Setenv` calls:

```go
	// These literals are fixture credentials that must never surface in CLI
	// output — the suite asserts that below. Registering them here masks them
	// in the demo journal too, before the value can reach disk.
	demo.RedactLiterals(destinationRawSecrets...)
```

Add the import:

```go
	"github.com/rudderlabs/rudder-iac/cli/tests/demo"
```

- [ ] **Step 6: Verify an ordinary e2e run is unchanged**

This is the plan's load-bearing safety property. Compile the package and confirm nothing about the non-journaling path changed:

```bash
export GVM_ROOT="$HOME/.gvm"
go vet ./cli/tests/...
go test ./cli/tests -run TestExecuteWritesNothingWhenJournalDisabled -v
```

Expected: PASS. If a backend is reachable, also run one real suite both ways and confirm identical pass/fail:

```bash
go test ./cli/tests -run TestAccountsApply -v 2>&1 | tail -5
```

- [ ] **Step 7: Lint and commit**

```bash
export GVM_ROOT="$HOME/.gvm"
make lint
git add cli/tests/helpers.go cli/tests/executor_journal_test.go cli/tests/command_destinations_apply_test.go
git commit -m "test(demo): record every CLI invocation from CmdExecutor.Execute

Every e2e CLI call funnels through Execute, so one hook captures a whole
suite run with no per-test changes. Off unless RUDDER_DEMO_JOURNAL is set.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

## Task 3: Parse `go test -json` and attribute records to subtests

**Files:**
- Create: `cli/tests/demo/cmd/demogen/events.go`
- Create: `cli/tests/demo/cmd/demogen/events_test.go`
- Create: `cli/tests/demo/cmd/demogen/join.go`
- Create: `cli/tests/demo/cmd/demogen/join_test.go`

**Interfaces:**
- Consumes: `demo.Record`, `demo.KindExec`, `demo.KindSay` from Task 1.
- Produces:
  - `type Event struct { Time time.Time; Action, Package, Test string }`
  - `func ParseEvents(r io.Reader, pkg string) ([]Event, error)` — filters to `pkg`, returns events in file order.
  - `var ErrParallelSubtest = errors.New("parallel subtest in the demo package")`
  - `type Step struct { Test string; Records []demo.Record }`
  - `func Join(events []Event, records []demo.Record) ([]Step, error)` — one `Step` per subtest that owns at least one record, in run order.

- [ ] **Step 1: Write the failing tests for event parsing**

Create `cli/tests/demo/cmd/demogen/events_test.go`:

```go
package main

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseEventsFiltersToPackage(t *testing.T) {
	in := strings.Join([]string{
		`{"Time":"2026-09-20T10:00:00Z","Action":"run","Package":"github.com/rudderlabs/rudder-iac/cli/tests","Test":"TestProjectApply"}`,
		`{"Time":"2026-09-20T10:00:01Z","Action":"run","Package":"github.com/rudderlabs/rudder-iac/cli/tests/helpers","Test":"TestComparator"}`,
		`{"Time":"2026-09-20T10:00:02Z","Action":"pass","Package":"github.com/rudderlabs/rudder-iac/cli/tests","Test":"TestProjectApply"}`,
	}, "\n")

	got, err := ParseEvents(strings.NewReader(in), e2ePackage)
	require.NoError(t, err)

	assert.Equal(t, []Event{
		{Time: time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC), Action: "run", Package: e2ePackage, Test: "TestProjectApply"},
		{Time: time.Date(2026, 9, 20, 10, 0, 2, 0, time.UTC), Action: "pass", Package: e2ePackage, Test: "TestProjectApply"},
	}, got)
}

func TestParseEventsRejectsParallelSubtest(t *testing.T) {
	in := strings.Join([]string{
		`{"Time":"2026-09-20T10:00:00Z","Action":"run","Package":"github.com/rudderlabs/rudder-iac/cli/tests","Test":"TestA"}`,
		`{"Time":"2026-09-20T10:00:01Z","Action":"pause","Package":"github.com/rudderlabs/rudder-iac/cli/tests","Test":"TestA"}`,
	}, "\n")

	_, err := ParseEvents(strings.NewReader(in), e2ePackage)
	require.ErrorIs(t, err, ErrParallelSubtest)
	assert.Contains(t, err.Error(), "TestA")
}

func TestParseEventsIgnoresParallelismOutsidePackage(t *testing.T) {
	// cli/tests/helpers is parallel throughout and drives no CLI, so its pause
	// events must not disqualify a run.
	in := `{"Time":"2026-09-20T10:00:00Z","Action":"pause","Package":"github.com/rudderlabs/rudder-iac/cli/tests/helpers","Test":"TestFileManager"}`

	got, err := ParseEvents(strings.NewReader(in), e2ePackage)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestParseEventsSkipsPackageLevelEvents(t *testing.T) {
	in := `{"Time":"2026-09-20T10:00:00Z","Action":"output","Package":"github.com/rudderlabs/rudder-iac/cli/tests"}`

	got, err := ParseEvents(strings.NewReader(in), e2ePackage)
	require.NoError(t, err)
	assert.Empty(t, got, "events with no Test name belong to the package, not a step")
}
```

- [ ] **Step 2: Run to verify failure**

```bash
go test ./cli/tests/demo/cmd/demogen/ -run TestParseEvents -v
```

Expected: FAIL to build — `undefined: ParseEvents`, `undefined: Event`, `undefined: ErrParallelSubtest`.

- [ ] **Step 3: Implement event parsing**

Create `cli/tests/demo/cmd/demogen/events.go`:

```go
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"
)

// e2ePackage is the only package that drives the CLI binary, and so the only
// one whose events can own a journal record.
const e2ePackage = "github.com/rudderlabs/rudder-iac/cli/tests"

// ErrParallelSubtest reports a parallel subtest in the package that drives the
// CLI. Journal records are attributed to subtests by time, which is exact only
// while that package runs serially; a parallel subtest would silently scramble
// a demo rather than fail it, so it is refused instead.
var ErrParallelSubtest = errors.New("parallel subtest in the demo package")

// Event is one `go test -json` record, narrowed to the fields attribution needs.
type Event struct {
	Time    time.Time `json:"Time"`
	Action  string    `json:"Action"`
	Package string    `json:"Package"`
	Test    string    `json:"Test"`
}

// ParseEvents reads a `go test -json` stream and returns the test-scoped events
// belonging to pkg, in file order.
//
// Events from other packages are dropped rather than refused: cli/tests/helpers
// is parallel throughout, but holds unit tests that never touch the CLI and so
// contribute no journal records.
func ParseEvents(r io.Reader, pkg string) ([]Event, error) {
	var (
		dec = json.NewDecoder(r)
		out []Event
	)

	for dec.More() {
		var e Event
		if err := dec.Decode(&e); err != nil {
			return nil, fmt.Errorf("decoding go test event: %w", err)
		}

		if e.Package != pkg || e.Test == "" {
			continue
		}

		if e.Action == "pause" || e.Action == "cont" {
			return nil, fmt.Errorf("%w: %s", ErrParallelSubtest, e.Test)
		}

		out = append(out, e)
	}

	return out, nil
}
```

- [ ] **Step 4: Run to verify the parsing tests pass**

```bash
go test ./cli/tests/demo/cmd/demogen/ -run TestParseEvents -v
```

Expected: PASS, four tests.

- [ ] **Step 5: Write the failing tests for the join**

Create `cli/tests/demo/cmd/demogen/join_test.go`:

```go
package main

import (
	"testing"
	"time"

	"github.com/rudderlabs/rudder-iac/cli/tests/demo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func at(sec int) time.Time {
	return time.Date(2026, 9, 20, 10, 0, sec, 0, time.UTC)
}

func TestJoinAttributesRecordsToDeepestActiveSubtest(t *testing.T) {
	events := []Event{
		{Time: at(0), Action: "run", Test: "TestProjectApply"},
		{Time: at(1), Action: "run", Test: "TestProjectApply/rudder_specs"},
		{Time: at(2), Action: "run", Test: "TestProjectApply/rudder_specs/should_create_entities"},
		{Time: at(6), Action: "pass", Test: "TestProjectApply/rudder_specs/should_create_entities"},
		{Time: at(7), Action: "run", Test: "TestProjectApply/rudder_specs/should_update_entities"},
		{Time: at(9), Action: "pass", Test: "TestProjectApply/rudder_specs/should_update_entities"},
	}
	records := []demo.Record{
		{Kind: demo.KindExec, Start: at(3), Argv: []string{"rudder-cli", "apply", "-l", "create"}},
		{Kind: demo.KindExec, Start: at(5), Argv: []string{"rudder-cli", "apply", "-l", "create"}},
		{Kind: demo.KindExec, Start: at(8), Argv: []string{"rudder-cli", "apply", "-l", "update"}},
	}

	got, err := Join(events, records)
	require.NoError(t, err)

	assert.Equal(t, []Step{
		{
			Test: "TestProjectApply/rudder_specs/should_create_entities",
			Records: []demo.Record{
				{Kind: demo.KindExec, Start: at(3), Argv: []string{"rudder-cli", "apply", "-l", "create"}},
				{Kind: demo.KindExec, Start: at(5), Argv: []string{"rudder-cli", "apply", "-l", "create"}},
			},
		},
		{
			Test: "TestProjectApply/rudder_specs/should_update_entities",
			Records: []demo.Record{
				{Kind: demo.KindExec, Start: at(8), Argv: []string{"rudder-cli", "apply", "-l", "update"}},
			},
		},
	}, got)
}

func TestJoinAttributesToParentWhenNoSubtestActive(t *testing.T) {
	events := []Event{
		{Time: at(0), Action: "run", Test: "TestAccountsApply"},
		{Time: at(5), Action: "pass", Test: "TestAccountsApply"},
	}
	records := []demo.Record{
		{Kind: demo.KindExec, Start: at(1), Argv: []string{"rudder-cli", "destroy", "--confirm=false"}},
	}

	got, err := Join(events, records)
	require.NoError(t, err)

	assert.Equal(t, []Step{{
		Test:    "TestAccountsApply",
		Records: []demo.Record{{Kind: demo.KindExec, Start: at(1), Argv: []string{"rudder-cli", "destroy", "--confirm=false"}}},
	}}, got)
}

func TestJoinKeepsNarrationInStreamOrder(t *testing.T) {
	events := []Event{
		{Time: at(0), Action: "run", Test: "TestA"},
		{Time: at(9), Action: "pass", Test: "TestA"},
	}
	records := []demo.Record{
		{Kind: demo.KindSay, Start: at(1), Text: "why this matters"},
		{Kind: demo.KindExec, Start: at(2), Argv: []string{"rudder-cli", "apply"}},
	}

	got, err := Join(events, records)
	require.NoError(t, err)

	require.Len(t, got, 1)
	require.Len(t, got[0].Records, 2)
	assert.Equal(t, demo.KindSay, got[0].Records[0].Kind)
	assert.Equal(t, demo.KindExec, got[0].Records[1].Kind)
}

func TestJoinDropsRecordsOutsideAnyTest(t *testing.T) {
	// TestMain builds the binary before the first `run` event; that go build
	// invocation is real but belongs to no test and must not open a demo.
	events := []Event{
		{Time: at(5), Action: "run", Test: "TestA"},
		{Time: at(9), Action: "pass", Test: "TestA"},
	}
	records := []demo.Record{
		{Kind: demo.KindExec, Start: at(1), Argv: []string{"go", "build", "-o", "/tmp/rudder-cli"}},
		{Kind: demo.KindExec, Start: at(6), Argv: []string{"rudder-cli", "apply"}},
	}

	got, err := Join(events, records)
	require.NoError(t, err)

	require.Len(t, got, 1)
	assert.Equal(t, []string{"rudder-cli", "apply"}, got[0].Records[0].Argv)
}

func TestJoinSkipsTestsThatRanNoCommands(t *testing.T) {
	events := []Event{
		{Time: at(0), Action: "run", Test: "TestA"},
		{Time: at(1), Action: "run", Test: "TestA/empty"},
		{Time: at(2), Action: "pass", Test: "TestA/empty"},
		{Time: at(3), Action: "run", Test: "TestA/busy"},
		{Time: at(5), Action: "pass", Test: "TestA/busy"},
	}
	records := []demo.Record{{Kind: demo.KindExec, Start: at(4), Argv: []string{"rudder-cli", "apply"}}}

	got, err := Join(events, records)
	require.NoError(t, err)

	require.Len(t, got, 1)
	assert.Equal(t, "TestA/busy", got[0].Test)
}
```

- [ ] **Step 6: Run to verify failure**

```bash
go test ./cli/tests/demo/cmd/demogen/ -run TestJoin -v
```

Expected: FAIL to build — `undefined: Join`, `undefined: Step`.

- [ ] **Step 7: Implement the join**

Create `cli/tests/demo/cmd/demogen/join.go`:

```go
package main

import (
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/tests/demo"
)

// maxInt64 stands in for "has not finished yet" in a span's stop time.
const maxInt64 = int64(^uint64(0) >> 1)

// Step is one subtest and the records it ran, in order.
type Step struct {
	Test    string
	Records []demo.Record
}

// span is one test's start and finish, in unix nanoseconds.
type span struct {
	test        string
	start, stop int64
}

// Join attributes each journal record to the subtest that was running when it
// started.
//
// The suite is serial, so "which test was running" is a well-defined question:
// walk the events in time order maintaining the set of started-but-not-finished
// tests, and a record belongs to the deepest one active at its start. Records
// that precede the first test — TestMain's `go build`, notably — belong to no
// step and are dropped.
func Join(events []Event, records []demo.Record) ([]Step, error) {
	var (
		open  = map[string]int{} // test name -> index into spans
		spans []span
	)

	for _, e := range events {
		switch e.Action {
		case "run":
			open[e.Test] = len(spans)
			spans = append(spans, span{test: e.Test, start: e.Time.UnixNano(), stop: maxInt64})
		case "pass", "fail", "skip":
			idx, ok := open[e.Test]
			if !ok {
				continue
			}
			spans[idx].stop = e.Time.UnixNano()
			delete(open, e.Test)
		}
	}

	byTest := map[string][]demo.Record{}
	var order []string

	for _, r := range records {
		owner := deepestActive(spans, r.Start.UnixNano())
		if owner == "" {
			continue
		}

		if _, seen := byTest[owner]; !seen {
			order = append(order, owner)
		}
		byTest[owner] = append(byTest[owner], r)
	}

	// Emit in the order the tests first claimed a record, which for a serial
	// suite is the order a viewer will watch them in.
	out := make([]Step, 0, len(order))
	for _, test := range order {
		out = append(out, Step{Test: test, Records: byTest[test]})
	}

	return out, nil
}

// deepestActive returns the most deeply nested test covering ts, or "" if none
// does. Depth is the number of "/" separators, so a subtest always wins over
// the parent whose span contains it.
func deepestActive(spans []span, ts int64) string {
	var (
		best      string
		bestDepth = -1
	)

	for _, s := range spans {
		if ts < s.start || ts > s.stop {
			continue
		}

		depth := strings.Count(s.test, "/")
		if depth > bestDepth {
			best, bestDepth = s.test, depth
		}
	}

	return best
}
```

No sorting anywhere: the journal is already in run order, and a serial suite
means run order is the order a viewer watches.

- [ ] **Step 8: Run the join tests to verify they pass**

```bash
go test ./cli/tests/demo/cmd/demogen/ -run TestJoin -v
```

Expected: PASS, five tests.

- [ ] **Step 9: Lint and commit**

```bash
export GVM_ROOT="$HOME/.gvm"
make lint
git add cli/tests/demo/cmd/demogen
git commit -m "test(demo): attribute journal records to subtests by time

Parse go test -json, build each test's span, and give every record to the
deepest test active at its start. Refuses a parallel subtest in the CLI
package rather than producing a scrambled demo; ignores parallelism in
cli/tests/helpers, which drives no CLI.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

## Task 4: Derive prose and classify verification

**Files:**
- Create: `cli/tests/demo/cmd/demogen/derive.go`
- Create: `cli/tests/demo/cmd/demogen/derive_test.go`

**Interfaces:**
- Consumes: `Step` from Task 3; `demo.Record`, `demo.KindExec`, `demo.KindSay` from Task 1.
- Produces:
  - `func DeriveProse(testName string) string` — last path segment → a sentence.
  - `func IsThin(testName string) bool` — true when the derived prose carries no information worth showing.
  - `func Annotated(s Step) bool` — true when the step contains at least one `KindSay` record.
  - `func Verification(s Step) string` — `"visible"` when the step's last exec is a read-only command, `"none"` otherwise.

- [ ] **Step 1: Write the failing tests**

Create `cli/tests/demo/cmd/demogen/derive_test.go`. The inputs are the real subtest names in the suite today:

```go
package main

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/tests/demo"
	"github.com/stretchr/testify/assert"
)

func TestDeriveProse(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"strips should", "TestProjectApply/rudder_specs/should_create_entities_in_catalog_from_project", "Create entities in catalog from project"},
		{"strips should mid-phrase", "TestProjectApply/migrated_create_specs_should_produce_the_same_state", "Migrated create specs produce the same state"},
		{"terse subtest", "TestAccountsApply/apply_create", "Apply create"},
		{"top-level test name", "TestConnectionsApply", "Connections apply"},
		{"already a sentence", "TestDestinationsApply/re-apply_churns_only_the_write-only_secret", "Re-apply churns only the write-only secret"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, DeriveProse(tc.in))
		})
	}
}

func TestIsThin(t *testing.T) {
	assert.True(t, IsThin("TestTransformationsTest/success"), "one word says nothing about what happened")
	assert.True(t, IsThin("TestTransformationsTest/failure"))
	assert.False(t, IsThin("TestProjectApply/should_create_entities_in_catalog_from_project"))
	assert.False(t, IsThin("TestAccountsApply/apply_create"))
}

func TestAnnotated(t *testing.T) {
	assert.True(t, Annotated(Step{Records: []demo.Record{
		{Kind: demo.KindSay, Text: "why"},
		{Kind: demo.KindExec, Argv: []string{"rudder-cli", "apply"}},
	}}))
	assert.False(t, Annotated(Step{Records: []demo.Record{
		{Kind: demo.KindExec, Argv: []string{"rudder-cli", "apply"}},
	}}))
}

func TestVerificationIsVisibleForReadOnlyTail(t *testing.T) {
	cases := []struct {
		name string
		argv []string
		want string
	}{
		{"list", []string{"rudder-cli", "workspace", "retl-connections", "list"}, "visible"},
		{"json flag", []string{"rudder-cli", "workspace", "accounts", "list", "--json"}, "visible"},
		{"validate", []string{"rudder-cli", "validate", "-l", "project"}, "visible"},
		{"dry run", []string{"rudder-cli", "apply", "-l", "project", "--dry-run"}, "visible"},
		{"preview", []string{"rudder-cli", "retl-sources", "preview", "vip"}, "visible"},
		{"apply is a write", []string{"rudder-cli", "apply", "-l", "project", "--confirm=false"}, "none"},
		{"destroy is a write", []string{"rudder-cli", "destroy", "--confirm=false"}, "none"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Verification(Step{Records: []demo.Record{
				{Kind: demo.KindExec, Argv: []string{"rudder-cli", "apply"}},
				{Kind: demo.KindExec, Argv: tc.argv},
			}})
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestVerificationIgnoresTrailingNarration(t *testing.T) {
	// A demo.Say after the read-back must not hide the verification.
	got := Verification(Step{Records: []demo.Record{
		{Kind: demo.KindExec, Argv: []string{"rudder-cli", "workspace", "accounts", "list"}},
		{Kind: demo.KindSay, Text: "and there it is"},
	}})
	assert.Equal(t, "visible", got)
}

func TestVerificationOfEmptyStep(t *testing.T) {
	assert.Equal(t, "none", Verification(Step{}))
}
```

- [ ] **Step 2: Run to verify failure**

```bash
go test ./cli/tests/demo/cmd/demogen/ -run 'TestDerive|TestIsThin|TestAnnotated|TestVerification' -v
```

Expected: FAIL to build — `undefined: DeriveProse` and the rest.

- [ ] **Step 3: Implement derivation**

Create `cli/tests/demo/cmd/demogen/derive.go`:

```go
package main

import (
	"strings"
	"unicode"

	"github.com/rudderlabs/rudder-iac/cli/tests/demo"
)

// readOnlyVerbs are CLI commands and flags that only read. A step ending in one
// shows its own proof on screen; a step that does not is asserted invisibly in
// Go, which is what a demo has to say out loud.
//
// ponytail: a flat allowlist, not command-tree introspection. It is advisory —
// a wrong answer mislabels a manifest field, it does not break a demo. Replace
// it with a real read-only annotation on the cobra commands if that ever matters.
var readOnlyVerbs = map[string]bool{
	"list":     true,
	"validate": true,
	"preview":  true,
	"get":      true,
	"view":     true,
	"info":     true,
	"diff":     true,
}

var readOnlyFlags = map[string]bool{
	"--json":    true,
	"--dry-run": true,
}

// DeriveProse turns a subtest name into a sentence.
//
// go test reports names with spaces replaced by underscores, so the raw form is
// "should_create_entities_in_catalog_from_project". The prose it yields carries
// the step but never the reason — that is what demo.Say is for.
func DeriveProse(testName string) string {
	name := testName
	if i := strings.LastIndex(name, "/"); i >= 0 {
		name = name[i+1:]
	}

	name = strings.TrimPrefix(name, "Test")
	words := strings.Split(strings.ReplaceAll(name, "_", " "), " ")

	// "should" adds nothing to a narrated step and reads oddly in the middle of
	// one ("migrated create specs should produce…" -> "…specs produce…").
	kept := make([]string, 0, len(words))
	for _, w := range words {
		if strings.EqualFold(w, "should") || w == "" {
			continue
		}
		kept = append(kept, w)
	}

	if len(kept) == 0 {
		return ""
	}

	// A bare test name is CamelCase; a subtest name is already spaced.
	if len(kept) == 1 && hasInnerUpper(kept[0]) {
		kept = splitCamel(kept[0])
	}

	joined := strings.Join(kept, " ")

	return strings.ToUpper(joined[:1]) + joined[1:]
}

func hasInnerUpper(s string) bool {
	for _, r := range s[1:] {
		if unicode.IsUpper(r) {
			return true
		}
	}

	return false
}

func splitCamel(s string) []string {
	var (
		out  []string
		word strings.Builder
	)

	for i, r := range s {
		if i > 0 && unicode.IsUpper(r) {
			out = append(out, strings.ToLower(word.String()))
			word.Reset()
		}
		word.WriteRune(r)
	}
	out = append(out, strings.ToLower(word.String()))

	return out
}

// IsThin reports whether derived prose is too short to tell a viewer anything.
// "success" and "failure" are real subtest names in this suite and describe
// nothing; a step like that is exactly where demo.Say earns its keep.
func IsThin(testName string) bool {
	return len(strings.Fields(DeriveProse(testName))) < 2
}

// Annotated reports whether a step carries hand-written narration.
func Annotated(s Step) bool {
	for _, r := range s.Records {
		if r.Kind == demo.KindSay {
			return true
		}
	}

	return false
}

// Verification reports whether the step proves its result on screen.
func Verification(s Step) string {
	for i := len(s.Records) - 1; i >= 0; i-- {
		r := s.Records[i]
		if r.Kind != demo.KindExec {
			continue
		}

		if isReadOnly(r.Argv) {
			return "visible"
		}

		return "none"
	}

	return "none"
}

func isReadOnly(argv []string) bool {
	for _, a := range argv {
		if readOnlyFlags[a] || readOnlyVerbs[a] {
			return true
		}
	}

	return false
}
```

- [ ] **Step 4: Run to verify the derivation tests pass**

```bash
go test ./cli/tests/demo/cmd/demogen/ -run 'TestDerive|TestIsThin|TestAnnotated|TestVerification' -v
```

Expected: PASS. If `TestDeriveProse/top-level_test_name` fails, the camel-split path is wrong — `TestConnectionsApply` must yield `Connections apply`, so `Test` is trimmed before splitting and only the first word is capitalised at the end.

- [ ] **Step 5: Lint and commit**

```bash
export GVM_ROOT="$HOME/.gvm"
make lint
git add cli/tests/demo/cmd/demogen/derive.go cli/tests/demo/cmd/demogen/derive_test.go
git commit -m "test(demo): derive step prose and classify visible verification

Subtest names carry the step; demo.Say carries the reason. Steps whose
derived prose says nothing are marked thin, and steps that end in a write
rather than a read are marked as verified invisibly in Go.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

## Task 5: Rewrite paths and copy fixtures

**Files:**
- Create: `cli/tests/demo/cmd/demogen/rewrite.go`
- Create: `cli/tests/demo/cmd/demogen/rewrite_test.go`

**Interfaces:**
- Consumes: `demo.Record` from Task 1.
- Produces:
  - `type Rewriter struct { RepoRoot, BinPath, TempRoot string }`
  - `func (rw Rewriter) Argv(argv []string) (out []string, gaps []string)` — portable argv plus any absolute path it could not rewrite.
  - `func (rw Rewriter) Fixtures(argv []string) []string` — repo-relative fixture directories the step reads.
  - `func CopyTree(src, dst string) error`

- [ ] **Step 1: Write the failing tests**

Create `cli/tests/demo/cmd/demogen/rewrite_test.go`:

```go
package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testRewriter() Rewriter {
	return Rewriter{
		RepoRoot: "/home/dev/rudder-iac",
		BinPath:  "/tmp/rudder-cli-bin-99/rudder-cli",
		TempRoot: "/tmp/TestProjectApply123",
	}
}

func TestArgvRewritesBinaryToCommandName(t *testing.T) {
	got, gaps := testRewriter().Argv([]string{"/tmp/rudder-cli-bin-99/rudder-cli", "apply", "-l", "x"})

	assert.Equal(t, []string{"rudder-cli", "apply", "-l", "x"}, got)
	assert.Empty(t, gaps)
}

func TestArgvRewritesTestdataToProject(t *testing.T) {
	got, gaps := testRewriter().Argv([]string{
		"/tmp/rudder-cli-bin-99/rudder-cli", "apply",
		"-l", "/home/dev/rudder-iac/cli/tests/testdata/project/create",
		"--var-file", "/home/dev/rudder-iac/cli/tests/testdata/project/substitution.vars.yaml",
	})

	assert.Equal(t, []string{
		"rudder-cli", "apply",
		"-l", "project/create",
		"--var-file", "project/substitution.vars.yaml",
	}, got)
	assert.Empty(t, gaps)
}

func TestArgvRewritesRelativeTestdataPaths(t *testing.T) {
	// Tests build these with filepath.Join("testdata", …), so they arrive relative.
	got, gaps := testRewriter().Argv([]string{"rudder-cli", "apply", "-l", "testdata/project/create"})

	assert.Equal(t, []string{"rudder-cli", "apply", "-l", "project/create"}, got)
	assert.Empty(t, gaps)
}

func TestArgvRewritesTempRootToWork(t *testing.T) {
	got, gaps := testRewriter().Argv([]string{"rudder-cli", "apply", "-l", "/tmp/TestProjectApply123/migrated/create"})

	assert.Equal(t, []string{"rudder-cli", "apply", "-l", "work/migrated/create"}, got)
	assert.Empty(t, gaps)
}

func TestArgvReportsUnrewritableAbsolutePath(t *testing.T) {
	got, gaps := testRewriter().Argv([]string{"rudder-cli", "apply", "-l", "/opt/somewhere/else"})

	assert.Equal(t, []string{"rudder-cli", "apply", "-l", "/opt/somewhere/else"}, got)
	assert.Equal(t, []string{"/opt/somewhere/else"}, gaps, "an unrewritten absolute path makes the demo unreproducible elsewhere")
}

func TestFixturesListsTestdataDirectoriesRead(t *testing.T) {
	got := testRewriter().Fixtures([]string{
		"rudder-cli", "apply",
		"-l", "/home/dev/rudder-iac/cli/tests/testdata/project/create",
		"--var-file", "testdata/project/substitution.vars.yaml",
	})

	assert.Equal(t, []string{
		"cli/tests/testdata/project/create",
		"cli/tests/testdata/project/substitution.vars.yaml",
	}, got)
}

func TestCopyTree(t *testing.T) {
	var (
		src = t.TempDir()
		dst = filepath.Join(t.TempDir(), "out")
	)
	require.NoError(t, os.MkdirAll(filepath.Join(src, "nested"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(src, "a.yaml"), []byte("a: 1"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(src, "nested", "b.yaml"), []byte("b: 2"), 0o644))

	require.NoError(t, CopyTree(src, dst))

	a, err := os.ReadFile(filepath.Join(dst, "a.yaml"))
	require.NoError(t, err)
	assert.Equal(t, "a: 1", string(a))

	b, err := os.ReadFile(filepath.Join(dst, "nested", "b.yaml"))
	require.NoError(t, err)
	assert.Equal(t, "b: 2", string(b))
}
```

- [ ] **Step 2: Run to verify failure**

```bash
go test ./cli/tests/demo/cmd/demogen/ -run 'TestArgv|TestFixtures|TestCopyTree' -v
```

Expected: FAIL to build — `undefined: Rewriter`, `undefined: CopyTree`.

- [ ] **Step 3: Implement rewriting**

Create `cli/tests/demo/cmd/demogen/rewrite.go`:

```go
package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const (
	// testdataRel is where the e2e suite keeps its spec fixtures.
	testdataRel = "cli/tests/testdata"

	// projectDir and workDir are where a generated demo keeps the same things,
	// relative to the demo's own directory.
	projectDir = "project"
	workDir    = "work"
)

// Rewriter turns the absolute paths a test run produced into paths a generated
// demo can use from its own directory.
type Rewriter struct {
	RepoRoot string // absolute path to the repository checkout
	BinPath  string // absolute path to the CLI binary TestMain built
	TempRoot string // the t.TempDir() root this test used, if any
}

// Argv returns a portable argv and any absolute path it could not rewrite.
// A gap is not fatal: the demo still runs where it was generated, and the gap
// is recorded so a reviewer knows the demo is not portable yet.
func (rw Rewriter) Argv(argv []string) ([]string, []string) {
	var (
		out  = make([]string, 0, len(argv))
		gaps []string
	)

	for i, a := range argv {
		switch {
		case i == 0 && a == rw.BinPath:
			out = append(out, "rudder-cli")
		case rw.TempRoot != "" && strings.HasPrefix(a, rw.TempRoot+string(os.PathSeparator)):
			out = append(out, filepath.Join(workDir, strings.TrimPrefix(a, rw.TempRoot+string(os.PathSeparator))))
		case rw.isTestdata(a):
			out = append(out, filepath.Join(projectDir, rw.testdataSuffix(a)))
		case filepath.IsAbs(a):
			out = append(out, a)
			gaps = append(gaps, a)
		default:
			out = append(out, a)
		}
	}

	return out, gaps
}

// Fixtures returns the repo-relative testdata paths a step reads, so the
// generator knows what to copy next to the demo.
func (rw Rewriter) Fixtures(argv []string) []string {
	var out []string

	for _, a := range argv {
		if !rw.isTestdata(a) {
			continue
		}
		out = append(out, filepath.Join(testdataRel, rw.testdataSuffix(a)))
	}

	return out
}

// isTestdata matches both the absolute form (an argument built from a path the
// harness resolved) and the relative form (filepath.Join("testdata", …), which
// tests use directly because they run with cli/tests as the working directory).
func (rw Rewriter) isTestdata(a string) bool {
	abs := filepath.Join(rw.RepoRoot, testdataRel)

	return strings.HasPrefix(a, abs+string(os.PathSeparator)) ||
		strings.HasPrefix(a, "testdata"+string(os.PathSeparator))
}

func (rw Rewriter) testdataSuffix(a string) string {
	abs := filepath.Join(rw.RepoRoot, testdataRel)
	if strings.HasPrefix(a, abs+string(os.PathSeparator)) {
		return strings.TrimPrefix(a, abs+string(os.PathSeparator))
	}

	return strings.TrimPrefix(a, "testdata"+string(os.PathSeparator))
}

// CopyTree copies a file or directory tree, creating parents as needed.
func CopyTree(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("stat %s: %w", src, err)
	}

	if !info.IsDir() {
		return copyFile(src, dst, info.Mode())
	}

	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return fmt.Errorf("relativising %s: %w", path, err)
		}

		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}

		fi, err := d.Info()
		if err != nil {
			return fmt.Errorf("stat %s: %w", path, err)
		}

		return copyFile(path, target, fi.Mode())
	})
}

func copyFile(src, dst string, mode fs.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", filepath.Dir(dst), err)
	}

	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("reading %s: %w", src, err)
	}

	if err := os.WriteFile(dst, data, mode); err != nil {
		return fmt.Errorf("writing %s: %w", dst, err)
	}

	return nil
}
```

- [ ] **Step 4: Run to verify the rewrite tests pass**

```bash
go test ./cli/tests/demo/cmd/demogen/ -run 'TestArgv|TestFixtures|TestCopyTree' -v
```

Expected: PASS, seven tests.

- [ ] **Step 5: Lint and commit**

```bash
export GVM_ROOT="$HOME/.gvm"
make lint
git add cli/tests/demo/cmd/demogen/rewrite.go cli/tests/demo/cmd/demogen/rewrite_test.go
git commit -m "test(demo): make recorded argv portable and collect fixtures

Rewrite the built binary, testdata paths and the t.TempDir root into paths a
generated demo can use from its own directory. Absolute paths that cannot be
rewritten are reported rather than silently shipped.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

## Task 6: Emit the demo directory

**Files:**
- Create: `cli/tests/demo/cmd/demogen/emit.go`
- Create: `cli/tests/demo/cmd/demogen/emit_test.go`
- Create: `cli/tests/demo/cmd/demogen/main.go`
- Create: `cli/tests/demo/cmd/demogen/testdata/journal.jsonl`
- Create: `cli/tests/demo/cmd/demogen/testdata/events.json`

**Interfaces:**
- Consumes: `Step`, `Join`, `ParseEvents` (Task 3); `DeriveProse`, `IsThin`, `Annotated`, `Verification` (Task 4); `Rewriter`, `CopyTree` (Task 5); `demo.Record` (Task 1).
- Produces:
  - `type Manifest struct { Test string; RecordedAt time.Time; Git GitInfo; Backend BackendInfo; CLIVersion string; Flags []string; Annotated bool; DurationSeconds float64; Steps []StepInfo; Gaps []string }`
  - `type GitInfo struct { Ref, SHA, Describe string; Dirty bool }`
  - `type BackendInfo struct { Profile, Kind, APIURL string }`
  - `type StepInfo struct { Test, Prose, Verification string; Annotated, Thin bool }`
  - `func Emit(dir string, steps []Step, rw Rewriter, m Manifest) error`

- [ ] **Step 1: Write the failing test**

Create `cli/tests/demo/cmd/demogen/emit_test.go`:

```go
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rudderlabs/rudder-iac/cli/tests/demo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func emitFixture(t *testing.T) (dir string, m Manifest) {
	t.Helper()

	dir = t.TempDir()
	steps := []Step{
		{
			Test: "TestProjectApply/rudder_specs/should_create_entities_in_catalog_from_project",
			Records: []demo.Record{
				{Kind: demo.KindExec, Start: at(1), End: at(3), Argv: []string{"/tmp/bin/rudder-cli", "apply", "-l", "testdata/project/create", "--confirm=false"}},
			},
		},
		{
			Test: "TestProjectApply/rudder_specs/verified",
			Records: []demo.Record{
				{Kind: demo.KindSay, Start: at(4), Text: "And the state the apply produced:"},
				{Kind: demo.KindExec, Start: at(5), End: at(6), Argv: []string{"/tmp/bin/rudder-cli", "workspace", "accounts", "list", "--json"}},
			},
		},
	}

	m = Manifest{
		Test:       "TestProjectApply",
		RecordedAt: time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC),
		Git:        GitInfo{Ref: "refs/heads/main", SHA: "1a2b3c4d", Describe: "v1.12.0-14-gce79745b"},
		Backend:    BackendInfo{Profile: "mini", Kind: "local", APIURL: "http://localhost:15580"},
		CLIVersion: "v1.12.0-14-gce79745b",
		Flags:      []string{"RUDDERSTACK_X_RETL_TABLE_SUPPORT=true"},
	}

	rw := Rewriter{RepoRoot: t.TempDir(), BinPath: "/tmp/bin/rudder-cli"}
	require.NoError(t, Emit(dir, steps, rw, m))

	return dir, m
}

func TestEmitWritesExecutableScript(t *testing.T) {
	dir, _ := emitFixture(t)

	info, err := os.Stat(filepath.Join(dir, "demo.sh"))
	require.NoError(t, err)
	assert.NotZero(t, info.Mode()&0o100, "demo.sh must be executable")
}

func TestEmitScriptContainsDerivedProseAndPortableArgv(t *testing.T) {
	dir, _ := emitFixture(t)

	data, err := os.ReadFile(filepath.Join(dir, "demo.sh"))
	require.NoError(t, err)
	script := string(data)

	assert.Contains(t, script, `p "# Create entities in catalog from project"`)
	assert.Contains(t, script, `pe "rudder-cli apply -l project/create --confirm=false"`)
	assert.NotContains(t, script, "/tmp/bin/rudder-cli", "the built binary path must not survive into the script")
	assert.Contains(t, script, `p "# And the state the apply produced:"`, "narration is emitted verbatim, not derived")
}

func TestEmitScriptFlagsInvisibleVerification(t *testing.T) {
	dir, _ := emitFixture(t)

	data, err := os.ReadFile(filepath.Join(dir, "demo.sh"))
	require.NoError(t, err)

	assert.Contains(t, string(data), "verified in Go, not visible here",
		"a step that ends in a write must say its proof is off-screen")
}

func TestEmitScriptExportsRecordedFlags(t *testing.T) {
	dir, _ := emitFixture(t)

	data, err := os.ReadFile(filepath.Join(dir, "demo.sh"))
	require.NoError(t, err)

	assert.Contains(t, string(data), "export RUDDERSTACK_X_RETL_TABLE_SUPPORT=true")
}

func TestEmitWritesManifest(t *testing.T) {
	dir, want := emitFixture(t)

	data, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	require.NoError(t, err)

	var got Manifest
	require.NoError(t, json.Unmarshal(data, &got))

	assert.Equal(t, want.Test, got.Test)
	assert.Equal(t, want.Git, got.Git)
	assert.Equal(t, want.Backend, got.Backend)
	assert.True(t, got.Annotated, "one step carries a demo.Say, so the demo is annotated")
	assert.Equal(t, []StepInfo{
		{
			Test:         "TestProjectApply/rudder_specs/should_create_entities_in_catalog_from_project",
			Prose:        "Create entities in catalog from project",
			Verification: "none",
			Annotated:    false,
			Thin:         false,
		},
		{
			Test:         "TestProjectApply/rudder_specs/verified",
			Prose:        "Verified",
			Verification: "visible",
			Annotated:    true,
			Thin:         true,
		},
	}, got.Steps)
}

func TestEmitWritesAnnotateWhenStepsAreUnannotated(t *testing.T) {
	dir, _ := emitFixture(t)

	data, err := os.ReadFile(filepath.Join(dir, "ANNOTATE.md"))
	require.NoError(t, err)

	body := string(data)
	assert.Contains(t, body, "should_create_entities_in_catalog_from_project", "the unannotated step is named")
	assert.Contains(t, body, "demo.Say", "the fix is spelled out")
	assert.NotContains(t, body, "/rudder_specs/verified\n", "an annotated step is not asked for again")
}

func TestEmitOmitsAnnotateWhenEveryStepIsAnnotated(t *testing.T) {
	dir := t.TempDir()
	steps := []Step{{
		Test: "TestA/only_step",
		Records: []demo.Record{
			{Kind: demo.KindSay, Start: at(1), Text: "narrated"},
			{Kind: demo.KindExec, Start: at(2), End: at(3), Argv: []string{"rudder-cli", "validate", "-l", "project"}},
		},
	}}

	require.NoError(t, Emit(dir, steps, Rewriter{RepoRoot: t.TempDir()}, Manifest{Test: "TestA"}))

	_, err := os.Stat(filepath.Join(dir, "ANNOTATE.md"))
	assert.True(t, os.IsNotExist(err), "a fully annotated demo asks for nothing")
}

func TestEmitWritesReadme(t *testing.T) {
	dir, _ := emitFixture(t)

	data, err := os.ReadFile(filepath.Join(dir, "README.md"))
	require.NoError(t, err)

	body := string(data)
	assert.Contains(t, body, "./demo.sh", "a reader must be told how to run it")
	assert.Contains(t, body, "http://localhost:15580", "and which backend it was recorded against")
	assert.Contains(t, body, "1a2b3c4d")
}
```

- [ ] **Step 2: Run to verify failure**

```bash
go test ./cli/tests/demo/cmd/demogen/ -run TestEmit -v
```

Expected: FAIL to build — `undefined: Emit`, `undefined: Manifest`.

- [ ] **Step 3: Implement emission**

Create `cli/tests/demo/cmd/demogen/emit.go`:

```go
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/rudderlabs/rudder-iac/cli/tests/demo"
)

// GitInfo is where the recorded binary came from.
type GitInfo struct {
	Ref      string `json:"ref"`
	SHA      string `json:"sha"`
	Describe string `json:"describe"`
	Dirty    bool   `json:"dirty"`
}

// BackendInfo is what the recording talked to. APIURL is omitted for
// production, where it is the well-known endpoint and says nothing.
type BackendInfo struct {
	Profile string `json:"profile"`
	Kind    string `json:"kind"`
	APIURL  string `json:"apiURL,omitempty"`
}

// StepInfo is one subtest as the demo presents it.
type StepInfo struct {
	Test         string `json:"test"`
	Prose        string `json:"prose"`
	Verification string `json:"verification"`
	Annotated    bool   `json:"annotated"`
	Thin         bool   `json:"thin"`
}

// Manifest is the demo's provenance and shape. It is what lets a viewer know
// which branch and which backend produced what they are looking at.
type Manifest struct {
	Test            string       `json:"test"`
	RecordedAt      time.Time    `json:"recordedAt"`
	Git             GitInfo      `json:"git"`
	Backend         BackendInfo  `json:"backend"`
	CLIVersion      string       `json:"cliVersion"`
	Flags           []string     `json:"flags,omitempty"`
	Annotated       bool         `json:"annotated"`
	DurationSeconds float64      `json:"durationSeconds"`
	Steps           []StepInfo   `json:"steps"`
	Gaps            []string     `json:"gaps,omitempty"`
}

// Emit writes a complete demo directory: the script, its fixtures, the
// manifest, a README, and — when narration is missing — the request for it.
func Emit(dir string, steps []Step, rw Rewriter, m Manifest) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", dir, err)
	}

	script, infos, gaps, fixtures := build(steps, rw, m)

	m.Steps = infos
	m.Gaps = gaps
	m.Annotated = anyAnnotated(infos)

	if err := os.WriteFile(filepath.Join(dir, "demo.sh"), []byte(script), 0o755); err != nil {
		return fmt.Errorf("writing demo.sh: %w", err)
	}

	if err := writeJSON(filepath.Join(dir, "manifest.json"), m); err != nil {
		return err
	}

	if err := copyFixtures(dir, rw.RepoRoot, fixtures); err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte(readme(m)), 0o644); err != nil {
		return fmt.Errorf("writing README.md: %w", err)
	}

	return writeAnnotate(dir, m)
}

// build assembles the script and everything derived alongside it in one pass,
// so the manifest cannot describe a script it did not produce.
func build(steps []Step, rw Rewriter, m Manifest) (script string, infos []StepInfo, gaps []string, fixtures []string) {
	var b strings.Builder

	b.WriteString("#!/usr/bin/env bash\n")
	b.WriteString(fmt.Sprintf("# Generated from %s by demogen. Do not edit — re-record instead.\n", m.Test))
	b.WriteString("# See README.md for how to run this, and manifest.json for where it came from.\n")
	b.WriteString("set -uo pipefail\n")
	b.WriteString(`cd "$(dirname "$0")" || exit 1` + "\n")
	b.WriteString(`: "${DEMO_MAGIC:=$HOME/workspace/demo-magic/demo-magic.sh}"` + "\n")
	b.WriteString(`. "$DEMO_MAGIC"` + "\n")
	b.WriteString("TYPE_SPEED=90\nNO_WAIT=true\n")
	b.WriteString(`DEMO_PROMPT="${GREEN}➜ ${CYAN}` + m.Test + ` ${COLOR_RESET}\$ "` + "\n")
	b.WriteString(`[ -f ./profile.env ] && . ./profile.env` + "\n")

	for _, f := range m.Flags {
		b.WriteString("export " + f + "\n")
	}

	b.WriteString("\nclear\n")

	for _, s := range steps {
		info := StepInfo{
			Test:         s.Test,
			Prose:        DeriveProse(s.Test),
			Verification: Verification(s),
			Annotated:    Annotated(s),
			Thin:         IsThin(s.Test),
		}
		infos = append(infos, info)

		b.WriteString("\n")
		b.WriteString(sayLine("# " + info.Prose))

		for _, r := range s.Records {
			if r.Kind == demo.KindSay {
				b.WriteString(sayLine("# " + r.Text))
				continue
			}

			argv, stepGaps := rw.Argv(r.Argv)
			gaps = append(gaps, stepGaps...)
			fixtures = append(fixtures, rw.Fixtures(r.Argv)...)

			b.WriteString("pe " + quote(strings.Join(argv, " ")) + "\n")
		}

		if info.Verification == "none" {
			b.WriteString(sayLine("# (verified in Go, not visible here — see " + s.Test + ")"))
		}
	}

	b.WriteString("\n")
	b.WriteString(annotationFooter())

	return b.String(), infos, dedupe(gaps), dedupe(fixtures)
}

// annotationFooter asks for narration the demo does not have. It never runs
// during recording — asciinema sets RUDDER_DEMO_RECORDING — because a prompt in
// the middle of a cast is exactly the thing nobody wants to watch.
func annotationFooter() string {
	return `
if [ -z "${RUDDER_DEMO_RECORDING:-}" ] && [ -f ANNOTATE.md ]; then
  if [ -t 1 ] && [ -z "${RUDDER_DEMO_AGENT:-}" ]; then
    printf '\n\033[33m%s\033[0m\n' "Some steps in this demo have no narration. See ANNOTATE.md."
  else
    printf '\n%s\n' "Unannotated demo. ANNOTATE.md lists the steps needing demo.Say and where to add them."
    exit 3
  fi
fi
`
}

func sayLine(s string) string {
	return "p " + quote(s) + "\n"
}

// quote wraps a line for the shell. demo-magic takes a single argument, and the
// recorded commands contain flags and JSON filters that must not be re-split.
func quote(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`, "$", `\$`, "`", "\\`").Replace(s) + `"`
}

func anyAnnotated(infos []StepInfo) bool {
	for _, i := range infos {
		if i.Annotated {
			return true
		}
	}

	return false
}

func copyFixtures(dir, repoRoot string, fixtures []string) error {
	for _, rel := range fixtures {
		var (
			src = filepath.Join(repoRoot, rel)
			dst = filepath.Join(dir, projectDir, strings.TrimPrefix(rel, testdataRel+"/"))
		)

		if _, err := os.Stat(src); err != nil {
			// A fixture the recording referenced but this checkout lacks is a
			// portability gap, not a reason to abandon the demo.
			continue
		}

		if err := CopyTree(src, dst); err != nil {
			return fmt.Errorf("copying fixture %s: %w", rel, err)
		}
	}

	return nil
}

func writeAnnotate(dir string, m Manifest) error {
	var missing []StepInfo
	for _, s := range m.Steps {
		if !s.Annotated {
			missing = append(missing, s)
		}
	}

	path := filepath.Join(dir, "ANNOTATE.md")
	if len(missing) == 0 {
		// A demo that was annotated since the last generation must stop asking.
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("removing stale ANNOTATE.md: %w", err)
		}

		return nil
	}

	var b strings.Builder
	b.WriteString("# Narration wanted: " + m.Test + "\n\n")
	b.WriteString("These steps show what ran but not why it matters. Add a `demo.Say` call\n")
	b.WriteString("immediately before the command in each subtest below, then re-record.\n\n")
	b.WriteString("```go\n\tdemo.Say(t, \"Why this step matters.\")\n```\n\n")

	for _, s := range missing {
		b.WriteString("- `" + s.Test + "`\n")
		b.WriteString("  - derived prose: \"" + s.Prose + "\"")
		if s.Thin {
			b.WriteString(" — **thin**, the subtest name says nothing")
		}
		b.WriteString("\n")
		if s.Verification == "none" {
			b.WriteString("  - verification is invisible here; if a read-only CLI command can show it, run one\n")
		}
	}

	b.WriteString("\nWhen this is done, open a PR. `demos/GAPS.md` lists anything that needed a\n")
	b.WriteString("command the CLI does not have yet — those are tickets, not narration.\n")

	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return fmt.Errorf("writing ANNOTATE.md: %w", err)
	}

	return nil
}

func readme(m Manifest) string {
	var b strings.Builder

	b.WriteString("# " + m.Test + "\n\n")
	b.WriteString("Generated from a real run of `" + m.Test + "`. Run it yourself:\n\n")
	b.WriteString("```bash\n./demo.sh\n```\n\n")
	b.WriteString("## Provenance\n\n")
	b.WriteString("| | |\n|---|---|\n")
	b.WriteString("| recorded | " + m.RecordedAt.Format(time.RFC3339) + " |\n")
	b.WriteString("| ref | `" + m.Git.Ref + "` |\n")
	b.WriteString("| commit | `" + m.Git.SHA + "`")
	if m.Git.Dirty {
		b.WriteString(" (working tree dirty)")
	}
	b.WriteString(" |\n")
	b.WriteString("| cli | `" + m.CLIVersion + "` |\n")
	b.WriteString("| backend | " + m.Backend.Profile + " (" + m.Backend.Kind + ")")
	if m.Backend.APIURL != "" {
		b.WriteString(" — `" + m.Backend.APIURL + "`")
	}
	b.WriteString(" |\n")

	b.WriteString("| narration | ")
	if m.Annotated {
		b.WriteString("annotated")
	} else {
		b.WriteString("derived from subtest names only")
	}
	b.WriteString(" |\n")

	if len(m.Gaps) > 0 {
		b.WriteString("\n## Not portable yet\n\nThese absolute paths could not be rewritten:\n\n")
		for _, g := range m.Gaps {
			b.WriteString("- `" + g + "`\n")
		}
	}

	return b.String()
}

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding %s: %w", filepath.Base(path), err)
	}

	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}

	return nil
}

func dedupe(in []string) []string {
	if len(in) == 0 {
		return nil
	}

	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	sort.Strings(out)

	return out
}
```

- [ ] **Step 4: Run the emit tests to verify they pass**

```bash
go test ./cli/tests/demo/cmd/demogen/ -run TestEmit -v
```

Expected: PASS, eight tests.

- [ ] **Step 5: Write `main.go`**

Create `cli/tests/demo/cmd/demogen/main.go`:

```go
// Command demogen turns a recorded e2e run into demo directories.
//
//	demogen -journal journal.jsonl -events events.json -out demos \
//	        -repo-root . -bin /tmp/rudder-cli-bin-x/rudder-cli \
//	        -profile mini -backend-kind local -api-url http://localhost:15580
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/rudderlabs/rudder-iac/cli/tests/demo"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "demogen:", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		journalPath = flag.String("journal", "", "path to the JSONL journal a recorded run produced")
		eventsPath  = flag.String("events", "", "path to the go test -json event stream")
		outDir      = flag.String("out", "demos", "directory to write demo directories into")
		repoRoot    = flag.String("repo-root", ".", "repository checkout the run happened in")
		binPath     = flag.String("bin", "", "absolute path of the CLI binary the run used")
		tempRoot    = flag.String("temp-root", "", "t.TempDir root the run used, if any")
		profile     = flag.String("profile", "", "backend profile name, e.g. mini")
		backendKind = flag.String("backend-kind", "", "local, staging or production")
		apiURL      = flag.String("api-url", "", "backend URL; omitted from the manifest for production")
	)
	flag.Parse()

	if *journalPath == "" || *eventsPath == "" {
		return fmt.Errorf("both -journal and -events are required")
	}

	records, err := readJournal(*journalPath)
	if err != nil {
		return err
	}

	eventsFile, err := os.Open(*eventsPath)
	if err != nil {
		return fmt.Errorf("opening events: %w", err)
	}
	defer eventsFile.Close()

	events, err := ParseEvents(eventsFile, e2ePackage)
	if err != nil {
		return err
	}

	steps, err := Join(events, records)
	if err != nil {
		return err
	}

	absRoot, err := filepath.Abs(*repoRoot)
	if err != nil {
		return fmt.Errorf("resolving repo root: %w", err)
	}

	rw := Rewriter{RepoRoot: absRoot, BinPath: *binPath, TempRoot: *tempRoot}
	git := gitInfo(absRoot)

	backend := BackendInfo{Profile: *profile, Kind: *backendKind, APIURL: *apiURL}
	if backend.Kind == "production" {
		backend.APIURL = ""
	}

	for test, group := range groupByTopLevel(steps) {
		m := Manifest{
			Test:            test,
			RecordedAt:      time.Now().UTC(),
			Git:             git,
			Backend:         backend,
			CLIVersion:      git.Describe,
			Flags:           flagsOf(group),
			DurationSeconds: durationOf(group),
		}

		dir := filepath.Join(*outDir, test)
		if err := Emit(dir, group, rw, m); err != nil {
			return fmt.Errorf("emitting %s: %w", test, err)
		}

		fmt.Println("wrote", dir)
	}

	return nil
}

func readJournal(path string) ([]demo.Record, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening journal: %w", err)
	}
	defer f.Close()

	var (
		dec  = json.NewDecoder(f)
		recs []demo.Record
	)
	for dec.More() {
		var r demo.Record
		if err := dec.Decode(&r); err != nil {
			return nil, fmt.Errorf("decoding journal record: %w", err)
		}
		recs = append(recs, r)
	}

	return recs, nil
}

// groupByTopLevel splits steps by the test function they belong to, so each
// top-level test becomes its own demo.
func groupByTopLevel(steps []Step) map[string][]Step {
	out := map[string][]Step{}

	for _, s := range steps {
		top := s.Test
		if i := strings.Index(top, "/"); i >= 0 {
			top = top[:i]
		}
		out[top] = append(out[top], s)
	}

	return out
}

// flagsOf returns the RUDDERSTACK_* settings the steps ran under, as KEY=VALUE.
// Redacted values are dropped: a demo must not instruct a reader to export the
// literal string "<redacted>".
func flagsOf(steps []Step) []string {
	seen := map[string]string{}

	for _, s := range steps {
		for _, r := range s.Records {
			for k, v := range r.Env {
				if v == demo.Redacted {
					continue
				}
				seen[k] = v
			}
		}
	}

	var out []string
	for k, v := range seen {
		out = append(out, k+"="+v)
	}
	sort.Strings(out)

	return out
}

func durationOf(steps []Step) float64 {
	var first, last time.Time

	for _, s := range steps {
		for _, r := range s.Records {
			if first.IsZero() || r.Start.Before(first) {
				first = r.Start
			}
			if r.End.After(last) {
				last = r.End
			}
		}
	}

	if first.IsZero() {
		return 0
	}

	return last.Sub(first).Seconds()
}

// gitInfo reads provenance from the checkout. Failures yield empty fields
// rather than an error: a demo generated outside a git worktree is still worth
// having, it just cannot say where it came from.
func gitInfo(root string) GitInfo {
	run := func(args ...string) string {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		out, err := cmd.Output()
		if err != nil {
			return ""
		}

		return strings.TrimSpace(string(out))
	}

	ref := os.Getenv("GITHUB_REF")
	if ref == "" {
		ref = run("rev-parse", "--symbolic-full-name", "HEAD")
	}

	return GitInfo{
		Ref:      ref,
		SHA:      run("rev-parse", "--short", "HEAD"),
		Describe: run("describe", "--tags", "--always"),
		Dirty:    run("status", "--porcelain") != "",
	}
}
```

Add `"encoding/json"` and `"sort"` to the import block.

- [ ] **Step 6: Verify the binary builds and its help is sane**

```bash
export GVM_ROOT="$HOME/.gvm"
go build -o /tmp/demogen ./cli/tests/demo/cmd/demogen
/tmp/demogen -h 2>&1 | head -20
```

Expected: the flag list prints; no build errors.

- [ ] **Step 7: Run the whole demogen package test suite**

```bash
go test ./cli/tests/demo/... -v 2>&1 | tail -20
```

Expected: PASS for every test written in Tasks 3–6.

- [ ] **Step 8: Lint and commit**

```bash
export GVM_ROOT="$HOME/.gvm"
make lint
git add cli/tests/demo/cmd/demogen
git commit -m "test(demo): emit demo directories from a recorded run

One directory per top-level test: demo.sh, manifest.json with git and backend
provenance, copied fixtures, a README, and ANNOTATE.md where narration is
missing. A fully annotated demo stops asking.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

## Task 7: Make targets and profiles

**Files:**
- Modify: `Makefile` (append a Demos section after `test-all`, around line 69)
- Create: `demos/profiles/mini.env`
- Create: `demos/profiles/cloud.env`
- Create: `demos/.gitignore`

**Interfaces:**
- Consumes: the `demogen` binary from Task 6; `demo.EnvJournal` from Task 1.
- Produces: `make demo-record TEST=<name> PROFILE=<name>` and `make demo-generate`, leaving `demos/<Test>/` on disk.

- [ ] **Step 1: Write the profiles**

Create `demos/profiles/mini.env`:

```bash
# Local rudder-mini stack. Source this, or let demo.sh pick it up as profile.env.
#
# Holds a URL and a token *reference*, never a token: RUDDERSTACK_ACCESS_TOKEN
# is expected to already be in your environment, or in ~/.rudder/config.json,
# which the CLI falls back to on its own.
export RUDDERSTACK_API_URL="${RUDDERSTACK_API_URL:-http://localhost:15580}"
export RUDDER_DEMO_BACKEND_KIND=local
export RUDDER_DEMO_PROFILE=mini
```

Create `demos/profiles/cloud.env`:

```bash
# A cloud backend supplied by the PR or the CI job running this.
#
# RUDDERSTACK_API_URL and RUDDERSTACK_ACCESS_TOKEN must already be set; this
# file only records how to classify what they point at.
export RUDDER_DEMO_BACKEND_KIND="${RUDDER_DEMO_BACKEND_KIND:-staging}"
export RUDDER_DEMO_PROFILE=cloud
```

Create `demos/.gitignore`:

```gitignore
# Recording intermediates. The generated demo directories are committed; the
# raw journal and event stream that produced them are not — they are large,
# machine-specific, and fully described by manifest.json.
*.jsonl
*.events.json
```

- [ ] **Step 2: Add the Make targets**

Append to `Makefile` after the `test-all` target:

```makefile
##@ Demos

DEMO_TEST ?= TestAccountsApply
DEMO_PROFILE ?= mini
DEMO_OUT ?= demos
DEMO_JOURNAL ?= $(DEMO_OUT)/$(DEMO_TEST).jsonl
DEMO_EVENTS ?= $(DEMO_OUT)/$(DEMO_TEST).events.json

.PHONY: demo-record
demo-record: ## Run one e2e test with journaling on (TEST=..., PROFILE=mini|cloud)
	@mkdir -p $(DEMO_OUT)
	@rm -f $(DEMO_JOURNAL)
	@set -a; . ./demos/profiles/$(DEMO_PROFILE).env; set +a; \
	RUDDER_DEMO_JOURNAL=$(abspath $(DEMO_JOURNAL)) \
	$(GO) test -json -timeout 20m ./cli/tests -run '^$(DEMO_TEST)$$' -v > $(DEMO_EVENTS) || \
		{ echo "e2e run failed — see $(DEMO_EVENTS)"; exit 1; }
	@echo "journal: $(DEMO_JOURNAL)"
	@echo "events:  $(DEMO_EVENTS)"

.PHONY: demo-generate
demo-generate: ## Generate demos/<Test>/ from the last demo-record
	@set -a; . ./demos/profiles/$(DEMO_PROFILE).env; set +a; \
	$(GO) run ./cli/tests/demo/cmd/demogen \
		-journal $(DEMO_JOURNAL) \
		-events $(DEMO_EVENTS) \
		-out $(DEMO_OUT) \
		-repo-root . \
		-bin "$$(ls -d /tmp/rudder-cli-bin-*/rudder-cli 2>/dev/null | tail -1)" \
		-profile "$$RUDDER_DEMO_PROFILE" \
		-backend-kind "$$RUDDER_DEMO_BACKEND_KIND" \
		-api-url "$$RUDDERSTACK_API_URL"

.PHONY: demo
demo: demo-record demo-generate ## Record and generate one demo end to end
```

Note the deliberate choices: `-run '^$(DEMO_TEST)$$'` anchors so `TestAccountsApply` does not also match `TestAccountsApplySomethingElse`; `./cli/tests` rather than `./cli/tests/...` keeps the parallel `helpers` package out of the event stream, as the spec requires; and `$(abspath …)` is needed because `go test` runs with the package directory as its working directory, so a relative journal path would land in `cli/tests/`.

- [ ] **Step 3: Verify the targets parse and print what they will do**

```bash
export GVM_ROOT="$HOME/.gvm"
make -n demo-record DEMO_TEST=TestAccountsApply | head
make help | grep demo
```

Expected: the recipe prints with `RUDDER_DEMO_JOURNAL` and `-run '^TestAccountsApply$'` visible; `make help` lists all three demo targets with their descriptions.

- [ ] **Step 4: Record and generate one real demo**

This needs a reachable backend. With `mini` up and `RUDDERSTACK_ACCESS_TOKEN` exported:

```bash
export GVM_ROOT="$HOME/.gvm"
make demo DEMO_TEST=TestAccountsApply DEMO_PROFILE=mini
```

Expected: `demos/TestAccountsApply/` exists and contains `demo.sh`, `manifest.json`, `project/`, `README.md`, and — since nothing is annotated yet — `ANNOTATE.md`.

- [ ] **Step 5: Inspect the generated demo and run it by hand**

```bash
cat demos/TestAccountsApply/manifest.json
cat demos/TestAccountsApply/demo.sh
cd demos/TestAccountsApply && ./demo.sh; echo "exit=$?"
```

Expected: the script runs the same commands the test ran, against the same backend; exit code `3` with the ANNOTATE notice, because no step carries a `demo.Say` yet. Check `manifest.json` names the right branch, SHA and `http://localhost:15580`.

- [ ] **Step 6: Commit**

```bash
export GVM_ROOT="$HOME/.gvm"
make lint
git add Makefile demos/profiles demos/.gitignore demos/TestAccountsApply
git commit -m "test(demo): make targets to record and generate demos

A profile is an env file, not a subsystem — the CLI already selects a backend
from RUDDERSTACK_API_URL and RUDDERSTACK_ACCESS_TOKEN. Records one test at a
time against ./cli/tests so the parallel helpers package stays out of the
event stream.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

## Task 8: Record casts with provenance

**Files:**
- Create: `scripts/demo-cast.sh`
- Modify: `Makefile` (add `demo-cast` to the Demos section)
- Create: `casts/.gitkeep`

**Interfaces:**
- Consumes: `demos/<Test>/demo.sh` and `manifest.json` from Task 6.
- Produces: `casts/<Test>.cast`, an asciicast v2 file whose header carries the provenance from the manifest.

- [ ] **Step 1: Write the recording script**

Create `scripts/demo-cast.sh`:

```bash
#!/usr/bin/env bash
# Record a generated demo to an asciicast, stamping its provenance into the
# cast header so a recording can never be mistaken for a different branch or a
# different backend.
#
# Usage: scripts/demo-cast.sh TestAccountsApply [outdir]
set -euo pipefail

TEST="${1:?usage: demo-cast.sh <TestName> [outdir]}"
OUT="${2:-casts}"
DEMO_DIR="demos/$TEST"
CAST="$OUT/$TEST.cast"

[ -x "$DEMO_DIR/demo.sh" ] || { echo "no generated demo at $DEMO_DIR — run make demo first" >&2; exit 1; }
command -v asciinema >/dev/null || { echo "asciinema not installed" >&2; exit 1; }
command -v jq >/dev/null || { echo "jq not installed" >&2; exit 1; }

mkdir -p "$OUT"
rm -f "$CAST"

# RUDDER_DEMO_RECORDING suppresses the annotation footer: a prompt in the
# middle of a cast is the one thing nobody wants to watch.
RUDDER_DEMO_RECORDING=1 asciinema rec \
  --command "$DEMO_DIR/demo.sh" \
  --title "$TEST" \
  --overwrite \
  "$CAST"

# Merge the manifest into the cast's v2 header. asciicast v2 line 1 is a JSON
# object and tolerates extra keys, so viewers that do not know about this one
# ignore it and castplay can render it as a provenance chip.
tmp="$(mktemp)"
{
  head -n 1 "$CAST" | jq -c --slurpfile m "$DEMO_DIR/manifest.json" '. + {rudderDemo: $m[0]}'
  tail -n +2 "$CAST"
} > "$tmp"
mv "$tmp" "$CAST"

echo "wrote $CAST"
jq -r '.rudderDemo | "  \(.git.ref) @ \(.git.sha) · \(.backend.profile) (\(.backend.kind)) · \(if .annotated then "annotated" else "derived" end)"' \
  <(head -n 1 "$CAST")
```

- [ ] **Step 2: Make it executable and add the Make target**

```bash
chmod +x scripts/demo-cast.sh
mkdir -p casts && touch casts/.gitkeep
```

Append to the Demos section of `Makefile`:

```makefile
.PHONY: demo-cast
demo-cast: ## Record demos/<TEST>/demo.sh to casts/<TEST>.cast with provenance
	@./scripts/demo-cast.sh $(DEMO_TEST)
```

- [ ] **Step 3: Verify the script's guards fire before anything is recorded**

```bash
./scripts/demo-cast.sh NoSuchTest; echo "exit=$?"
```

Expected: `no generated demo at demos/NoSuchTest — run make demo first`, exit 1.

- [ ] **Step 4: Record a real cast**

```bash
export GVM_ROOT="$HOME/.gvm"
make demo-cast DEMO_TEST=TestAccountsApply
```

Expected: `wrote casts/TestAccountsApply.cast`, followed by a provenance line like
`refs/heads/main @ 1a2b3c4 · mini (local) · derived`.

- [ ] **Step 5: Verify the header round-trips and holds no secrets**

```bash
head -n 1 casts/TestAccountsApply.cast | jq '.rudderDemo | {git, backend, annotated}'
grep -c 'RUDDERSTACK_ACCESS_TOKEN=' casts/TestAccountsApply.cast || true
```

Expected: the provenance object prints with the right branch and backend; the token grep returns `0` matches.

- [ ] **Step 6: Commit**

```bash
git add scripts/demo-cast.sh Makefile casts
git commit -m "test(demo): record casts with provenance in the header

asciicast v2's header is a JSON object that tolerates extra keys, so the
manifest rides along in the cast itself — a recording cannot be separated
from the branch and backend that produced it.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

## Task 9: The drift check

**Files:**
- Create: `scripts/demo-check.sh`
- Create: `cli/tests/demo/cmd/demogen/diff.go`
- Create: `cli/tests/demo/cmd/demogen/diff_test.go`
- Modify: `Makefile` (add `demo-check`)

**Interfaces:**
- Consumes: `demo.Record` (Task 1), `Rewriter` (Task 5).
- Produces:
  - `func DiffArgv(want, got []demo.Record, rw Rewriter) []string` — human-readable differences; empty means no drift.
  - `demogen -diff-want X -diff-got Y` mode.

- [ ] **Step 1: Write the failing tests**

Create `cli/tests/demo/cmd/demogen/diff_test.go`:

```go
package main

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/tests/demo"
	"github.com/stretchr/testify/assert"
)

func TestDiffArgvReportsNothingWhenScriptMatchesTest(t *testing.T) {
	rw := Rewriter{RepoRoot: "/repo", BinPath: "/tmp/bin/rudder-cli"}

	want := []demo.Record{
		{Kind: demo.KindExec, Argv: []string{"/tmp/bin/rudder-cli", "apply", "-l", "testdata/project/create"}},
	}
	got := []demo.Record{
		{Kind: demo.KindExec, Argv: []string{"rudder-cli", "apply", "-l", "project/create"}},
	}

	assert.Empty(t, DiffArgv(want, got, rw))
}

func TestDiffArgvReportsChangedCommand(t *testing.T) {
	rw := Rewriter{RepoRoot: "/repo", BinPath: "/tmp/bin/rudder-cli"}

	want := []demo.Record{{Kind: demo.KindExec, Argv: []string{"/tmp/bin/rudder-cli", "apply", "-l", "testdata/project/create"}}}
	got := []demo.Record{{Kind: demo.KindExec, Argv: []string{"rudder-cli", "apply", "-l", "project/update"}}}

	out := DiffArgv(want, got, rw)
	assert.Len(t, out, 1)
	assert.Contains(t, out[0], "step 1")
	assert.Contains(t, out[0], "project/create")
	assert.Contains(t, out[0], "project/update")
}

func TestDiffArgvReportsMissingStep(t *testing.T) {
	rw := Rewriter{RepoRoot: "/repo"}

	want := []demo.Record{
		{Kind: demo.KindExec, Argv: []string{"rudder-cli", "apply"}},
		{Kind: demo.KindExec, Argv: []string{"rudder-cli", "validate"}},
	}
	got := []demo.Record{{Kind: demo.KindExec, Argv: []string{"rudder-cli", "apply"}}}

	out := DiffArgv(want, got, rw)
	assert.Len(t, out, 1)
	assert.Contains(t, out[0], "missing")
	assert.Contains(t, out[0], "validate")
}

func TestDiffArgvIgnoresNarration(t *testing.T) {
	rw := Rewriter{RepoRoot: "/repo"}

	want := []demo.Record{
		{Kind: demo.KindSay, Text: "narration exists only in the test"},
		{Kind: demo.KindExec, Argv: []string{"rudder-cli", "apply"}},
	}
	got := []demo.Record{{Kind: demo.KindExec, Argv: []string{"rudder-cli", "apply"}}}

	assert.Empty(t, DiffArgv(want, got, rw), "a replayed script runs commands, not demo.Say calls")
}
```

- [ ] **Step 2: Run to verify failure**

```bash
go test ./cli/tests/demo/cmd/demogen/ -run TestDiffArgv -v
```

Expected: FAIL to build — `undefined: DiffArgv`.

- [ ] **Step 3: Implement the diff**

Create `cli/tests/demo/cmd/demogen/diff.go`:

```go
package main

import (
	"fmt"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/tests/demo"
)

// DiffArgv compares the commands a test ran against the commands its generated
// script ran, and returns one line per difference.
//
// This is what keeps a generated demo honest. Everything else in this tool is a
// transformation nobody re-checks; a demo that has quietly stopped matching the
// test it claims to project is worse than no demo, because it is believed.
//
// The test's records are normalised through the same Rewriter the generator
// used, so the comparison is between portable forms and path rewriting cannot
// register as drift.
func DiffArgv(want, got []demo.Record, rw Rewriter) []string {
	var (
		wantCmds = portableCommands(want, rw)
		gotCmds  = portableCommands(got, rw)
		out      []string
	)

	for i := range max(len(wantCmds), len(gotCmds)) {
		switch {
		case i >= len(gotCmds):
			out = append(out, fmt.Sprintf("step %d missing from the script: %s", i+1, wantCmds[i]))
		case i >= len(wantCmds):
			out = append(out, fmt.Sprintf("step %d extra in the script: %s", i+1, gotCmds[i]))
		case wantCmds[i] != gotCmds[i]:
			out = append(out, fmt.Sprintf("step %d differs:\n  test:   %s\n  script: %s", i+1, wantCmds[i], gotCmds[i]))
		}
	}

	return out
}

// portableCommands reduces records to the comparable form: exec steps only,
// with paths rewritten. Narration is dropped — demo.Say exists in the test and
// is printed by the script, never executed by either.
func portableCommands(records []demo.Record, rw Rewriter) []string {
	var out []string

	for _, r := range records {
		if r.Kind != demo.KindExec {
			continue
		}

		argv, _ := rw.Argv(r.Argv)
		out = append(out, strings.Join(argv, " "))
	}

	return out
}
```

Add a `-diff-want` / `-diff-got` mode to `main.go`'s `run()`, immediately after `flag.Parse()`:

```go
	if *diffWant != "" || *diffGot != "" {
		return runDiff(*diffWant, *diffGot, *repoRoot, *binPath, *tempRoot)
	}
```

with the flags declared alongside the others:

```go
		diffWant = flag.String("diff-want", "", "journal from the original test run, for drift checking")
		diffGot  = flag.String("diff-got", "", "journal from replaying the generated script")
```

and the handler at the bottom of `main.go`:

```go
// runDiff reports drift between a test run and a replay of its generated
// script. It exits non-zero through its error return, so a Make target or CI
// job fails on drift without any extra wiring.
func runDiff(wantPath, gotPath, repoRoot, binPath, tempRoot string) error {
	want, err := readJournal(wantPath)
	if err != nil {
		return err
	}

	got, err := readJournal(gotPath)
	if err != nil {
		return err
	}

	absRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		return fmt.Errorf("resolving repo root: %w", err)
	}

	diffs := DiffArgv(want, got, Rewriter{RepoRoot: absRoot, BinPath: binPath, TempRoot: tempRoot})
	if len(diffs) == 0 {
		fmt.Println("no drift")
		return nil
	}

	for _, d := range diffs {
		fmt.Println(d)
	}

	return fmt.Errorf("%d difference(s) between the test and its generated demo", len(diffs))
}
```

- [ ] **Step 4: Run to verify the diff tests pass**

```bash
go test ./cli/tests/demo/cmd/demogen/ -run TestDiffArgv -v
```

Expected: PASS, four tests.

- [ ] **Step 5: Write the replay script**

Create `scripts/demo-check.sh`:

```bash
#!/usr/bin/env bash
# Replay a generated demo with journaling on and diff what it ran against what
# the test ran. Non-zero exit means the demo no longer projects its test.
#
# Usage: scripts/demo-check.sh TestAccountsApply
set -euo pipefail

TEST="${1:?usage: demo-check.sh <TestName>}"
DEMO_DIR="demos/$TEST"
WANT="demos/$TEST.jsonl"
GOT="$(mktemp -t demo-replay-XXXXXX.jsonl)"

[ -x "$DEMO_DIR/demo.sh" ] || { echo "no generated demo at $DEMO_DIR" >&2; exit 1; }
[ -f "$WANT" ] || { echo "no recorded journal at $WANT — run make demo-record first" >&2; exit 1; }

# RUDDER_DEMO_RECORDING suppresses the annotation footer's exit 3, which would
# otherwise read as a replay failure.
RUDDER_DEMO_RECORDING=1 RUDDER_DEMO_JOURNAL="$GOT" "$DEMO_DIR/demo.sh" >/dev/null

go run ./cli/tests/demo/cmd/demogen \
  -diff-want "$WANT" \
  -diff-got "$GOT" \
  -repo-root . \
  -bin "$(ls -d /tmp/rudder-cli-bin-*/rudder-cli 2>/dev/null | tail -1)"
```

`-bin` matters only for the original journal, which holds the absolute path of the binary `TestMain` built. The replayed script calls `rudder-cli` from `PATH`, so its journal is already portable. Both sides run through the same `Rewriter`, which is what makes the two comparable.

- [ ] **Step 6: Make it executable and add the target**

```bash
chmod +x scripts/demo-check.sh
```

Append to the Demos section of `Makefile`:

```makefile
.PHONY: demo-check
demo-check: ## Replay demos/<TEST>/demo.sh and fail if it drifted from the test
	@./scripts/demo-check.sh $(DEMO_TEST)
```

- [ ] **Step 7: Verify no drift on a freshly generated demo**

```bash
export GVM_ROOT="$HOME/.gvm"
make demo-check DEMO_TEST=TestAccountsApply
```

Expected: `no drift`, exit 0.

- [ ] **Step 8: Verify drift is actually caught**

Introduce a deliberate difference, confirm it fails, then undo it:

```bash
sed -i.bak 's/--confirm=false/--confirm=true/' demos/TestAccountsApply/demo.sh
make demo-check DEMO_TEST=TestAccountsApply; echo "exit=$?"
mv demos/TestAccountsApply/demo.sh.bak demos/TestAccountsApply/demo.sh
```

Expected: the first run prints `step N differs:` showing both forms and exits non-zero. A check that cannot fail is not a check — do not skip this step.

- [ ] **Step 9: Lint and commit**

```bash
export GVM_ROOT="$HOME/.gvm"
make lint
git add scripts/demo-check.sh cli/tests/demo/cmd/demogen Makefile
git commit -m "test(demo): fail when a generated demo drifts from its test

Replay the script with journaling on and diff its commands against the test's,
both normalised through the same rewriter so path rewriting cannot register as
drift. A demo that silently stopped matching its test is worse than none,
because it is believed.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

## Task 10: Annotate one demo end to end, and write the gaps file

**Files:**
- Modify: `cli/tests/command_accounts_apply_test.go` (add `demo.Say` calls)
- Create: `demos/GAPS.md`
- Modify: `cli/tests/demo/cmd/demogen/main.go` (write `GAPS.md` across all demos)
- Modify: `cli/tests/README.md` (document the demo workflow)

**Interfaces:**
- Consumes: everything from Tasks 1–9.
- Produces: a worked example proving the annotation loop closes, and `demos/GAPS.md` as the standing list of missing CLI verification commands.

- [ ] **Step 1: Read the current test and its ANNOTATE.md**

```bash
cat demos/TestAccountsApply/ANNOTATE.md
sed -n '40,128p' cli/tests/command_accounts_apply_test.go
```

Expected: `ANNOTATE.md` names the subtests `apply_create`, `apply_update` and `re-apply_leaves_non-secret_upstream_state_unchanged`, each with its derived prose.

- [ ] **Step 2: Add narration to the test**

In `cli/tests/command_accounts_apply_test.go`, inside each `t.Run` body, before the first `executor.Execute`, add a `demo.Say` that supplies the reason the derived prose cannot:

```go
	t.Run("apply create", func(t *testing.T) {
		demo.Say(t, "Accounts are created from spec with no ids anywhere — everything is by reference.")
		// … existing body …
	})

	t.Run("apply update", func(t *testing.T) {
		demo.Say(t, "The same specs with changed values update in place rather than recreating.")
		// … existing body …
	})

	t.Run("re-apply leaves non-secret upstream state unchanged", func(t *testing.T) {
		demo.Say(t, "Re-applying an unchanged project is a no-op upstream, except for write-only secrets, which the API never returns so the CLI must always re-send them.")
		// … existing body …
	})
```

Add the import:

```go
	"github.com/rudderlabs/rudder-iac/cli/tests/demo"
```

- [ ] **Step 3: Verify the test still passes with journaling off**

```bash
export GVM_ROOT="$HOME/.gvm"
go test ./cli/tests -run '^TestAccountsApply$' -v 2>&1 | tail -10
```

Expected: PASS, unchanged from before the narration was added. This confirms `demo.Say` is inert when unrecorded.

- [ ] **Step 4: Re-record and regenerate**

```bash
make demo DEMO_TEST=TestAccountsApply DEMO_PROFILE=mini
```

Expected: `demos/TestAccountsApply/ANNOTATE.md` is now **gone**, and `manifest.json` reports `"annotated": true` with every step's `annotated` field true.

- [ ] **Step 5: Confirm the annotated demo no longer asks and no longer exits 3**

```bash
cd demos/TestAccountsApply && ./demo.sh; echo "exit=$?"
```

Expected: the narration appears in the output as `# Accounts are created from spec…`, and the exit code is `0`.

- [ ] **Step 6: Add GAPS.md generation to `demogen`**

In `main.go`'s `run()`, after the `for test, group := range …` loop, collect steps whose verification is invisible across every demo and write one file:

```go
	if err := writeGaps(*outDir, allSteps); err != nil {
		return err
	}
```

Accumulate `allSteps` inside the loop (`allSteps = append(allSteps, m.Steps...)`, declared as `var allSteps []StepInfo` before it), and add to `emit.go`:

```go
// writeGaps records every step whose proof is invisible on screen. Where no
// read-only CLI command exists to show a result, that absence is a defect in
// the CLI, not in the demo — DEX-921 states the rule: "A demo that has to leave
// the tool it is demonstrating is a gap in the tool." This file is the list.
func writeGaps(outDir string, steps []StepInfo) error {
	var missing []StepInfo
	for _, s := range steps {
		if s.Verification == "none" {
			missing = append(missing, s)
		}
	}

	path := filepath.Join(outDir, "GAPS.md")
	if len(missing) == 0 {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("removing stale GAPS.md: %w", err)
		}

		return nil
	}

	var b strings.Builder
	b.WriteString("# Verification gaps\n\n")
	b.WriteString("These steps prove their result in Go, where a viewer cannot see it.\n")
	b.WriteString("Each one wants a read-only CLI command that shows the same thing on screen.\n")
	b.WriteString("Where no such command exists, that is a ticket against the CLI.\n\n")

	for _, s := range missing {
		b.WriteString("- `" + s.Test + "` — " + s.Prose + "\n")
	}

	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return fmt.Errorf("writing GAPS.md: %w", err)
	}

	return nil
}
```

- [ ] **Step 7: Regenerate and read the gaps**

```bash
make demo-generate DEMO_TEST=TestAccountsApply
cat demos/GAPS.md
```

Expected: the file lists the `apply create` and `apply update` steps, whose last command is a write. That is the correct answer — an apply proves itself only through a read-back the CLI cannot yet do for accounts, which is exactly the DEX-920 class of missing lister.

- [ ] **Step 8: Document the workflow**

Append to `cli/tests/README.md`:

```markdown
## Demos

Every test in this suite can be turned into a narrated, reproducible terminal
demo from a real run. Nothing is hand-written and nothing is simulated: the
demo replays the commands the test actually executed.

```bash
make demo DEMO_TEST=TestAccountsApply DEMO_PROFILE=mini   # record + generate
make demo-cast DEMO_TEST=TestAccountsApply                # record a .cast
make demo-check DEMO_TEST=TestAccountsApply               # fail if it drifted
```

The result is `demos/<Test>/` — a `demo.sh` you can run by hand, the spec
fixtures it needs, and a `manifest.json` naming the branch, commit and backend
it was recorded against.

**Narration.** Step prose is derived from subtest names, so every test is
demoable with no work. Where the *reason* matters and a subtest name cannot
carry it, add one line in the test:

```go
demo.Say(t, "The API never returns write-only secrets, so the CLI re-sends them every time.")
```

It writes into the demo journal and is inert when no demo is being recorded. A
demo with no narration writes an `ANNOTATE.md` naming the steps that want it,
and exits 3 when run non-interactively, so an agent is told to add the lines
and open a PR rather than leaving the gap unnoticed.

**Verification.** A step whose last command writes rather than reads proves
itself in Go, invisibly. Those steps are listed in `demos/GAPS.md`; each one
wants a read-only CLI command that shows the result on screen. Where no such
command exists, file it — a demo that has to leave the tool it demonstrates is
a gap in the tool, not in the demo.

**Backends.** A profile is an env file under `demos/profiles/`. The CLI already
selects its backend from `RUDDERSTACK_API_URL` and `RUDDERSTACK_ACCESS_TOKEN`,
so a demo runs against rudder-mini locally and a cloud backend in CI with no
code change. The manifest records which.
```

- [ ] **Step 9: Generate all six demos and verify the day-one claim**

The spec's scope says every existing e2e test is demoable the day this lands.
That is an assertion until it is run. Generate each one and record what came
out:

```bash
export GVM_ROOT="$HOME/.gvm"
for T in TestProjectApply TestDestinationsApply TestAccountsApply \
         TestAccountsImportWorkspace TestConnectionsApply TestTransformationsTest; do
  echo "=== $T ==="
  make demo DEMO_TEST=$T DEMO_PROFILE=mini || echo "  FAILED: $T"
done
ls -d demos/Test*
```

Expected: six directories, each with `demo.sh`, `manifest.json` and `README.md`.
Five will also have `ANNOTATE.md` — only `TestAccountsApply` was annotated in
Step 2, and that is the correct state, not a defect.

Two failures are expected and are information, not blockers:

- `TestConnectionsApply` is gated on `RUN_CONNECTION_E2E`, which is passed by
  the workflow but never defined as a repository variable (DEX-901). If it
  skips, its demo is empty — record that in `demos/GAPS.md` by hand and move
  on.
- `TestDestinationsApply` is long and touches many destinations. If it exceeds
  the 20m timeout against mini, note it and generate the other five.

Report which of the six produced a demo. Do not claim all six work unless all
six ran.

- [ ] **Step 10: Run everything one last time**

```bash
export GVM_ROOT="$HOME/.gvm"
make lint
make test
make demo-check DEMO_TEST=TestAccountsApply
```

Expected: lint clean, unit suite green (including `cli/tests/demo` and `cli/tests/demo/cmd/demogen`), `no drift`.

- [ ] **Step 11: Commit**

```bash
git add cli/tests/command_accounts_apply_test.go cli/tests/README.md cli/tests/demo/cmd/demogen demos
git commit -m "test(demo): annotate TestAccountsApply and record verification gaps

Closes the loop end to end: the demo asked for narration, the test supplies it,
ANNOTATE.md disappears and the exit code returns to zero. GAPS.md now lists
every step whose proof is invisible on screen — the standing input to the
missing read-only commands.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

## Sequencing and parallelism

Tasks 1 → 2 are strictly ordered (2 imports 1's API and renames `collectEnv`).

Tasks 3, 4 and 5 are independent of each other and depend only on Task 1's
`demo.Record`. They can be executed in parallel by three workers.

Task 6 needs 3, 4 and 5. Task 7 needs 6. Tasks 8 and 9 both need 7 and are
independent of each other. Task 10 needs 9.

```
1 ──► 2
 └──► 3 ─┐
     4 ─┼─► 6 ──► 7 ─┬─► 8
     5 ─┘            └─► 9 ──► 10
```

Tasks 7–10 each need a reachable backend (rudder-mini is enough). Tasks 1–6 are
entirely hermetic and need nothing running.
