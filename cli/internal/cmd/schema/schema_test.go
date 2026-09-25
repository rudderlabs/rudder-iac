package schema

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	schemapkg "github.com/rudderlabs/rudder-iac/cli/internal/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSchemaCommandListsKinds(t *testing.T) {
	config.InitConfig(filepath.Join(t.TempDir(), "config.json"))
	cmd := NewCmdSchema()
	var output bytes.Buffer
	cmd.SetOut(&output)

	require.NoError(t, cmd.Execute())
	assert.Contains(t, output.String(), "data-graph\n")
	assert.Contains(t, output.String(), "destination\n")
}

func TestSchemaCommandWritesAll(t *testing.T) {
	config.InitConfig(filepath.Join(t.TempDir(), "config.json"))
	outDir := filepath.Join(t.TempDir(), "schemas")
	cmd := NewCmdSchema()
	cmd.SetArgs([]string{"--out", outDir})

	require.NoError(t, cmd.Execute())
	assert.FileExists(t, filepath.Join(outDir, schemapkg.RootFileName))
	assert.FileExists(t, filepath.Join(outDir, schemapkg.FileName("data-graph")))
}

func TestSchemaCommandRejectsUnknownKind(t *testing.T) {
	config.InitConfig(filepath.Join(t.TempDir(), "config.json"))
	cmd := NewCmdSchema()
	cmd.SetArgs([]string{"not-a-kind"})
	assert.Error(t, cmd.Execute())
}
