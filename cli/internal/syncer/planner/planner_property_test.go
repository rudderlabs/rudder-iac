package planner_test

import (
	"encoding/json"
	"fmt"
	"maps"
	"testing"

	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"

	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/planner"
)

const propertyWorkspaceID = "workspace-id"

// TestPlanner_Convergence checks that applying a plan leaves nothing to do:
// planning again against the post-apply state yields no operations. The
// post-apply state stores values as the remote returns them (JSON-decoded),
// so a type mismatch between spec and remote that re-diffs every apply fails
// this test.
func TestPlanner_Convergence(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		var (
			source = genGraph(t, "source", false)
			target = genGraph(t, "target", true)
			p      = planner.New(propertyWorkspaceID)
		)

		plan := p.Plan(source, target)
		applied := apply(source, plan)

		again := p.Plan(applied, target)
		require.Empty(t, opStrings(again.Operations), "plan after apply: %v", opStrings(plan.Operations))
	})
}

// TestPlanner_DependencyOrder checks that a resource is created, imported or
// updated only after the resources it references, and deleted only after the
// resources that reference it.
func TestPlanner_DependencyOrder(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		var (
			source = genGraph(t, "source", false)
			target = genGraph(t, "target", true)
			plan   = planner.New(propertyWorkspaceID).Plan(source, target)
			ops    = opStrings(plan.Operations)
		)

		position := map[string]int{}
		for i, op := range plan.Operations {
			position[op.Resource.URN()] = i
		}

		for i, op := range plan.Operations {
			urn := op.Resource.URN()
			if op.Type == planner.Delete {
				for _, dependent := range source.GetDependents(urn) {
					if j, ok := position[dependent]; ok {
						require.Less(t, j, i, "%s deleted before its dependent %s: %v", urn, dependent, ops)
					}
				}
				continue
			}
			for _, dep := range target.GetDependencies(urn) {
				j, ok := position[dep]
				if !ok {
					continue
				}
				require.Less(t, j, i, "%s planned before its dependency %s: %v", urn, dep, ops)
			}
		}
	})
}

// genGraph draws a DAG over a small shared ID pool so source and target
// overlap often. A resource may only reference a lower-numbered ID, which
// keeps the graph acyclic. Refs point at IDs present in the same graph,
// matching what validation guarantees for real specs.
func genGraph(t *rapid.T, label string, withImports bool) *resources.Graph {
	g := resources.NewGraph()
	var present []string

	for i := range 6 {
		id := fmt.Sprintf("r%d", i)
		if !rapid.Bool().Draw(t, label+"-has-"+id) {
			continue
		}

		data := resources.ResourceData{
			"name":  rapid.SampledFrom([]string{"a", "b"}).Draw(t, label+"-name-"+id),
			"count": rapid.IntRange(0, 2).Draw(t, label+"-count-"+id),
			"tags": []any{
				map[string]any{"key": rapid.SampledFrom([]string{"x", "y"}).Draw(t, label+"-tag-"+id)},
			},
		}
		if len(present) > 0 && rapid.Bool().Draw(t, label+"-hasref-"+id) {
			dep := rapid.SampledFrom(present).Draw(t, label+"-ref-"+id)
			data["ref"] = resources.PropertyRef{URN: resources.URN(dep, "some-type"), Property: "name"}
		}

		var opts []resources.ResourceOpts
		if withImports && rapid.Bool().Draw(t, label+"-import-"+id) {
			opts = append(opts, resources.WithResourceImportMetadata("remote-"+id, propertyWorkspaceID))
		}

		g.AddResource(resources.NewResource(id, "some-type", data, []string{}, opts...))
		present = append(present, id)
	}

	return g
}

// apply simulates a successful apply against an in-memory remote: every
// create, import and update stores the target resource as the remote echoes
// it back, and every delete removes it.
func apply(source *resources.Graph, plan *planner.Plan) *resources.Graph {
	state := map[string]*resources.Resource{}
	maps.Copy(state, source.Resources())

	for _, op := range plan.Operations {
		if op.Type == planner.Delete {
			delete(state, op.Resource.URN())
			continue
		}
		state[op.Resource.URN()] = remoteEcho(op.Resource)
	}

	g := resources.NewGraph()
	for _, r := range state {
		g.AddResource(r)
	}
	return g
}

// remoteEcho round-trips plain values through JSON, the way remote state
// arrives (ints become float64). References stay typed because state mapping
// rebuilds them before the diff.
func remoteEcho(r *resources.Resource) *resources.Resource {
	data := resources.ResourceData{}
	for k, v := range r.Data() {
		if _, isRef := v.(resources.PropertyRef); isRef {
			data[k] = v
			continue
		}
		raw, err := json.Marshal(v)
		if err != nil {
			panic(err)
		}
		var decoded any
		if err := json.Unmarshal(raw, &decoded); err != nil {
			panic(err)
		}
		data[k] = decoded
	}
	return resources.NewResource(r.ID(), r.Type(), data, []string{})
}

func opStrings(ops []*planner.Operation) []string {
	out := make([]string, 0, len(ops))
	for _, op := range ops {
		out = append(out, op.String())
	}
	return out
}
