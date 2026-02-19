package datagraph

import (
	"fmt"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	dgClient "github.com/rudderlabs/rudder-iac/api/client/datagraph"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/handlers/datagraph"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/handlers/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/handlers/relationship"
	dgModel "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

// Provider wraps the base provider to provide a concrete type for dependency injection
// The provider owns all spec parsing logic for data-graph specs
type Provider struct {
	*provider.BaseProvider
	dataGraphHandler    *datagraph.DataGraphHandler
	modelHandler        *model.ModelHandler
	relationshipHandler *relationship.RelationshipHandler
}

// NewProvider creates a new data graph provider instance
func NewProvider(client dgClient.DataGraphClient) *Provider {
	dgHandler := datagraph.NewHandler(client)
	modelHandler := model.NewHandler(client)
	relationshipHandler := relationship.NewHandler(client)

	handlers := []provider.Handler{
		dgHandler,
		modelHandler,
		relationshipHandler,
	}

	return &Provider{
		BaseProvider:        provider.NewBaseProvider(handlers),
		dataGraphHandler:    dgHandler,
		modelHandler:        modelHandler,
		relationshipHandler: relationshipHandler,
	}
}

// ParseSpec overrides BaseProvider.ParseSpec to extract IDs from data-graph specs with inline models
func (p *Provider) ParseSpec(path string, s *specs.Spec) (*specs.ParsedSpec, error) {
	// For data-graph specs, extract both data graph ID and inline model IDs
	if s.Kind == "data-graph" {
		return p.parseDataGraphWithInlineModels(s)
	}

	// For other specs, use base implementation
	return p.BaseProvider.ParseSpec(path, s)
}

// LoadSpec overrides BaseProvider.LoadSpec to handle inline models in data-graph specs
func (p *Provider) LoadSpec(path string, s *specs.Spec) error {
	// For data-graph specs, check for inline models
	if s.Kind == "data-graph" {
		return p.loadDataGraphWithInlineModels(s)
	}

	// For other specs, use base implementation
	return p.BaseProvider.LoadSpec(path, s)
}

func (p *Provider) parseDataGraphWithInlineModels(s *specs.Spec) (*specs.ParsedSpec, error) {
	// Parse the data-graph spec to extract IDs
	var dgSpec dgModel.DataGraphSpec
	if err := mapstructure.Decode(s.Spec, &dgSpec); err != nil {
		return nil, fmt.Errorf("decoding data graph spec: %w", err)
	}

	localIDs := []specs.LocalID{{ID: dgSpec.ID, JSONPointerPath: "/spec/id"}}

	for i, modelSpec := range dgSpec.Models {
		if modelSpec.ID != "" {
			localIDs = append(localIDs, specs.LocalID{
				ID:              modelSpec.ID,
				JSONPointerPath: fmt.Sprintf("/spec/models/%d/id", i),
			})
		}
		for j, relSpec := range modelSpec.Relationships {
			if relSpec.ID != "" {
				localIDs = append(localIDs, specs.LocalID{
					ID:              relSpec.ID,
					JSONPointerPath: fmt.Sprintf("/spec/models/%d/relationships/%d/id", i, j),
				})
			}
		}
	}

	return &specs.ParsedSpec{LocalIDs: localIDs}, nil
}

func (p *Provider) loadDataGraphWithInlineModels(s *specs.Spec) error {
	// Parse the data-graph spec
	var dgSpec dgModel.DataGraphSpec
	if err := mapstructure.Decode(s.Spec, &dgSpec); err != nil {
		return fmt.Errorf("decoding data graph spec: %w", err)
	}

	// Validate the data graph spec
	if err := p.validateDataGraphSpec(&dgSpec); err != nil {
		return fmt.Errorf("validating data graph spec: %w", err)
	}

	// Extract the data graph resource
	dgResource, err := p.extractDataGraphResource(&dgSpec)
	if err != nil {
		return fmt.Errorf("extracting data graph resource: %w", err)
	}

	// Register the data graph resource with its handler
	if err := p.dataGraphHandler.AddResource(dgSpec.ID, dgResource); err != nil {
		return fmt.Errorf("adding data graph resource: %w", err)
	}

	// Process inline models if any
	for _, modelSpec := range dgSpec.Models {
		if err := p.processInlineModel(dgSpec.ID, modelSpec); err != nil {
			return fmt.Errorf("processing inline model %s: %w", modelSpec.ID, err)
		}
	}

	return nil
}

func (p *Provider) processInlineModel(dataGraphID string, modelSpec dgModel.ModelSpec) error {
	// Validate the inline model spec
	if err := p.validateModelSpec(&modelSpec); err != nil {
		return fmt.Errorf("validating model spec: %w", err)
	}

	// Extract model resource from spec
	resource, err := p.extractModelResource(dataGraphID, &modelSpec)
	if err != nil {
		return fmt.Errorf("extracting model resource: %w", err)
	}

	// Register resource with handler
	if err := p.modelHandler.AddResource(modelSpec.ID, resource); err != nil {
		return fmt.Errorf("adding model resource: %w", err)
	}

	// Process inline relationships if any
	for _, relSpec := range modelSpec.Relationships {
		if err := p.processInlineRelationship(dataGraphID, modelSpec.ID, &relSpec); err != nil {
			return fmt.Errorf("processing inline relationship %s: %w", relSpec.ID, err)
		}
	}

	return nil
}

// validateModelSpec validates an inline model spec
func (p *Provider) validateModelSpec(spec *dgModel.ModelSpec) error {
	if spec.ID == "" {
		return fmt.Errorf("id is required")
	}
	if spec.DisplayName == "" {
		return fmt.Errorf("display_name is required")
	}
	if spec.Type != "entity" && spec.Type != "event" {
		return fmt.Errorf("type must be 'entity' or 'event'")
	}
	if spec.Table == "" {
		return fmt.Errorf("table is required")
	}

	// Type-specific validation
	switch spec.Type {
	case "entity":
		if spec.PrimaryID == "" {
			return fmt.Errorf("primary_id is required for entity models")
		}
	case "event":
		if spec.Timestamp == "" {
			return fmt.Errorf("timestamp is required for event models")
		}
	}

	return nil
}

// extractModelResource creates a ModelResource from an inline model spec
func (p *Provider) extractModelResource(dataGraphID string, spec *dgModel.ModelSpec) (*dgModel.ModelResource, error) {
	// Create URN for the parent data graph
	dataGraphURN := resources.URN(dataGraphID, datagraph.HandlerMetadata.ResourceType)

	// Create PropertyRef to the data graph
	dataGraphRef := datagraph.CreateDataGraphReference(dataGraphURN)

	resource := &dgModel.ModelResource{
		ID:           spec.ID,
		DisplayName:  spec.DisplayName,
		Type:         spec.Type,
		Table:        spec.Table,
		Description:  spec.Description,
		DataGraphRef: dataGraphRef,
		PrimaryID:    spec.PrimaryID,
		Root:         spec.Root,
		Timestamp:    spec.Timestamp,
	}

	return resource, nil
}

// validateDataGraphSpec validates a data graph spec
func (p *Provider) validateDataGraphSpec(spec *dgModel.DataGraphSpec) error {
	if spec.ID == "" {
		return fmt.Errorf("id is required")
	}
	if spec.AccountID == "" {
		return fmt.Errorf("account_id is required")
	}
	return nil
}

// extractDataGraphResource creates a DataGraphResource from a spec
func (p *Provider) extractDataGraphResource(spec *dgModel.DataGraphSpec) (*dgModel.DataGraphResource, error) {
	resource := &dgModel.DataGraphResource{
		ID:        spec.ID,
		AccountID: spec.AccountID,
	}
	return resource, nil
}

func (p *Provider) processInlineRelationship(dataGraphID, sourceModelID string, relSpec *dgModel.RelationshipSpec) error {
	// Validate relationship spec
	if err := p.validateRelationshipSpec(relSpec); err != nil {
		return fmt.Errorf("validating relationship spec: %w", err)
	}

	// Extract relationship resource
	resource, err := p.extractRelationshipResource(dataGraphID, sourceModelID, relSpec)
	if err != nil {
		return fmt.Errorf("extracting relationship: %w", err)
	}

	// Register with relationship handler
	if err := p.relationshipHandler.AddResource(relSpec.ID, resource); err != nil {
		return fmt.Errorf("adding relationship: %w", err)
	}

	return nil
}

func (p *Provider) validateRelationshipSpec(spec *dgModel.RelationshipSpec) error {
	if spec.ID == "" {
		return fmt.Errorf("id is required")
	}
	if spec.DisplayName == "" {
		return fmt.Errorf("display_name is required")
	}
	if spec.Cardinality == "" {
		return fmt.Errorf("cardinality is required")
	}
	// Validate cardinality value
	validCardinalities := map[string]bool{
		"one-to-one":  true,
		"one-to-many": true,
		"many-to-one": true,
	}
	if !validCardinalities[spec.Cardinality] {
		return fmt.Errorf("cardinality must be one of: one-to-one, one-to-many, many-to-one")
	}
	if spec.Target == "" {
		return fmt.Errorf("target is required")
	}
	if spec.SourceJoinKey == "" {
		return fmt.Errorf("source_join_key is required")
	}
	if spec.TargetJoinKey == "" {
		return fmt.Errorf("target_join_key is required")
	}
	return nil
}

func (p *Provider) extractRelationshipResource(dataGraphID, sourceModelID string, spec *dgModel.RelationshipSpec) (*dgModel.RelationshipResource, error) {
	// Create URN for data graph (parent)
	dataGraphURN := resources.URN(dataGraphID, datagraph.HandlerMetadata.ResourceType)
	dataGraphRef := relationship.CreateDataGraphReference(dataGraphURN)

	// Create URN for source model (from)
	sourceModelURN := resources.URN(sourceModelID, model.HandlerMetadata.ResourceType)
	sourceModelRef := relationship.CreateModelReference(sourceModelURN)

	// Parse target model reference from spec
	targetModelURN, err := parseModelReference(spec.Target)
	if err != nil {
		return nil, fmt.Errorf("parsing target model reference: %w", err)
	}
	targetModelRef := relationship.CreateModelReference(targetModelURN)

	return &dgModel.RelationshipResource{
		ID:             spec.ID,
		DisplayName:    spec.DisplayName,
		DataGraphRef:   dataGraphRef,
		SourceModelRef: sourceModelRef,
		TargetModelRef: targetModelRef,
		SourceJoinKey:  spec.SourceJoinKey,
		TargetJoinKey:  spec.TargetJoinKey,
		Cardinality:    spec.Cardinality,
	}, nil
}

// parseModelReference parses a model reference like '#data-graph-model:user' and returns the URN
func parseModelReference(ref string) (string, error) {
	if !strings.HasPrefix(ref, "#data-graph-model:") {
		return "", fmt.Errorf("invalid model reference format, expected '#data-graph-model:<id>', got %q", ref)
	}
	modelID := strings.TrimPrefix(ref, "#data-graph-model:")
	if modelID == "" {
		return "", fmt.Errorf("model ID cannot be empty in reference")
	}
	return resources.URN(modelID, model.HandlerMetadata.ResourceType), nil
}
