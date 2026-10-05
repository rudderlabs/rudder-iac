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
// day one even where unused, and Importable lets a matcher consult other
// remotes' matches (e.g. a child resource looking up its parent's match).
type Scope struct {
	// LocalGraph is the project's resource graph built from local specs.
	LocalGraph *resources.Graph
	// RemoteGraph is the managed remote state graph.
	RemoteGraph *resources.Graph
	// Importable is the in-flight collection of unmanaged remote resources.
	Importable *resources.RemoteResources

	// resolver matches importable remotes on demand while Mark runs; nil
	// outside Mark, where the recorded MatchedWith is the answer.
	resolver *resolver
}

// Func reports which local project resource uniquely matches the given remote
// resource, or nil when there is no match. A Func that depends on another
// remote's match reads it through MatchedLocal, ResolveLocalURN or
// EndpointURN — never that remote's MatchedWith, which is only recorded once
// its own matcher's turn comes — and returns nil when the dependency has no
// local counterpart.
type Func func(scope Scope, r *resources.RemoteResource) *resources.Resource

// Matcher pairs a resource type with its uniqueness-match function, one per
// resource type. Matcher order is immaterial: a matcher whose result depends
// on another remote's match gets it resolved on demand.
type Matcher struct {
	ResourceType string
	Match        Func
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
//
// Matches are resolved on demand, so a matcher may depend on any other
// matcher's result — including another provider's — whatever their order.
func Mark(scope Scope, matchers []Matcher) []MultipleClaimed {
	// Set on the scope matchers receive, so their dependency lookups share it.
	scope.resolver = newResolver(matchers)

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
			local := scope.resolver.match(scope, m.ResourceType, remote)
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

// resolver memoizes each importable remote's match, so a remote is matched once
// however many dependents ask for it before its own matcher's turn in Mark.
type resolver struct {
	byType  map[string]Matcher
	matched map[*resources.RemoteResource]*resources.Resource
}

func newResolver(matchers []Matcher) *resolver {
	r := &resolver{
		byType:  make(map[string]Matcher, len(matchers)),
		matched: make(map[*resources.RemoteResource]*resources.Resource),
	}
	for _, m := range matchers {
		r.byType[m.ResourceType] = m
	}
	return r
}

// match returns the local resource the importable remote matches, running its
// type's matcher at most once; a type without a matcher keeps its recorded mark.
func (r *resolver) match(scope Scope, resourceType string, remote *resources.RemoteResource) *resources.Resource {
	if local, done := r.matched[remote]; done {
		return local
	}
	m, ok := r.byType[resourceType]
	if !ok {
		return remote.MatchedWith
	}

	// Recorded as unmatched up front, so a dependency cycle leading back to this
	// remote resolves as unmatched instead of recursing forever.
	r.matched[remote] = nil
	local := m.Match(scope, remote)
	r.matched[remote] = local
	return local
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

// MatchedLocal reports what an importable remote matches: resolved on demand
// inside Mark, the recorded MatchedWith outside it. importable is false when
// remoteID is not an importable remote of that type.
func MatchedLocal(scope Scope, resourceType, remoteID string) (local *resources.Resource, importable bool) {
	if scope.Importable == nil {
		return nil, false
	}
	remote, ok := scope.Importable.GetByID(resourceType, remoteID)
	if !ok {
		return nil, false
	}
	if scope.resolver != nil {
		return scope.resolver.match(scope, resourceType, remote), true
	}
	return remote.MatchedWith, true
}

// ResolveLocalURN maps a remote resource ID of the given type to the local
// resource URN it corresponds to: either the URN of the resource it matches in
// this import (see MatchedLocal), or of one already managed locally (found by
// its import metadata's remote ID). ok is false when the ID has no local
// counterpart — an importable-but-unmatched remote, or an ID the project does
// not know. For matchers whose remote resources reference other resources by
// remote ID (datacatalog custom types, event stream connection endpoints).
func ResolveLocalURN(scope Scope, resourceType string, remoteID string) (urn string, ok bool) {
	local, importable := MatchedLocal(scope, resourceType, remoteID)
	if local != nil {
		return local.URN(), true
	}
	if importable {
		return "", false
	}
	for _, local := range scope.LocalGraph.ResourcesByType(resourceType) {
		if meta := local.ImportMetadata(); meta != nil && meta.RemoteId == remoteID {
			return local.URN(), true
		}
	}
	return "", false
}

// EndpointURN maps a resource referenced by remote ID to the local resource URN
// it corresponds to: through ResolveLocalURN — matched in this import, or
// linked by local import metadata — or, for an already-managed resource,
// straight from its externalId, which is its local resource id. ok is false
// when there is no local counterpart, so the referencing remote stays
// unmatched.
//
// It exists for matchers whose remotes are identified only by what they point
// at, such as the connection matchers' source–destination pairs: resolving both
// endpoints to local URNs is what lets them find their local counterpart.
func EndpointURN(scope Scope, resourceType string, remoteID string, externalID string) (string, bool) {
	if urn, ok := ResolveLocalURN(scope, resourceType, remoteID); ok {
		return urn, true
	}
	if externalID == "" {
		return "", false
	}
	urn := resources.URN(externalID, resourceType)
	_, ok := scope.LocalGraph.GetResource(urn)
	return urn, ok
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
