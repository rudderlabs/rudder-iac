package plan

import (
	"fmt"
)

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

// ParseEventType converts a string to an EventType, returning an error for unknown types
func ParseEventType(s string) (EventType, error) {
	switch s {
	case string(EventTypeTrack):
		return EventTypeTrack, nil
	case string(EventTypeIdentify):
		return EventTypeIdentify, nil
	case string(EventTypePage):
		return EventTypePage, nil
	case string(EventTypeScreen):
		return EventTypeScreen, nil
	case string(EventTypeGroup):
		return EventTypeGroup, nil
	default:
		return "", fmt.Errorf("invalid event type: %s", s)
	}
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
	PrimitiveTypeAny     PrimitiveType = "any"
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
	Type        []PropertyType  `json:"type"`
	ItemType    []PropertyType  `json:"itemType,omitempty"` // Used if Type includes PrimitiveTypeArray
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

// ParsePrimitiveType converts a string to a PrimitiveType, returning an error for unknown types
func ParsePrimitiveType(s string) (PrimitiveType, error) {
	switch s {
	case string(PrimitiveTypeString):
		return PrimitiveTypeString, nil
	case string(PrimitiveTypeInteger):
		return PrimitiveTypeInteger, nil
	case string(PrimitiveTypeNumber):
		return PrimitiveTypeNumber, nil
	case string(PrimitiveTypeBoolean):
		return PrimitiveTypeBoolean, nil
	case string(PrimitiveTypeArray):
		return PrimitiveTypeArray, nil
	case string(PrimitiveTypeObject):
		return PrimitiveTypeObject, nil
	case string(PrimitiveTypeAny):
		return PrimitiveTypeAny, nil
	default:
		return "", fmt.Errorf("invalid primitive type: %s", s)
	}
}

// IsCustomType checks if the PropertyType is a CustomType
func IsCustomType(t PropertyType) bool {
	_, ok := t.(CustomType)
	return ok
}

// AsCustomType converts a PropertyType to a CustomType pointer if possible, else returns nil
func AsCustomType(t PropertyType) *CustomType {
	if customType, ok := t.(CustomType); ok {
		return &customType
	}
	return nil
}

/*
 * Custom Type related types
 */

// CustomType represents a custom type definition
type CustomType struct {
	Name        string        `json:"name"`
	Description string        `json:"description,omitempty"`
	Type        PrimitiveType `json:"type"`
	// Schema defines the structure of the custom type if it's an object
	Schema *ObjectSchema `json:"schema,omitempty"`
}

func (c *CustomType) IsPrimitive() bool {
	return c.Type != PrimitiveTypeObject
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

// ParseIdentitySection converts a string to an IdentitySection, returning an error for unknown sections
func ParseIdentitySection(s string) (IdentitySection, error) {
	switch s {
	case string(IdentitySectionProperties):
		return IdentitySectionProperties, nil
	case string(IdentitySectionTraits):
		return IdentitySectionTraits, nil
	default:
		return "", fmt.Errorf("invalid identity section: %s", s)
	}
}

// EventRule represents a rule for an event
type EventRule struct {
	Event   Event           `json:"event"`
	Section IdentitySection `json:"section"`
	Schema  ObjectSchema    `json:"schema"`
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
}
