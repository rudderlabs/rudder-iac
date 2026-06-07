package resourceops

import (
	"context"
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

// Scope controls which rows ListRows returns.
type Scope int

const (
	// ScopeAll returns both managed and unmanaged resources.
	ScopeAll Scope = iota
	// ScopeManaged returns only resources that have an external ID (IaC-managed).
	ScopeManaged
	// ScopeUnmanaged returns only resources that are not IaC-managed.
	ScopeUnmanaged
)

// Row is one entry in a `get <type>` listing.
// Managed is true iff the resource has an external ID (i.e. is IaC-managed).
type Row struct {
	ExternalID string
	RemoteID   string
	Name       string
	Managed    bool
}

// SupportsUnmanaged reports whether prov implements UnmanagedRemoteResourceLoader,
// meaning it can enumerate unmanaged remote resources.
// It is the companion probe for the two-call degraded protocol: callers that need
// to warn the user should call this before (or after) ListRows and print a one-line
// note when it returns false and the requested scope includes unmanaged resources.
func SupportsUnmanaged(prov any) bool {
	_, ok := prov.(provider.UnmanagedRemoteResourceLoader)
	return ok
}

// ListRows loads managed (and, if supported, unmanaged) remote resources of
// resourceType, merges them — managed entries win on duplicate remote ID —
// and returns them filtered by scope.
//
// Two-call degraded protocol: when prov does not implement
// UnmanagedRemoteResourceLoader, unmanaged rows cannot be listed and ListRows
// silently returns only managed rows (no error). Callers that need to warn the
// user should probe with SupportsUnmanaged(prov) and print a one-line note when
// it returns false and the requested scope includes unmanaged resources.
func ListRows(ctx context.Context, prov provider.ManagedRemoteResourceLoader, resourceType string, scope Scope) ([]Row, error) {
	merged, err := mergedRemote(ctx, prov, resourceType)
	if err != nil {
		return nil, err
	}

	rows := make([]Row, 0, len(merged))
	for _, res := range merged {
		row := Row{
			ExternalID: res.ExternalID,
			RemoteID:   res.ID,
			Name:       extractName(res.Data),
			Managed:    res.ExternalID != "",
		}
		switch scope {
		case ScopeManaged:
			if !row.Managed {
				continue
			}
		case ScopeUnmanaged:
			if row.Managed {
				continue
			}
		case ScopeAll:
			// include everything
		default:
			// unreachable; new scopes must update this switch
		}
		rows = append(rows, row)
	}
	return rows, nil
}

// mergedRemote loads managed resources and, if the provider supports it, unmanaged
// resources, then merges them keyed by remote ID (managed wins on collision).
// A nil collection from either load is treated as empty.
func mergedRemote(ctx context.Context, prov provider.ManagedRemoteResourceLoader, resourceType string) (map[string]*resources.RemoteResource, error) {
	managedCollection, err := prov.LoadResourcesFromRemote(ctx)
	if err != nil {
		return nil, err
	}

	var managed map[string]*resources.RemoteResource
	if managedCollection != nil {
		managed = managedCollection.GetAll(resourceType)
	}

	// Start the result with all managed entries.
	result := make(map[string]*resources.RemoteResource, len(managed))
	for id, res := range managed {
		result[id] = res
	}

	// Merge in unmanaged resources when supported; managed wins on collision.
	unmanagedLoader, ok := prov.(provider.UnmanagedRemoteResourceLoader)
	if !ok {
		return result, nil
	}

	// Kebab-case matches the importer's namer (project/importer/importer.go) for
	// consistency; a fresh namer per call is fine because it only generates
	// suggested external IDs for importable resources during this read.
	importableCollection, err := unmanagedLoader.LoadImportable(ctx, namer.NewExternalIdNamer(namer.NewKebabCase()))
	if err != nil {
		return nil, err
	}
	if importableCollection == nil {
		return result, nil
	}
	for id, res := range importableCollection.GetAll(resourceType) {
		if _, exists := result[id]; !exists {
			result[id] = res
		}
	}

	return result, nil
}

// extractName attempts to pull a "name" field out of a resource's Data when it
// is a map. Returns an empty string when the field is absent or Data is not a map.
func extractName(data any) string {
	m, ok := data.(map[string]any)
	if !ok {
		return ""
	}
	name, _ := m["name"].(string)
	return name
}

// SpecYAML materializes a single managed remote resource into a re-appliable YAML spec string.
// It finds the resource matching id (external-ID first, then remote-ID), runs it through
// the provider's FormatForExport, and encodes the first entity as YAML.
func SpecYAML(ctx context.Context, prov provider.Provider, resourceType, id string) (string, error) {
	content, err := specContent(ctx, prov, resourceType, id)
	if err != nil {
		return "", err
	}
	return EncodeYAML(content)
}

// SpecJSON materializes a single managed remote resource into a re-appliable JSON string.
// It finds the resource matching id (external-ID first, then remote-ID), runs it through
// the provider's FormatForExport, and encodes the first entity as JSON.
func SpecJSON(ctx context.Context, prov provider.Provider, resourceType, id string) (string, error) {
	content, err := specContent(ctx, prov, resourceType, id)
	if err != nil {
		return "", err
	}
	return EncodeJSON(content)
}

// specContent is the shared find-and-export path used by SpecYAML and SpecJSON.
// It loads managed remote resources, finds the one matching id (external-ID-first
// then remote-ID, mirroring Resolver.FindRemote), builds a single-entry collection,
// and delegates to FormatForExport to get the formattable entity content.
func specContent(ctx context.Context, prov provider.Provider, resourceType, id string) (any, error) {
	managed, err := prov.LoadResourcesFromRemote(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading remote resources: %w", err)
	}
	if managed == nil {
		managed = resources.NewRemoteResources()
	}

	all := managed.GetAll(resourceType)

	found := findInMap(all, id)
	if found == nil {
		return nil, fmt.Errorf("%s %q: %w", resourceType, id, ErrResourceNotFound)
	}

	coll := resources.NewRemoteResources()
	coll.Set(resourceType, map[string]*resources.RemoteResource{found.ID: found})

	idNamer := namer.NewExternalIdNamer(namer.NewKebabCase())
	// Remote is populated with this provider's full managed collection so that
	// same-provider cross-resource references (e.g. a tracking plan owned by this
	// provider) can be resolved via ImportRefResolver.ResolveToReference.
	// Known limitation: cross-PROVIDER references (e.g. a tracking plan owned by a
	// different provider) cannot be resolved here because only this provider's remote
	// is loaded; treated as a follow-up.
	refResolver := &resolver.ImportRefResolver{
		Remote:     managed,
		Graph:      resources.NewGraph(),
		Importable: coll,
	}

	entities, err := prov.FormatForExport(coll, idNamer, refResolver)
	if err != nil {
		return nil, fmt.Errorf("materializing spec for %s %q (cross-resource references such as tracking plans may not resolve in single-resource export): %w", resourceType, id, err)
	}
	if len(entities) == 0 {
		return nil, fmt.Errorf("%s %q: %w", resourceType, id, ErrResourceNotFound)
	}

	return entities[0].Content, nil
}
