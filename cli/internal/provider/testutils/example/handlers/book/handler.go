package book

import (
	"context"
	"fmt"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/handler"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/handler/export"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/testutils/example/backend"
	examplewriter "github.com/rudderlabs/rudder-iac/cli/internal/provider/testutils/example/handlers/writer"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/testutils/example/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/secret"
)

type BookHandler = handler.BaseHandler[model.BookSpec, model.BookResource, model.BookState, model.RemoteBook]

var HandlerMetadata = handler.HandlerMetadata{
	ResourceType:     "example-book",
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

func (h *HandlerImpl) ExtractResourcesFromSpec(path string, spec *model.BookSpec) (map[string]*model.BookResource, error) {
	res := make(map[string]*model.BookResource)
	for _, bookItem := range spec.Books {
		// Parse the author reference string into a URN
		authorURN, err := examplewriter.ParseWriterReference(bookItem.Author)
		if err != nil {
			return nil, fmt.Errorf("parsing author reference for book %s: %w", bookItem.ID, err)
		}

		resource := &model.BookResource{
			ID:        bookItem.ID,
			Name:      bookItem.Name,
			Author:    examplewriter.CreateWriterReference(authorURN),
			AccessKey: &bookItem.AccessKey,
		}
		res[bookItem.ID] = resource
	}
	return res, nil
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

	unknownKey := secret.NewUnknown()
	resource := &model.BookResource{
		ID:     remote.ExternalID,
		Name:   remote.Name,
		Author: examplewriter.CreateWriterReference(authorURN),
		// The backend, like a real API, never returns the secret's value.
		AccessKey: &unknownKey,
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
	remoteBook, err := h.backend.CreateBook(data.Name, data.Author.Value, data.ID, revealAccessKey(data))
	if err != nil {
		return nil, fmt.Errorf("creating book in backend: %w", err)
	}

	return &model.BookState{
		ID: remoteBook.ID,
	}, nil
}

func (h *HandlerImpl) Update(ctx context.Context, newData *model.BookResource, oldData *model.BookResource, oldState *model.BookState) (*model.BookState, error) {
	// Verify both new and old PropertyRefs were dereferenced
	// Both should be dereferenced: handlers often need to compare old vs new values
	if newData.Author == nil || newData.Author.Value == "" {
		return nil, fmt.Errorf("cannot update book: new author reference not dereferenced (PropertyRef.Value is empty)")
	}
	if oldData.Author == nil || oldData.Author.Value == "" {
		return nil, fmt.Errorf("cannot update book: old author reference not dereferenced (PropertyRef.Value is empty)")
	}

	remoteBook, err := h.backend.UpdateBook(oldState.ID, newData.Name, newData.Author.Value, revealAccessKey(newData))
	if err != nil {
		return nil, fmt.Errorf("updating book in backend: %w", err)
	}

	return &model.BookState{
		ID: remoteBook.ID,
	}, nil
}

func (h *HandlerImpl) Import(ctx context.Context, data *model.BookResource, remoteId string) (*model.BookState, error) {
	// Set external ID on the remote resource, marking it managed so the next
	// apply diffs it instead of importing it again
	if err := h.backend.SetBookExternalID(remoteId, data.ID); err != nil {
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
			// Unknown on purpose: export can never recover the real value. The
			// variable name is derived from the resource's identity so it stays
			// stable across re-imports; kebab-case IDs are folded to the
			// substitutor's variable grammar.
			AccessKey: secret.NewUnknown(secret.WithVariableName(accessKeyVarName(externalID))),
		})
	}

	return &export.SpecExportData[model.BookSpec]{
		RelativePath: "books/books.yaml",
		Data:         &model.BookSpec{Books: books},
	}, nil
}

func (h *HandlerImpl) Delete(ctx context.Context, id string, oldData *model.BookResource, oldState *model.BookState) error {
	// Verify PropertyRef was dereferenced (demonstrates PRO-5272 fix)
	if oldData.Author == nil || oldData.Author.Value == "" {
		return fmt.Errorf("cannot delete book: author reference not dereferenced (PropertyRef.Value is empty)")
	}
	return h.backend.DeleteBook(oldState.ID)
}

func accessKeyVarName(externalID string) string {
	return fmt.Sprintf("BOOK_%s_ACCESS_KEY", strings.ToUpper(strings.ReplaceAll(externalID, "-", "_")))
}

// revealAccessKey is the single point where the real secret escapes toward the
// backend API; a spec without an access key sends the empty string.
func revealAccessKey(data *model.BookResource) string {
	if data.AccessKey == nil {
		return ""
	}
	return data.AccessKey.Reveal()
}

func ParseBookReference(ref string) (string, error) {
	specRef, err := specs.ParseSpecReference(ref, map[string]string{HandlerMetadata.SpecKind: HandlerMetadata.ResourceType})
	if err != nil {
		return "", err
	}
	return specRef.URN, nil
}
