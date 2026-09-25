package schema

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	schemapkg "github.com/rudderlabs/rudder-iac/cli/internal/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testSpec struct {
	ID string `json:"id"`
}

func testSchemas() schemapkg.Set {
	return schemapkg.Set{
		"beta":  schemapkg.ForKind("beta", testSpec{}),
		"alpha": schemapkg.ForKind("alpha", testSpec{}),
	}
}

func executeCommand(t *testing.T, cmdArgs ...string) (string, error) {
	t.Helper()
	cmd := newCmdSchema(func() (schemapkg.Set, error) { return testSchemas(), nil })
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	cmd.SetArgs(cmdArgs)
	err := cmd.Execute()
	return output.String(), err
}

func TestSchemaListsKindsInOrder(t *testing.T) {
	output, err := executeCommand(t)
	require.NoError(t, err)
	assert.Equal(t, "alpha\nbeta\n", output)
}

func TestSchemaPrintsKnownKind(t *testing.T) {
	output, err := executeCommand(t, "alpha")
	require.NoError(t, err)
	assert.True(t, json.Valid([]byte(output)))
	assert.Contains(t, output, `"const": "alpha"`)
}

func TestSchemaRejectsUnknownKind(t *testing.T) {
	_, err := executeCommand(t, "missing")
	require.ErrorIs(t, err, schemapkg.ErrUnknownKind)
}

func TestSchemaOutCreatesDirectoryAndWritesAllArtifacts(t *testing.T) {
	outDir := filepath.Join(t.TempDir(), "nested", "schemas")
	_, err := executeCommand(t, "--out", outDir)
	require.NoError(t, err)

	for _, name := range []string{
		schemapkg.FileName("alpha"),
		schemapkg.FileName("beta"),
		schemapkg.RootFileName,
	} {
		data, err := os.ReadFile(filepath.Join(outDir, name))
		require.NoError(t, err)
		assert.True(t, json.Valid(data), name)
	}
}

func TestSchemaOutRefusesAnyExistingArtifactBeforeWriting(t *testing.T) {
	outDir := t.TempDir()
	existing := filepath.Join(outDir, schemapkg.RootFileName)
	require.NoError(t, os.WriteFile(existing, []byte("keep me"), 0o644))

	_, err := executeCommand(t, "--out", outDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")

	data, readErr := os.ReadFile(existing)
	require.NoError(t, readErr)
	assert.Equal(t, "keep me", string(data))
	_, statErr := os.Stat(filepath.Join(outDir, schemapkg.FileName("alpha")))
	assert.ErrorIs(t, statErr, os.ErrNotExist)
	_, statErr = os.Stat(filepath.Join(outDir, schemapkg.FileName("beta")))
	assert.ErrorIs(t, statErr, os.ErrNotExist)
}

func TestSchemaOutOverwritesOnlyWithExplicitFlag(t *testing.T) {
	outDir := t.TempDir()
	for _, name := range []string{
		schemapkg.FileName("alpha"),
		schemapkg.FileName("beta"),
		schemapkg.RootFileName,
	} {
		require.NoError(t, os.WriteFile(filepath.Join(outDir, name), []byte("old"), 0o644))
	}

	_, err := executeCommand(t, "--out", outDir, "--overwrite")
	require.NoError(t, err)
	for _, name := range []string{
		schemapkg.FileName("alpha"),
		schemapkg.FileName("beta"),
		schemapkg.RootFileName,
	} {
		data, readErr := os.ReadFile(filepath.Join(outDir, name))
		require.NoError(t, readErr)
		assert.True(t, json.Valid(data), name)
		assert.NotEqual(t, "old", strings.TrimSpace(string(data)))
	}
}

func TestSchemaCommandDoesNotRequireCredentials(t *testing.T) {
	t.Setenv("RUDDERSTACK_ACCESS_TOKEN", "")
	t.Setenv("RUDDERSTACK_CLI_EXPERIMENTAL", "false")
	config.InitConfig(filepath.Join(t.TempDir(), "config.json"))

	cmd := NewCmdSchema()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	cmd.SetArgs(nil)
	require.NoError(t, cmd.Execute())
	assert.Contains(t, output.String(), "data-graph\n")
}
