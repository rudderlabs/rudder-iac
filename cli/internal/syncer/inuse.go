package syncer

import (
	"fmt"
	"sort"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/planner"
)

// inUseReference is one surviving resource still naming a resource the plan
// deletes.
type inUseReference struct {
	deletedURN string
	byURN      string
	field      string
}

// guardInUseDeletes refuses a plan that deletes a resource another resource in
// the project still points at by its raw remote id.
//
// A reference written as "#account:snf-test" is a dependency the graph models,
// and dropping its target is caught at validate time. A reference written as a
// raw id — account_id, destination_id — is deliberately opaque to the graph, so
// nothing connected the two and the delete went through silently, leaving the
// source pointing at an account that no longer exists (DEX-959).
//
// ponytail: compares against every string in the surviving resources' data
// rather than asking each type which of its fields hold ids. A control-plane id
// appearing verbatim in another resource IS a reference to it, whatever the
// field is called, so the general scan costs less than a per-type registry and
// misses less. Swap it for a declared-reference lookup if a type ever stores an
// unrelated value that collides with an id.
//
// This is a project-local check, and deliberately not a guarantee: a source
// created in the webapp, outside this project, can hold the same reference and
// is invisible here. The backend refusing the delete is the guarantee; this
// catches the case the CLI can see, before the call.
func guardInUseDeletes(plan *planner.Plan, st *state.State, target *resources.Graph) error {
	if plan == nil || st == nil || target == nil {
		return nil
	}

	// Remote id -> URN, for the resources this plan deletes. A resource whose
	// state we cannot see contributes nothing to check against.
	deletedByRemoteID := map[string]string{}
	for _, op := range plan.Operations {
		if op.Type != planner.Delete {
			continue
		}
		sr, ok := st.Resources[op.Resource.URN()]
		if !ok || sr.ID == "" {
			continue
		}
		deletedByRemoteID[sr.ID] = op.Resource.URN()
	}
	if len(deletedByRemoteID) == 0 {
		return nil
	}

	var found []inUseReference
	for urn, r := range target.Resources() {
		walkStrings(r.Data(), "", func(field, value string) {
			deletedURN, isDeleted := deletedByRemoteID[value]
			if !isDeleted {
				return
			}
			found = append(found, inUseReference{deletedURN: deletedURN, byURN: urn, field: field})
		})
	}
	if len(found) == 0 {
		return nil
	}

	sort.Slice(found, func(i, j int) bool {
		if found[i].deletedURN != found[j].deletedURN {
			return found[i].deletedURN < found[j].deletedURN
		}
		if found[i].byURN != found[j].byURN {
			return found[i].byURN < found[j].byURN
		}
		return found[i].field < found[j].field
	})

	var b strings.Builder
	b.WriteString("refusing to delete resources the project still references:\n")
	for _, f := range found {
		fmt.Fprintf(&b, "  - %s is still referenced by %s (%s)\n", f.deletedURN, f.byURN, f.field)
	}
	b.WriteString("\nRemove or repoint the reference, or keep the resource's spec in the project.")
	return fmt.Errorf("%s", b.String())
}

// walkStrings visits every string leaf in a resource's data, reporting a
// dotted path so the message can name the field that holds the reference.
func walkStrings(value any, path string, visit func(field, value string)) {
	switch v := value.(type) {
	case string:
		if path != "" {
			visit(path, v)
		}
	case map[string]any:
		for k, item := range v {
			walkStrings(item, joinPath(path, k), visit)
		}
	case resources.ResourceData:
		for k, item := range v {
			walkStrings(item, joinPath(path, k), visit)
		}
	case []any:
		for i, item := range v {
			walkStrings(item, fmt.Sprintf("%s[%d]", path, i), visit)
		}
	}
}

func joinPath(path, key string) string {
	if path == "" {
		return key
	}
	return path + "." + key
}
