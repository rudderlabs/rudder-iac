package provider

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/importmanifest"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/loader"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/writer"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/importremote/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/types"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
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
}

func NewTrackingPlanImportProvider(client catalog.DataCatalog, log logger.Logger, baseImportDir string) *TrackingPlanImportProvider {
	return &TrackingPlanImportProvider{
		log:           log,
		baseImportDir: baseImportDir,
		client:        client,
	}
}

func (p *TrackingPlanImportProvider) LoadImportable(ctx context.Context, idNamer namer.Namer, filter ...resources.ImportableFilter) (*resources.RemoteResources, error) {
	p.log.Debug("loading importable tracking plans from remote catalog")
	collection := resources.NewRemoteResources()
	f := resources.ImportableFilterOf(filter)

	trackingPlans, err := p.client.GetTrackingPlansWithIdentifiers(ctx, catalog.ListOptions{HasExternalID: f.UnmanagedOnly()})
	if err != nil {
		return nil, fmt.Errorf("getting tracking plans from remote catalog: %w", err)
	}

	resourceMap := make(map[string]*resources.RemoteResource)
	for _, trackingPlan := range trackingPlans {
		if trackingPlan.ExternalID != "" && !f.IncludeManaged {
			continue
		}
		resourceMap[trackingPlan.ID] = &resources.RemoteResource{
			ID: trackingPlan.ID,
			// Carried through so idResources keeps an already-managed
			// resource's upstream identifier instead of renaming it.
			ExternalID: f.KeepID(trackingPlan.ExternalID),
			Data:       trackingPlan,
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

	candidates := make([]namer.IDCandidate, 0, len(trackingPlans))
	for id, tp := range trackingPlans {
		data, ok := tp.Data.(*catalog.TrackingPlanWithIdentifiers)
		if !ok {
			return fmt.Errorf("unable to cast remote resource to catalog tracking plan")
		}
		candidates = append(candidates, namer.IDCandidate{
			Key:        id,
			Name:       data.Name,
			ExternalID: tp.ExternalID,
		})
	}

	externalIDs, err := namer.ResolveIDs(idNamer, types.TrackingPlanResourceType, candidates)
	if err != nil {
		return err
	}

	for id, tp := range trackingPlans {
		tp.ExternalID = externalIDs[id]
		tp.Reference = fmt.Sprintf("#%s:%s", localcatalog.KindTrackingPlansV1, tp.ExternalID)
	}
	return nil
}

// FormatForExport formats the tracking plans for export to file
func (p *TrackingPlanImportProvider) FormatForExport(
	collection *resources.RemoteResources,
	idNamer namer.Namer,
	resolver resolver.ReferenceResolver,
) ([]writer.FormattableEntity, []importmanifest.ImportEntry, error) {
	p.log.Debug("formatting tracking plans for export to file")

	trackingPlans := collection.GetAll(types.TrackingPlanResourceType)
	if len(trackingPlans) == 0 {
		return nil, nil, nil
	}

	formattables := make([]writer.FormattableEntity, 0)
	var entries []importmanifest.ImportEntry
	for _, trackingPlan := range trackingPlans {
		p.log.Debug("formatting tracking plan", "remoteID", trackingPlan.ID, "externalID", trackingPlan.ExternalID)

		data, ok := trackingPlan.Data.(*catalog.TrackingPlanWithIdentifiers)
		if !ok {
			return nil, nil, fmt.Errorf("unable to cast remote resource: %s to catalog tracking plan", trackingPlan.ID)
		}

		urn := resources.URN(trackingPlan.ExternalID, types.TrackingPlanResourceType)

		// Matched tracking plans (import --merge) adopt an existing local
		// spec: manifest entry only — no spec file is written for them.
		if trackingPlan.MatchedWith != nil {
			entries = append(entries, importmanifest.ImportEntry{
				WorkspaceID: data.WorkspaceID,
				URN:         urn,
				RemoteID:    trackingPlan.ID,
			})
			continue
		}

		workspaceMetadata := specs.WorkspaceImportMetadata{
			WorkspaceID: data.WorkspaceID,
			Resources: []specs.ImportIds{
				{
					URN:      urn,
					RemoteID: trackingPlan.ID,
				},
			},
		}
		entries = append(entries, importEntriesFromWorkspace(workspaceMetadata)...)

		importableTrackingPlan := &model.ImportableTrackingPlanV1{}
		formatted, err := importableTrackingPlan.ForExport(trackingPlan.ExternalID, data, resolver, idNamer)
		if err != nil {
			return nil, nil, fmt.Errorf("formatting tracking plan %s for export: %w", trackingPlan.ID, err)
		}

		kind := localcatalog.KindTrackingPlansV1
		version := specs.SpecVersionV1

		spec, err := toImportSpec(
			version,
			kind,
			trackingPlan.ExternalID,
			workspaceMetadata,
			formatted,
		)
		if err != nil {
			return nil, nil, fmt.Errorf("creating spec for tracking plan %s: %w", trackingPlan.ID, err)
		}

		fName, err := idNamer.Name(namer.ScopeName{
			Name:  trackingPlan.ExternalID,
			Scope: trackingPlanFileNameScope,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("generating file path for tracking plan %s: %w", trackingPlan.ID, err)
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

	return formattables, entries, nil
}
