package state

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/rudderlabs/rudder-iac/cli/pkg/localcatalog"
)

// CustomTypeArgs holds the necessary information to create a custom type
type CustomTypeArgs struct {
	LocalID     string
	Name        string
	Description string
	Type        string
	Config      map[string]any
	Properties  []*CustomTypeProperty // For object-type custom types
}

// CustomTypeProperty represents a property reference in a custom type
type CustomTypeProperty struct {
	RefToID  any
	ID       string
	Required bool
}

// ToResourceData converts CustomTypeArgs to ResourceData for use in the resource graph
func (args *CustomTypeArgs) ToResourceData() resources.ResourceData {
	properties := make([]map[string]any, 0, len(args.Properties))
	for _, prop := range args.Properties {
		properties = append(properties, map[string]any{
			"refToId":  prop.RefToID,
			"id":       prop.ID,
			"required": prop.Required,
		})
	}

	return resources.ResourceData{
		"localId":     args.LocalID,
		"name":        args.Name,
		"description": args.Description,
		"type":        args.Type,
		"config":      args.Config,
		"properties":  properties,
	}
}

// FromResourceData populates CustomTypeArgs from ResourceData
func (args *CustomTypeArgs) FromResourceData(from resources.ResourceData) {
	args.LocalID = MustString(from, "localId")
	args.Name = MustString(from, "name")
	args.Description = MustString(from, "description")
	args.Type = MustString(from, "type")
	args.Config = MapStringInterface(from, "config", make(map[string]any))

	// Handle properties array using similar pattern to TrackingPlan
	var properties []any

	// Try both patterns to handle different data structures
	properties = InterfaceSlice(from, "properties", nil)
	if len(properties) == 0 {
		propertiesMap := MapStringInterfaceSlice(from, "properties", nil)
		for _, prop := range propertiesMap {
			properties = append(properties, prop)
		}
	}

	// Create properties array
	customTypeProperties := make([]*CustomTypeProperty, len(properties))
	for idx, prop := range properties {
		propMap := prop.(map[string]any)

		inst := &CustomTypeProperty{
			Required: MustBool(propMap, "required"),
			ID:       MustString(propMap, "id"),
			RefToID:  MustString(propMap, "refToId"),
		}
		inst.ID = inst.RefToID.(string)
		customTypeProperties[idx] = inst
	}

	args.Properties = customTypeProperties
}

func (args *CustomTypeArgs) FromCatalogCustomType(from *localcatalog.CustomType, urnFromRef func(urn string) string) {
	args.LocalID = from.LocalID
	args.Name = from.Name
	args.Description = from.Description
	args.Type = from.Type
	args.Config = from.Config

	properties := make([]*CustomTypeProperty, 0, len(from.Properties))
	for _, prop := range from.Properties {
		properties = append(properties, &CustomTypeProperty{
			RefToID: resources.PropertyRef{
				URN:      urnFromRef(prop.Ref),
				Property: "id",
			},
			Required: prop.Required,
		})
	}

	// BUGGY CODE TO BE FIXED IN A BETTER
	itemTypes, ok := args.Config["itemTypes"]
	if ok {

		for idx, item := range itemTypes.([]any) {

			if !localcatalog.CustomTypeRegex.Match([]byte(item.(string))) {
				continue
			}

			typesWithPropRef := make([]any, len(itemTypes.([]any)))
			typesWithPropRef[idx] = resources.PropertyRef{
				URN:      urnFromRef(item.(string)),
				Property: "name",
			}

			args.Config["itemTypes"] = typesWithPropRef
		}

	}

	args.Properties = properties
}

type CustomTypeState struct {
	CustomTypeArgs
	ID              string
	LocalID         string
	Name            string
	Description     string
	Type            string
	Config          map[string]any
	Version         int
	ItemDefinitions []string
	Rules           map[string]any
	WorkspaceID     string
	CreatedAt       string
	UpdatedAt       string
	Properties      []*CustomTypePropertyState
}

type CustomTypePropertyState struct {
	ID       string
	Required bool
}

func (s *CustomTypeState) ToResourceData() resources.ResourceData {
	properties := make([]map[string]interface{}, 0, len(s.Properties))
	for _, property := range s.Properties {
		properties = append(properties, map[string]any{
			"id":       property.ID,
			"required": property.Required,
		})
	}

	return resources.ResourceData{
		"id":              s.ID,
		"localId":         s.LocalID,
		"name":            s.Name,
		"description":     s.Description,
		"type":            s.Type,
		"config":          s.Config,
		"version":         s.Version,
		"itemDefinitions": s.ItemDefinitions,
		"rules":           s.Rules,
		"workspaceId":     s.WorkspaceID,
		"createdAt":       s.CreatedAt,
		"updatedAt":       s.UpdatedAt,
		"properties":      properties,
		"customTypeArgs":  map[string]interface{}(s.CustomTypeArgs.ToResourceData()),
	}
}

func (s *CustomTypeState) FromResourceData(from resources.ResourceData) {
	s.ID = MustString(from, "id")
	s.LocalID = MustString(from, "localId")
	s.Name = MustString(from, "name")
	s.Description = MustString(from, "description")
	s.Type = MustString(from, "type")
	s.Config = MapStringInterface(from, "config", make(map[string]any))
	s.Version = int(MustFloat64(from, "version"))
	s.ItemDefinitions = MustStringSlice(from, "itemDefinitions")
	s.Rules = MapStringInterface(from, "rules", make(map[string]any))
	s.WorkspaceID = MustString(from, "workspaceId")
	s.CreatedAt = MustString(from, "createdAt")
	s.UpdatedAt = MustString(from, "updatedAt")

	properties := InterfaceSlice(from, "properties", nil)
	if len(properties) == 0 {
		propertiesMap := MapStringInterfaceSlice(from, "properties", nil)
		for _, prop := range propertiesMap {
			properties = append(properties, prop)
		}
	}

	s.Properties = make([]*CustomTypePropertyState, len(properties))
	for idx, property := range properties {
		property := property.(map[string]any)
		s.Properties[idx] = &CustomTypePropertyState{
			ID:       MustString(property, "id"),
			Required: MustBool(property, "required"),
		}
	}

	s.CustomTypeArgs.FromResourceData(
		MustMapStringInterface(from, "customTypeArgs"),
	)
}
