package reporters

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"sort"
	"strings"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/secret"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/differ"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/planner"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
)

type planReporter struct {
	Writer    io.Writer
	workspace *client.Workspace
}

func (r *planReporter) SetWriter(writer io.Writer) {
	r.Writer = writer
}

func (r *planReporter) SetWorkspace(workspace *client.Workspace) {
	r.workspace = workspace
}

func (r *planReporter) getWriter() io.Writer {
	if r.Writer == nil {
		return os.Stdout
	}
	return r.Writer
}

func (r *planReporter) ReportPlan(plan *planner.Plan) {
	renderedDiff := renderDiff(plan.Diff)
	if renderedDiff == "" {
		return
	}

	if banner := r.renderWorkspaceBanner(); banner != "" {
		fmt.Fprint(r.getWriter(), banner)
	}

	fmt.Fprint(r.getWriter(), renderedDiff)
}

func (r *planReporter) renderWorkspaceBanner() string {
	if r.workspace == nil || r.workspace.Name == "" || r.workspace.ID == "" {
		return ""
	}

	return fmt.Sprintf("Workspace: %s (%s)\n\n", r.workspace.Name, r.workspace.ID)
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
		renderUpdatedResources(b, diff)
	}

	if len(diff.RemovedResources) > 0 {
		listResources(b, "Removed resources", diff.RemovedResources, nil)
	}

	return b.String()
}

// renderUpdatedResources lists updated resources, splitting off the secret-only
// ones — which re-apply every run because their remote value can't be read — into
// their own section so they don't read as user-made drift. This is the only place
// the section layout is secret-aware; the per-line rendering stays generic.
func renderUpdatedResources(b *strings.Builder, diff *differ.Diff) {
	var updated, secretOnly []string
	for _, r := range diff.UpdatedResources {
		if r.IsSecretOnly() {
			secretOnly = append(secretOnly, r.URN)
		} else {
			updated = append(updated, r.URN)
		}
	}

	detail := func(urn string) string {
		r := diff.UpdatedResources[urn]
		details := ""
		diffKeys := make([]string, 0, len(r.Diffs))
		for k := range r.Diffs {
			diffKeys = append(diffKeys, k)
		}
		sort.Strings(diffKeys)
		for _, k := range diffKeys {
			for _, line := range renderPropertyDiff(r.Diffs[k]) {
				details += line
			}
		}
		return details
	}

	if len(updated) > 0 {
		sort.Strings(updated)
		listResources(b, "Updated resources", updated, detail)
	}
	if len(secretOnly) > 0 {
		sort.Strings(secretOnly)
		listResources(b, "Always re-applied (secret values can't be read back)", secretOnly, detail)
	}
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

func renderPropertyRef(ref *resources.PropertyRef) string {
	if ref == nil {
		return ui.Color("<nil>", ui.ColorBlue)
	}
	return ui.Color(ref.URN, ui.ColorGreen)
}

// renderSecret masks a secret value wherever it surfaces in a diff (e.g. nested in
// a map that also has real changes). It is the secret analogue of renderPropertyRef:
// the type-specific rendering lives here, dispatched from printable.
func renderSecret(s *secret.String) string {
	if s == nil {
		return ui.Color("<nil>", ui.ColorBlue)
	}
	return ui.Color(s.String(), ui.ColorWhite)
}

func printable(val any) string {
	if val == nil {
		return ui.Color("<nil>", ui.ColorBlue)
	}
	if s, ok := val.(secret.String); ok {
		return renderSecret(&s)
	}
	if s, ok := val.(*secret.String); ok {
		return renderSecret(s)
	}
	if ref, ok := val.(*resources.PropertyRef); ok {
		return renderPropertyRef(ref)
	}
	if ref, ok := val.(resources.PropertyRef); ok {
		return renderPropertyRef(&ref)
	}
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
	// A secret-only diff has no meaningful old => new — the remote value is unknown
	// — so it is rendered by the dedicated secret helper and never enters the
	// generic value/nested-diff path below.
	if diff.SecretOnly {
		return []string{renderSecretDiff(diff.Property)}
	}

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
		// if path is a bracket notation, use it as is, otherwise add a dot
		var fullPath string
		if strings.HasPrefix(path, "[") {
			fullPath = diff.Property + path
		} else {
			fullPath = diff.Property + "." + path
		}
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

// renderSecretDiff renders a property whose only change is a secret. The remote
// value is unknown, so there is no old => new to show — it will be re-applied every
// run. Mirrors renderPropertyRef: secret-specific rendering lives here, not in the
// generic line formatter.
func renderSecretDiff(property string) string {
	return fmt.Sprintf(
		"    - %s: %s\n",
		ui.Color(property, ui.ColorWhite),
		ui.Color("(secret, always re-applied)", ui.ColorYellow),
	)
}
