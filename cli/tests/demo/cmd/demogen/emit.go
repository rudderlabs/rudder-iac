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
	Test            string      `json:"test"`
	RecordedAt      time.Time   `json:"recordedAt"`
	Git             GitInfo     `json:"git"`
	Backend         BackendInfo `json:"backend"`
	CLIVersion      string      `json:"cliVersion"`
	Flags           []string    `json:"flags,omitempty"`
	Annotated       bool        `json:"annotated"`
	DurationSeconds float64     `json:"durationSeconds"`
	Steps           []StepInfo  `json:"steps"`
	Gaps            []string    `json:"gaps,omitempty"`
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
