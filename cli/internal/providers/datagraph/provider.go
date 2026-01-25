package datagraph

import (
	"fmt"

	"github.com/go-viper/mapstructure/v2"
	dgClient "github.com/rudderlabs/rudder-iac/api/client/datagraph"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/handlers/datagraph"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/handlers/model"
	dgModel "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

// Provider wraps the base provider to provide a concrete type for dependency injection
// The provider owns all spec parsing logic for data-graph specs
type Provider struct {
	*provider.BaseProvider
	dataGraphHandler *datagraph.DataGraphHandler
	modelHandler     *model.ModelHandler
}

// NewProvider creates a new data graph provider instance
func NewProvider(client dgClient.DataGraphClient) *Provider {
	dgHandler := datagraph.NewHandler(client)
	modelHandler := model.NewHandler(client)

	handlers := []provider.Handler{
		dgHandler,
		modelHandler,
	}

	return &Provider{
		BaseProvider:     provider.NewBaseProvider(handlers),
		dataGraphHandler: dgHandler,
		modelHandler:     modelHandler,
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
		return p.loadDataGraphWithInlineModels(path, s)
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

	// Start with the data graph ID
	externalIDs := []string{dgSpec.ID}

	// Add all inline model IDs
	for _, modelSpec := range dgSpec.Models {
		if modelSpec.ID != "" {
			externalIDs = append(externalIDs, modelSpec.ID)
		}
	}

	return &specs.ParsedSpec{ExternalIDs: externalIDs}, nil
}

func (p *Provider) loadDataGraphWithInlineModels(path string, s *specs.Spec) error {
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
		if err := p.processInlineModel(path, dgSpec.ID, modelSpec); err != nil {
			return fmt.Errorf("processing inline model %s: %w", modelSpec.ID, err)
		}
	}

	return nil
}

func (p *Provider) processInlineModel(path string, dataGraphID string, modelSpec dgModel.ModelSpec) error {
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
