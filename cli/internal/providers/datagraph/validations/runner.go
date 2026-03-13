package validations

import (
	"context"
	"fmt"

	dgClient "github.com/rudderlabs/rudder-iac/api/client/datagraph"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/handlers/datagraph"
	dgModel "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
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

	// Load remote state for --modified mode and for data graph ID resolution
	remoteResources, err := r.loader.LoadResourcesFromRemote(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading remote resources: %w", err)
	}

	remoteState, err := r.loader.MapRemoteToState(remoteResources)
	if err != nil {
		return nil, fmt.Errorf("building remote state: %w", err)
	}

	remoteGraph := syncer.StateToGraph(remoteState)

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

	// Resolve data graph remote IDs for each unit
	if err := r.resolveDataGraphIDs(plan, remoteState); err != nil {
		return nil, err
	}

	// Run validations concurrently
	validations := runValidationTasks(ctx, r.client, r.graph, plan.Units)

	return &ValidationResults{
		Status:    RunStatusExecuted,
		Resources: validations,
	}, nil
}

// resolveDataGraphIDs resolves the remote data graph ID for each validation unit.
// The validate endpoints need the data graph's remote ID in the URL path.
func (r *Runner) resolveDataGraphIDs(plan *ValidationPlan, remoteState *state.State) error {
	// Build a cache of data graph URN -> remote ID from state
	dgRemoteIDs := make(map[string]string)
	for urn, rs := range remoteState.Resources {
		if rs.Type == datagraph.HandlerMetadata.ResourceType {
			dgState, ok := rs.OutputRaw.(*dgModel.DataGraphState)
			if ok {
				dgRemoteIDs[urn] = dgState.ID
			}
		}
	}

	for _, unit := range plan.Units {
		dgURN := r.findDataGraphURN(unit)
		if dgURN == "" {
			return fmt.Errorf("could not determine data graph for resource %s/%s", unit.ResourceType, unit.ID)
		}

		remoteID, ok := dgRemoteIDs[dgURN]
		if !ok {
			return fmt.Errorf("data graph %s has not been synced yet — run 'apply' first to create it remotely", dgURN)
		}

		unit.DataGraphID = remoteID
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
