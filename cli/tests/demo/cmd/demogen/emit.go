package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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
	Test            string      `json:"test"`
	RecordedAt      time.Time   `json:"recordedAt"`
	Git             GitInfo     `json:"git"`
	Backend         BackendInfo `json:"backend"`
	CLIVersion      string      `json:"cliVersion"`
	Flags           []string    `json:"flags,omitempty"`
	Annotated       bool        `json:"annotated"`
	DurationSeconds float64     `json:"durationSeconds"`
	Steps           []StepInfo  `json:"steps"`
	// PortabilityGaps are absolute paths Rewriter.Argv could not make
	// portable. Distinct from a GAPS.md gap (a missing read-only CLI
	// command) — same word, different meaning, so it gets its own name here.
	PortabilityGaps []string `json:"portabilityGaps,omitempty"`
}

// Emit writes a complete demo directory: the script, its fixtures, the
// manifest, a README, and — when narration is missing — the request for it.
func Emit(dir string, steps []Step, rw Rewriter, m Manifest) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", dir, err)
	}

	script, infos, gaps, fixtures := build(steps, rw, m)

	m.Steps = infos
	m.PortabilityGaps = gaps
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

	// Flags are emitted before profile.env is sourced, and conditionally, so
	// profile.env — whatever it contains, however it's written — always has
	// the last word on the values that matter to it. Placing this after the
	// source (as a plain `export`, the previous behaviour) would silently
	// override whatever backend a profile selected; see IMPORTANT 2 in the
	// review this fixes.
	for _, f := range m.Flags {
		b.WriteString(flagAssignment(f) + "\n")
	}

	b.WriteString(`[ -f ./profile.env ] && . ./profile.env` + "\n")
	b.WriteString(productionGuard())

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

			safe := make([]string, len(argv))
			for i, a := range argv {
				safe[i] = shellQuote(a)
			}

			b.WriteString("pe " + quote(strings.Join(safe, " ")) + "\n")
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

// prodAPIHost mirrors api/client/client.go's BASE_URL — the host the CLI
// silently falls back to (with whatever real token ~/.rudder/config.json
// holds) whenever RUDDERSTACK_API_URL is unset. It is also what the Makefile's
// demo-record guard and scripts/demo-cast.sh already refuse against (R17);
// this is the third place that needs to, since a generated demo.sh is the
// one a human is actually told to run.
const prodAPIHost = "api.rudderstack.com"

// productionGuard refuses to run the rest of the script when
// RUDDERSTACK_API_URL is unset or points at production. It must be emitted
// after both the flags block and the profile.env source, so it sees the
// value the run will actually use, not a value about to be overridden.
//
// This is not redundant with the Makefile/demo-cast guards: recording with
// credentials taken from ~/.rudder/config.json — the method
// demos/profiles/mini.env itself blesses — never puts RUDDERSTACK_API_URL
// into os.Environ(), so flagsOf captures nothing for it and no flag line
// above sets it. Without this, a demo.sh recorded that way opens with
// `destroy --confirm=false` against production using a real token.
func productionGuard() string {
	return `
if [ -z "${RUDDERSTACK_API_URL:-}" ]; then
  echo "refusing: RUDDERSTACK_API_URL is unset, so this script would target ` + prodAPIHost + `." >&2
  echo "  The first command below is 'destroy --confirm=false' — it wipes the target workspace." >&2
  echo "  Source a profile first, e.g.: . ./profile.env" >&2
  exit 1
fi
case "$RUDDERSTACK_API_URL" in
*` + prodAPIHost + `*)
  echo "refusing: RUDDERSTACK_API_URL points at production, and this script wipes its workspace." >&2
  echo "  The first command below is 'destroy --confirm=false'." >&2
  exit 1
  ;;
esac
`
}

// flagAssignment turns one recorded "KEY=VALUE" flag into a conditional
// shell assignment — `: "${KEY:=VALUE}"` only takes effect when KEY is not
// already set. Emitted before profile.env is sourced, this makes the flag a
// default a profile (or the invoking shell) can always override, rather than
// a value that clobbers whatever they chose.
//
// The value is escaped for the double-quoted context it sits in, the same
// character set quote() escapes for the same reason (R15): unescaped, a
// value carrying $(…) would execute at script load, before any command runs
// and before demo-magic's eval-based playback is even reached.
func flagAssignment(f string) string {
	key, value, _ := strings.Cut(f, "=")

	return `: "${` + key + `:=` + doubleQuoteEscape(value) + `}"`
}

// doubleQuoteEscape escapes a string for embedding inside a double-quoted
// shell context, without adding the surrounding quotes itself — the inverse
// of what quote() returns, needed here because flagAssignment nests the
// value inside "${KEY:=...}" rather than emitting a standalone quoted word.
func doubleQuoteEscape(s string) string {
	return strings.NewReplacer(`\`, `\\`, `"`, `\"`, "$", `\$`, "`", "\\`").Replace(s)
}

// quote wraps a line for the shell. demo-magic takes a single argument, and the
// recorded commands contain flags and JSON filters that must not be re-split.
func quote(s string) string {
	return `"` + doubleQuoteEscape(s) + `"`
}

// shellSafe is the set of characters a token may carry and still be left
// bare. It is a strict allowlist, never a denylist: a character not listed
// here gets quoted, so an unanticipated metacharacter fails closed.
var shellSafe = regexp.MustCompile(`^[A-Za-z0-9_@%+=:,./-]+$`)

// shellQuote makes one recorded argv token safe for demo-magic's playback,
// which runs `eval $@` on the line (see run_cmd in demo-magic.sh) — so a
// token carrying $(…), a backtick or a pipe would otherwise execute or
// re-parse rather than being passed through as a single argument. quote()
// alone does not protect against this: it escapes the joined line for the
// outer double-quoted string literal in demo.sh, but demo-magic's eval acts
// on the *unquoted* result of that string being word-split again at runtime.
//
// Tokens that need no quoting are left bare: these scripts are written to be
// read, and quoting every token (à la shlex.quote's conservative default)
// would bury the command in punctuation for no safety benefit.
func shellQuote(token string) string {
	if token != "" && shellSafe.MatchString(token) {
		return token
	}

	// A single quote cannot appear inside single quotes, so close, escape, reopen.
	return "'" + strings.ReplaceAll(token, "'", `'\''`) + "'"
}

func anyAnnotated(infos []StepInfo) bool {
	for _, i := range infos {
		if i.Annotated {
			return true
		}
	}

	return false
}

// copyFixtures copies the testdata each step reads next to the demo, flat:
// cli/tests/testdata/<rest> lands at <dir>/<rest>. Ruling R3 — there is no
// shared "project" directory to reintroduce here; fixture trees like
// destinations/... have no such segment at all, and testdata/project/create
// already carries its own leading "project" segment.
//
// Ruling R14: the demo directory can hold files this generator does not own
// (profile.env, notably), so regenerating must not blanket-remove it. Instead,
// clear only the top-level segments the fixtures about to be copied will
// write into, so a fixture set that shrank does not leave orphaned files in a
// committed artifact.
func copyFixtures(dir, repoRoot string, fixtures []string) error {
	for _, root := range fixtureRoots(fixtures) {
		if err := os.RemoveAll(filepath.Join(dir, root)); err != nil {
			return fmt.Errorf("clearing stale fixture root %s: %w", root, err)
		}
	}

	for _, rel := range fixtures {
		var (
			src = filepath.Join(repoRoot, rel)
			dst = filepath.Join(dir, strings.TrimPrefix(rel, testdataRel+"/"))
		)

		if _, err := os.Stat(src); err != nil {
			// A fixture the recording referenced but this checkout lacks is
			// skipped rather than abandoning the demo — it is not recorded
			// as a portability gap (it never reaches m.PortabilityGaps or the
			// README), so a demo missing part of its fixture set here still
			// reports as fully portable. That's a known blind spot, not
			// silent-by-design: revisit if it ever bites.
			continue
		}

		if err := CopyTree(src, dst); err != nil {
			return fmt.Errorf("copying fixture %s: %w", rel, err)
		}
	}

	return nil
}

// fixtureRoots returns the distinct first path segments the fixture
// destinations will land under, sorted for a deterministic RemoveAll order.
func fixtureRoots(fixtures []string) []string {
	seen := map[string]bool{}
	var out []string

	for _, rel := range fixtures {
		suffix := strings.TrimPrefix(rel, testdataRel+"/")
		root := suffix
		if i := strings.Index(suffix, "/"); i >= 0 {
			root = suffix[:i]
		}
		if root == "" || seen[root] {
			continue
		}
		seen[root] = true
		out = append(out, root)
	}

	sort.Strings(out)

	return out
}

// writeGaps records every step, across every demo under outDir, whose proof
// is invisible on screen. Where no read-only CLI command exists to show a
// result, that absence is a defect in the CLI, not in the demo — DEX-921
// states the rule: "A demo that has to leave the tool it is demonstrating is
// a gap in the tool." This file is the list.
//
// Ruling R24: demogen processes one test per invocation (one -journal,
// one -events), so building this from only the steps this run just emitted
// would make generating any one demo silently erase every other demo's
// gaps — the opposite of "standing list." Instead this reads every
// demos/<Test>/manifest.json already on disk, including the one(s) this
// invocation just wrote, and unions their gaps.
func writeGaps(outDir string) error {
	manifests, err := filepath.Glob(filepath.Join(outDir, "*", "manifest.json"))
	if err != nil {
		return fmt.Errorf("listing manifests: %w", err)
	}

	byTest := map[string][]StepInfo{}
	for _, path := range manifests {
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading %s: %w", path, err)
		}

		var m Manifest
		if err := json.Unmarshal(data, &m); err != nil {
			return fmt.Errorf("decoding %s: %w", path, err)
		}

		for _, s := range m.Steps {
			if s.Verification == "none" {
				byTest[m.Test] = append(byTest[m.Test], s)
			}
		}
	}

	path := filepath.Join(outDir, "GAPS.md")
	if len(byTest) == 0 {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("removing stale GAPS.md: %w", err)
		}

		return nil
	}

	// Ruling R5 / R24: byTest is a map, so Go randomises its iteration order —
	// sort the test names, and each test's own steps, so regeneration does
	// not churn the diff for no reason.
	tests := make([]string, 0, len(byTest))
	for test := range byTest {
		tests = append(tests, test)
	}
	sort.Strings(tests)

	var b strings.Builder
	b.WriteString("# Verification gaps\n\n")
	b.WriteString("These steps prove their result in Go, where a viewer cannot see it.\n")
	b.WriteString("Each one wants a read-only CLI command that shows the same thing on screen.\n")
	b.WriteString("Where no such command exists, that is a ticket against the CLI.\n\n")

	for _, test := range tests {
		steps := byTest[test]
		sort.Slice(steps, func(i, j int) bool { return steps[i].Test < steps[j].Test })

		b.WriteString("## " + test + "\n\n")
		for _, s := range steps {
			b.WriteString("- `" + s.Test + "` — " + s.Prose + "\n")
		}
		b.WriteString("\n")
	}

	if err := os.WriteFile(path, []byte(strings.TrimRight(b.String(), "\n")+"\n"), 0o644); err != nil {
		return fmt.Errorf("writing GAPS.md: %w", err)
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
	b.WriteString("Generated from a real run of `" + m.Test + "`.\n\n")

	if len(m.PortabilityGaps) > 0 {
		// -temp-root has no caller (neither the Makefile nor demo-check.sh
		// passes it), so a step that ran under t.TempDir() leaves an absolute
		// path here that exists only on the machine that recorded it. Saying
		// "run it yourself" over that path would be a lie — README must not
		// invite a run this script cannot actually make.
		b.WriteString("**Not runnable as recorded.** It references absolute paths from the machine\n")
		b.WriteString("it was recorded on; see \"Not portable yet\" below. Re-record it (`make demo`)\n")
		b.WriteString("to fix this.\n\n")
	} else {
		b.WriteString("Run it yourself:\n\n")
		b.WriteString("```bash\n./demo.sh\n```\n\n")
	}

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

	if len(m.PortabilityGaps) > 0 {
		b.WriteString("\n## Not portable yet\n\nThese absolute paths could not be rewritten:\n\n")
		for _, g := range m.PortabilityGaps {
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
