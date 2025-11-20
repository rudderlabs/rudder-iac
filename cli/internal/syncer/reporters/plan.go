package reporters

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/differ"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/planner"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
)

type planReporter struct{}

func (r *planReporter) ReportPlan(plan *planner.Plan) {
	ui.Print(renderDiff(plan.Diff))
}

func renderDiff(diff *differ.Diff) string {
	b := &strings.Builder{}

	if len(diff.ImportableResources) > 0 {
		listResources(b, "Importable resources", diff.ImportableResources, nil)
	}

	if len(diff.NewResources) > 0 {
		listResources(b, "New resources", diff.NewResources, nil)
	}

	if len(diff.UpdatedResources) > 0 {
		urns := []string{}
		for _, r := range diff.UpdatedResources {
			urns = append(urns, r.URN)
		}
		listResources(b, "Updated resources", urns, func(urn string) string {
			r := diff.UpdatedResources[urn]
			details := ""
			diffKeys := []string{}
			for k := range r.Diffs {
				diffKeys = append(diffKeys, k)
			}
			sort.Strings(diffKeys)
			for _, k := range diffKeys {
				d := r.Diffs[k]
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
		listResources(b, "Removed resources", diff.RemovedResources, nil)
	}

	return b.String()
}

func listResources(b *strings.Builder, label string, resources []string, detailFn func(string) string) {
	fmt.Fprintln(b, ui.Bold(label)+":")
	for _, urn := range resources {
		fmt.Fprintf(b, "  - %s\n", ui.Color(urn, ui.ColorWhite))
		if detailFn != nil {
			fmt.Fprint(b, detailFn(urn))
		}
	}
	fmt.Fprintln(b)
}

func printable(val any) string {
	if val == nil {
		return ui.Color("<nil>", ui.ColorBlue)
	}

	if reflect.ValueOf(val).Kind() == reflect.Pointer {
		return fmt.Sprintf("%v", reflect.ValueOf(val).Elem())
	}

	return fmt.Sprintf("%v", val)
}
