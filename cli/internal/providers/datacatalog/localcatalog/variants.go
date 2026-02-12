package localcatalog

import "fmt"

// V0 Variants - used by tracking plans with $ref field
// Variants represents a slice of conditional variants for local YAML configuration parsing.
// It provides the foundational data structure for conditional validation of properties
type Variants []Variant

type Variant struct {
	Type          string              `json:"type" validate:"required,eq=discriminator"`
	Discriminator string              `json:"discriminator" validate:"required,pattern=legacy_property_ref"`
	Cases         []VariantCase       `json:"cases" validate:"required,min=1,dive"`
	Default       []PropertyReference `json:"default,omitempty" validate:"omitempty,dive"`
}

type VariantCase struct {
	DisplayName string              `json:"display_name" validate:"required"`
	Match       []any               `json:"match" validate:"required,min=1,array_item_types=string bool integer"`
	Description string              `json:"description"`
	Properties  []PropertyReference `json:"properties" validate:"required,min=1,dive"`
}

type PropertyReference struct {
	Ref      string `json:"$ref" validate:"required,pattern=legacy_property_ref"`
	Required bool   `json:"required"`
}

// V1 Variants - used by custom types with property field instead of $ref
type VariantsV1 []VariantV1

type VariantV1 struct {
	Type          string              `json:"type"`
	Discriminator string              `json:"discriminator"`
	Cases         []VariantCaseV1     `json:"cases"`
	Default       DefaultPropertiesV1 `json:"default"`
}

type DefaultPropertiesV1 struct {
	Properties []PropertyReferenceV1 `json:"properties"`
}

type VariantCaseV1 struct {
	DisplayName string                `json:"display_name"`
	Match       []any                 `json:"match"`
	Description string                `json:"description"`
	Properties  []PropertyReferenceV1 `json:"properties"`
}

type PropertyReferenceV1 struct {
	Property string `json:"property"`
	Required bool   `json:"required"`
}

func (v *VariantV1) FromV0(v0 Variant) error {
	v.Type = v0.Type
	v.Discriminator = v0.Discriminator

	// Convert cases from V0 to V1
	if len(v0.Cases) > 0 {
		v.Cases = make([]VariantCaseV1, 0, len(v0.Cases))
		for _, v0Case := range v0.Cases {
			vc := VariantCaseV1{}
			vc.DisplayName = v0Case.DisplayName
			vc.Match = v0Case.Match
			vc.Description = v0Case.Description

			// Convert properties from V0 to V1
			if len(v0Case.Properties) > 0 {
				vc.Properties = make([]PropertyReferenceV1, 0, len(v0Case.Properties))
				for _, v0Property := range v0Case.Properties {
					vc.Properties = append(vc.Properties, PropertyReferenceV1{
						Property: v0Property.Ref,
						Required: v0Property.Required,
					})
				}
			} else {
				vc.Properties = []PropertyReferenceV1{}
			}
			v.Cases = append(v.Cases, vc)
		}
	} else {
		v.Cases = []VariantCaseV1{}
	}

	// Convert default from V0 to V1
	if len(v0.Default) > 0 {
		v.Default.Properties = make([]PropertyReferenceV1, 0, len(v0.Default))
		for _, v0Default := range v0.Default {
			v.Default.Properties = append(v.Default.Properties, PropertyReferenceV1{
				Property: v0Default.Ref,
				Required: v0Default.Required,
			})
		}
	} else {
		v.Default.Properties = []PropertyReferenceV1{}
	}

	return nil
}

func (vs *VariantsV1) FromV0(v0 Variants) error {
	if len(v0) > 0 {
		*vs = make([]VariantV1, 0, len(v0))
		for _, v0Variant := range v0 {
			v1Variant := VariantV1{}
			err := v1Variant.FromV0(v0Variant)
			if err != nil {
				return fmt.Errorf("converting variant to v1: %w", err)
			}
			*vs = append(*vs, v1Variant)
		}
	}

	return nil
}
