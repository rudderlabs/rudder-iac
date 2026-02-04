package provider

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
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
	CustomTypesRelativePath = "custom-types/custom-types.yaml"
)

var (
	_ WorkspaceImporter = &CustomTypeImportProvider{}
)

type CustomTypeImportProvider struct {
	client        catalog.DataCatalog
	log           logger.Logger
	filepath      string
	v1SpecSupport bool
}

func NewCustomTypeImportProvider(client catalog.DataCatalog, log logger.Logger, importDir string) *CustomTypeImportProvider {
	return &CustomTypeImportProvider{
		log:           log,
		filepath:      filepath.Join(importDir, CustomTypesRelativePath),
		client:        client,
		v1SpecSupport: config.GetConfig().ExperimentalFlags.V1SpecSupport,
	}
}

func (p *CustomTypeImportProvider) LoadImportable(ctx context.Context, idNamer namer.Namer) (*resources.RemoteResources, error) {
	p.log.Debug("loading importable custom types from remote catalog")
	collection := resources.NewRemoteResources()

	customTypes, err := p.client.GetCustomTypes(ctx, catalog.ListOptions{HasExternalID: lo.ToPtr(false)})
	if err != nil {
		return nil, fmt.Errorf("getting custom types from remote catalog: %w", err)
	}

	resourceMap := make(map[string]*resources.RemoteResource)
	for _, customType := range customTypes {
		if customType.ExternalID != "" {
			continue
		}
		resourceMap[customType.ID] = &resources.RemoteResource{
			ID:   customType.ID,
			Data: customType,
		}
	}

	collection.Set(
		types.CustomTypeResourceType,
		resourceMap,
	)

	if err := p.idResources(collection, idNamer); err != nil {
		return nil, fmt.Errorf("assigning identifiers to custom types: %w", err)
	}

	return collection, nil
}

func (p *CustomTypeImportProvider) idResources(
	collection *resources.RemoteResources,
	idNamer namer.Namer,
) error {
	p.log.Debug("assigning identifiers to custom types")
	customTypes := collection.GetAll(types.CustomTypeResourceType)

	for _, customType := range customTypes {
		data, ok := customType.Data.(*catalog.CustomType)
		if !ok {
			return fmt.Errorf("unable to cast remote resource to catalog custom type")
		}

		externalID, err := idNamer.Name(namer.ScopeName{
			Name:  data.Name,
			Scope: types.CustomTypeResourceType})
		if err != nil {
			return fmt.Errorf("generating externalID for custom type %s: %w", data.Name, err)
		}

		customType.ExternalID = externalID
		customType.Reference = fmt.Sprintf("#%s:%s", types.CustomTypeResourceType, externalID)
	}
	return nil
}

// FormatForExport formats custom types for export to file
func (p *CustomTypeImportProvider) FormatForExport(
	collection *resources.RemoteResources,
	idNamer namer.Namer,
	resolver resolver.ReferenceResolver,
) ([]writer.FormattableEntity, error) {
	p.log.Debug("formatting custom types for export to file")

	customTypes := collection.GetAll(types.CustomTypeResourceType)
	if len(customTypes) == 0 {
		return nil, nil
	}

	workspaceMetadata := specs.WorkspaceImportMetadata{
		Resources: make([]specs.ImportIds, 0),
	}

	formattedTypes := make([]map[string]any, 0)
	for _, customType := range customTypes {
		p.log.Debug("formatting custom type", "remoteID", customType.ID, "externalID", customType.ExternalID)

		data, ok := customType.Data.(*catalog.CustomType)
		if !ok {
			return nil, fmt.Errorf("unable to cast remote resource to catalog custom type")
		}

		workspaceMetadata.WorkspaceID = data.WorkspaceId // Similar for all the custom types
		workspaceMetadata.Resources = append(workspaceMetadata.Resources, specs.ImportIds{
			LocalID:  customType.ExternalID,
			RemoteID: customType.ID,
		})

		var formatted map[string]any
		var err error
		if p.v1SpecSupport {
			importableCustomType := &model.ImportableCustomTypeV1{}
			formatted, err = importableCustomType.ForExport(customType.ExternalID, data, resolver)
		} else {
			importableCustomType := &model.ImportableCustomType{}
			formatted, err = importableCustomType.ForExport(customType.ExternalID, data, resolver)
		}
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

	return []writer.FormattableEntity{
		{
			Content:      spec,
			RelativePath: p.filepath,
		},
	}, nil
}
