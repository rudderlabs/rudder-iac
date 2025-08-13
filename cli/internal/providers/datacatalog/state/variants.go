package state

import (
	"fmt"
	"reflect"

	"github.com/samber/lo"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
)

// Variants represents a slice of conditional variants for state management.
// It provides the foundational data structure for conditional validation with PropertyRef support.
type Variants []Variant

type Variant struct {
	Type          string              `json:"type"`
	Discriminator any                 `json:"discriminator"`
	Cases         []VariantCase       `json:"cases"`
	Default       []PropertyReference `json:"default"`
}

func (v *Variant) FromLocalCatalogVariant(
	localVariant localcatalog.Variant,
	urnFromRef func(string) string,
	urnFromLocalID func(string) string,
) error {
	if urnFromLocalID(localVariant.Discriminator) == "" {
		return fmt.Errorf("lookup from local id failed for discriminator %s", localVariant.Discriminator)
	}

	v.Type = localVariant.Type
	v.Discriminator = urnFromLocalID(localVariant.Discriminator)

	for _, localCase := range localVariant.Cases {
		v.Cases = append(v.Cases, VariantCase{
			DisplayName: localCase.DisplayName,
			Match:       localCase.Match,
			Description: localCase.Description,
			Properties: lo.Map(localCase.Properties, func(localProp localcatalog.PropertyReference, _ int) PropertyReference {
				return PropertyReference{
					ID:       urnFromRef(localProp.Ref),
					Required: localProp.Required,
				}
			}),
		})
	}

	v.Default = lo.Map(localVariant.Default, func(localProp localcatalog.PropertyReference, _ int) PropertyReference {
		return PropertyReference{
			ID:       urnFromRef(localProp.Ref),
			Required: localProp.Required,
		}
	})

	return nil
}

type VariantCase struct {
	DisplayName string              `json:"display_name"`
	Match       []any               `json:"match"`
	Description *string             `json:"description"`
	Properties  []PropertyReference `json:"properties"`
}

type PropertyReference struct {
	ID       any  `json:"id"`
	Required bool `json:"required"`
}

func (v Variants) Diff(against Variants) bool {
	if len(v) != len(against) {
		return true
	}

	if v == nil && against == nil {
		return false
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

	if len(v.Default) != len(against.Default) {
		return true
	}
	for i := range v.Default {
		if v.Default[i].diffPropertyReference(against.Default[i]) {
			return true
		}
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

	if (vc.Description == nil) != (against.Description == nil) {
		return true
	}
	if vc.Description != nil && against.Description != nil && *vc.Description != *against.Description {
		return true
	}

	if len(vc.Properties) != len(against.Properties) {
		return true
	}
	for i := range vc.Properties {
		if vc.Properties[i].diffPropertyReference(against.Properties[i]) {
			return true
		}
	}

	return false
}

func (pr PropertyReference) diffPropertyReference(against PropertyReference) bool {
	if !reflect.DeepEqual(pr.ID, against.ID) {
		return true
	}

	return pr.Required != against.Required
}
