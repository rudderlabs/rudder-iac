package state

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
)

// CategoryArgs holds the necessary information to create a category
type CategoryArgs struct {
	ProjectId string
	Name      string
}

func (args *CategoryArgs) ToResourceData() resources.ResourceData {
	return resources.ResourceData{
		"projectId": args.ProjectId,
		"name":      args.Name,
	}
}

func (args *CategoryArgs) FromResourceData(from resources.ResourceData) {
	args.ProjectId = String(from, "projectId", "")
	args.Name = MustString(from, "name")
}

func (args *CategoryArgs) FromCatalogCategory(category *localcatalog.Category) {
	args.ProjectId = category.LocalID
	args.Name = category.Name
}

// CategoryState represents the full state of a category
type CategoryState struct {
	CategoryArgs
	ID          string
	Name        string
	WorkspaceID string
	CreatedAt   string
	UpdatedAt   string
}

func (c *CategoryState) ToResourceData() resources.ResourceData {
	return resources.ResourceData{
		"id":           c.ID,
		"name":         c.Name,
		"workspaceId":  c.WorkspaceID,
		"createdAt":    c.CreatedAt,
		"updatedAt":    c.UpdatedAt,
		"categoryArgs": map[string]interface{}(c.CategoryArgs.ToResourceData()),
	}
}

func (c *CategoryState) FromResourceData(from resources.ResourceData) {
	c.ID = MustString(from, "id")
	c.Name = MustString(from, "name")
	c.WorkspaceID = MustString(from, "workspaceId")
	c.CreatedAt = MustString(from, "createdAt")
	c.UpdatedAt = MustString(from, "updatedAt")
	c.CategoryArgs.FromResourceData(resources.ResourceData(
		MustMapStringInterface(from, "categoryArgs"),
	))
}
