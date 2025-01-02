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
	ID           string                    `json:"id"`
	Name         string                    `json:"name"`
	Description  string                    `json:"description,omitempty"`
	Version      int                       `json:"version"`
	CreationType string                    `json:"creationType"`
	WorkspaceID  string                    `json:"workspaceId"`
	CreatedAt    string                    `json:"created_at"`
	UpdatedAt    string                    `json:"updated_at"`
	Events       []*TrackingPlanEventState `json:"events"`
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
	ID             string                       `json:"id"`
	EventID        string                       `json:"eventId"`
	LocalID        string                       `json:"localId"`
	Name           string                       `json:"name"`
	Description    string                       `json:"description"`
	EventType      string                       `json:"eventType"`
	AllowUnplanned bool                         `json:"allowUnplanned"`
	Properties     []*TrackingPlanPropertyState `json:"properties"`
}

func (t *TrackingPlanState) EventByLocalID(id string) *TrackingPlanEventState {
	for _, event := range t.Events {
		if event.ID == id {
			return event
		}
	}
	return nil
}

func (t *TrackingPlanEventState) PropertyByLocalID(id string) *TrackingPlanPropertyState {
	for _, property := range t.Properties {
		if property.LocalID == id {
			return property
		}
	}
	return nil
}

type TrackingPlanPropertyState struct {
	Name        string                 `json:"name"`
	LocalID     string                 `json:"localId"`
	Description string                 `json:"description"`
	Type        string                 `json:"type"`
	Config      map[string]interface{} `json:"config"`
	Required    bool                   `json:"required"`
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

		var properties []map[string]interface{}
		for _, property := range event.Properties {
			properties = append(properties, map[string]interface{}{
				"name":        property.Name,
				"description": property.Description,
				"type":        property.Type,
				"config":      property.Config,
				"required":    property.Required,
			})
		}
		events = append(events, map[string]interface{}{
			"id":             event.ID,
			"eventId":        event.EventID,
			"name":           event.Name,
			"description":    event.Description,
			"eventType":      event.EventType,
			"allowUnplanned": event.AllowUnplanned,
			"properties":     properties,
		})
	}

	return resources.ResourceData{
		"id":                t.ID,
		"name":              t.Name,
		"description":       t.Description,
		"version":           t.Version,
		"creationType":      t.CreationType,
		"workspaceId":       t.WorkspaceID,
		"created_at":        t.CreatedAt,
		"updated_at":        t.UpdatedAt,
		"events":            events,
		"trackingplan_args": t.TrackingPlanArgs.ToResourceData(),
	}
}

func (t *TrackingPlanState) FromResourceData(from resources.ResourceData) {

	t.ID = MustString(from, "id")
	t.Name = MustString(from, "name")
	t.Description = MustString(from, "description")
	t.Version = int(MustFloat64(from, "version"))
	t.CreationType = MustString(from, "creationType")
	t.WorkspaceID = MustString(from, "workspaceId")
	t.CreatedAt = MustString(from, "created_at")
	t.UpdatedAt = MustString(from, "updated_at")
	t.TrackingPlanArgs.FromResourceData(
		MustMapStringInterface(from, "trackingplan_args"),
	)

	events := MapStringInterfaceSlice(from, "events", nil)
	if len(events) == 0 {
		return
	}

	tpEvents := make([]*TrackingPlanEventState, len(events))
	for idx, event := range events {

		tpEvents[idx] = &TrackingPlanEventState{
			ID:             MustString(event, "id"),
			EventID:        MustString(event, "eventId"),
			Name:           MustString(event, "name"),
			Description:    MustString(event, "description"),
			EventType:      MustString(event, "eventType"),
			AllowUnplanned: MustBool(event, "allowUnplanned"),
			Properties:     make([]*TrackingPlanPropertyState, 0),
		}

		properties := MapStringInterfaceSlice(event, "properties", nil)
		if len(properties) == 0 {
			continue
		}

		tpProperties := make([]*TrackingPlanPropertyState, 0, len(properties))
		for idx, property := range properties {
			tpProperties[idx] = &TrackingPlanPropertyState{
				Name:        MustString(property, "name"),
				Description: MustString(property, "description"),
				Type:        MustString(property, "type"),
				Config:      MapStringInterface(property, "config", make(map[string]interface{})),
				Required:    MustBool(property, "required"),
			}
		}
		tpEvents[idx].Properties = tpProperties
	}

	t.Events = tpEvents
}

// Encapsulates the catalog argument which is added as a resource
// when registering the tracking plan
type TrackingPlanArgs struct {
	Name        string                   `json:"display_name"`
	LocalID     string                   `json:"id"`
	Description string                   `json:"description"`
	Events      []*TrackingPlanEventArgs `json:"events"`
}

type TrackingPlanEventArgs struct {
	Name           string                      `json:"name"`
	LocalID        string                      `json:"id"`
	Description    string                      `json:"description"`
	Type           string                      `json:"type"`
	AllowUnplanned bool                        `json:"allow_unplanned"`
	Properties     []*TrackingPlanPropertyArgs `json:"properties"`
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
	Name        string                 `json:"name"`
	LocalID     string                 `json:"id"`
	Description string                 `json:"description"`
	Type        string                 `json:"type"`
	Config      map[string]interface{} `json:"config"`
	Required    bool                   `json:"required"`
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

	args.Name = MustString(from, "display_name")
	args.Description = MustString(from, "description")
	args.LocalID = MustString(from, "local_id")

	events := MapStringInterfaceSlice(from, "events", nil)
	if len(events) == 0 {
		return
	}

	eventProps := make([]*TrackingPlanEventArgs, len(events))
	for idx, event := range events {

		eventProps[idx] = &TrackingPlanEventArgs{
			Name:           MustString(event, "name"),
			Description:    MustString(event, "description"),
			LocalID:        MustString(event, "local_id"),
			Type:           MustString(event, "type"),
			AllowUnplanned: MustBool(event, "allow_unplanned"),
			Properties:     make([]*TrackingPlanPropertyArgs, 0),
		}

		properties := MapStringInterfaceSlice(event, "properties", nil)
		if len(properties) == 0 {
			continue
		}

		tpProperties := make([]*TrackingPlanPropertyArgs, len(properties))
		for idx, property := range properties {
			tpProperties[idx] = &TrackingPlanPropertyArgs{
				LocalID:     MustString(property, "local_id"),
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
				"local_id":    property.LocalID,
				"type":        property.Type,
				"config":      property.Config,
				"required":    property.Required,
			})
		}

		events = append(events, map[string]interface{}{
			"local_id":        event.LocalID,
			"name":            event.Name,
			"description":     event.Description,
			"type":            event.Type,
			"allow_unplanned": event.AllowUnplanned,
			"properties":      properties,
		})
	}

	return resources.ResourceData{
		"display_name": args.Name,
		"description":  args.Description,
		"local_id":     args.LocalID,
		"events":       events,
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

// ConstructTPEventState constructs the tracking plan event state from the catalog event and with the event created
// upstream in the upsert event request
func ConstructTPEventState(localEvent *TrackingPlanEventArgs, catalogEvent *client.TrackingPlanEvent) *TrackingPlanEventState {

	properties := make([]*TrackingPlanPropertyState, 0, len(localEvent.Properties))
	for _, prop := range localEvent.Properties {
		properties = append(properties, &TrackingPlanPropertyState{
			Name:        prop.Name,
			Description: prop.Description,
			LocalID:     prop.LocalID,
			Type:        prop.Type,
			Config:      prop.Config,
			Required:    prop.Required,
		})
	}

	return &TrackingPlanEventState{
		ID:             catalogEvent.ID,
		EventID:        catalogEvent.EventID,
		LocalID:        localEvent.LocalID,
		Name:           localEvent.Name,
		Description:    localEvent.Description,
		EventType:      localEvent.Type,
		AllowUnplanned: localEvent.AllowUnplanned,
		Properties:     properties,
	}
}
