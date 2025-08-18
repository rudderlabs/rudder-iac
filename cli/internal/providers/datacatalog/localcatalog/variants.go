package localcatalog

import (
	"encoding/json"
	"math"
)

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

func (vc *VariantCase) UnmarshalJSON(b []byte) error {
	type alias struct {
		DisplayName string              `json:"display_name"`
		Match       []any               `json:"match"`
		Description string              `json:"description"`
		Properties  []PropertyReference `json:"properties"`
	}
	var tmp alias
	if err := json.Unmarshal(b, &tmp); err != nil {
		return err
	}

	normalized := make([]any, len(tmp.Match))
	for i, v := range tmp.Match {
		switch n := v.(type) {
		case float64:
			if n == math.Trunc(n) {
				normalized[i] = int(n)
			} else {
				normalized[i] = n
			}
		default:
			normalized[i] = v
		}
	}

	vc.DisplayName = tmp.DisplayName
	vc.Match = normalized
	vc.Description = tmp.Description
	vc.Properties = tmp.Properties
	return nil
}
