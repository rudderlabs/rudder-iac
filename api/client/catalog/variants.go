package catalog

type Variants []Variant

type Variant struct {
	Type          string              `json:"type"`
	Discriminator string              `json:"discriminator"`
	Cases         []VariantCase       `json:"cases"`
	Default       []PropertyReference `json:"default,omitempty"`
}

type VariantCase struct {
	DisplayName string              `json:"display_name"`
	Match       []any               `json:"match"`
	Description string              `json:"description,omitempty"`
	Properties  []PropertyReference `json:"properties"`
}

type PropertyReference struct {
	ID       string `json:"id"`
	Required bool   `json:"required"`
}
