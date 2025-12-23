// Package diagnostic provides diagnostic management for the LSP server
package diagnostic

import (
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// Manager handles publishing diagnostics to the LSP client
type Manager struct {
	context *glsp.Context
}

// NewManager creates a new diagnostic manager
func NewManager() *Manager {
	return &Manager{}
}

// SetContext sets the GLSP context for publishing diagnostics
func (m *Manager) SetContext(context *glsp.Context) {
	m.context = context
}

// PublishDiagnostics publishes diagnostics for a document URI
func (m *Manager) PublishDiagnostics(uri string, diagnostics []protocol.Diagnostic) error {
	if m.context == nil {
		// Context not set yet, skip publishing
		return nil
	}

	params := &protocol.PublishDiagnosticsParams{
		URI:         uri,
		Diagnostics: diagnostics,
	}

	m.context.Notify(protocol.ServerTextDocumentPublishDiagnostics, params)
	return nil
}

// ClearDiagnostics clears all diagnostics for a document URI
func (m *Manager) ClearDiagnostics(uri string) error {
	return m.PublishDiagnostics(uri, []protocol.Diagnostic{})
}
