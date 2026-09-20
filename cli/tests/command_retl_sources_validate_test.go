package tests

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRETLSourceValidateNoQuery covers the case this PR turns from a failure
// into a pass: a source with no warehouse query to run.
//
// An s3 table source has no query, and that is its steady state rather than a
// defect — so a CI step that validates every source in a project must not go
// red the day an s3 source joins it. Before this change `validate` reported
// "❌ SQL query failed to execute" and exited non-zero.
//
// Notably this is the first e2e in the package to invoke `validate` at all.
// DEX-903 recorded that every suite went straight to `apply`, so the command a
// user reaches for first had no end-to-end coverage. It is also the cheapest
// possible such test: `validate` builds the project graph locally and the s3
// branch returns before any API call, so nothing needs to exist upstream.
func TestRETLSourceValidateNoQuery(t *testing.T) {
	t.Setenv("RUDDERSTACK_CLI_EXPERIMENTAL", "true")
	t.Setenv("RUDDERSTACK_X_RETL_TABLE_SUPPORT", "true")

	executor, err := NewCmdExecutor("")
	require.NoError(t, err)

	// No var file: `retl-sources validate` and `preview` take only --location,
	// so a fixture that uses variables cannot be validated by them at all. The
	// password here is a literal for that reason; the account is never applied.
	projectDir := filepath.Join("testdata", "retl_source_validate")

	t.Run("validate passes and says there was nothing to run", func(t *testing.T) {
		out, err := executor.Execute(cliBinPath, "retl-sources", "validate", "archive-bucket",
			"--location", projectDir)

		require.NoError(t, err, "validate should pass for a source with no query: %s", out)
		assert.Contains(t, string(out), "Nothing to validate",
			"a pass must say why there was nothing to run, not print a bare success")
	})

	// The complement: preview asks for rows, and for a source that has none to
	// give the answer is an error rather than an empty result.
	t.Run("preview refuses and names the source kind", func(t *testing.T) {
		out, err := executor.Execute(cliBinPath, "retl-sources", "preview", "archive-bucket",
			"--location", projectDir)

		require.Error(t, err, "preview should refuse an s3 source, got: %s", out)
		assert.Contains(t, string(out), "preview is not supported for s3 table sources")
	})
}
