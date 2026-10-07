package renderer

import "github.com/rudderlabs/rudder-iac/cli/internal/validation"

// CaptureRenderer accumulates diagnostics instead of writing them out, for
// callers that post-process them rather than show them (JSON output, editor
// integrations).
//
// A single load calls Render more than once — syntax diagnostics render before
// the graph is built, semantic ones after — so diagnostics accumulate across
// calls rather than replacing what came before.
type CaptureRenderer struct {
	diagnostics validation.Diagnostics
}

func NewCaptureRenderer() *CaptureRenderer {
	return &CaptureRenderer{
		diagnostics: make(validation.Diagnostics, 0),
	}
}

func (r *CaptureRenderer) Render(diagnostics validation.Diagnostics) error {
	r.diagnostics = append(r.diagnostics, diagnostics...)
	return nil
}

// Diagnostics returns everything captured so far, sorted by file and position.
func (r *CaptureRenderer) Diagnostics() validation.Diagnostics {
	r.diagnostics.Sort()
	return r.diagnostics
}
