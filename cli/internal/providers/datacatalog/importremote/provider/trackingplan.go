package provider

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/importremote"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/loader"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/importremote/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
)

const (
	trackingPlanFileNameScope = "file-name-trackingplan"
	trackingPlansRelativePath = "trackingplans"
)

var (
	_ importremote.WorkspaceImporter = &TrackingPlanImportProvider{}
)

type TrackingPlanImportProvider struct {
	client        catalog.DataCatalog
	log           logger.Logger
	baseImportDir string
}

func NewTrackingPlanImportProvider(client catalog.DataCatalog, log logger.Logger, baseImportDir string) *TrackingPlanImportProvider {
	return &TrackingPlanImportProvider{
		log:           log,
		baseImportDir: baseImportDir,
		client:        client,
	}
}

func (p *TrackingPlanImportProvider) LoadImportable(ctx context.Context, idNamer namer.Namer) (*resources.ResourceCollection, error) {
	p.log.Debug("loading importable tracking plans from remote catalog")
	collection := resources.NewResourceCollection()

	trackingPlans, err := p.client.GetTrackingPlans(ctx)
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
		state.TrackingPlanResourceType,
		resourceMap,
	)

	if err := p.idResources(collection, idNamer); err != nil {
		return nil, fmt.Errorf("assigning identifiers to tracking plans: %w", err)
	}

	return collection, nil
}

func (p *TrackingPlanImportProvider) idResources(
	collection *resources.ResourceCollection,
	idNamer namer.Namer,
) error {
	p.log.Debug("assigning identifiers to tracking plans")
	trackingPlans := collection.GetAll(state.TrackingPlanResourceType)

	for _, tp := range trackingPlans {
		data, ok := tp.Data.(*catalog.TrackingPlanWithIdentifiers)
		if !ok {
			return fmt.Errorf("unable to cast remote resource to catalog tracking plan")
		}

		externalID, err := idNamer.Name(namer.ScopeName{
			Name:  data.Name,
			Scope: state.TrackingPlanResourceType})
		if err != nil {
			return fmt.Errorf("generating externalID for tracking plan %s: %w", data.Name, err)
		}

		tp.ExternalID = externalID
		tp.Reference = fmt.Sprintf("#/%s/%s/%s",
			localcatalog.KindTrackingPlanForReference,
			externalID,
			externalID,
		)
	}
	return nil
}

// FormatForExport formats the tracking plans for export to file
func (p *TrackingPlanImportProvider) FormatForExport(
	ctx context.Context,
	collection *resources.ResourceCollection,
	idNamer namer.Namer,
	resolver resolver.ReferenceResolver,
) ([]importremote.FormattableEntity, error) {
	p.log.Debug("formatting tracking plans for export to file")

	trackingPlans := collection.GetAll(state.TrackingPlanResourceType)
	if len(trackingPlans) == 0 {
		return nil, nil
	}

	formattables := make([]importremote.FormattableEntity, 0)
	for _, trackingPlan := range trackingPlans {
		p.log.Debug("formatting tracking plan", "remoteID", trackingPlan.ID, "externalID", trackingPlan.ExternalID)

		data, ok := trackingPlan.Data.(*catalog.TrackingPlanWithIdentifiers)
		if !ok {
			return nil, fmt.Errorf("unable to cast remote resource: %s to catalog tracking plan", trackingPlan.ID)
		}

		workspaceMetadata := importremote.WorkspaceImportMetadata{
			WorkspaceID: data.WorkspaceID,
			Resources: []importremote.ImportIds{
				{
					LocalID:  trackingPlan.ExternalID,
					RemoteID: trackingPlan.ID,
				},
			},
		}

		importableTrackingPlan := &model.ImportableTrackingPlan{}
		formatted, err := importableTrackingPlan.ForExport(trackingPlan.ExternalID, data, resolver, idNamer)
		if err != nil {
			return nil, fmt.Errorf("formatting tracking plan %s for export: %w", trackingPlan.ID, err)
		}

		spec, err := toImportSpec(
			localcatalog.KindTrackingPlans,
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

		formattables = append(formattables, importremote.FormattableEntity{
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
