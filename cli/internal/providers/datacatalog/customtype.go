package datacatalog

import (
	"context"
	"fmt"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	syncerstate "github.com/rudderlabs/rudder-iac/cli/internal/syncer/state"
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
		Variants:    toArgs.Variants.ToCatalogVariants(),
		ExternalId:  ID,
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

	prevState := state.CustomTypeState{}
	prevState.FromResourceData(olds)

	toArgs := state.CustomTypeArgs{}
	toArgs.FromResourceData(input)

	var (
		updated *catalog.CustomType
		err     error
	)

	// Check if there are any changes using the Diff method
	if prevState.CustomTypeArgs.Diff(&toArgs) {
		properties := make([]catalog.CustomTypeProperty, 0, len(toArgs.Properties))
		for _, prop := range toArgs.Properties {
			properties = append(properties, catalog.CustomTypeProperty{
				ID:       prop.ID,
				Required: prop.Required,
			})
		}

		updated, err = p.client.UpdateCustomType(ctx, prevState.ID, &catalog.CustomTypeUpdate{
			Name:        toArgs.Name,
			Description: toArgs.Description,
			Type:        toArgs.Type,
			Config:      toArgs.Config,
			Properties:  properties,
			Variants:    toArgs.Variants.ToCatalogVariants(),
		})
		if err != nil {
			return nil, fmt.Errorf("updating custom type resource in upstream catalog: %w", err)
		}
	}

	var toState state.CustomTypeState

	if updated == nil {
		// No changes were made, copy from previous state with new args
		toState = state.CustomTypeState{
			CustomTypeArgs:  toArgs,
			ID:              prevState.ID,
			LocalID:         toArgs.LocalID,
			Name:            prevState.Name,
			Description:     prevState.Description,
			Type:            prevState.Type,
			Config:          prevState.Config,
			Version:         prevState.Version,
			ItemDefinitions: prevState.ItemDefinitions,
			Rules:           prevState.Rules,
			WorkspaceID:     prevState.WorkspaceID,
			CreatedAt:       prevState.CreatedAt,
			UpdatedAt:       prevState.UpdatedAt,
		}
	} else {
		// Changes were made, use updated state
		toState = state.CustomTypeState{
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

// LoadResourcesFromRemote loads all custom types from the remote catalog
func (p *CustomTypeProvider) LoadResourcesFromRemote(ctx context.Context) (*resources.ResourceCollection, error) {
	p.log.Debug("loading custom types from remote catalog")
	collection := resources.NewResourceCollection()

	// fetch custom types from remote
	customTypes, err := p.client.GetCustomTypes(ctx)
	if err != nil {
		return nil, err
	}

	// Convert slice to map[string]interface{} where key = customType's remoteId
	resourceMap := make(map[string]*resources.RemoteResource)
	for _, customType := range customTypes {
		resourceMap[customType.ID] = &resources.RemoteResource{
			ID:         customType.ID,
			ExternalID: customType.ExternalId,
			Data:       customType,
		}
	}
	collection.Set(state.CustomTypeResourceType, resourceMap)

	return collection, nil
}

func (p *CustomTypeProvider) LoadStateFromResources(ctx context.Context, collection *resources.ResourceCollection) (*syncerstate.State, error) {
	return syncerstate.EmptyState(), nil
}
