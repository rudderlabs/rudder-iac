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
	PrimitiveTypeInteger PrimitiveType = "integer"
	PrimitiveTypeNumber  PrimitiveType = "number"
	PrimitiveTypeBoolean PrimitiveType = "boolean"
	PrimitiveTypeArray   PrimitiveType = "array"
	PrimitiveTypeObject  PrimitiveType = "object"
	PrimitiveTypeNull    PrimitiveType = "null"
)

// PropertyType represents either a primitive type or a custom type
// In practice, this will contain either a PrimitiveType or CustomType
type PropertyType any

// PropertyConfig represents additional configuration for a property
type PropertyConfig struct {
	Enum []any `json:"enum,omitempty"`
}

// Property represents a property definition
type Property struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Types       []PropertyType  `json:"type"`
	ItemTypes   []PropertyType  `json:"itemType,omitempty"` // Used if Type includes PrimitiveTypeArray
	Config      *PropertyConfig `json:"config,omitempty"`
}

// IsPrimitiveType checks if the PropertyType is a PrimitiveType
func IsPrimitiveType(t PropertyType) bool {
	_, ok := t.(PrimitiveType)
	return ok
}

// AsPrimitiveType converts a PropertyType to a PrimitiveType pointer if possible, else returns nil
func AsPrimitiveType(t PropertyType) *PrimitiveType {
	if primitiveType, ok := t.(PrimitiveType); ok {
		return &primitiveType
	}
	return nil
}

// IsCustomType checks if the PropertyType is a CustomType (value or pointer)
func IsCustomType(t PropertyType) bool {
	_, ok := t.(CustomType)
	if ok {
		return true
	}
	_, ok = t.(*CustomType)
	return ok
}

// AsCustomType converts a PropertyType to a CustomType pointer if possible, else returns nil
func AsCustomType(t PropertyType) *CustomType {
	if customType, ok := t.(CustomType); ok {
		return &customType
	}
	if customType, ok := t.(*CustomType); ok {
		return customType
	}
	return nil
}

/*
 * Custom Type related types
 */

// Variant represents a discriminator-based variant definition
type Variant struct {
	Type          string        `json:"type"` // Always "discriminator" for now
	Discriminator string        `json:"discriminator"`
	Cases         []VariantCase `json:"cases"`
	DefaultSchema *ObjectSchema `json:"defaultSchema,omitempty"`
}

// VariantCase represents a single case in a variant
type VariantCase struct {
	DisplayName string       `json:"displayName,omitempty"`
	Match       []any        `json:"match"`
	Description string       `json:"description,omitempty"`
	Schema      ObjectSchema `json:"schema"`
}

// CustomType represents a custom type definition
type CustomType struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Type        PrimitiveType   `json:"type"`
	ItemType    PropertyType    `json:"itemType,omitempty"` // Used if Type is PrimitiveTypeArray
	Config      *PropertyConfig `json:"config,omitempty"`
	// Schema defines the structure of the custom type if it's an object
	Schema   *ObjectSchema `json:"schema,omitempty"`
	Variants []Variant     `json:"variants,omitempty"`
}

func (c *CustomType) IsPrimitive() bool {
	return c.Type != PrimitiveTypeObject || c.Schema == nil || len(c.Schema.Properties) == 0
}

/*
 * Event Rule related types
 */

// IdentitySection represents the section of an event rule
type IdentitySection string

const (
	IdentitySectionProperties IdentitySection = "properties"
	IdentitySectionTraits     IdentitySection = "traits"
)

// EventRule represents a rule for an event
type EventRule struct {
	Event    Event           `json:"event"`
	Section  IdentitySection `json:"section"`
	Schema   ObjectSchema    `json:"schema"`
	Variants []Variant       `json:"variants,omitempty"`
}

// ObjectSchema represents the schema for an object
type ObjectSchema struct {
	Properties           map[string]PropertySchema `json:"properties"`
	AdditionalProperties bool                      `json:"additionalProperties"`
}

// PropertySchema represents the schema for a property within an ObjectSchema
type PropertySchema struct {
	Property Property `json:"property"`
	Required bool     `json:"required"`
	// Schema represents a nested object schema for the property, if applicable
	Schema *ObjectSchema `json:"schema,omitempty"`
}

/*
 * Plan related types
 */

// Plan represents a tracking plan with its rules
type TrackingPlan struct {
	Name  string      `json:"name"`
	Rules []EventRule `json:"rules"`

	// Metadata represents additional fixed context to be included with every event
	Metadata PlanMetadata `json:"eventContext,omitempty"`
}

type PlanMetadata struct {
	TrackingPlanID      string `json:"trackingPlanId,omitempty"`
	TrackingPlanVersion int    `json:"trackingPlanVersion,omitempty"`
	URL                 string `json:"url,omitempty"`
}
