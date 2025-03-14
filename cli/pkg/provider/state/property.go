package state

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
)

type PropertyArgs struct {
	Name        string
	Description string
	Type        string
	Config      map[string]interface{}
}

func (args *PropertyArgs) ToResourceData() resources.ResourceData {
	return resources.ResourceData{
		"name":        args.Name,
		"description": args.Description,
		"type":        args.Type,
		"config":      args.Config,
	}
}

func (args *PropertyArgs) FromResourceData(from resources.ResourceData) {
	args.Name = MustString(from, "name")
	args.Description = MustString(from, "description")
	args.Type = MustString(from, "type")
	args.Config = MapStringInterface(from, "config", make(map[string]interface{}))
}

type PropertyState struct {
	PropertyArgs
	ID          string
	Name        string
	Description string
	Type        string
	WorkspaceID string
	Config      map[string]interface{}
	CreatedAt   string
	UpdatedAt   string
}

func (p *PropertyState) ToResourceData() resources.ResourceData {
	return resources.ResourceData{
		"id":           p.ID,
		"name":         p.Name,
		"description":  p.Description,
		"type":         p.Type,
		"config":       p.Config,
		"workspaceId":  p.WorkspaceID,
		"createdAt":    p.CreatedAt,
		"updatedAt":    p.UpdatedAt,
		"propertyArgs": map[string]interface{}(p.PropertyArgs.ToResourceData()),
	}
}

func (p *PropertyState) FromResourceData(from resources.ResourceData) {
	p.ID = MustString(from, "id")
	p.Name = MustString(from, "name")
	p.Description = MustString(from, "description")
	p.Type = MustString(from, "type")
	p.Config = MapStringInterface(from, "config", make(map[string]interface{}))
	p.WorkspaceID = MustString(from, "workspaceId")
	p.CreatedAt = MustString(from, "createdAt")
	p.UpdatedAt = MustString(from, "updatedAt")

	p.PropertyArgs.FromResourceData(
		MustMapStringInterface(from, "propertyArgs"),
	)
}
