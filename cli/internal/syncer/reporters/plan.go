package reporters

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"sort"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/differ"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/planner"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
)

type planReporter struct {
	Writer io.Writer
}

func (r *planReporter) SetWriter(writer io.Writer) {
	r.Writer = writer
}

func (r *planReporter) getWriter() io.Writer {
	if r.Writer == nil {
		return os.Stdout
	}
	return r.Writer
}

func (r *planReporter) ReportPlan(plan *planner.Plan) {
	fmt.Fprint(r.getWriter(), renderDiff(plan.Diff))
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
		sort.Strings(urns)
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
				diffLines := renderPropertyDiff(d)
				for _, line := range diffLines {
					details += line
				}
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

	// Handle PropertyRef specially
	if ref, ok := val.(resources.PropertyRef); ok {
		return ui.Color(ref.URN, ui.ColorGreen)
	}

	// Handle pointer to PropertyRef
	if reflect.ValueOf(val).Kind() == reflect.Pointer {
		v := reflect.ValueOf(val).Elem()
		if v.IsValid() {
			if ref, ok := v.Interface().(resources.PropertyRef); ok {
				return ui.Color(ref.URN, ui.ColorGreen)
			}
			return fmt.Sprintf("%v", v)
		}
	}

	return fmt.Sprintf("%v", val)
}

// renderPropertyDiff renders a single property diff, either as a flat diff or expanded nested diff
func renderPropertyDiff(diff differ.PropertyDiff) []string {
	// If nested diff printing is disabled, use old behavior
	useNestedDiffPrinting := config.GetConfig().ExperimentalFlags.NestedDiffs
	if !useNestedDiffPrinting {
		line := formattedLine(diff.Property, ValuePair{Source: diff.SourceValue, Target: diff.TargetValue})
		return []string{line}
	}

	// Compute nested diffs
	nestedDiffs := ComputeNestedDiffs(diff.SourceValue, diff.TargetValue)

	// If no diffs found (values are equal), return empty
	if len(nestedDiffs) == 0 {
		return []string{}
	}

	// If only one diff and it's at the root level (empty path), use old behavior
	if len(nestedDiffs) == 1 {
		if pair, hasRoot := nestedDiffs[""]; hasRoot {
			line := formattedLine(diff.Property, pair)
			return []string{line}
		}
	}

	// Multiple nested diffs - format each with full path
	lines := []string{}

	// Sort paths for consistent output
	paths := make([]string, 0, len(nestedDiffs))
	for path := range nestedDiffs {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	for _, path := range paths {
		pair := nestedDiffs[path]
		fullPath := diff.Property + "." + path
		line := formattedLine(fullPath, pair)
		lines = append(lines, line)
	}
	return lines
}

func formattedLine(property string, pair ValuePair) string {
	return fmt.Sprintf(
		"    - %s: %s %s %s\n",
		ui.Color(property, ui.ColorWhite),
		printable(pair.Source),
		ui.Color("=>", ui.ColorYellow),
		printable(pair.Target),
	)
}
