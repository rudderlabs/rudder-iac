// Package importmatcher implements conflict detection for
// `import workspace --merge`. It matches unmanaged remote resources against
// the local project graph using per-resource-type matcher functions exposed by
// providers. Matched remotes adopt the local resource identity so the import
// produces a manifest link instead of a duplicate spec.
package importmatcher

import (
	"fmt"
	"sort"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

var log = logger.New("importmatcher")

// Scope is the data universe a matcher consults — deliberately NOT named
// "Context": in Go, ctx/Context signals context.Context (cancellation), which
// this is not, and the name would clash if a matcher ever needs a real one.
// A struct so the matcher signature never churns: RemoteGraph is included from
// day one even where unused, and Importable lets ordered matchers consult
// earlier matches (e.g. a child resource looking up its parent's match).
type Scope struct {
	// LocalGraph is the project's resource graph built from local specs.
	LocalGraph *resources.Graph
	// RemoteGraph is the managed remote state graph.
	RemoteGraph *resources.Graph
	// Importable is the in-flight collection of unmanaged remote resources.
	Importable *resources.RemoteResources
}

// Func reports which local project resource uniquely matches the given remote
// resource, or nil when there is no match.
type Func func(scope Scope, r *resources.RemoteResource) *resources.Resource

// Matcher pairs a resource type with its uniqueness-match function. Providers
// return matchers in resolution order — parents before children — so matchers
// for child types can rely on parent matches being recorded already.
type Matcher struct {
	ResourceType string
	Match        Func
}

// Ref is a cross-resource reference a remote payload holds: the referenced
// entity's resource type and the upstream (remote) ID it points at.
type Ref struct {
	EntityType string
	RemoteID   string
}

// RefsFunc reports the cross-resource references an importable resource holds.
// It reads r.Data (the parsed upstream payload) and returns every reference it
// contains — the same IDs the handler resolves during FormatForExport. It
// returns nil when the payload references nothing.
type RefsFunc func(r *resources.RemoteResource) []Ref

// RefLister pairs a resource type with the function listing its importables'
// references. `import workspace --merge` uses these to detect, before matching
// or file generation, when an importable references a resource that is deleted
// locally but whose deletion is not yet applied.
type RefLister struct {
	ResourceType string
	Refs         RefsFunc
}

// MultipleClaimed reports two remote resources that both matched the same
// local resource: ClaimedByRemoteID won (matched first, sorted), RemoteID is
// the extra. Matcher predicates are meant to mirror upstream uniqueness, so a
// claim collision signals flawed matching or dirty upstream data — the caller
// is expected to fail fast on any of these.
type MultipleClaimed struct {
	ResourceType      string
	LocalURN          string
	ClaimedByRemoteID string
	RemoteID          string
}

func (c MultipleClaimed) String() string {
	return fmt.Sprintf("local resource %q is matched by multiple remotes %q and %q",
		c.LocalURN, c.ClaimedByRemoteID, c.RemoteID)
}

// Mark executes the matchers against the importable collection, mutating
// matched resources in place: MatchedWith is set, ExternalID becomes the local
// ID and the Reference's trailing ID segment is rewritten. The first (sorted)
// remote to match a local resource claims it; any later remote matching the
// same local is returned as a MultipleClaimed and keeps the namer identity it
// already carries, so the caller can decide (fail fast) rather than silently
// dropping it.
func Mark(scope Scope, matchers []Matcher) []MultipleClaimed {
	var multipleClaimed []MultipleClaimed
	for _, m := range matchers {
		remotes := scope.Importable.GetAll(m.ResourceType)
		if len(remotes) == 0 {
			continue
		}

		// Sort remote IDs to maintain deterministic collision reporting
		ids := make([]string, 0, len(remotes))
		for id := range remotes {
			ids = append(ids, id)
		}
		sort.Strings(ids)

		// claimedBy maps a local ID to the remote ID that first claimed it, so a
		// collision can report both remotes.
		claimedBy := make(map[string]string, len(remotes))
		for _, id := range ids {
			remote := remotes[id]
			local := m.Match(scope, remote)
			if local == nil {
				continue
			}

			if claimer, ok := claimedBy[local.ID()]; ok {
				multipleClaimed = append(multipleClaimed, MultipleClaimed{
					ResourceType:      m.ResourceType,
					LocalURN:          local.URN(),
					ClaimedByRemoteID: claimer,
					RemoteID:          remote.ID,
				})
				continue
			}

			log.Debug("matching remote resource to local project resource",
				"type", m.ResourceType, "remoteID", remote.ID, "localURN", local.URN())
			claimedBy[local.ID()] = remote.ID
			remote.MatchedWith = local
			remote.ExternalID = local.ID()
			remote.Reference = rewriteTrailingID(remote.Reference, local.ID())
		}
	}
	return multipleClaimed
}

// ByData finds the local resource of the given type whose Data() map satisfies
// matches. For handlers that populate resource data maps (datacatalog,
// event-stream, retl).
func ByData(g *resources.Graph, resourceType string, matches func(data resources.ResourceData) bool) (*resources.Resource, bool) {
	return bySorted(g, resourceType, func(r *resources.Resource) bool {
		return matches(r.Data())
	})
}

// ByRawData finds the local resource of the given type whose RawData()
// satisfies matches. For BaseHandler-backed handlers, whose typed resource
// structs live in RawData and whose Data() maps are empty.
func ByRawData(g *resources.Graph, resourceType string, matches func(raw any) bool) (*resources.Resource, bool) {
	return bySorted(g, resourceType, func(r *resources.Resource) bool {
		return matches(r.RawData())
	})
}

// bySorted returns the first match iterating candidates by sorted ID — graph
// map iteration is unordered, and which local resource a remote links to must
// be stable across runs.
func bySorted(g *resources.Graph, resourceType string, matches func(*resources.Resource) bool) (*resources.Resource, bool) {
	candidates := g.ResourcesByType(resourceType)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].ID() < candidates[j].ID()
	})

	for _, r := range candidates {
		if matches(r) {
			return r, true
		}
	}
	return nil, false
}

// rewriteTrailingID swaps the trailing ID segment of a reference with the
// local ID. Every provider reference shape ends with the external ID as the
// final ':' or '/' delimited segment (e.g. "#category:id",
// "#/transformation/transformations/id"), so the provider-specific prefix is
// preserved without providers having to expose a reference builder.
func rewriteTrailingID(reference, localID string) string {
	idx := strings.LastIndexAny(reference, ":/")
	if idx < 0 {
		return reference
	}
	return reference[:idx+1] + localID
}
