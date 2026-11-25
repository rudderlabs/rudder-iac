package importer

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/formatter"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/importmanifest"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/loader"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/writer"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

var (
	ErrProjectPresent      = errors.New("reseed needs a directory with no project YAML files")
	ErrTypesWithoutHandler = errors.New("reseed cannot write every resource type this workspace holds")
)

// ReseedProvider writes a whole workspace: the resources the CLI already
// manages and the ones it does not, in one pass.
type ReseedProvider interface {
	provider.UnmanagedRemoteResourceLoader
	provider.Exporter
	// LoadManaged returns the managed resources and the resource types whose
	// provider cannot export them.
	LoadManaged(ctx context.Context) (*resources.RemoteResources, []string, error)
}

// Reseed writes a complete project for the workspace into location, which must
// hold no project YAML. Managed resources (those carrying an externalId) get
// specs keyed by that externalId, so a following apply matches them instead of
// recreating them; unmanaged ones get specs plus an import-manifest entry, so
// apply adopts them. A mixed workspace reseeds in one pass.
//
// Because the directory holds no project, there is nothing to reconcile, so
// none of the divergence checks a plain import runs are needed here. Reseed
// recovers what the specs say, not how the owner laid them out: file layout,
// comments, variables and secret values are lost, and secrets come back as
// '{{ .VAR }}' placeholders listed in the generated var file.
func Reseed(ctx context.Context, location string, p ReseedProvider) error {
	if err := requireNoProject(location); err != nil {
		return err
	}

	managed, unsupported, err := p.LoadManaged(ctx)
	if err != nil {
		return fmt.Errorf("loading managed resources: %w", err)
	}
	// A type nothing can export would be missing from the project, and the next
	// apply would read that absence as a deletion. Refuse before writing.
	if len(unsupported) > 0 {
		return fmt.Errorf("%w: %s", ErrTypesWithoutHandler, strings.Join(unsupported, ", "))
	}
	managed = managed.Without(func(_ string, r *resources.RemoteResource) bool { return r.ExternalID == "" })

	// The namer names the unmanaged half. The managed externalIds are taken
	// first, or a generated name could land on one and two specs would claim
	// the same resource.
	idNamer := namer.NewExternalIdNamer(namer.NewKebabCase())
	if err := idNamer.Load(takenNames(managed)); err != nil {
		return fmt.Errorf("reserving managed external ids: %w", err)
	}

	importable, err := p.LoadImportable(ctx, idNamer)
	if err != nil {
		return fmt.Errorf("loading importable resources: %w", err)
	}

	all, err := union(managed, importable)
	if err != nil {
		return err
	}
	if all.Len() == 0 {
		fmt.Println("Workspace holds no resources to reseed")
		return nil
	}

	// Nothing is local, so every reference resolves through the collection
	// being written rather than through an existing spec.
	refResolver := &resolver.ImportRefResolver{
		Remote:     all,
		Importable: all,
		Graph:      resources.NewGraph(),
	}

	entities, entries, err := p.FormatForExport(all, idNamer, refResolver)
	if err != nil {
		return fmt.Errorf("formatting workspace for export: %w", err)
	}

	// A managed resource is already linked by its externalId. A manifest entry
	// for it would tell apply to adopt what it owns.
	entries = unmanagedEntries(entries, managed)

	// Checked before anything is written, so a refused reseed leaves no specs
	// behind referencing a variable that would fill two secrets.
	if err := checkVariableOwners(location, entities); err != nil {
		return err
	}

	formatters := formatter.Setup(formatter.DefaultYAML, formatter.DefaultText)
	if err := writeSpecs(ctx, location, formatters, entities); err != nil {
		return err
	}

	return writeImportManifest(ctx, location, formatters, entries)
}

// requireNoProject refuses a directory that already holds project YAML. It
// reads the directory the way the project loader does, so the files it counts
// are exactly the ones a later load would pick up, var files excluded.
func requireNoProject(location string) error {
	raw, err := (&loader.Loader{}).Load(location)
	if err != nil {
		return fmt.Errorf("reading %s: %w", location, err)
	}
	if len(raw) == 0 {
		return nil
	}

	paths := make([]string, 0, len(raw))
	for path := range raw {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return fmt.Errorf("%w: %s holds %s. Reseed into an empty directory",
		ErrProjectPresent, location, strings.Join(paths, ", "))
}

// union combines the managed and unmanaged halves of one workspace.
// RemoteResources.Merge cannot: it is built for providers with disjoint
// resource types and refuses a type both sides hold, which is every type a
// mixed workspace has. A remote id in both halves means one of the two loaders
// is wrong about who manages it, so that is an error.
func union(managed, importable *resources.RemoteResources) (*resources.RemoteResources, error) {
	byType := make(map[string]map[string]*resources.RemoteResource)
	for _, half := range []*resources.RemoteResources{managed, importable} {
		for _, resourceType := range half.Types() {
			if byType[resourceType] == nil {
				byType[resourceType] = make(map[string]*resources.RemoteResource)
			}
			for id, r := range half.GetAll(resourceType) {
				if _, dup := byType[resourceType][id]; dup {
					return nil, fmt.Errorf("%s %s loaded as both managed and unmanaged", resourceType, id)
				}
				byType[resourceType][id] = r
			}
		}
	}

	all := resources.NewRemoteResources()
	for resourceType, m := range byType {
		all.Set(resourceType, m)
	}
	return all, nil
}

// takenNames lists the managed external ids as names the namer must not reuse.
func takenNames(managed *resources.RemoteResources) []namer.ScopeName {
	var taken []namer.ScopeName
	for _, resourceType := range managed.Types() {
		for _, r := range managed.GetAll(resourceType) {
			taken = append(taken, namer.ScopeName{Name: r.ExternalID, Scope: resourceType})
		}
	}
	return taken
}

// unmanagedEntries drops the manifest entries describing managed resources.
// The manifest is a record the project keeps in version control, and an entry
// in it claims its resource still needs adopting, which a managed one does
// not. Providers also write the same claim inline into each spec they export,
// and that copy stays: it is theirs to write, it costs the plan nothing, and
// import workspace --managed leaves it too.
func unmanagedEntries(entries []importmanifest.ImportEntry, managed *resources.RemoteResources) []importmanifest.ImportEntry {
	urns := make(map[string]bool)
	for _, resourceType := range managed.Types() {
		for _, r := range managed.GetAll(resourceType) {
			urns[resources.URN(r.ExternalID, resourceType)] = true
		}
	}

	kept := make([]importmanifest.ImportEntry, 0, len(entries))
	for _, e := range entries {
		if !urns[e.URN] {
			kept = append(kept, e)
		}
	}
	return kept
}

// writeImportManifest writes the manifest that makes apply adopt the unmanaged
// resources reseed just wrote specs for.
func writeImportManifest(
	ctx context.Context,
	location string,
	formatters formatter.Formatters,
	entries []importmanifest.ImportEntry,
) error {
	node, err := importmanifest.BuildNode(entries)
	if err != nil {
		return fmt.Errorf("building import manifest: %w", err)
	}
	if node == nil {
		return nil
	}

	entity := writer.FormattableEntity{
		Content:      node,
		RelativePath: importmanifest.FileName,
	}
	if err := writer.Write(ctx, location, formatters, []writer.FormattableEntity{entity}); err != nil {
		return fmt.Errorf("writing import manifest: %w", err)
	}
	return nil
}
