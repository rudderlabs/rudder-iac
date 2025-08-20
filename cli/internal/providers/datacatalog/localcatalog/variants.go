package localcatalog

// Variants represents a slice of conditional variants for local YAML configuration parsing.
// It provides the foundational data structure for conditional validation of properties
type Variants []Variant

type Variant struct {
	Type          string              `json:"type"`
	Discriminator string              `json:"discriminator"`
	Cases         []VariantCase       `json:"cases"`
	Default       []PropertyReference `json:"default"`
}

type VariantCase struct {
	DisplayName string              `json:"display_name"`
	Match       []any               `json:"match"`
	Description string              `json:"description"`
	Properties  []PropertyReference `json:"properties"`
}

type PropertyReference struct {
	Ref      string `json:"$ref"`
	Required bool   `json:"required"`
}