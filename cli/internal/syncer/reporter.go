package syncer

import (
	"fmt"
	"reflect"

	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/differ"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/planner"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
)

func NewDefaultSyncReporter() SyncReporter {
	if ui.IsTerminal() {
		return &TerminalSyncReporter{
			reporter: ui.NewTaskReporter(),
		}
	} else {
		return &PlainSyncReporter{}
	}
}

type planReporter struct{}

func (r *planReporter) ReportPlan(plan *planner.Plan) {
	printDiff(plan.Diff)
}

type TerminalSyncReporter struct {
	planReporter
	reporter *ui.TaskReporter
}

func (r *TerminalSyncReporter) AskConfirmation() (bool, error) {
	return ui.Confirm("Do you want to apply these changes?")
}

func (r *TerminalSyncReporter) SyncStarted(totalTasks int) {
	r.reporter.SetTotalTasks(totalTasks)
	go func() {
		err := r.reporter.Run()
		if err != nil {
			ui.PrintError(err)
		}
	}()
}

func (r *TerminalSyncReporter) SyncCompleted() {
	r.reporter.Done()
}

func (r *TerminalSyncReporter) TaskStarted(taskId string, description string) {
	r.reporter.Start(taskId, description)
}

func (r *TerminalSyncReporter) TaskCompleted(taskId string, description string, err error) {
	r.reporter.Complete(taskId, description, err)
}

type PlainSyncReporter struct {
	planReporter
}

// AskConfirmation in PlainSyncReporter is a no-op and always returns false, since in non-interactive environments we cannot prompt the user.
// This typically runs in CI/CD pipelines or when output is being redirected to a file,
// which assumes non-interactive execution with --config=false for the apply command.
func (r *PlainSyncReporter) AskConfirmation() (bool, error) { return false, nil }

func (r *PlainSyncReporter) TaskCompleted(taskId string, description string, err error) {
	if err != nil {
		ui.PrintFailure(description)
	} else {
		ui.PrintSuccess(description)
	}
}

func (r *PlainSyncReporter) SyncStarted(totalTasks int)                    {}
func (r *PlainSyncReporter) SyncCompleted()                                {}
func (r *PlainSyncReporter) TaskStarted(taskId string, description string) {}

func printDiff(diff *differ.Diff) {
	if len(diff.ImportableResources) > 0 {
		listResources("Importable resources", diff.ImportableResources, nil)
	}

	if len(diff.NewResources) > 0 {
		listResources("New resources", diff.NewResources, nil)
	}

	if len(diff.UpdatedResources) > 0 {
		urns := []string{}
		for _, r := range diff.UpdatedResources {
			urns = append(urns, r.URN)
		}
		listResources("Updated resources", urns, func(urn string) string {
			r := diff.UpdatedResources[urn]
			details := ""
			for _, d := range r.Diffs {
				details += fmt.Sprintf(
					"    - %s: %s %s %s\n",
					ui.Color(d.Property, ui.ColorWhite),
					printable(d.SourceValue),
					ui.Color("=>", ui.ColorYellow),
					printable(d.TargetValue),
				)
			}
			return details
		})
	}

	if len(diff.RemovedResources) > 0 {
		listResources("Removed resources", diff.RemovedResources, nil)
	}
}

func listResources(label string, resources []string, detailFn func(string) string) {
	fmt.Println(ui.Bold(label) + ":")
	for _, urn := range resources {
		fmt.Printf("  - %s\n", ui.Color(urn, ui.ColorWhite))
		if detailFn != nil {
			fmt.Printf("%s\n", detailFn(urn))
		}
	}
	fmt.Println()
}

func printable(val interface{}) string {
	if val == nil {
		return ui.Color("<nil>", ui.ColorBlue)
	}

	if reflect.ValueOf(val).Kind() == reflect.Pointer {
		return fmt.Sprintf("%v", reflect.ValueOf(val).Elem())
	}

	return fmt.Sprintf("%v", val)
}
