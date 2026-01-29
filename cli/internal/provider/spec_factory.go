package provider

import "github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"

// SpecFactory provides a factory method for creating spec instances
// and examples for documentation/error reporting. This enables go-validator
// based syntactic validation across all provider types.
type SpecFactory interface {
	// Kind returns the spec kind this factory handles (e.g., "properties", "event-stream-source")
	Kind() string

	// NewSpec creates a new zero-value instance of the spec type.
	NewSpec() any

	// SpecFieldName returns the field name in the spec that contains
	// the primary entities (e.g., "properties", "events", "types")
	// Returns empty string if the spec doesn't have a container field.
	SpecFieldName() string

	// Examples returns valid and invalid YAML snippets for this spec type.
	// These are used in error messages to show users correct/incorrect formats.
	Examples() rules.Examples
}

// SpecFactoryProvider is an optional interface that providers can implement
// to supply spec factories for validation.
type SpecFactoryProvider interface {
	// SpecFactories returns all spec factories this provider supports.
	SpecFactories() []SpecFactory
}
