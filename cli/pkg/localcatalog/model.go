package localcatalog

import (
	"encoding/json"
	"fmt"
)

type ResourceDefinition struct {
	Version  string `yaml:"version"`
	Kind     string `yaml:"kind"`
	Metadata struct {
		Name string `yaml:"name"`
	} `yaml:"metadata"`
	Spec map[string]interface{} `yaml:"spec"`
}

type Property struct {
	LocalID     string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Type        string                 `json:"type"`
	Config      map[string]interface{} `json:"propConfig"`
}

type PropertySpec struct {
	Properties []Property `json:"properties"`
}

// This method is used to extract the entity from the byte representation of it
func ExtractProperties(rd *ResourceDefinition) ([]Property, error) {
	spec := PropertySpec{}

	jsonByt, err := json.Marshal(rd.Spec)
	if err != nil {
		return nil, fmt.Errorf("marshalling the spec: %w", err)
	}

	if err := json.Unmarshal(jsonByt, &spec); err != nil {
		return nil, fmt.Errorf("extracting the property spec: %w", err)
	}

	return spec.Properties, nil
}

type Event struct {
	LocalID     string   `json:"id"`
	Name        string   `json:"name"`
	Type        string   `json:"event_type"`
	Description string   `json:"description"`
	Categories  []string `json:"categories"`
}

type EventSpec struct {
	Events []Event `json:"events"`
}

// ExtractEvents simply parses the whole file defined as resource definition
// and returns the events from it.
func ExtractEvents(rd *ResourceDefinition) ([]Event, error) {
	spec := EventSpec{}

	jsonByt, err := json.Marshal(rd.Spec)
	if err != nil {
		return nil, fmt.Errorf("marshalling the spec: %w", err)
	}

	if err := json.Unmarshal(jsonByt, &spec); err != nil {
		return nil, fmt.Errorf("extracting events spec: %w", err)
	}

	return spec.Events, nil
}

// CustomType represents a user-defined custom type
type CustomType struct {
	LocalID     string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Type        string                 `json:"type"`
	Config      map[string]interface{} `json:"config"`
	Properties  []CustomTypeProperty   `json:"properties,omitempty"`
}

// CustomTypeProperty represents a property reference within a custom type
type CustomTypeProperty struct {
	Ref      string `json:"$ref"`
	Required bool   `json:"required"`
}

// CustomTypeSpec represents the spec section of a custom-types resource
type CustomTypeSpec struct {
	Types []CustomType `json:"types"`
}

// ExtractCustomTypes parses a resource definition and extracts custom types
func ExtractCustomTypes(rd *ResourceDefinition) ([]CustomType, error) {
	spec := CustomTypeSpec{}

	jsonByt, err := json.Marshal(rd.Spec)
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
