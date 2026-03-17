package validator

import (
	"context"
	"fmt"

	dgClient "github.com/rudderlabs/rudder-iac/api/client/datagraph"
	dgModel "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/pkg/tasker"
)

const concurrency = 4

// ValidationReporter receives notifications about validation task progress
type ValidationReporter interface {
	TaskStarted(id, description string)
	TaskCompleted(id, description string, err error)
}

// noopReporter is a no-op implementation of ValidationReporter
type noopReporter struct{}

func (noopReporter) TaskStarted(string, string)        {}
func (noopReporter) TaskCompleted(string, string, error) {}

// validateTask implements tasker.Task for a single validation unit
type validateTask struct {
	unit *ValidationUnit
}

func (t *validateTask) Id() string {
	return t.unit.URN
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
	reporter ValidationReporter,
) []*ResourceValidation {
	tasks := make([]tasker.Task, 0, len(units))
	for _, u := range units {
		tasks = append(tasks, &validateTask{unit: u})
	}

	results := tasker.NewResults[*ResourceValidation]()
	_ = tasker.RunTasks(ctx, tasks, concurrency, true, func(task tasker.Task) error {
		vt := task.(*validateTask)
		u := vt.unit

		description := fmt.Sprintf("Validating %s %s (%s)", u.ResourceType, u.DisplayName, u.URN)
		reporter.TaskStarted(u.URN, description)
		result := executeValidation(ctx, client, graph, u)
		reporter.TaskCompleted(u.URN, description, result.completionError())

		results.Store(u.URN, result)
		return nil
	})

	// Collect results in order of units
	validations := make([]*ResourceValidation, 0, len(units))
	for _, u := range units {
		if r, ok := results.Get(u.URN); ok {
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
			URN:          unit.URN,
			ResourceType: unit.ResourceType,
			Err:          fmt.Errorf("unknown resource type: %s", unit.ResourceType),
		}
	}
}

func validateModel(ctx context.Context, client dgClient.DataGraphClient, unit *ValidationUnit) *ResourceValidation {
	modelRes := unit.Resource.(*dgModel.ModelResource)

	req := &dgClient.ValidateModelRequest{
		AccountID: unit.AccountID,
		Type:      modelRes.Type,
		TableRef:  modelRes.Table,
		PrimaryID: modelRes.PrimaryID,
		Root:      modelRes.Root,
		Timestamp: modelRes.Timestamp,
	}

	report, err := client.ValidateModel(ctx, req)
	if err != nil {
		return &ResourceValidation{
			ID:           unit.ID,
			URN:          unit.URN,
			DisplayName:  modelRes.DisplayName,
			ResourceType: "model",
			Err:          err,
		}
	}

	return &ResourceValidation{
		ID:           unit.ID,
		URN:          unit.URN,
		DisplayName:  modelRes.DisplayName,
		ResourceType: "model",
		Issues:       report.Issues,
	}
}

func validateRelationship(ctx context.Context, client dgClient.DataGraphClient, graph *resources.Graph, unit *ValidationUnit) *ResourceValidation {
	relRes := unit.Resource.(*dgModel.RelationshipResource)

	sourceTableRef, err := resolveModelTableRef(graph, relRes.SourceModelRef)
	if err != nil {
		return &ResourceValidation{
			ID:           unit.ID,
			URN:          unit.URN,
			DisplayName:  relRes.DisplayName,
			ResourceType: "relationship",
			Err:          fmt.Errorf("resolving source model table ref: %w", err),
		}
	}

	targetTableRef, err := resolveModelTableRef(graph, relRes.TargetModelRef)
	if err != nil {
		return &ResourceValidation{
			ID:           unit.ID,
			URN:          unit.URN,
			DisplayName:  relRes.DisplayName,
			ResourceType: "relationship",
			Err:          fmt.Errorf("resolving target model table ref: %w", err),
		}
	}

	req := &dgClient.ValidateRelationshipRequest{
		AccountID:   unit.AccountID,
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
			URN:          unit.URN,
			DisplayName:  relRes.DisplayName,
			ResourceType: "relationship",
			Err:          err,
		}
	}

	return &ResourceValidation{
		ID:           unit.ID,
		URN:          unit.URN,
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
