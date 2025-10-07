package importer

import (
	"context"
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
)

func WorkspaceImport(
	ctx context.Context,
	location string,
	project project.Project,
	p project.Provider) error {
	idNamer, err := initNamer(project)
	if err != nil {
		return fmt.Errorf("initializing namer: %w", err)
	}

	importable, err := p.LoadImportable(ctx, idNamer)
	if err != nil {
		return fmt.Errorf("loading importable resources: %w", err)
	}

	if importable.Len() == 0 {
		return nil
	}

	resolver, err := initResolver(ctx, p, importable)
	if err != nil {
		return fmt.Errorf("setting up import ref resolver: %w", err)
	}

	_, err = p.FormatForExport(ctx, importable, idNamer, resolver)
	if err != nil {
		return fmt.Errorf("normalizing for import: %w", err)
	}

	return nil
}

func initNamer(p project.Project) (namer.Namer, error) {
	idNamer := namer.NewExternalIdNamer(namer.NewKebabCase())

	graph, err := p.GetResourceGraph()
	if err != nil {
		return nil, fmt.Errorf("getting resource graph: %w", err)
	}

	resourcesMap := graph.Resources()
	externalIDs := make([]string, 0, len(resourcesMap))
	for _, r := range resourcesMap {
		externalIDs = append(externalIDs, r.ID())
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
) (*resolver.ImportRefResolver, error) {
	remoteCollection, err := p.LoadResourcesFromRemote(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading remote resources: %w", err)
	}

	graph, err := p.GetResourceGraph()
	if err != nil {
		return nil, fmt.Errorf("getting resource graph: %w", err)
	}

	return &resolver.ImportRefResolver{
		Remote:     remoteCollection,
		Graph:      graph,
		Importable: importable,
	}, nil
}
