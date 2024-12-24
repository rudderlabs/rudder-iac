package syncer

import (
	"context"

	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/planner"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/state"
	"github.com/rudderlabs/rudder-iac/cli/pkg/logger"
)

var log = logger.New("syncer")

type ProjectSyncer struct {
	provider     Provider
	stateManager StateManager
}

type Provider interface {
	Create(ctx context.Context, ID string, resourceType string, data resources.ResourceData) (*resources.ResourceData, error)
	Update(ctx context.Context, ID string, resourceType string, data resources.ResourceData) (*resources.ResourceData, error)
	Delete(ctx context.Context, ID string, resourceType string, data resources.ResourceData) error
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
		// add any explicit dependencies, not mentioned through references
		graph.AddDependencies(resource.URN(), stateResource.Dependencies)
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

	if o.Type == planner.Delete {
		err := s.provider.Delete(ctx, r.ID(), r.Type(), dereferenced)
		if err != nil {
			return nil, err
		}

		st.RemoveResource(r.URN())
	} else {
		var f func(ctx context.Context, ID string, resourceType string, data resources.ResourceData) (*resources.ResourceData, error)
		if o.Type == planner.Create {
			f = s.provider.Create
		} else if o.Type == planner.Update {
			f = s.provider.Update
		}

		output, err := f(ctx, r.ID(), r.Type(), dereferenced)
		if err != nil {
			return nil, err
		}

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
	}

	return st, nil
}
