package syncer_test

import (
	"context"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/planner"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/testutils"
	internalTestutils "github.com/rudderlabs/rudder-iac/cli/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// planWarnerProvider wraps the test DataCatalogProvider with a fixed set of
// plan-time warnings to exercise the syncer's planner.PlanWarner integration
// without depending on real domain code. Captures whether PlanWarnings was
// called so we can prove the syncer wired it before ReportPlan.
type planWarnerProvider struct {
	*internalTestutils.DataCatalogProvider
	warnings           []string
	planWarningsCalled int
}

func (p *planWarnerProvider) PlanWarnings(_ *planner.Plan) []string {
	p.planWarningsCalled++
	return p.warnings
}

// TestSyncer_PlanWarner_PopulatesPlanAndAppearsOnDryRun proves that a provider
// implementing planner.PlanWarner has its warnings copied onto plan.Warnings
// and that this happens during dry-run too — i.e. the user sees the v1
// column-metadata "row will remain" advisory without actually applying
// changes.
func TestSyncer_PlanWarner_PopulatesPlanAndAppearsOnDryRun(t *testing.T) {
	warner := &planWarnerProvider{
		DataCatalogProvider: &internalTestutils.DataCatalogProvider{
			InitialState:       state.EmptyState(),
			ReconstructedState: state.EmptyState(),
		},
		warnings: []string{
			"metadata for created_at will remain in the workspace; v1 has no clear/delete path",
		},
	}

	mockReporter := testutils.NewMockReporter()
	s, err := syncer.New(
		warner,
		mockWorkspace(),
		syncer.WithReporter(mockReporter),
		syncer.WithDryRun(true),
	)
	require.NoError(t, err)

	target := resources.NewGraph()
	// Add at least one resource so the plan is not entirely empty.
	target.AddResource(internalTestutils.NewMockEvent("event1", resources.ResourceData{
		"name": "Test Event",
	}))

	require.NoError(t, s.Sync(context.Background(), target))

	require.Equal(t, 1, warner.planWarningsCalled, "PlanWarnings should be invoked once per apply")
	require.Len(t, mockReporter.ReportPlanCalls, 1)
	assert.Equal(t,
		[]string{"metadata for created_at will remain in the workspace; v1 has no clear/delete path"},
		mockReporter.ReportPlanCalls[0].Warnings,
		"warnings must be carried on the plan handed to ReportPlan",
	)
	// Dry-run must not execute any operations.
	assert.Empty(t, warner.OperationLog, "no provider mutations on dry-run")
}

// TestSyncer_PlanWarner_AlsoAppearsOnRealApply guards the apply path: the
// same advisory must be visible when the user runs without --dry-run, BEFORE
// any resource mutation kicks off, since the orphan really will persist.
func TestSyncer_PlanWarner_AlsoAppearsOnRealApply(t *testing.T) {
	warner := &planWarnerProvider{
		DataCatalogProvider: &internalTestutils.DataCatalogProvider{
			InitialState:       state.EmptyState(),
			ReconstructedState: state.EmptyState(),
		},
		warnings: []string{
			"metadata for email will remain in the workspace; v1 has no clear/delete path",
		},
	}

	mockReporter := testutils.NewMockReporter()
	s, err := syncer.New(
		warner,
		mockWorkspace(),
		syncer.WithReporter(mockReporter),
	)
	require.NoError(t, err)

	target := resources.NewGraph()
	target.AddResource(internalTestutils.NewMockEvent("event1", resources.ResourceData{
		"name": "Test Event",
	}))

	require.NoError(t, s.Sync(context.Background(), target))

	require.Equal(t, 1, warner.planWarningsCalled, "PlanWarnings should be invoked once per apply")
	require.Len(t, mockReporter.ReportPlanCalls, 1)
	assert.Equal(t,
		[]string{"metadata for email will remain in the workspace; v1 has no clear/delete path"},
		mockReporter.ReportPlanCalls[0].Warnings,
		"warnings must be carried on the plan even on real apply",
	)
	// Real apply still performs the mutation; the warning is advisory, not blocking.
	assert.Len(t, warner.OperationLog, 1, "real apply must still execute the operation")
}

// TestSyncer_PlanWarner_NotImplementedLeavesPlanUnannotated guards the
// type-assertion guard: providers that don't implement PlanWarner must not
// trigger panics, errors, or spurious Warnings on the plan.
func TestSyncer_PlanWarner_NotImplementedLeavesPlanUnannotated(t *testing.T) {
	provider := &internalTestutils.DataCatalogProvider{
		InitialState:       state.EmptyState(),
		ReconstructedState: state.EmptyState(),
	}

	mockReporter := testutils.NewMockReporter()
	s, err := syncer.New(
		provider,
		mockWorkspace(),
		syncer.WithReporter(mockReporter),
		syncer.WithDryRun(true),
	)
	require.NoError(t, err)

	target := resources.NewGraph()
	target.AddResource(internalTestutils.NewMockEvent("event1", resources.ResourceData{
		"name": "Test Event",
	}))

	require.NoError(t, s.Sync(context.Background(), target))

	require.Len(t, mockReporter.ReportPlanCalls, 1)
	assert.Nil(t, mockReporter.ReportPlanCalls[0].Warnings, "providers that don't implement PlanWarner contribute no warnings")
}
