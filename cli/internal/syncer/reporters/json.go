package reporters

import (
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"
	"sync"

	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/differ"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/planner"
)

// JSONSyncReporter accumulates the plan and per-operation results and emits them
// as a single JSON document, so a machine consumer can reason about blast radius
// before approving an apply.
//
// It writes nothing from the callbacks. A dry run, an empty plan, a declined
// confirmation and a failed load all return before SyncCompleted, so the command
// calls Flush on every exit path instead.
type JSONSyncReporter struct {
	w io.Writer

	// mu guards out: the syncer reports task results from many goroutines.
	mu  sync.Mutex
	out jsonSyncOutput
}

// Task result statuses and plan operation types, as they appear in the document.
const (
	resultOK     = "ok"
	resultFailed = "failed"

	opCreate = "create"
	opUpdate = "update"
	opDelete = "delete"
	opImport = "import"
)

// NewJSONSyncReporter returns a reporter that writes its document to w when
// Flush is called. dryRun is recorded in the document.
func NewJSONSyncReporter(w io.Writer, dryRun bool) *JSONSyncReporter {
	return &JSONSyncReporter{w: w, out: jsonSyncOutput{DryRun: dryRun, Results: []jsonTaskResult{}}}
}

type jsonOperation struct {
	Type string `json:"type"`
	URN  string `json:"urn"`
	// Properties names the fields that changed on an update or an import. Names
	// only, never values: a diff can carry secrets, and the name alone is what a
	// consumer needs to judge blast radius.
	Properties []string `json:"properties,omitempty"`
	// SecretOnly marks a resource that differs only by an unreadable secret, so
	// it re-applies on every run. Without it a consumer sees permanent drift and
	// concludes the apply never converges.
	SecretOnly bool `json:"secretOnly,omitempty"`
}

type jsonTaskResult struct {
	URN    string `json:"urn"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type jsonPlan struct {
	Operations []jsonOperation `json:"operations"`
	Summary    map[string]int  `json:"summary"`
}

// jsonSyncOutput is the document apply --json writes. Plan is null when the run
// failed before a plan existed, and Error then says why: without it that run
// would look the same as one with nothing to change.
type jsonSyncOutput struct {
	DryRun  bool             `json:"dryRun"`
	Plan    *jsonPlan        `json:"plan"`
	Results []jsonTaskResult `json:"results"`
	Error   string           `json:"error,omitempty"`
}

func (r *JSONSyncReporter) ReportPlan(plan *planner.Plan) {
	r.mu.Lock()
	defer r.mu.Unlock()

	out := &jsonPlan{
		Operations: make([]jsonOperation, 0, len(plan.Operations)),
		Summary:    map[string]int{opCreate: 0, opUpdate: 0, opDelete: 0, opImport: 0},
	}

	for _, op := range plan.Operations {
		urn := op.Resource.URN()
		entry := jsonOperation{Type: strings.ToLower(op.Type.String()), URN: urn}

		// An import keeps its changes in ImportableDiffs, not UpdatedResources.
		diffs := plan.Diff.UpdatedResources
		if op.Type == planner.Import {
			diffs = plan.Diff.ImportableDiffs
		}
		if diff, ok := diffs[urn]; ok {
			entry.SecretOnly = diff.IsSecretOnly()
			entry.Properties = propertyNames(diff)
		}

		out.Operations = append(out.Operations, entry)
		out.Summary[entry.Type]++
	}

	r.out.Plan = out
}

func propertyNames(diff differ.ResourceDiff) []string {
	return slices.Sorted(maps.Keys(diff.Diffs))
}

// AskConfirmation always declines, matching PlainSyncReporter: there is no way to
// prompt a consumer that is parsing stdout. The command refuses --json without
// --dry-run or --confirm=false, so a declined apply cannot be mistaken for one
// that ran.
func (r *JSONSyncReporter) AskConfirmation() (bool, error) { return false, nil }

func (r *JSONSyncReporter) SyncStarted(int)            {}
func (r *JSONSyncReporter) SyncCompleted()             {}
func (r *JSONSyncReporter) TaskStarted(string, string) {}

func (r *JSONSyncReporter) TaskCompleted(taskID string, _ string, err error) {
	result := jsonTaskResult{URN: taskID, Status: resultOK}
	if err != nil {
		result.Status = resultFailed
		result.Error = err.Error()
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.out.Results = append(r.out.Results, result)
}

// Flush writes the document. runErr is what the command is returning: a failure
// is recorded in the document, so it is never left holding only "ok" results.
func (r *JSONSyncReporter) Flush(runErr error) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if runErr != nil {
		r.out.Error = runErr.Error()
	}

	encoder := json.NewEncoder(r.w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(r.out); err != nil {
		return fmt.Errorf("encoding sync report: %w", err)
	}

	return nil
}
