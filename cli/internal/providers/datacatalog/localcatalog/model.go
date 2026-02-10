package localcatalog

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
)

// strictUnmarshal performs JSON unmarshaling with strict validation
// It rejects unknown fields to catch configuration errors
func strictUnmarshal(data []byte, v any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields() // Enable strict mode - reject unknown fields
	return decoder.Decode(v)
}

type Property struct {
	LocalID     string                 `mapstructure:"id" json:"id" validate:"required"`
	Name        string                 `mapstructure:"name" json:"name" validate:"required,gte=1,lte=65"`
	Description string                 `mapstructure:"description,omitempty" json:"description" validate:"omitempty,gte=3,lte=2000"`
	Type        string                 `mapstructure:"type,omitempty" json:"type" validate:"omitempty,primitive_or_reference"`
	Config      map[string]interface{} `mapstructure:"propConfig,omitempty" json:"propConfig,omitempty"`
}

type PropertySpec struct {
	Properties []Property `json:"properties" validate:"dive"`
}

// Event represents a user-defined event (V0 spec)
type Event struct {
	LocalID     string  `json:"id" mapstructure:"id" validate:"required"`
	Name        string  `json:"name" mapstructure:"name,omitempty"`
	Type        string  `json:"event_type" mapstructure:"event_type" validate:"oneof=track screen identify group page"`
	Description string  `json:"description" mapstructure:"description,omitempty" validate:"omitempty,gte=3,lte=2000,pattern=letter_start"`
	CategoryRef *string `json:"category" mapstructure:"category,omitempty" validate:"omitempty,pattern=legacy_category_ref"`
}

type EventSpec struct {
	Events []Event `json:"events" validate:"dive"`
}

// This method is used to extract the entity from the byte representation of it
func ExtractProperties(s *specs.Spec) ([]PropertyV1, error) {
	spec := PropertySpec{}

	jsonByt, err := json.Marshal(s.Spec)
	if err != nil {
		return nil, fmt.Errorf("marshalling the spec: %w", err)
	}

	if err := strictUnmarshal(jsonByt, &spec); err != nil {
		return nil, fmt.Errorf("extracting the property spec: %w", err)
	}

	v1Properties := make([]PropertyV1, 0, len(spec.Properties))
	for _, property := range spec.Properties {
		v1Property := PropertyV1{}
		err := v1Property.FromV0(property)
		if err != nil {
			return nil, fmt.Errorf("converting property to v1: %w", err)
		}
		v1Properties = append(v1Properties, v1Property)
	}

	return v1Properties, nil
}

type EventV1 struct {
	LocalID     string  `json:"id" mapstructure:"id"`
	Name        string  `json:"name,omitempty" mapstructure:"name,omitempty"`
	Type        string  `json:"event_type" mapstructure:"event_type"`
	Description string  `json:"description,omitempty" mapstructure:"description,omitempty"`
	CategoryRef *string `json:"category,omitempty" mapstructure:"category,omitempty"`
}

type EventSpecV1 struct {
	Events []EventV1 `json:"events"`
}

// ExtractEvents simply parses the whole file defined as resource definition
// and returns the events from it.
func ExtractEvents(s *specs.Spec) ([]EventV1, error) {
	spec := EventSpecV1{}

	jsonByt, err := json.Marshal(s.Spec)
	if err != nil {
		return nil, fmt.Errorf("marshalling the spec: %w", err)
	}

	if err := strictUnmarshal(jsonByt, &spec); err != nil {
		return nil, fmt.Errorf("extracting events spec: %w", err)
	}

	return spec.Events, nil
}

// CategoryV1 represents a user-defined category
type CategoryV1 struct {
	LocalID string `mapstructure:"id" json:"id"`
	Name    string `mapstructure:"name" json:"name"`
}

// Category represents a user-defined category (V0 spec)
type Category struct {
	LocalID string `mapstructure:"id" json:"id" validate:"required"`
	Name    string `mapstructure:"name" json:"name" validate:"required,pattern=category_name"`
}

type CategorySpec struct {
	Categories []Category `json:"categories" validate:"dive"`
}

// CategorySpecV1 represents the spec section of a categories resource
type CategorySpecV1 struct {
	Categories []CategoryV1 `json:"categories"`
}

// ExtractCategories parses a resource definition and extracts categories
func ExtractCategories(s *specs.Spec) ([]CategoryV1, error) {
	spec := CategorySpecV1{}

	jsonByt, err := json.Marshal(s.Spec)
	if err != nil {
		return nil, fmt.Errorf("marshalling the spec: %w", err)
	}

	if err := strictUnmarshal(jsonByt, &spec); err != nil {
		return nil, fmt.Errorf("extracting categories spec: %w", err)
	}

	return spec.Categories, nil
}

// CustomType represents a user-defined custom type
type CustomType struct {
	LocalID     string               `mapstructure:"id" json:"id" validate:"required"`
	Name        string               `mapstructure:"name" json:"name" validate:"required,gte=2,lte=65,pattern=custom_type_name"`
	Description string               `mapstructure:"description,omitempty" json:"description,omitempty" validate:"omitempty,gte=3,lte=2000,pattern=letter_start"`
	Type        string               `mapstructure:"type" json:"type" validate:"required,pattern=primitive_type"`
	Config      map[string]any       `mapstructure:"config,omitempty" json:"config,omitempty"`
	Properties  []CustomTypeProperty `mapstructure:"properties,omitempty" json:"properties,omitempty" validate:"omitempty,dive"`
	Variants    Variants             `mapstructure:"variants,omitempty" json:"variants,omitempty"`
}

// CustomTypeProperty represents a property reference within a custom type
type CustomTypeProperty struct {
	Ref      string `mapstructure:"$ref" json:"$ref" validate:"required,pattern=legacy_property_ref"`
	Required bool   `mapstructure:"required" json:"required"`
}

// CustomTypeSpec represents the spec section of a custom-types resource
type CustomTypeSpec struct {
	Types []CustomType `json:"types" validate:"required,dive"`
}

// ExtractCustomTypes parses a resource definition and extracts custom types
func ExtractCustomTypes(s *specs.Spec) ([]CustomTypeV1, error) {
	spec := CustomTypeSpec{}

	jsonByt, err := json.Marshal(s.Spec)
	if err != nil {
		return nil, fmt.Errorf("marshalling the spec: %w", err)
	}

	if err := strictUnmarshal(jsonByt, &spec); err != nil {
		return nil, fmt.Errorf("extracting custom types spec: %w", err)
	}

	// Ensure config is initialized as an empty map when nil
	for i := range spec.Types {
		if spec.Types[i].Config == nil {
			spec.Types[i].Config = make(map[string]interface{})
		}
	}

	// Convert V0 to V1
	v1CustomTypes := make([]CustomTypeV1, 0, len(spec.Types))
	for _, customType := range spec.Types {
		v1CustomType := CustomTypeV1{}
		err := v1CustomType.FromV0(customType)
		if err != nil {
			return nil, fmt.Errorf("converting custom type to v1: %w", err)
		}
		v1CustomTypes = append(v1CustomTypes, v1CustomType)
	}

	return v1CustomTypes, nil
}
