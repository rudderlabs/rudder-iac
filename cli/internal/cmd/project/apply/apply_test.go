package apply

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry/telemetrytest"
	"github.com/rudderlabs/rudder-iac/cli/internal/testutils"
)

func TestApplyTracksRunFailureAsErrored(t *testing.T) {
	testutils.UseFakeAPI(t)
	calls := telemetrytest.Record(t)
	location := t.TempDir()

	err := telemetrytest.Execute(NewCmdApply(), nil, []string{"--location", location, "--confirm=false"})

	require.Error(t, err)
	assert.Equal(t, []telemetrytest.Call{{
		Command: "apply",
		Errored: true,
		Extras: []telemetry.KV{
			{K: "location", V: location},
			{K: "dryRun", V: false},
			{K: "confirm", V: false},
			{K: "json", V: false},
		},
	}}, *calls)
}

// Under --json stdout is one JSON document and nothing else. The banner goes to
// stderr, and a run that fails before it has a plan says so in the document
// rather than looking like a run with nothing to change.
func TestApplyJSONWritesOnlyTheDocumentToStdout(t *testing.T) {
	testutils.UseFakeAPI(t)
	telemetrytest.Record(t)

	cmd := NewCmdApply()
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)

	err := telemetrytest.Execute(cmd, nil, []string{"--location", t.TempDir(), "--json", "--dry-run"})
	require.Error(t, err, "the fake API fails the remote load")

	var doc struct {
		Plan    *json.RawMessage `json:"plan"`
		Results []any            `json:"results"`
		Error   string           `json:"error"`
	}
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &doc), "stdout must be a single JSON document, got:\n%s", stdout)
	assert.Nil(t, doc.Plan, "a failure before planning has no plan")
	assert.NotEmpty(t, doc.Error)
	assert.Empty(t, doc.Results)

	assert.NotContains(t, stdout.String(), "Workspace:")
	assert.Contains(t, stderr.String(), "Workspace:")
}

// The JSON reporter declines every confirmation, so --json alone would plan,
// apply nothing and exit 0.
func TestApplyJSONNeedsDryRunOrConfirmFalse(t *testing.T) {
	testutils.UseFakeAPI(t)
	telemetrytest.Record(t)

	err := telemetrytest.Execute(NewCmdApply(), nil, []string{"--location", t.TempDir(), "--json"})

	assert.ErrorContains(t, err, "--json needs --dry-run")
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("stdout closed") }

// A document that cannot be written must fail the command. Exiting 0 would
// leave the caller with no document, or a truncated one, and no error to see.
func TestApplyJSONFailsWhenTheDocumentCannotBeWritten(t *testing.T) {
	testutils.UseFakeAPI(t)
	calls := telemetrytest.Record(t)

	cmd := NewCmdApply()
	cmd.SetOut(failingWriter{})
	cmd.SetErr(&bytes.Buffer{})

	err := telemetrytest.Execute(cmd, nil, []string{"--location", t.TempDir(), "--json", "--dry-run"})

	assert.ErrorContains(t, err, "stdout closed")
	require.Len(t, *calls, 1)
	assert.True(t, (*calls)[0].Errored)
}
