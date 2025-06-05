package models

// YAMLMetadata represents the metadata section in YAML files
type YAMLMetadata struct {
	Name string `yaml:"name"`
}

// EventsYAML represents the events.yaml file structure
type EventsYAML struct {
	Version  string       `yaml:"version"`
	Kind     string       `yaml:"kind"`
	Metadata YAMLMetadata `yaml:"metadata"`
	Spec     EventsSpec   `yaml:"spec"`
}

// EventsSpec contains the events specification
type EventsSpec struct {
	Events []EventDefinition `yaml:"events"`
}

// EventDefinition represents an individual event
type EventDefinition struct {
	ID          string `yaml:"id"`
	Name        string `yaml:"name,omitempty"`
	EventType   string `yaml:"event_type"`
	Description string `yaml:"description,omitempty"`
}

// PropertiesYAML represents the properties.yaml file structure
type PropertiesYAML struct {
	Version  string         `yaml:"version"`
	Kind     string         `yaml:"kind"`
	Metadata YAMLMetadata   `yaml:"metadata"`
	Spec     PropertiesSpec `yaml:"spec"`
}

// PropertiesSpec contains the properties specification
type PropertiesSpec struct {
	Properties []PropertyDefinition `yaml:"properties"`
}

// PropertyDefinition represents an individual property
type PropertyDefinition struct {
	ID          string          `yaml:"id"`
	Name        string          `yaml:"name"`
	Type        string          `yaml:"type"`
	Description string          `yaml:"description,omitempty"`
	PropConfig  *PropertyConfig `yaml:"propConfig,omitempty"`
}

// PropertyConfig contains validation rules for properties
type PropertyConfig struct {
	MinLength   *int     `yaml:"minLength,omitempty"`
	MaxLength   *int     `yaml:"maxLength,omitempty"`
	Pattern     string   `yaml:"pattern,omitempty"`
	Enum        []string `yaml:"enum,omitempty"`
	MinItems    *int     `yaml:"minItems,omitempty"`
	MaxItems    *int     `yaml:"maxItems,omitempty"`
	UniqueItems *bool    `yaml:"uniqueItems,omitempty"`
	Minimum     *float64 `yaml:"minimum,omitempty"`
	Maximum     *float64 `yaml:"maximum,omitempty"`
}

// CustomTypesYAML represents the custom-types.yaml file structure
type CustomTypesYAML struct {
	Version  string          `yaml:"version"`
	Kind     string          `yaml:"kind"`
	Metadata YAMLMetadata    `yaml:"metadata"`
	Spec     CustomTypesSpec `yaml:"spec"`
}

// CustomTypesSpec contains the custom types specification
type CustomTypesSpec struct {
	Types []CustomTypeDefinition `yaml:"types"`
}

// CustomTypeDefinition represents an individual custom type
type CustomTypeDefinition struct {
	ID          string            `yaml:"id"`
	Name        string            `yaml:"name"`
	Type        string            `yaml:"type"`
	Description string            `yaml:"description,omitempty"`
	Config      *CustomTypeConfig `yaml:"config,omitempty"`
	Properties  []PropertyRef     `yaml:"properties,omitempty"`
}

// CustomTypeConfig contains validation rules for custom types
type CustomTypeConfig struct {
	MinLength    *int     `yaml:"minLength,omitempty"`
	MaxLength    *int     `yaml:"maxLength,omitempty"`
	Pattern      string   `yaml:"pattern,omitempty"`
	Format       string   `yaml:"format,omitempty"`
	Enum         []string `yaml:"enum,omitempty"`
	Minimum      *float64 `yaml:"minimum,omitempty"`
	Maximum      *float64 `yaml:"maximum,omitempty"`
	ExclusiveMin *float64 `yaml:"exclusiveMinimum,omitempty"`
	ExclusiveMax *float64 `yaml:"exclusiveMaximum,omitempty"`
	MultipleOf   *float64 `yaml:"multipleOf,omitempty"`
	ItemTypes    []string `yaml:"itemTypes,omitempty"`
	MinItems     *int     `yaml:"minItems,omitempty"`
	MaxItems     *int     `yaml:"maxItems,omitempty"`
	UniqueItems  *bool    `yaml:"uniqueItems,omitempty"`
}

// PropertyRef represents a reference to a property in custom types
type PropertyRef struct {
	Ref      string `yaml:"$ref"`
	Required bool   `yaml:"required"`
}

// TrackingPlanYAML represents the tracking plan YAML file structure
type TrackingPlanYAML struct {
	Version  string           `yaml:"version"`
	Kind     string           `yaml:"kind"`
	Metadata YAMLMetadata     `yaml:"metadata"`
	Spec     TrackingPlanSpec `yaml:"spec"`
}

// TrackingPlanSpec contains the tracking plan specification
type TrackingPlanSpec struct {
	ID          string      `yaml:"id"`
	DisplayName string      `yaml:"display_name"`
	Description string      `yaml:"description,omitempty"`
	Rules       []EventRule `yaml:"rules"`
}

// EventRule represents an event rule in tracking plans
type EventRule struct {
	Type       string            `yaml:"type"`
	ID         string            `yaml:"id"`
	Event      EventRuleRef      `yaml:"event"`
	Properties []PropertyRuleRef `yaml:"properties"`
}

// EventRuleRef represents a reference to an event in tracking plans
type EventRuleRef struct {
	Ref             string `yaml:"$ref"`
	AllowUnplanned  bool   `yaml:"allow_unplanned"`
	IdentitySection string `yaml:"identity_section,omitempty"`
}

// PropertyRuleRef represents a reference to a property in event rules
type PropertyRuleRef struct {
	Ref      string `yaml:"$ref"`
	Required bool   `yaml:"required"`
}
