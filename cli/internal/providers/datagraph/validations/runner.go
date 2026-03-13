package validations

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

var validationLog = logger.New("validations")

// remoteStateLoader abstracts provider methods needed by the runner
type remoteStateLoader interface {
	provider.ManagedRemoteResourceLoader
	provider.StateLoader
}

// Runner orchestrates data graph validation
type Runner struct {
	loader remoteStateLoader
	client dgClient.DataGraphClient
	graph  *resources.Graph
}

// NewRunner creates a new validation runner
func NewRunner(client dgClient.DataGraphClient, loader remoteStateLoader, graph *resources.Graph) *Runner {
	return &Runner{
		loader: loader,
		client: client,
		graph:  graph,
	}
}

// Run executes validation based on the specified mode and returns results
func (r *Runner) Run(ctx context.Context, mode Mode, resourceType, targetID, workspaceID string) (*ValidationResults, error) {
	validationLog.Info("Starting validation run", "mode", mode, "resourceType", resourceType, "targetID", targetID)

	// Only load remote state for --modified mode (needed for diff computation)
	var remoteGraph *resources.Graph
	if mode == ModeModified {
		remoteResources, err := r.loader.LoadResourcesFromRemote(ctx)
		if err != nil {
			return nil, fmt.Errorf("loading remote resources: %w", err)
		}

		remoteState, err := r.loader.MapRemoteToState(remoteResources)
		if err != nil {
			return nil, fmt.Errorf("building remote state: %w", err)
		}

		remoteGraph = syncer.StateToGraph(remoteState)
	}

	// Build plan
	planner := NewPlanner(r.graph)
	plan, err := planner.BuildPlan(remoteGraph, mode, resourceType, targetID, differ.DiffOptions{WorkspaceID: workspaceID})
	if err != nil {
		return nil, fmt.Errorf("building validation plan: %w", err)
	}

	if len(plan.Units) == 0 {
		return &ValidationResults{Status: RunStatusNoResources}, nil
	}

	validationLog.Info("Validation plan created", "units", len(plan.Units))

	// Resolve account IDs from the local graph
	if err := r.resolveAccountIDs(plan); err != nil {
		return nil, err
	}

	// Run validations concurrently
	validations := runValidationTasks(ctx, r.client, r.graph, plan.Units)

	return &ValidationResults{
		Status:    RunStatusExecuted,
		Resources: validations,
	}, nil
}

// resolveAccountIDs resolves the account ID for each validation unit by looking up
// the parent data graph resource in the local graph.
func (r *Runner) resolveAccountIDs(plan *ValidationPlan) error {
	cache := make(map[string]string)

	for _, unit := range plan.Units {
		dgURN := r.findDataGraphURN(unit)
		if dgURN == "" {
			return fmt.Errorf("could not determine data graph for resource %s/%s", unit.ResourceType, unit.ID)
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
