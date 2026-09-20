package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

// The emitFixture narration text ("And the state the apply produced:") is
// already capitalized with no underscores, so DeriveProse happens to be a
// no-op on it — a mutation that ran narration through DeriveProse instead of
// emitting it verbatim would slip past TestEmitScriptContainsDerivedProseAndPortableArgv
// undetected. Pick narration DeriveProse actually changes (a "should_" prefix
// and underscores, both of which DeriveProse strips/rewrites) to close that gap.
func TestEmitEmitsSayNarrationVerbatimNotDerived(t *testing.T) {
	dir := t.TempDir()
	steps := []Step{{
		Test: "TestX/step",
		Records: []demo.Record{
			{Kind: demo.KindSay, Start: at(1), Text: "should_show_the_raw_json_output"},
			{Kind: demo.KindExec, Start: at(2), End: at(3), Argv: []string{"rudder-cli", "validate", "-l", "project"}},
		},
	}}

	require.NoError(t, Emit(dir, steps, Rewriter{RepoRoot: t.TempDir()}, Manifest{Test: "TestX"}))

	data, err := os.ReadFile(filepath.Join(dir, "demo.sh"))
	require.NoError(t, err)

	assert.Contains(t, string(data), `p "# should_show_the_raw_json_output"`,
		"narration must be emitted verbatim; DeriveProse would turn this into \"Show the raw json output\"")
}

// Ruling R14: regenerating a demo whose fixture set shrank must not leave
// orphaned files in what is a committed artifact, but the demo directory can
// hold files this generator does not own (demo.sh sources ./profile.env, the
// operator's backend config). Both directions matter, so one test proves both.
func TestEmitRegenerationClearsStaleFixturesButKeepsOperatorFiles(t *testing.T) {
	dir := t.TempDir()
	repoRoot := t.TempDir()

	require.NoError(t, os.MkdirAll(filepath.Join(repoRoot, testdataRel, "project", "create"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(repoRoot, testdataRel, "project", "create", "spec.yaml"), []byte("kind: project\n"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(repoRoot, testdataRel, "project", "update"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(repoRoot, testdataRel, "project", "update", "spec.yaml"), []byte("kind: project\nupdated: true\n"), 0o644))

	rw := Rewriter{RepoRoot: repoRoot, BinPath: "/tmp/bin/rudder-cli"}

	firstSteps := []Step{{
		Test: "TestX/first",
		Records: []demo.Record{
			{Kind: demo.KindExec, Start: at(1), End: at(2), Argv: []string{"/tmp/bin/rudder-cli", "apply", "-l", "testdata/project/create", "--confirm=false"}},
		},
	}}
	require.NoError(t, Emit(dir, firstSteps, rw, Manifest{Test: "TestX"}))

	staleFixturePath := filepath.Join(dir, "project", "create", "spec.yaml")
	_, err := os.Stat(staleFixturePath)
	require.NoError(t, err, "the first run's fixture must be copied")

	// A file the generator does not own: demo.sh sources it if present.
	profilePath := filepath.Join(dir, "profile.env")
	require.NoError(t, os.WriteFile(profilePath, []byte("export RUDDERSTACK_API_URL=http://localhost:15580\n"), 0o644))

	// The second run's fixture set shrank: it still touches the "project" root,
	// but a different subdirectory of it — create/ is no longer referenced.
	secondSteps := []Step{{
		Test: "TestX/second",
		Records: []demo.Record{
			{Kind: demo.KindExec, Start: at(3), End: at(4), Argv: []string{"/tmp/bin/rudder-cli", "apply", "-l", "testdata/project/update", "--confirm=false"}},
		},
	}}
	require.NoError(t, Emit(dir, secondSteps, rw, Manifest{Test: "TestX"}))

	_, err = os.Stat(staleFixturePath)
	assert.True(t, os.IsNotExist(err), "a fixture no longer referenced must not survive regeneration")

	_, err = os.Stat(filepath.Join(dir, "project", "update", "spec.yaml"))
	assert.NoError(t, err, "the second run's fixture must still be copied")

	_, err = os.Stat(profilePath)
	assert.NoError(t, err, "the generator must not delete files it does not own")
}

// quote() is the closest thing to a security-relevant path in this package: a
// generated script runs on a developer's machine, and recorded argv can carry
// a jq filter with embedded double quotes, like a hand-written demo already
// does: `.data[] | select(.externalId=="vip")`.
func TestQuoteRoundTripsShellMetacharactersWithoutExecutingThem(t *testing.T) {
	cases := map[string]string{
		"jq filter with embedded double quotes": `.data[] | select(.externalId=="vip")`,
		"backtick command substitution":         "echo `id`",
		"dollar-paren command substitution":     "echo $(id)",
		"bare dollar variable reference":        "price is $HOME today",
		"trailing backslash":                    `C:\path\`,
		"attempt to close the quote and inject": `foo"; echo INJECTED; echo "bar`,
	}

	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			out, err := exec.Command("bash", "-c", "printf '%s' "+quote(input)).CombinedOutput()
			require.NoError(t, err)
			assert.Equal(t, input, string(out), "quoting must round-trip through the shell verbatim, proving nothing inside it executed")
		})
	}
}

func TestFlagsOfDropsRedactedValues(t *testing.T) {
	steps := []Step{{
		Test: "TestX/step",
		Records: []demo.Record{
			{Kind: demo.KindExec, Env: map[string]string{
				"RUDDERSTACK_ACCESS_TOKEN":         demo.Redacted,
				"RUDDERSTACK_X_RETL_TABLE_SUPPORT": "true",
			}},
		},
	}}

	got := flagsOf(steps)

	assert.Equal(t, []string{"RUDDERSTACK_X_RETL_TABLE_SUPPORT=true"}, got)
	assert.NotContains(t, got, "RUDDERSTACK_ACCESS_TOKEN="+demo.Redacted,
		"a demo must never instruct a reader to export a redacted literal")
}

func TestGroupByTopLevelSplitsByTestFunction(t *testing.T) {
	steps := []Step{
		{Test: "TestA/sub_one"},
		{Test: "TestB/sub_one"},
		{Test: "TestA/sub_two"},
		{Test: "TestC"},
	}

	got := groupByTopLevel(steps)

	assert.Equal(t, map[string][]Step{
		"TestA": {{Test: "TestA/sub_one"}, {Test: "TestA/sub_two"}},
		"TestB": {{Test: "TestB/sub_one"}},
		"TestC": {{Test: "TestC"}},
	}, got)
}

func TestDurationOfIsZeroForNoRecords(t *testing.T) {
	assert.Equal(t, 0.0, durationOf(nil))
	assert.Equal(t, 0.0, durationOf([]Step{{Test: "TestX/step"}}))
}

// demo.KindSay records carry no End — they are narration, not a timed
// command — so a single such record must not make the span run backwards to
// the year-1 zero time.
func TestDurationOfHandlesZeroEndWithoutGoingNegative(t *testing.T) {
	steps := []Step{{
		Test: "TestX/step",
		Records: []demo.Record{
			{Kind: demo.KindSay, Start: at(5), Text: "narrated"},
		},
	}}

	got := durationOf(steps)

	assert.GreaterOrEqual(t, got, 0.0, "a single record with a zero End must not yield a negative duration")
	assert.Less(t, got, 60.0, "must not report an absurd multi-decade duration measured from the zero time")
}

// Ruling R15: quote() alone protects the outer double-quoted string literal
// in demo.sh, but demo-magic's run_cmd runs `eval $@` on the *result* of that
// string being word-split again at playback time (see
// /Users/shanmukh/workspace/demo-magic/demo-magic.sh). A token carrying
// $(...), a backtick, or a bare pipe survives quote()'s escaping and then
// executes or re-parses during that eval. These tests exercise the real
// playback path end to end — through Emit's actual demo.sh output, through a
// faithful reproduction of pe -> run_cmd -> eval — rather than only the
// outer-embedding boundary TestQuoteRoundTripsShellMetacharactersWithoutExecutingThem
// already covers (that test uses printf, which never evaluates its argument;
// it could not have caught this).

// emitPeLine runs one exec record through the real Emit/build pipeline and
// returns the "pe ..." line demo.sh contains for it, so these tests exercise
// production code rather than a hand-rolled mirror of it.
func emitPeLine(t *testing.T, argv []string) string {
	t.Helper()

	dir := t.TempDir()
	steps := []Step{{
		Test: "TestX/step",
		Records: []demo.Record{
			{Kind: demo.KindExec, Start: at(1), End: at(2), Argv: argv},
		},
	}}
	require.NoError(t, Emit(dir, steps, Rewriter{RepoRoot: t.TempDir()}, Manifest{Test: "TestX"}))

	data, err := os.ReadFile(filepath.Join(dir, "demo.sh"))
	require.NoError(t, err)

	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "pe ") {
			return line
		}
	}

	t.Fatal("generated demo.sh has no pe line")
	return ""
}

// evalPeLine runs a generated "pe ..." line through a faithful reproduction
// of demo-magic's playback: pe prints then calls run_cmd, and run_cmd does
// exactly `eval $@` (unquoted) on it. cmdStub, if non-empty, defines the bash
// function argv[0] resolves to.
func evalPeLine(t *testing.T, peLine, cmdStub string) string {
	t.Helper()

	script := "set -u\n" + cmdStub + "\n" +
		"run_cmd() {\n  eval $@\n}\n" +
		"pe() {\n  run_cmd \"$@\"\n}\n" +
		peLine + "\n"

	out, _ := exec.Command("bash", "-c", script).CombinedOutput()

	return string(out)
}

func TestShellQuotePlaybackDoesNotExecuteDollarParenToken(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "pwned")

	peLine := emitPeLine(t, []string{"true", "--filter", "$(touch " + marker + ")"})
	evalPeLine(t, peLine, "")

	_, err := os.Stat(marker)
	assert.True(t, os.IsNotExist(err), "a $(...) token must not execute during demo-magic's eval playback")
}

func TestShellQuotePlaybackDoesNotExecuteBacktickToken(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "pwned")

	peLine := emitPeLine(t, []string{"true", "--filter", "`touch " + marker + "`"})
	evalPeLine(t, peLine, "")

	_, err := os.Stat(marker)
	assert.True(t, os.IsNotExist(err), "a backtick token must not execute during demo-magic's eval playback")
}

func TestShellQuotePlaybackKeepsJQFilterAsOneArgumentWithPipeIntact(t *testing.T) {
	dir := t.TempDir()
	capture := filepath.Join(dir, "args")

	filter := `.data[] | select(.externalId=="vip")`
	peLine := emitPeLine(t, []string{"capture", "events", "list", "--filter", filter})

	stub := "capture() {\n" +
		"  : > " + capture + "\n" +
		"  printf '%s\\n' \"$#\" >> " + capture + "\n" +
		"  for a in \"$@\"; do printf '%s\\x1f' \"$a\" >> " + capture + "; done\n" +
		"}"
	evalPeLine(t, peLine, stub)

	data, err := os.ReadFile(capture)
	require.NoError(t, err)

	nl := strings.IndexByte(string(data), '\n')
	require.GreaterOrEqual(t, nl, 0, "capture file missing arg count line")

	count := string(data)[:nl]
	args := strings.Split(strings.TrimSuffix(string(data)[nl+1:], "\x1f"), "\x1f")

	// "capture" itself becomes the command name, not one of its own
	// arguments, so $# counts the remaining four: events, list, --filter,
	// and the jq filter as one argument.
	assert.Equal(t, "4", count, "argv must arrive as 4 separate arguments, not re-split on the pipe")
	assert.Equal(t, filter, args[len(args)-1], "the jq filter must arrive as a single argument with its pipe intact")
}

func TestShellQuotePlaybackRoundTripsLiteralSingleQuote(t *testing.T) {
	dir := t.TempDir()
	capture := filepath.Join(dir, "args")

	peLine := emitPeLine(t, []string{"capture", "don't", "stop"})

	stub := "capture() {\n" +
		"  : > " + capture + "\n" +
		"  for a in \"$@\"; do printf '%s\\x1f' \"$a\" >> " + capture + "; done\n" +
		"}"
	evalPeLine(t, peLine, stub)

	data, err := os.ReadFile(capture)
	require.NoError(t, err)

	args := strings.Split(strings.TrimSuffix(string(data), "\x1f"), "\x1f")
	assert.Equal(t, []string{"don't", "stop"}, args)
}

// The common case — a plain flag-and-path command — must stay readable: no
// token in it needs quoting, so shellQuote must add none. This pins the
// readability requirement so a later change cannot quietly over-quote every
// token (à la shlex.quote's conservative default).
func TestShellQuoteLeavesOrdinaryArgvBare(t *testing.T) {
	peLine := emitPeLine(t, []string{"rudder-cli", "apply", "-l", "project/create", "--confirm=false"})

	assert.Equal(t, `pe "rudder-cli apply -l project/create --confirm=false"`, peLine)
}

func TestShellQuoteOfEmptyStringIsTwoSingleQuotes(t *testing.T) {
	assert.Equal(t, "''", shellQuote(""))
}
