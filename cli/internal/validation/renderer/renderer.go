package renderer

import "github.com/rudderlabs/rudder-iac/cli/internal/validation"

// Renderer takes validation diagnostics and renders them to appropriate output.
// Different implementations can format diagnostics for CLI text, JSON, LSP, etc.
type Renderer interface {
	// Render outputs diagnostics in the renderer's specific format.
	Render(diagnostics validation.Diagnostics) error
}

// LoadFailureRuleID labels the diagnostic a FailureRenderer adds when loading
// stops on an error that carries no diagnostics of its own.
const LoadFailureRuleID = "project/load-failed"

// FailureRenderer is implemented by a renderer whose output a machine reads. A
// load can stop on an error with no diagnostics (a provider rejecting a spec, a
// cycle in the graph), and a consumer must not read the resulting empty document
// as "validated clean". The renderer records the cause in the output, next to
// any warnings already collected. A text renderer does not implement it: the
// caller prints the error itself.
type FailureRenderer interface {
	RenderFailure(diagnostics validation.Diagnostics, cause error) error
}
