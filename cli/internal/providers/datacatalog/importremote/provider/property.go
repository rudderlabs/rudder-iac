package provider

import (
	"context"
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/importremote"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/importremote/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
)

const (
	SpecVersion  = "rudder/v0.1"
	Kind         = "properties"
	MetadataName = "properties"
	RelativePath = "./imported/data-catalog/properties/properties.yaml"
)

var (
	_ importremote.WorkspaceImporter = &PropertyImportProvider{}
)

type PropertyImportProvider struct {
	client catalog.DataCatalog
	log    logger.Logger
}

func NewPropertyImportProvider(client catalog.DataCatalog, log logger.Logger) *PropertyImportProvider {
	return &PropertyImportProvider{
		log:    log,
		client: client,
	}
}

func (p *PropertyImportProvider) LoadImportable(ctx context.Context) (*resources.ResourceCollection, error) {
	p.log.Debug("loading importable properties from remote catalog")
	collection := resources.NewResourceCollection()

	properties, err := p.client.GetProperties(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting properties from remote catalog: %w", err)
	}

	resourceMap := make(map[string]*resources.RemoteResource)
	for _, property := range properties {
		resourceMap[property.ID] = &resources.RemoteResource{
			ID:   property.ID,
			Data: property,
		}
	}
	collection.Set(state.PropertyResourceType, resourceMap)

	return collection, nil
}

func (p *PropertyImportProvider) IDResources(ctx context.Context, collection *resources.ResourceCollection, idNamer namer.Namer) error {
	p.log.Debug("assigning identifiers to properties")

	properties := collection.GetAll(state.PropertyResourceType)

	for _, property := range properties {
		data, ok := property.Data.(*catalog.Property)
		if !ok {
			return fmt.Errorf("unable to cast remote resource to catalog property")
		}

		externalID, err := idNamer.Name(data.Name)
		if err != nil {
			return fmt.Errorf("generating externalID for property %s: %w", data.Name, err)
		}

		property.ExternalID = externalID
		property.Reference = fmt.Sprintf("#/properties/%s/%s", MetadataName, externalID)
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

	spec, err := p.toSpec(formattedProps, workspaceMetadata)
	if err != nil {
		return nil, fmt.Errorf("creating spec: %w", err)
	}

	return []importremote.FormattableEntity{
		{
			Content:      spec,
			RelativePath: RelativePath,
		},
	}, nil
}

func (p *PropertyImportProvider) toSpec(properties []map[string]any, workspaceMetadata importremote.WorkspaceImportMetadata) (*specs.Spec, error) {
	metadata := importremote.Metadata{
		Name: MetadataName,
		Import: importremote.WorkspacesImportMetadata{
			Workspaces: []importremote.WorkspaceImportMetadata{workspaceMetadata},
		},
	}

	metadataMap := make(map[string]any)
	err := mapstructure.Decode(metadata, &metadataMap)
	if err != nil {
		return nil, fmt.Errorf("decoding metadata: %w", err)
	}

	return &specs.Spec{
		Version:  SpecVersion,
		Kind:     Kind,
		Metadata: metadataMap,
		Spec: map[string]any{
			"properties": properties,
		},
	}, nil
}
