package provider

import (
	"context"
	"embed"
	"fmt"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/workspace"
	"github.com/rudderlabs/rudder-iac/cli/pkg/logger"
	providerstate "github.com/rudderlabs/rudder-iac/cli/pkg/provider/state"
)

//go:embed templates/property.yaml.tmpl
var propertyTemplate embed.FS

type PropertyProvider struct {
	providers.DefaultResourceProvider
	client catalog.DataCatalog
	log    logger.Logger
}

func NewPropertyProvider(dc catalog.DataCatalog) *PropertyProvider {
	return &PropertyProvider{
		client: dc,
		log: logger.Logger{
			Logger: logger.New("provider").With("type", "property"),
		},
	}
}

func (p *PropertyProvider) List(ctx context.Context, resourceType string) ([]*workspace.Resource, error) {
	properties, err := p.client.ListProperties(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing properties in upstream catalog: %w", err)
	}

	resources := make([]*workspace.Resource, len(properties))
	for i, prop := range properties {
		resources[i] = &workspace.Resource{
			ID:          prop.ID,
			Type:        resourceType,
			Name:        prop.Name,
			Description: prop.Description,
			Data:        prop,
		}
	}
	return resources, nil
}

func (p *PropertyProvider) Template(ctx context.Context, resource *workspace.Resource) ([]byte, error) {
	p.log.Debug("generating YAML spec for property", "id", resource.ID)

	// Read template content
	return propertyTemplate.ReadFile("templates/property.yaml.tmpl")
}

func (p *PropertyProvider) ImportState(ctx context.Context, resource *workspace.Resource) (*state.ResourceState, error) {
	property, ok := resource.Data.(*catalog.Property)
	if !ok {
		return nil, fmt.Errorf("invalid property data type: %T", resource.Data)
	}

	args := providerstate.PropertyArgs{
		Name:        property.Name,
		Description: property.Description,
		Type:        property.Type,
		Config:      property.Config,
	}

	propertyState := providerstate.PropertyState{
		PropertyArgs: args,
		ID:           property.ID,
		Name:         property.Name,
		Description:  property.Description,
		Type:         property.Type,
		Config:       property.Config,
		WorkspaceID:  property.WorkspaceId,
		CreatedAt:    property.CreatedAt.UTC().String(),
		UpdatedAt:    property.UpdatedAt.UTC().String(),
	}

	resourceState := &state.ResourceState{
		ID:           property.ID,
		Type:         PropertyResourceType,
		Input:        args.ToResourceData(),
		Output:       propertyState.ToResourceData(),
		Dependencies: make([]string, 0),
	}

	return resourceState, nil
}

func (p *PropertyProvider) Create(ctx context.Context, ID string, data resources.ResourceData) (*resources.ResourceData, error) {
	p.log.With("provider", "property").Debug("creating property resource in upstream catalog", "id", ID)

	toArgs := providerstate.PropertyArgs{}
	toArgs.FromResourceData(data)

	property, err := p.client.CreateProperty(ctx, catalog.PropertyCreate{
		Name:        toArgs.Name,
		Description: toArgs.Description,
		Type:        toArgs.Type.(string),
		Config:      toArgs.Config,
	})

	if err != nil {
		return nil, fmt.Errorf("creating property resource in upstream catalog: %w", err)
	}

	propertyState := providerstate.PropertyState{
		PropertyArgs: toArgs,
		ID:           property.ID,
		Name:         property.Name,
		Description:  property.Description,
		Type:         property.Type,
		Config:       property.Config,
		WorkspaceID:  property.WorkspaceId,
		CreatedAt:    property.CreatedAt.UTC().String(),
		UpdatedAt:    property.UpdatedAt.UTC().String(),
	}

	resourceData := propertyState.ToResourceData()
	return &resourceData, nil
}

func (p *PropertyProvider) Update(ctx context.Context, ID string, input resources.ResourceData, olds resources.ResourceData) (*resources.ResourceData, error) {
	p.log.Debug("updating property resource in upstream catalog", "id", ID)

	toArgs := providerstate.PropertyArgs{}
	toArgs.FromResourceData(input)

	oldState := providerstate.PropertyState{}
	oldState.FromResourceData(olds)

	updated, err := p.client.UpdateProperty(ctx, oldState.ID, &catalog.Property{
		ID:          oldState.ID,
		Name:        toArgs.Name,
		Description: toArgs.Description,
		Type:        toArgs.Type.(string),
		Config:      toArgs.Config,
		WorkspaceId: oldState.WorkspaceID,
	})

	if err != nil {
		return nil, fmt.Errorf("updating property resource in upstream catalog: %w", err)
	}

	toState := providerstate.PropertyState{
		PropertyArgs: toArgs,
		ID:           updated.ID,
		Name:         updated.Name,
		Description:  updated.Description,
		Type:         updated.Type,
		Config:       updated.Config,
		WorkspaceID:  updated.WorkspaceId,
		CreatedAt:    updated.CreatedAt.String(),
		UpdatedAt:    updated.UpdatedAt.String(),
	}

	resourceData := toState.ToResourceData()
	return &resourceData, nil
}

func (p *PropertyProvider) Delete(ctx context.Context, ID string, data resources.ResourceData) error {
	p.log.Debug("deleting property resource in upstream catalog", "id", ID)

	err := p.client.DeleteProperty(ctx, data["id"].(string))

	if err != nil && !catalog.IsCatalogNotFoundError(err) {
		return fmt.Errorf("deleting property resource in upstream catalog: %w", err)
	}

	return nil
}
