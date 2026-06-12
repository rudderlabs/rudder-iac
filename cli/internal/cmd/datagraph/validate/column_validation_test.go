package validate_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/renderer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests assert that the `rudder-cli data-graphs validate` command
// surfaces the column-level validation rules added in Task 3b.2.
//
// The validate command's project-load gate (PreRunE) calls project.Load,
// which runs the datagraph provider's SyntacticRules — including
// `datagraph/data-graph/spec-syntax-valid`, which embeds validateModelColumns.
// We exercise that same pipeline here against the real datagraph provider,
// avoiding the full cobra/app.NewDeps stack so the test stays hermetic and
// doesn't need an auth token or running backend.
//
// Renderer output (the user-facing string the validate command prints) is
// captured into a buffer via project.WithRenderer for substring assertions.

func writeDataGraphYAML(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "data-graph.yaml")
	require.NoError(t, os.WriteFile(path, []byte(body), 0o600))
	return dir
}

// loadDataGraphProject runs the same load+validate pipeline the validate
// command runs in its PreRunE. Returns the loader error and the captured
// renderer output.
func loadDataGraphProject(t *testing.T, dir string) (error, string) {
	t.Helper()
	var buf bytes.Buffer
	dgp := datagraph.NewProvider(nil, nil)
	p := project.New(dgp, project.WithRenderer(renderer.NewTextRenderer(&buf)))
	err := p.Load(dir)
	return err, buf.String()
}

func TestValidate_ColumnsBlock_DuplicateName(t *testing.T) {
	t.Parallel()

	dir := writeDataGraphYAML(t, `version: rudder/v1
kind: data-graph
metadata:
  name: smoke-test
spec:
  id: smoke-test
  account_id: wh-account-123
  models:
    - id: user
      display_name: User
      type: entity
      table: db.schema.users
      primary_id: id
      columns:
        - name: id
          display_name: User ID
        - name: id
          display_name: Identifier
`)

	err, out := loadDataGraphProject(t, dir)
	require.Error(t, err, "duplicate column name must fail validation")
	assert.Contains(t, err.Error(), "syntax validation failed")

	// Both columns are flagged so users can find either one.
	assert.Contains(t, out, `duplicate column name "id"`)
	assert.Equal(t, 2, strings.Count(out, `duplicate column name "id"`),
		"expected both duplicate-name entries to be flagged; got:\n%s", out)
	assert.Contains(t, out, "datagraph/data-graph/spec-syntax-valid")
	assert.Contains(t, out, "Found 2 error(s)")
}

func TestValidate_ColumnsBlock_DuplicateDisplayNameCI(t *testing.T) {
	t.Parallel()

	dir := writeDataGraphYAML(t, `version: rudder/v1
kind: data-graph
metadata:
  name: smoke-test
spec:
  id: smoke-test
  account_id: wh-account-123
  models:
    - id: user
      display_name: User
      type: entity
      table: db.schema.users
      primary_id: id
      columns:
        - name: id
          display_name: User ID
        - name: uid
          display_name: user id
`)

	err, out := loadDataGraphProject(t, dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "syntax validation failed")

	// Message must name both conflicting columns so users can resolve it without
	// re-running. Substrings checked instead of full match to keep the test
	// resilient to wording tweaks.
	assert.Contains(t, out, "duplicate column display name")
	assert.Contains(t, out, "case-insensitive")
	assert.Contains(t, out, `"id"`)
	assert.Contains(t, out, `"uid"`)
	assert.Contains(t, out, "Found 2 error(s)")
}

func TestValidate_ColumnsBlock_DisplayNameWithNewline(t *testing.T) {
	t.Parallel()

	// JSON-style double-quoted scalar with explicit \n escape.
	dir := writeDataGraphYAML(t, `version: rudder/v1
kind: data-graph
metadata:
  name: smoke-test
spec:
  id: smoke-test
  account_id: wh-account-123
  models:
    - id: user
      display_name: User
      type: entity
      table: db.schema.users
      primary_id: id
      columns:
        - name: id
          display_name: "User\nID"
`)

	err, out := loadDataGraphProject(t, dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "syntax validation failed")

	assert.Contains(t, out, "'display_name' must not contain control characters")
	assert.Contains(t, out, "Found 1 error(s)")
}

func TestValidate_ColumnsBlock_DisplayNameEmpty(t *testing.T) {
	t.Parallel()

	dir := writeDataGraphYAML(t, `version: rudder/v1
kind: data-graph
metadata:
  name: smoke-test
spec:
  id: smoke-test
  account_id: wh-account-123
  models:
    - id: user
      display_name: User
      type: entity
      table: db.schema.users
      primary_id: id
      columns:
        - name: id
          display_name: ""
`)

	err, out := loadDataGraphProject(t, dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "syntax validation failed")

	assert.Contains(t, out, "at least one of 'display_name', 'description', or 'pii_mask'")
	assert.Contains(t, out, "Found 1 error(s)")
}

func TestValidate_ColumnsBlock_DisplayNameTooLong(t *testing.T) {
	t.Parallel()

	tooLong := strings.Repeat("a", 256)
	dir := writeDataGraphYAML(t, `version: rudder/v1
kind: data-graph
metadata:
  name: smoke-test
spec:
  id: smoke-test
  account_id: wh-account-123
  models:
    - id: user
      display_name: User
      type: entity
      table: db.schema.users
      primary_id: id
      columns:
        - name: id
          display_name: "`+tooLong+`"
`)

	err, out := loadDataGraphProject(t, dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "syntax validation failed")

	assert.Contains(t, out, "'display_name' length must be less than or equal to 255")
	assert.Contains(t, out, "Found 1 error(s)")
}

func TestValidate_ColumnsBlock_NameMissing(t *testing.T) {
	t.Parallel()

	dir := writeDataGraphYAML(t, `version: rudder/v1
kind: data-graph
metadata:
  name: smoke-test
spec:
  id: smoke-test
  account_id: wh-account-123
  models:
    - id: user
      display_name: User
      type: entity
      table: db.schema.users
      primary_id: id
      columns:
        - display_name: User ID
`)

	err, out := loadDataGraphProject(t, dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "syntax validation failed")
	assert.Contains(t, out, "'name' is required")
}

func TestValidate_ColumnsBlock_Valid(t *testing.T) {
	t.Parallel()

	dir := writeDataGraphYAML(t, `version: rudder/v1
kind: data-graph
metadata:
  name: smoke-test
spec:
  id: smoke-test
  account_id: wh-account-123
  models:
    - id: user
      display_name: User
      type: entity
      table: db.schema.users
      primary_id: id
      columns:
        - name: id
          display_name: User ID
        - name: email_address
          display_name: Email
`)

	// A clean columns block passes the local validate pipeline. The validate
	// command itself would then proceed into the remote validator runner; this
	// test only confirms the local gate is not falsely flagging anything.
	err, out := loadDataGraphProject(t, dir)
	require.NoError(t, err, "valid columns block must pass local validation; renderer output was:\n%s", out)
	assert.NotContains(t, out, "error[")
}
