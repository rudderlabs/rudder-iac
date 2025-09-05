package syncer

import (
	"context"
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/differ"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/planner"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
)

type ProjectSyncer struct {
	provider SyncProvider
}

type SyncProvider interface {
	LoadState(ctx context.Context) (*state.State, error)
	PutResourceState(ctx context.Context, URN string, state *state.ResourceState) error
	DeleteResourceState(ctx context.Context, state *state.ResourceState) error
	LoadResourcesFromRemote(ctx context.Context) (*resources.ResourceCollection, error)
	LoadStateFromResources(ctx context.Context, resources *resources.ResourceCollection) (*state.State, error)
	Create(ctx context.Context, ID string, resourceType string, data resources.ResourceData) (*resources.ResourceData, error)
	Update(ctx context.Context, ID string, resourceType string, data resources.ResourceData, state resources.ResourceData) (*resources.ResourceData, error)
	Delete(ctx context.Context, ID string, resourceType string, state resources.ResourceData) error
	Import(ctx context.Context, ID string, resourceType string, data resources.ResourceData, workspaceId, remoteId string) (*resources.ResourceData, error)
}

func New(p SyncProvider) (*ProjectSyncer, error) {
	if p == nil {
		return nil, fmt.Errorf("provider is required")
	}

	return &ProjectSyncer{
		provider: p,
	}, nil
}

type SyncOptions struct {
	// DryRun is a flag to indicate if the syncer should only plan the changes, without applying them
	DryRun bool
	// Confirm is a flag to indicate if the syncer should ask for confirmation before applying the changes
	Confirm bool
}

func (s *ProjectSyncer) Sync(ctx context.Context, target *resources.Graph, options SyncOptions) error {
	errs := s.apply(ctx, target, options, false)
	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

func (s *ProjectSyncer) Destroy(ctx context.Context, options SyncOptions) []error {
	return s.apply(ctx, resources.NewGraph(), options, true)
}

func (s *ProjectSyncer) apply(ctx context.Context, target *resources.Graph, options SyncOptions, continueOnFail bool) []error {
	state, err := s.provider.LoadState(ctx)
	if err != nil {
		return []error{err}
	}

	source := stateToGraph(state)

	p := planner.New()
	plan := p.Plan(source, target)

	differ.PrintDiff(plan.Diff)

	if options.DryRun {
		return nil
	}

	if len(plan.Operations) == 0 {
		fmt.Println("No changes to apply")
		return nil
	}

	if options.Confirm {
		confirm, err := ui.Confirm("Do you want to apply these changes?")
		if err != nil {
			return []error{err}
		}

		if !confirm {
			return nil
		}
	}

	return s.executePlan(ctx, state, plan, continueOnFail)
}

func stateToGraph(state *state.State) *resources.Graph {
	graph := resources.NewGraph()

	for _, stateResource := range state.Resources {
		resource := resources.NewResource(stateResource.ID, stateResource.Type, stateResource.Input, stateResource.Dependencies)
		graph.AddResource(resource)
		// add any explicit dependencies, not mentioned through references
		graph.AddDependencies(resource.URN(), stateResource.Dependencies)
	}

	return graph
}

type OperationError struct {
	Operation *planner.Operation
	Err       error
}

func (e *OperationError) Error() string {
	return e.Err.Error()
}

func (e *OperationError) Unwrap() error {
	return e.Err
}

func (s *ProjectSyncer) executePlan(ctx context.Context, state *state.State, plan *planner.Plan, continueOnFail bool) []error {
	var (
		errors       []error
		currentState = state
	)

	for _, o := range plan.Operations {
		operationString := o.String()
		spinner := ui.NewSpinner(operationString)
		spinner.Start()

		outputState, providerErr := s.providerOperation(ctx, o, currentState)
		spinner.Stop()
		if providerErr != nil {
			fmt.Printf("%s %s\n", ui.Color("x", ui.Red), operationString)
			errors = append(errors, &OperationError{Operation: o, Err: providerErr})
			if !continueOnFail {
				return errors
			}
		}

		if outputState == nil {
			outputState = currentState
		}

		if providerErr == nil {
			fmt.Printf("%s %s\n", ui.Color("✔", ui.Green), operationString)
		}

		currentState = outputState
	}

	return errors
}

func (s *ProjectSyncer) createOperation(ctx context.Context, r *resources.Resource, st *state.State) (*state.State, error) {
	input := r.Data()
	dereferenced, err := state.Dereference(input, st)
	if err != nil {
		return nil, err
	}

	output, err := s.provider.Create(ctx, r.ID(), r.Type(), dereferenced)
	if err != nil {
		return nil, err
	}

	sr := &state.ResourceState{
		ID:           r.ID(),
		Type:         r.Type(),
		Input:        input,
		Output:       *output,
		Dependencies: r.Dependencies(),
	}

	if err := s.provider.PutResourceState(ctx, r.URN(), sr); err != nil {
		return nil, fmt.Errorf("failed to update resource state: %w", err)
	}

	st.AddResource(sr)

	return st, nil
}

func (s *ProjectSyncer) importOperation(ctx context.Context, r *resources.Resource, st *state.State) (*state.State, error) {
	input := r.Data()
	dereferenced, err := state.Dereference(input, st)
	if err != nil {
		return nil, err
	}

	output, err := s.provider.Import(ctx, r.ID(), r.Type(), dereferenced, r.ImportMetadata().WorkspaceId, r.ImportMetadata().RemoteId)
	if err != nil {
		return nil, err
	}

	sr := &state.ResourceState{
		ID:           r.ID(),
		Type:         r.Type(),
		Input:        input,
		Output:       *output,
		Dependencies: r.Dependencies(),
	}

	if err := s.provider.PutResourceState(ctx, r.URN(), sr); err != nil {
		return nil, fmt.Errorf("failed to update resource state: %w", err)
	}

	st.AddResource(sr)

	return st, nil
}

func (s *ProjectSyncer) updateOperation(ctx context.Context, r *resources.Resource, st *state.State) (*state.State, error) {
	input := r.Data()
	dereferenced, err := state.Dereference(input, st)
	if err != nil {
		return nil, err
	}

	sr := st.GetResource(r.URN())
	if sr == nil {
		return nil, fmt.Errorf("resource not found in state: %s", r.URN())
	}

	output, err := s.provider.Update(ctx, r.ID(), r.Type(), dereferenced, sr.Data())
	if err != nil {
		return nil, err
	}

	sr = &state.ResourceState{
		ID:           sr.ID,
		Type:         sr.Type,
		Input:        input,
		Output:       *output,
		Dependencies: r.Dependencies(),
	}

	if err := s.provider.PutResourceState(ctx, r.URN(), sr); err != nil {
		return nil, fmt.Errorf("failed to update resource state: %w", err)
	}

	st.AddResource(sr)

	return st, nil
}

func (s *ProjectSyncer) deleteOperation(ctx context.Context, r *resources.Resource, st *state.State) (*state.State, error) {
	sr := st.GetResource(r.URN())
	if sr == nil {
		return nil, fmt.Errorf("resource not found in state: %s", r.URN())
	}

	if err := s.provider.DeleteResourceState(ctx, sr); err != nil {
		return nil, fmt.Errorf("failed to delete resource state: %w", err)
	}

	err := s.provider.Delete(ctx, r.ID(), r.Type(), sr.Data())
	if err != nil {
		return nil, err
	}

	st.RemoveResource(r.URN())

	return st, nil
}

func (s *ProjectSyncer) providerOperation(ctx context.Context, o *planner.Operation, st *state.State) (*state.State, error) {
	r := o.Resource

	switch o.Type {
	case planner.Create:
		return s.createOperation(ctx, r, st)
	case planner.Update:
		return s.updateOperation(ctx, r, st)
	case planner.Delete:
		return s.deleteOperation(ctx, r, st)
	case planner.Import:
		return s.importOperation(ctx, r, st)
	default:
		return nil, fmt.Errorf("unknown operation type with code: %d", o.Type)
	}
}
