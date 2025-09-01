package datacatalog

import (
	"context"
	"fmt"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
)

type CategoryProvider struct {
	client catalog.DataCatalog
	log    logger.Logger
}

func NewCategoryProvider(dc catalog.DataCatalog) *CategoryProvider {
	return &CategoryProvider{
		client: dc,
		log: logger.Logger{
			Logger: logger.New("provider").With("type", "category"),
		},
	}
}

func (p *CategoryProvider) Create(ctx context.Context, ID string, data resources.ResourceData) (*resources.ResourceData, error) {
	p.log.Debug("creating category resource in upstream catalog", "id", ID)

	toArgs := state.CategoryArgs{}
	toArgs.FromResourceData(data)

	category, err := p.client.CreateCategory(ctx, catalog.CategoryCreate{
		Name:      toArgs.Name,
		ProjectId: toArgs.ProjectId,
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
		Name:      toArgs.Name,
		ProjectId: toArgs.ProjectId,
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
