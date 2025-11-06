package provider

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/importremote"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/importremote/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/samber/lo"
)

const (
	CustomTypesRelativePath = "custom-types/custom-types.yaml"
)

var (
	_ importremote.WorkspaceImporter = &CustomTypeImportProvider{}
)

type CustomTypeImportProvider struct {
	client   catalog.DataCatalog
	log      logger.Logger
	filepath string
}

func NewCustomTypeImportProvider(client catalog.DataCatalog, log logger.Logger, importDir string) *CustomTypeImportProvider {
	return &CustomTypeImportProvider{
		log:      log,
		filepath: filepath.Join(importDir, CustomTypesRelativePath),
		client:   client,
	}
}

func (p *CustomTypeImportProvider) LoadImportable(ctx context.Context, idNamer namer.Namer) (*resources.ResourceCollection, error) {
	p.log.Debug("loading importable custom types from remote catalog")
	collection := resources.NewResourceCollection()

	customTypes, err := p.client.GetCustomTypes(ctx, catalog.ListOptions{HasExternalId: lo.ToPtr(false)})
	if err != nil {
		return nil, fmt.Errorf("getting custom types from remote catalog: %w", err)
	}

	resourceMap := make(map[string]*resources.RemoteResource)
	for _, customType := range customTypes {
		if customType.ExternalId != "" {
			continue
		}
		resourceMap[customType.ID] = &resources.RemoteResource{
			ID:   customType.ID,
			Data: customType,
		}
	}

	collection.Set(
		state.CustomTypeResourceType,
		resourceMap,
	)

	if err := p.idResources(collection, idNamer); err != nil {
		return nil, fmt.Errorf("assigning identifiers to custom types: %w", err)
	}

	return collection, nil
}

func (p *CustomTypeImportProvider) idResources(
	collection *resources.ResourceCollection,
	idNamer namer.Namer,
) error {
	p.log.Debug("assigning identifiers to custom types")
	customTypes := collection.GetAll(state.CustomTypeResourceType)

	for _, customType := range customTypes {
		data, ok := customType.Data.(*catalog.CustomType)
		if !ok {
			return fmt.Errorf("unable to cast remote resource to catalog custom type")
		}

		externalID, err := idNamer.Name(namer.ScopeName{
			Name:  data.Name,
			Scope: state.CustomTypeResourceType})
		if err != nil {
			return fmt.Errorf("generating externalID for custom type %s: %w", data.Name, err)
		}

		customType.ExternalID = externalID
		customType.Reference = fmt.Sprintf("#/%s/%s/%s",
			localcatalog.KindCustomTypes,
			MetadataNameCustomTypes,
			externalID,
		)
	}
	return nil
}

// FormatForExport formats custom types for export to file
func (p *CustomTypeImportProvider) FormatForExport(
	ctx context.Context,
	collection *resources.ResourceCollection,
	idNamer namer.Namer,
	resolver resolver.ReferenceResolver,
) ([]importremote.FormattableEntity, error) {
	p.log.Debug("formatting custom types for export to file")

	customTypes := collection.GetAll(state.CustomTypeResourceType)
	if len(customTypes) == 0 {
		return nil, nil
	}

	workspaceMetadata := importremote.WorkspaceImportMetadata{
		Resources: make([]importremote.ImportIds, 0),
	}

	formattedTypes := make([]map[string]any, 0)
	for _, customType := range customTypes {
		p.log.Debug("formatting custom type", "remoteID", customType.ID, "externalID", customType.ExternalID)

		data, ok := customType.Data.(*catalog.CustomType)
		if !ok {
			return nil, fmt.Errorf("unable to cast remote resource to catalog custom type")
		}

		workspaceMetadata.WorkspaceID = data.WorkspaceId // Similar for all the custom types
		workspaceMetadata.Resources = append(workspaceMetadata.Resources, importremote.ImportIds{
			LocalID:  customType.ExternalID,
			RemoteID: customType.ID,
		})

		importableCustomType := &model.ImportableCustomType{}
		formatted, err := importableCustomType.ForExport(customType.ExternalID, data, resolver)
		if err != nil {
			return nil, fmt.Errorf("formatting custom type: %w", err)
		}
		formattedTypes = append(formattedTypes, formatted)
	}

	spec, err := toImportSpec(
		localcatalog.KindCustomTypes,
		MetadataNameCustomTypes,
		workspaceMetadata,
		map[string]any{
			"types": formattedTypes,
		})
	if err != nil {
		return nil, fmt.Errorf("creating spec: %w", err)
	}

	return []importremote.FormattableEntity{
		{
			Content:      spec,
			RelativePath: p.filepath,
		},
	}, nil
}
