package importer

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/writer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeProjectFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	path := filepath.Join(dir, rel)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
}

func newSpecEntity(kind, rel string, spec map[string]any) writer.FormattableEntity {
	return writer.FormattableEntity{
		Content:      &specs.Spec{Version: specs.SpecVersionV1, Kind: kind, Metadata: map[string]any{"name": kind}, Spec: spec},
		RelativePath: rel,
	}
}

// TestCheckExistingVariableOwners_RefusesAVariableAnExistingSpecUses pins the
// slice-by-slice case: the project already manages account "pg", whose
// password fills PG_PASSWORD, and a newly imported destination "pg" would be
// handed the same variable for its own password.
func TestCheckExistingVariableOwners_RefusesAVariableAnExistingSpecUses(t *testing.T) {
	dir := t.TempDir()
	writeProjectFile(t, dir, "accounts/pg.yaml", `version: rudder/v1
kind: account
metadata:
  name: pg
spec:
  id: pg
  secret:
    password: '{{ .PG_PASSWORD }}'
`)

	err := importInto(t, dir, newSpecEntity("destination", "destinations/pg.yaml", map[string]any{
		"id": "pg", "config": map[string]any{"password": "{{ .PG_PASSWORD }}"},
	}))
	require.EqualError(t, err, "imported specs would share secret variables with specs already in the project: "+
		"PG_PASSWORD (in accounts/pg.yaml and new destinations/pg.yaml); "+
		"the names come from each resource's id, so rename the new or the existing resource and import again")
	_, statErr := os.Stat(filepath.Join(dir, ImportedDir))
	assert.True(t, os.IsNotExist(statErr), "a refused import writes nothing")
}

// importInto runs a workspace import into the project at dir that exports the
// given entities.
func importInto(t *testing.T, dir string, entities ...writer.FormattableEntity) error {
	t.Helper()
	return WorkspaceImport(context.Background(), &stubProject{location: dir, graph: resources.NewGraph()},
		&stubImportProvider{importable: importableCollection(), entities: entities}, ImportOptions{})
}

// TestCheckExistingVariableOwners_AllowsTheSameSecretAgain pins that the same
// variable filling the same spec key — same kind, spec id and key path — is the
// same secret, not a collision. Raw code and non-spec files are not read.
func TestCheckExistingVariableOwners_AllowsTheSameSecretAgain(t *testing.T) {
	dir := t.TempDir()
	writeProjectFile(t, dir, "destinations/hook.yaml", `version: rudder/v1
kind: destination
metadata:
  name: destination
spec:
  id: hook
  config:
    bearer_token: '{{ .HOOK_BEARER_TOKEN }}'
`)
	writeProjectFile(t, dir, "secrets.vars.yaml", "HOOK_BEARER_TOKEN: 'x'\nOTHER: 'y'\n")
	writeProjectFile(t, dir, "transformations/code.js", "export function t() { return '{{ .OTHER }}' }\n")

	assert.NoError(t, importInto(t, dir,
		newSpecEntity("destination", "destinations/hook.yaml", map[string]any{
			"id": "hook", "config": map[string]any{"bearer_token": "{{ .HOOK_BEARER_TOKEN }}"},
		}),
		newSpecEntity("destination", "destinations/other.yaml", map[string]any{
			"id": "other", "config": map[string]any{"token": "{{ .OTHER }}"},
		}),
	))
}
