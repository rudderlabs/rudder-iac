// Package server provides the LSP server implementation using GLSP SDK
package server

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/tliron/commonlog"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"github.com/tliron/glsp/server"

	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/lsp/completion"
	"github.com/rudderlabs/rudder-iac/cli/internal/lsp/converter"
	"github.com/rudderlabs/rudder-iac/cli/internal/lsp/diagnostic"
	"github.com/rudderlabs/rudder-iac/cli/internal/lsp/document"
	"github.com/rudderlabs/rudder-iac/cli/internal/lsp/utils"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/engine"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/registry"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules/datacatalog"
)

// RudderLSPServer represents the LSP server for Rudder CLI YAML files
type RudderLSPServer struct {
	documentStore *document.DocumentStore
	diagnosticMgr *diagnostic.Manager
	handler       protocol.Handler
	server        *server.Server
	workspaceRoot string

	// Validation engine fields
	validationEngine *engine.Engine
	engineMu         sync.RWMutex // Protects engine access
	engineError      error        // Tracks initialization errors

	// Completion provider
	completionProvider *completion.CompletionProvider
}

// NewRudderLSPServer creates a new LSP server instance
func NewRudderLSPServer() *RudderLSPServer {
	documentStore := document.NewDocumentStore()
	diagnosticMgr := diagnostic.NewManager()

	rudderServer := &RudderLSPServer{
		documentStore:      documentStore,
		diagnosticMgr:      diagnosticMgr,
		completionProvider: completion.NewProvider(),
	}

	// Initialize handler directly in the struct to avoid copying mutex
	rudderServer.handler = protocol.Handler{
		Initialize:             initialize,
		Initialized:            initialized,
		Shutdown:               shutdown,
		TextDocumentDidOpen:    handleTextDocumentDidOpen,
		TextDocumentDidChange:  handleTextDocumentDidChange,
		TextDocumentDidSave:    handleTextDocumentDidSave,
		TextDocumentDidClose:   handleTextDocumentDidClose,
		TextDocumentCompletion: handleTextDocumentCompletion,
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

	// Configure completion capabilities
	capabilities.CompletionProvider = &protocol.CompletionOptions{
		TriggerCharacters: []string{"#", "/"},
		ResolveProvider:   &protocol.False,
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

	// Initialize validation engine now that workspace is ready
	if globalServer.workspaceRoot != "" {
		if err := globalServer.initializeValidationEngine(); err != nil {
			// Log warning but don't fail - LSP can still provide basic features
			context.Notify(protocol.ServerWindowLogMessage, &protocol.LogMessageParams{
				Type:    protocol.MessageTypeWarning,
				Message: fmt.Sprintf("Validation engine initialization failed: %v. Authentication may be required.", err),
			})
		} else {
			// Success notification
			context.Notify(protocol.ServerWindowLogMessage, &protocol.LogMessageParams{
				Type:    protocol.MessageTypeInfo,
				Message: "Rudder validation engine initialized successfully",
			})
		}
	}

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
	// Log when didChange fires (use stderr, not stdout - stdout is for LSP protocol)
	fmt.Fprintf(os.Stderr, "[didChange] URI=%s, version=%d, changes=%d\n",
		params.TextDocument.URI, params.TextDocument.Version, len(params.ContentChanges))

	// Update the document
	globalServer.documentStore.UpdateFromDidChange(*params)

	// Skip validation on change - only validate on save
	// This is because the engine reads from disk, and we validate on save events

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

// handleTextDocumentCompletion handles textDocument/completion requests
func handleTextDocumentCompletion(
	context *glsp.Context,
	params *protocol.CompletionParams,
) (any, error) {

	// Get document content
	content, exists := globalServer.documentStore.GetContent(params.TextDocument.URI)
	if !exists {
		return []protocol.CompletionItem{}, nil
	}

	// Check if engine is available
	globalServer.engineMu.RLock()
	eng := globalServer.validationEngine
	globalServer.engineMu.RUnlock()

	if eng == nil {
		return []protocol.CompletionItem{}, nil
	}

	// Get resource graph
	graph := eng.ResourceGraph()
	if graph == nil {
		return []protocol.CompletionItem{}, nil
	}

	// Get completions
	items, err := globalServer.completionProvider.GetCompletions(
		content,
		int(params.Position.Line),
		int(params.Position.Character),
		graph,
	)

	if err != nil {
		context.Notify(protocol.ServerWindowLogMessage, &protocol.LogMessageParams{
			Type:    protocol.MessageTypeWarning,
			Message: fmt.Sprintf("Completion error: %v", err),
		})
		return []protocol.CompletionItem{}, nil
	}

	return items, nil
}

// validateDocument validates a single document and publishes diagnostics
func (s *RudderLSPServer) validateDocument(uri string) {
	content, exists := s.documentStore.GetContent(uri)
	if !exists {
		return
	}

	// Quick check: is this a rudder file?
	if !document.IsRudderFile(content) {
		// Not a Rudder file (e.g., docker-compose.yaml, GitHub workflows)
		// Just clear diagnostics and return silently - don't show error
		s.diagnosticMgr.ClearDiagnostics(uri)
		return
	}

	// Check if engine is available
	s.engineMu.RLock()
	eng := s.validationEngine
	s.engineMu.RUnlock()

	if eng == nil {
		// Engine not initialized - clear any previous diagnostics
		s.diagnosticMgr.ClearDiagnostics(uri)
		return
	}

	// Convert URI to path
	filePath, err := utils.URIToPath(uri)
	if err != nil {
		s.diagnosticMgr.ClearDiagnostics(uri)
		return
	}

	protocolDiagnostics := converter.EngineDiagnosticsToProtocol(
		eng.ValidateFile(filePath),
	)
	s.diagnosticMgr.PublishDiagnostics(uri, protocolDiagnostics)
}

// initializeValidationEngine creates and initializes the validation engine
func (s *RudderLSPServer) initializeValidationEngine() error {
	s.engineMu.Lock()
	defer s.engineMu.Unlock()

	// Already initialized
	if s.validationEngine != nil {
		return nil
	}

	// Convert workspace URI to path
	workspacePath, err := utils.URIToPath(s.workspaceRoot)
	if err != nil {
		s.engineError = fmt.Errorf("invalid workspace URI: %w", err)
		return s.engineError
	}

	// Initialize app dependencies (requires auth)
	deps, err := app.NewDeps()
	if err != nil {
		s.engineError = fmt.Errorf("failed to initialize dependencies (auth required): %w", err)
		return s.engineError
	}

	// Create rule registry
	reg := registry.NewRegistry()

	// Register validation rules
	if err := reg.Register(&datacatalog.RequiredFieldsRule{}); err != nil {
		s.engineError = fmt.Errorf("failed to register rules: %w", err)
		return s.engineError
	}
	if err := reg.Register(&datacatalog.CustomTypeNameRule{}); err != nil {
		s.engineError = fmt.Errorf("failed to register rules: %w", err)
		return s.engineError
	}
	if err := reg.Register(&datacatalog.CategoryNameRule{}); err != nil {
		s.engineError = fmt.Errorf("failed to register rules: %w", err)
		return s.engineError
	}

	// Get composite provider
	provider := deps.CompositeProvider()

	// Create validation engine
	eng, err := engine.NewEngine(workspacePath, reg, provider)
	if err != nil {
		s.engineError = fmt.Errorf("failed to create validation engine: %w", err)
		return s.engineError
	}

	s.validationEngine = eng

	// Build completion cache
	if eng.ResourceGraph() != nil {
		s.completionProvider.RebuildCache(eng.ResourceGraph())
	}

	return nil
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
