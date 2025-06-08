package converter

import "github.com/rudderlabs/rudder-iac/cli/internal/schema/models"

// SchemaAnalyzer analyzes schemas to extract events, properties, and custom types
type SchemaAnalyzer struct {
	Events      map[string]*EventInfo
	Properties  map[string]*PropertyInfo
	CustomTypes map[string]*CustomTypeInfo

	// Uniqueness tracking
	UsedCustomTypeNames map[string]bool
	UsedPropertyIDs     map[string]bool
	UsedEventIDs        map[string]bool
}

// EventInfo holds information about an extracted event
type EventInfo struct {
	ID          string
	Name        string
	EventType   string
	Description string
	Original    models.Schema
}

// PropertyInfo holds information about an extracted property
type PropertyInfo struct {
	ID          string
	Name        string
	Type        string
	Description string
	Path        string
	JsonType    string
}

// CustomTypeInfo holds information about a custom type
type CustomTypeInfo struct {
	ID            string
	Name          string
	Type          string
	Description   string
	Structure     map[string]string
	ArrayItemType string
	Hash          string
}

// SanitizationMode defines different sanitization strategies
type SanitizationMode int

const (
	SanitizationModeBasic SanitizationMode = iota
	SanitizationModeEvent
)

// UniquenessStrategy defines how to resolve naming conflicts
type UniquenessStrategy int

const (
	UniquenessStrategyCounter UniquenessStrategy = iota
	UniquenessStrategyLetterSuffix
)
