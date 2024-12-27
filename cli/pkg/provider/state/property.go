package state

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
)

var (
	_ StateMapper = &PropertyArgs{}
	_ StateMapper = &PropertyState{}
)

type PropertyArgs struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Type        string                 `json:"type"`
	Config      map[string]interface{} `json:"propConfig"`
}

func (args *PropertyArgs) ToResourceData() *resources.ResourceData {
	return &resources.ResourceData{
		"display_name": args.Name,
		"description":  args.Description,
		"type":         args.Type,
		"config":       args.Config,
	}
}

func (args *PropertyArgs) FromResourceData(from *resources.ResourceData) {
	args.Name = MustString(*from, "display_name")
	args.Description = MustString(*from, "description")
	args.Type = MustString(*from, "type")
	args.Config = MapStringInterface(*from, "config", nil)
}

type PropertyState struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Type        string                 `json:"type"`
	WorkspaceID string                 `json:"workspaceId"`
	Config      map[string]interface{} `json:"config"`
	CreatedAt   string                 `json:"created_at"`
	UpdatedAt   string                 `json:"updated_at"`
}

func (p *PropertyState) ToResourceData() *resources.ResourceData {
	return &resources.ResourceData{
		"id":           p.ID,
		"display_name": p.Name,
		"description":  p.Description,
		"type":         p.Type,
		"config":       p.Config,
		"workspaceId":  p.WorkspaceID,
		"created_at":   p.CreatedAt,
		"updated_at":   p.UpdatedAt,
	}
}

func (p *PropertyState) FromResourceData(from *resources.ResourceData) {
	p.ID = MustString(*from, "id")
	p.Name = MustString(*from, "display_name")
	p.Description = MustString(*from, "description")
	p.Type = MustString(*from, "type")
	p.Config = MapStringInterface(*from, "config", nil)
	p.WorkspaceID = MustString(*from, "workspaceId")
	p.CreatedAt = MustString(*from, "created_at")
	p.UpdatedAt = MustString(*from, "updated_at")
}
