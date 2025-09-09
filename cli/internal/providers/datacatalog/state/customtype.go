package state

import (
	"fmt"
	"reflect"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
)

// CustomTypeArgs holds the necessary information to create a custom type
type CustomTypeArgs struct {
	LocalID     string
	Name        string
	Description string
	Type        string
	Config      map[string]any
	Properties  []*CustomTypeProperty // For object-type custom types
	Variants    Variants
}

// CustomTypeProperty represents a property reference in a custom type
type CustomTypeProperty struct {
	RefToID  any
	ID       string
	Required bool
}

// Diff compares two CustomTypeProperty instances and returns true if they differ
func (prop *CustomTypeProperty) Diff(other *CustomTypeProperty) bool {
	if prop.ID != other.ID {
		return true
	}

	if prop.Required != other.Required {
		return true
	}

	if !reflect.DeepEqual(prop.RefToID, other.RefToID) {
		return true
	}

	return false
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
		"variants":    args.Variants.ToResourceData(),
	}
}

// FromResourceData populates CustomTypeArgs from ResourceData
func (args *CustomTypeArgs) FromResourceData(from resources.ResourceData) {
	args.LocalID = MustString(from, "localId")
	args.Name = MustString(from, "name")
	args.Description = MustString(from, "description")
	args.Type = MustString(from, "type")
	args.Config = MapStringInterface(from, "config", make(map[string]any))

	variants := NormalizeToSliceMap(from, "variants")
	var variantsToAdd Variants
	variantsToAdd.FromResourceData(variants)
	args.Variants = variantsToAdd

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

func (args *CustomTypeArgs) FromCatalogCustomType(from *localcatalog.CustomType, urnFromRef func(urn string) string) error {
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

	// VARIANT HANDLING
	variants := make([]Variant, 0, len(from.Variants))
	for _, variant := range from.Variants {
		toAdd := Variant{}

		if err := toAdd.FromLocalCatalogVariant(variant, urnFromRef); err != nil {
			return fmt.Errorf("processing %s variant %s: %w", variant.Type, variant.Discriminator, err)
		}
		variants = append(variants, toAdd)
	}
	args.Variants = variants

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
	return nil
}

// FromRemoteCustomType converts from remote API CustomType to CustomTypeArgs
func (args *CustomTypeArgs) FromRemoteCustomType(customType *catalog.CustomType, resourceCollection *resources.ResourceCollection) {
}

// PropertyByID finds a property by its ID within the custom type
func (args *CustomTypeArgs) PropertyByID(id string) *CustomTypeProperty {
	for _, prop := range args.Properties {
		if prop.ID == id {
			return prop
		}
	}
	return nil
}

// Diff compares two CustomTypeArgs instances and returns true if they differ
func (args *CustomTypeArgs) Diff(other *CustomTypeArgs) bool {
	// Compare basic fields
	if args.LocalID != other.LocalID {
		return true
	}

	if args.Name != other.Name {
		return true
	}

	if args.Description != other.Description {
		return true
	}

	if args.Type != other.Type {
		return true
	}

	// Compare config maps using deep equality
	if !reflect.DeepEqual(args.Config, other.Config) {
		return true
	}

	// Compare properties arrays
	if len(args.Properties) != len(other.Properties) {
		return true
	}

	for _, prop := range args.Properties {
		otherProp := other.PropertyByID(prop.ID)
		if otherProp == nil {
			return true
		}

		if prop.Diff(otherProp) {
			return true
		}
	}

	// Compare variants using existing Variants.Diff method
	if args.Variants.Diff(other.Variants) {
		return true
	}

	return false
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
}

type CustomTypePropertyState struct {
	ID       string
	Required bool
}

func (s *CustomTypeState) ToResourceData() resources.ResourceData {

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
		"customTypeArgs":  map[string]any(s.CustomTypeArgs.ToResourceData()),
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

	s.CustomTypeArgs.FromResourceData(
		MustMapStringInterface(from, "customTypeArgs"),
	)
}

// FromRemoteCustomType converts from catalog.CustomType to CustomTypeState
func (s *CustomTypeState) FromRemoteCustomType(customType *catalog.CustomType, resourceCollection *resources.ResourceCollection) {
}
