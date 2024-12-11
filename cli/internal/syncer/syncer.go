package syncer

import (
	"context"

	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/planner"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/state"
)

type ProjectSyncer struct {
	provider     Provider
	stateManager StateManager
}

type Provider interface {
	Create(ctx context.Context, ID string, resourceType string, data resources.ResourceData) *resources.ResourceData
	Update(ctx context.Context, ID string, resourceType string, data resources.ResourceData) *resources.ResourceData
	Delete(ctx context.Context, ID string, resourceType string, data resources.ResourceData) *resources.ResourceData
}

type StateManager interface {
	Load(context.Context) (*state.State, error)
	Save(context.Context, *state.State) error
}

func New(p Provider, sm StateManager) *ProjectSyncer {
	return &ProjectSyncer{
		provider:     p,
		stateManager: sm,
	}
}

func (s *ProjectSyncer) Sync(ctx context.Context, target *resources.Graph) error {
	state, err := s.stateManager.Load(ctx)
	if err != nil {
		return err
	}

	source := stateToGraph(state)

	p := planner.New()
	plan := p.Plan(source, target)

	outputState, err := s.executePlan(ctx, state, plan)
	if err != nil {
		return err
	}

	if err = s.stateManager.Save(ctx, outputState); err != nil {
		return err
	}

	return nil
}

func stateToGraph(state *state.State) *resources.Graph {
	graph := resources.NewGraph()

	for _, stateResource := range state.Resources {
		resource := resources.NewResource(stateResource.ID, stateResource.Type, stateResource.Input)
		graph.AddResource(resource)
	}

	return graph
}

func (s *ProjectSyncer) executePlan(ctx context.Context, state *state.State, plan *planner.Plan) (*state.State, error) {
	currentState := state
	for _, o := range plan.Operations {
		outputState, err := s.providerOperation(ctx, o, currentState)
		if err != nil {
			return nil, err
		}

		currentState = outputState
	}

	return currentState, nil
}

func (s *ProjectSyncer) providerOperation(ctx context.Context, o *planner.Operation, st *state.State) (*state.State, error) {
	r := o.Resource
	input := r.Data()
	dereferenced, err := state.Dereference(input, st)
	if err != nil {
		return nil, err
	}

	var f func(ctx context.Context, ID string, resourceType string, data resources.ResourceData) *resources.ResourceData

	switch o.Type {
	case planner.Create:
		f = s.provider.Create
	case planner.Update:
		f = s.provider.Update
	case planner.Delete:
		f = s.provider.Delete
	}

	output := f(ctx, r.ID(), r.Type(), dereferenced)

	sr := st.GetResource(r.URN())
	if sr == nil {
		sr = &state.StateResource{
			ID:   r.ID(),
			Type: r.Type(),
		}
	}

	sr.Input = input
	sr.Output = *output

	st.AddResource(sr)

	return st, nil
}
