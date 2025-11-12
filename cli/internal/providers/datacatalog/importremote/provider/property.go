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
	PropertiesRelativePath = "properties/properties.yaml"
	PropertyScope          = "property"
)

var (
	_ importremote.WorkspaceImporter = &PropertyImportProvider{}
)

type PropertyImportProvider struct {
	client   catalog.DataCatalog
	log      logger.Logger
	filepath string
}

func NewPropertyImportProvider(client catalog.DataCatalog, log logger.Logger, importDir string) *PropertyImportProvider {
	return &PropertyImportProvider{
		log:      log,
		filepath: filepath.Join(importDir, PropertiesRelativePath),
		client:   client,
	}
}

func (p *PropertyImportProvider) LoadImportable(ctx context.Context, idNamer namer.Namer) (*resources.ResourceCollection, error) {
	p.log.Debug("loading importable properties from remote catalog")
	collection := resources.NewResourceCollection()

	properties, err := p.client.GetProperties(ctx, catalog.ListOptions{HasExternalId: lo.ToPtr(false)})
	if err != nil {
		return nil, fmt.Errorf("getting properties from remote catalog: %w", err)
	}

	resourceMap := make(map[string]*resources.RemoteResource)
	for _, property := range properties {
		if property.ExternalId != "" {
			continue
		}
		resourceMap[property.ID] = &resources.RemoteResource{
			ID:   property.ID,
			Data: property,
		}
	}

	collection.Set(
		state.PropertyResourceType,
		resourceMap,
	)

	if err := p.idResources(collection, idNamer); err != nil {
		return nil, fmt.Errorf("assigning identifiers to properties: %w", err)
	}

	return collection, nil
}

func (p *PropertyImportProvider) idResources(
	collection *resources.ResourceCollection,
	idNamer namer.Namer,
) error {
	p.log.Debug("assigning identifiers to properties")
	properties := collection.GetAll(state.PropertyResourceType)

	for _, property := range properties {
		data, ok := property.Data.(*catalog.Property)
		if !ok {
			return fmt.Errorf("unable to cast remote resource to catalog property")
		}

		externalID, err := idNamer.Name(namer.ScopeName{
			Name:  data.Name,
			Scope: state.PropertyResourceType})
		if err != nil {
			return fmt.Errorf("generating externalID for property %s: %w", data.Name, err)
		}

		property.ExternalID = externalID
		property.Reference = fmt.Sprintf("#/%s/%s/%s",
			localcatalog.KindProperties,
			MetadataNameProperties,
			externalID,
		)
	}
	return nil
}

// NormalizeForImport normalizes the properties for import
func (p *PropertyImportProvider) FormatForExport(
	ctx context.Context,
	collection *resources.ResourceCollection,
	idNamer namer.Namer,
	resolver resolver.ReferenceResolver,
) ([]importremote.FormattableEntity, error) {
	p.log.Debug("formatting properties for export to file")

	properties := collection.GetAll(state.PropertyResourceType)
	if len(properties) == 0 {
		return nil, nil
	}

	workspaceMetadata := importremote.WorkspaceImportMetadata{
		Resources: make([]importremote.ImportIds, 0),
	}

	formattedProps := make([]map[string]any, 0)
	for _, property := range properties {
		p.log.Debug("formatting property", "remoteID", property.ID, "externalID", property.ExternalID)

		data, ok := property.Data.(*catalog.Property)
		if !ok {
			return nil, fmt.Errorf("unable to cast remote resource to catalog property")
		}

		workspaceMetadata.WorkspaceID = data.WorkspaceId // Similar for all the properties
		workspaceMetadata.Resources = append(workspaceMetadata.Resources, importremote.ImportIds{
			LocalID:  property.ExternalID,
			RemoteID: property.ID,
		})

		importableProp := &model.ImportableProperty{}
		formatted, err := importableProp.ForExport(property.ExternalID, data, resolver)
		if err != nil {
			return nil, fmt.Errorf("formatting property: %w", err)
		}
		formattedProps = append(formattedProps, formatted)
	}

	spec, err := toImportSpec(
		localcatalog.KindProperties,
		MetadataNameProperties,
		workspaceMetadata,
		map[string]any{
			"properties": formattedProps,
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
