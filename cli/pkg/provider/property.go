package provider

import (
	"context"
	"fmt"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/rudderlabs/rudder-iac/cli/pkg/logger"
	"github.com/rudderlabs/rudder-iac/cli/pkg/provider/state"
)

type propertyProvider struct {
	client client.DataCatalog
	log    logger.Logger
}

func NewPropertyProvider(dc client.DataCatalog) syncer.Provider {
	return &propertyProvider{
		client: dc,
		log: logger.Logger{
			Logger: logger.New("provider").With("type", "property"),
		},
	}
}

func (p *propertyProvider) Create(ctx context.Context, ID string, resourceType string, data resources.ResourceData) (*resources.ResourceData, error) {
	p.log.With("provider", "property").Debug("creating property resource in upstream catalog", "id", ID)

	toArgs := state.PropertyArgs{}
	toArgs.FromResourceData(data)

	property, err := p.client.CreateProperty(ctx, client.PropertyCreate{
		Name:        toArgs.Name,
		Description: toArgs.Description,
		Type:        toArgs.Type,
		Config:      toArgs.Config,
	})

	if err != nil {
		return nil, fmt.Errorf("creating property resource in upstream catalog: %w", err)
	}

	propertyState := state.PropertyState{
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

func (p *propertyProvider) Update(ctx context.Context, ID string, resourceType string, input resources.ResourceData, olds resources.ResourceData) (*resources.ResourceData, error) {
	p.log.Debug("updating property resource in upstream catalog", "id", ID)

	toArgs := state.PropertyArgs{}
	toArgs.FromResourceData(input)

	oldState := state.PropertyState{}
	oldState.FromResourceData(olds)

	updated, err := p.client.UpdateProperty(ctx, oldState.ID, &client.Property{
		ID:          oldState.ID,
		Name:        toArgs.Name,
		Description: toArgs.Description,
		Type:        toArgs.Type,
		Config:      toArgs.Config,
		WorkspaceId: oldState.WorkspaceID,
	})

	if err != nil {
		return nil, fmt.Errorf("updating property resource in upstream catalog: %w", err)
	}

	toState := state.PropertyState{
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

func (p *propertyProvider) Delete(ctx context.Context, ID string, resourceType string, data resources.ResourceData) error {
	p.log.Debug("deleting property resource in upstream catalog", "id", ID)

	err := p.client.DeleteProperty(ctx, data["id"].(string))

	if err != nil && !IsNotFound(err) {
		return fmt.Errorf("deleting property resource in upstream catalog: %w", err)
	}

	return nil
}
