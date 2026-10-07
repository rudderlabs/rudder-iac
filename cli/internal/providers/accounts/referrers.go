package accounts

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/rudderlabs/rudder-iac/api/client"
	dgClient "github.com/rudderlabs/rudder-iac/api/client/datagraph"
	"github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/planner"
)

// retlSourceURNPrefix starts the resource type of every rETL source kind
// (retl-source-table, retl-source-sql-model). The accounts package cannot import
// those packages, because they import this one.
const retlSourceURNPrefix = "retl-source-"

// dataGraphURNPrefix starts the URN of a data graph resource. Same import cycle
// as above.
const dataGraphURNPrefix = "data-graph:"

// dataGraphPageSize is the page size for listing data graphs.
const dataGraphPageSize = 100

// DataGraphLister lists data graphs. The data graph client satisfies it.
type DataGraphLister interface {
	ListDataGraphs(ctx context.Context, req *dgClient.ListDataGraphsRequest) (*dgClient.ListDataGraphsResponse, error)
}

// RETLSourceLister lists rETL sources. *retl.RudderRETLStore satisfies it.
type RETLSourceLister interface {
	ListRetlSources(ctx context.Context, opts ...retl.ListRetlSourcesOption) (*retl.RETLSources, error)
}

// SourceLister lists every source with its raw config. client.Client.Sources
// satisfies it.
type SourceLister interface {
	GetAll(ctx context.Context) ([]client.Source, error)
}

// DestinationLister lists every destination with its raw config.
// client.Client.Destinations satisfies it.
type DestinationLister interface {
	GetAll(ctx context.Context, opts ...client.ListDestinationsOption) ([]client.Destination, error)
}

// ReferrerClients are the API reads that find what still uses an account. A nil
// field skips that read.
type ReferrerClients struct {
	RETLSources  RETLSourceLister
	Sources      SourceLister
	Destinations DestinationLister
	DataGraphs   DataGraphLister
}

// accountReference is one workspace resource that still points at an account.
type accountReference struct {
	kind  string // "rETL source", "source", "destination" or "data graph"
	id    string
	name  string
	field string // where the account id was found, for the message
	// managedByCLI is true for an rETL source or data graph the CLI created (it
	// has an external id).
	managedByCLI bool
}

// CheckPlan refuses a plan that removes an account something else still uses,
// before the plan is shown. The backend refuses the same delete mid-apply, after
// earlier operations have run, and names the dependents without saying which the
// CLI can remove. Reading them here costs API calls only when the plan removes an
// account.
//
// The account's remote id is not in the plan: the planner's graph carries the
// spec input, and the id lives in the syncer's state. The id is resolved by
// listing accounts with an external id and matching the local id, which is the
// same match MapRemoteToState makes.
//
// What each API exposes, read from the client types:
//
//	rETL sources   RETLSource.AccountID, set for table and SQL-model sources
//	               alike, so config.accountId and the sql_models row are covered.
//	sources        client.Source.Config is raw, so config.rudderAccountId,
//	               config.accountId, config.rudderDeleteAccountId and the singer
//	               config.config.credentials values are read from it. The
//	               event-stream source listing drops config and is not used. The
//	               singer credential key name varies by source type, so any string
//	               value under credentials equal to the account id counts.
//	destinations   client.Destination.Config is raw: config.rudderAccountId and
//	               config.rudderDeleteAccountId.
//	data graphs    DataGraph.AccountID, mirroring the backend's usedByDataGraph,
//	               which matches the data graph's accountId column.
func (p *Provider) CheckPlan(ctx context.Context, plan *planner.Plan) error {
	removed := removedAccountIDs(plan)
	if len(removed) == 0 || p.referrers == nil {
		return nil
	}

	accounts, err := p.store.ListAll(ctx, client.WithHasExternalID(true))
	if err != nil {
		return fmt.Errorf("listing accounts to check what uses them: %w", err)
	}
	remoteIDs := map[string]string{} // remote id -> local id
	for _, a := range accounts {
		if slices.Contains(removed, a.ExternalID) {
			remoteIDs[a.ID] = a.ExternalID
		}
	}
	if len(remoteIDs) == 0 {
		return nil
	}

	refs, err := p.findReferences(ctx, remoteIDs, deletedInPlan(plan))
	if err != nil {
		return err
	}

	var problems []string
	for _, localID := range removed {
		for _, r := range refs[localID] {
			problems = append(problems, fmt.Sprintf("%s %q (%s) uses it through %s: %s",
				r.kind, r.name, r.id, r.field, r.advice()))
		}
	}
	if len(problems) == 0 {
		return nil
	}
	return fmt.Errorf("cannot remove account that is still in use:\n  %s", strings.Join(problems, "\n  "))
}

// advice mirrors provider.ExplainBlockingAccountUsage, per referrer.
func (r accountReference) advice() string {
	if r.managedByCLI && r.kind == "data graph" {
		return "the CLI manages it, so remove it from the project in the same run, or point it at another account"
	}
	if r.managedByCLI {
		return fmt.Sprintf("the CLI created it, so re-run with %s to let this run remove it", provider.RETLSourceFlags())
	}
	return "the CLI does not manage it, so delete it in the workspace first"
}

// removedAccountIDs returns the local ids of the accounts the plan removes.
func removedAccountIDs(plan *planner.Plan) []string {
	var ids []string
	for _, urn := range plan.Diff.RemovedResources {
		if id, ok := strings.CutPrefix(urn, AccountResourceType+":"); ok {
			ids = append(ids, id)
		}
	}
	slices.Sort(ids)
	return ids
}

// deletedReferrers holds the external ids of the CLI-managed resources the same
// plan deletes. The planner orders dependents first, so these do not block.
type deletedReferrers struct {
	retlSources map[string]bool
	dataGraphs  map[string]bool
}

func deletedInPlan(plan *planner.Plan) deletedReferrers {
	deleted := deletedReferrers{retlSources: map[string]bool{}, dataGraphs: map[string]bool{}}
	for _, urn := range plan.Diff.RemovedResources {
		if rest, ok := strings.CutPrefix(urn, retlSourceURNPrefix); ok {
			if _, id, found := strings.Cut(rest, ":"); found {
				deleted.retlSources[id] = true
			}
		}
		if id, ok := strings.CutPrefix(urn, dataGraphURNPrefix); ok {
			deleted.dataGraphs[id] = true
		}
	}
	return deleted
}

// findReferences returns, per local account id, what still references it.
func (p *Provider) findReferences(ctx context.Context, remoteIDs map[string]string, deleted deletedReferrers) (map[string][]accountReference, error) {
	refs := map[string][]accountReference{}
	add := func(remoteID string, r accountReference) {
		if local, ok := remoteIDs[remoteID]; ok {
			refs[local] = append(refs[local], r)
		}
	}
	seen := map[string]bool{}

	if p.referrers.RETLSources != nil {
		list, err := p.referrers.RETLSources.ListRetlSources(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing rETL sources to check what uses the account: %w", err)
		}
		for _, s := range list.Data {
			seen[s.ID] = true
			if s.ExternalID != "" && deleted.retlSources[s.ExternalID] {
				continue
			}
			add(s.AccountID, accountReference{
				kind: "rETL source", id: s.ID, name: s.Name,
				field:        "accountId",
				managedByCLI: s.ExternalID != "",
			})
		}
	}

	if p.referrers.Sources != nil {
		sources, err := p.referrers.Sources.GetAll(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing sources to check what uses the account: %w", err)
		}
		for _, s := range sources {
			if seen[s.ID] {
				continue
			}
			for remoteID := range remoteIDs {
				if field := accountField(s.Config, remoteID, sourceAccountKeys, true); field != "" {
					add(remoteID, accountReference{kind: "source", id: s.ID, name: s.Name, field: field})
				}
			}
		}
	}

	if p.referrers.Destinations != nil {
		dests, err := p.referrers.Destinations.GetAll(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing destinations to check what uses the account: %w", err)
		}
		for _, d := range dests {
			for remoteID := range remoteIDs {
				if field := accountField(d.Config, remoteID, destinationAccountKeys, false); field != "" {
					add(remoteID, accountReference{kind: "destination", id: d.ID, name: d.Name, field: field})
				}
			}
		}
	}

	if p.referrers.DataGraphs != nil {
		for page := 1; ; page++ {
			resp, err := p.referrers.DataGraphs.ListDataGraphs(ctx, &dgClient.ListDataGraphsRequest{Page: page, PageSize: dataGraphPageSize})
			if err != nil {
				return nil, fmt.Errorf("listing data graphs to check what uses the account: %w", err)
			}
			for _, g := range resp.Data {
				if g.ExternalID != "" && deleted.dataGraphs[g.ExternalID] {
					continue
				}
				add(g.AccountID, accountReference{
					kind: "data graph", id: g.ID, name: g.ID,
					field:        "accountId",
					managedByCLI: g.ExternalID != "",
				})
			}
			if resp.Paging.Next == "" {
				break
			}
		}
	}
	return refs, nil
}

var (
	sourceAccountKeys      = []string{"rudderAccountId", "accountId", "rudderDeleteAccountId"}
	destinationAccountKeys = []string{"rudderAccountId", "rudderDeleteAccountId"}
)

// accountField returns the config path that holds accountID, or "" if none does.
// With singer set, any string under config.config.credentials counts.
func accountField(raw json.RawMessage, accountID string, keys []string, singer bool) string {
	if len(raw) == 0 {
		return ""
	}
	var cfg map[string]any
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return ""
	}
	for _, k := range keys {
		if v, _ := cfg[k].(string); v == accountID {
			return "config." + k
		}
	}
	if !singer {
		return ""
	}
	inner, _ := cfg["config"].(map[string]any)
	creds, _ := inner["credentials"].(map[string]any)
	for _, k := range sortedKeys(creds) {
		if v, _ := creds[k].(string); v == accountID {
			return "config.config.credentials." + k
		}
	}
	return ""
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}
