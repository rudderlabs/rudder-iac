package backend

import (
	"fmt"
	"sync"
)

// WorkspaceID is the fixed workspace all backend resources belong to, so
// import metadata generated from this backend matches the workspace used by
// tests that apply imported specs.
const WorkspaceID = "test-workspace-id"

// Backend provides in-memory storage for the example provider
// It uses maps with remote IDs as keys to simulate a remote system
type Backend struct {
	mu              sync.RWMutex
	remoteIdCounter int
	books           map[string]*RemoteBook
	writers         map[string]*RemoteWriter
}

// RemoteBook represents a book in the remote system
type RemoteBook struct {
	ID         string
	ExternalID string
	Name       string
	AuthorID   string // Remote ID of the writer
	// AccessKey simulates a secret: the backend stores the real value, but —
	// like a real API — the provider never maps it back as a known value.
	AccessKey string
}

// RemoteWriter represents a writer in the remote system
type RemoteWriter struct {
	ID         string
	ExternalID string
	Name       string
}

// NewBackend creates a new backend instance
func NewBackend() *Backend {
	return &Backend{
		books:   make(map[string]*RemoteBook),
		writers: make(map[string]*RemoteWriter),
	}
}

// remoteID returns a consistent remote ID so that it can be predicted in tests
func (b *Backend) remoteID(resourceType string, externalID string) string {
	if externalID == "" {
		b.remoteIdCounter++
		return fmt.Sprintf("remote-%s-%d", resourceType, b.remoteIdCounter)
	}

	return fmt.Sprintf("remote-%s-%s", resourceType, externalID)
}

// Book operations
func (b *Backend) CreateBook(name, authorID, externalID, accessKey string) (*RemoteBook, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	id := b.remoteID("book", externalID)
	book := &RemoteBook{
		ID:         id,
		ExternalID: externalID,
		Name:       name,
		AuthorID:   authorID,
		AccessKey:  accessKey,
	}
	b.books[id] = book
	return book, nil
}

func (b *Backend) UpdateBook(id, name, authorID, accessKey string) (*RemoteBook, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	book, ok := b.books[id]
	if !ok {
		return nil, fmt.Errorf("book with ID %s not found", id)
	}
	book.Name = name
	book.AuthorID = authorID
	book.AccessKey = accessKey
	return book, nil
}

func (b *Backend) DeleteBook(id string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, ok := b.books[id]; !ok {
		return fmt.Errorf("book with ID %s not found", id)
	}
	delete(b.books, id)
	return nil
}

func (b *Backend) Book(id string) (*RemoteBook, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	book, ok := b.books[id]
	if !ok {
		return nil, fmt.Errorf("book with ID %s not found", id)
	}
	return book, nil
}

func (b *Backend) AllBooks() []*RemoteBook {
	b.mu.RLock()
	defer b.mu.RUnlock()

	result := make([]*RemoteBook, 0, len(b.books))
	for _, book := range b.books {
		result = append(result, book)
	}
	return result
}

func (b *Backend) SetBookExternalID(id, externalID string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	book, ok := b.books[id]
	if !ok {
		return fmt.Errorf("book with ID %s not found", id)
	}
	book.ExternalID = externalID
	return nil
}

// Writer operations
func (b *Backend) CreateWriter(name, externalID string) (*RemoteWriter, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	id := b.remoteID("writer", externalID)
	writer := &RemoteWriter{
		ID:         id,
		ExternalID: externalID,
		Name:       name,
	}
	b.writers[id] = writer
	return writer, nil
}

func (b *Backend) UpdateWriter(id, name string) (*RemoteWriter, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	writer, ok := b.writers[id]
	if !ok {
		return nil, fmt.Errorf("writer with ID %s not found", id)
	}
	writer.Name = name
	return writer, nil
}

func (b *Backend) DeleteWriter(id string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, ok := b.writers[id]; !ok {
		return fmt.Errorf("writer with ID %s not found", id)
	}
	delete(b.writers, id)
	return nil
}

func (b *Backend) Writer(id string) (*RemoteWriter, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	writer, ok := b.writers[id]
	if !ok {
		return nil, fmt.Errorf("writer with ID %s not found", id)
	}
	return writer, nil
}

func (b *Backend) AllWriters() []*RemoteWriter {
	b.mu.RLock()
	defer b.mu.RUnlock()

	result := make([]*RemoteWriter, 0, len(b.writers))
	for _, writer := range b.writers {
		result = append(result, writer)
	}
	return result
}

func (b *Backend) SetWriterExternalID(id, externalID string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	writer, ok := b.writers[id]
	if !ok {
		return fmt.Errorf("writer with ID %s not found", id)
	}
	writer.ExternalID = externalID
	return nil
}
