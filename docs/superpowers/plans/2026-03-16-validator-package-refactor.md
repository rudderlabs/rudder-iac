# Validator Package Refactor Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Consolidate all data-graph validation components into a single `validator/` package, refactoring based on 12 PR review threads covering architecture, testing, UX, and code quality.

**Architecture:** Move `validations/` and `display/` packages into `cli/internal/providers/datagraph/validator/`. Extract orchestration from the cobra command into a standalone `Validate()` function (importer.go pattern). Replace the single spinner with per-task spinners via `ui.TaskReporter`. Add `SilentError` for clean `--json` exit behavior.

**Tech Stack:** Go, Cobra CLI, Bubble Tea (charmbracelet), testify, golang.org/x/term

---

## File Structure

### New files (all under `cli/internal/providers/datagraph/validator/`)

| File | Responsibility |
|------|---------------|
| `mode.go` | `Mode` sealed interface + `ModeAll`, `ModeModified`, `ModeSingle` concrete types |
| `planner.go` | `PlannerFunc` type, `PlanAll`, `PlanModified`, `PlanSingle` standalone functions, `resourcesToUnits` helper |
| `planner_test.go` | Full-content struct assertions for all plan functions |
| `report.go` | `ValidationReport` (renamed from `ValidationResults`), `ResourceValidation`, `RunStatus` |
| `runner.go` | `Runner` struct orchestrating plan → resolve account IDs → execute tasks |
| `runner_test.go` | Runner tests updated for new Mode types and ValidationReport |
| `tasker.go` | `validateTask`, `runValidationTasks` using URN as key, `ValidationReporter` interface |
| `tasker_test.go` | Tasker tests updated for URN-based keys |
| `displayer.go` | `Displayer` interface |
| `terminal_displayer.go` | `TerminalDisplayer` with dynamic terminal width |
| `terminal_displayer_test.go` | Full-output assertions for terminal display |
| `json_displayer.go` | `JSONDisplayer` |
| `json_displayer_test.go` | JSON output assertions |
| `validator.go` | Top-level `Validate()` orchestrator function + dependency interfaces |

### Modified files

| File | Changes |
|------|---------|
| `cli/internal/cmd/root.go` | Add `SilentError` type + handling in `Execute()`, unhide `datagraphCmd` in `initConfig()` |
| `cli/internal/cmd/datagraph/datagraph.go` | Set `Hidden: true` |
| `cli/internal/cmd/datagraph/validate/validate.go` | Slim down to collect args + call `validator.Validate()` |
| `cli/internal/cmd/datagraph/validate/validate_test.go` | Update for new command structure |

### Deleted files

| File | Reason |
|------|--------|
| `cli/internal/providers/datagraph/validations/*.go` | Moved to `validator/` |
| `cli/internal/providers/datagraph/display/*.go` | Moved to `validator/` |

---

## Chunk 1: Foundation types (mode, report, planner)

### Task 1: Create `mode.go` — Mode sealed interface

**Files:**
- Create: `cli/internal/providers/datagraph/validator/mode.go`

- [ ] **Step 1: Create mode.go with sealed interface**

```go
package validator

// Mode is a sealed interface representing the validation mode.
// Implemented by ModeAll, ModeModified, and ModeSingle.
type Mode interface {
	validationMode()
}

// ModeAll validates all resources in the project.
type ModeAll struct{}

// ModeModified validates only new or modified resources (compared to remote state).
type ModeModified struct{}

// ModeSingle validates a specific resource by type and ID.
type ModeSingle struct {
	ResourceType string
	TargetID     string
}

func (ModeAll) validationMode()      {}
func (ModeModified) validationMode() {}
func (ModeSingle) validationMode()   {}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd cli && go build ./internal/providers/datagraph/validator/`

### Task 2: Create `report.go` — ValidationReport (renamed from ValidationResults)

**Files:**
- Create: `cli/internal/providers/datagraph/validator/report.go`

- [ ] **Step 1: Create report.go**

Move the content from `validations/results.go`, renaming `ValidationResults` → `ValidationReport`. Keep all methods (`HasFailures`, `ErrorCount`, `WarningCount`, `PassCount`, `ResourcesByType`). Keep `ResourceValidation` and `RunStatus` as-is.

```go
package validator

import dgClient "github.com/rudderlabs/rudder-iac/api/client/datagraph"

type RunStatus int

const (
	RunStatusExecuted    RunStatus = iota
	RunStatusNoResources
)

type ResourceValidation struct {
	ID           string
	DisplayName  string
	ResourceType string
	Issues       []dgClient.ValidationIssue
	Err          error
}

func (rv *ResourceValidation) HasErrors() bool {
	if rv.Err != nil {
		return true
	}
	for _, issue := range rv.Issues {
		if issue.Severity == "error" {
			return true
		}
	}
	return false
}

func (rv *ResourceValidation) HasWarnings() bool {
	for _, issue := range rv.Issues {
		if issue.Severity == "warning" {
			return true
		}
	}
	return false
}

// ValidationReport holds all validation results for a run.
type ValidationReport struct {
	Status    RunStatus
	Resources []*ResourceValidation
}

func (vr *ValidationReport) HasFailures() bool {
	for _, r := range vr.Resources {
		if r.HasErrors() {
			return true
		}
	}
	return false
}

func (vr *ValidationReport) ErrorCount() int {
	count := 0
	for _, r := range vr.Resources {
		if r.HasErrors() {
			count++
		}
	}
	return count
}

func (vr *ValidationReport) WarningCount() int {
	count := 0
	for _, r := range vr.Resources {
		if !r.HasErrors() && r.HasWarnings() {
			count++
		}
	}
	return count
}

func (vr *ValidationReport) PassCount() int {
	count := 0
	for _, r := range vr.Resources {
		if !r.HasErrors() && !r.HasWarnings() {
			count++
		}
	}
	return count
}

func (vr *ValidationReport) ResourcesByType(resourceType string) []*ResourceValidation {
	var result []*ResourceValidation
	for _, r := range vr.Resources {
		if r.ResourceType == resourceType {
			result = append(result, r)
		}
	}
	return result
}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd cli && go build ./internal/providers/datagraph/validator/`

### Task 3: Create `planner.go` — Standalone plan functions with PlannerFunc

**Files:**
- Create: `cli/internal/providers/datagraph/validator/planner.go`

- [ ] **Step 1: Create planner.go**

Standalone functions `PlanAll`, `PlanModified`, `PlanSingle` with shared `resourcesToUnits` helper. `ValidationUnit` gains a `URN` field. All functions return `*ValidationPlan`.

```go
package validator

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/handlers/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/handlers/relationship"
	dgModel "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/differ"
)

// ValidationUnit represents a single resource to validate.
type ValidationUnit struct {
	URN          string // resource URN, used as task key
	ResourceType string // "model" or "relationship"
	ID           string // local resource ID
	Resource     any    // *dgModel.ModelResource or *dgModel.RelationshipResource
	AccountID    string // resolved account ID from the parent data graph spec
}

// ValidationPlan holds the units to validate.
type ValidationPlan struct {
	Units []*ValidationUnit
}

// PlanAll returns a validation plan containing all model and relationship resources in the graph.
func PlanAll(graph *resources.Graph) (*ValidationPlan, error) {
	var allResources []*resources.Resource
	allResources = append(allResources, graph.ResourcesByType(model.HandlerMetadata.ResourceType)...)
	allResources = append(allResources, graph.ResourcesByType(relationship.HandlerMetadata.ResourceType)...)

	units := resourcesToUnits(allResources)
	return &ValidationPlan{Units: units}, nil
}

// PlanModified returns a validation plan containing only new or modified resources
// compared to the remote state.
func PlanModified(graph *resources.Graph, remoteGraph *resources.Graph, opts differ.DiffOptions) (*ValidationPlan, error) {
	diff := differ.ComputeDiff(remoteGraph, graph, opts)

	modifiedURNs := make(map[string]bool)
	for _, urn := range diff.NewResources {
		modifiedURNs[urn] = true
	}
	for _, urn := range diff.ImportableResources {
		modifiedURNs[urn] = true
	}
	for urn := range diff.UpdatedResources {
		modifiedURNs[urn] = true
	}

	var modified []*resources.Resource
	for _, r := range graph.ResourcesByType(model.HandlerMetadata.ResourceType) {
		if modifiedURNs[r.URN()] {
			modified = append(modified, r)
		}
	}
	for _, r := range graph.ResourcesByType(relationship.HandlerMetadata.ResourceType) {
		if modifiedURNs[r.URN()] {
			modified = append(modified, r)
		}
	}

	units := resourcesToUnits(modified)
	return &ValidationPlan{Units: units}, nil
}

// PlanSingle returns a validation plan for a specific resource identified by type and ID.
func PlanSingle(graph *resources.Graph, resourceType, targetID string) (*ValidationPlan, error) {
	var handlerType string
	switch resourceType {
	case "model":
		handlerType = model.HandlerMetadata.ResourceType
	case "relationship":
		handlerType = relationship.HandlerMetadata.ResourceType
	default:
		return nil, fmt.Errorf("unknown resource type: %s, must be 'model' or 'relationship'", resourceType)
	}

	urn := resources.URN(targetID, handlerType)
	r, exists := graph.GetResource(urn)
	if !exists {
		return nil, fmt.Errorf("resource %q of type %q not found in project", targetID, resourceType)
	}

	units := resourcesToUnits([]*resources.Resource{r})
	return &ValidationPlan{Units: units}, nil
}

// resourcesToUnits converts graph resources to validation units.
func resourcesToUnits(rs []*resources.Resource) []*ValidationUnit {
	var units []*ValidationUnit
	for _, r := range rs {
		unit := resourceToUnit(r)
		if unit != nil {
			units = append(units, unit)
		}
	}
	return units
}

// resourceToUnit converts a single graph resource to a validation unit.
// Returns nil if the resource's raw data doesn't match a known type.
func resourceToUnit(r *resources.Resource) *ValidationUnit {
	switch raw := r.RawData().(type) {
	case *dgModel.ModelResource:
		return &ValidationUnit{
			URN:          r.URN(),
			ResourceType: "model",
			ID:           r.ID(),
			Resource:     raw,
		}
	case *dgModel.RelationshipResource:
		return &ValidationUnit{
			URN:          r.URN(),
			ResourceType: "relationship",
			ID:           r.ID(),
			Resource:     raw,
		}
	default:
		return nil
	}
}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd cli && go build ./internal/providers/datagraph/validator/`

### Task 4: Create `planner_test.go` — Full-content struct assertions

**Files:**
- Create: `cli/internal/providers/datagraph/validator/planner_test.go`

- [ ] **Step 1: Write planner tests with full struct comparisons**

Port tests from `validations/planner_test.go`, replacing count-based assertions with full `[]*ValidationUnit` struct comparisons. Include `newTestGraph` helper.

Key changes from old tests:
- `TestPlanAll`: assert exact `plan.Units` contents with `URN`, `ResourceType`, `ID`, `Resource` fields
- `TestPlanModified`: assert exact modified unit contents
- `TestPlanSingle`: assert exact single unit contents
- All use `assert.Equal(t, expected, actual)` on full structs

- [ ] **Step 2: Run tests**

Run: `cd cli && go test ./internal/providers/datagraph/validator/ -run TestPlan -v`
Expected: All PASS

- [ ] **Step 3: Commit**

```
git add cli/internal/providers/datagraph/validator/
git commit -m "refactor(datagraph): create validator package with mode, report, and planner"
```

---

## Chunk 2: Runner, tasker, and displayers

### Task 5: Create `tasker.go` — URN-based task keys + ValidationReporter

**Files:**
- Create: `cli/internal/providers/datagraph/validator/tasker.go`

- [ ] **Step 1: Create tasker.go**

Port from `validations/tasker.go`. Changes:
- `validateTask.Id()` returns `t.unit.URN` instead of `resourceType:ID`
- Add `ValidationReporter` interface with `TaskStarted(id, description)` / `TaskCompleted(id, description, err)`
- `runValidationTasks` accepts a `ValidationReporter` and calls it inside the task closure
- Results collected by URN key

```go
package validator

import (
	"context"
	"fmt"

	dgClient "github.com/rudderlabs/rudder-iac/api/client/datagraph"
	dgModel "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/pkg/tasker"
)

const concurrency = 4

// ValidationReporter receives callbacks as validation tasks start and complete.
type ValidationReporter interface {
	TaskStarted(id string, description string)
	TaskCompleted(id string, description string, err error)
}

// noopReporter is a no-op implementation used when no interactive reporting is needed.
type noopReporter struct{}

func (noopReporter) TaskStarted(string, string)        {}
func (noopReporter) TaskCompleted(string, string, error) {}

type validateTask struct {
	unit *ValidationUnit
}

func (t *validateTask) Id() string {
	return t.unit.URN
}

func (t *validateTask) Dependencies() []string {
	return nil
}

func runValidationTasks(
	ctx context.Context,
	client dgClient.DataGraphClient,
	graph *resources.Graph,
	units []*ValidationUnit,
	reporter ValidationReporter,
) []*ResourceValidation {
	tasks := make([]tasker.Task, 0, len(units))
	for _, u := range units {
		tasks = append(tasks, &validateTask{unit: u})
	}

	results := tasker.NewResults[*ResourceValidation]()
	_ = tasker.RunTasks(ctx, tasks, concurrency, true, func(task tasker.Task) error {
		vt := task.(*validateTask)
		description := fmt.Sprintf("Validating %s %s", vt.unit.ResourceType, vt.unit.ID)
		reporter.TaskStarted(vt.Id(), description)

		result := executeValidation(ctx, client, graph, vt.unit)
		results.Store(vt.Id(), result)

		reporter.TaskCompleted(vt.Id(), description, result.Err)
		return nil
	})

	validations := make([]*ResourceValidation, 0, len(units))
	for _, u := range units {
		if r, ok := results.Get(u.URN); ok {
			validations = append(validations, r)
		}
	}

	return validations
}

func executeValidation(
	ctx context.Context,
	client dgClient.DataGraphClient,
	graph *resources.Graph,
	unit *ValidationUnit,
) *ResourceValidation {
	switch unit.ResourceType {
	case "model":
		return validateModel(ctx, client, unit)
	case "relationship":
		return validateRelationship(ctx, client, graph, unit)
	default:
		return &ResourceValidation{
			ID:           unit.ID,
			ResourceType: unit.ResourceType,
			Err:          fmt.Errorf("unknown resource type: %s", unit.ResourceType),
		}
	}
}

func validateModel(ctx context.Context, client dgClient.DataGraphClient, unit *ValidationUnit) *ResourceValidation {
	modelRes := unit.Resource.(*dgModel.ModelResource)

	req := &dgClient.ValidateModelRequest{
		AccountID: unit.AccountID,
		Type:      modelRes.Type,
		TableRef:  modelRes.Table,
		PrimaryID: modelRes.PrimaryID,
		Root:      modelRes.Root,
		Timestamp: modelRes.Timestamp,
	}

	report, err := client.ValidateModel(ctx, req)
	if err != nil {
		return &ResourceValidation{
			ID:           unit.ID,
			DisplayName:  modelRes.DisplayName,
			ResourceType: "model",
			Err:          err,
		}
	}

	return &ResourceValidation{
		ID:           unit.ID,
		DisplayName:  modelRes.DisplayName,
		ResourceType: "model",
		Issues:       report.Issues,
	}
}

func validateRelationship(ctx context.Context, client dgClient.DataGraphClient, graph *resources.Graph, unit *ValidationUnit) *ResourceValidation {
	relRes := unit.Resource.(*dgModel.RelationshipResource)

	sourceTableRef, err := resolveModelTableRef(graph, relRes.SourceModelRef)
	if err != nil {
		return &ResourceValidation{
			ID:           unit.ID,
			DisplayName:  relRes.DisplayName,
			ResourceType: "relationship",
			Err:          fmt.Errorf("resolving source model table ref: %w", err),
		}
	}

	targetTableRef, err := resolveModelTableRef(graph, relRes.TargetModelRef)
	if err != nil {
		return &ResourceValidation{
			ID:           unit.ID,
			DisplayName:  relRes.DisplayName,
			ResourceType: "relationship",
			Err:          fmt.Errorf("resolving target model table ref: %w", err),
		}
	}

	req := &dgClient.ValidateRelationshipRequest{
		AccountID:   unit.AccountID,
		Cardinality: relRes.Cardinality,
		SourceModel: dgClient.ValidationModelRef{
			TableRef: sourceTableRef,
			JoinKey:  relRes.SourceJoinKey,
		},
		TargetModel: dgClient.ValidationModelRef{
			TableRef: targetTableRef,
			JoinKey:  relRes.TargetJoinKey,
		},
	}

	report, err := client.ValidateRelationship(ctx, req)
	if err != nil {
		return &ResourceValidation{
			ID:           unit.ID,
			DisplayName:  relRes.DisplayName,
			ResourceType: "relationship",
			Err:          err,
		}
	}

	return &ResourceValidation{
		ID:           unit.ID,
		DisplayName:  relRes.DisplayName,
		ResourceType: "relationship",
		Issues:       report.Issues,
	}
}

func resolveModelTableRef(graph *resources.Graph, ref *resources.PropertyRef) (string, error) {
	if ref == nil {
		return "", fmt.Errorf("model reference is nil")
	}

	r, exists := graph.GetResource(ref.URN)
	if !exists {
		return "", fmt.Errorf("model %s not found in graph", ref.URN)
	}

	modelRes, ok := r.RawData().(*dgModel.ModelResource)
	if !ok {
		return "", fmt.Errorf("resource %s is not a model", ref.URN)
	}

	return modelRes.Table, nil
}
```

### Task 6: Create `runner.go` — Updated with Mode interface

**Files:**
- Create: `cli/internal/providers/datagraph/validator/runner.go`

- [ ] **Step 1: Create runner.go**

Port from `validations/runner.go`. Changes:
- `Run` signature: `func (r *Runner) Run(ctx context.Context, mode Mode, workspaceID string) (*ValidationReport, error)`
- Use type switch on `mode` for planning and remote state loading
- Error messages use `unit.URN`
- Returns `*ValidationReport` (renamed)
- Accepts `ValidationReporter` in constructor

```go
package validator

import (
	"context"
	"fmt"

	dgClient "github.com/rudderlabs/rudder-iac/api/client/datagraph"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	dgModel "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/differ"
)

var validationLog = logger.New("validator")

type remoteStateLoader interface {
	provider.ManagedRemoteResourceLoader
	provider.StateLoader
}

type Runner struct {
	loader   remoteStateLoader
	client   dgClient.DataGraphClient
	graph    *resources.Graph
	reporter ValidationReporter
}

func NewRunner(client dgClient.DataGraphClient, loader remoteStateLoader, graph *resources.Graph, reporter ValidationReporter) *Runner {
	if reporter == nil {
		reporter = noopReporter{}
	}
	return &Runner{
		loader:   loader,
		client:   client,
		graph:    graph,
		reporter: reporter,
	}
}

func (r *Runner) Run(ctx context.Context, mode Mode, workspaceID string) (*ValidationReport, error) {
	validationLog.Info("Starting validation run", "mode", fmt.Sprintf("%T", mode))

	var remoteGraph *resources.Graph
	if _, ok := mode.(ModeModified); ok {
		remoteResources, err := r.loader.LoadResourcesFromRemote(ctx)
		if err != nil {
			return nil, fmt.Errorf("loading remote resources: %w", err)
		}

		remoteState, err := r.loader.MapRemoteToState(remoteResources)
		if err != nil {
			return nil, fmt.Errorf("building remote state: %w", err)
		}

		remoteGraph = syncer.StateToGraph(remoteState)
	}

	plan, err := r.buildPlan(mode, remoteGraph, differ.DiffOptions{WorkspaceID: workspaceID})
	if err != nil {
		return nil, fmt.Errorf("building validation plan: %w", err)
	}

	if len(plan.Units) == 0 {
		return &ValidationReport{Status: RunStatusNoResources}, nil
	}

	validationLog.Info("Validation plan created", "units", len(plan.Units))

	if err := r.resolveAccountIDs(plan); err != nil {
		return nil, err
	}

	validations := runValidationTasks(ctx, r.client, r.graph, plan.Units, r.reporter)

	return &ValidationReport{
		Status:    RunStatusExecuted,
		Resources: validations,
	}, nil
}

func (r *Runner) buildPlan(mode Mode, remoteGraph *resources.Graph, opts differ.DiffOptions) (*ValidationPlan, error) {
	switch m := mode.(type) {
	case ModeAll:
		return PlanAll(r.graph)
	case ModeModified:
		return PlanModified(r.graph, remoteGraph, opts)
	case ModeSingle:
		return PlanSingle(r.graph, m.ResourceType, m.TargetID)
	default:
		return nil, fmt.Errorf("unknown validation mode: %T", mode)
	}
}

func (r *Runner) resolveAccountIDs(plan *ValidationPlan) error {
	cache := make(map[string]string)

	for _, unit := range plan.Units {
		dgURN := r.findDataGraphURN(unit)
		if dgURN == "" {
			return fmt.Errorf("could not determine data graph for resource %s", unit.URN)
		}

		accountID, ok := cache[dgURN]
		if !ok {
			res, exists := r.graph.GetResource(dgURN)
			if !exists {
				return fmt.Errorf("data graph %s not found in local graph", dgURN)
			}

			dgRes, ok := res.RawData().(*dgModel.DataGraphResource)
			if !ok {
				return fmt.Errorf("resource %s is not a data graph", dgURN)
			}

			accountID = dgRes.AccountID
			cache[dgURN] = accountID
		}

		unit.AccountID = accountID
	}

	return nil
}

func (r *Runner) findDataGraphURN(unit *ValidationUnit) string {
	switch unit.ResourceType {
	case "model":
		modelRes, ok := unit.Resource.(*dgModel.ModelResource)
		if ok && modelRes.DataGraphRef != nil {
			return modelRes.DataGraphRef.URN
		}
	case "relationship":
		relRes, ok := unit.Resource.(*dgModel.RelationshipResource)
		if ok && relRes.DataGraphRef != nil {
			return relRes.DataGraphRef.URN
		}
	}
	return ""
}
```

- [ ] **Step 2: Port runner_test.go and tasker_test.go**

Update for new Mode types, `ValidationReport`, URN-based keys, and `ValidationReporter` parameter.

- [ ] **Step 3: Verify tests pass**

Run: `cd cli && go test ./internal/providers/datagraph/validator/ -run "TestRunner|TestTasker" -v`

- [ ] **Step 4: Commit**

```
git add cli/internal/providers/datagraph/validator/
git commit -m "refactor(datagraph): add runner and tasker to validator package"
```

### Task 7: Create displayers — `displayer.go`, `terminal_displayer.go`, `json_displayer.go`

**Files:**
- Create: `cli/internal/providers/datagraph/validator/displayer.go`
- Create: `cli/internal/providers/datagraph/validator/terminal_displayer.go`
- Create: `cli/internal/providers/datagraph/validator/json_displayer.go`

- [ ] **Step 1: Create displayer.go — interface only**

```go
package validator

// Displayer formats and renders a validation report.
type Displayer interface {
	Display(report *ValidationReport)
}
```

- [ ] **Step 2: Create terminal_displayer.go — dynamic width**

Port terminal rendering from `display/validation.go`. Changes:
- Use `ui.GetTerminalWidth()` instead of constant `lineWidth = 80`
- Apply minimum width guard of 60
- Derive `statusColumn` as 75% of width
- All terminal-only helpers (`countBySeverity`, `countStatuses`, symbols) live here

```go
package validator

import (
	"fmt"
	"io"
	"strings"

	dgClient "github.com/rudderlabs/rudder-iac/api/client/datagraph"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
)

const (
	minTerminalWidth = 60

	symbolPass    = "✓"
	symbolWarning = "⚠"
	symbolError   = "✕"
)

// TerminalDisplayer renders validation reports as formatted terminal output.
type TerminalDisplayer struct {
	w io.Writer
}

func NewTerminalDisplayer(w io.Writer) *TerminalDisplayer {
	return &TerminalDisplayer{w: w}
}

func (d *TerminalDisplayer) terminalWidth() int {
	width := ui.GetTerminalWidth()
	if width < minTerminalWidth {
		width = minTerminalWidth
	}
	return width
}

func (d *TerminalDisplayer) Display(report *ValidationReport) {
	width := d.terminalWidth()
	statusCol := width * 3 / 4

	fmt.Fprintln(d.w)
	fmt.Fprintln(d.w, ui.Bold("Data Graph Validation Report"))
	d.printSeparator("=", width)

	models := report.ResourcesByType("model")
	relationships := report.ResourcesByType("relationship")

	if len(models) > 0 {
		d.displaySection("MODELS", models, width, statusCol)
	}

	if len(relationships) > 0 {
		d.displaySection("RELATIONSHIPS", relationships, width, statusCol)
	}

	d.displaySummary(models, relationships, width)
}

func (d *TerminalDisplayer) displaySection(title string, rvs []*ResourceValidation, width, statusCol int) {
	fmt.Fprintln(d.w)
	fmt.Fprintln(d.w, ui.Bold(title))
	d.printSeparator("-", width)

	for _, rv := range rvs {
		d.displayResource(rv, statusCol)
	}
}

func (d *TerminalDisplayer) displayResource(rv *ResourceValidation, statusCol int) {
	name := rv.DisplayName
	if name == "" {
		name = rv.ID
	}

	if rv.Err != nil {
		d.printWithPadding(
			fmt.Sprintf("  %s  %s", ui.Color(symbolError, ui.ColorRed), name),
			ui.Color("error", ui.ColorRed),
			statusCol,
		)
		fmt.Fprintf(d.w, "       %s\n", rv.Err.Error())
		return
	}

	if rv.HasErrors() {
		errorCount := countBySeverity(rv.Issues, "error")
		warningCount := countBySeverity(rv.Issues, "warning")
		status := fmt.Sprintf("%d error", errorCount)
		if errorCount > 1 {
			status += "s"
		}
		if warningCount > 0 {
			status += fmt.Sprintf("  %d warning", warningCount)
			if warningCount > 1 {
				status += "s"
			}
		}
		d.printWithPadding(
			fmt.Sprintf("  %s  %s", ui.Color(symbolError, ui.ColorRed), name),
			status,
			statusCol,
		)
		d.printIssues(rv.Issues)
		return
	}

	if rv.HasWarnings() {
		warningCount := countBySeverity(rv.Issues, "warning")
		status := fmt.Sprintf("%d warning", warningCount)
		if warningCount > 1 {
			status += "s"
		}
		d.printWithPadding(
			fmt.Sprintf("  %s  %s", ui.Color(symbolWarning, ui.ColorYellow), name),
			status,
			statusCol,
		)
		d.printIssues(rv.Issues)
		return
	}

	d.printWithPadding(
		fmt.Sprintf("  %s  %s", ui.Color(symbolPass, ui.ColorGreen), name),
		"pass",
		statusCol,
	)
}

func (d *TerminalDisplayer) printIssues(issues []dgClient.ValidationIssue) {
	for _, issue := range issues {
		color := ui.ColorYellow
		if issue.Severity == "error" {
			color = ui.ColorRed
		}
		fmt.Fprintf(d.w, "       %s: %s\n", ui.Color(issue.Rule, color), issue.Message)
	}
}

func (d *TerminalDisplayer) displaySummary(models, relationships []*ResourceValidation, width int) {
	fmt.Fprintln(d.w)
	fmt.Fprintln(d.w, ui.Bold("SUMMARY"))
	d.printSeparator("=", width)

	if len(models) > 0 {
		p, e, w := countStatuses(models)
		fmt.Fprintf(d.w, "Models:         %d passed   %d errors   %d warnings\n", p, e, w)
	}
	if len(relationships) > 0 {
		p, e, w := countStatuses(relationships)
		fmt.Fprintf(d.w, "Relationships:  %d passed   %d errors   %d warnings\n", p, e, w)
	}

	d.printSeparator("-", width)

	hasFailures := false
	for _, rv := range append(models, relationships...) {
		if rv.HasErrors() {
			hasFailures = true
			break
		}
	}

	if hasFailures {
		fmt.Fprintln(d.w, ui.Color("Result: FAILED", ui.ColorRed))
	} else {
		fmt.Fprintln(d.w, ui.Color("Result: PASSED", ui.ColorGreen))
	}

	d.printSeparator("=", width)
}

func (d *TerminalDisplayer) printSeparator(char string, width int) {
	fmt.Fprintf(d.w, "%s\n", strings.Repeat(char, width))
}

func (d *TerminalDisplayer) printWithPadding(leftText, rightText string, rightTextStart int) {
	padding := max(rightTextStart-len(leftText), 1)
	fmt.Fprintf(d.w, "%s%s%s\n", leftText, strings.Repeat(" ", padding), rightText)
}

func countBySeverity(issues []dgClient.ValidationIssue, severity string) int {
	count := 0
	for _, i := range issues {
		if i.Severity == severity {
			count++
		}
	}
	return count
}

func countStatuses(rvs []*ResourceValidation) (passed, errors, warnings int) {
	for _, rv := range rvs {
		if rv.HasErrors() {
			errors++
		} else if rv.HasWarnings() {
			warnings++
		} else {
			passed++
		}
	}
	return
}
```

- [ ] **Step 3: Create json_displayer.go**

```go
package validator

import (
	"encoding/json"
	"fmt"
	"io"
)

// JSONDisplayer renders validation reports as JSON to the provided writer.
type JSONDisplayer struct {
	w io.Writer
}

func NewJSONDisplayer(w io.Writer) *JSONDisplayer {
	return &JSONDisplayer{w: w}
}

func (d *JSONDisplayer) Display(report *ValidationReport) {
	type jsonIssue struct {
		Rule     string `json:"rule"`
		Severity string `json:"severity"`
		Message  string `json:"message"`
	}
	type jsonResource struct {
		ID           string      `json:"id"`
		DisplayName  string      `json:"displayName"`
		ResourceType string      `json:"resourceType"`
		Status       string      `json:"status"`
		Issues       []jsonIssue `json:"issues,omitempty"`
		Error        string      `json:"error,omitempty"`
	}
	type jsonOutput struct {
		Status    string         `json:"status"`
		Resources []jsonResource `json:"resources"`
	}

	out := jsonOutput{
		Status:    "executed",
		Resources: make([]jsonResource, 0, len(report.Resources)),
	}

	for _, rv := range report.Resources {
		jr := jsonResource{
			ID:           rv.ID,
			DisplayName:  rv.DisplayName,
			ResourceType: rv.ResourceType,
		}

		if rv.Err != nil {
			jr.Status = "error"
			jr.Error = rv.Err.Error()
		} else if rv.HasErrors() {
			jr.Status = "failed"
		} else if rv.HasWarnings() {
			jr.Status = "warning"
		} else {
			jr.Status = "passed"
		}

		for _, issue := range rv.Issues {
			jr.Issues = append(jr.Issues, jsonIssue{
				Rule:     issue.Rule,
				Severity: issue.Severity,
				Message:  issue.Message,
			})
		}

		out.Resources = append(out.Resources, jr)
	}

	data, _ := json.MarshalIndent(out, "", "  ")
	fmt.Fprintln(d.w, string(data))
}
```

- [ ] **Step 4: Create displayer tests with full-output assertions**

Create `cli/internal/providers/datagraph/validator/terminal_displayer_test.go` and `json_displayer_test.go`. Terminal tests use `assert.Equal` on complete buffer contents. JSON tests parse and compare structured output.

Note: For terminal displayer tests, you'll need to account for dynamic width. Either mock `GetTerminalWidth` or test at a fixed width. Since tests run in non-TTY mode, `GetTerminalWidth()` returns 80 (the default). Use that as the expected width.

- [ ] **Step 5: Verify tests pass**

Run: `cd cli && go test ./internal/providers/datagraph/validator/ -run "TestDisplay|TestTerminal|TestJSON" -v`

- [ ] **Step 6: Commit**

```
git add cli/internal/providers/datagraph/validator/
git commit -m "refactor(datagraph): add displayers to validator package"
```

---

## Chunk 3: Validator orchestrator, SilentError, command updates

### Task 8: Create `validator.go` — Top-level Validate() orchestrator

**Files:**
- Create: `cli/internal/providers/datagraph/validator/validator.go`

- [ ] **Step 1: Create validator.go**

The `Validate()` function takes interface dependencies and a mode, orchestrates the full flow. Similar to `importer.WorkspaceImport()`.

```go
package validator

import (
	"context"
	"errors"
	"fmt"
	"io"

	dgClient "github.com/rudderlabs/rudder-iac/api/client/datagraph"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
)

var ErrValidationFailed = errors.New("one or more validations failed")

// ValidatorProvider abstracts the provider methods needed by the validator.
type ValidatorProvider interface {
	provider.ManagedRemoteResourceLoader
	provider.StateLoader
}

// Project abstracts the project methods needed by the validator.
type Project interface {
	ResourceGraph() (*resources.Graph, error)
}

// Config holds all configuration needed by the Validate function.
type Config struct {
	Mode        Mode
	WorkspaceID string
	JSONOutput  bool
	Writer      io.Writer
}

// Validate orchestrates data graph validation: builds a plan, runs validations,
// and displays results. Returns ErrValidationFailed if any resource has errors.
func Validate(
	ctx context.Context,
	project Project,
	dgClient dgClient.DataGraphClient,
	dgProvider ValidatorProvider,
	cfg Config,
) error {
	graph, err := project.ResourceGraph()
	if err != nil {
		return fmt.Errorf("getting resource graph: %w", err)
	}

	var (
		displayer Displayer
		reporter  ValidationReporter
	)

	if cfg.JSONOutput {
		displayer = NewJSONDisplayer(cfg.Writer)
		reporter = noopReporter{}
	} else {
		displayer = NewTerminalDisplayer(cfg.Writer)
		reporter = newProgressReporter()
	}

	runner := NewRunner(dgClient, dgProvider, graph, reporter)
	report, err := runner.Run(ctx, cfg.Mode, cfg.WorkspaceID)
	if err != nil {
		return fmt.Errorf("running validations: %w", err)
	}

	if report.Status == RunStatusNoResources {
		if !cfg.JSONOutput {
			ui.Println("No resources to validate")
		}
		return nil
	}

	displayer.Display(report)

	if report.HasFailures() {
		return ErrValidationFailed
	}

	return nil
}

// newProgressReporter creates a TaskReporter-backed reporter for interactive mode.
// Returns a noopReporter if not running in a terminal.
func newProgressReporter() ValidationReporter {
	if !ui.IsTerminal() {
		return noopReporter{}
	}
	return &progressReporter{}
}

// progressReporter wraps ui.TaskReporter for validation progress display.
type progressReporter struct {
	reporter *ui.TaskReporter
	started  bool
}

func (r *progressReporter) start(total int) {
	r.reporter = ui.NewTaskReporter(total)
	r.started = true
	go r.reporter.Run() //nolint:errcheck
}

func (r *progressReporter) TaskStarted(id string, description string) {
	if r.reporter != nil {
		r.reporter.Start(id, description)
	}
}

func (r *progressReporter) TaskCompleted(id string, description string, err error) {
	if r.reporter != nil {
		r.reporter.Complete(id, description, err)
	}
}

func (r *progressReporter) done() {
	if r.reporter != nil {
		r.reporter.Done()
	}
}
```

Note: The `progressReporter` needs to be started before tasks run and stopped after. Update `runner.go`'s `Run` method to call `start(total)` and `done()` on the reporter if it implements a `Lifecycle` interface, OR handle it in `Validate()` by wrapping around the `runner.Run()` call. Simpler approach: have `Validate()` call start/done directly:

In `validator.go`, replace the runner call with:

```go
	if pr, ok := reporter.(*progressReporter); ok {
		// Start must be called before RunTasks, and done must be called after
		// The runner will call TaskStarted/TaskCompleted during execution
		pr.start(/* need plan size */)
		defer pr.done()
	}
```

Since we don't know the plan size until the runner builds it, the cleaner approach is to have the Runner expose the total and let `Validate()` manage the reporter lifecycle. Alternatively, move reporter start/done into `Runner.Run()`. Choose the approach that keeps the runner self-contained:

In `runner.go`, `Run()`, after building the plan:

```go
	// Start progress reporting if the reporter supports lifecycle
	if pr, ok := r.reporter.(*progressReporter); ok {
		pr.start(len(plan.Units))
		defer pr.done()
	}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd cli && go build ./internal/providers/datagraph/validator/`

### Task 9: Add `SilentError` to `root.go`

**Files:**
- Modify: `cli/internal/cmd/root.go`

- [ ] **Step 1: Add SilentError type and update Execute()**

```go
// SilentError wraps an error that should cause a non-zero exit code without
// printing an error message to stderr. This is useful for commands that produce
// structured output (e.g., JSON) where the output already contains all failure
// information and an additional stderr message would be redundant or disruptive
// to machine-readable output.
//
// Usage: return &SilentError{Err: someErr} from a command's RunE when the
// command has already communicated the failure through its primary output channel.
type SilentError struct {
	Err error
}

func (e *SilentError) Error() string {
	return e.Err.Error()
}

func (e *SilentError) Unwrap() error {
	return e.Err
}
```

Update `Execute()`:

```go
// Execute runs the root command. If the command returns an error, it is printed
// to stderr and the process exits with code 1. Errors wrapped in SilentError
// skip the stderr output — the command is expected to have already communicated
// the failure through its primary output (e.g., JSON to stdout).
func Execute() {
	defer recovery()

	if err := rootCmd.Execute(); err != nil {
		var silent *SilentError
		if !errors.As(err, &silent) {
			ui.PrintError(err)
		}
		os.Exit(1)
	}
}
```

Add `"errors"` to imports.

- [ ] **Step 2: Verify it compiles**

Run: `cd cli && go build ./internal/cmd/`

### Task 10: Update `datagraph.go` — Hidden command

**Files:**
- Modify: `cli/internal/cmd/datagraph/datagraph.go`

- [ ] **Step 1: Set Hidden: true**

Add `Hidden: true` to the cobra.Command struct.

- [ ] **Step 2: Update root.go initConfig() to unhide**

Capture the datagraph command variable at package level (like `debugCmd`), add in `initConfig()`:

```go
if config.GetConfig().ExperimentalFlags.DataGraph {
	datagraphCmd.Hidden = false
}
```

### Task 11: Update `validate.go` — Slim command

**Files:**
- Modify: `cli/internal/cmd/datagraph/validate/validate.go`

- [ ] **Step 1: Rewrite validate command**

The command should only:
1. Validate flags (PreRunE)
2. Initialize deps and load project (PreRunE)
3. Determine mode from flags/args
4. Call `validator.Validate()`
5. If `--json` and `ErrValidationFailed`, wrap in `SilentError`

```go
package validate

import (
	"context"
	"errors"
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"

	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/validator"
)

var validateLog = logger.New("datagraph", logger.Attr{
	Key:   "cmd",
	Value: "validate",
})

func NewCmdValidate() *cobra.Command {
	var (
		deps       app.Deps
		p          project.Project
		err        error
		location   string
		all        bool
		modified   bool
		jsonOutput bool
	)

	cmdDef := &cobra.Command{
		Use:   "validate [type] [id]",
		Short: "Validate data graph resources",
		Long: heredoc.Doc(`
			Validates data graph resources (models and relationships) against the warehouse.

			Checks include table existence, column existence, type compatibility, and more.
			You can validate all resources, only modified ones, or a specific resource by type and ID.
		`),
		Example: heredoc.Doc(`
			# Validate all resources
			$ rudder-cli data-graphs validate --all

			# Validate only modified resources
			$ rudder-cli data-graphs validate --modified

			# Validate a specific model
			$ rudder-cli data-graphs validate model my-model-id

			# Validate a specific relationship
			$ rudder-cli data-graphs validate relationship my-relationship-id

			# Output as JSON
			$ rudder-cli data-graphs validate --all --json
		`),
		PreRunE: func(c *cobra.Command, args []string) error {
			cfg := config.GetConfig()
			if !cfg.ExperimentalFlags.DataGraph {
				return fmt.Errorf("data-graphs commands require the experimental flag 'data_graph' to be enabled in your configuration")
			}

			if err := validateFlags(args, all, modified); err != nil {
				return err
			}

			deps, err = app.NewDeps()
			if err != nil {
				return fmt.Errorf("initialising dependencies: %w", err)
			}

			p = deps.NewProject()

			if err := p.Load(location); err != nil {
				return fmt.Errorf("loading and validating project: %w", err)
			}

			return nil
		},
		RunE: func(c *cobra.Command, args []string) error {
			defer func() {
				telemetry.TrackCommand("data-graphs validate", err, []telemetry.KV{
					{K: "location", V: location},
					{K: "all", V: all},
					{K: "modified", V: modified},
					{K: "json", V: jsonOutput},
				}...)
			}()

			validateLog.Debug("validate", "location", location, "all", all, "modified", modified, "json", jsonOutput)

			ctx := context.Background()

			workspace, err := deps.Client().Workspaces.GetByAuthToken(ctx)
			if err != nil {
				return fmt.Errorf("fetching workspace information: %w", err)
			}

			var mode validator.Mode
			if all {
				mode = validator.ModeAll{}
			} else if modified {
				mode = validator.ModeModified{}
			} else {
				mode = validator.ModeSingle{ResourceType: args[0], TargetID: args[1]}
			}

			dgProvider := deps.Providers().DataGraph

			err = validator.Validate(ctx, p, dgProvider.Client(), dgProvider, validator.Config{
				Mode:        mode,
				WorkspaceID: workspace.ID,
				JSONOutput:  jsonOutput,
				Writer:      c.OutOrStdout(),
			})

			if err != nil && jsonOutput && errors.Is(err, validator.ErrValidationFailed) {
				return &cmd.SilentError{Err: err}
			}

			return err
		},
	}

	cmdDef.Flags().StringVarP(&location, "location", "l", ".", "Path to the directory containing the project files or a specific file")
	cmdDef.Flags().BoolVar(&all, "all", false, "Validate all data graph resources in the project")
	cmdDef.Flags().BoolVar(&modified, "modified", false, "Validate only new or modified data graph resources")
	cmdDef.Flags().BoolVarP(&jsonOutput, "json", "j", false, "Output results as JSON")

	return cmdDef
}

func validateFlags(args []string, all, modified bool) error {
	modes := 0
	hasArgs := len(args) > 0
	if hasArgs {
		modes++
	}
	if all {
		modes++
	}
	if modified {
		modes++
	}

	if modes == 0 {
		return fmt.Errorf("must specify either <type> <id>, --all, or --modified")
	}
	if modes > 1 {
		return fmt.Errorf("cannot combine validation modes: specify only one of <type> <id>, --all, or --modified")
	}

	if hasArgs {
		if len(args) != 2 {
			return fmt.Errorf("expected exactly 2 arguments: <type> <id>, got %d", len(args))
		}

		resourceType := args[0]
		if resourceType != "model" && resourceType != "relationship" {
			return fmt.Errorf("invalid resource type %q: must be 'model' or 'relationship'", resourceType)
		}
	}

	return nil
}
```

Note: The `SilentError` is referenced as `cmd.SilentError` — make sure it's exported from the `cmd` package (root.go is in package `cmd`).

- [ ] **Step 2: Update validate_test.go for new structure**

- [ ] **Step 3: Verify compilation and tests**

Run: `cd cli && go build ./... && go test ./internal/cmd/datagraph/... -v`

- [ ] **Step 4: Commit**

```
git add cli/internal/providers/datagraph/validator/ cli/internal/cmd/
git commit -m "refactor(datagraph): add validator orchestrator and update command"
```

---

## Chunk 4: Delete old packages + final verification

### Task 12: Delete old packages

**Files:**
- Delete: `cli/internal/providers/datagraph/validations/` (entire directory)
- Delete: `cli/internal/providers/datagraph/display/` (entire directory)

- [ ] **Step 1: Remove old validations/ and display/ directories**

```bash
rm -rf cli/internal/providers/datagraph/validations/
rm -rf cli/internal/providers/datagraph/display/
```

- [ ] **Step 2: Verify no remaining references**

```bash
grep -r "datagraph/validations" cli/ --include="*.go"
grep -r "datagraph/display" cli/ --include="*.go"
```

Both should return no results.

- [ ] **Step 3: Run full test suite**

```bash
make lint
make build
make test
```

All must pass.

- [ ] **Step 4: Commit**

```
git commit -m "refactor(datagraph): remove old validations and display packages"
```

### Task 13: Final verification + lint

- [ ] **Step 1: Run make lint**

Run: `make lint`
Expected: PASS

- [ ] **Step 2: Run make test**

Run: `make test`
Expected: PASS

- [ ] **Step 3: Run make build**

Run: `make build`
Expected: PASS
