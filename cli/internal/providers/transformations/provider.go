package transformations

import (
	"context"
	"fmt"

	"github.com/rudderlabs/rudder-iac/api/client"
	transformations "github.com/rudderlabs/rudder-iac/api/client/transformations"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/handlers/library"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/handlers/transformation"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/parser"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
)

// Provider wraps BaseProvider and adds transformations-specific functionality
type Provider struct {
	*provider.BaseProvider
	store transformations.TransformationStore
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

// ResourceGraph overrides BaseProvider.ResourceGraph to add transformation→library dependencies
// by parsing transformation code to extract library imports
func (p *Provider) ResourceGraph() (*resources.Graph, error) {
	graph := resources.NewGraph()

	// libraries
	libraryHandler, ok := p.BaseProvider.GetHandler(library.HandlerMetadata.ResourceType)
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
	transformationHandler, ok := p.BaseProvider.GetHandler(transformation.HandlerMetadata.ResourceType)
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
					"transformation %s imports library '%s' which is not found in the project. "+
						"Ensure you have a transformation-library spec with import_name: '%s'",
					r.ID(),
					handleName,
					handleName,
				)
			}

			graph.AddDependency(r.URN(), libraryURN)
		}
		graph.AddResource(r)
	}

	return graph, nil
}

// ConsolidateSync implements batch publishing of all transformations and libraries
// This is called after all Create/Update operations to publish changes in a single batch
func (p *Provider) ConsolidateSync(ctx context.Context, st *state.State) error {
	// Build batch publish request by extracting versions from state
	req, err := p.buildBatchPublishRequest(st)
	if err != nil {
		return fmt.Errorf("building batch public request: %w", err)
	}

	// If no versions to publish, we're done
	if len(req.Transformations) == 0 && len(req.Libraries) == 0 {
		return nil
	}

	// Batch publish all versions
	if err := p.store.BatchPublish(ctx, req); err != nil {
		return fmt.Errorf("batch publishing %d transformations and %d libraries: %w",
			len(req.Transformations), len(req.Libraries), err)
	}

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
