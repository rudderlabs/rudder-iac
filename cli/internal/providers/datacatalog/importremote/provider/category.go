package provider

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/writer"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/importremote/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/samber/lo"
)

const (
	CategoriesRelativePath = "categories/categories.yaml"
	CategoryScope          = "category"
)

var (
	_ WorkspaceImporter = &CategoryImportProvider{}
)

type CategoryImportProvider struct {
	client   catalog.DataCatalog
	log      logger.Logger
	filepath string
}

func NewCategoryImportProvider(client catalog.DataCatalog, log logger.Logger, importDir string) *CategoryImportProvider {
	return &CategoryImportProvider{
		log:      log,
		filepath: filepath.Join(importDir, CategoriesRelativePath),
		client:   client,
	}
}

func (p *CategoryImportProvider) LoadImportable(ctx context.Context, idNamer namer.Namer) (*resources.RemoteResources, error) {
	p.log.Debug("loading importable categories from remote catalog")
	collection := resources.NewRemoteResources()

	categories, err := p.client.GetCategories(ctx, catalog.ListOptions{HasExternalID: lo.ToPtr(false)})
	if err != nil {
		return nil, fmt.Errorf("getting categories from remote catalog: %w", err)
	}

	resourceMap := make(map[string]*resources.RemoteResource)
	for _, category := range categories {
		if category.ExternalID != "" {
			continue
		}

		if category.WorkspaceID == "" {
			continue
		}

		resourceMap[category.ID] = &resources.RemoteResource{
			ID:   category.ID,
			Data: category,
		}
	}

	collection.Set(
		state.CategoryResourceType,
		resourceMap,
	)

	if err := p.idResources(collection, idNamer); err != nil {
		return nil, fmt.Errorf("assigning identifiers to categories: %w", err)
	}

	return collection, nil
}

func (p *CategoryImportProvider) idResources(
	collection *resources.RemoteResources,
	idNamer namer.Namer,
) error {
	p.log.Debug("assigning identifiers to categories")
	categories := collection.GetAll(state.CategoryResourceType)

	for _, category := range categories {
		data, ok := category.Data.(*catalog.Category)
		if !ok {
			return fmt.Errorf("unable to cast remote resource to catalog category")
		}

		externalID, err := idNamer.Name(namer.ScopeName{
			Name:  data.Name,
			Scope: state.CategoryResourceType})
		if err != nil {
			return fmt.Errorf("generating externalID for category %s: %w", data.Name, err)
		}

		category.ExternalID = externalID
		category.Reference = fmt.Sprintf("#/%s/%s/%s",
			localcatalog.KindCategories,
			MetadataNameCategories,
			externalID,
		)
	}
	return nil
}

// FormatForExport formats the categories for export to file
func (p *CategoryImportProvider) FormatForExport(
	collection *resources.RemoteResources,
	idNamer namer.Namer,
	resolver resolver.ReferenceResolver,
) ([]writer.FormattableEntity, error) {
	p.log.Debug("formatting categories for export to file")

	categories := collection.GetAll(state.CategoryResourceType)
	if len(categories) == 0 {
		return nil, nil
	}

	workspaceMetadata := specs.WorkspaceImportMetadata{
		Resources: make([]specs.ImportIds, 0),
	}

	formattedCategories := make([]map[string]any, 0)
	for _, category := range categories {
		p.log.Debug("formatting category", "remoteID", category.ID, "externalID", category.ExternalID)

		data, ok := category.Data.(*catalog.Category)
		if !ok {
			return nil, fmt.Errorf("unable to cast remote resource to catalog category")
		}

		workspaceMetadata.WorkspaceID = data.WorkspaceID // Similar for all the categories
		workspaceMetadata.Resources = append(workspaceMetadata.Resources, specs.ImportIds{
			LocalID:  category.ExternalID,
			RemoteID: category.ID,
		})

		importableCategory := &model.ImportableCategory{}
		formatted, err := importableCategory.ForExport(category.ExternalID, data, resolver)
		if err != nil {
			return nil, fmt.Errorf("formatting category: %w", err)
		}
		formattedCategories = append(formattedCategories, formatted)
	}

	spec, err := toImportSpec(
		localcatalog.KindCategories,
		MetadataNameCategories,
		workspaceMetadata,
		map[string]any{
			"categories": formattedCategories,
		})
	if err != nil {
		return nil, fmt.Errorf("creating spec: %w", err)
	}

	return []writer.FormattableEntity{
		{
			Content:      spec,
			RelativePath: p.filepath,
		},
	}, nil
}
