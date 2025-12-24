// Package server provides the LSP server implementation using GLSP SDK
package server

import (
	"context"

	"github.com/tliron/commonlog"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"github.com/tliron/glsp/server"

	"github.com/rudderlabs/rudder-iac/cli/internal/lsp/diagnostic"
	"github.com/rudderlabs/rudder-iac/cli/internal/lsp/document"
)

// RudderLSPServer represents the LSP server for Rudder CLI YAML files
type RudderLSPServer struct {
	documentStore *document.DocumentStore
	diagnosticMgr *diagnostic.Manager
	handler       protocol.Handler
	server        *server.Server
	workspaceRoot string
}

// NewRudderLSPServer creates a new LSP server instance
func NewRudderLSPServer() *RudderLSPServer {
	documentStore := document.NewDocumentStore()
	diagnosticMgr := diagnostic.NewManager()

	rudderServer := &RudderLSPServer{
		documentStore: documentStore,
		diagnosticMgr: diagnosticMgr,
	}

	// Initialize handler directly in the struct to avoid copying mutex
	rudderServer.handler = protocol.Handler{
		Initialize:            initialize,
		Initialized:           initialized,
		Shutdown:              shutdown,
		TextDocumentDidOpen:   handleTextDocumentDidOpen,
		TextDocumentDidChange: handleTextDocumentDidChange,
		TextDocumentDidSave:   handleTextDocumentDidSave,
		TextDocumentDidClose:  handleTextDocumentDidClose,
	}

	rudderServer.server = server.NewServer(&rudderServer.handler, "rudder-lsp", false)

	// Store server reference for use in handlers
	globalServer = rudderServer

	return rudderServer
}

// RunStdio starts the LSP server using stdio communication
func (s *RudderLSPServer) RunStdio(ctx context.Context) error {
	commonlog.Configure(1, nil)
	return s.server.RunStdio()
}

// Global server reference for handlers (not ideal but necessary for GLSP pattern)
var globalServer *RudderLSPServer

// initialize handles the initialize request
func initialize(context *glsp.Context, params *protocol.InitializeParams) (any, error) {
	// Set the GLSP context for diagnostic publishing
	globalServer.diagnosticMgr.SetContext(context)
	capabilities := globalServer.handler.CreateServerCapabilities()

	if params.RootURI != nil {
		globalServer.workspaceRoot = *params.RootURI
	}

	// Configure text document synchronization
	openClose := true
	changeKind := protocol.TextDocumentSyncKindFull
	includeText := true
	version := "0.1.0"
	name := "rudder-lsp"

	capabilities.TextDocumentSync = &protocol.TextDocumentSyncOptions{
		OpenClose: &openClose,
		Change:    &changeKind,
		Save: &protocol.SaveOptions{
			IncludeText: &includeText,
		},
	}

	return protocol.InitializeResult{
		Capabilities: capabilities,
		ServerInfo: &protocol.InitializeResultServerInfo{
			Name:    name,
			Version: &version,
		},
	}, nil
}

// initialized handles the initialized notification
func initialized(context *glsp.Context, params *protocol.InitializedParams) error {
	// Server is now initialized and ready to handle requests
	return nil
}

// shutdown handles the shutdown request
func shutdown(context *glsp.Context) error {
	// Clean up resources
	return nil
}

// handleTextDocumentDidOpen handles textDocument/didOpen notifications
func handleTextDocumentDidOpen(context *glsp.Context, params *protocol.DidOpenTextDocumentParams) error {
	// Store the document
	globalServer.documentStore.UpdateFromDidOpen(*params)
	globalServer.validateDocument(params.TextDocument.URI)

	return nil
}

// handleTextDocumentDidChange handles textDocument/didChange notifications
func handleTextDocumentDidChange(context *glsp.Context, params *protocol.DidChangeTextDocumentParams) error {
	// Update the document
	globalServer.documentStore.UpdateFromDidChange(*params)

	// Validate the document
	globalServer.validateDocument(params.TextDocument.URI)

	return nil
}

// handleTextDocumentDidSave handles textDocument/didSave notifications
func handleTextDocumentDidSave(context *glsp.Context, params *protocol.DidSaveTextDocumentParams) error {
	// Update the document
	globalServer.documentStore.UpdateFromDidSave(*params)

	// Validate the document
	globalServer.validateDocument(params.TextDocument.URI)

	return nil
}

// handleTextDocumentDidClose handles textDocument/didClose notifications
func handleTextDocumentDidClose(context *glsp.Context, params *protocol.DidCloseTextDocumentParams) error {
	// Remove the document from store
	globalServer.documentStore.UpdateFromDidClose(*params)

	// Clear diagnostics for this document
	globalServer.diagnosticMgr.ClearDiagnostics(params.TextDocument.URI)

	return nil
}

// validateDocument validates a single document and publishes diagnostics
func (s *RudderLSPServer) validateDocument(uri string) {
	content, exists := s.documentStore.GetContent(uri)
	if !exists {
		return
	}

	var diagnostics []protocol.Diagnostic

	// Check if this is a rudder file
	if !document.IsRudderFile(content) {
		// Not a rudder file - publish a diagnostic
		severity := protocol.DiagnosticSeverityError
		source := "rudder-lsp"
		diagnostic := protocol.Diagnostic{
			Range: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 0},
			},
			Severity: &severity,
			Message:  "This does not appear to be a valid Rudder CLI YAML file. Expected 'version: rudder/v0.1' at the top.",
			Source:   &source,
		}
		diagnostics = append(diagnostics, diagnostic)
	}

	// Publish diagnostics
	s.diagnosticMgr.PublishDiagnostics(uri, diagnostics)
}

func (s *RudderLSPServer) getWorkspaceRoot() string {
	if s.workspaceRoot == "" {
		return "."
	}
	return s.workspaceRoot
}

func (s *RudderLSPServer) validateDocumentUpdated(uri string) {
	content, exists := s.documentStore.GetContent(uri)
	if !exists {
		return
	}

	// Only validate if the file is a rudder file
	if !document.IsRudderFile(content) {
		return
	}

}
