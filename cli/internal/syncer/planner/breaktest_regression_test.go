package planner

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanDeletesConflictingIdentityBeforeCreatingReplacement(t *testing.T) {
	for _, tt := range []struct {
		name, resourceType, displayName, eventType string
	}{
		{name: "renamed ID with same name", resourceType: "event", displayName: "Order Placed", eventType: "track"},
		{name: "page event singleton", resourceType: "event", displayName: "", eventType: "page"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			source, target := resources.NewGraph(), resources.NewGraph()
			data := resources.ResourceData{"name": tt.displayName, "eventType": tt.eventType}
			source.AddResource(resources.NewResource("old-id", tt.resourceType, data, nil))
			target.AddResource(resources.NewResource("new-id", tt.resourceType, data, nil))

			plan := New("ws").Plan(source, target)
			require.Len(t, plan.Operations, 2)
			assert.Equal(t, []OperationType{Delete, Create}, []OperationType{
				plan.Operations[0].Type, plan.Operations[1].Type,
			})
		})
	}
}
