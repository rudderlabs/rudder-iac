package reporters

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/differ"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/planner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func operation(opType planner.OperationType, id, kind string) *planner.Operation {
	return &planner.Operation{
		Type:     opType,
		Resource: resources.NewResource(id, kind, nil, nil),
	}
}

// The document is a contract for agents, so these tests compare the wire format
// against a literal. Decoding into the reporter's own types would keep passing
// after a field was renamed or an omitempty dropped.

// A dry run reports a plan and no task results.
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

	require.NoError(t, r.Flush(nil))

	// Property names are sorted and carry no values: a diff can hold secrets.
	assert.JSONEq(t, `{
		"dryRun": true,
		"plan": {
			"operations": [
				{"type": "create", "urn": "property:fresh"},
				{"type": "update", "urn": "property:renamed", "properties": ["description", "name"]},
				{"type": "update", "urn": "destination:creds", "properties": ["apiKey"], "secretOnly": true},
				{"type": "delete", "urn": "property:gone"}
			],
			"summary": {"create": 1, "update": 2, "delete": 1, "import": 0}
		},
		"results": []
	}`, buf.String())
}

// An import keeps its changes in ImportableDiffs. One that overwrites remote
// fields must not look like one that only adopts.
func TestJSONSyncReporter_ImportListsTheFieldsItOverwrites(t *testing.T) {
	var buf bytes.Buffer

	r := NewJSONSyncReporter(&buf, true)
	r.ReportPlan(&planner.Plan{
		Diff: &differ.Diff{
			ImportableDiffs: map[string]differ.ResourceDiff{
				"source:overwrites": {
					URN:   "source:overwrites",
					Diffs: map[string]differ.PropertyDiff{"name": {Property: "name"}, "enabled": {Property: "enabled"}},
				},
			},
		},
		Operations: []*planner.Operation{
			operation(planner.Import, "overwrites", "source"),
			operation(planner.Import, "adopts", "source"),
		},
	})

	require.NoError(t, r.Flush(nil))

	assert.JSONEq(t, `{
		"dryRun": true,
		"plan": {
			"operations": [
				{"type": "import", "urn": "source:overwrites", "properties": ["enabled", "name"]},
				{"type": "import", "urn": "source:adopts"}
			],
			"summary": {"create": 0, "update": 0, "delete": 0, "import": 2}
		},
		"results": []
	}`, buf.String())
}

// An empty plan must still produce a document: silence cannot distinguish
// "nothing to do" from "died before reporting".
func TestJSONSyncReporter_EmptyPlanStillEmitsDocument(t *testing.T) {
	var buf bytes.Buffer

	r := NewJSONSyncReporter(&buf, false)
	r.ReportPlan(&planner.Plan{Diff: &differ.Diff{}})

	require.NoError(t, r.Flush(nil))

	assert.JSONEq(t, `{
		"dryRun": false,
		"plan": {"operations": [], "summary": {"create": 0, "update": 0, "delete": 0, "import": 0}},
		"results": []
	}`, buf.String())
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

	require.NoError(t, r.Flush(errors.New("syncing resources: 1 task failed")))

	assert.JSONEq(t, `{
		"dryRun": false,
		"plan": {
			"operations": [{"type": "create", "urn": "property:ok"}],
			"summary": {"create": 1, "update": 0, "delete": 0, "import": 0}
		},
		"results": [
			{"urn": "property:ok", "status": "ok"},
			{"urn": "property:bad", "status": "failed", "error": "upstream rejected the request"}
		],
		"error": "syncing resources: 1 task failed"
	}`, buf.String())
}

// A run that failed before it had a plan must not write the document of a run
// with nothing to change: plan is null and the error says why.
func TestJSONSyncReporter_FailureBeforeAPlanIsNotTheNoChangesDocument(t *testing.T) {
	var buf bytes.Buffer

	require.NoError(t, NewJSONSyncReporter(&buf, false).Flush(errors.New("syncing resources: 401 unauthorized")))

	assert.JSONEq(t, `{
		"dryRun": false,
		"plan": null,
		"results": [],
		"error": "syncing resources: 401 unauthorized"
	}`, buf.String())
}

// The syncer reports task results from many goroutines. Under -race this fails
// on an unguarded append, and the count catches results lost to it.
func TestJSONSyncReporter_ConcurrentTaskResultsAreAllKept(t *testing.T) {
	const tasks = 200

	var buf bytes.Buffer
	r := NewJSONSyncReporter(&buf, false)

	var wg sync.WaitGroup
	for i := range tasks {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var err error
			if i%2 == 0 {
				err = errors.New("failed")
			}
			r.TaskCompleted(fmt.Sprintf("property:p%d", i), "", err)
		}()
	}
	wg.Wait()

	require.NoError(t, r.Flush(nil))

	var out struct {
		Results []struct {
			Status string `json:"status"`
		} `json:"results"`
	}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &out))
	assert.Len(t, out.Results, tasks)

	failed := 0
	for _, result := range out.Results {
		if result.Status == "failed" {
			failed++
		}
	}
	assert.Equal(t, tasks/2, failed, "a failed operation must not be lost")
}

// The reporter has no way to prompt a consumer that is parsing stdout, so it
// declines. If it ever returned true, a plain apply --json would apply changes.
func TestJSONSyncReporter_AlwaysDeclinesConfirmation(t *testing.T) {
	confirmed, err := NewJSONSyncReporter(&bytes.Buffer{}, false).AskConfirmation()

	require.NoError(t, err)
	assert.False(t, confirmed)
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }

// A document that cannot be written is reported to the command, which turns it
// into a failure rather than exiting 0 with nothing on stdout.
func TestJSONSyncReporter_FlushReportsAWriteFailure(t *testing.T) {
	err := NewJSONSyncReporter(failingWriter{}, false).Flush(nil)

	assert.ErrorContains(t, err, "write failed")
}
