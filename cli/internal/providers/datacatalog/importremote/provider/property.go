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
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/types"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/samber/lo"
)

const (
	PropertiesRelativePath = "properties/properties.yaml"
	PropertyScope          = "property"
)

var (
	_ WorkspaceImporter = &PropertyImportProvider{}
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

func (p *PropertyImportProvider) LoadImportable(ctx context.Context, idNamer namer.Namer) (*resources.RemoteResources, error) {
	p.log.Debug("loading importable properties from remote catalog")
	collection := resources.NewRemoteResources()

	properties, err := p.client.GetProperties(ctx, catalog.ListOptions{HasExternalID: lo.ToPtr(false)})
	if err != nil {
		return nil, fmt.Errorf("getting properties from remote catalog: %w", err)
	}

	resourceMap := make(map[string]*resources.RemoteResource)
	for _, property := range properties {
		if property.ExternalID != "" {
			continue
		}
		resourceMap[property.ID] = &resources.RemoteResource{
			ID:   property.ID,
			Data: property,
		}
	}

	collection.Set(
		types.PropertyResourceType,
		resourceMap,
	)

	if err := p.idResources(collection, idNamer); err != nil {
		return nil, fmt.Errorf("assigning identifiers to properties: %w", err)
	}

	return collection, nil
}

func (p *PropertyImportProvider) idResources(
	collection *resources.RemoteResources,
	idNamer namer.Namer,
) error {
	p.log.Debug("assigning identifiers to properties")
	properties := collection.GetAll(types.PropertyResourceType)

	for _, property := range properties {
		data, ok := property.Data.(*catalog.Property)
		if !ok {
			return fmt.Errorf("unable to cast remote resource to catalog property")
		}

		externalID, err := idNamer.Name(namer.ScopeName{
			Name:  data.Name,
			Scope: types.PropertyResourceType})
		if err != nil {
			return fmt.Errorf("generating externalID for property %s: %w", data.Name, err)
		}

		property.ExternalID = externalID
		property.Reference = fmt.Sprintf("#%s:%s", types.PropertyResourceType, externalID)
	}
	return nil
}

// NormalizeForImport normalizes the properties for import
func (p *PropertyImportProvider) FormatForExport(
	collection *resources.RemoteResources,
	idNamer namer.Namer,
	resolver resolver.ReferenceResolver,
) ([]writer.FormattableEntity, error) {
	p.log.Debug("formatting properties for export to file")

	properties := collection.GetAll(types.PropertyResourceType)
	if len(properties) == 0 {
		return nil, nil
	}

	workspaceMetadata := specs.WorkspaceImportMetadata{
		Resources: make([]specs.ImportIds, 0),
	}

	formattedProps := make([]map[string]any, 0)
	for _, property := range properties {
		p.log.Debug("formatting property", "remoteID", property.ID, "externalID", property.ExternalID)

		data, ok := property.Data.(*catalog.Property)
		if !ok {
			return nil, fmt.Errorf("unable to cast remote resource to catalog property")
		}

		workspaceMetadata.WorkspaceID = data.WorkspaceId // Similar for all the properties
		workspaceMetadata.Resources = append(workspaceMetadata.Resources, specs.ImportIds{
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

	return []writer.FormattableEntity{
		{
			Content:      spec,
			RelativePath: p.filepath,
		},
	}, nil
}
