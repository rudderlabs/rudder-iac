package transformations

import (
	"context"
	"fmt"
	"sync"

	"github.com/rudderlabs/rudder-iac/api/client"
	transformations "github.com/rudderlabs/rudder-iac/api/client/transformations"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/handlers/library"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/handlers/transformation"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/parser"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
)

var log = logger.New("transformationsprovider")

// pendingDeletes collects delete intents for deferred execution.
// Deletes are deferred because the backend enforces referential integrity
// against published state — a library cannot be deleted while a published
// transformation still imports it. By deferring deletes until after batch
// publish in ConsolidateSync, the published dependency graph is updated first.
type pendingDeletes struct {
	mu              sync.RWMutex
	transformations []string // remote IDs
	libraries       []string // remote IDs
}

// Provider wraps BaseProvider and adds transformations-specific functionality
type Provider struct {
	*provider.BaseProvider
	store          transformations.TransformationStore
	pendingDeletes pendingDeletes
}

// NewProvider creates a new transformations provider with all resource handlers
func NewProvider(client *client.Client) *Provider {
	// Create the transformation store
	store := transformations.NewRudderTransformationStore(client)
	return NewProviderWithStore(store)
}

// NewProviderWithStore creates a new transformations provider with a custom store
// This is useful for testing with mock stores
func NewProviderWithStore(store transformations.TransformationStore) *Provider {
	// Register handlers - libraries first (dependencies), then transformations
	handlers := []provider.Handler{
		library.NewHandler(store),
		transformation.NewHandler(store),
	}

	return &Provider{
		BaseProvider: provider.NewBaseProvider(handlers),
		store:        store,
	}
}

// MapRemoteToState overrides BaseProvider.MapRemoteToState to populate dependencies for transformations
// Dependencies are extracted from the remote transformation's Imports field and resolved to library URNs
func (p *Provider) MapRemoteToState(collection *resources.RemoteResources) (*state.State, error) {
	// Call base implementation to get initial state
	st, err := p.BaseProvider.MapRemoteToState(collection)
	if err != nil {
		return nil, err
	}

	// Build handleName -> library URN mapping from remote libraries
	libraryResources := collection.GetAll(library.HandlerMetadata.ResourceType)
	handleNameToURN := make(map[string]string)

	for _, remoteLib := range libraryResources {
		lib := remoteLib.Data.(*model.RemoteLibrary)
		libraryURN := resources.URN(lib.ExternalID, library.HandlerMetadata.ResourceType)
		handleNameToURN[lib.ImportName] = libraryURN
	}

	// Populate dependencies for each transformation using the Imports field
	transformationResources := collection.GetAll(transformation.HandlerMetadata.ResourceType)

	for _, remoteTrans := range transformationResources {
		trans := remoteTrans.Data.(*model.RemoteTransformation)
		transURN := resources.URN(trans.ExternalID, transformation.HandlerMetadata.ResourceType)
		resourceState := st.GetResource(transURN)

		// Resolve imports to library URNs
		dependencies := make([]string, 0, len(trans.Imports))
		for _, handleName := range trans.Imports {
			libraryURN, exists := handleNameToURN[handleName]
			if !exists {
				log.Warn("transformation references library not found in managed resources",
					"transformation", trans.ExternalID,
					"import_name", handleName)
				continue
			}
			dependencies = append(dependencies, libraryURN)
		}

		// Update resource state with dependencies
		resourceState.Dependencies = dependencies
	}

	return st, nil
}

// ResourceGraph overrides BaseProvider.ResourceGraph to add transformation→library dependencies
// by parsing transformation code to extract library imports
func (p *Provider) ResourceGraph() (*resources.Graph, error) {
	graph := resources.NewGraph()

	// libraries
	libraryHandler, ok := p.GetHandler(library.HandlerMetadata.ResourceType)
	if !ok {
		return nil, fmt.Errorf("library handler not found")
	}

	libraryResources, err := libraryHandler.Resources()
	if err != nil {
		return nil, fmt.Errorf("getting library resources: %w", err)
	}

	// Build handleName → URN mapping
	handleNameToURN := make(map[string]string)

	for _, r := range libraryResources {
		lib, ok := r.RawData().(*model.LibraryResource)
		if !ok {
			return nil, fmt.Errorf("invalid RawData type for library %s: expected *model.LibraryResource", r.URN())
		}

		handleNameToURN[lib.ImportName] = r.URN()
		graph.AddResource(r)
	}

	// transformations
	transformationHandler, ok := p.GetHandler(transformation.HandlerMetadata.ResourceType)
	if !ok {
		return nil, fmt.Errorf("transformation handler not found")
	}

	transformationResources, err := transformationHandler.Resources()
	if err != nil {
		return nil, fmt.Errorf("getting transformation resources: %w", err)
	}

	for _, r := range transformationResources {
		tf, ok := r.RawData().(*model.TransformationResource)
		if !ok {
			return nil, fmt.Errorf("invalid RawData type for transformation %s: expected *model.TransformationResource", r.URN())
		}

		codeParser, err := parser.NewParser(tf.Language)
		if err != nil {
			return nil, fmt.Errorf("creating parser for %s: %w", r.URN(), err)
		}

		handleNames, err := codeParser.ExtractImports(tf.Code)
		if err != nil {
			return nil, fmt.Errorf("parsing imports for %s: %w", r.URN(), err)
		}

		// Resolve handleNames to library URNs and add dependencies
		for _, handleName := range handleNames {
			libraryURN, exists := handleNameToURN[handleName]
			if !exists {
				return nil, fmt.Errorf(
					"transformation %s importing library with import_name '%s' not found",
					r.ID(),
					handleName,
				)
			}

			graph.AddDependency(r.URN(), libraryURN)
		}
		graph.AddResource(r)
	}

	return graph, nil
}

// DeleteRaw overrides BaseProvider.DeleteRaw to defer deletion until ConsolidateSync.
// Instead of immediately deleting resources, it records the delete intent so that
// batch publish can update the published dependency graph first.
func (p *Provider) DeleteRaw(_ context.Context, _ string, resourceType string, _ any, oldState any) error {
	switch resourceType {
	case transformation.HandlerMetadata.ResourceType:
		st, ok := oldState.(*model.TransformationState)
		if !ok {
			return fmt.Errorf("invalid state type for transformation delete")
		}
		p.pendingDeletes.mu.Lock()
		p.pendingDeletes.transformations = append(p.pendingDeletes.transformations, st.ID)
		p.pendingDeletes.mu.Unlock()

	case library.HandlerMetadata.ResourceType:
		st, ok := oldState.(*model.LibraryState)
		if !ok {
			return fmt.Errorf("invalid state type for library delete")
		}
		p.pendingDeletes.mu.Lock()
		p.pendingDeletes.libraries = append(p.pendingDeletes.libraries, st.ID)
		p.pendingDeletes.mu.Unlock()

	default:
		return fmt.Errorf("unsupported resource type for delete: %s", resourceType)
	}
	return nil
}

// ConsolidateSync implements batch publishing of all transformations and libraries
// This is called after all Create/Update operations to publish changes in a single batch
func (p *Provider) ConsolidateSync(ctx context.Context, st *state.State) error {
	// Phase 1: Batch publish all draft versions
	req, err := p.buildBatchPublishRequest(st)
	if err != nil {
		return fmt.Errorf("building batch publish request: %w", err)
	}

	if len(req.Transformations) > 0 || len(req.Libraries) > 0 {
		if err := p.store.BatchPublish(ctx, req); err != nil {
			return fmt.Errorf("batch publishing %d transformations and %d libraries: %w",
				len(req.Transformations), len(req.Libraries), err)
		}
		log.Info("Successfully published transformations and libraries",
			"transformations", len(req.Transformations), "libraries", len(req.Libraries))
	}

	// Phase 2: Execute deferred deletes (now safe — published state is current)
	if err := p.executePendingDeletes(ctx); err != nil {
		return fmt.Errorf("executing pending deletes: %w", err)
	}

	return nil
}

// executePendingDeletes processes all deferred delete operations.
// Transformations are deleted first since they depend on libraries.
func (p *Provider) executePendingDeletes(ctx context.Context) error {
	p.pendingDeletes.mu.RLock()
	defer p.pendingDeletes.mu.RUnlock()

	for _, id := range p.pendingDeletes.transformations {
		if err := p.store.DeleteTransformation(ctx, id); err != nil {
			return fmt.Errorf("deleting transformation %s: %w", id, err)
		}
	}

	for _, id := range p.pendingDeletes.libraries {
		if err := p.store.DeleteLibrary(ctx, id); err != nil {
			return fmt.Errorf("deleting library %s: %w", id, err)
		}
	}

	log.Info("Successfully deleted transformations and libraries",
		"transformations", len(p.pendingDeletes.transformations), "libraries", len(p.pendingDeletes.libraries))
	return nil
}

// buildBatchPublishRequest builds the batch publish request by extracting version IDs from state
func (p *Provider) buildBatchPublishRequest(st *state.State) (*transformations.BatchPublishRequest, error) {
	req := &transformations.BatchPublishRequest{
		Transformations: []transformations.BatchPublishTransformation{},
		Libraries:       []transformations.BatchPublishLibrary{},
	}

	for urn, resource := range st.Resources {
		switch resource.Type {
		case library.HandlerMetadata.ResourceType:
			libState, ok := resource.OutputRaw.(*model.LibraryState)
			if !ok {
				return nil, fmt.Errorf("resource %s has invalid OutputRaw type for library", urn)
			}
			if libState.VersionID == "" {
				return nil, fmt.Errorf("resource %s has empty version ID", urn)
			}

			req.Libraries = append(req.Libraries, transformations.BatchPublishLibrary{
				VersionID: libState.VersionID,
			})

		case transformation.HandlerMetadata.ResourceType:
			transState, ok := resource.OutputRaw.(*model.TransformationState)
			if !ok {
				return nil, fmt.Errorf("resource %s has invalid OutputRaw type for transformation", urn)
			}
			if transState.VersionID == "" {
				return nil, fmt.Errorf("resource %s has empty version ID", urn)
			}

			req.Transformations = append(req.Transformations, transformations.BatchPublishTransformation{
				VersionID: transState.VersionID,
			})
		}
	}

	return req, nil
}
