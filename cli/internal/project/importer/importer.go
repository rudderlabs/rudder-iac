package importer

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/formatter"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/differ"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
)

const (
	importedDir = "imported"
)

var ErrProjectNotSynced = errors.New("import not allowed as project has changes to be synced")

func WorkspaceImport(
	ctx context.Context,
	location string,
	p project.Provider) error {

	pState, err := p.LoadState(ctx)
	if err != nil {
		return fmt.Errorf("loading state: %w", err)
	}

	sourceGraph := syncer.StateToGraph(pState)
	targetGraph, err := p.GetResourceGraph()
	if err != nil {
		return fmt.Errorf("getting resource graph: %w", err)
	}

	diff := differ.ComputeDiff(sourceGraph, targetGraph, differ.DiffOptions{})
	if diff.IsDiffed() {
		return fmt.Errorf("%w", ErrProjectNotSynced)
	}

	idNamer, err := initNamer(targetGraph)
	if err != nil {
		return fmt.Errorf("initializing namer: %w", err)
	}

	importable, err := p.LoadImportable(ctx, idNamer)
	if err != nil {
		return fmt.Errorf("loading importable resources: %w", err)
	}

	if importable.Len() == 0 {
		fmt.Println("No resources to import")
		return nil
	}

	resolver, err := initResolver(ctx, p, importable, targetGraph)
	if err != nil {
		return fmt.Errorf("setting up import ref resolver: %w", err)
	}

	entities, err := p.FormatForExport(ctx, importable, idNamer, resolver)
	if err != nil {
		return fmt.Errorf("normalizing for import: %w", err)
	}

	formatters := formatter.Setup(formatter.DefaultYAML)

	if err := Write(ctx, filepath.Join(location, importedDir), formatters, entities); err != nil {
		return fmt.Errorf("writing files for formattable entities: %w", err)
	}

	return nil
}

func initNamer(graph *resources.Graph) (namer.Namer, error) {
	idNamer := namer.NewExternalIdNamer(namer.NewKebabCase())

	resourcesMap := graph.Resources()
	externalIDs := make([]namer.ScopeName, 0, len(resourcesMap))
	for _, r := range resourcesMap {
		externalIDs = append(externalIDs, namer.ScopeName{
			Name:  r.ID(),
			Scope: r.Type(),
		})
	}

	if err := idNamer.Load(externalIDs); err != nil {
		return nil, fmt.Errorf("preloading namer with project IDs: %w", err)
	}

	return idNamer, nil
}

func initResolver(
	ctx context.Context,
	p project.Provider,
	importable *resources.ResourceCollection,
	graph *resources.Graph,
) (*resolver.ImportRefResolver, error) {
	remoteCollection, err := p.LoadResourcesFromRemote(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading remote resources: %w", err)
	}

	return &resolver.ImportRefResolver{
		Remote:     remoteCollection,
		Graph:      graph,
		Importable: importable,
	}, nil
}
