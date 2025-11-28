package datacatalog

import (
	"context"
	"fmt"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	impProvider "github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/importremote/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	rstate "github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
	"github.com/samber/lo"
)

type CategoryEntityProvider struct {
	*CategoryProvider
	*impProvider.CategoryImportProvider
}

type CategoryProvider struct {
	client catalog.DataCatalog
	log    logger.Logger
}

func NewCategoryProvider(dc catalog.DataCatalog, importDir string) *CategoryEntityProvider {

	cp := &CategoryProvider{
		client: dc,
		log: logger.Logger{
			Logger: logger.New("provider").With("type", "category"),
		},
	}

	imp := impProvider.NewCategoryImportProvider(
		dc,
		logger.Logger{
			Logger: logger.New("importremote.provider").With("type", "category"),
		},
		importDir,
	)

	return &CategoryEntityProvider{
		CategoryProvider:       cp,
		CategoryImportProvider: imp,
	}
}

func (p *CategoryProvider) Create(ctx context.Context, ID string, data resources.ResourceData) (*resources.ResourceData, error) {
	p.log.Debug("creating category resource in upstream catalog", "id", ID)

	toArgs := state.CategoryArgs{}
	toArgs.FromResourceData(data)

	category, err := p.client.CreateCategory(ctx, catalog.CategoryCreate{
		Name:       toArgs.Name,
		ExternalId: ID,
	})

	if err != nil {
		return nil, fmt.Errorf("creating category resource in upstream catalog: %w", err)
	}

	categoryState := state.CategoryState{
		CategoryArgs: toArgs,
		ID:           category.ID,
		Name:         category.Name,
		WorkspaceID:  category.WorkspaceID,
		CreatedAt:    category.CreatedAt.UTC().String(),
		UpdatedAt:    category.UpdatedAt.UTC().String(),
	}

	resourceData := categoryState.ToResourceData()
	return &resourceData, nil
}

func (p *CategoryProvider) Update(ctx context.Context, ID string, input resources.ResourceData, olds resources.ResourceData) (*resources.ResourceData, error) {
	p.log.Debug("updating category resource in upstream catalog", "id", ID)

	toArgs := state.CategoryArgs{}
	toArgs.FromResourceData(input)

	oldState := state.CategoryState{}
	oldState.FromResourceData(olds)

	updated, err := p.client.UpdateCategory(ctx, oldState.ID, catalog.CategoryUpdate{
		Name: toArgs.Name,
	})

	if err != nil {
		return nil, fmt.Errorf("updating category resource in upstream catalog: %w", err)
	}

	toState := state.CategoryState{
		CategoryArgs: toArgs,
		ID:           updated.ID,
		Name:         updated.Name,
		WorkspaceID:  updated.WorkspaceID,
		CreatedAt:    updated.CreatedAt.String(),
		UpdatedAt:    updated.UpdatedAt.String(),
	}

	resourceData := toState.ToResourceData()
	return &resourceData, nil
}

func (p *CategoryProvider) Delete(ctx context.Context, ID string, data resources.ResourceData) error {
	p.log.Debug("deleting category resource in upstream catalog", "id", ID)

	err := p.client.DeleteCategory(ctx, data["id"].(string))

	if err != nil && !catalog.IsCatalogNotFoundError(err) {
		return fmt.Errorf("deleting category resource in upstream catalog: %w", err)
	}

	return nil
}

func (p *CategoryProvider) Import(ctx context.Context, ID string, data resources.ResourceData, remoteId string) (*resources.ResourceData, error) {
	p.log.Debug("importing category resource", "id", ID, "remoteId", remoteId)

	// Get the category from upstream based on the remoteId
	category, err := p.client.GetCategory(ctx, remoteId)
	if err != nil {
		return nil, fmt.Errorf("getting category from upstream: %w", err)
	}

	// Convert input data to CategoryArgs
	toArgs := state.CategoryArgs{}
	toArgs.FromResourceData(data)

	// Check if there are any differences and update if needed
	if toArgs.Name != category.Name {
		p.log.Debug("category has differences, updating", "id", ID, "remoteId", remoteId)
		// Call the updateCategory if there are any differences
		category, err = p.client.UpdateCategory(ctx, remoteId, catalog.CategoryUpdate{
			Name: toArgs.Name,
		})
		if err != nil {
			return nil, fmt.Errorf("updating category during import: %w", err)
		}
	}

	// Set the external ID on the category
	err = p.client.SetCategoryExternalId(ctx, remoteId, ID)
	if err != nil {
		return nil, fmt.Errorf("setting category external id: %w", err)
	}

	// Build and return the category state
	categoryState := state.CategoryState{
		CategoryArgs: toArgs,
		ID:           category.ID,
		Name:         category.Name,
		WorkspaceID:  category.WorkspaceID,
		CreatedAt:    category.CreatedAt.String(),
		UpdatedAt:    category.UpdatedAt.String(),
	}

	resourceData := categoryState.ToResourceData()
	return &resourceData, nil
}

// LoadResourcesFromRemote loads all categories from the remote catalog
func (p *CategoryProvider) LoadResourcesFromRemote(ctx context.Context) (*resources.ResourceCollection, error) {
	p.log.Debug("loading categories from remote catalog")
	collection := resources.NewResourceCollection()

	// fetch categories from remote
	categories, err := p.client.GetCategories(ctx, catalog.ListOptions{HasExternalID: lo.ToPtr(true)})
	if err != nil {
		return nil, err
	}

	// Convert slice to map[string]interface{} where key = category's remoteId
	resourceMap := make(map[string]*resources.RemoteResource)
	for _, category := range categories {
		resourceMap[category.ID] = &resources.RemoteResource{
			ID:         category.ID,
			ExternalID: category.ExternalID,
			Data:       category,
		}
	}
	collection.Set(state.CategoryResourceType, resourceMap)

	return collection, nil
}

func (p *CategoryProvider) MapRemoteToState(collection *resources.ResourceCollection) (*rstate.State, error) {
	s := rstate.EmptyState()
	categories := collection.GetAll(state.CategoryResourceType)
	for _, remoteCategory := range categories {
		if remoteCategory.ExternalID == "" {
			continue
		}
		category, ok := remoteCategory.Data.(*catalog.Category)
		if !ok {
			return nil, fmt.Errorf("MapRemoteToState: unable to cast remote resource to catalog.Category")
		}
		args := &state.CategoryArgs{}
		args.FromRemoteCategory(category, collection.GetURNByID)

		stateArgs := state.CategoryState{}
		stateArgs.FromRemoteCategory(category, collection.GetURNByID)

		resourceState := &rstate.ResourceState{
			Type:         state.CategoryResourceType,
			ID:           category.ExternalID,
			Input:        args.ToResourceData(),
			Output:       stateArgs.ToResourceData(),
			Dependencies: make([]string, 0),
		}

		urn := resources.URN(category.ExternalID, state.CategoryResourceType)
		s.Resources[urn] = resourceState
	}
	return s, nil
}
