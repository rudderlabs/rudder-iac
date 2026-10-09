package syncer_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/planner"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/testutils"
)

type refusingProvider struct {
	mockRawProvider
	err error
}

func (p *refusingProvider) CheckPlan(context.Context, *planner.Plan, *state.State) error { return p.err }

// A provider that refuses the plan stops the run before the plan is shown and
// before any resource is created, dry run or not.
func TestSync_PlanCheckerRefusalStopsBeforeAnyChange(t *testing.T) {
	for _, dryRun := range []bool{true, false} {
		t.Run(fmt.Sprintf("dryRun=%t", dryRun), func(t *testing.T) {
			reporter := testutils.NewMockReporter()
			p := &refusingProvider{mockRawProvider: mockRawProvider{initialState: state.EmptyState()}, err: errors.New("refused")}
			target := resources.NewGraph()
			target.AddResource(resources.NewResource("p", "mock-parent", resources.ResourceData{"name": "p"}, nil))

			s, err := syncer.New(p, mockWorkspace(), syncer.WithReporter(reporter), syncer.WithDryRun(dryRun))
			require.NoError(t, err)

			err = s.Sync(context.Background(), target)

			assert.EqualError(t, err, "refused")
			assert.Empty(t, reporter.ReportPlanCalls, "the plan must not be shown")
			assert.Nil(t, p.capturedCreateData, "nothing may be created")
		})
	}
}
