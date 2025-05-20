package provider

import (
	"context"
	"fmt"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/rudderlabs/rudder-iac/cli/pkg/logger"
	"github.com/rudderlabs/rudder-iac/cli/pkg/provider/state"
)

type CustomTypeProvider struct {
	client catalog.DataCatalog
	log    logger.Logger
}

func NewCustomTypeProvider(dc catalog.DataCatalog) *CustomTypeProvider {
	return &CustomTypeProvider{
		client: dc,
		log: logger.Logger{
			Logger: logger.New("provider").With("type", "customtype"),
		},
	}
}

func (p *CustomTypeProvider) Create(ctx context.Context, ID string, data resources.ResourceData) (*resources.ResourceData, error) {
	p.log.Debug("creating custom type in upstream catalog", "id", ID)

	toArgs := state.CustomTypeArgs{}
	toArgs.FromResourceData(data)

	properties := make([]catalog.CustomTypeProperty, 0, len(toArgs.Properties))
	for _, prop := range toArgs.Properties {
		properties = append(properties, catalog.CustomTypeProperty{
			ID:       prop.ID,
			Required: prop.Required,
		})
	}

	input := catalog.CustomTypeCreate{
		Name:        toArgs.Name,
		Description: toArgs.Description,
		Type:        toArgs.Type,
		Config:      toArgs.Config,
		Properties:  properties,
	}

	customType, err := p.client.CreateCustomType(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("creating custom type: %w", err)
	}

	customTypeState := state.CustomTypeState{
		CustomTypeArgs:  toArgs,
		ID:              customType.ID,
		LocalID:         toArgs.LocalID,
		Name:            customType.Name,
		Description:     customType.Description,
		Type:            customType.Type,
		Config:          customType.Config,
		Version:         customType.Version,
		ItemDefinitions: customType.ItemDefinitions,
		Rules:           customType.Rules,
		WorkspaceID:     customType.WorkspaceId,
		CreatedAt:       customType.CreatedAt.String(),
		UpdatedAt:       customType.UpdatedAt.String(),
	}

	resourceData := customTypeState.ToResourceData()
	return &resourceData, nil
}

func (p *CustomTypeProvider) Update(ctx context.Context, ID string, input resources.ResourceData, olds resources.ResourceData) (*resources.ResourceData, error) {
	p.log.Debug("updating custom type in upstream catalog", "id", ID)

	toArgs := state.CustomTypeArgs{}
	toArgs.FromResourceData(input)

	oldState := state.CustomTypeState{}
	oldState.FromResourceData(olds)

	properties := make([]catalog.CustomTypeProperty, 0, len(toArgs.Properties))
	for _, prop := range toArgs.Properties {
		properties = append(properties, catalog.CustomTypeProperty{
			ID:       prop.ID,
			Required: prop.Required,
		})
	}

	updated, err := p.client.UpdateCustomType(ctx, oldState.ID, &catalog.CustomType{
		ID:          oldState.ID,
		Name:        toArgs.Name,
		Description: toArgs.Description,
		Type:        toArgs.Type,
		Config:      toArgs.Config,
		Properties:  properties,
	})
	if err != nil {
		return nil, fmt.Errorf("updating custom type resource in upstream catalog: %w", err)
	}

	toState := state.CustomTypeState{
		CustomTypeArgs:  toArgs,
		ID:              updated.ID,
		LocalID:         toArgs.LocalID,
		Name:            updated.Name,
		Description:     updated.Description,
		Type:            updated.Type,
		Config:          updated.Config,
		Version:         updated.Version,
		ItemDefinitions: updated.ItemDefinitions,
		Rules:           updated.Rules,
		WorkspaceID:     updated.WorkspaceId,
		CreatedAt:       updated.CreatedAt.String(),
		UpdatedAt:       updated.UpdatedAt.String(),
	}

	resourceData := toState.ToResourceData()
	return &resourceData, nil
}

func (p *CustomTypeProvider) Delete(ctx context.Context, ID string, data resources.ResourceData) error {
	p.log.Debug("deleting custom type in upstream catalog", "id", ID)

	err := p.client.DeleteCustomType(ctx, data["id"].(string))
	if err != nil && !catalog.IsCatalogNotFoundError(err) {
		return fmt.Errorf("deleting custom type resource in upstream catalog: %w", err)
	}

	return nil
}
