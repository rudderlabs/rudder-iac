package state

import (
	"fmt"
	"maps"
	"reflect"
	"sort"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
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

const CustomTypeResourceType = "custom-type"

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
	args.Config = make(map[string]any)
	maps.Copy(args.Config, from.Config)

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
	// sort properties by RefToID.URN
	sort.Slice(properties, func(i, j int) bool {
		return properties[i].RefToID.(resources.PropertyRef).URN < properties[j].RefToID.(resources.PropertyRef).URN
	})

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
func (args *CustomTypeArgs) FromRemoteCustomType(customType *catalog.CustomType, getURNFromRemoteId func(resourceType string, remoteId string) (string, error)) error {
	args.LocalID = customType.ExternalID
	args.Name = customType.Name
	args.Description = customType.Description
	args.Type = customType.Type
	// Deep copy the config map
	args.Config = make(map[string]any)
	for key, value := range customType.Config {
		args.Config[key] = value
	}

	properties := make([]*CustomTypeProperty, 0, len(customType.Properties))
	for _, prop := range customType.Properties {
		urn, err := getURNFromRemoteId(PropertyResourceType, prop.ID)
		switch {
		case err == nil:
			properties = append(properties, &CustomTypeProperty{
				Required: prop.Required,
				RefToID: resources.PropertyRef{
					URN:      urn,
					Property: "id",
				},
			})
		case err == resources.ErrRemoteResourceExternalIdNotFound:
			properties = append(properties, &CustomTypeProperty{
				Required: prop.Required,
				RefToID:  nil,
			})
		default:
			return err
		}
	}
	// sort properties by RefToID.URN
	sort.Slice(properties, func(i, j int) bool {
		return properties[i].RefToID.(resources.PropertyRef).URN < properties[j].RefToID.(resources.PropertyRef).URN
	})
	args.Properties = properties

	variants := make([]Variant, 0, len(customType.Variants))
	for _, variant := range customType.Variants {
		toAdd := Variant{}

		if err := toAdd.FromRemoteVariant(variant, getURNFromRemoteId, true); err != nil {
			return fmt.Errorf("processing %s variant %s: %w", variant.Type, variant.Discriminator, err)
		}
		variants = append(variants, toAdd)
	}
	args.Variants = variants

	if len(customType.ItemDefinitions) != 0 {
		for _, item := range customType.ItemDefinitions {
			id := MustString(item.(map[string]interface{}), "id")
			urn, err := getURNFromRemoteId(CustomTypeResourceType, id)
			switch {
			case err == nil:
				args.Config["itemTypes"] = []any{
					resources.PropertyRef{
						URN:      urn,
						Property: "name",
					},
				}
			case err == resources.ErrRemoteResourceExternalIdNotFound:
				args.Config["itemTypes"] = []any{nil}
			default:
				return err
			}
			// we only support one itemdefinition, so we dont need to loop over the whole array
			break
		}
	}

	return nil
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

// DiffUpstream compares CustomTypeArgs with an upstream CustomType and returns true if they are different
func (args *CustomTypeArgs) DiffUpstream(upstream *catalog.CustomType) bool {
	if args.Name != upstream.Name {
		return true
	}

	if args.Description != upstream.Description {
		return true
	}

	if args.Type != upstream.Type {
		return true
	}

	// DeepEqual will fail if one is empty vs other nil
	// so we check only if atleast one of them is set, otherwise treating them as equal.
	if len(args.Config) != 0 || len(upstream.Config) != 0 {
		if !reflect.DeepEqual(args.Config, upstream.Config) {
			return true
		}
	}

	if len(args.Properties) != len(upstream.Properties) {
		return true
	}

	upstreamProps := make(map[string]catalog.CustomTypeProperty)
	for _, prop := range upstream.Properties {
		upstreamProps[prop.ID] = prop
	}

	for _, localProp := range args.Properties {
		upstreamProp, ok := upstreamProps[localProp.ID]
		if !ok {
			return true
		}

		if localProp.Required != upstreamProp.Required {
			return true
		}
	}

	var upstreamVariants Variants
	upstreamVariants.FromCatalogVariants(upstream.Variants)

	if args.Variants.Diff(upstreamVariants) {
		return true
	}

	return false
}

// Diff compares two CustomTypeArgs instances and returns true if they differ
func (args *CustomTypeArgs) Diff(other *CustomTypeArgs) bool {
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

	if !reflect.DeepEqual(args.Config, other.Config) {
		return true
	}

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
	ItemDefinitions []any
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
	s.Rules = MapStringInterface(from, "rules", make(map[string]any))
	s.WorkspaceID = MustString(from, "workspaceId")
	s.CreatedAt = MustString(from, "createdAt")
	s.UpdatedAt = MustString(from, "updatedAt")

	// version can be either an int or a float64
	// in our old stateful approach, we used to get the version as a float64 as we used json.Unmarshall to decode the state api's response into a map[string]interface{}
	// in the stateless approach, we derive the state from the remote CustomType which is a strongly typed struct where the version field is of type int
	// we handle both types to be compatible with both approaches
	s.Version = Int(from, "version", 0)
	if s.Version == 0 {
		s.Version = int(Float64(from, "version", 0))
	}

	if itemDef, ok := from["itemDefinitions"].([]any); ok {
		s.ItemDefinitions = itemDef
	}
	s.CustomTypeArgs.FromResourceData(
		MustMapStringInterface(from, "customTypeArgs"),
	)
}

// FromRemoteCustomType converts from catalog.CustomType to CustomTypeState
func (s *CustomTypeState) FromRemoteCustomType(customType *catalog.CustomType, getURNFromRemoteId func(resourceType string, remoteId string) (string, error)) error {
	s.ID = customType.ID
	s.LocalID = customType.ExternalID
	s.Name = customType.Name
	s.Description = customType.Description
	s.Type = customType.Type
	s.Config = customType.Config
	s.Version = customType.Version
	s.Rules = customType.Rules
	s.WorkspaceID = customType.WorkspaceId
	s.CreatedAt = customType.CreatedAt.String()
	s.UpdatedAt = customType.UpdatedAt.String()
	if len(customType.ItemDefinitions) != 0 {
		for _, item := range customType.ItemDefinitions {
			if itemMap, ok := item.(map[string]interface{}); ok {
				if itemMap["name"] != "" {
					s.ItemDefinitions = append(s.ItemDefinitions, itemMap["name"])
					break
				}

			}
		}
	}

	// create custom type args
	s.CustomTypeArgs.LocalID = customType.ExternalID
	s.CustomTypeArgs.Name = customType.Name
	s.CustomTypeArgs.Description = customType.Description
	s.CustomTypeArgs.Type = customType.Type
	s.CustomTypeArgs.Config = customType.Config

	// create properties and add them to custom type args
	s.CustomTypeArgs.Properties = make([]*CustomTypeProperty, len(customType.Properties))
	for idx, prop := range customType.Properties {
		s.CustomTypeArgs.Properties[idx] = &CustomTypeProperty{
			Required: prop.Required,
			ID:       prop.ID,
			RefToID:  prop.ID,
		}
	}

	// create variants and add them to custom type args
	variants := make([]Variant, len(customType.Variants))
	for idx, variant := range customType.Variants {
		v := &Variant{}
		v.Type = variant.Type
		v.Discriminator = variant.Discriminator
		cases := make([]VariantCase, 0, len(variant.Cases))
		for _, remoteCase := range variant.Cases {
			properties := make([]PropertyReference, len(remoteCase.Properties))
			for i, prop := range remoteCase.Properties {
				properties[i] = PropertyReference{
					ID:       prop.ID,
					Required: prop.Required,
				}
			}
			cases = append(cases, VariantCase{
				DisplayName: remoteCase.DisplayName,
				Match:       remoteCase.Match,
				Description: remoteCase.Description,
				Properties:  properties,
			})
		}
		v.Cases = cases

		defaultProps := make([]PropertyReference, len(variant.Default))
		for i, prop := range variant.Default {
			defaultProps[i] = PropertyReference{
				ID:       prop.ID,
				Required: prop.Required,
			}
		}
		v.Default = defaultProps

		variants[idx] = *v
	}
	s.CustomTypeArgs.Variants = variants
	return nil
}
