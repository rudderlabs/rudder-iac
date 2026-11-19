package importer

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/writer"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
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

// TestCheckVariableOwners_RefusesAVariableAnExistingSpecUses pins the
// slice-by-slice case: the project already manages account "pg", whose
// password fills PG_PASSWORD, and a newly imported destination "pg" would be
// handed the same variable for its own password.
func TestCheckVariableOwners_RefusesAVariableAnExistingSpecUses(t *testing.T) {
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
	require.EqualError(t, err, "imported specs would share secret variables: "+
		"PG_PASSWORD (in accounts/pg.yaml and imported/destinations/pg.yaml); "+
		"the names come from each resource's id, so rename one of the resources whose ids collide and import again")
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

// TestCheckVariableOwners_AllowsTheSameSecretAgain pins that the same
// variable filling the same spec key — same kind, spec id and key path — is the
// same secret, not a collision. Raw code and non-spec files are not read.
func TestCheckVariableOwners_AllowsTheSameSecretAgain(t *testing.T) {
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

// TestCheckVariableOwners_LeavesExistingSharedVariablesAlone pins that only a
// clash the import introduces counts: two specs already in the project may
// share a variable on purpose, and a new import that does not touch it runs.
func TestCheckVariableOwners_LeavesExistingSharedVariablesAlone(t *testing.T) {
	dir := t.TempDir()
	for _, id := range []string{"one", "two"} {
		writeProjectFile(t, dir, "destinations/"+id+".yaml", `version: rudder/v1
kind: destination
metadata:
  name: destination
spec:
  id: `+id+`
  config:
    api_key: '{{ .SHARED_API_KEY }}'
`)
	}
	assert.NoError(t, importInto(t, dir, newSpecEntity("destination", "destinations/three.yaml", map[string]any{
		"id": "three", "config": map[string]any{"api_key": "{{ .THREE_API_KEY }}"},
	})))
}

func existingAccount(t *testing.T, dir, passwordLine string) {
	t.Helper()
	writeProjectFile(t, dir, "accounts/pg.yaml", "version: rudder/v1\nkind: account\nmetadata:\n  name: pg\nspec:\n  id: pg\n  config:\n    "+passwordLine+"\n")
}

func newPGDestination() writer.FormattableEntity {
	return newSpecEntity("destination", "destinations/pg.yaml", map[string]any{
		"id": "pg", "config": map[string]any{"password": "{{ .PG_PASSWORD }}"},
	})
}

// TestCheckVariableOwners_ReadsReferencesAsSubstitutionDoes pins that existing
// references are found the way substitution finds them, in the bytes as
// written: an unquoted reference, which does not parse as YAML until it is
// substituted, still counts, and an escaped literal that only decodes to a
// reference does not.
func TestCheckVariableOwners_ReadsReferencesAsSubstitutionDoes(t *testing.T) {
	for _, line := range []string{"password: {{ .PG_PASSWORD }}", `password: "{{ .PG_PASSWORD }}"`, "password: '{{ .PG_PASSWORD }}'"} {
		dir := t.TempDir()
		existingAccount(t, dir, line)
		assert.ErrorContains(t, importInto(t, dir, newPGDestination()), "PG_PASSWORD (in accounts/pg.yaml and imported/destinations/pg.yaml)", line)
	}

	dir := t.TempDir()
	existingAccount(t, dir, `password: "\x7B{ .PG_PASSWORD }}"`)
	assert.NoError(t, importInto(t, dir, newPGDestination()), "substitution never sees an escaped literal")
}

// TestCheckVariableOwners_FailsOnASpecItCannotRead pins that a project spec
// holding references is never skipped silently: one that cannot be parsed
// fails the import, naming the file.
func TestCheckVariableOwners_FailsOnASpecItCannotRead(t *testing.T) {
	dir := t.TempDir()
	writeProjectFile(t, dir, "accounts/pg.yaml", "version: rudder/v1\nkind: account\nspec: [unclosed\n  password: '{{ .PG_PASSWORD }}'\n")
	assert.ErrorContains(t, importInto(t, dir, newPGDestination()), "reading project spec accounts/pg.yaml for secret variables")
}

// TestCheckVariableOwners_AllowsReimportingAnExistingOwner pins that a new
// reference whose spec key already uses the variable is fine, even when other
// existing specs share it too.
func TestCheckVariableOwners_AllowsReimportingAnExistingOwner(t *testing.T) {
	dir := t.TempDir()
	for _, id := range []string{"one", "two"} {
		writeProjectFile(t, dir, "destinations/"+id+".yaml", "version: rudder/v1\nkind: destination\nmetadata:\n  name: d\nspec:\n  id: "+id+"\n  config:\n    api_key: '{{ .ONE_API_KEY }}'\n")
	}
	assert.NoError(t, importInto(t, dir, newSpecEntity("destination", "imported-one.yaml", map[string]any{
		"id": "one", "config": map[string]any{"api_key": "{{ .ONE_API_KEY }}"},
	})))
}

// TestCheckVariableOwners_CountsReferencesAnywhereInTheDocument pins that a
// reference counts wherever substitution would replace it, not only in the
// spec body: in metadata, and as a mapping key.
func TestCheckVariableOwners_CountsReferencesAnywhereInTheDocument(t *testing.T) {
	for name, content := range map[string]string{
		"metadata.name": "version: rudder/v1\nkind: account\nmetadata:\n  name: '{{ .PG_PASSWORD }}'\nspec:\n  id: pg\n",
		"mapping key":   "version: rudder/v1\nkind: account\nmetadata:\n  name: pg\nspec:\n  id: pg\n  config:\n    '{{ .PG_PASSWORD }}': x\n",
	} {
		dir := t.TempDir()
		writeProjectFile(t, dir, "accounts/pg.yaml", content)
		assert.ErrorContains(t, importInto(t, dir, newPGDestination()), "PG_PASSWORD (in accounts/pg.yaml and imported/destinations/pg.yaml)", name)
	}
}
