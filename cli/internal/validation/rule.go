package validation

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/location"
)

// Severity represents the severity level of a validation error
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

// ValidationError represents an error found during validation
type ValidationError struct {
	Msg      string
	Fragment string
	Pos      location.Position
}

// Rule defines the interface for a validation rule
type Rule interface {
	// ID returns a unique identifier for the rule
	ID() string
	
	// Validate executes the validation logic
	Validate(ctx *ValidationContext, graph *resources.Graph) []ValidationError
	
	// Severity returns the severity level of the rule
	Severity() Severity
	
	// Description returns a description of the rule
	Description() string
	
	// Examples returns examples of valid/invalid data for the rule
	Examples() [][]byte
	
	// AppliesTo returns the kinds of resources this rule applies to
	AppliesTo() []string
}

