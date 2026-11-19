package connection

import (
	"testing"

	retlConnection "github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/connection"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInheritedFanInRule(t *testing.T) {
	for _, tt := range []struct {
		name             string
		first, second    *resources.Resource
		warning, problem bool
	}{
		{"distinct imported connection IDs", fanInResource("one", "users-model", "remote-1", "ws-1"), fanInResource("two", "orders-model", "remote-2", "ws-1"), true, false},
		{"duplicate connection ID is bogus", fanInResource("one", "users-model", "remote-1", "ws-1"), fanInResource("two", "orders-model", "remote-1", "ws-1"), false, true},
		{"missing import", fanInResource("one", "users-model", "remote-1", "ws-1"), fanInResource("two", "orders-model", "", ""), false, true},
		{"wrong workspace", fanInResource("one", "users-model", "remote-1", "ws-1"), fanInResource("two", "orders-model", "remote-2", "ws-2"), false, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			graph := resources.NewGraph()
			addSQLModelResource(graph, postgresModel())
			orders := postgresModel()
			orders.id = "orders-model"
			addSQLModelResource(graph, orders)
			addDestinationResource(graph, httpDestination())
			graph.AddResource(tt.first)
			graph.AddResource(tt.second)
			ctx := semanticContext(validRawSpec(), graph)
			ctx.WorkspaceID = "ws-1"
			warning := NewInheritedFanInRule()
			assert.Equal(t, rules.Warning, warning.Severity())
			got := warning.Validate(ctx)
			if tt.warning {
				require.Len(t, got, 1)
				assert.Equal(t, "/spec/connections/0/destination", got[0].Reference)
				assert.Contains(t, got[0].Message, "each importing a distinct connection")
				assert.Contains(t, got[0].Message, "inherited fan-in")
				assert.NotContains(t, got[0].Message, "remote-1", "no single ID represents the group")
			} else {
				assert.Empty(t, got)
			}
			errors := NewConnectionSemanticValidRule(newTestRegistry(t)).Validate(ctx)
			if tt.problem {
				require.NotEmpty(t, errors)
				assert.Contains(t, errors[0].Message, "can only receive from one rETL source")
			} else {
				assert.Empty(t, errors)
			}
			ctx.WorkspaceID = "ws-2"
			assert.Empty(t, warning.Validate(ctx), "other workspace must not inherit")
			ctx.WorkspaceID = ""
			assert.Empty(t, warning.Validate(ctx), "unknown workspace must fail closed")
			assert.NotEmpty(t, NewConnectionSemanticValidRule(newTestRegistry(t)).Validate(ctx))
		})
	}
}

func TestScopedFanInRuleUsesTypedDispatchAndDecode(t *testing.T) {
	graph := resources.NewGraph()
	addSQLModelResource(graph, postgresModel())
	addDestinationResource(graph, httpDestination())
	ctx := semanticContext(validRawSpec(), graph)
	ctx.WorkspaceID = "ws-1"
	ctx.Version = "rudder/v2"
	assert.Empty(t, NewConnectionSemanticValidRule(newTestRegistry(t)).Validate(ctx))
	assert.Empty(t, NewInheritedFanInRule().Validate(ctx))

	ctx.Version = "rudder/v1"
	ctx.Spec = map[string]any{"connections": "invalid"}
	for _, rule := range []rules.Rule{NewConnectionSemanticValidRule(newTestRegistry(t)), NewInheritedFanInRule()} {
		got := rule.Validate(ctx)
		require.Len(t, got, 1)
		assert.Equal(t, "/spec", got[0].Reference)
		assert.Contains(t, got[0].Message, "failed to unmarshal spec")
	}
}

func TestInheritedFanInThreeDistinctSources(t *testing.T) {
	graph := resources.NewGraph()
	for _, id := range []string{"users-model", "orders-model", "products-model"} {
		model := postgresModel()
		model.id = id
		addSQLModelResource(graph, model)
	}
	addDestinationResource(graph, httpDestination())
	graph.AddResource(fanInResource("one", "users-model", "connection-1", "ws-1"))
	graph.AddResource(fanInResource("two", "orders-model", "connection-2", "ws-1"))
	graph.AddResource(fanInResource("three", "products-model", "connection-3", "ws-1"))
	ctx := semanticContext(validRawSpec(), graph)
	ctx.WorkspaceID = "ws-1"
	got := NewInheritedFanInRule().Validate(ctx)
	require.Len(t, got, 1)
	assert.Contains(t, got[0].Message, "each importing a distinct connection")
	assert.Empty(t, NewConnectionSemanticValidRule(newTestRegistry(t)).Validate(ctx))

	// A partial import does not grandfather any member of the group.
	graph.AddResource(fanInResource("three", "products-model", "", ""))
	assert.Empty(t, NewInheritedFanInRule().Validate(ctx))
	errors := NewConnectionSemanticValidRule(newTestRegistry(t)).Validate(ctx)
	require.NotEmpty(t, errors)
	assert.Contains(t, errors[0].Message, "can only receive from one rETL source")
}

func TestFanInKeepsIndependentTopologyErrors(t *testing.T) {
	graph := resources.NewGraph()
	addSQLModelResource(graph, postgresModel())
	orders := postgresModel()
	orders.id = "orders-model"
	addSQLModelResource(graph, orders)
	addDestinationResource(graph, httpDestination())
	graph.AddResource(fanInResource("one", "users-model", "remote-1", "ws-1"))
	graph.AddResource(fanInResource("two", "orders-model", "remote-1", "ws-1"))
	// V-C3 must still reject a duplicate pair in an inherited group.
	graph.AddResource(fanInResource("duplicate", "users-model", "remote-1", "ws-1"))
	raw := validRawSpec()
	first := rawConnection(raw)
	second := map[string]any{}
	for key, value := range first {
		second[key] = value
	}
	second["id"] = "duplicate"
	raw["connections"] = []any{first, second}
	ctx := semanticContext(raw, graph)
	ctx.WorkspaceID = "ws-1"
	got := NewConnectionSemanticValidRule(newTestRegistry(t)).Validate(ctx)
	require.NotEmpty(t, got)
	assert.Contains(t, got[0].Message, "connected more than once")
}

func fanInResource(id, source, remoteID, workspace string) *resources.Resource {
	var opts []resources.ResourceOpts
	if remoteID != "" {
		opts = append(opts, resources.WithResourceImportMetadata(remoteID, workspace))
	}
	return resources.NewResource(id, retlConnection.ResourceType, resources.ResourceData{
		retlConnection.SourceKey:      &resources.PropertyRef{URN: sqlModelURN(source), Property: "id"},
		retlConnection.DestinationKey: &resources.PropertyRef{URN: destinationResourceURN("my-http-destination"), Property: "id"},
	}, nil, opts...)
}
