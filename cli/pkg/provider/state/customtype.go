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

// CatalogCustomType ->  state.CustomTypeArgs ->  ResourcesData ->  Syncer ->  Provider ( Create (ResourcesData) (CustomTypeArgs FromResourcesData )) [ ID ]

// FromCatalogCustomType populates CustomTypeArgs from a localcatalog.CustomType
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

		for idx, item := range itemTypes.([]string) {
			if !localcatalog.CustomTypeRegex.Match([]byte(item)) {
				continue
			}

			typesWithPropRef := make([]any, len(itemTypes.([]string)))
			typesWithPropRef[idx] = resources.PropertyRef{
				URN:      urnFromRef(item),
				Property: "name",
			}

			args.Config["itemTypes"] = typesWithPropRef
		}

	}

	args.Properties = properties
}
