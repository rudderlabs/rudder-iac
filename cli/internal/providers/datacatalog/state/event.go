package state

import (
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
)

type EventArgs struct {
	Name        string
	Description string
	EventType   string
	Category    *EventCategoryArgs
}

// TODO: Do we need to store the category ref too?
// We are currently storing only the URN of the category. The ref gets converted to the URN in the FromCatalogEvent function.
type EventCategoryArgs struct {
	URN  string
	Name string
}

func (ecArgs *EventCategoryArgs) ToResourceData() resources.ResourceData {
	if ecArgs == nil {
		return nil
	}
	return resources.ResourceData{
		"categoryRef": &resources.PropertyRef{
			URN:      ecArgs.URN,
			Property: "name",
		},
		"name": ecArgs.Name,
	}
}

func (ecArgs *EventCategoryArgs) FromResourceData(from resources.ResourceData) {
	ecArgs.Name = MustString(from, "name")
	if ref, ok := from["categoryRef"].(*resources.PropertyRef); ok && ref != nil {
		ecArgs.URN = ref.URN
	}
}

func (args *EventArgs) ToResourceData() resources.ResourceData {
	return resources.ResourceData{
		"name":        args.Name,
		"description": args.Description,
		"eventType":   args.EventType,
		"category":    args.Category.ToResourceData(),
	}
}

func (args *EventArgs) FromResourceData(from resources.ResourceData) {
	args.Name = MustString(from, "name")
	args.Description = MustString(from, "description")
	args.EventType = MustString(from, "eventType")
	if category, ok := from["category"].(resources.ResourceData); ok && category != nil {
		args.Category = &EventCategoryArgs{}
		args.Category.FromResourceData(category)
	}
}

func (args *EventArgs) FromCatalogEvent(event *localcatalog.Event, getURNFromRef func(ref string) string) {
	args.Name = event.Name
	args.Description = event.Description
	args.EventType = event.Type
	if event.CategoryRef != nil {
		s := strings.Split(*event.CategoryRef, "/")
		args.Category = &EventCategoryArgs{
			URN:  getURNFromRef(*event.CategoryRef),
			Name: s[len(s)-1],
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
	e.EventArgs.FromResourceData(resources.ResourceData(
		MustMapStringInterface(from, "eventArgs"),
	))
}
