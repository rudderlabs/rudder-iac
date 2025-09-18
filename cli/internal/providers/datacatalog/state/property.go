package state

import (
	"fmt"
	"strings"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
)

type PropertyArgs struct {
	Name        string
	Description string
	Type        any
	Config      map[string]interface{}
}

func (args *PropertyArgs) FromCatalogPropertyType(prop localcatalog.Property, urnFromRef func(string) string) error {
	args.Name = prop.Name
	args.Description = prop.Description
	args.Type = prop.Type
	args.Config = prop.Config

	if strings.HasPrefix(prop.Type, "#/custom-types/") {
		customTypeURN := urnFromRef(prop.Type)

		if customTypeURN == "" {
			return fmt.Errorf("unable to resolve custom type reference urn: %s", prop.Type)
		}
		args.Type = resources.PropertyRef{
			URN:      customTypeURN,
			Property: "name",
		}
	}

	if prop.Type == "array" && prop.Config != nil {
		itemTypes, ok := prop.Config["itemTypes"]
		if !ok {
			return nil
		}

		for _, item := range itemTypes.([]any) {
			val := item.(string)

			if !strings.HasPrefix(val, "#/custom-types/") {
				continue
			}

			customTypeURN := urnFromRef(val)
			if customTypeURN == "" {
				return fmt.Errorf("unable to resolve ref to the custom type urn: %s", val)
			}

			args.Config["itemTypes"] = []interface{}{
				resources.PropertyRef{
					URN:      customTypeURN,
					Property: "name",
				},
			}
		}
	}

	return nil
}

// FromRemoteProperty converts from remote API Property to PropertyArgs
func (args *PropertyArgs) FromRemoteProperty(property *catalog.Property, getURNFromRemoteId func(resourceType string, remoteId string) (string, error)) error {
	return fmt.Errorf("not implemented")
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

// FromRemoteProperty converts from catalog.Property to PropertyState
func (p *PropertyState) FromRemoteProperty(property *catalog.Property, getURNFromRemoteId func(resourceType string, remoteId string) (string, error)) error {
	return fmt.Errorf("not implemented")
}
