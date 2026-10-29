package destination

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	ttypes "github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/types"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/planner"
)

func destinationOp(opType planner.OperationType, id, transformationID string) *planner.Operation {
	data := &DestinationResource{ID: id, Type: "webhook"}
	if transformationID != "" {
		data.Transformation = createTransformationRef(resources.URN(transformationID, ttypes.TransformationResourceType))
	}
	return &planner.Operation{
		Type:     opType,
		Resource: resources.NewResource(id, DestinationResourceType, resources.ResourceData{}, nil, resources.WithRawData(data)),
	}
}

func transformationOp(opType planner.OperationType, id string) *planner.Operation {
	return &planner.Operation{
		Type:     opType,
		Resource: resources.NewResource(id, ttypes.TransformationResourceType, resources.ResourceData{}, nil),
	}
}

// A new transformation is published only at the end of the apply that creates
// it, and the API links a destination only to a published transformation, so
// linking one in the same apply would fail on every apply. The plan is refused
// before anything is applied instead.
func TestCheckPlanRefusesLinkToTransformationCreatedInSameApply(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		ops         []*planner.Operation
		wantRefused []string
	}{
		{
			name: "new destination linked to a new transformation",
			ops: []*planner.Operation{
				transformationOp(planner.Create, "tx-1"),
				destinationOp(planner.Create, "dest-1", "tx-1"),
			},
			wantRefused: []string{"destination:dest-1 links #transformation:tx-1"},
		},
		{
			name: "existing destination switched to a new transformation",
			ops: []*planner.Operation{
				transformationOp(planner.Create, "tx-2"),
				destinationOp(planner.Update, "dest-1", "tx-2"),
			},
			wantRefused: []string{"destination:dest-1 links #transformation:tx-2"},
		},
		{
			name: "imported destination linked to a new transformation",
			ops: []*planner.Operation{
				transformationOp(planner.Create, "tx-1"),
				destinationOp(planner.Import, "dest-1", "tx-1"),
			},
			wantRefused: []string{"destination:dest-1 links #transformation:tx-1"},
		},
		{
			name: "every refused destination is named",
			ops: []*planner.Operation{
				transformationOp(planner.Create, "tx-1"),
				transformationOp(planner.Create, "tx-2"),
				destinationOp(planner.Create, "dest-1", "tx-1"),
				destinationOp(planner.Update, "dest-2", "tx-2"),
			},
			wantRefused: []string{
				"destination:dest-1 links #transformation:tx-1",
				"destination:dest-2 links #transformation:tx-2",
			},
		},
		{
			name: "destination linked to a transformation updated in the same apply",
			ops: []*planner.Operation{
				transformationOp(planner.Update, "tx-1"),
				destinationOp(planner.Create, "dest-1", "tx-1"),
			},
		},
		{
			name: "destination linked to a transformation the plan leaves alone",
			ops: []*planner.Operation{
				destinationOp(planner.Update, "dest-1", "tx-1"),
			},
		},
		{
			name: "destination without a transformation",
			ops: []*planner.Operation{
				transformationOp(planner.Create, "tx-1"),
				destinationOp(planner.Create, "dest-1", ""),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := NewProvider(nil, nil).CheckPlan(context.Background(), &planner.Plan{Operations: tt.ops}, state.EmptyState())

			if len(tt.wantRefused) == 0 {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			for _, refused := range tt.wantRefused {
				assert.ErrorContains(t, err, refused)
			}
			assert.ErrorContains(t, err, "leaving the destination's transformation as it is, then link it in a second apply")
		})
	}
}
