package syncer

import (
	"context"
	"fmt"
	"slices"
	"sync"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/differ"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/planner"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
)

type ProjectSyncer struct {
	provider   SyncProvider
	workspace  *client.Workspace
	stateMutex sync.RWMutex
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

func New(p SyncProvider, workspace *client.Workspace) (*ProjectSyncer, error) {
	if p == nil {
		return nil, fmt.Errorf("provider is required")
	}
	if workspace == nil {
		return nil, fmt.Errorf("workspace is required")
	}

	return &ProjectSyncer{
		provider:  p,
		workspace: workspace,
	}, nil
}

type SyncOptions struct {
	// DryRun is a flag to indicate if the syncer should only plan the changes, without applying them
	DryRun bool
	// Confirm is a flag to indicate if the syncer should ask for confirmation before applying the changes
	Confirm bool
	// Concurrency is the number of concurrent operations to run
	Concurrency int
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
	resources, err := s.provider.LoadResourcesFromRemote(ctx)
	if err != nil {
		return []error{err}
	}

	state, err := s.provider.LoadStateFromResources(ctx, resources)
	if err != nil {
		return []error{err}
	}
	source := StateToGraph(state)

	p := planner.New(s.workspace.ID)
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

	return s.executePlan(ctx, state, plan, target, continueOnFail, options)
}

func StateToGraph(state *state.State) *resources.Graph {
	graph := resources.NewGraph()

	for _, stateResource := range state.Resources {
		resource := resources.NewResource(stateResource.ID, stateResource.Type, stateResource.Input, stateResource.Dependencies)
		graph.AddResource(resource)
		// add any explicit dependencies, not mentioned through references
		graph.AddDependencies(resource.URN(), stateResource.Dependencies)
	}

	return graph
}

func removeStateForResourceTypes(state *state.State, resourceTypes []string) *state.State {
	// loop over all resources in the state and remove a resource if it matches any of the resource types
	for _, resource := range state.Resources {
		if slices.Contains(resourceTypes, resource.Type) {
			delete(state.Resources, resources.URN(resource.ID, resource.Type))
		}
	}
	return state
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

func (s *ProjectSyncer) executePlan(ctx context.Context, state *state.State, plan *planner.Plan, target *resources.Graph, continueOnFail bool, options SyncOptions) []error {
	if config.GetConfig().ExperimentalFlags.ConcurrentSyncs {
		return s.executePlanConcurrently(ctx, state, plan, target, continueOnFail, options)
	}
	return s.executePlanSequentially(ctx, state, plan, continueOnFail)
}

func (s *ProjectSyncer) executePlanSequentially(ctx context.Context, state *state.State, plan *planner.Plan, continueOnFail bool) []error {
	var errors []error

	for _, o := range plan.Operations {
		operationString := o.String()
		spinner := ui.NewSpinner(operationString)
		spinner.Start()

		providerErr := s.providerOperation(ctx, o, state)
		spinner.Stop()
		if providerErr != nil {
			fmt.Printf("%s %s\n", ui.Color("x", ui.Red), operationString)
			errors = append(errors, &OperationError{Operation: o, Err: providerErr})
			if !continueOnFail {
				return errors
			}
		}

		if providerErr == nil {
			fmt.Printf("%s %s\n", ui.Color("✔", ui.Green), operationString)
		}
	}

	return errors
}

func (s *ProjectSyncer) executePlanConcurrently(ctx context.Context, state *state.State, plan *planner.Plan, target *resources.Graph, continueOnFail bool, options SyncOptions) []error {
	tasks := make([]Task, 0, len(plan.Operations))
	sourceGraph := StateToGraph(state)
	for _, o := range plan.Operations {
		tasks = append(tasks, newOperationTask(o, sourceGraph, target))
	}
	return RunTasks(ctx, tasks, options.Concurrency, continueOnFail, func(task Task) error {
		opTask, ok := task.(*operationTask)
		if !ok {
			return fmt.Errorf("invalid task type: %T", task)
		}
		o := opTask.operation
		operationString := o.String()
		spinner := ui.NewSpinner(operationString)
		spinner.Start()
		providerErr := s.providerOperation(ctx, o, state)
		spinner.Stop()
		if providerErr != nil {
			fmt.Printf("%s %s\n", ui.Color("x", ui.Red), operationString)
			return &OperationError{Operation: o, Err: providerErr}
		}
		fmt.Printf("%s %s\n", ui.Color("✔", ui.Green), operationString)
		return nil
	})
}

func (s *ProjectSyncer) createOperation(ctx context.Context, r *resources.Resource, st *state.State) error {
	input := r.Data()
	s.stateMutex.RLock()
	dereferenced, err := state.Dereference(input, st)
	s.stateMutex.RUnlock()
	if err != nil {
		return err
	}

	output, err := s.provider.Create(ctx, r.ID(), r.Type(), dereferenced)
	if err != nil {
		return err
	}

	sr := &state.ResourceState{
		ID:           r.ID(),
		Type:         r.Type(),
		Input:        input,
		Output:       *output,
		Dependencies: r.Dependencies(),
	}

	if err := s.provider.PutResourceState(ctx, r.URN(), sr); err != nil {
		return fmt.Errorf("failed to update resource state: %w", err)
	}
	s.stateMutex.Lock()
	st.AddResource(sr)
	s.stateMutex.Unlock()
	return nil
}

func (s *ProjectSyncer) importOperation(ctx context.Context, r *resources.Resource, st *state.State) error {
	input := r.Data()
	s.stateMutex.RLock()
	dereferenced, err := state.Dereference(input, st)
	s.stateMutex.RUnlock()
	if err != nil {
		return err
	}

	output, err := s.provider.Import(ctx, r.ID(), r.Type(), dereferenced, r.ImportMetadata().WorkspaceId, r.ImportMetadata().RemoteId)
	if err != nil {
		return err
	}

	sr := &state.ResourceState{
		ID:           r.ID(),
		Type:         r.Type(),
		Input:        input,
		Output:       *output,
		Dependencies: r.Dependencies(),
	}

	if err := s.provider.PutResourceState(ctx, r.URN(), sr); err != nil {
		return fmt.Errorf("failed to update resource state: %w", err)
	}
	s.stateMutex.Lock()
	st.AddResource(sr)
	s.stateMutex.Unlock()
	return nil
}

func (s *ProjectSyncer) updateOperation(ctx context.Context, r *resources.Resource, st *state.State) error {
	input := r.Data()
	s.stateMutex.RLock()
	dereferenced, err := state.Dereference(input, st)
	s.stateMutex.RUnlock()
	if err != nil {
		return err
	}

	sr := st.GetResource(r.URN())
	if sr == nil {
		return fmt.Errorf("resource not found in state: %s", r.URN())
	}

	output, err := s.provider.Update(ctx, r.ID(), r.Type(), dereferenced, sr.Data())
	if err != nil {
		return err
	}

	sr = &state.ResourceState{
		ID:           sr.ID,
		Type:         sr.Type,
		Input:        input,
		Output:       *output,
		Dependencies: r.Dependencies(),
	}

	if err := s.provider.PutResourceState(ctx, r.URN(), sr); err != nil {
		return fmt.Errorf("failed to update resource state: %w", err)
	}
	s.stateMutex.Lock()
	st.AddResource(sr)
	s.stateMutex.Unlock()

	return nil
}

func (s *ProjectSyncer) deleteOperation(ctx context.Context, r *resources.Resource, st *state.State) error {
	s.stateMutex.RLock()
	sr := st.GetResource(r.URN())
	s.stateMutex.RUnlock()
	if sr == nil {
		return fmt.Errorf("resource not found in state: %s", r.URN())
	}

	if err := s.provider.DeleteResourceState(ctx, sr); err != nil {
		return fmt.Errorf("failed to delete resource state: %w", err)
	}

	err := s.provider.Delete(ctx, r.ID(), r.Type(), sr.Data())
	if err != nil {
		return err
	}
	s.stateMutex.Lock()
	st.RemoveResource(r.URN())
	s.stateMutex.Unlock()

	return nil
}

func (s *ProjectSyncer) providerOperation(ctx context.Context, o *planner.Operation, st *state.State) error {
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
		return fmt.Errorf("unknown operation type with code: %d", o.Type)
	}
}
