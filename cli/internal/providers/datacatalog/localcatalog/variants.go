package localcatalog

// Variants represents a slice of conditional variants for local YAML configuration parsing.
// It provides the foundational data structure for conditional validation of properties
type Variants []Variant

type Variant struct {
	Type          string              `json:"type" validate:"required,eq=discriminator"`
	Discriminator string              `json:"discriminator" validate:"required,reference"`
	Cases         []VariantCase       `json:"cases" validate:"required,min=1,dive"`
	Default       []PropertyReference `json:"default,omitempty" validate:"omitempty,dive"`
}

type VariantCase struct {
	DisplayName string              `json:"display_name" validate:"required"`
	Match       []any               `json:"match"`
	Description string              `json:"description"`
	Properties  []PropertyReference `json:"properties" validate:"required,min=1,dive"`
}

type PropertyReference struct {
	Ref      string `json:"$ref" validate:"required,reference"`
	Required bool   `json:"required"`
}
