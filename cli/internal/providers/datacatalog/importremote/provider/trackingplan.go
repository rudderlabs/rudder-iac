package provider

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/loader"
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
	trackingPlanFileNameScope = "file-name-trackingplan"
	trackingPlansRelativePath = "trackingplans"
)

var (
	_ WorkspaceImporter = &TrackingPlanImportProvider{}
)

type TrackingPlanImportProvider struct {
	client        catalog.DataCatalog
	log           logger.Logger
	baseImportDir string
	v1SpecSupport bool
}

func NewTrackingPlanImportProvider(client catalog.DataCatalog, log logger.Logger, baseImportDir string) *TrackingPlanImportProvider {
	return &TrackingPlanImportProvider{
		log:           log,
		baseImportDir: baseImportDir,
		client:        client,
		v1SpecSupport: config.GetConfig().ExperimentalFlags.V1SpecSupport,
	}
}

func (p *TrackingPlanImportProvider) LoadImportable(ctx context.Context, idNamer namer.Namer) (*resources.RemoteResources, error) {
	p.log.Debug("loading importable tracking plans from remote catalog")
	collection := resources.NewRemoteResources()

	trackingPlans, err := p.client.GetTrackingPlansWithIdentifiers(ctx, catalog.ListOptions{HasExternalID: lo.ToPtr(false)})
	if err != nil {
		return nil, fmt.Errorf("getting tracking plans from remote catalog: %w", err)
	}

	resourceMap := make(map[string]*resources.RemoteResource)
	for _, trackingPlan := range trackingPlans {
		if trackingPlan.ExternalID != "" {
			continue
		}
		resourceMap[trackingPlan.ID] = &resources.RemoteResource{
			ID:   trackingPlan.ID,
			Data: trackingPlan,
		}
	}

	collection.Set(
		types.TrackingPlanResourceType,
		resourceMap,
	)

	if err := p.idResources(collection, idNamer); err != nil {
		return nil, fmt.Errorf("assigning identifiers to tracking plans: %w", err)
	}

	return collection, nil
}

func (p *TrackingPlanImportProvider) idResources(
	collection *resources.RemoteResources,
	idNamer namer.Namer,
) error {
	p.log.Debug("assigning identifiers to tracking plans")
	trackingPlans := collection.GetAll(types.TrackingPlanResourceType)

	for _, tp := range trackingPlans {
		data, ok := tp.Data.(*catalog.TrackingPlanWithIdentifiers)
		if !ok {
			return fmt.Errorf("unable to cast remote resource to catalog tracking plan")
		}

		externalID, err := idNamer.Name(namer.ScopeName{
			Name:  data.Name,
			Scope: types.TrackingPlanResourceType})
		if err != nil {
			return fmt.Errorf("generating externalID for tracking plan %s: %w", data.Name, err)
		}

		tp.ExternalID = externalID
		tp.Reference = fmt.Sprintf("#tp:%s", externalID)
	}
	return nil
}

// FormatForExport formats the tracking plans for export to file
func (p *TrackingPlanImportProvider) FormatForExport(
	collection *resources.RemoteResources,
	idNamer namer.Namer,
	resolver resolver.ReferenceResolver,
) ([]writer.FormattableEntity, error) {
	p.log.Debug("formatting tracking plans for export to file")

	trackingPlans := collection.GetAll(types.TrackingPlanResourceType)
	if len(trackingPlans) == 0 {
		return nil, nil
	}

	formattables := make([]writer.FormattableEntity, 0)
	for _, trackingPlan := range trackingPlans {
		p.log.Debug("formatting tracking plan", "remoteID", trackingPlan.ID, "externalID", trackingPlan.ExternalID)

		data, ok := trackingPlan.Data.(*catalog.TrackingPlanWithIdentifiers)
		if !ok {
			return nil, fmt.Errorf("unable to cast remote resource: %s to catalog tracking plan", trackingPlan.ID)
		}

		workspaceMetadata := specs.WorkspaceImportMetadata{
			WorkspaceID: data.WorkspaceID,
			Resources: []specs.ImportIds{
				{
					LocalID:  trackingPlan.ExternalID,
					RemoteID: trackingPlan.ID,
				},
			},
		}

		var formatted map[string]any
		var err error
		if p.v1SpecSupport {
			importableTrackingPlan := &model.ImportableTrackingPlanV1{}
			formatted, err = importableTrackingPlan.ForExport(trackingPlan.ExternalID, data, resolver, idNamer)
		} else {
			importableTrackingPlan := &model.ImportableTrackingPlan{}
			formatted, err = importableTrackingPlan.ForExport(trackingPlan.ExternalID, data, resolver, idNamer)
		}
		if err != nil {
			return nil, fmt.Errorf("formatting tracking plan %s for export: %w", trackingPlan.ID, err)
		}

		kind := localcatalog.KindTrackingPlans
		if p.v1SpecSupport {
			kind = localcatalog.KindTrackingPlansV1
		}

		spec, err := toImportSpec(
			kind,
			trackingPlan.ExternalID,
			workspaceMetadata,
			formatted,
		)
		if err != nil {
			return nil, fmt.Errorf("creating spec for tracking plan %s: %w", trackingPlan.ID, err)
		}

		fName, err := idNamer.Name(namer.ScopeName{
			Name:  trackingPlan.ExternalID,
			Scope: trackingPlanFileNameScope,
		})
		if err != nil {
			return nil, fmt.Errorf("generating file path for tracking plan %s: %w", trackingPlan.ID, err)
		}

		formattables = append(formattables, writer.FormattableEntity{
			Content: spec,
			RelativePath: filepath.Join(
				p.baseImportDir,
				trackingPlansRelativePath,
				fmt.Sprintf("%s%s", fName, loader.ExtensionYAML),
			),
		})

	}

	return formattables, nil
}
