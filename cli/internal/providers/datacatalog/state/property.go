package state

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/utils"
)

type PropertyArgs struct {
	Name        string
	Description string
	Type        any
	Config      map[string]interface{}
}

// DiffUpstream compares PropertyArgs with an upstream Property and returns true if they are different
func (args *PropertyArgs) DiffUpstream(upstream *catalog.Property) bool {
	if args.Name != upstream.Name {
		return true
	}

	if args.Description != upstream.Description {
		return true
	}

	if args.Type.(string) != upstream.Type {
		return true
	}

	upstreamConf := upstream.Config
	if upstream.DefinitionId != "" {
		upstreamConf = make(map[string]any)
	}
	return !reflect.DeepEqual(args.Config, upstreamConf)
}

const PropertyResourceType = "property"

func (args *PropertyArgs) FromCatalogPropertyType(prop localcatalog.Property, urnFromRef func(string) string) error {
	args.Name = prop.Name
	args.Description = prop.Description
	args.Type = prop.Type
	args.Config = make(map[string]interface{})
	for k, v := range prop.Config {
		args.Config[k] = v
	}

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

		// sort itemTypes array lexicographically
		itemTypesArray := itemTypes.([]any)
		utils.SortLexicographically(itemTypesArray)

		for _, item := range itemTypesArray {
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

	// sort the order of types for a multi type property
	if multiTypeProp := strings.Split(prop.Type, ","); len(multiTypeProp) > 1 {
		sort.Strings(multiTypeProp)
		args.Type = strings.Join(multiTypeProp, ",")
	}

	return nil
}

// FromRemoteProperty converts from remote API Property to PropertyArgs
func (args *PropertyArgs) FromRemoteProperty(property *catalog.Property, getURNFromRemoteId func(string, string) (string, error)) error {
	args.Name = property.Name
	args.Description = property.Description
	args.Type = property.Type
	// Deep copy the config map
	args.Config = make(map[string]interface{})
	for k, v := range property.Config {
		// sort itemTypes array lexicographically
		if k == "itemTypes" {
			utils.SortLexicographically(v.([]any))
		}
		args.Config[k] = v
	}

	// Check if the property is referring to a customType using property.DefinitionId
	if property.DefinitionId != "" {
		urn, err := getURNFromRemoteId(CustomTypeResourceType, property.DefinitionId)
		switch {
		case err == nil:
			args.Type = resources.PropertyRef{
				URN:      urn,
				Property: "name",
			}
		case err == resources.ErrRemoteResourceExternalIdNotFound:
			args.Type = nil
		default:
			return err
		}
		args.Config = map[string]interface{}{}
	}

	// Handle array types with custom type references in itemTypes
	if property.Type == "array" && property.Config != nil && property.ItemDefinitionId != "" {
		urn, err := getURNFromRemoteId(CustomTypeResourceType, property.ItemDefinitionId)
		switch {
		case err == nil:
			// Update itemTypes in config to reference the same custom type
			args.Config["itemTypes"] = []interface{}{
				resources.PropertyRef{
					URN:      urn,
					Property: "name",
				},
			}
		case err == resources.ErrRemoteResourceExternalIdNotFound:
			args.Config["itemTypes"] = []interface{}{nil}
		default:
			return err
		}

	}

	// sort the order of types for a multi type property
	if multiTypeProp := strings.Split(property.Type, ","); len(multiTypeProp) > 1 {
		sort.Strings(multiTypeProp)
		args.Type = strings.Join(multiTypeProp, ",")
	}
	return nil
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
func (p *PropertyState) FromRemoteProperty(property *catalog.Property, getURNFromRemoteId func(string, string) (string, error)) error {
	p.PropertyArgs.Name = property.Name
	p.PropertyArgs.Description = property.Description
	p.PropertyArgs.Type = property.Type
	// Do not copy config for custom types. Copy only for non-custom types
	if property.DefinitionId == "" {
		p.PropertyArgs.Config = property.Config
	}
	p.ID = property.ID
	p.Name = property.Name
	p.Description = property.Description
	p.Type = property.Type
	p.WorkspaceID = property.WorkspaceId
	p.Config = property.Config
	p.CreatedAt = property.CreatedAt.String()
	p.UpdatedAt = property.UpdatedAt.String()
	return nil
}
