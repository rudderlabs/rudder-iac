package validation

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/location"
)

// Metadata represents common metadata fields for validation
type Metadata struct {
	Name string
}

// ValidationContext provides information needed for rule validation
type ValidationContext struct {
	Metadata  *Metadata
	Spec      any    // parsed spec content -> type
	Path      string // file path
	Filename  string
	Kind      string // "properties", "events", "tp", etc.
	Version   string
	PathIndex *location.PathIndex // for position lookup
}
