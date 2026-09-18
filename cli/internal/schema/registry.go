// Package schema generates JSON Schema (Draft 2020-12) for rudder-cli spec
// kinds directly from the typed Go spec structs each provider parses into.
//
// The registry below is the single link between a spec kind and its Go type.
// Because schemas are reflected from the live production structs, adding or
// changing a field on a spec struct is reflected in the emitted schema with no
// hand-authored JSON to keep in sync. Adding a new kind is a single registry
// entry pointing at the struct — never a hand-written schema file.
package schema

import (
	"sort"

	esSource "github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream/source"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/sqlmodel"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
)

// kindType associates a spec kind with the Go type of its top-level `spec:`
// block — i.e. the value the owning handler decodes the YAML `spec` map into.
type kindType struct {
	kind   string
	sample any
}

// registry is the ordered source-of-truth list of kinds and their spec-block
// types. Only kinds whose handler decodes into a reflectable, tagged struct are
// listed; kinds parsed manually into opaque maps (e.g. the experimental
// data-graph kinds) are intentionally omitted rather than emitting a
// meaningless schema.
var registry = []kindType{
	{kind: localcatalog.KindProperties, sample: localcatalog.PropertySpecV1{}},
	{kind: localcatalog.KindEvents, sample: localcatalog.EventSpecV1{}},
	{kind: localcatalog.KindCategories, sample: localcatalog.CategorySpecV1{}},
	{kind: localcatalog.KindCustomTypes, sample: localcatalog.CustomTypeSpecV1{}},
	{kind: localcatalog.KindTrackingPlansV1, sample: trackingPlanSpec{}},
	{kind: sqlmodel.ResourceKind, sample: sqlmodel.SQLModelSpec{}},
	{kind: esSource.ResourceKind, sample: esSource.SourceSpec{}},
	{kind: "transformation", sample: specs.TransformationSpec{}},
	{kind: "transformation-library", sample: specs.TransformationLibrarySpec{}},
}

// trackingPlanSpec is the top-level `spec:` shape for the tracking-plan kind.
// The handler decodes the spec block directly into a TrackingPlanV1, so this
// wrapper simply embeds it to present the same reflected fields the parser sees.
type trackingPlanSpec = localcatalog.TrackingPlanV1

// Kinds returns the spec kinds the schema package can generate, sorted for
// deterministic output.
func Kinds() []string {
	out := make([]string, 0, len(registry))
	for _, e := range registry {
		out = append(out, e.kind)
	}
	sort.Strings(out)
	return out
}

// sampleFor returns the spec-block sample value for a kind, if registered.
func sampleFor(kind string) (any, bool) {
	for _, e := range registry {
		if e.kind == kind {
			return e.sample, true
		}
	}
	return nil, false
}
