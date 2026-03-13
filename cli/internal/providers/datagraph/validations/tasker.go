package validations

import (
	"context"
	"fmt"

	dgClient "github.com/rudderlabs/rudder-iac/api/client/datagraph"
	dgModel "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/pkg/tasker"
)

const concurrency = 4

// validateTask implements tasker.Task for a single validation unit
type validateTask struct {
	unit *ValidationUnit
}

func (t *validateTask) Id() string {
	return fmt.Sprintf("%s:%s", t.unit.ResourceType, t.unit.ID)
}

func (t *validateTask) Dependencies() []string {
	return nil
}

// runValidationTasks executes all validation tasks concurrently
func runValidationTasks(
	ctx context.Context,
	client dgClient.DataGraphClient,
	graph *resources.Graph,
	units []*ValidationUnit,
) []*ResourceValidation {
	tasks := make([]tasker.Task, 0, len(units))
	for _, u := range units {
		tasks = append(tasks, &validateTask{unit: u})
	}

	results := tasker.NewResults[*ResourceValidation]()
	_ = tasker.RunTasks(ctx, tasks, concurrency, true, func(task tasker.Task) error {
		vt := task.(*validateTask)
		result := executeValidation(ctx, client, graph, vt.unit)
		results.Store(vt.Id(), result)
		return nil
	})

	// Collect results in order of units
	validations := make([]*ResourceValidation, 0, len(units))
	for _, u := range units {
		key := fmt.Sprintf("%s:%s", u.ResourceType, u.ID)
		if r, ok := results.Get(key); ok {
			validations = append(validations, r)
		}
	}

	return validations
}

// executeValidation runs a single validation and returns the result
func executeValidation(
	ctx context.Context,
	client dgClient.DataGraphClient,
	graph *resources.Graph,
	unit *ValidationUnit,
) *ResourceValidation {
	switch unit.ResourceType {
	case "model":
		return validateModel(ctx, client, unit)
	case "relationship":
		return validateRelationship(ctx, client, graph, unit)
	default:
		return &ResourceValidation{
			ID:           unit.ID,
			ResourceType: unit.ResourceType,
			Err:          fmt.Errorf("unknown resource type: %s", unit.ResourceType),
		}
	}
}

func validateModel(ctx context.Context, client dgClient.DataGraphClient, unit *ValidationUnit) *ResourceValidation {
	modelRes := unit.Resource.(*dgModel.ModelResource)

	req := &dgClient.ValidateModelRequest{
		DataGraphID: unit.DataGraphID,
		Type:        modelRes.Type,
		TableRef:    modelRes.Table,
		PrimaryID:   modelRes.PrimaryID,
		Root:        modelRes.Root,
		Timestamp:   modelRes.Timestamp,
	}

	report, err := client.ValidateModel(ctx, req)
	if err != nil {
		return &ResourceValidation{
			ID:           unit.ID,
			DisplayName:  modelRes.DisplayName,
			ResourceType: "model",
			Err:          err,
		}
	}

	return &ResourceValidation{
		ID:           unit.ID,
		DisplayName:  modelRes.DisplayName,
		ResourceType: "model",
		Issues:       report.Issues,
	}
}

func validateRelationship(ctx context.Context, client dgClient.DataGraphClient, graph *resources.Graph, unit *ValidationUnit) *ResourceValidation {
	relRes := unit.Resource.(*dgModel.RelationshipResource)

	// Resolve source and target model table refs from the graph
	sourceTableRef, err := resolveModelTableRef(graph, relRes.SourceModelRef)
	if err != nil {
		return &ResourceValidation{
			ID:           unit.ID,
			DisplayName:  relRes.DisplayName,
			ResourceType: "relationship",
			Err:          fmt.Errorf("resolving source model table ref: %w", err),
		}
	}

	targetTableRef, err := resolveModelTableRef(graph, relRes.TargetModelRef)
	if err != nil {
		return &ResourceValidation{
			ID:           unit.ID,
			DisplayName:  relRes.DisplayName,
			ResourceType: "relationship",
			Err:          fmt.Errorf("resolving target model table ref: %w", err),
		}
	}

	req := &dgClient.ValidateRelationshipRequest{
		DataGraphID: unit.DataGraphID,
		Cardinality: relRes.Cardinality,
		SourceModel: dgClient.ValidationModelRef{
			TableRef: sourceTableRef,
			JoinKey:  relRes.SourceJoinKey,
		},
		TargetModel: dgClient.ValidationModelRef{
			TableRef: targetTableRef,
			JoinKey:  relRes.TargetJoinKey,
		},
	}

	report, err := client.ValidateRelationship(ctx, req)
	if err != nil {
		return &ResourceValidation{
			ID:           unit.ID,
			DisplayName:  relRes.DisplayName,
			ResourceType: "relationship",
			Err:          err,
		}
	}

	return &ResourceValidation{
		ID:           unit.ID,
		DisplayName:  relRes.DisplayName,
		ResourceType: "relationship",
		Issues:       report.Issues,
	}
}

// resolveModelTableRef looks up a model's table ref from the graph using a PropertyRef
func resolveModelTableRef(graph *resources.Graph, ref *resources.PropertyRef) (string, error) {
	if ref == nil {
		return "", fmt.Errorf("model reference is nil")
	}

	r, exists := graph.GetResource(ref.URN)
	if !exists {
		return "", fmt.Errorf("model %s not found in graph", ref.URN)
	}

	modelRes, ok := r.RawData().(*dgModel.ModelResource)
	if !ok {
		return "", fmt.Errorf("resource %s is not a model", ref.URN)
	}

	return modelRes.Table, nil
}
