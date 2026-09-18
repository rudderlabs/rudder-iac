package connection

import (
	"fmt"
	"regexp"
	"strings"

	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/sqlmodel"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/table"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

// SourceKind is one rETL source kind a connection may reference: how it is
// written in a spec, how it appears in the resource graph, and what the API
// calls it.
type SourceKind struct {
	Kind         string                // spec reference kind, e.g. retl-source-sql-model
	ResourceType string                // graph resource type
	SourceType   retlClient.SourceType // API sourceType
}

// SourceKinds are the rETL source kinds a connection may reference. Adding one
// is more than a line here: the kind's handler must publish the shared source
// keys below plus an "id" in its output, the warehouse table must cover the
// source definitions it accepts, and both need tests. Audience sources are
// absent because none of that exists for them yet.
//
// The table row is safe to register unconditionally even though the kind is
// gated behind retlTableSupport. This table only decides which references a
// connection may *name*; whether the referenced resource exists is the source
// handler's business. With the flag off the table spec kind fails to load in
// its own right, and the connection's reference is then reported as
// unresolved — two errors, the first of which names the flag. Narrowing this
// list to the enabled kinds is DEX-826's job, not a correctness fix here; see
// the note in the PR.
var SourceKinds = []SourceKind{
	{Kind: sqlmodel.ResourceKind, ResourceType: sqlmodel.ResourceType, SourceType: retlClient.ModelSourceType},
	{Kind: table.ResourceKind, ResourceType: table.ResourceType, SourceType: retlClient.TableSourceType},
}

// The graph data every rETL source handler publishes about its source,
// whatever its kind, so connection validation can read a source's warehouse,
// primary key and enabled flag without knowing which kind it is. Aliased to
// the SQL model handler's keys rather than restated, so renaming one there
// breaks the build instead of a test.
const (
	SourceDefinitionKey = sqlmodel.SourceDefinitionKey
	SourcePrimaryKeyKey = sqlmodel.PrimaryKeyKey
	SourceEnabledKey    = sqlmodel.EnabledKey
)

func SourceKindByKind(kind string) (SourceKind, bool) {
	for _, sk := range SourceKinds {
		if sk.Kind == kind {
			return sk, true
		}
	}
	return SourceKind{}, false
}

func SourceKindByResourceType(resourceType string) (SourceKind, bool) {
	for _, sk := range SourceKinds {
		if sk.ResourceType == resourceType {
			return sk, true
		}
	}
	return SourceKind{}, false
}

func SourceKindBySourceType(sourceType retlClient.SourceType) (SourceKind, bool) {
	for _, sk := range SourceKinds {
		if sk.SourceType == sourceType {
			return sk, true
		}
	}
	return SourceKind{}, false
}

// parseSourceRef turns a scalar "#<kind>:<id>" reference into a PropertyRef on
// the referenced kind's resource type. rETL source handlers put a literal "id"
// in their output data — the SQL model handler supplies the remote source id
// there — so the plain Property form resolves without a typed Resolve func.
// A reference of the wrong family and a malformed one fail differently: only
// the first can name the kind the author actually wrote.
func parseSourceRef(ref string) (*resources.PropertyRef, error) {
	kind, id, ok := refID(ref)
	if !ok {
		return nil, fmt.Errorf("invalid source reference %q: expected %s", ref, sourceKindRefForms())
	}
	sourceKind, ok := SourceKindByKind(kind)
	if !ok {
		return nil, fmt.Errorf("source reference %q is not a rETL source: expected %s", ref, sourceKindRefForms())
	}
	return &resources.PropertyRef{
		URN:      resources.URN(id, sourceKind.ResourceType),
		Property: "id",
	}, nil
}

// scalarRefRegex matches a well-formed scalar reference "#<kind>:<id>". The id
// side deliberately accepts any non-empty single-line value — endpoint local
// ids carry no charset restriction, but neither the kind nor the id may span
// multiple lines.
var scalarRefRegex = regexp.MustCompile(`^#([a-zA-Z0-9_-]+):(.+)$`)

// refID splits a scalar "#<kind>:<id>" reference into its parts, reporting
// whether it is well formed at all.
func refID(ref string) (kind string, id string, ok bool) {
	matches := scalarRefRegex.FindStringSubmatch(strings.TrimSpace(ref))
	if matches == nil {
		return "", "", false
	}
	return matches[1], matches[2], true
}

// sourceKindRefForms lists the reference forms a source may take, for errors.
func sourceKindRefForms() string {
	forms := make([]string, len(SourceKinds))
	for i, sk := range SourceKinds {
		forms[i] = fmt.Sprintf("#%s:<id>", sk.Kind)
	}
	return strings.Join(forms, " or ")
}
