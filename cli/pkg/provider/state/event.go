package state

import "github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"

var (
	_ StateMapper = &EventState{}
	_ StateMapper = &EventArgs{}
)

type EventArgs struct {
	Name        string
	Description string
	EventType   string
	CategoryID  *string
}

func (args *EventArgs) ToResourceData() resources.ResourceData {
	return resources.ResourceData{
		"name":        args.Name,
		"description": args.Description,
		"eventType":   args.EventType,
		"categoryId":  args.CategoryID,
	}
}

func (args *EventArgs) FromResourceData(from resources.ResourceData) {
	args.Name = MustString(from, "name")
	args.Description = MustString(from, "description")
	args.EventType = MustString(from, "eventType")
	args.CategoryID = StringPtr(from, "categoryId", nil)
}

type EventState struct {
	EventArgs
	ID          string
	Name        string
	Description string
	EventType   string
	WorkspaceID string
	CategoryID  *string
	CreatedAt   string
	UpdatedAt   string
}

func (e *EventState) ToResourceData() resources.ResourceData {
	return resources.ResourceData{
		"id":          e.ID,
		"name":        e.Name,
		"description": e.Description,
		"eventType":   e.EventType,
		"workspaceId": e.WorkspaceID,
		"categoryId":  e.CategoryID,
		"createdAt":   e.CreatedAt,
		"updatedAt":   e.UpdatedAt,
		"eventArgs":   map[string]interface{}(e.EventArgs.ToResourceData()),
	}
}

func (e *EventState) FromResourceData(from resources.ResourceData) {
	e.ID = MustString(from, "id")
	e.Name = MustString(from, "name")
	e.Description = MustString(from, "description")
	e.EventType = MustString(from, "eventType")
	e.WorkspaceID = MustString(from, "workspaceId")
	e.CreatedAt = MustString(from, "createdAt")
	e.UpdatedAt = MustString(from, "updatedAt")
	e.CategoryID = StringPtr(from, "categoryId", nil)
	e.EventArgs.FromResourceData(resources.ResourceData(
		MustMapStringInterface(from, "eventArgs"),
	))
}
