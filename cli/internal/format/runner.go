package format

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	udiff "github.com/aymanbagabas/go-udiff"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/loader"
)

// Options controls how Run applies formatting.
type Options struct {
	// Check reports which files would change and does not write; the caller
	// typically exits non-zero when any Result is Changed.
	Check bool
	// Diff populates Result.Diff with a unified diff and does not write.
	Diff bool
}

// Result is the outcome of formatting a single file.
type Result struct {
	Path    string
	Changed bool
	// Diff is a unified diff of the change, populated only in diff mode.
	Diff string
	// Err records a per-file failure (e.g. invalid YAML) so one bad file does
	// not abort the whole run.
	Err error
}

// Run discovers spec YAML files under the given paths (files or directories;
// defaults to the current directory when empty) and formats each one according
// to opts. In the default mode it rewrites changed files in place; Check and
// Diff modes never write.
//
// Discovery errors (a missing path) abort the run; per-file formatting errors
// are recorded on the corresponding Result instead.
func Run(paths []string, opts Options) ([]Result, error) {
	if len(paths) == 0 {
		paths = []string{"."}
	}

	files, err := discover(paths)
	if err != nil {
		return nil, err
	}

	results := make([]Result, 0, len(files))
	for _, path := range files {
		results = append(results, formatFile(path, opts))
	}
	return results, nil
}

func formatFile(path string, opts Options) Result {
	res := Result{Path: path}

	original, err := os.ReadFile(path)
	if err != nil {
		res.Err = fmt.Errorf("reading %s: %w", path, err)
		return res
	}

	formatted, err := Source(original)
	if err != nil {
		res.Err = fmt.Errorf("formatting %s: %w", path, err)
		return res
	}

	res.Changed = !bytes.Equal(original, formatted)
	if !res.Changed {
		return res
	}

	switch {
	case opts.Diff:
		res.Diff = udiff.Unified(path, path, string(original), string(formatted))
	case opts.Check:
		// report only
	default:
		if err := os.WriteFile(path, formatted, filePerm(path)); err != nil {
			res.Err = fmt.Errorf("writing %s: %w", path, err)
		}
	}
	return res
}

// discover walks the given paths and returns the sorted, deduplicated set of
// spec YAML files (.yaml/.yml, including .vars.yaml var files). A path that is
// itself a YAML file is included directly; a directory is walked recursively.
func discover(paths []string) ([]string, error) {
	seen := make(map[string]struct{})

	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			return nil, fmt.Errorf("accessing path %s: %w", p, err)
		}

		if !info.IsDir() {
			if isYAML(p) {
				seen[p] = struct{}{}
			}
			continue
		}

		err = filepath.WalkDir(p, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return fmt.Errorf("walking %s: %w", path, err)
			}
			if d.IsDir() || !isYAML(path) {
				return nil
			}
			seen[path] = struct{}{}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	files := make([]string, 0, len(seen))
	for f := range seen {
		files = append(files, f)
	}
	sort.Strings(files)
	return files, nil
}

// isYAML reports whether path is a spec or var YAML file. Var files
// (*.vars.yaml) are included: fmt normalizes any YAML, unlike the spec loader
// which skips them.
func isYAML(path string) bool {
	ext := filepath.Ext(path)
	return ext == loader.ExtensionYAML || ext == loader.ExtensionYML
}

// filePerm preserves the existing file's permission bits, falling back to a
// sane default if it cannot be stat'd.
func filePerm(path string) os.FileMode {
	if info, err := os.Stat(path); err == nil {
		return info.Mode().Perm()
	}
	return 0o644
}
