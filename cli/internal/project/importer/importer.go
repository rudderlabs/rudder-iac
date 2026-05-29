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
)

const ImportManifestFile = "import-manifest.yaml"

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
	p ImportProvider,
	merge bool) error {

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
	if !merge {
		if diff.HasDiff() {
			return fmt.Errorf("%w", ErrProjectNotSynced)
		}
	} else if len(diff.RemovedResources) > 0 {
		return fmt.Errorf("%w: pending deletions must be applied before importing with --merge: %v",
			ErrProjectNotSynced, diff.RemovedResources)
	}

	idNamer, err := initNamer(targetGraph)
	if err != nil {
		return fmt.Errorf("initializing namer: %w", err)
	}

	// Pass localGraph when in merge mode, nil otherwise
	var localGraph *resources.Graph
	if merge {
		localGraph = targetGraph
	}

	importable, err := p.LoadImportable(ctx, idNamer, localGraph)
	if err != nil {
		return fmt.Errorf("loading importable resources: %w", err)
	}

	if importable.Len() == 0 {
		fmt.Println("No resources to import")
		return nil
	}

	// Print auto-link summary when in merge mode
	if merge {
		autoLinked := collectAutoLinkEntries(importable, targetGraph)
		if summary := formatAutoLinkSummary(autoLinked); summary != "" {
			fmt.Print(summary)
		}
	}

	resolver, err := initResolver(remoteCollection, importable, targetGraph)
	if err != nil {
		return fmt.Errorf("setting up import ref resolver: %w", err)
	}

	entities, entries, err := p.FormatForExport(importable, idNamer, resolver)
	if err != nil {
		return fmt.Errorf("normalizing for import: %w", err)
	}

	formatters := formatter.Setup(formatter.DefaultYAML, formatter.DefaultText)

	location := project.Location()
	outputDir := filepath.Join(location, ImportedDir)
	if err := writer.Write(ctx, outputDir, formatters, entities); err != nil {
		return fmt.Errorf("writing files for formattable entities: %w", err)
	}

	// Emit the aggregated import-manifest alongside the resource specs so
	// URN → remote-ID mappings live in one queryable place instead of being
	// scattered across every resource's inline metadata.import block.
	if len(entries) > 0 {
		manifestEntity := []writer.FormattableEntity{{
			Content:      importmanifest.BuildSpec(entries),
			RelativePath: ImportManifestFile,
		}}
		if err := writer.Write(ctx, outputDir, formatters, manifestEntity); err != nil {
			return fmt.Errorf("writing import manifest: %w", err)
		}
	}

	return nil
}

// collectAutoLinkEntries identifies imported resources whose ExternalID matches
// an existing resource in the local graph — these are the auto-linked ones.
// It also captures the WorkspaceID from the local graph's import metadata so
// manifest entries can be grouped correctly.
func collectAutoLinkEntries(importable *resources.RemoteResources, localGraph *resources.Graph) []autoLinkEntry {
	var entries []autoLinkEntry
	for _, r := range localGraph.Resources() {
		urn := r.URN()
		for _, remote := range importable.GetAll(r.Type()) {
			remoteURN := resources.URN(remote.ExternalID, r.Type())
			if remoteURN == urn {
				entries = append(entries, autoLinkEntry{
					URN:      urn,
					RemoteID: remote.ID,
				})
			}
		}
	}
	return entries
}

type autoLinkEntry struct {
	URN      string
	RemoteID string
}

func formatAutoLinkSummary(entries []autoLinkEntry) string {
	if len(entries) == 0 {
		return ""
	}
	summary := fmt.Sprintf("Auto-linked %d resources to existing local project:\n", len(entries))
	for _, e := range entries {
		summary += fmt.Sprintf("  %s → %s\n", e.URN, e.RemoteID)
	}
	return summary
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
