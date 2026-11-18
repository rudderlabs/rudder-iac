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
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/differ"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/planner"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/testutils"
)

type previewingProvider struct {
	mockRawProvider
	previewed []string
	err       error
	imported  int
}

func (p *previewingProvider) ImportRaw(context.Context, *resources.Resource, string) (any, error) {
	p.imported++
	return nil, nil
}

func (p *previewingProvider) PreviewImports(_ context.Context, plan *planner.Plan, _ *state.State) (map[string]differ.ResourceDiff, error) {
	for _, op := range plan.Operations {
		p.previewed = append(p.previewed, op.String())
	}
	if p.err != nil {
		return nil, p.err
	}
	return map[string]differ.ResourceDiff{"mock-parent:p": {URN: "mock-parent:p", Diffs: map[string]differ.PropertyDiff{
		"name": {Property: "name", SourceValue: "remote", TargetValue: "p"},
	}}}, nil
}

// An importable resource's preview reaches the plan that is shown, so a dry
// run lists the changes the import would write.
func TestSync_ImportPreviewReachesTheShownPlan(t *testing.T) {
	reporter := testutils.NewMockReporter()
	p := &previewingProvider{mockRawProvider: mockRawProvider{initialState: state.EmptyState()}}
	target := resources.NewGraph()
	target.AddResource(resources.NewResource("p", "mock-parent", resources.ResourceData{"name": "p"}, nil,
		resources.WithResourceImportMetadata("remote-p", mockWorkspace().ID)))

	s, err := syncer.New(p, mockWorkspace(), syncer.WithReporter(reporter), syncer.WithDryRun(true))
	require.NoError(t, err)
	require.NoError(t, s.Sync(context.Background(), target))

	assert.Equal(t, []string{"Import mock-parent:p"}, p.previewed)
	require.Len(t, reporter.ReportPlanCalls, 1)
	assert.Equal(t, map[string]differ.ResourceDiff{"mock-parent:p": {URN: "mock-parent:p", Diffs: map[string]differ.PropertyDiff{
		"name": {Property: "name", SourceValue: "remote", TargetValue: "p"},
	}}}, reporter.ReportPlanCalls[0].Diff.ImportableDiffs)
}

// A preview that fails stops the run before the plan is shown and before any
// resource is imported or created, dry run or not: an adopt whose changes
// could not be shown must not run.
func TestSync_ImportPreviewFailureStopsBeforeAnyChange(t *testing.T) {
	for _, dryRun := range []bool{true, false} {
		t.Run(fmt.Sprintf("dryRun=%t", dryRun), func(t *testing.T) {
			reporter := testutils.NewMockReporter()
			p := &previewingProvider{mockRawProvider: mockRawProvider{initialState: state.EmptyState()}, err: errors.New("previewing imports for provider d: boom")}
			target := resources.NewGraph()
			target.AddResource(resources.NewResource("p", "mock-parent", resources.ResourceData{"name": "p"}, nil,
				resources.WithResourceImportMetadata("remote-p", mockWorkspace().ID)))
			target.AddResource(resources.NewResource("q", "mock-parent", resources.ResourceData{"name": "q"}, nil))

			s, err := syncer.New(p, mockWorkspace(), syncer.WithReporter(reporter), syncer.WithDryRun(dryRun))
			require.NoError(t, err)

			err = s.Sync(context.Background(), target)

			assert.EqualError(t, err, "previewing imports for provider d: boom")
			assert.Empty(t, reporter.ReportPlanCalls, "the plan must not be shown")
			assert.Zero(t, p.imported, "nothing may be imported")
			assert.Nil(t, p.capturedCreateData, "nothing may be created")
		})
	}
}
