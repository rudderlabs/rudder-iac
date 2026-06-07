package resourceops

import (
	"context"

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
