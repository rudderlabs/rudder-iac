package reporters

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/planner"
)

// JSONSyncReporter accumulates the plan and per-operation results and emits them
// as a single JSON document, so a machine consumer can reason about blast radius
// before approving an apply.
//
// It accumulates rather than writing per callback because the reporter's own
// lifecycle is not a reliable flush point: a dry run, an empty plan and a
// declined confirmation all return from the syncer before SyncCompleted is ever
// called. The command owns the flush instead — see Flush.
type JSONSyncReporter struct {
	w       io.Writer
	dryRun  bool
	plan    *planner.Plan
	results []jsonTaskResult
}

func NewJSONSyncReporter(w io.Writer, dryRun bool) *JSONSyncReporter {
	return &JSONSyncReporter{w: w, dryRun: dryRun}
}

type jsonOperation struct {
	Type string `json:"type"`
	URN  string `json:"urn"`
	// Properties names the fields that changed on an update. Names only, never
	// values: a diff can carry secrets, and the name alone is what a consumer
	// needs to judge blast radius.
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

type jsonSyncOutput struct {
	DryRun  bool             `json:"dryRun"`
	Plan    jsonPlan         `json:"plan"`
	Results []jsonTaskResult `json:"results"`
}

func (r *JSONSyncReporter) ReportPlan(plan *planner.Plan) {
	r.plan = plan
}

// AskConfirmation always declines, matching PlainSyncReporter: there is no way to
// prompt a consumer that is parsing stdout. A caller that means to apply passes
// --confirm=false, which is the same contract CI already runs under.
func (r *JSONSyncReporter) AskConfirmation() (bool, error) { return false, nil }

func (r *JSONSyncReporter) SyncStarted(int)            {}
func (r *JSONSyncReporter) SyncCompleted()             {}
func (r *JSONSyncReporter) TaskStarted(string, string) {}

func (r *JSONSyncReporter) TaskCompleted(taskID string, _ string, err error) {
	result := jsonTaskResult{URN: taskID, Status: "ok"}
	if err != nil {
		result.Status = "failed"
		result.Error = err.Error()
	}
	r.results = append(r.results, result)
}

// Flush writes the accumulated document. The command defers it so every exit
// path is covered — including the ones that never reach SyncCompleted.
func (r *JSONSyncReporter) Flush() error {
	out := jsonSyncOutput{
		DryRun: r.dryRun,
		Plan: jsonPlan{
			Operations: r.operations(),
			Summary:    map[string]int{"create": 0, "update": 0, "delete": 0, "import": 0},
		},
		Results: r.results,
	}

	if out.Results == nil {
		out.Results = []jsonTaskResult{}
	}

	for _, op := range out.Plan.Operations {
		out.Plan.Summary[op.Type]++
	}

	encoder := json.NewEncoder(r.w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(out); err != nil {
		return fmt.Errorf("encoding sync report: %w", err)
	}

	return nil
}

func (r *JSONSyncReporter) operations() []jsonOperation {
	operations := make([]jsonOperation, 0)
	if r.plan == nil {
		return operations
	}

	for _, op := range r.plan.Operations {
		urn := op.Resource.URN()
		entry := jsonOperation{
			Type: strings.ToLower(op.Type.String()),
			URN:  urn,
		}

		if diff, ok := r.plan.Diff.UpdatedResources[urn]; ok {
			entry.SecretOnly = diff.IsSecretOnly()
			for property := range diff.Diffs {
				entry.Properties = append(entry.Properties, property)
			}
			sort.Strings(entry.Properties)
		}

		operations = append(operations, entry)
	}

	return operations
}
