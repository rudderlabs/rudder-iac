package state

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
)

type CustomTypeArgs struct {
	Name        string
	LocalID     string
	Description string
	Type        string
	Config      map[string]interface{}
	Properties  []*CustomTypePropertyArgs
}

type CustomTypePropertyArgs struct {
	RefID      string
	PropertyID string
	Required   bool
}

type CustomTypePropertyState struct {
	LocalID    string
	PropertyID string
	Required   bool
}

type CustomTypeState struct {
	CustomTypeArgs
	ID              string
	LocalID         string
	Name            string
	Description     string
	Type            string
	Config          map[string]interface{}
	Version         int
	ItemDefinitions []string
	Rules           map[string]interface{}
	WorkspaceID     string
	CreatedAt       string
	UpdatedAt       string
	Properties      []*CustomTypePropertyState
}

func (args *CustomTypeArgs) ToResourceData() resources.ResourceData {
	properties := make([]map[string]interface{}, 0, len(args.Properties))
	for _, property := range args.Properties {
		properties = append(properties, map[string]interface{}{
			"refId":      property.RefID,
			"propertyId": property.PropertyID,
			"required":   property.Required,
		})
	}

	return resources.ResourceData{
		"name":        args.Name,
		"localId":     args.LocalID,
		"description": args.Description,
		"type":        args.Type,
		"config":      args.Config,
		"properties":  properties,
	}
}

func (args *CustomTypeArgs) FromResourceData(from resources.ResourceData) {
	args.Name = MustString(from, "name")
	args.LocalID = MustString(from, "localId")
	args.Description = MustString(from, "description")
	args.Type = MustString(from, "type")
	args.Config = MapStringInterface(from, "config", make(map[string]interface{}))

	properties := InterfaceSlice(from, "properties", nil)
	if len(properties) == 0 {
		propertiesMap := MapStringInterfaceSlice(from, "properties", nil)
		for _, prop := range propertiesMap {
			properties = append(properties, prop)
		}
	}

	args.Properties = make([]*CustomTypePropertyArgs, len(properties))
	for idx, property := range properties {
		property := property.(map[string]interface{})
		args.Properties[idx] = &CustomTypePropertyArgs{
			RefID:      MustString(property, "refId"),
			PropertyID: MustString(property, "propertyId"),
			Required:   MustBool(property, "required"),
		}
	}
}

func (s *CustomTypeState) ToResourceData() resources.ResourceData {
	properties := make([]map[string]interface{}, 0, len(s.Properties))
	for _, property := range s.Properties {
		properties = append(properties, map[string]interface{}{
			"localId":    property.LocalID,
			"propertyId": property.PropertyID,
			"required":   property.Required,
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
	s.Config = MapStringInterface(from, "config", make(map[string]interface{}))
	s.Version = MustInt(from, "version")
	s.ItemDefinitions = MustStringSlice(from, "itemDefinitions")
	s.Rules = MapStringInterface(from, "rules", make(map[string]interface{}))
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
		property := property.(map[string]interface{})
		s.Properties[idx] = &CustomTypePropertyState{
			LocalID:    MustString(property, "localId"),
			PropertyID: MustString(property, "propertyId"),
			Required:   MustBool(property, "required"),
		}
	}

	s.CustomTypeArgs.FromResourceData(
		MustMapStringInterface(from, "customTypeArgs"),
	)
}
