package importer

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/formatter"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/importmanifest"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/writer"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/importmatcher"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/differ"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
)

const (
	ImportedDir = "imported"
)

var (
	ErrProjectNotSynced = errors.New("import not allowed as project has changes to be synced")
	ErrAmbiguousMatch   = errors.New("merge import matched multiple remote resources to one local resource")
)

type ImportProvider interface {
	provider.RemoteResourceLoader
	provider.StateLoader
	provider.Exporter
	provider.ResourceMatcherProvider
}

type Project interface {
	ResourceGraph() (*resources.Graph, error)
	Location() string
}

// ImportOptions configures a workspace import. Merge enables smart-import
// conflict detection (link matching remote resources to existing local
// resources instead of writing duplicate specs).
type ImportOptions struct {
	Merge bool
}

func WorkspaceImport(
	ctx context.Context,
	project Project,
	p ImportProvider,
	opts ImportOptions) error {

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
	if err := checkSyncStatus(diff, opts.Merge); err != nil {
		return err
	}

	idNamer, err := initNamer(targetGraph, sourceGraph)
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

	if opts.Merge {
		if err := markMatchedWith(p, sourceGraph, targetGraph, importable); err != nil {
			return err
		}
	}

	resolver, err := initResolver(remoteCollection, importable, targetGraph, opts.Merge)
	if err != nil {
		return fmt.Errorf("setting up import ref resolver: %w", err)
	}

	entities, importEntries, err := p.FormatForExport(importable, idNamer, resolver)
	if err != nil {
		return fmt.Errorf("normalizing for import: %w", err)
	}

	formatters := formatter.Setup(formatter.DefaultYAML, formatter.DefaultText)

	location := project.Location()
	importDir := filepath.Join(location, ImportedDir)
	if err := writer.Write(ctx, importDir, formatters, entities); err != nil {
		return fmt.Errorf("writing files for formattable entities: %w", err)
	}

	// Only emit the import-manifest when the importMerge experimental flag is
	// enabled — the feature is incomplete and the artifact would confuse users.
	if config.GetConfig().ExperimentalFlags.ImportMerge {
		manifestNode, err := importmanifest.BuildNode(importEntries)
		if err != nil {
			return fmt.Errorf("building import manifest: %w", err)
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

// checkSyncStatus guards the import against a diverged project. Without merge,
// any pending change blocks — HasNonSecretDiff (not HasDiff) so resources that
// only re-apply an unknown secret, which is expected on every run, do not
// permanently block imports. With merge, divergence is the point — including
// pending deletions: only deletions an importable actually references error,
// precisely at reference resolution (resolver.ErrPendingDeleteConflict), and
// before any file is written.
func checkSyncStatus(diff *differ.Diff, merge bool) error {
	if merge {
		return nil
	}

	if diff.HasNonSecretDiff() {
		return fmt.Errorf("%w", ErrProjectNotSynced)
	}
	return nil
}

// markMatchedWith runs merge conflict detection against the local project
// graph, marking matched importable resources in place. Providers without
// matchers contribute nothing, leaving their resources on namer identities.
//
// A claim collision — two remote resources matching one local resource — means
// a matcher predicate does not mirror upstream uniqueness or the upstream data
// is dirty. Either way the merge result would be wrong, so fail fast rather
// than silently importing an ambiguous mapping.
func markMatchedWith(
	matchers provider.ResourceMatcherProvider,
	sourceGraph *resources.Graph,
	targetGraph *resources.Graph,
	importable *resources.RemoteResources,
) error {
	claimed := importmatcher.Mark(importmatcher.Scope{
		LocalGraph:  targetGraph,
		RemoteGraph: sourceGraph,
		Importable:  importable,
	}, matchers.ResourceMatchers())

	if len(claimed) > 0 {
		details := make([]string, len(claimed))
		for i, c := range claimed {
			details[i] = c.String()
		}
		return fmt.Errorf("%w: %s", ErrAmbiguousMatch, strings.Join(details, "; "))
	}
	return nil
}

// initNamer preloads the union of target (local project) and source (managed
// remote) IDs. The union matters under merge with pending deletions: a
// locally-deleted resource is absent from target but its remote twin still
// holds the ID upstream, so it must stay reserved until the deletion applies.
func initNamer(targetGraph, sourceGraph *resources.Graph) (namer.Namer, error) {
	idNamer := namer.NewExternalIdNamer(namer.NewKebabCase())

	seen := make(map[string]bool)
	externalIDs := make([]namer.ScopeName, 0)
	for _, graph := range []*resources.Graph{targetGraph, sourceGraph} {
		for urn, r := range graph.Resources() {
			if seen[urn] {
				continue
			}
			seen[urn] = true
			externalIDs = append(externalIDs, namer.ScopeName{
				Name:  r.ID(),
				Scope: r.Type(),
			})
		}
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
	merge bool,
) (*resolver.ImportRefResolver, error) {

	return &resolver.ImportRefResolver{
		Remote:     remoteCollection,
		Graph:      graph,
		Importable: importable,
		Merge:      merge,
	}, nil
}
