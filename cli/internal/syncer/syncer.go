package syncer

import (
	"context"
	"fmt"
	"sync"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/planner"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/reporters"
	"github.com/rudderlabs/rudder-iac/cli/pkg/tasker"
)

type ProjectSyncer struct {
	provider        SyncProvider
	reporter        SyncReporter
	workspace       *client.Workspace
	stateMutex      sync.RWMutex
	concurrency     int
	dryRun          bool
	askConfirmation bool
}

type SyncProvider interface {
	provider.ManagedRemoteResourceLoader
	provider.StateLoader
	provider.LifecycleManager
	provider.ConsolidateSyncer
}

func New(p SyncProvider, workspace *client.Workspace, options ...Option) (*ProjectSyncer, error) {
	if p == nil {
		return nil, fmt.Errorf("provider is required")
	}
	if workspace == nil {
		return nil, fmt.Errorf("workspace is required")
	}

	syncer := &ProjectSyncer{
		provider:        p,
		workspace:       workspace,
		concurrency:     1,
		dryRun:          false,
		askConfirmation: false,
	}

	for _, option := range options {
		if err := option(syncer); err != nil {
			return nil, err
		}
	}

	if syncer.reporter == nil {
		syncer.reporter = &reporters.PlainSyncReporter{}
	}

	return syncer, nil
}

type Option func(*ProjectSyncer) error

func WithReporter(reporter SyncReporter) Option {
	return func(s *ProjectSyncer) error {
		if reporter == nil {
			return fmt.Errorf("reporter cannot be nil")
		}
		s.reporter = reporter
		return nil
	}
}

func WithConcurrency(concurrency int) Option {
	return func(s *ProjectSyncer) error {
		if concurrency < 1 {
			return fmt.Errorf("concurrency must be at least 1, got %d", concurrency)
		}
		s.concurrency = concurrency
		return nil
	}
}

func WithDryRun(dryRun bool) Option {
	return func(s *ProjectSyncer) error {
		s.dryRun = dryRun
		return nil
	}
}

func WithAskConfirmation(askConfirmation bool) Option {
	return func(s *ProjectSyncer) error {
		s.askConfirmation = askConfirmation
		return nil
	}
}

type SyncReporter interface {
	ReportPlan(plan *planner.Plan)
	AskConfirmation() (bool, error)
	SyncStarted(totalTasks int)
	SyncCompleted()
	TaskStarted(taskId string, description string)
	TaskCompleted(taskId string, description string, err error)
}

func (s *ProjectSyncer) Sync(ctx context.Context, target *resources.Graph) error {
	errs := s.apply(ctx, target, false)
	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

func (s *ProjectSyncer) Destroy(ctx context.Context) []error {
	return s.apply(ctx, resources.NewGraph(), true)
}

func (s *ProjectSyncer) apply(ctx context.Context, target *resources.Graph, continueOnFail bool) []error {
	fmt.Println("0")
	resources, err := s.provider.LoadResourcesFromRemote(ctx)
	if err != nil {
		return []error{err}
	}
	fmt.Println("1")
	state, err := s.provider.MapRemoteToState(resources)
	if err != nil {
		return []error{err}
	}
	source := StateToGraph(state)
	fmt.Println("2")
	p := planner.New(s.workspace.ID)
	plan := p.Plan(source, target)
	fmt.Println("3")
	s.reporter.ReportPlan(plan)

	if s.dryRun {
		return nil
	}
	fmt.Println("4")

	if len(plan.Operations) == 0 {
		fmt.Println("No changes to apply")
		return nil
	}

	if s.askConfirmation {
		confirm, err := s.reporter.AskConfirmation()
		if err != nil {
			return []error{err}
		}

		if !confirm {
			return nil
		}
	}

	// Execute the plan (create/update/delete operations)
	errors := s.executePlan(ctx, state, plan, target, continueOnFail)

	// If plan execution failed, don't proceed with consolidation
	if len(errors) > 0 {
		return errors
	}

	// Consolidate sync: providers can perform batch operations or multi-resource
	// coordination after all individual resources have been processed
	if err := s.provider.ConsolidateSync(ctx, state); err != nil {
		return []error{fmt.Errorf("consolidate sync failed: %w", err)}
	}

	return nil
}

func StateToGraph(state *state.State) *resources.Graph {
	graph := resources.NewGraph()

	for _, stateResource := range state.Resources {
		resource := resources.NewResource(
			stateResource.ID,
			stateResource.Type,
			stateResource.Input,
			stateResource.Dependencies,
			resources.WithRawData(stateResource.InputRaw),
		)
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

func (s *ProjectSyncer) executePlan(ctx context.Context, state *state.State, plan *planner.Plan, target *resources.Graph, continueOnFail bool) []error {
	if s.concurrency > 1 {
		return s.executePlanConcurrently(ctx, state, plan, target, continueOnFail)
	} else {
		return s.executePlanSequentially(ctx, state, plan, continueOnFail)
	}
}

func (s *ProjectSyncer) executePlanSequentially(ctx context.Context, state *state.State, plan *planner.Plan, continueOnFail bool) []error {
	var errors []error

	s.reporter.SyncStarted(len(plan.Operations))

	for _, o := range plan.Operations {
		operationString := o.String()

		s.reporter.TaskStarted(o.Resource.URN(), operationString)
		providerErr := s.providerOperation(ctx, o, state)
		s.reporter.TaskCompleted(o.Resource.URN(), operationString, providerErr)
		if providerErr != nil {
			errors = append(errors, &OperationError{Operation: o, Err: providerErr})
			if !continueOnFail {
				return errors
			}
		}
	}

	s.reporter.SyncCompleted()

	return errors
}

func (s *ProjectSyncer) executePlanConcurrently(ctx context.Context, state *state.State, plan *planner.Plan, target *resources.Graph, continueOnFail bool) []error {
	tasks := make([]tasker.Task, 0, len(plan.Operations))

	sourceGraph := StateToGraph(state)
	for _, o := range plan.Operations {
		tasks = append(tasks, newOperationTask(o, sourceGraph, target))
	}

	s.reporter.SyncStarted(len(tasks))

	taskErrors := tasker.RunTasks(ctx, tasks, s.concurrency, continueOnFail, func(task tasker.Task) error {
		opTask, ok := task.(*operationTask)
		if !ok {
			return fmt.Errorf("invalid task type: %T", task)
		}
		o := opTask.operation
		operationString := o.String()
		s.reporter.TaskStarted(task.Id(), operationString)
		providerErr := s.providerOperation(ctx, o, state)
		s.reporter.TaskCompleted(task.Id(), operationString, providerErr)
		if providerErr != nil {
			return &OperationError{Operation: o, Err: providerErr}
		}

		return nil
	})

	s.reporter.SyncCompleted()

	return taskErrors
}

func (s *ProjectSyncer) createOperation(ctx context.Context, r *resources.Resource, st *state.State) error {
	input := r.Data()
	var output *resources.ResourceData
	var sr *state.ResourceState

	if r.RawData() != nil {
		s.stateMutex.RLock()
		err := state.DereferenceByReflection(r.RawData(), st)
		s.stateMutex.RUnlock()
		if err != nil {
			return err
		}

		outputRaw, err := s.provider.CreateRaw(ctx, r)
		if err != nil {
			return err
		}

		sr = &state.ResourceState{
			ID:           r.ID(),
			Type:         r.Type(),
			Input:        input,
			InputRaw:     r.RawData(),
			OutputRaw:    outputRaw,
			Dependencies: r.Dependencies(),
		}
	} else {
		s.stateMutex.RLock()
		dereferenced, err := state.Dereference(input, st)
		s.stateMutex.RUnlock()
		if err != nil {
			return err
		}

		output, err = s.provider.Create(ctx, r.ID(), r.Type(), dereferenced)
		if err != nil {
			return err
		}

		sr = &state.ResourceState{
			ID:           r.ID(),
			Type:         r.Type(),
			Input:        input,
			Output:       *output,
			Dependencies: r.Dependencies(),
		}
	}

	s.stateMutex.Lock()
	st.AddResource(sr)
	s.stateMutex.Unlock()
	return nil

}

func (s *ProjectSyncer) importOperation(ctx context.Context, r *resources.Resource, st *state.State) error {
	input := r.Data()
	var output *resources.ResourceData
	var sr *state.ResourceState

	if r.RawData() != nil {
		s.stateMutex.RLock()
		err := state.DereferenceByReflection(r.RawData(), st)
		s.stateMutex.RUnlock()
		if err != nil {
			return err
		}

		outputRaw, err := s.provider.ImportRaw(ctx, r, r.ImportMetadata().RemoteId)
		if err != nil {
			return err
		}

		sr = &state.ResourceState{
			ID:           r.ID(),
			Type:         r.Type(),
			Input:        input,
			InputRaw:     r.RawData(),
			OutputRaw:    outputRaw,
			Dependencies: r.Dependencies(),
		}
	} else {
		s.stateMutex.RLock()
		dereferenced, err := state.Dereference(input, st)
		s.stateMutex.RUnlock()
		if err != nil {
			return err
		}

		output, err = s.provider.Import(ctx, r.ID(), r.Type(), dereferenced, r.ImportMetadata().RemoteId)
		if err != nil {
			return err
		}

		sr = &state.ResourceState{
			ID:           r.ID(),
			Type:         r.Type(),
			Input:        input,
			Output:       *output,
			Dependencies: r.Dependencies(),
		}
	}

	s.stateMutex.Lock()
	st.AddResource(sr)
	s.stateMutex.Unlock()
	return nil
}

func (s *ProjectSyncer) updateOperation(ctx context.Context, r *resources.Resource, st *state.State) error {
	input := r.Data()
	var output *resources.ResourceData

	sr := st.GetResource(r.URN())
	if sr == nil {
		return fmt.Errorf("resource not found in state: %s", r.URN())
	}

	if r.RawData() != nil {
		s.stateMutex.RLock()
		err := state.DereferenceByReflection(r.RawData(), st)
		s.stateMutex.RUnlock()
		if err != nil {
			return err
		}

		outputRaw, err := s.provider.UpdateRaw(ctx, r, sr.InputRaw, sr.OutputRaw)
		if err != nil {
			return err
		}

		sr = &state.ResourceState{
			ID:           sr.ID,
			Type:         sr.Type,
			Input:        input,
			InputRaw:     r.RawData(),
			OutputRaw:    outputRaw,
			Dependencies: r.Dependencies(),
		}
	} else {
		s.stateMutex.RLock()
		dereferenced, err := state.Dereference(input, st)
		s.stateMutex.RUnlock()
		if err != nil {
			return err
		}
		output, err = s.provider.Update(ctx, r.ID(), r.Type(), dereferenced, sr.Data())
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

	if r.RawData() != nil {
		err := s.provider.DeleteRaw(ctx, r.ID(), r.Type(), sr.InputRaw, sr.OutputRaw)
		if err != nil {
			return err
		}
	} else {
		err := s.provider.Delete(ctx, r.ID(), r.Type(), sr.Data())
		if err != nil {
			return err
		}
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
