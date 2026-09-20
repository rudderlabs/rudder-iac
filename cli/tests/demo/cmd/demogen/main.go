// Command demogen turns a recorded e2e run into demo directories.
//
//	demogen -journal journal.jsonl -events events.json -out demos \
//	        -repo-root . -bin /tmp/rudder-cli-bin-x/rudder-cli \
//	        -profile mini -backend-kind local -api-url http://localhost:15580
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
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

// durationOf spans the earliest Start to the latest effective end across all
// records. demo.KindSay records carry no End (they are narration, not a
// command with a duration), so a zero End is treated as ending where it
// started rather than as the year-1 zero time — otherwise a single narration
// record with no exec after it would make the span run backwards.
func durationOf(steps []Step) float64 {
	var first, last time.Time

	for _, s := range steps {
		for _, r := range s.Records {
			if first.IsZero() || r.Start.Before(first) {
				first = r.Start
			}

			end := r.End
			if end.IsZero() {
				end = r.Start
			}
			if end.After(last) {
				last = end
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
