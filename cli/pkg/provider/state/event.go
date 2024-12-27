package state

import "github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"

var (
	_ StateMapper = &EventState{}
	_ StateMapper = &EventArgs{}
)

type EventArgs struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	EventType   string  `json:"eventType"`
	CategoryID  *string `json:"categoryId"`
}

func (args *EventArgs) ToResourceData() *resources.ResourceData {
	return &resources.ResourceData{
		"display_name": args.Name,
		"description":  args.Description,
		"event_type":   args.EventType,
		"categoryId":   args.CategoryID,
	}
}

func (args *EventArgs) FromResourceData(from *resources.ResourceData) {
	args.Name = MustString(*from, "display_name")
	args.Description = MustString(*from, "description")
	args.EventType = MustString(*from, "event_type")
	args.CategoryID = StringPtr(*from, "categoryId", nil)
}

type EventState struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	EventType   string  `json:"eventType"`
	WorkspaceID string  `json:"workspaceId"`
	CategoryID  *string `json:"categoryId"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

func (e *EventState) ToResourceData() *resources.ResourceData {
	return &resources.ResourceData{
		"id":           e.ID,
		"display_name": e.Name,
		"description":  e.Description,
		"event_type":   e.EventType,
		"workspaceId":  e.WorkspaceID,
		"categoryId":   e.CategoryID,
		"created_at":   e.CreatedAt,
		"updated_at":   e.UpdatedAt,
	}
}

func (e *EventState) FromResourceData(from *resources.ResourceData) {
	e.ID = MustString(*from, "id")
	e.Name = MustString(*from, "display_name")
	e.Description = MustString(*from, "description")
	e.EventType = MustString(*from, "event_type")
	e.WorkspaceID = MustString(*from, "workspaceId")
	e.CreatedAt = MustString(*from, "created_at")
	e.UpdatedAt = MustString(*from, "updated_at")
	e.CategoryID = StringPtr(*from, "categoryId", nil)
}
