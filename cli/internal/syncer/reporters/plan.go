package reporters

import (
	"fmt"
	"reflect"

	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/differ"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/planner"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
)

type planReporter struct{}

func (r *planReporter) ReportPlan(plan *planner.Plan) {
	printDiff(plan.Diff)
}

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
