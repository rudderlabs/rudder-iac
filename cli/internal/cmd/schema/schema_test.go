package schema

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	schemagen "github.com/rudderlabs/rudder-iac/cli/internal/schema"
)

func runCmd(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := NewCmdSchema()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return buf.String(), err
}

func TestSchemaCmdListsKinds(t *testing.T) {
	out, err := runCmd(t)
	require.NoError(t, err)
	for _, kind := range schemagen.Kinds() {
		assert.Contains(t, out, kind)
	}
}

func TestSchemaCmdPrintsKind(t *testing.T) {
	out, err := runCmd(t, "tracking-plan")
	require.NoError(t, err)

	var doc map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &doc))
	assert.Equal(t, "https://json-schema.org/draft/2020-12/schema", doc["$schema"])
}

func TestSchemaCmdUnknownKind(t *testing.T) {
	_, err := runCmd(t, "not-a-kind")
	assert.Error(t, err)
}

func TestSchemaCmdWritesAll(t *testing.T) {
	dir := t.TempDir()
	_, err := runCmd(t, "--out", dir)
	require.NoError(t, err)

	for _, kind := range schemagen.Kinds() {
		path := filepath.Join(dir, schemagen.FileName(kind))
		b, err := os.ReadFile(path)
		require.NoError(t, err, "expected schema file for %q", kind)

		var doc map[string]any
		require.NoError(t, json.Unmarshal(b, &doc))
		assert.Equal(t, "https://json-schema.org/draft/2020-12/schema", doc["$schema"])
	}

	rootPath := filepath.Join(dir, schemagen.RootFileName)
	_, err = os.Stat(rootPath)
	require.NoError(t, err, "expected combined root schema file")
}
