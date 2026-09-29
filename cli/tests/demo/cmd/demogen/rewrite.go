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

	// workDir is where a generated demo keeps the t.TempDir() root's contents,
	// relative to the demo's own directory.
	workDir = "work"
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
			// Flat mapping: cli/tests/testdata/<rest> -> <rest>, relative to
			// the demo directory. Fixture trees don't all share a "project"
			// segment (see TestDestinationsApply's destinations/... tree), so
			// there is no common prefix to reintroduce here.
			out = append(out, rw.testdataSuffix(a))
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
//
// Both prefix checks include a trailing path separator, so a bare "testdata"
// (or the repo's testdata directory with no trailing element) does not match:
// there is nothing after it to become the rewritten path, and matching it
// would hand testdataSuffix an empty string.
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
