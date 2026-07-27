package format

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/cmderrors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

func runCmd(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := NewCmdFmt()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

func TestFmt_WriteMode_FormatsInPlace(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "a.yaml", "metadata:\n    name: a\n")

	out, err := runCmd(t, path)
	require.NoError(t, err)

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "metadata:\n  name: a\n", string(got))
	assert.Contains(t, out, "a.yaml")
}

func TestFmt_CheckMode_ExitsNonZeroWhenUnformatted(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "a.yaml", "metadata:\n    name: a\n")

	out, err := runCmd(t, "--check", path)
	require.Error(t, err)

	var silent *cmderrors.SilentError
	assert.True(t, errors.As(err, &silent), "check failure must be a SilentError for clean exit")

	// File is untouched in check mode.
	got, _ := os.ReadFile(path)
	assert.Equal(t, "metadata:\n    name: a\n", string(got))
	assert.Contains(t, out, "a.yaml")
}

func TestFmt_CheckMode_ZeroWhenClean(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.yaml", "metadata:\n  name: a\n")

	_, err := runCmd(t, "--check", dir)
	assert.NoError(t, err)
}

func TestFmt_DiffMode_PrintsUnifiedDiffWithoutWriting(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "a.yaml", "metadata:\n    name: a\n")

	out, err := runCmd(t, "--diff", path)
	require.NoError(t, err)

	assert.Contains(t, out, "---")
	assert.Contains(t, out, "+++")
	assert.Contains(t, out, "name: a")

	got, _ := os.ReadFile(path)
	assert.Equal(t, "metadata:\n    name: a\n", string(got))
}

func TestFmt_InvalidYAML_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "bad.yaml", "key: : : bad\n")

	_, err := runCmd(t, dir)
	assert.Error(t, err)
}

func TestFmt_CheckAndDiffMutuallyExclusive(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.yaml", "metadata:\n  name: a\n")

	_, err := runCmd(t, "--check", "--diff", dir)
	assert.Error(t, err)
}
