package state

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/rudderlabs/rudder-iac/cli/pkg/localcatalog"
)

const (
	PropertiesIdentity    = "properties"
	TraitsIdentity        = "traits"
	ContextTraitsIdentity = "context.traits"
)

type TrackingPlanState struct {
	TrackingPlanArgs
	ID           string
	Name         string
	Description  string
	Version      int
	CreationType string
	WorkspaceID  string
	CreatedAt    string
	UpdatedAt    string
	Events       []*TrackingPlanEventState
}

func (t *TrackingPlanState) LocalIDForCatalogEventID(eventID string) string {
	for _, event := range t.Events {
		if event.EventID == eventID {
			return event.LocalID
		}
	}
	return ""
}

func (t *TrackingPlanState) CatalogEventIDForLocalID(localID string) string {
	for _, event := range t.Events {
		if event.LocalID == localID {
			return event.EventID
		}
	}
	return ""
}

type TrackingPlanEventState struct {
	ID      string
	LocalID string
	EventID string
}

func (t *TrackingPlanState) EventByLocalID(localID string) *TrackingPlanEventState {
	for _, event := range t.Events {
		if event.LocalID == localID {
			return event
		}
	}
	return nil
}

type TrackingPlanPropertyState struct {
	Name        string
	LocalID     string
	Description string
	Type        string
	Config      map[string]interface{}
	Required    bool
}

type TrackingPlanArgsDiff struct {
	Added   []*TrackingPlanEventArgs
	Updated []*TrackingPlanEventArgs
	Deleted []*TrackingPlanEventArgs
}

func (t *TrackingPlanState) ToResourceData() resources.ResourceData {

	var (
		events []map[string]interface{}
	)

	for _, event := range t.Events {

		events = append(events, map[string]interface{}{
			"id":      event.ID,
			"eventId": event.EventID,
			"localId": event.LocalID,
		})
	}

	return resources.ResourceData{
		"id":               t.ID,
		"name":             t.Name,
		"description":      t.Description,
		"version":          t.Version,
		"creationType":     t.CreationType,
		"workspaceId":      t.WorkspaceID,
		"createdAt":        t.CreatedAt,
		"updatedAt":        t.UpdatedAt,
		"events":           events,
		"trackingPlanArgs": map[string]interface{}(t.TrackingPlanArgs.ToResourceData()),
	}
}

func (t *TrackingPlanState) FromResourceData(from resources.ResourceData) {

	t.ID = MustString(from, "id")
	t.Name = MustString(from, "name")
	t.Description = MustString(from, "description")
	t.Version = int(MustFloat64(from, "version"))
	t.CreationType = MustString(from, "creationType")
	t.WorkspaceID = MustString(from, "workspaceId")
	t.CreatedAt = MustString(from, "createdAt")
	t.UpdatedAt = MustString(from, "updatedAt")
	t.TrackingPlanArgs.FromResourceData(
		MustMapStringInterface(from, "trackingPlanArgs"),
	)

	events := InterfaceSlice(from, "events", nil)
	if len(events) == 0 {
		return
	}

	tpEvents := make([]*TrackingPlanEventState, len(events))
	for idx, event := range events {
		event := event.(map[string]interface{})

		tpEvents[idx] = &TrackingPlanEventState{
			ID:      MustString(event, "id"),
			EventID: MustString(event, "eventId"),
			LocalID: MustString(event, "localId"),
		}
	}

	t.Events = tpEvents
}

// Encapsulates the catalog argument which is added as a resource
// when registering the tracking plan
type TrackingPlanArgs struct {
	Name        string
	LocalID     string
	Description string
	Events      []*TrackingPlanEventArgs
}

func (args TrackingPlanArgs) Diff(other TrackingPlanArgs) *TrackingPlanArgsDiff {

	diffed := &TrackingPlanArgsDiff{
		Added:   make([]*TrackingPlanEventArgs, 0),
		Updated: make([]*TrackingPlanEventArgs, 0),
		Deleted: make([]*TrackingPlanEventArgs, 0),
	}

	for _, otherEvent := range other.Events {
		if args.EventByLocalID(otherEvent.LocalID) == nil {
			diffed.Added = append(diffed.Added, otherEvent)
		}
	}

	for _, event := range args.Events {

		otherEvent := other.EventByLocalID(event.LocalID)

		if otherEvent == nil {
			diffed.Deleted = append(diffed.Deleted, event)
			continue
		}

		if event.Diff(otherEvent) {
			diffed.Updated = append(diffed.Updated, otherEvent)
		}

	}

	return diffed
}

type TrackingPlanEventArgs struct {
	Name            string
	LocalID         string
	Description     string
	Type            string
	AllowUnplanned  bool
	IdentitySection string
	Properties      []*TrackingPlanPropertyArgs
}

func (args *TrackingPlanEventArgs) Diff(other *TrackingPlanEventArgs) bool {
	if args.LocalID != other.LocalID {
		return true
	}

	if args.AllowUnplanned != other.AllowUnplanned {
		return true
	}

	if args.IdentitySection != other.IdentitySection {
		return true
	}

	if len(args.Properties) != len(other.Properties) {
		return true
	}

	for _, prop := range args.Properties {

		otherProp := other.PropertyByLocalID(prop.LocalID)
		if otherProp == nil {
			return true
		}

		if prop.Diff(otherProp) {
			return true
		}
	}

	return false
}

func (args *TrackingPlanEventArgs) PropertyByLocalID(id string) *TrackingPlanPropertyArgs {
	for _, prop := range args.Properties {
		if prop.LocalID == id {
			return prop
		}
	}
	return nil
}

type TrackingPlanPropertyArgs struct {
	Name        string
	LocalID     string
	Description string
	Type        string
	Config      map[string]interface{}
	Required    bool
}

func (args *TrackingPlanPropertyArgs) Diff(other *TrackingPlanPropertyArgs) bool {
	if args.LocalID != other.LocalID {
		return true
	}

	return args.Required != other.Required
}

func (args *TrackingPlanArgs) FromCatalogTrackingPlan(from *localcatalog.TrackingPlan) {

	args.Name = from.Name
	args.LocalID = from.LocalID
	args.Description = from.Description

	events := make([]*TrackingPlanEventArgs, 0, len(from.EventProps))
	for _, event := range from.EventProps {
		properties := make([]*TrackingPlanPropertyArgs, 0, len(event.Properties))
		for _, prop := range event.Properties {
			properties = append(properties, &TrackingPlanPropertyArgs{
				Name:        prop.Name,
				Description: prop.Description,
				LocalID:     prop.LocalID,
				Type:        prop.Type,
				Config:      prop.Config,
				Required:    prop.Required,
			})
		}

		events = append(events, &TrackingPlanEventArgs{
			Name:            event.Name,
			LocalID:         event.LocalID,
			Description:     event.Description,
			Type:            event.Type,
			AllowUnplanned:  event.AllowUnplanned,
			IdentitySection: event.IdentitySection,
			Properties:      properties,
		})
	}

	args.Events = events
}

func (args *TrackingPlanArgs) EventByLocalID(id string) *TrackingPlanEventArgs {

	for _, event := range args.Events {
		if event.LocalID == id {
			return event
		}
	}
	return nil
}

func (args *TrackingPlanArgs) PropertyByLocalID(eventID, id string) *TrackingPlanPropertyArgs {
	event := args.EventByLocalID(eventID)
	if event == nil {
		return nil
	}

	for _, property := range event.Properties {
		if property.LocalID == id {
			return property
		}
	}
	return nil
}

func (args *TrackingPlanArgs) FromResourceData(from resources.ResourceData) {

	args.Name = MustString(from, "name")
	args.Description = MustString(from, "description")
	args.LocalID = MustString(from, "localId")

	var (
		events []interface{}
	)

	// When loading the args from the state []map[string]interface{} is treated as []interface{}
	// but when we have events from catalog being registered as a resource, it is []map[string]interface{}
	events = InterfaceSlice(from, "events", nil)
	if len(events) == 0 {
		eventsMap := MapStringInterfaceSlice(from, "events", nil)
		for _, event := range eventsMap {
			events = append(events, event)
		}
	}

	eventProps := make([]*TrackingPlanEventArgs, len(events))
	for idx, event := range events {
		event := event.(map[string]interface{})

		eventProps[idx] = &TrackingPlanEventArgs{
			Name:            MustString(event, "name"),
			Description:     MustString(event, "description"),
			LocalID:         MustString(event, "localId"),
			Type:            MustString(event, "type"),
			AllowUnplanned:  MustBool(event, "allowUnplanned"),
			IdentitySection: String(event, "identitySection", ""),
			Properties:      make([]*TrackingPlanPropertyArgs, 0),
		}

		// Same situation as the events
		properties := InterfaceSlice(event, "properties", nil)
		if len(properties) == 0 {
			propertiesMap := MapStringInterfaceSlice(event, "properties", nil)
			for _, prop := range propertiesMap {
				properties = append(properties, prop)
			}
		}

		tpProperties := make([]*TrackingPlanPropertyArgs, len(properties))
		for idx, property := range properties {
			property := property.(map[string]interface{})
			tpProperties[idx] = &TrackingPlanPropertyArgs{
				LocalID:     MustString(property, "localId"),
				Name:        MustString(property, "name"),
				Description: MustString(property, "description"),
				Type:        MustString(property, "type"),
				Config:      MapStringInterface(property, "config", make(map[string]interface{})),
				Required:    MustBool(property, "required"),
			}
		}
		eventProps[idx].Properties = tpProperties
	}
	args.Events = eventProps
}

func (args *TrackingPlanArgs) ToResourceData() resources.ResourceData {

	events := make([]map[string]interface{}, 0)
	for _, event := range args.Events {

		properties := make([]map[string]interface{}, 0)
		for _, property := range event.Properties {
			properties = append(properties, map[string]interface{}{
				"name":        property.Name,
				"description": property.Description,
				"localId":     property.LocalID,
				"type":        property.Type,
				"config":      property.Config,
				"required":    property.Required,
			})
		}

		events = append(events, map[string]interface{}{
			"localId":         event.LocalID,
			"name":            event.Name,
			"description":     event.Description,
			"type":            event.Type,
			"allowUnplanned":  event.AllowUnplanned,
			"identitySection": event.IdentitySection,
			"properties":      properties,
		})
	}

	return resources.ResourceData{
		"name":        args.Name,
		"description": args.Description,
		"localId":     args.LocalID,
		"events":      events,
	}
}
