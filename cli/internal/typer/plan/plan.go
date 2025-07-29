package plan

/*
 * Event related types
 */

// EventType represents the type of analytics event
type EventType string

const (
	EventTypeTrack    EventType = "track"
	EventTypeIdentify EventType = "identify"
	EventTypePage     EventType = "page"
	EventTypeScreen   EventType = "screen"
	EventTypeGroup    EventType = "group"
)

// Event represents an analytics event
type Event struct {
	EventType   EventType `json:"eventType"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
}

/*
 * Property related types
 */

// PrimitiveType represents primitive data types
type PrimitiveType string

const (
	PrimitiveTypeString  PrimitiveType = "string"
	PrimitiveTypeNumber  PrimitiveType = "number"
	PrimitiveTypeBoolean PrimitiveType = "boolean"
	PrimitiveTypeArray   PrimitiveType = "array"
	PrimitiveTypeObject  PrimitiveType = "object"
	PrimitiveTypeDate    PrimitiveType = "date"
)

// PropertyType represents either a primitive type or a custom type
// In practice, this will contain either a PrimitiveType or CustomType
type PropertyType any

// PropertyConfig represents additional configuration for a property
type PropertyConfig struct {
	Enum []string `json:"enum,omitempty"`
}

// Property represents a property definition
type Property struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Type        PropertyType    `json:"type"`
	Config      *PropertyConfig `json:"config,omitempty"`
}

func (p *Property) IsPrimitive() bool {
	_, ok := p.Type.(PrimitiveType)
	return ok
}

func (p Property) IsCustomType() bool {
	_, ok := p.Type.(CustomType)
	return ok
}

func (p *Property) CustomType() *CustomType {
	if customType, ok := p.Type.(CustomType); ok {
		return &customType
	}
	return nil
}

func (p *Property) PrimitiveType() PrimitiveType {
	if primitiveType, ok := p.Type.(PrimitiveType); ok {
		return primitiveType
	}
	return ""
}

/*
 * Custom Type related types
 */

// CustomType represents a custom type definition
type CustomType struct {
	Name        string        `json:"name"`
	Description string        `json:"description,omitempty"`
	Type        PrimitiveType `json:"type"`
	Schema      ObjectSchema  `json:"schema,omitempty"`
}

func (c *CustomType) IsPrimitive() bool {
	return c.Type != PrimitiveTypeObject
}

/*
 * Event Rule related types
 */

// EventRuleSection represents the section of an event rule
type EventRuleSection string

const (
	EventRuleSectionProperties EventRuleSection = "properties"
	EventRuleSectionTraits     EventRuleSection = "traits"
)

// EventRule represents a rule for an event
type EventRule struct {
	Event   Event            `json:"event"`
	Section EventRuleSection `json:"section"`
	Schema  ObjectSchema     `json:"schema"`
}

/*
 * Plan related types
 */

// Plan represents a tracking plan with its rules
type TrackingPlan struct {
	Name  string      `json:"name"`
	Rules []EventRule `json:"rules"`
}

// ObjectSchema represents the schema for an object
type ObjectSchema struct {
	Properties           map[string]PropertySchema `json:"properties"`
	AdditionalProperties bool                      `json:"additionalProperties"`
}

// PropertySchema represents the schema for a property within an ObjectSchema
type PropertySchema struct {
	Property Property      `json:"property"`
	Required bool          `json:"required"`
	Schema   *ObjectSchema `json:"schema,omitempty"`
}
