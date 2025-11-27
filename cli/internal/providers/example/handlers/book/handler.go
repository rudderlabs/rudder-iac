package book

import (
	"context"
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/writer"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/handler"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/example/backend"
	examplewriter "github.com/rudderlabs/rudder-iac/cli/internal/providers/example/handlers/writer"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

type BookHandler = handler.BaseHandler[BookSpec, BookResource, BookState, RemoteBook]

const (
	ResourceType = "example_book"
	SpecKind     = "books"
)

func ParseBookReference(ref string) (string, error) {
	specRef, err := specs.ParseSpecReference(ref, map[string]string{SpecKind: ResourceType})
	if err != nil {
		return "", err
	}
	return specRef.URN, nil
}

// HandlerImpl implements the HandlerImpl interface for book resources
type HandlerImpl struct {
	backend *backend.Backend
}

func NewHandlerImpl(backend *backend.Backend) *HandlerImpl {
	return &HandlerImpl{
		backend: backend,
	}
}

func (h *HandlerImpl) NewSpec() *BookSpec {
	return &BookSpec{}
}

func (h *HandlerImpl) ValidateSpec(spec *BookSpec) error {
	if len(spec.Books) == 0 {
		return fmt.Errorf("at least one book is required")
	}
	for i, book := range spec.Books {
		if book.ID == "" {
			return fmt.Errorf("book[%d]: id is required", i)
		}
		if book.Name == "" {
			return fmt.Errorf("book[%d]: name is required", i)
		}
		if book.Author == "" {
			return fmt.Errorf("book[%d]: author is required", i)
		}
	}
	return nil
}

func (h *HandlerImpl) ExtractResourcesFromSpec(path string, spec *BookSpec) (map[string]*BookResource, error) {
	res := make(map[string]*BookResource)
	for _, bookItem := range spec.Books {
		// Parse the author reference string into a URN
		authorURN, err := examplewriter.ParseWriterReference(bookItem.Author)
		if err != nil {
			return nil, fmt.Errorf("parsing author reference for book %s: %w", bookItem.ID, err)
		}

		resource := &BookResource{
			ID:     bookItem.ID,
			Name:   bookItem.Name,
			Author: examplewriter.CreateWriterReference(authorURN),
		}
		res[bookItem.ID] = resource
	}
	return res, nil
}

func (h *HandlerImpl) ValidateResource(resource *BookResource, graph *resources.Graph) error {
	if resource.Name == "" {
		return fmt.Errorf("name is required")
	}
	if resource.Author == nil {
		return fmt.Errorf("author is required")
	}

	// Validate that the author URN exists in the graph
	if _, exists := graph.GetResource(resource.Author.URN); !exists {
		return fmt.Errorf("author URN %s does not exist", resource.Author.URN)
	}

	return nil
}

func (h *HandlerImpl) LoadRemoteResources(ctx context.Context) ([]*RemoteBook, error) {
	remoteBooks := h.backend.GetAllBooks()

	// Filter only managed resources (those with external IDs)
	result := make([]*RemoteBook, 0)
	for _, b := range remoteBooks {
		if b.ExternalID != "" {
			result = append(result, &RemoteBook{RemoteBook: b})
		}
	}
	return result, nil
}

func (h *HandlerImpl) LoadImportableResources(ctx context.Context) ([]*RemoteBook, error) {
	remoteBooks := h.backend.GetAllBooks()

	// Return all resources for import
	result := make([]*RemoteBook, 0, len(remoteBooks))
	for _, b := range remoteBooks {
		result = append(result, &RemoteBook{RemoteBook: b})
	}
	return result, nil
}

func (h *HandlerImpl) MapRemoteToState(remote *RemoteBook, urnResolver handler.URNResolver) (*BookResource, *BookState, error) {
	// Resolve the author remote ID to URN
	authorURN, err := urnResolver.GetURNByID(examplewriter.ResourceType, remote.AuthorID)
	if err != nil {
		return nil, nil, fmt.Errorf("resolving author URN for book %s: %w", remote.ID, err)
	}

	resource := &BookResource{
		ID:     remote.ExternalID,
		Name:   remote.Name,
		Author: examplewriter.CreateWriterReference(authorURN),
	}

	state := &BookState{
		ID: remote.ID,
	}

	return resource, state, nil
}

func (h *HandlerImpl) Create(ctx context.Context, data *BookResource) (*BookState, error) {
	// In a real implementation, we would need to resolve the author URN to a remote ID
	// For this example, we'll store the URN directly
	// A more complete implementation would use a PropertyRef resolver
	remoteBook, err := h.backend.CreateBook(data.Name, data.Author.Value, data.ID)
	if err != nil {
		return nil, fmt.Errorf("creating book in backend: %w", err)
	}

	return &BookState{
		ID: remoteBook.ID,
	}, nil
}

func (h *HandlerImpl) Update(ctx context.Context, newData *BookResource, oldData *BookResource, oldState *BookState) (*BookState, error) {
	remoteBook, err := h.backend.UpdateBook(oldState.ID, newData.Name, newData.Author.Value)
	if err != nil {
		return nil, fmt.Errorf("updating book in backend: %w", err)
	}

	return &BookState{
		ID: remoteBook.ID,
	}, nil
}

func (h *HandlerImpl) Import(ctx context.Context, data *BookResource, remoteId string) (*BookState, error) {
	// Set external ID on the remote resource
	if err := h.backend.SetBookExternalID(remoteId, ""); err != nil {
		return nil, fmt.Errorf("setting external ID on book: %w", err)
	}

	remoteBook, err := h.backend.GetBook(remoteId)
	if err != nil {
		return nil, fmt.Errorf("getting book from backend: %w", err)
	}

	return &BookState{
		ID: remoteBook.ID,
	}, nil
}

func (h *HandlerImpl) Delete(ctx context.Context, id string, oldData *BookResource, oldState *BookState) error {
	return h.backend.DeleteBook(oldState.ID)
}

func (h *HandlerImpl) FormatForExport(
	ctx context.Context,
	collection *resources.ResourceCollection,
	idNamer namer.Namer,
	inputResolver resolver.ReferenceResolver,
) ([]writer.FormattableEntity, error) {
	// Example provider doesn't support export
	return nil, nil
}

// NewHandler creates a new BaseHandler for book resources
func NewHandler(backend *backend.Backend) *BookHandler {
	return handler.NewHandler(
		SpecKind,
		ResourceType,
		"books", // import metadata name
		NewHandlerImpl(backend),
	)
}
