package datacatalog

import (
	"context"
	"fmt"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	impProvider "github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/importremote/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	syncerstate "github.com/rudderlabs/rudder-iac/cli/internal/syncer/state"
)

type PropertyEntityProvider struct {
	*PropertyProvider
	*impProvider.PropertyImportProvider
}

type PropertyProvider struct {
	client catalog.DataCatalog
	log    logger.Logger
}

func NewPropertyProvider(dc catalog.DataCatalog, importDir string) *PropertyEntityProvider {

	pp := &PropertyProvider{
		client: dc,
		log: logger.Logger{
			Logger: logger.New("provider").With("type", "property"),
		},
	}

	imp := impProvider.NewPropertyImportProvider(
		dc,
		logger.Logger{
			Logger: logger.New("importremote.provider").With("type", "property"),
		},
		importDir,
	)

	return &PropertyEntityProvider{
		PropertyProvider:       pp,
		PropertyImportProvider: imp,
	}
}

func (p *PropertyProvider) Create(ctx context.Context, ID string, data resources.ResourceData) (*resources.ResourceData, error) {
	p.log.With("provider", "property").Debug("creating property resource in upstream catalog", "id", ID)

	toArgs := state.PropertyArgs{}
	toArgs.FromResourceData(data)

	property, err := p.client.CreateProperty(ctx, catalog.PropertyCreate{
		Name:        toArgs.Name,
		Description: toArgs.Description,
		Type:        toArgs.Type.(string),
		Config:      toArgs.Config,
		ExternalId:  ID,
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

func (p *PropertyProvider) Update(ctx context.Context, ID string, input resources.ResourceData, olds resources.ResourceData) (*resources.ResourceData, error) {
	p.log.Debug("updating property resource in upstream catalog", "id", ID)

	toArgs := state.PropertyArgs{}
	toArgs.FromResourceData(input)

	oldState := state.PropertyState{}
	oldState.FromResourceData(olds)

	updated, err := p.client.UpdateProperty(ctx, oldState.ID, &catalog.PropertyUpdate{
		Name:        toArgs.Name,
		Description: toArgs.Description,
		Type:        toArgs.Type.(string),
		Config:      toArgs.Config,
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

func (p *PropertyProvider) Delete(ctx context.Context, ID string, data resources.ResourceData) error {
	p.log.Debug("deleting property resource in upstream catalog", "id", ID)

	err := p.client.DeleteProperty(ctx, data["id"].(string))

	if err != nil && !catalog.IsCatalogNotFoundError(err) {
		return fmt.Errorf("deleting property resource in upstream catalog: %w", err)
	}

	return nil
}

func (p *PropertyProvider) Import(ctx context.Context, ID string, data resources.ResourceData, remoteId string) (*resources.ResourceData, error) {
	p.log.Debug("importing property resource", "id", ID, "remoteId", remoteId)

	// Get the property from upstream based on the remoteId
	property, err := p.client.GetProperty(ctx, remoteId)
	if err != nil {
		return nil, fmt.Errorf("getting property from upstream: %w", err)
	}

	// Convert input data to PropertyArgs
	toArgs := state.PropertyArgs{}
	toArgs.FromResourceData(data)

	// Compare the diff between the args and the property
	if toArgs.DiffUpstream(property) {
		p.log.Debug("property has differences, updating", "id", ID, "remoteId", remoteId)
		// Call the updateProperty if there are any differences
		property, err = p.client.UpdateProperty(ctx, remoteId, &catalog.PropertyUpdate{
			Name:        toArgs.Name,
			Description: toArgs.Description,
			Type:        toArgs.Type.(string),
			Config:      toArgs.Config,
		})
		if err != nil {
			return nil, fmt.Errorf("updating property during import: %w", err)
		}
	}

	// Set the external ID on the property
	err = p.client.SetPropertyExternalId(ctx, remoteId, ID)
	if err != nil {
		return nil, fmt.Errorf("setting property external id: %w", err)
	}

	// Build and return the property state
	propertyState := state.PropertyState{
		PropertyArgs: toArgs,
		ID:           property.ID,
		Name:         property.Name,
		Description:  property.Description,
		Type:         property.Type,
		Config:       property.Config,
		WorkspaceID:  property.WorkspaceId,
		CreatedAt:    property.CreatedAt.String(),
		UpdatedAt:    property.UpdatedAt.String(),
	}

	resourceData := propertyState.ToResourceData()
	return &resourceData, nil
}

// LoadResourcesFromRemote loads all properties from the remote catalog
func (p *PropertyProvider) LoadResourcesFromRemote(ctx context.Context) (*resources.ResourceCollection, error) {
	p.log.Debug("loading properties from remote catalog")
	collection := resources.NewResourceCollection()

	// fetch properties from remote
	properties, err := p.client.GetProperties(ctx)
	if err != nil {
		return nil, err
	}

	// Convert slice to map[string]interface{} where key is the property's remoteId
	resourceMap := make(map[string]*resources.RemoteResource)
	for _, property := range properties {
		resourceMap[property.ID] = &resources.RemoteResource{
			ID:         property.ID,
			ExternalID: property.ExternalId,
			Data:       property,
		}
	}
	collection.Set(state.PropertyResourceType, resourceMap)
	return collection, nil
}

func (p *PropertyProvider) LoadStateFromResources(ctx context.Context, collection *resources.ResourceCollection) (*syncerstate.State, error) {
	s := syncerstate.EmptyState()
	properties := collection.GetAll(state.PropertyResourceType)
	for _, remoteProperty := range properties {
		if remoteProperty.ExternalID == "" {
			continue
		}
		property, ok := remoteProperty.Data.(*catalog.Property)
		if !ok {
			return nil, fmt.Errorf("LoadStateFromResources: unable to cast remote resource to catalog.Property")
		}
		args := &state.PropertyArgs{}
		args.FromRemoteProperty(property, collection.GetURNByID)

		stateArgs := state.PropertyState{}
		stateArgs.FromRemoteProperty(property, collection.GetURNByID)

		resourceState := &syncerstate.ResourceState{
			Type:         state.PropertyResourceType,
			ID:           property.ExternalId,
			Input:        args.ToResourceData(),
			Output:       stateArgs.ToResourceData(),
			Dependencies: make([]string, 0),
		}

		urn := resources.URN(property.ExternalId, state.PropertyResourceType)
		s.Resources[urn] = resourceState
	}
	return s, nil
}
