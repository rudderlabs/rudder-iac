// Package resolve implements conflict detection for `import workspace --merge`.
// It matches unmanaged remote resources against the local project graph using
// per-resource-type matcher functions registered by providers. Matched remotes
// adopt the local resource identity so the import produces a manifest link
// instead of a duplicate spec.
package resolve

import (
	"sort"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

var log = logger.New("resolve")

// MatchContext carries everything a matcher may need. A struct rather than
// positional parameters so the matcher signature never churns: RemoteGraph is
// included from day one even though most matchers only use LocalGraph, and
// Importable lets ordered matchers consult earlier matches (e.g. a child
// resource looking up its parent's match).
type MatchContext struct {
	// LocalGraph is the project's resource graph built from local specs.
	LocalGraph *resources.Graph
	// RemoteGraph is the managed remote state graph.
	RemoteGraph *resources.Graph
	// Importable is the in-flight collection of unmanaged remote resources.
	Importable *resources.RemoteResources
}

// MatcherFunc reports which local project resource uniquely matches the given
// remote resource, or nil when there is no match.
type MatcherFunc func(ctx MatchContext, r *resources.RemoteResource) *resources.Resource

// Matcher pairs a resource type with its uniqueness-match function. Providers
// return matchers in resolution order — parents before children — so matchers
// for child types can rely on parent matches being recorded already.
type Matcher struct {
	ResourceType string
	Match        MatcherFunc
}

// MatcherProvider is implemented by providers that participate in merge
// conflict detection. Providers that don't implement it are simply skipped —
// their resources keep the namer-generated identity.
type MatcherProvider interface {
	ResourceMatchers() []Matcher
}

// Run executes the matchers against the importable collection. A match adopts
// the local resource's identity: Matched is set, ExternalID becomes the local
// ID and the Reference's trailing ID segment is rewritten. First (sorted)
// remote to match a local resource claims it; later remotes matching the same
// local keep the namer identity they already carry.
func Run(ctx MatchContext, matchers []Matcher) {
	for _, m := range matchers {
		remotes := ctx.Importable.GetAll(m.ResourceType)
		if len(remotes) == 0 {
			continue
		}

		// Sort remote IDs so first-wins claiming is deterministic.
		ids := make([]string, 0, len(remotes))
		for id := range remotes {
			ids = append(ids, id)
		}
		sort.Strings(ids)

		claimed := make(map[string]bool, len(remotes))
		for _, id := range ids {
			remote := remotes[id]
			local := m.Match(ctx, remote)
			if local == nil || claimed[local.ID()] {
				continue
			}

			claimed[local.ID()] = true
			remote.Matched = local
			remote.ExternalID = local.ID()
			remote.Reference = rewriteTrailingID(remote.Reference, local.ID())
			log.Debug("matched remote resource to local project resource",
				"type", m.ResourceType, "remoteID", remote.ID, "localURN", local.URN())
		}
	}
}

// MatchByData finds the local resource of the given type whose Data() map
// satisfies the predicate. For handlers that populate resource data maps
// (datacatalog, event-stream, retl).
func MatchByData(g *resources.Graph, resourceType string, pred func(data resources.ResourceData) bool) (*resources.Resource, bool) {
	return matchSorted(g, resourceType, func(r *resources.Resource) bool {
		return pred(r.Data())
	})
}

// MatchByRawData finds the local resource of the given type whose RawData()
// satisfies the predicate. For BaseHandler-backed handlers, whose typed
// resource structs live in RawData and whose Data() maps are empty.
func MatchByRawData(g *resources.Graph, resourceType string, pred func(raw any) bool) (*resources.Resource, bool) {
	return matchSorted(g, resourceType, func(r *resources.Resource) bool {
		return pred(r.RawData())
	})
}

// matchSorted returns the first predicate match iterating candidates by
// sorted ID — graph map iteration is unordered, and which local resource a
// remote links to must be stable across runs.
func matchSorted(g *resources.Graph, resourceType string, pred func(*resources.Resource) bool) (*resources.Resource, bool) {
	candidates := g.ResourcesByType(resourceType)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].ID() < candidates[j].ID()
	})

	for _, r := range candidates {
		if pred(r) {
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
