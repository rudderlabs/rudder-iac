package localcatalog

import (
	"encoding/json"
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
)

type Property struct {
	LocalID     string                 `mapstructure:"id" json:"id"`
	Name        string                 `mapstructure:"name" json:"name"`
	Description string                 `mapstructure:"description,omitempty" json:"description"`
	Type        string                 `mapstructure:"type,omitempty" json:"type"`
	Config      map[string]interface{} `mapstructure:"propConfig,omitempty" json:"propConfig"`
}

type PropertySpec struct {
	Properties []Property `json:"properties"`
}

// This method is used to extract the entity from the byte representation of it
func ExtractProperties(s *specs.Spec) ([]Property, error) {
	spec := PropertySpec{}

	jsonByt, err := json.Marshal(s.Spec)
	if err != nil {
		return nil, fmt.Errorf("marshalling the spec: %w", err)
	}

	if err := json.Unmarshal(jsonByt, &spec); err != nil {
		return nil, fmt.Errorf("extracting the property spec: %w", err)
	}

	return spec.Properties, nil
}

type Event struct {
	LocalID     string  `json:"id" mapstructure:"id"`
	Name        string  `json:"name" mapstructure:"name,omitempty"`
	Type        string  `json:"event_type" mapstructure:"event_type"`
	Description string  `json:"description" mapstructure:"description,omitempty"`
	CategoryRef *string `json:"category" mapstructure:"category,omitempty"`
}

type EventSpec struct {
	Events []Event `json:"events"`
}

// ExtractEvents simply parses the whole file defined as resource definition
// and returns the events from it.
func ExtractEvents(s *specs.Spec) ([]Event, error) {
	spec := EventSpec{}

	jsonByt, err := json.Marshal(s.Spec)
	if err != nil {
		return nil, fmt.Errorf("marshalling the spec: %w", err)
	}

	if err := json.Unmarshal(jsonByt, &spec); err != nil {
		return nil, fmt.Errorf("extracting events spec: %w", err)
	}

	return spec.Events, nil
}

// Category represents a user-defined category
type Category struct {
	LocalID string `mapstructure:"id" json:"id"`
	Name    string `mapstructure:"name" json:"name"`
}

// CategorySpec represents the spec section of a categories resource
type CategorySpec struct {
	Categories []Category `json:"categories"`
}

// ExtractCategories parses a resource definition and extracts categories
func ExtractCategories(s *specs.Spec) ([]Category, error) {
	spec := CategorySpec{}

	jsonByt, err := json.Marshal(s.Spec)
	if err != nil {
		return nil, fmt.Errorf("marshalling the spec: %w", err)
	}

	if err := json.Unmarshal(jsonByt, &spec); err != nil {
		return nil, fmt.Errorf("extracting categories spec: %w", err)
	}

	return spec.Categories, nil
}

// CustomType represents a user-defined custom type
type CustomType struct {
	LocalID     string               `mapstructure:"id" json:"id"`
	Name        string               `mapstructure:"name" json:"name"`
	Description string               `mapstructure:"description,omitempty" json:"description,omitempty"`
	Type        string               `mapstructure:"type" json:"type"`
	Config      map[string]any       `mapstructure:"config,omitempty" json:"config,omitempty"`
	Properties  []CustomTypeProperty `mapstructure:"properties,omitempty" json:"properties,omitempty"`
	Variants    Variants             `mapstructure:"variants,omitempty" json:"variants,omitempty"`
}

// CustomTypeProperty represents a property reference within a custom type
type CustomTypeProperty struct {
	Ref      string `mapstructure:"$ref" json:"$ref"`
	Required bool   `mapstructure:"required" json:"required"`
}

// CustomTypeSpec represents the spec section of a custom-types resource
type CustomTypeSpec struct {
	Types []CustomType `json:"types"`
}

// ExtractCustomTypes parses a resource definition and extracts custom types
func ExtractCustomTypes(s *specs.Spec) ([]CustomType, error) {
	spec := CustomTypeSpec{}

	jsonByt, err := json.Marshal(s.Spec)
	if err != nil {
		return nil, fmt.Errorf("marshalling the spec: %w", err)
	}

	if err := json.Unmarshal(jsonByt, &spec); err != nil {
		return nil, fmt.Errorf("extracting custom types spec: %w", err)
	}

	// Ensure config is initialized as an empty map when nil
	for i := range spec.Types {
		if spec.Types[i].Config == nil {
			spec.Types[i].Config = make(map[string]interface{})
		}
	}

	return spec.Types, nil
}
