package events

import (
	"fmt"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/schema/models"
	"github.com/rudderlabs/rudder-iac/cli/pkg/logger"
	schemaErrors "github.com/rudderlabs/rudder-iac/cli/pkg/schema/errors"
)

// EventInfo holds extracted event information
type EventInfo struct {
	Name        string                 `json:"name"`
	Identifier  string                 `json:"identifier"`
	EventType   string                 `json:"event_type"`
	WriteKey    string                 `json:"write_key"`
	Schema      map[string]interface{} `json:"schema"`
	Count       int                    `json:"count"`
	Description string                 `json:"description,omitempty"`
}

// NameGenerator generates clean event names from identifiers
type NameGenerator interface {
	GenerateName(identifier string) string
	ValidateName(name string) error
}

// IDValidator validates event identifiers
type IDValidator interface {
	ValidateID(id string) error
	IsValidID(id string) bool
}

// EventExtractor extracts and processes events from schemas
type EventExtractor struct {
	nameGenerator NameGenerator
	idValidator   IDValidator
	logger        *logger.Logger
}

// defaultNameGenerator provides basic name generation
type defaultNameGenerator struct{}

func (g *defaultNameGenerator) GenerateName(identifier string) string {
	// Convert to PascalCase
	parts := strings.FieldsFunc(identifier, func(r rune) bool {
		return r == '_' || r == '-' || r == ' '
	})

	var result strings.Builder
	for _, part := range parts {
		if len(part) > 0 {
			result.WriteString(strings.ToUpper(part[:1]))
			if len(part) > 1 {
				result.WriteString(strings.ToLower(part[1:]))
			}
		}
	}

	name := result.String()
	if name == "" {
		return "UnknownEvent"
	}
	return name
}

func (g *defaultNameGenerator) ValidateName(name string) error {
	if name == "" {
		return schemaErrors.NewProcessError(
			schemaErrors.ErrorTypeProcessValidation,
			"Event name cannot be empty",
			schemaErrors.AsUserError(),
		)
	}

	if len(name) > 100 {
		return schemaErrors.NewProcessError(
			schemaErrors.ErrorTypeProcessValidation,
			"Event name too long (max 100 characters)",
			schemaErrors.AsUserError(),
		)
	}

	return nil
}

// defaultIDValidator provides basic ID validation
type defaultIDValidator struct{}

func (v *defaultIDValidator) ValidateID(id string) error {
	if id == "" {
		return schemaErrors.NewProcessError(
			schemaErrors.ErrorTypeProcessValidation,
			"Event identifier cannot be empty",
			schemaErrors.AsUserError(),
		)
	}

	if len(id) > 200 {
		return schemaErrors.NewProcessError(
			schemaErrors.ErrorTypeProcessValidation,
			"Event identifier too long (max 200 characters)",
			schemaErrors.AsUserError(),
		)
	}

	return nil
}

func (v *defaultIDValidator) IsValidID(id string) bool {
	return v.ValidateID(id) == nil
}

// NewEventExtractor creates a new event extractor
func NewEventExtractor(log *logger.Logger) *EventExtractor {
	return &EventExtractor{
		nameGenerator: &defaultNameGenerator{},
		idValidator:   &defaultIDValidator{},
		logger:        log,
	}
}

// NewEventExtractorWithDeps creates an event extractor with custom dependencies
func NewEventExtractorWithDeps(nameGen NameGenerator, idValidator IDValidator, log *logger.Logger) *EventExtractor {
	return &EventExtractor{
		nameGenerator: nameGen,
		idValidator:   idValidator,
		logger:        log,
	}
}

// ExtractEvents extracts events from a collection of schemas
func (e *EventExtractor) ExtractEvents(schemas []models.Schema) (map[string]*EventInfo, error) {
	events := make(map[string]*EventInfo)

	for _, schema := range schemas {
		// Validate the schema
		if err := e.validateSchema(schema); err != nil {
			e.logger.Warn(fmt.Sprintf("Skipping invalid schema %s: %v", schema.UID, err))
			continue
		}

		// Generate event name
		eventName := e.nameGenerator.GenerateName(schema.EventIdentifier)
		if err := e.nameGenerator.ValidateName(eventName); err != nil {
			return nil, schemaErrors.NewProcessError(
				schemaErrors.ErrorTypeProcessValidation,
				fmt.Sprintf("Invalid generated event name for %s", schema.EventIdentifier),
				schemaErrors.WithCause(err),
				schemaErrors.WithSchemaUID(schema.UID),
			)
		}

		// Create or update event info
		if existing, exists := events[eventName]; exists {
			// Merge with existing event
			existing.Count += schema.Count
			e.mergeSchemas(existing, &schema)
		} else {
			// Create new event
			events[eventName] = &EventInfo{
				Name:        eventName,
				Identifier:  schema.EventIdentifier,
				EventType:   schema.EventType,
				WriteKey:    schema.WriteKey,
				Schema:      schema.Schema,
				Count:       schema.Count,
				Description: e.generateDescription(schema),
			}
		}
	}

	e.logger.Info(fmt.Sprintf("Extracted %d unique events from %d schemas", len(events), len(schemas)))

	return events, nil
}

// validateSchema validates a schema for event extraction
func (e *EventExtractor) validateSchema(schema models.Schema) error {
	if schema.UID == "" {
		return schemaErrors.NewProcessError(
			schemaErrors.ErrorTypeProcessValidation,
			"Schema missing UID",
			schemaErrors.AsUserError(),
		)
	}

	if err := e.idValidator.ValidateID(schema.EventIdentifier); err != nil {
		return schemaErrors.NewProcessError(
			schemaErrors.ErrorTypeProcessValidation,
			"Invalid event identifier",
			schemaErrors.WithCause(err),
			schemaErrors.WithSchemaUID(schema.UID),
		)
	}

	if schema.EventType == "" {
		return schemaErrors.NewProcessError(
			schemaErrors.ErrorTypeProcessValidation,
			"Schema missing event type",
			schemaErrors.WithSchemaUID(schema.UID),
			schemaErrors.AsUserError(),
		)
	}

	return nil
}

// mergeSchemas merges schema information from multiple sources
func (e *EventExtractor) mergeSchemas(existing *EventInfo, newSchema *models.Schema) {
	// For now, keep the schema from the most recent entry
	// In a more sophisticated implementation, we might merge the schemas
	if newSchema.Schema != nil && len(newSchema.Schema) > len(existing.Schema) {
		existing.Schema = newSchema.Schema
	}

	// Update write key if it's more specific (non-empty)
	if existing.WriteKey == "" && newSchema.WriteKey != "" {
		existing.WriteKey = newSchema.WriteKey
	}
}

// generateDescription creates a description for the event
func (e *EventExtractor) generateDescription(schema models.Schema) string {
	if schema.EventType == "track" {
		return fmt.Sprintf("Track event: %s", schema.EventIdentifier)
	}
	return fmt.Sprintf("%s event: %s", strings.Title(schema.EventType), schema.EventIdentifier)
}

// GetEventsByWriteKey groups events by write key
func (e *EventExtractor) GetEventsByWriteKey(events map[string]*EventInfo) map[string]map[string]*EventInfo {
	eventsByWriteKey := make(map[string]map[string]*EventInfo)

	for eventName, eventInfo := range events {
		writeKey := eventInfo.WriteKey
		if writeKey == "" {
			writeKey = "default"
		}

		if eventsByWriteKey[writeKey] == nil {
			eventsByWriteKey[writeKey] = make(map[string]*EventInfo)
		}

		eventsByWriteKey[writeKey][eventName] = eventInfo
	}

	return eventsByWriteKey
}

// FilterEventsByType filters events by event type
func (e *EventExtractor) FilterEventsByType(events map[string]*EventInfo, eventType string) map[string]*EventInfo {
	filtered := make(map[string]*EventInfo)

	for eventName, eventInfo := range events {
		if eventInfo.EventType == eventType {
			filtered[eventName] = eventInfo
		}
	}

	return filtered
}

// GetEventStats returns statistics about extracted events
func (e *EventExtractor) GetEventStats(events map[string]*EventInfo) map[string]interface{} {
	stats := make(map[string]interface{})

	totalEvents := len(events)
	totalCount := 0
	eventTypes := make(map[string]int)
	writeKeys := make(map[string]int)

	for _, eventInfo := range events {
		totalCount += eventInfo.Count
		eventTypes[eventInfo.EventType]++

		writeKey := eventInfo.WriteKey
		if writeKey == "" {
			writeKey = "default"
		}
		writeKeys[writeKey]++
	}

	stats["total_events"] = totalEvents
	stats["total_count"] = totalCount
	stats["event_types"] = eventTypes
	stats["write_keys"] = writeKeys

	if totalEvents > 0 {
		stats["average_count"] = totalCount / totalEvents
	}

	return stats
}
