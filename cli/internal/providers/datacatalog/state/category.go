package state

import (
	"time"

	catalog "github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
)

// CategoryArgs holds the necessary information to create a category
type CategoryArgs struct {
	Name string
}

func (args *CategoryArgs) ToResourceData() resources.ResourceData {
	return resources.ResourceData{
		"name": args.Name,
	}
}

func (args *CategoryArgs) FromResourceData(from resources.ResourceData) {
	args.Name = MustString(from, "name")
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

// CategoryStateFromAPI converts API Category to CategoryState
func CategoryStateFromAPI(category *catalog.Category) *CategoryState {
	return &CategoryState{
		CategoryArgs: CategoryArgs{
			Name: category.Name,
		},
		ID:          category.ID,
		Name:        category.Name,
		WorkspaceID: category.WorkspaceID,
		CreatedAt:   category.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   category.UpdatedAt.Format(time.RFC3339),
	}
}

// CategoryStateToAPI converts CategoryState to API Category
func CategoryStateToAPI(state *CategoryState) *catalog.Category {
	createdAt, _ := time.Parse(time.RFC3339, state.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339, state.UpdatedAt)

	return &catalog.Category{
		ID:          state.ID,
		Name:        state.Name,
		WorkspaceID: state.WorkspaceID,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}
}

// CategoryStateCollection represents a collection of category states
type CategoryStateCollection []*CategoryState

// GetByID returns a category by its ID
func (collection CategoryStateCollection) GetByID(id string) *CategoryState {
	for _, category := range collection {
		if category.ID == id {
			return category
		}
	}
	return nil
}

// GetByName returns a category by its name
func (collection CategoryStateCollection) GetByName(name string) *CategoryState {
	for _, category := range collection {
		if category.Name == name {
			return category
		}
	}
	return nil
}

// Add adds a category to the collection
func (collection *CategoryStateCollection) Add(category *CategoryState) {
	*collection = append(*collection, category)
}

// Update updates a category in the collection, returns true if found and updated
func (collection CategoryStateCollection) Update(category *CategoryState) bool {
	for i, existingCategory := range collection {
		if existingCategory.ID == category.ID {
			collection[i] = category
			return true
		}
	}
	return false
}

// Delete removes a category from the collection by ID, returns true if found and deleted
func (collection *CategoryStateCollection) Delete(id string) bool {
	for i, category := range *collection {
		if category.ID == id {
			*collection = append((*collection)[:i], (*collection)[i+1:]...)
			return true
		}
	}
	return false
}

// All returns all categories in the collection
func (collection CategoryStateCollection) All() []*CategoryState {
	return []*CategoryState(collection)
}
