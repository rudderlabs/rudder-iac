package validator

import (
	"context"
	"fmt"

	dgClient "github.com/rudderlabs/rudder-iac/api/client/datagraph"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	dgModel "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/differ"
)

var validationLog = logger.New("validator")

// reporterLifecycle is an optional interface for reporters that need
// to be notified about the start and end of a validation run.
type reporterLifecycle interface {
	start(total int)
	done()
}

// remoteStateLoader abstracts provider methods needed by the runner
type remoteStateLoader interface {
	provider.ManagedRemoteResourceLoader
	provider.StateLoader
}

// Runner orchestrates data graph validation
type Runner struct {
	loader      remoteStateLoader
	client      dgClient.DataGraphClient
	graph       *resources.Graph
	reporter    ValidationReporter
	concurrency int
}

// NewRunner creates a new validation runner.
// When concurrency is 0, it defaults to 1.
func NewRunner(client dgClient.DataGraphClient, loader remoteStateLoader, graph *resources.Graph, reporter ValidationReporter, concurrency int) *Runner {
	if reporter == nil {
		reporter = noopReporter{}
	}
	if concurrency <= 0 {
		concurrency = 1
	}
	return &Runner{
		loader:      loader,
		client:      client,
		graph:       graph,
		reporter:    reporter,
		concurrency: concurrency,
	}
}

// Run executes validation based on the specified mode and returns a report
func (r *Runner) Run(ctx context.Context, mode Mode, workspaceID string) (*ValidationReport, error) {
	validationLog.Info("Starting validation run", "mode", fmt.Sprintf("%T", mode))

	var (
		plan *ValidationPlan
		err  error
	)

	switch m := mode.(type) {
	case ModeAll:
		plan, err = PlanAll(r.graph)
	case ModeModified:
		remoteGraph, loadErr := r.loadRemoteGraph(ctx)
		if loadErr != nil {
			return nil, loadErr
		}
		plan, err = PlanModified(r.graph, remoteGraph, differ.DiffOptions{WorkspaceID: workspaceID})
		_ = m
	case ModeSingle:
		plan, err = PlanSingle(r.graph, m.ResourceType, m.TargetID)
	default:
		return nil, fmt.Errorf("unknown validation mode: %T", mode)
	}

	if err != nil {
		return nil, fmt.Errorf("building validation plan: %w", err)
	}

	if len(plan.Units) == 0 {
		return &ValidationReport{Status: RunStatusNoResources}, nil
	}

	validationLog.Info("Validation plan created", "units", len(plan.Units))

	if lc, ok := r.reporter.(reporterLifecycle); ok {
		lc.start(len(plan.Units))
		defer lc.done()
	}

	if err := r.resolveAccountIDs(plan); err != nil {
		return nil, err
	}

	validations := runValidationTasks(ctx, r.client, r.graph, plan.Units, r.reporter, r.concurrency)

	return &ValidationReport{
		Status:    RunStatusExecuted,
		Resources: validations,
	}, nil
}

func (r *Runner) loadRemoteGraph(ctx context.Context) (*resources.Graph, error) {
	remoteResources, err := r.loader.LoadResourcesFromRemote(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading remote resources: %w", err)
	}

	remoteState, err := r.loader.MapRemoteToState(remoteResources)
	if err != nil {
		return nil, fmt.Errorf("building remote state: %w", err)
	}

	return syncer.StateToGraph(remoteState), nil
}

// resolveAccountIDs resolves the account ID for each validation unit by looking up
// the parent data graph resource in the local graph.
func (r *Runner) resolveAccountIDs(plan *ValidationPlan) error {
	cache := make(map[string]string)

	for _, unit := range plan.Units {
		dgURN := r.findDataGraphURN(unit)
		if dgURN == "" {
			return fmt.Errorf("could not determine data graph for resource %s", unit.URN)
		}

		accountID, ok := cache[dgURN]
		if !ok {
			res, exists := r.graph.GetResource(dgURN)
			if !exists {
				return fmt.Errorf("data graph %s not found in local graph", dgURN)
			}

			dgRes, ok := res.RawData().(*dgModel.DataGraphResource)
			if !ok {
				return fmt.Errorf("resource %s is not a data graph", dgURN)
			}

			accountID = dgRes.AccountID
			cache[dgURN] = accountID
		}

		unit.AccountID = accountID
	}

	return nil
}

// findDataGraphURN extracts the data graph URN from a resource's reference
func (r *Runner) findDataGraphURN(unit *ValidationUnit) string {
	switch unit.ResourceType {
	case "model":
		modelRes, ok := unit.Resource.(*dgModel.ModelResource)
		if ok && modelRes.DataGraphRef != nil {
			return modelRes.DataGraphRef.URN
		}
	case "relationship":
		relRes, ok := unit.Resource.(*dgModel.RelationshipResource)
		if ok && relRes.DataGraphRef != nil {
			return relRes.DataGraphRef.URN
		}
	}
	return ""
}
