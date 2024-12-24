package differ

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
)

func PrintDiff(diff *Diff) {
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
				details += fmt.Sprintf("    - %s: %v %s %v\n", ui.Color(d.Property, ui.White), d.SourceValue, ui.Color("=>", ui.Yellow), d.TargetValue)
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
		fmt.Printf("  - %s\n", ui.Color(urn, ui.White))
		if detailFn != nil {
			fmt.Printf("%s\n", detailFn(urn))
		}
	}
	fmt.Println()
}
