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

// TestArgvOnlyRewritesBinaryAtPositionZero pins that the binary rewrite
// applies to the command being executed (argv[0]) only. If BinPath happens to
// reappear later in argv — e.g. an argument that itself names the binary
// path — that occurrence is an ordinary unrewritten absolute path, not the
// command, so it must still surface as a gap.
func TestArgvOnlyRewritesBinaryAtPositionZero(t *testing.T) {
	rw := testRewriter()

	got, gaps := rw.Argv([]string{rw.BinPath, "diff", "--against", rw.BinPath})

	assert.Equal(t, []string{"rudder-cli", "diff", "--against", rw.BinPath}, got)
	assert.Equal(t, []string{rw.BinPath}, gaps)
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

// TestCopyTreeCopiesEmptyDirectories pins that CopyTree recreates every
// directory it walks, not just the parents of files it happens to copy.
// copyFile also calls os.MkdirAll on a file's parent, which would silently
// paper over a WalkDir that skipped directory nodes entirely — so this needs
// a nested directory with nothing inside it to catch that gap.
func TestCopyTreeCopiesEmptyDirectories(t *testing.T) {
	var (
		src = t.TempDir()
		dst = filepath.Join(t.TempDir(), "out")
	)
	require.NoError(t, os.MkdirAll(filepath.Join(src, "empty"), 0o755))

	require.NoError(t, CopyTree(src, dst))

	info, err := os.Stat(filepath.Join(dst, "empty"))
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

// TestArgvDoesNotMatchBareTestdataSegment pins the boundary the prefix checks
// rely on: "testdata" (relative) and the repo's testdata dir with no trailing
// element must NOT match isTestdata, since matching would strip the whole
// element and hand testdataSuffix an empty string (an empty/"." demo path).
// A bare "testdata" argument is passed through untouched (it isn't absolute,
// so it isn't reported as a gap either).
func TestArgvDoesNotMatchBareTestdataSegment(t *testing.T) {
	got, gaps := testRewriter().Argv([]string{
		"rudder-cli", "apply",
		"-l", "testdata",
		"-l", "/home/dev/rudder-iac/cli/tests/testdata",
	})

	assert.Equal(t, []string{
		"rudder-cli", "apply",
		"-l", "testdata",
		"-l", "/home/dev/rudder-iac/cli/tests/testdata",
	}, got)
	assert.Equal(t, []string{"/home/dev/rudder-iac/cli/tests/testdata"}, gaps)
}
