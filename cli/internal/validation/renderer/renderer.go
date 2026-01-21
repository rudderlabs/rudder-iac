package renderer

import "github.com/rudderlabs/rudder-iac/cli/internal/validation"

// Renderer takes validation diagnostics and renders them to appropriate output.
// Different implementations can format diagnostics for CLI text, JSON, LSP, etc.
type Renderer interface {
	// Render outputs diagnostics in the renderer's specific format.
	Render(diagnostics []validation.Diagnostic) error
}
