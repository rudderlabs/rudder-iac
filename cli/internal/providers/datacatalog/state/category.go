package state

import (
	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
)

// CategoryArgs holds the necessary information to create a category
type CategoryArgs struct {
	Name string
}

const CategoryResourceType = "category"

func (args *CategoryArgs) ToResourceData() resources.ResourceData {
	return resources.ResourceData{
		"name": args.Name,
	}
}

func (args *CategoryArgs) FromResourceData(from resources.ResourceData) {
	args.Name = MustString(from, "name")
}

func (args *CategoryArgs) FromCatalogCategory(category *localcatalog.Category) {
	args.Name = category.Name
}

// FromRemoteCategory converts from remote API Category to CategoryArgs
func (args *CategoryArgs) FromRemoteCategory(category *catalog.Category, getURNFromRemoteId func(string, string) string) {
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

// FromRemoteCategory converts from catalog.Category to CategoryState
func (c *CategoryState) FromRemoteCategory(category *catalog.Category, getURNFromRemoteId func(string, string) string) {
	c.CategoryArgs.FromRemoteCategory(category, getURNFromRemoteId)
	c.ID = category.ID
	c.Name = category.Name
	c.WorkspaceID = category.WorkspaceID
	c.CreatedAt = category.CreatedAt.String()
	c.UpdatedAt = category.UpdatedAt.String()
}
