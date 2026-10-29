package syncer_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type breakTestProvider struct {
	mockRawProvider
	mu               sync.Mutex
	calls            []string
	fail             map[string]error
	started          chan struct{}
	proceed          chan struct{}
	linkNeedsPublish bool
}

func (p *breakTestProvider) CreateRaw(_ context.Context, r *resources.Resource) (any, error) {
	if p.started != nil {
		p.started <- struct{}{}
		<-p.proceed
	}
	p.mu.Lock()
	published := false
	for _, call := range p.calls {
		if call == "consolidate" {
			published = true
		}
	}
	p.calls = append(p.calls, "create "+r.ID())
	p.mu.Unlock()
	if p.linkNeedsPublish && r.ID() == "destination" && !published {
		return nil, fmt.Errorf("link returned 404: transformation is an unpublished draft")
	}
	if err := p.fail[r.ID()]; err != nil {
		return nil, err
	}
	return &mockParentState{RemoteID: "remote-" + r.ID()}, nil
}

func TestSyncPublishesTransformationBeforeDependentDestinationLink(t *testing.T) {
	p := &breakTestProvider{
		mockRawProvider:  mockRawProvider{initialState: state.EmptyState()},
		linkNeedsPublish: true,
	}
	graph := breakTestGraph("transformation", "destination")
	graph.AddDependencies(resources.URN("destination", "mock-parent"), []string{resources.URN("transformation", "mock-parent")})
	s, err := syncer.New(p, mockWorkspace())
	require.NoError(t, err)

	err = s.Sync(context.Background(), graph)
	require.NoError(t, err)
	assert.Equal(t, []string{"create transformation", "consolidate", "create destination", "consolidate"}, p.calls)
}

func (p *breakTestProvider) ConsolidateSync(_ context.Context, _ *resources.Graph, _ *state.State) error {
	p.mu.Lock()
	p.calls = append(p.calls, "consolidate")
	p.mu.Unlock()
	return nil
}

func breakTestGraph(ids ...string) *resources.Graph {
	graph := resources.NewGraph()
	for _, id := range ids {
		graph.AddResource(resources.NewResource(id, "mock-parent", resources.ResourceData{"name": id}, nil,
			resources.WithRawData(&mockParentData{ID: id, Name: id})))
	}
	return graph
}

func TestSyncReportsEveryConcurrentOperationFailure(t *testing.T) {
	first, second := errors.New("first backend failure"), errors.New("second backend failure")
	p := &breakTestProvider{
		mockRawProvider: mockRawProvider{initialState: state.EmptyState()},
		fail:            map[string]error{"first": first, "second": second},
		started:         make(chan struct{}, 2), proceed: make(chan struct{}),
	}
	s, err := syncer.New(p, mockWorkspace(), syncer.WithConcurrency(2))
	require.NoError(t, err)
	go func() {
		<-p.started
		<-p.started
		close(p.proceed)
	}()
	err = s.Sync(context.Background(), breakTestGraph("first", "second"))
	require.Error(t, err)
	assert.ErrorIs(t, err, first)
	assert.ErrorIs(t, err, second)
}

func TestSyncFinalizesCreatedDraftAfterLaterFailure(t *testing.T) {
	p := &breakTestProvider{
		mockRawProvider: mockRawProvider{initialState: state.EmptyState()},
		fail:            map[string]error{"destination": fmt.Errorf("link returned 404")},
	}
	graph := breakTestGraph("transformation", "destination")
	graph.AddDependencies(resources.URN("destination", "mock-parent"), []string{resources.URN("transformation", "mock-parent")})
	s, err := syncer.New(p, mockWorkspace())
	require.NoError(t, err)
	err = s.Sync(context.Background(), graph)
	require.ErrorContains(t, err, "link returned 404")
	assert.Equal(t, []string{"create transformation", "create destination", "consolidate"}, p.calls,
		"a failed link must not strand a draft for the next apply")
}

// An apply after a partial import has no durable indication that the resource
// was adopted in an earlier failed process. Default apply therefore needs an
// explicit destructive opt-in before pruning absent managed resources.
func TestSyncDoesNotPruneAbsentResourcesByDefault(t *testing.T) {
	initial := state.EmptyState()
	initial.AddResource(&state.ResourceState{
		ID: "adopted", Type: "mock-parent",
		Input:     resources.ResourceData{"name": "adopted"},
		InputRaw:  &mockParentData{ID: "adopted", Name: "adopted"},
		OutputRaw: &mockParentState{RemoteID: "remote-adopted"},
	})
	p := &mockRawProvider{initialState: initial}
	reporter := testutils.NewMockReporter()
	s, err := syncer.New(p, mockWorkspace(), syncer.WithReporter(reporter))
	require.NoError(t, err)

	err = s.Sync(context.Background(), resources.NewGraph())
	require.NoError(t, err)
	assert.Nil(t, p.capturedDeleteData, "pruning a spec after failed import must not delete the adopted remote resource")
	require.Len(t, reporter.ReportPlanCalls, 1)
	assert.Empty(t, reporter.ReportPlanCalls[0].Diff.RemovedResources, "a partial apply must not plan unrelated deletions")
}
