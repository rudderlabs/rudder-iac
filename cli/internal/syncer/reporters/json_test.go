package reporters

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/differ"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/planner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func operation(t planner.OperationType, urn, kind string) *planner.Operation {
	return &planner.Operation{
		Type:     t,
		Resource: resources.NewResource(urn, kind, nil, nil),
	}
}

func decode(t *testing.T, buf *bytes.Buffer) jsonSyncOutput {
	t.Helper()

	var out jsonSyncOutput
	require.NoError(t, json.Unmarshal(buf.Bytes(), &out), "output must be a single valid JSON document")
	return out
}

// A dry run returns from the syncer before SyncCompleted is ever reached, which
// is why the command flushes explicitly rather than relying on the lifecycle.
func TestJSONSyncReporter_DryRunPlan(t *testing.T) {
	var buf bytes.Buffer

	r := NewJSONSyncReporter(&buf, true)
	r.ReportPlan(&planner.Plan{
		Diff: &differ.Diff{
			UpdatedResources: map[string]differ.ResourceDiff{
				"property:renamed": {
					URN: "property:renamed",
					Diffs: map[string]differ.PropertyDiff{
						"name":        {Property: "name"},
						"description": {Property: "description"},
					},
				},
				"destination:creds": {
					URN:        "destination:creds",
					SecretOnly: true,
					Diffs:      map[string]differ.PropertyDiff{"apiKey": {Property: "apiKey", SecretOnly: true}},
				},
			},
		},
		Operations: []*planner.Operation{
			operation(planner.Create, "fresh", "property"),
			operation(planner.Update, "renamed", "property"),
			operation(planner.Update, "creds", "destination"),
			operation(planner.Delete, "gone", "property"),
		},
	})

	require.NoError(t, r.Flush())

	assert.Equal(t, jsonSyncOutput{
		DryRun: true,
		Plan: jsonPlan{
			Operations: []jsonOperation{
				{Type: "create", URN: "property:fresh"},
				// Property names are sorted and carry no values — a diff can hold secrets.
				{Type: "update", URN: "property:renamed", Properties: []string{"description", "name"}},
				{Type: "update", URN: "destination:creds", Properties: []string{"apiKey"}, SecretOnly: true},
				{Type: "delete", URN: "property:gone"},
			},
			Summary: map[string]int{"create": 1, "update": 2, "delete": 1, "import": 0},
		},
		Results: []jsonTaskResult{},
	}, decode(t, &buf))
}

// An empty plan must still produce a document: silence cannot distinguish
// "nothing to do" from "died before reporting".
func TestJSONSyncReporter_EmptyPlanStillEmitsDocument(t *testing.T) {
	var buf bytes.Buffer

	r := NewJSONSyncReporter(&buf, false)
	r.ReportPlan(&planner.Plan{Diff: &differ.Diff{}})

	require.NoError(t, r.Flush())

	assert.Equal(t, jsonSyncOutput{
		DryRun:  false,
		Plan:    jsonPlan{Operations: []jsonOperation{}, Summary: map[string]int{"create": 0, "update": 0, "delete": 0, "import": 0}},
		Results: []jsonTaskResult{},
	}, decode(t, &buf))
}

func TestJSONSyncReporter_RecordsTaskOutcomes(t *testing.T) {
	var buf bytes.Buffer

	r := NewJSONSyncReporter(&buf, false)
	r.ReportPlan(&planner.Plan{
		Diff:       &differ.Diff{},
		Operations: []*planner.Operation{operation(planner.Create, "ok", "property")},
	})
	r.TaskCompleted("property:ok", "Create property:ok", nil)
	r.TaskCompleted("property:bad", "Create property:bad", errors.New("upstream rejected the request"))

	require.NoError(t, r.Flush())

	assert.Equal(t, []jsonTaskResult{
		{URN: "property:ok", Status: "ok"},
		{URN: "property:bad", Status: "failed", Error: "upstream rejected the request"},
	}, decode(t, &buf).Results)
}

// Flush with no plan at all — the syncer failed before planning — must not panic
// and must still hand the caller a parseable document.
func TestJSONSyncReporter_FlushWithoutPlan(t *testing.T) {
	var buf bytes.Buffer

	require.NoError(t, NewJSONSyncReporter(&buf, false).Flush())
	assert.Empty(t, decode(t, &buf).Plan.Operations)
}
