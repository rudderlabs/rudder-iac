package importer

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/formatter"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/importmanifest"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/writer"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/differ"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
)

const (
	ImportedDir = "imported"
)

var ErrProjectNotSynced = errors.New("import not allowed as project has changes to be synced")

type ImportProvider interface {
	provider.RemoteResourceLoader
	provider.StateLoader
	provider.Exporter
}

type Project interface {
	ResourceGraph() (*resources.Graph, error)
	Location() string
}

func WorkspaceImport(
	ctx context.Context,
	project Project,
	p ImportProvider) error {

	remoteCollection, err := p.LoadResourcesFromRemote(ctx)
	if err != nil {
		return fmt.Errorf("loading remote resources: %w", err)
	}

	pstate, err := p.MapRemoteToState(remoteCollection)
	if err != nil {
		return fmt.Errorf("loading state from resources: %w", err)
	}

	sourceGraph := syncer.StateToGraph(pstate)
	targetGraph, err := project.ResourceGraph()
	if err != nil {
		return fmt.Errorf("getting resource graph: %w", err)
	}

	diff := differ.ComputeDiff(sourceGraph, targetGraph, differ.DiffOptions{})
	// HasNonSecretDiff (not HasDiff) so resources that only re-apply an unknown
	// secret — which is expected on every run — do not permanently block imports.
	if diff.HasNonSecretDiff() {
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

	entities, importEntries, err := p.FormatForExport(importable, idNamer, resolver)
	if err != nil {
		return fmt.Errorf("normalizing for import: %w", err)
	}

	// Aggregate the URN → remote-ID mappings the providers emitted into a single
	// project-level import-manifest, written alongside the resource specs. The
	// node carries a header comment discouraging hand-edits.
	manifestNode, err := importmanifest.BuildNode(importEntries)
	if err != nil {
		return fmt.Errorf("building import manifest: %w", err)
	}

	formatters := formatter.Setup(formatter.DefaultYAML, formatter.DefaultText)

	location := project.Location()
	importDir := filepath.Join(location, ImportedDir)
	if err := writer.Write(ctx, importDir, formatters, entities); err != nil {
		return fmt.Errorf("writing files for formattable entities: %w", err)
	}

	if manifestNode != nil {
		manifestEntity := writer.FormattableEntity{
			Content:      manifestNode,
			RelativePath: importmanifest.FileName,
		}
		if err := writer.Write(ctx, importDir, formatters, []writer.FormattableEntity{manifestEntity}); err != nil {
			return fmt.Errorf("writing import manifest: %w", err)
		}
	}

	varFile, err := scaffoldSecretsVarFile(ctx, importDir, entities)
	if err != nil {
		return fmt.Errorf("scaffolding secrets var file: %w", err)
	}
	if varFile != "" {
		ui.PrintInfo(fmt.Sprintf("Imported specs reference variables for secret values.\n"+
			"Fill in the placeholders in %s (keep it out of version control) and pass it to apply via --var-file.", varFile))
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
	remoteCollection *resources.RemoteResources,
	importable *resources.RemoteResources,
	graph *resources.Graph,
) (*resolver.ImportRefResolver, error) {

	return &resolver.ImportRefResolver{
		Remote:     remoteCollection,
		Graph:      graph,
		Importable: importable,
	}, nil
}
