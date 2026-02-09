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
	EventsRelativePath = "events/events.yaml"
	EventScope         = "event"
)

var (
	_ WorkspaceImporter = &EventImportProvider{}
)

type EventImportProvider struct {
	client        catalog.DataCatalog
	log           logger.Logger
	filepath      string
	v1SpecSupport bool
}

func NewEventImportProvider(client catalog.DataCatalog, log logger.Logger, importDir string) *EventImportProvider {
	return &EventImportProvider{
		log:           log,
		filepath:      filepath.Join(importDir, EventsRelativePath),
		client:        client,
		v1SpecSupport: config.GetConfig().ExperimentalFlags.V1SpecSupport,
	}
}

func (p *EventImportProvider) LoadImportable(ctx context.Context, idNamer namer.Namer) (*resources.RemoteResources, error) {
	p.log.Debug("loading importable events from remote catalog")
	collection := resources.NewRemoteResources()

	events, err := p.client.GetEvents(ctx, catalog.ListOptions{HasExternalID: lo.ToPtr(false)})
	if err != nil {
		return nil, fmt.Errorf("getting events from remote catalog: %w", err)
	}

	resourceMap := make(map[string]*resources.RemoteResource)
	for _, event := range events {
		if event.ExternalID != "" {
			continue
		}
		resourceMap[event.ID] = &resources.RemoteResource{
			ID:   event.ID,
			Data: event,
		}
	}

	collection.Set(
		types.EventResourceType,
		resourceMap,
	)

	if err := p.idResources(collection, idNamer); err != nil {
		return nil, fmt.Errorf("assigning identifiers to events: %w", err)
	}

	return collection, nil
}

func (p *EventImportProvider) idResources(
	collection *resources.RemoteResources,
	idNamer namer.Namer,
) error {
	p.log.Debug("assigning identifiers to events")
	events := collection.GetAll(types.EventResourceType)

	for _, event := range events {
		data, ok := event.Data.(*catalog.Event)
		if !ok {
			return fmt.Errorf("unable to cast remote resource to catalog event")
		}

		name := data.Name
		if data.Name == "" {
			// Identify, Alias events have empty names
			// and hence we use the event-type
			name = data.EventType
		}

		externalID, err := idNamer.Name(namer.ScopeName{
			Name:  name,
			Scope: types.EventResourceType})
		if err != nil {
			return fmt.Errorf("generating externalID for event %s: %w", data.Name, err)
		}

		event.ExternalID = externalID
		event.Reference = fmt.Sprintf("#%s/%s/%s", types.EventResourceType, MetadataNameEvents, externalID)
		if p.v1SpecSupport {
			event.Reference = fmt.Sprintf("#%s:%s", types.EventResourceType, externalID)
		}
	}
	return nil
}

// FormatForExport formats the events for export to file
func (p *EventImportProvider) FormatForExport(
	collection *resources.RemoteResources,
	idNamer namer.Namer,
	resolver resolver.ReferenceResolver,
) ([]writer.FormattableEntity, error) {
	p.log.Debug("formatting events for export to file")

	events := collection.GetAll(types.EventResourceType)
	if len(events) == 0 {
		return nil, nil
	}

	workspaceMetadata := specs.WorkspaceImportMetadata{
		Resources: make([]specs.ImportIds, 0),
	}
	version := specs.SpecVersionV0_1
	if p.v1SpecSupport {
		version = specs.SpecVersionV1
	}

	formattedEvents := make([]map[string]any, 0)
	for _, event := range events {
		p.log.Debug("formatting event", "remoteID", event.ID, "externalID", event.ExternalID)

		data, ok := event.Data.(*catalog.Event)
		if !ok {
			return nil, fmt.Errorf("unable to cast remote resource to catalog event")
		}

		workspaceMetadata.WorkspaceID = data.WorkspaceId // Similar for all the events
		workspaceMetadata.Resources = append(workspaceMetadata.Resources, specs.ImportIds{
			LocalID:  event.ExternalID,
			RemoteID: event.ID,
		})

		var formatted map[string]any
		var err error
		if p.v1SpecSupport {
			importableEvent := &model.ImportableEventV1{}
			formatted, err = importableEvent.ForExport(event.ExternalID, data, resolver)
		} else {
			importableEvent := &model.ImportableEvent{}
			formatted, err = importableEvent.ForExport(event.ExternalID, data, resolver)
		}
		if err != nil {
			return nil, fmt.Errorf("formatting event: %w", err)
		}
		formattedEvents = append(formattedEvents, formatted)
	}

	spec, err := toImportSpec(
		version,
		localcatalog.KindEvents,
		MetadataNameEvents,
		workspaceMetadata,
		map[string]any{
			"events": formattedEvents,
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
