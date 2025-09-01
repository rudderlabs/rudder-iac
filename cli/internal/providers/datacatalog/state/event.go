package state

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
)

type EventArgs struct {
	ProjectId   string
	Name        string
	Description string
	EventType   string
	CategoryId  *resources.PropertyRef
}

func (args *EventArgs) ToResourceData() resources.ResourceData {
	return resources.ResourceData{
		"projectId":   args.ProjectId,
		"name":        args.Name,
		"description": args.Description,
		"eventType":   args.EventType,
		"categoryId":  args.CategoryId,
	}
}

func (args *EventArgs) FromResourceData(from resources.ResourceData) {
	args.ProjectId = MustString(from, "projectId")
	args.Name = MustString(from, "name")
	args.Description = MustString(from, "description")
	args.EventType = MustString(from, "eventType")
	if categoryId, ok := from["categoryId"].(*resources.PropertyRef); ok {
		args.CategoryId = categoryId
	}
}

func (args *EventArgs) FromCatalogEvent(event *localcatalog.Event, getURNFromRef func(ref string) string) {
	args.ProjectId = event.LocalID
	args.Name = event.Name
	args.Description = event.Description
	args.EventType = event.Type
	if event.CategoryRef != nil {
		args.CategoryId = &resources.PropertyRef{
			URN:      getURNFromRef(*event.CategoryRef),
			Property: "id",
		}
	}
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
	e.CategoryID = StringPtr(from, "categoryId", nil)
	e.CreatedAt = MustString(from, "createdAt")
	e.UpdatedAt = MustString(from, "updatedAt")
	e.EventArgs.FromResourceData(resources.ResourceData(
		MustMapStringInterface(from, "eventArgs"),
	))
}
