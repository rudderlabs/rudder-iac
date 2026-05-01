package importer

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/formatter"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/writer"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/differ"
)

const (
	ImportedDir = "imported"
)

// FilterOption specifies which resources to import based on their management status.
type FilterOption string

const (
	// FilterUnmanaged imports only resources without external IDs (not yet managed by IaC).
	FilterUnmanaged FilterOption = "unmanaged"
	// FilterManaged imports only resources with external IDs (already managed by IaC).
	FilterManaged FilterOption = "managed"
	// FilterAll imports all resources regardless of their management status.
	FilterAll FilterOption = "all"
)

// ConflictResolution specifies how to handle conflicts when importing managed resources
// that already exist locally.
type ConflictResolution string

const (
	// ConflictKeepLocal skips importing resources that already exist locally.
	ConflictKeepLocal ConflictResolution = "keep-local"
	// ConflictAcceptIncoming overwrites local files with the incoming/remote version.
	ConflictAcceptIncoming ConflictResolution = "accept-incoming"
	// ConflictKeepBoth keeps both versions by creating a new file with a suffix for the incoming version.
	ConflictKeepBoth ConflictResolution = "keep-both"
)

// ImportOptions configures the workspace import behavior.
type ImportOptions struct {
	// Filter determines which resources to import based on their management status.
	// Defaults to FilterUnmanaged if not specified.
	Filter FilterOption
	// OnConflict determines how to handle conflicts when importing managed resources
	// that already exist locally. Only applies when Filter is FilterManaged or FilterAll.
	// Defaults to ConflictKeepLocal if not specified.
	OnConflict ConflictResolution
}

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
	opts ImportOptions,
) error {
	// Default to unmanaged if no filter specified
	if opts.Filter == "" {
		opts.Filter = FilterUnmanaged
	}
	// Default to keep-local if no conflict resolution specified
	if opts.OnConflict == "" {
		opts.OnConflict = ConflictKeepLocal
	}

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
	if diff.HasDiff() {
		return fmt.Errorf("%w", ErrProjectNotSynced)
	}

	idNamer, err := initNamer(targetGraph)
	if err != nil {
		return fmt.Errorf("initializing namer: %w", err)
	}

	// Load resources based on filter option
	importable, err := loadResourcesForImport(ctx, p, opts.Filter, idNamer, remoteCollection)
	if err != nil {
		return fmt.Errorf("loading resources for import: %w", err)
	}

	if importable.Len() == 0 {
		fmt.Println("No resources to import")
		return nil
	}

	resolver, err := initResolver(remoteCollection, importable, targetGraph)
	if err != nil {
		return fmt.Errorf("setting up import ref resolver: %w", err)
	}

	entities, err := p.FormatForExport(importable, idNamer, resolver)
	if err != nil {
		return fmt.Errorf("normalizing for import: %w", err)
	}

	// Apply conflict resolution when importing managed resources
	if opts.Filter == FilterManaged || opts.Filter == FilterAll {
		entities = applyConflictResolution(entities, targetGraph, opts.OnConflict)
	}

	if len(entities) == 0 {
		fmt.Println("No resources to import")
		return nil
	}

	formatters := formatter.Setup(formatter.DefaultYAML, formatter.DefaultText)

	location := project.Location()
	if err := writer.Write(ctx, filepath.Join(location, ImportedDir), formatters, entities); err != nil {
		return fmt.Errorf("writing files for formattable entities: %w", err)
	}

	return nil
}

// loadResourcesForImport loads resources based on the filter option.
// - FilterUnmanaged: loads only unmanaged resources (without external IDs)
// - FilterManaged: loads only managed resources (with external IDs), using existing IDs
// - FilterAll: combines both managed and unmanaged resources
func loadResourcesForImport(
	ctx context.Context,
	p ImportProvider,
	filter FilterOption,
	idNamer namer.Namer,
	remoteCollection *resources.RemoteResources,
) (*resources.RemoteResources, error) {
	switch filter {
	case FilterUnmanaged:
		return p.LoadImportable(ctx, idNamer)

	case FilterManaged:
		return loadManagedForExport(remoteCollection)

	case FilterAll:
		unmanaged, err := p.LoadImportable(ctx, idNamer)
		if err != nil {
			return nil, fmt.Errorf("loading unmanaged resources: %w", err)
		}
		managed, err := loadManagedForExport(remoteCollection)
		if err != nil {
			return nil, fmt.Errorf("loading managed resources: %w", err)
		}
		return unmanaged.Merge(managed)

	default:
		return nil, fmt.Errorf("unknown filter option: %s", filter)
	}
}

// loadManagedForExport returns the remote collection as-is since it already contains
// managed resources with their existing external IDs from LoadResourcesFromRemote.
func loadManagedForExport(remoteCollection *resources.RemoteResources) (*resources.RemoteResources, error) {
	return remoteCollection, nil
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

// applyConflictResolution filters and modifies entities based on the conflict resolution strategy.
// It checks if resources already exist in the target graph and applies the appropriate action:
// - ConflictKeepLocal: Skip entities for resources that exist locally
// - ConflictAcceptIncoming: Keep all entities (they will overwrite local files)
// - ConflictKeepBoth: Modify paths for conflicting entities by adding a suffix
func applyConflictResolution(
	entities []writer.FormattableEntity,
	targetGraph *resources.Graph,
	onConflict ConflictResolution,
) []writer.FormattableEntity {
	if len(entities) == 0 {
		return entities
	}

	var result []writer.FormattableEntity

	for _, entity := range entities {
		hasConflict := entityHasConflict(entity, targetGraph)

		switch onConflict {
		case ConflictKeepLocal:
			if !hasConflict {
				result = append(result, entity)
			}
		case ConflictAcceptIncoming:
			result = append(result, entity)
		case ConflictKeepBoth:
			if hasConflict {
				entity.RelativePath = generateConflictPath(entity.RelativePath)
			}
			result = append(result, entity)
		default:
			// Unknown conflict resolution, default to keep-local behavior
			if !hasConflict {
				result = append(result, entity)
			}
		}
	}

	return result
}

// entityHasConflict checks if any resource in the entity already exists in the target graph.
func entityHasConflict(entity writer.FormattableEntity, targetGraph *resources.Graph) bool {
	urns := extractURNsFromEntity(entity)
	for _, urn := range urns {
		if _, exists := targetGraph.GetResource(urn); exists {
			return true
		}
	}
	return false
}

// extractURNsFromEntity extracts URNs from the entity's metadata.
// It handles both specs.Spec pointers and values.
func extractURNsFromEntity(entity writer.FormattableEntity) []string {
	var spec *specs.Spec

	switch s := entity.Content.(type) {
	case *specs.Spec:
		spec = s
	case specs.Spec:
		spec = &s
	default:
		return nil
	}

	metadata, err := spec.CommonMetadata()
	if err != nil || metadata.Import == nil {
		return nil
	}

	var urns []string
	for _, workspace := range metadata.Import.Workspaces {
		for _, resource := range workspace.Resources {
			if resource.URN != "" {
				urns = append(urns, resource.URN)
			}
		}
	}

	return urns
}

// generateConflictPath adds a suffix to the file path for keep-both conflict resolution.
// For example, "sources/my-source.yaml" becomes "sources/my-source-imported.yaml"
func generateConflictPath(relativePath string) string {
	ext := filepath.Ext(relativePath)
	base := strings.TrimSuffix(relativePath, ext)
	return base + "-imported" + ext
}
