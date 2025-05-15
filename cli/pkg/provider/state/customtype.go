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
	Name        string
	LocalID     string
	Description string
	Type        string
	Config      map[string]interface{}
	Required    bool
}

func (args *CustomTypeArgs) ToResourceData() resources.ResourceData {
	properties := make([]map[string]interface{}, 0)
	for _, property := range args.Properties {
		properties = append(properties, map[string]interface{}{
			"name":        property.Name,
			"localId":     property.LocalID,
			"description": property.Description,
			"type":        property.Type,
			"config":      property.Config,
			"required":    property.Required,
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
			Name:        MustString(property, "name"),
			LocalID:     MustString(property, "localId"),
			Description: MustString(property, "description"),
			Type:        MustString(property, "type"),
			Config:      MapStringInterface(property, "config", make(map[string]interface{})),
			Required:    MustBool(property, "required"),
		}
	}
}

type CustomTypeState struct {
	CustomTypeArgs
	ID              string
	Name            string
	Description     string
	Type            string
	Version         int
	DataType        string
	Rules           map[string]interface{}
	ItemDefinitions []string
	WorkspaceID     string
	Config          map[string]interface{}
	CreatedAt       string
	UpdatedAt       string
	Properties      []*CustomTypePropertyState
}

type CustomTypePropertyState struct {
	ID         string
	LocalID    string
	PropertyID string
}

func (c *CustomTypeState) ToResourceData() resources.ResourceData {
	var properties []map[string]interface{}
	for _, property := range c.Properties {
		properties = append(properties, map[string]interface{}{
			"id":         property.ID,
			"localId":    property.LocalID,
			"propertyId": property.PropertyID,
		})
	}

	return resources.ResourceData{
		"id":              c.ID,
		"name":            c.Name,
		"description":     c.Description,
		"type":            c.Type,
		"version":         c.Version,
		"dataType":        c.DataType,
		"rules":           c.Rules,
		"itemDefinitions": c.ItemDefinitions,
		"config":          c.Config,
		"workspaceId":     c.WorkspaceID,
		"createdAt":       c.CreatedAt,
		"updatedAt":       c.UpdatedAt,
		"properties":      properties,
		"customTypeArgs":  map[string]interface{}(c.CustomTypeArgs.ToResourceData()),
	}
}

func (c *CustomTypeState) FromResourceData(from resources.ResourceData) {
	c.ID = MustString(from, "id")
	c.Name = MustString(from, "name")
	c.Description = MustString(from, "description")
	c.Type = MustString(from, "type")
	c.Version = MustInt(from, "version")
	c.DataType = MustString(from, "dataType")
	c.Rules = MapStringInterface(from, "rules", make(map[string]interface{}))
	c.ItemDefinitions = MustStringSlice(from, "itemDefinitions")
	c.Config = MapStringInterface(from, "config", make(map[string]interface{}))
	c.WorkspaceID = MustString(from, "workspaceId")
	c.CreatedAt = MustString(from, "createdAt")
	c.UpdatedAt = MustString(from, "updatedAt")

	properties := InterfaceSlice(from, "properties", nil)
	if len(properties) == 0 {
		propertiesMap := MapStringInterfaceSlice(from, "properties", nil)
		for _, prop := range propertiesMap {
			properties = append(properties, prop)
		}
	}

	c.Properties = make([]*CustomTypePropertyState, len(properties))
	for idx, property := range properties {
		property := property.(map[string]interface{})
		c.Properties[idx] = &CustomTypePropertyState{
			ID:         MustString(property, "id"),
			LocalID:    MustString(property, "localId"),
			PropertyID: MustString(property, "propertyId"),
		}
	}

	c.CustomTypeArgs.FromResourceData(
		MustMapStringInterface(from, "customTypeArgs"),
	)
}
