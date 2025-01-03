package state

import (
	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/rudderlabs/rudder-iac/cli/pkg/localcatalog"
)

var (
	_ StateMapper = &TrackingPlanState{}
	_ StateMapper = &TrackingPlanArgs{}
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
	// Name           string
	// Description    string
	// EventType      string
	// AllowUnplanned bool
	// Properties     []*TrackingPlanPropertyState
}

func (t *TrackingPlanState) EventByLocalID(localID string) *TrackingPlanEventState {
	for _, event := range t.Events {
		if event.LocalID == localID {
			return event
		}
	}
	return nil
}

// func (t *TrackingPlanEventState) PropertyByLocalID(id string) *TrackingPlanPropertyState {
// 	for _, property := range t.Properties {
// 		if property.LocalID == id {
// 			return property
// 		}
// 	}
// 	return nil
// }

type TrackingPlanPropertyState struct {
	Name        string
	LocalID     string
	Description string
	Type        string
	Config      map[string]interface{}
	Required    bool
}

type TrackingPlanStateDiff struct {
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

type TrackingPlanEventArgs struct {
	Name           string
	LocalID        string
	Description    string
	Type           string
	AllowUnplanned bool
	Properties     []*TrackingPlanPropertyArgs
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
			Name:           event.Name,
			LocalID:        event.LocalID,
			Description:    event.Description,
			Type:           event.Type,
			AllowUnplanned: event.AllowUnplanned,
			Properties:     properties,
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

	events = InterfaceSlice(from, "events", nil)
	// When loading the args from the state []map[string]interface{} is treated as []interface{}
	// but when we have events from catalog being registered as a resource, it is []map[string]interface{}
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
			Name:           MustString(event, "name"),
			Description:    MustString(event, "description"),
			LocalID:        MustString(event, "localId"),
			Type:           MustString(event, "type"),
			AllowUnplanned: MustBool(event, "allowUnplanned"),
			Properties:     make([]*TrackingPlanPropertyArgs, 0),
		}

		// Same issue as the events
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
			"localId":        event.LocalID,
			"name":           event.Name,
			"description":    event.Description,
			"type":           event.Type,
			"allowUnplanned": event.AllowUnplanned,
			"properties":     properties,
		})
	}

	return resources.ResourceData{
		"name":        args.Name,
		"description": args.Description,
		"localId":     args.LocalID,
		"events":      events,
	}
}

func GetUpsertEventPayload(from *TrackingPlanEventArgs) client.TrackingPlanUpsertEvent {
	// Get the properties in correct shape before we can
	// send it to the catalog
	var (
		requiredProps = make([]string, 0)
		propLookup    = make(map[string]interface{})
	)

	// Only for simple types
	for _, prop := range from.Properties {
		propLookup[prop.Name] = map[string]interface{}{
			"type": prop.Type,
		}

		for k, v := range prop.Config {
			propLookup[prop.Name].(map[string]interface{})[k] = v
		}

		// keep on updating the required properties
		if prop.Required {
			requiredProps = append(requiredProps, prop.Name)
		}
	}

	return client.TrackingPlanUpsertEvent{
		Name:        from.Name,
		Description: from.Description,
		EventType:   from.Type,
		Rules: client.TrackingPlanUpsertEventRules{
			Type: "object",
			Properties: struct {
				Properties struct {
					Type                 string                 `json:"type"`
					AdditionalProperties bool                   `json:"additionalProperties"`
					Properties           map[string]interface{} `json:"properties"`
					Required             []string               `json:"required"`
				} `json:"properties"`
			}{
				Properties: struct {
					Type                 string                 `json:"type"`
					AdditionalProperties bool                   `json:"additionalProperties"`
					Properties           map[string]interface{} `json:"properties"`
					Required             []string               `json:"required"`
				}{
					Type:                 "object",
					AdditionalProperties: from.AllowUnplanned,
					Properties:           propLookup, // all the information about properties gets added here
					Required:             requiredProps,
				},
			},
		},
	}
}
