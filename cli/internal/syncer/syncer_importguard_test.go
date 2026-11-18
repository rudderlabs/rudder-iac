package syncer_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/importreport"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/planner"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/testutils"
)

type guardingProvider struct {
	mockRawProvider
	err          error
	reportSeen   *importreport.Report
	checkedCount int
}

func (p *guardingProvider) CheckImports(_ context.Context, _ *planner.Plan, _ *state.State, report *importreport.Report) error {
	p.checkedCount++
	p.reportSeen = report
	return p.err
}

// A provider that refuses an unacknowledged adopt stops the run before the plan
// is shown and before any resource is created, dry run or not, and sees the
// report the syncer was given.
func TestSync_ImportGuardRefusalStopsBeforeAnyChange(t *testing.T) {
	for _, dryRun := range []bool{true, false} {
		t.Run(fmt.Sprintf("dryRun=%t", dryRun), func(t *testing.T) {
			reporter := testutils.NewMockReporter()
			report := &importreport.Report{Entries: []importreport.Entry{{RemoteID: "d1"}}}
			p := &guardingProvider{mockRawProvider: mockRawProvider{initialState: state.EmptyState()}, err: errors.New("unacknowledged")}
			target := resources.NewGraph()
			target.AddResource(resources.NewResource("p", "mock-parent", resources.ResourceData{"name": "p"}, nil))

			s, err := syncer.New(p, mockWorkspace(),
				syncer.WithReporter(reporter), syncer.WithDryRun(dryRun), syncer.WithImportReport(report))
			require.NoError(t, err)

			err = s.Sync(context.Background(), target)

			assert.EqualError(t, err, "unacknowledged")
			assert.Equal(t, 1, p.checkedCount)
			assert.Same(t, report, p.reportSeen)
			assert.Empty(t, reporter.ReportPlanCalls, "the plan must not be shown")
			assert.Nil(t, p.capturedCreateData, "nothing may be created")
		})
	}
}
