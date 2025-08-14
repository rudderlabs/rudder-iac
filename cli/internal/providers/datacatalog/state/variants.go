package state

import (
	"reflect"

	"github.com/samber/lo"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
)

// Variants represents a slice of conditional variants for state management.
// It provides the foundational data structure for conditional validation with PropertyRef support.
type Variants []Variant

func (v Variants) ToCatalogVariants() catalog.Variants {
	variants := make(catalog.Variants, 0)

	for _, variant := range v {
		variants = append(variants, catalog.Variant{
			Type:          variant.Type,
			Discriminator: variant.Discriminator.(string),
			Cases: lo.Map(variant.Cases, func(vc VariantCase, _ int) catalog.VariantCase {
				return catalog.VariantCase{
					DisplayName: vc.DisplayName,
					Match:       vc.Match,
					Description: vc.Description,
					Properties: lo.Map(vc.Properties, func(pr PropertyReference, _ int) catalog.PropertyReference {
						return catalog.PropertyReference{
							ID:       pr.ID.(string),
							Required: pr.Required,
						}
					}),
				}
			}),
			Default: lo.Map(variant.Default, func(pr PropertyReference, _ int) catalog.PropertyReference {
				return catalog.PropertyReference{
					ID:       pr.ID.(string),
					Required: pr.Required,
				}
			}),
		})
	}

	return variants
}

func (v *Variants) ToResourceData() []map[string]any {
	toReturn := make([]map[string]any, 0, len(*v))
	for _, variant := range *v {

		cases := make([]map[string]any, 0, len(variant.Cases))
		for _, vc := range variant.Cases {
			cases = append(cases, map[string]any{
				"display_name": vc.DisplayName,
				"match":        vc.Match,
				"description":  vc.Description,
				"properties": lo.Map(vc.Properties, func(pr PropertyReference, _ int) map[string]any {
					return map[string]any{
						"id":       pr.ID,
						"required": pr.Required,
					}
				}),
			})
		}

		toReturn = append(toReturn, map[string]any{
			"type":          variant.Type,
			"discriminator": variant.Discriminator,
			"cases":         cases,
			"default": lo.Map(variant.Default, func(pr PropertyReference, _ int) map[string]any {
				return map[string]any{
					"id":       pr.ID,
					"required": pr.Required,
				}
			}),
		})
	}

	return toReturn
}

func NormalizeToSliceMap(from map[string]any, key string) []map[string]any {
	var toReturn []map[string]any

	toReturn = MapStringInterfaceSlice(from, key, nil)
	if len(toReturn) == 0 {
		fallBack := InterfaceSlice(from, key, nil)
		for _, entity := range fallBack {
			toReturn = append(toReturn, entity.(map[string]any))
		}
	}

	return toReturn
}

func (v *Variants) FromResourceData(from []map[string]any) {
	for _, entry := range from {
		variantMap := entry
		variant := Variant{
			Type:          variantMap["type"].(string),
			Discriminator: variantMap["discriminator"].(string),
		}

		cases := NormalizeToSliceMap(variantMap, "cases")
		for _, entry := range cases {
			variantCase := entry
			variant.Cases = append(variant.Cases, VariantCase{
				DisplayName: variantCase["display_name"].(string),
				Match:       variantCase["match"].([]any),
				Description: variantCase["description"].(string),
				Properties: lo.Map(NormalizeToSliceMap(variantCase, "properties"), func(pr map[string]any, _ int) PropertyReference {
					return PropertyReference{
						ID:       pr["id"].(string),
						Required: pr["required"].(bool),
					}
				}),
			})
		}

		variant.Default = lo.Map(NormalizeToSliceMap(variantMap, "default"), func(pr map[string]any, _ int) PropertyReference {
			return PropertyReference{
				ID:       pr["id"].(string),
				Required: pr["required"].(bool),
			}
		})

		*v = append(*v, variant)
	}
}

type Variant struct {
	Type          string              `json:"type"`
	Discriminator any                 `json:"discriminator"`
	Cases         []VariantCase       `json:"cases"`
	Default       []PropertyReference `json:"default"`
}

func (v *Variant) FromLocalCatalogVariant(
	localVariant localcatalog.Variant,
	urnFromRef func(string) string,
) error {

	v.Type = localVariant.Type
	v.Discriminator = resources.PropertyRef{
		URN:      urnFromRef(localVariant.Discriminator),
		Property: "id",
	}

	for _, localCase := range localVariant.Cases {
		v.Cases = append(v.Cases, VariantCase{
			DisplayName: localCase.DisplayName,
			Match:       localCase.Match,
			Description: localCase.Description,
			Properties: lo.Map(localCase.Properties, func(localProp localcatalog.PropertyReference, _ int) PropertyReference {
				return PropertyReference{
					ID: resources.PropertyRef{
						URN:      urnFromRef(localProp.Ref),
						Property: "id",
					},
					Required: localProp.Required,
				}
			}),
		})
	}

	v.Default = lo.Map(localVariant.Default, func(localProp localcatalog.PropertyReference, _ int) PropertyReference {
		return PropertyReference{
			ID: resources.PropertyRef{
				URN:      urnFromRef(localProp.Ref),
				Property: "id",
			},
			Required: localProp.Required,
		}
	})

	return nil
}

type VariantCase struct {
	DisplayName string              `json:"display_name"`
	Match       []any               `json:"match"`
	Description string              `json:"description"`
	Properties  []PropertyReference `json:"properties"`
}

type PropertyReference struct {
	ID       any  `json:"id"`
	Required bool `json:"required"`
}

func (v Variants) Diff(against Variants) bool {
	if v == nil && against == nil {
		return false
	}

	if len(v) != len(against) {
		return true
	}

	if v == nil || against == nil {
		return true
	}

	for i := range v {
		if v[i].diffVariant(against[i]) {
			return true
		}
	}

	return false
}

func (v Variant) diffVariant(against Variant) bool {
	if v.Type != against.Type {
		return true
	}

	if !reflect.DeepEqual(v.Discriminator, against.Discriminator) {
		return true
	}

	if len(v.Cases) != len(against.Cases) {
		return true
	}
	for i := range v.Cases {
		if v.Cases[i].diffVariantCase(against.Cases[i]) {
			return true
		}
	}

	if diffPropertyReferences(v.Default, against.Default) {
		return true
	}

	return false
}

func (vc VariantCase) diffVariantCase(against VariantCase) bool {
	if vc.DisplayName != against.DisplayName {
		return true
	}

	// checking equality without order here of the elements in the slice
	// is gonna create unnecessary complexity,
	// so we generate diff, it the customer changes order of the elements in match
	if !reflect.DeepEqual(vc.Match, against.Match) {
		return true
	}

	if vc.Description != against.Description {
		return true
	}

	if diffPropertyReferences(vc.Properties, against.Properties) {
		return true
	}

	return false
}

// diffPropertyReferencesUnordered compares two slices of PropertyReference irrespective of order.
// Properties are matched by equality of ID (using reflect.DeepEqual) and then Required flag is compared.
// Returns true if they differ.
func diffPropertyReferences(left []PropertyReference, right []PropertyReference) bool {
	if len(left) != len(right) {
		return true
	}

	for _, l := range left {
		matched, found := lo.Find(right, func(r PropertyReference) bool {
			return reflect.DeepEqual(l.ID, r.ID)
		})
		if !found {
			return true
		}
		if l.Required != matched.Required {
			return true
		}
	}

	return false
}
