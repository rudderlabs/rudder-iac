// Package document provides document state management for the LSP server
package document

import (
	"fmt"
	"os"
	"sync"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

// Document represents a text document in the LSP server
type Document struct {
	URI     string
	Content []byte
	Version int
}

// DocumentStore manages open documents in memory
type DocumentStore struct {
	mu   sync.RWMutex
	docs map[string]*Document
}

// NewDocumentStore creates a new document store
func NewDocumentStore() *DocumentStore {
	return &DocumentStore{
		docs: make(map[string]*Document),
	}
}

// Open adds a new document to the store or updates an existing one
func (ds *DocumentStore) Open(uri string, content []byte, version int) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	doc := &Document{
		URI:     uri,
		Content: make([]byte, len(content)),
		Version: version,
	}
	copy(doc.Content, content)
	ds.docs[uri] = doc
}

// Update modifies an existing document's content
func (ds *DocumentStore) Update(uri string, content []byte, version int) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	if doc, exists := ds.docs[uri]; exists {
		doc.Content = make([]byte, len(content))
		copy(doc.Content, content)
		doc.Version = version
	}
}

// Close removes a document from the store
func (ds *DocumentStore) Close(uri string) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	delete(ds.docs, uri)
}

// Get retrieves a document by URI
func (ds *DocumentStore) Get(uri string) (*Document, bool) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	doc, exists := ds.docs[uri]
	return doc, exists
}

// GetContent returns the content of a document by URI
func (ds *DocumentStore) GetContent(uri string) ([]byte, bool) {
	doc, exists := ds.Get(uri)
	if !exists {
		return nil, false
	}
	return doc.Content, true
}

// GetAllURIs returns all currently open document URIs
func (ds *DocumentStore) GetAllURIs() []string {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	uris := make([]string, 0, len(ds.docs))
	for uri := range ds.docs {
		uris = append(uris, uri)
	}
	return uris
}

// UpdateFromDidOpen updates the document store from a didOpen notification
func (ds *DocumentStore) UpdateFromDidOpen(params protocol.DidOpenTextDocumentParams) {
	content := []byte(params.TextDocument.Text)
	ds.Open(params.TextDocument.URI, content, int(params.TextDocument.Version))
}

// UpdateFromDidChange updates the document store from a didChange notification
func (ds *DocumentStore) UpdateFromDidChange(params protocol.DidChangeTextDocumentParams) {
	// For this minimal implementation, we assume full content sync (change = 1)
	// In a full implementation, we'd handle incremental changes
	if len(params.ContentChanges) > 0 {
		change := params.ContentChanges[0]

		// Try TextDocumentContentChangeEventWhole (full document sync)
		if tce, ok := change.(protocol.TextDocumentContentChangeEventWhole); ok {
			content := []byte(tce.Text)
			ds.Update(params.TextDocument.URI, content, int(params.TextDocument.Version))
			return
		}

		// Try TextDocumentContentChangeEvent (incremental sync)
		if tce, ok := change.(protocol.TextDocumentContentChangeEvent); ok {
			content := []byte(tce.Text)
			ds.Update(params.TextDocument.URI, content, int(params.TextDocument.Version))
			return
		}

		// Try as a map (some clients send it this way)
		if m, ok := change.(map[string]any); ok {
			if text, ok := m["text"].(string); ok {
				content := []byte(text)
				ds.Update(params.TextDocument.URI, content, int(params.TextDocument.Version))
				return
			}
		}

		// Log unknown type for debugging
		fmt.Fprintf(os.Stderr, "[document] UpdateFromDidChange: unknown change type %T\n", change)
	}
}

// UpdateFromDidSave updates the document store from a didSave notification
func (ds *DocumentStore) UpdateFromDidSave(params protocol.DidSaveTextDocumentParams) {
	if params.Text != nil {
		content := []byte(*params.Text)
		ds.Update(params.TextDocument.URI, content, 0) // Version not provided in didSave
	}
}

// UpdateFromDidClose removes the document from the store
func (ds *DocumentStore) UpdateFromDidClose(params protocol.DidCloseTextDocumentParams) {
	ds.Close(params.TextDocument.URI)
}
