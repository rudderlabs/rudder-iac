package state

import (
	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/types"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

type EventArgs struct {
	Name        string
	Description string
	EventType   string
	CategoryId  any
}

func (args *EventArgs) ToResourceData() resources.ResourceData {
	return resources.ResourceData{
		"name":        args.Name,
		"description": args.Description,
		"eventType":   args.EventType,
		"categoryId":  args.CategoryId,
	}
}

func (args *EventArgs) FromResourceData(from resources.ResourceData) {
	args.Name = MustString(from, "name")
	args.Description = MustString(from, "description")
	args.EventType = MustString(from, "eventType")
	if from["categoryId"] != nil {
		args.CategoryId = String(from, "categoryId", "")
	}
}

func (args *EventArgs) FromCatalogEvent(event *localcatalog.EventV1, getURNFromRef func(ref string) string) {
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

func (args *EventArgs) DiffUpstream(upstream *catalog.Event) bool {
	if args.Name != upstream.Name {
		return true
	}

	if args.Description != upstream.Description {
		return true
	}

	if args.EventType != upstream.EventType {
		return true
	}

	if args.CategoryId != nil && upstream.CategoryId == nil {
		return true
	}

	if args.CategoryId == nil && upstream.CategoryId != nil {
		return true
	}

	if args.CategoryId != nil && upstream.CategoryId != nil {
		if strId, ok := args.CategoryId.(string); ok {
			if *upstream.CategoryId != strId {
				return true
			}
		}
	}
	return false
}

// FromRemoteEvent converts from remote API Event to EventArgs
func (args *EventArgs) FromRemoteEvent(event *catalog.Event, getURNFromRemoteId func(resourceType string, remoteId string) (string, error)) error {
	args.Name = event.Name
	args.Description = event.Description
	args.EventType = event.EventType
	if event.CategoryId != nil {
		// get URN for the category using remoteId
		urn, err := getURNFromRemoteId(types.CategoryResourceType, *event.CategoryId)
		switch {
		case err == nil:
			args.CategoryId = &resources.PropertyRef{
				URN:      urn,
				Property: "id",
			}
		case err == resources.ErrRemoteResourceExternalIdNotFound:
			args.CategoryId = nil
		default:
			return err
		}
	}
	return nil
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

// FromRemoteEvent converts from catalog.Event to EventState
func (e *EventState) FromRemoteEvent(event *catalog.Event, getURNFromRemoteId func(resourceType string, remoteId string) (string, error)) error {
	e.EventArgs = EventArgs{
		Name:        event.Name,
		Description: event.Description,
		EventType:   event.EventType,
	}
	if event.CategoryId != nil {
		e.EventArgs.CategoryId = *event.CategoryId
	}
	e.ID = event.ID
	e.Name = event.Name
	e.Description = event.Description
	e.EventType = event.EventType
	e.WorkspaceID = event.WorkspaceId
	e.CategoryID = event.CategoryId
	e.CreatedAt = event.CreatedAt.String()
	e.UpdatedAt = event.UpdatedAt.String()
	return nil
}
