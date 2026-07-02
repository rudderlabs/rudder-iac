package resourceops

import (
	"context"
	"fmt"
	"reflect"

	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
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
			// Unmanaged resources are surfaced for discovery only. LoadImportable
			// attaches a namer-suggested external id; clear it so the read layer
			// reports them as unmanaged (blank external-id / MANAGED=no), and so
			// delete correctly rejects them. The suggestion is an import-time
			// concern, not a managed-ness signal.
			u := *res
			u.ExternalID = ""
			result[id] = &u
		}
	}

	return result, nil
}

// extractName attempts to pull a "name" field out of a resource's Data when it
// is a map. Returns an empty string when the field is absent or Data is not a map.
func extractName(data any) string {
	if m, ok := data.(map[string]any); ok {
		name, _ := m["name"].(string)
		return name
	}
	// Providers may carry Data as a typed struct (e.g. *EventStreamSource); read
	// its Name field reflectively so the table's NAME column is populated.
	v := reflect.Indirect(reflect.ValueOf(data))
	if v.IsValid() && v.Kind() == reflect.Struct {
		if f := v.FieldByName("Name"); f.IsValid() && f.Kind() == reflect.String {
			return f.String()
		}
	}
	return ""
}

// SpecYAML materializes a single managed remote resource into a re-appliable YAML spec string.
// It finds the resource matching id (external-ID first, then remote-ID), runs it through
// the provider's FormatForExport, and encodes the first entity as YAML.
func SpecYAML(ctx context.Context, prov provider.Provider, router provider.TypeRouter, resourceType, id string) (string, error) {
	content, _, err := specContent(ctx, prov, router, resourceType, id)
	if err != nil {
		return "", err
	}
	return EncodeYAML(content)
}

// SpecYAMLWithManaged materializes the re-appliable YAML spec for a single resource
// and reports whether it is managed (has an external ID), reusing the same single-resource
// lookup as SpecYAML with no extra remote round-trip beyond it.
func SpecYAMLWithManaged(ctx context.Context, prov provider.Provider, router provider.TypeRouter, resourceType, id string) (string, bool, error) {
	content, managed, err := specContent(ctx, prov, router, resourceType, id)
	if err != nil {
		return "", false, err
	}
	yamlStr, err := EncodeYAML(content)
	if err != nil {
		return "", false, err
	}
	return yamlStr, managed, nil
}

// SpecJSON materializes a single managed remote resource into a re-appliable JSON string.
// It finds the resource matching id (external-ID first, then remote-ID), runs it through
// the provider's FormatForExport, and encodes the first entity as JSON.
func SpecJSON(ctx context.Context, prov provider.Provider, router provider.TypeRouter, resourceType, id string) (string, error) {
	content, _, err := specContent(ctx, prov, router, resourceType, id)
	if err != nil {
		return "", err
	}
	return EncodeJSON(content)
}

// exportRefResolver resolves a cross-resource reference (e.g. a source's tracking
// plan) encountered while materializing a single resource's spec. It routes the
// referenced type to its own provider (so cross-PROVIDER references work) and,
// per the reference:
//   - if the dependency is MANAGED, returns its local `#type:external-id` reference;
//   - if it is UNMANAGED but exists, returns its namer-suggested (adopt-ready) reference;
//   - if it can't be found, degrades to the raw `#type:remote-id` so single-resource
//     export never hard-fails on a dangling reference.
//
// Lookups are lazy and cached per type, so only the types actually referenced are
// loaded. A nil router degrades every reference (used where cross-provider context
// is unavailable, e.g. unit tests without a composite).
type exportRefResolver struct {
	ctx        context.Context
	router     provider.TypeRouter
	managed    map[string]map[string]*resources.RemoteResource
	importable map[string]map[string]*resources.RemoteResource
}

func newExportRefResolver(ctx context.Context, router provider.TypeRouter) *exportRefResolver {
	return &exportRefResolver{
		ctx:        ctx,
		router:     router,
		managed:    map[string]map[string]*resources.RemoteResource{},
		importable: map[string]map[string]*resources.RemoteResource{},
	}
}

func (r *exportRefResolver) managedOf(entityType string) map[string]*resources.RemoteResource {
	if m, ok := r.managed[entityType]; ok {
		return m
	}
	m := map[string]*resources.RemoteResource{}
	if r.router != nil {
		if p, err := r.router.ProviderForType(entityType); err == nil {
			if coll, err := p.LoadResourcesFromRemote(r.ctx); err == nil && coll != nil {
				m = coll.GetAll(entityType)
			}
		}
	}
	r.managed[entityType] = m
	return m
}

func (r *exportRefResolver) importableOf(entityType string) map[string]*resources.RemoteResource {
	if m, ok := r.importable[entityType]; ok {
		return m
	}
	m := map[string]*resources.RemoteResource{}
	if r.router != nil {
		if p, err := r.router.ProviderForType(entityType); err == nil {
			if loader, ok := p.(provider.UnmanagedRemoteResourceLoader); ok {
				if coll, err := loader.LoadImportable(r.ctx, namer.NewExternalIdNamer(namer.NewKebabCase())); err == nil && coll != nil {
					m = coll.GetAll(entityType)
				}
			}
		}
	}
	r.importable[entityType] = m
	return m
}

func (r *exportRefResolver) ResolveToReference(entityType, remoteID string) (string, error) {
	if res, ok := r.managedOf(entityType)[remoteID]; ok && res.ExternalID != "" {
		return fmt.Sprintf("#%s:%s", entityType, res.ExternalID), nil
	}
	if res, ok := r.importableOf(entityType)[remoteID]; ok && res.Reference != "" {
		return res.Reference, nil
	}
	return fmt.Sprintf("#%s:%s", entityType, remoteID), nil
}

// specContent is the shared find-and-export path used by SpecYAML, SpecYAMLWithManaged,
// and SpecJSON. It loads managed (and, as a fallback, unmanaged) remote resources,
// finds the one matching id (external-ID-first then remote-ID), builds a single-entry
// collection, and delegates to FormatForExport. Cross-resource references in the
// exported spec are resolved via router (see exportRefResolver). The returned bool
// reports whether the resource is managed (found in the managed load).
func specContent(ctx context.Context, prov provider.Provider, router provider.TypeRouter, resourceType, id string) (any, bool, error) {
	managed, err := prov.LoadResourcesFromRemote(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("loading remote resources: %w", err)
	}
	if managed == nil {
		managed = resources.NewRemoteResources()
	}

	// Managed resources (those with an external id) come from LoadResourcesFromRemote.
	found := findInMap(managed.GetAll(resourceType), id)
	isManaged := found != nil

	// Fall back to unmanaged (importable) resources so `-o yaml` / describe also
	// work for a resource that only lives upstream — the table path (ListRows)
	// already surfaces those, so materialization must too. LoadImportable attaches
	// a namer-suggested external id, so the emitted spec is adopt-ready (applying
	// it with `apply -f` brings the resource under management).
	if found == nil {
		if loader, ok := prov.(provider.UnmanagedRemoteResourceLoader); ok {
			importable, err := loader.LoadImportable(ctx, namer.NewExternalIdNamer(namer.NewKebabCase()))
			if err != nil {
				return nil, false, fmt.Errorf("loading importable resources: %w", err)
			}
			if importable != nil {
				found = findInMap(importable.GetAll(resourceType), id)
			}
		}
	}
	if found == nil {
		return nil, false, fmt.Errorf("%s %q: %w", resourceType, id, ErrResourceNotFound)
	}

	coll := resources.NewRemoteResources()
	coll.Set(resourceType, map[string]*resources.RemoteResource{found.ID: found})

	idNamer := namer.NewExternalIdNamer(namer.NewKebabCase())
	// Resolve cross-resource references (e.g. a source's tracking plan) against the
	// full workspace via the router: managed dependencies resolve to a local
	// reference, unmanaged ones to an adopt-ready suggestion, and anything missing
	// degrades to a raw remote-id reference so export never hard-fails.
	refResolver := newExportRefResolver(ctx, router)

	entities, err := prov.FormatForExport(coll, idNamer, refResolver)
	if err != nil {
		return nil, false, fmt.Errorf("materializing spec for %s %q: %w", resourceType, id, err)
	}
	if len(entities) == 0 {
		return nil, false, fmt.Errorf("%s %q: %w", resourceType, id, ErrResourceNotFound)
	}

	return entities[0].Content, isManaged, nil
}
