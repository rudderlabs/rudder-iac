package book

import (
	"context"
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/handler"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/handler/export"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/testutils/example/backend"
	examplewriter "github.com/rudderlabs/rudder-iac/cli/internal/provider/testutils/example/handlers/writer"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/testutils/example/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

type BookHandler = handler.BaseHandler[model.BookSpec, model.BookResource, model.BookState, model.RemoteBook]

var HandlerMetadata = handler.HandlerMetadata{
	ResourceType:     "example_book",
	SpecKind:         "books",
	SpecMetadataName: "books",
}

type HandlerImpl struct {
	*export.SingleSpecExportStrategy[model.BookSpec, model.RemoteBook]
	backend *backend.Backend
}

// NewHandler creates a new BaseHandler for book resources
func NewHandler(backend *backend.Backend) *BookHandler {
	h := &HandlerImpl{backend: backend}
	h.SingleSpecExportStrategy = &export.SingleSpecExportStrategy[model.BookSpec, model.RemoteBook]{Handler: h}
	return handler.NewHandler(h)
}

func (h *HandlerImpl) Metadata() handler.HandlerMetadata {
	return HandlerMetadata
}

func (h *HandlerImpl) NewSpec() *model.BookSpec {
	return &model.BookSpec{}
}

func (h *HandlerImpl) ValidateSpec(spec *model.BookSpec) error {
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

func (h *HandlerImpl) ExtractResourcesFromSpec(path string, spec *model.BookSpec) (map[string]*model.BookResource, error) {
	res := make(map[string]*model.BookResource)
	for _, bookItem := range spec.Books {
		// Parse the author reference string into a URN
		authorURN, err := examplewriter.ParseWriterReference(bookItem.Author)
		if err != nil {
			return nil, fmt.Errorf("parsing author reference for book %s: %w", bookItem.ID, err)
		}

		resource := &model.BookResource{
			ID:     bookItem.ID,
			Name:   bookItem.Name,
			Author: examplewriter.CreateWriterReference(authorURN),
		}
		res[bookItem.ID] = resource
	}
	return res, nil
}

func (h *HandlerImpl) ValidateResource(resource *model.BookResource, graph *resources.Graph) error {
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

func (h *HandlerImpl) LoadRemoteResources(ctx context.Context) ([]*model.RemoteBook, error) {
	remoteBooks := h.backend.AllBooks()

	// Filter only managed resources (those with external IDs)
	result := make([]*model.RemoteBook, 0)
	for _, b := range remoteBooks {
		if b.ExternalID != "" {
			result = append(result, &model.RemoteBook{RemoteBook: b})
		}
	}
	return result, nil
}

func (h *HandlerImpl) LoadImportableResources(ctx context.Context) ([]*model.RemoteBook, error) {
	remoteBooks := h.backend.AllBooks()

	// Return all resources for import
	result := make([]*model.RemoteBook, 0, len(remoteBooks))
	for _, b := range remoteBooks {
		result = append(result, &model.RemoteBook{RemoteBook: b})
	}
	return result, nil
}

func (h *HandlerImpl) MapRemoteToState(remote *model.RemoteBook, urnResolver handler.URNResolver) (*model.BookResource, *model.BookState, error) {
	// Resolve the author remote ID to URN
	authorURN, err := urnResolver.GetURNByID(examplewriter.HandlerMetadata.ResourceType, remote.AuthorID)
	if err != nil {
		return nil, nil, fmt.Errorf("resolving author URN for book %s: %w", remote.ID, err)
	}

	resource := &model.BookResource{
		ID:     remote.ExternalID,
		Name:   remote.Name,
		Author: examplewriter.CreateWriterReference(authorURN),
	}

	state := &model.BookState{
		ID: remote.ID,
	}

	return resource, state, nil
}

func (h *HandlerImpl) Create(ctx context.Context, data *model.BookResource) (*model.BookState, error) {
	// In a real implementation, we would need to resolve the author URN to a remote ID
	// For this example, we'll store the URN directly
	// A more complete implementation would use a PropertyRef resolver
	remoteBook, err := h.backend.CreateBook(data.Name, data.Author.Value, data.ID)
	if err != nil {
		return nil, fmt.Errorf("creating book in backend: %w", err)
	}

	return &model.BookState{
		ID: remoteBook.ID,
	}, nil
}

func (h *HandlerImpl) Update(ctx context.Context, newData *model.BookResource, oldData *model.BookResource, oldState *model.BookState) (*model.BookState, error) {
	remoteBook, err := h.backend.UpdateBook(oldState.ID, newData.Name, newData.Author.Value)
	if err != nil {
		return nil, fmt.Errorf("updating book in backend: %w", err)
	}

	return &model.BookState{
		ID: remoteBook.ID,
	}, nil
}

func (h *HandlerImpl) Import(ctx context.Context, data *model.BookResource, remoteId string) (*model.BookState, error) {
	// Set external ID on the remote resource
	if err := h.backend.SetBookExternalID(remoteId, ""); err != nil {
		return nil, fmt.Errorf("setting external ID on book: %w", err)
	}

	remoteBook, err := h.backend.Book(remoteId)
	if err != nil {
		return nil, fmt.Errorf("getting book from backend: %w", err)
	}

	return &model.BookState{
		ID: remoteBook.ID,
	}, nil
}

func (h *HandlerImpl) MapRemoteToSpec(data map[string]*model.RemoteBook, inputResolver resolver.ReferenceResolver) (*export.SpecExportData[model.BookSpec], error) {
	books := make([]model.BookItem, 0, len(data))

	for externalID, res := range data {
		// Resolve the author reference
		authorRef, err := inputResolver.ResolveToReference(examplewriter.HandlerMetadata.ResourceType, res.AuthorID)
		if err != nil {
			return nil, fmt.Errorf("resolving author reference for book %s: %w", res.ExternalID, err)
		}

		// Create book item for the spec
		books = append(books, model.BookItem{
			ID:     externalID,
			Name:   res.Name,
			Author: authorRef,
		})
	}

	return &export.SpecExportData[model.BookSpec]{
		RelativePath: "books/books.yaml",
		Data:         &model.BookSpec{Books: books},
	}, nil
}

func (h *HandlerImpl) Delete(ctx context.Context, id string, oldData *model.BookResource, oldState *model.BookState) error {
	return h.backend.DeleteBook(oldState.ID)
}

func ParseBookReference(ref string) (string, error) {
	specRef, err := specs.ParseSpecReference(ref, map[string]string{HandlerMetadata.SpecKind: HandlerMetadata.ResourceType})
	if err != nil {
		return "", err
	}
	return specRef.URN, nil
}
