package engine

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/validation"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/location"
)

// Diagnostic represents an enriched validation error with file context
type Diagnostic struct {
	File     string              // file path relative to project root
	Rule     string              // rule ID that generated this (empty for parse errors)
	Severity validation.Severity // error/warning/info
	Message  string              // human-readable error message
	Position location.Position   // line/column position
	Fragment string              // code fragment associated with the error (optional)
}
