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
	ImportedDir = "imported"
)

var ErrProjectNotSynced = errors.New("import not allowed as project has changes to be synced")

func WorkspaceImport(
	ctx context.Context,
	location string,
	p project.Provider) error {

	remoteCollection, err := p.LoadResourcesFromRemote(ctx)
	if err != nil {
		return fmt.Errorf("loading remote resources: %w", err)
	}

	pstate, err := p.LoadStateFromResources(ctx, remoteCollection)
	if err != nil {
		return fmt.Errorf("loading state from resources: %w", err)
	}

	sourceGraph := syncer.StateToGraph(pstate)
	targetGraph, err := p.GetResourceGraph()
	if err != nil {
		return fmt.Errorf("getting resource graph: %w", err)
	}

	diff := differ.ComputeDiff(sourceGraph, targetGraph, differ.DiffOptions{})
	if diff.HasDiff() {
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

	resolver, err := initResolver(remoteCollection, importable, targetGraph)
	if err != nil {
		return fmt.Errorf("setting up import ref resolver: %w", err)
	}

	entities, err := p.FormatForExport(ctx, importable, idNamer, resolver)
	if err != nil {
		return fmt.Errorf("normalizing for import: %w", err)
	}

	formatters := formatter.Setup(formatter.DefaultYAML)

	if err := Write(ctx, filepath.Join(location, ImportedDir), formatters, entities); err != nil {
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
	remoteCollection *resources.ResourceCollection,
	importable *resources.ResourceCollection,
	graph *resources.Graph,
) (*resolver.ImportRefResolver, error) {

	return &resolver.ImportRefResolver{
		Remote:     remoteCollection,
		Graph:      graph,
		Importable: importable,
	}, nil
}
