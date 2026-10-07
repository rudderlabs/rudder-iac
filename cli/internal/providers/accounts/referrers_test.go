package accounts_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/api/client"
	dgClient "github.com/rudderlabs/rudder-iac/api/client/datagraph"
	"github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/accounts"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/differ"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/planner"
)

type fakeAccounts struct {
	accounts.AccountStore
	list  []client.Account
	err   error
	calls int
}

func (f *fakeAccounts) ListAll(context.Context, ...client.ListAccountsOption) ([]client.Account, error) {
	f.calls++
	return f.list, f.err
}

type fakeRETL struct {
	sources []retl.RETLSource
	err     error
	calls   int
}

func (f *fakeRETL) ListRetlSources(context.Context, ...retl.ListRetlSourcesOption) (*retl.RETLSources, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	return &retl.RETLSources{Data: f.sources}, nil
}

type fakeSources struct {
	sources []client.Source
	err     error
	calls   int
}

func (f *fakeSources) GetAll(context.Context) ([]client.Source, error) {
	f.calls++
	return f.sources, f.err
}

type fakeDestinations struct {
	dests []client.Destination
	err   error
	calls int
}

func (f *fakeDestinations) GetAll(context.Context, ...client.ListDestinationsOption) ([]client.Destination, error) {
	f.calls++
	return f.dests, f.err
}

type fakeDataGraphs struct {
	pages [][]dgClient.DataGraph
	err   error
	calls int
}

func (f *fakeDataGraphs) ListDataGraphs(_ context.Context, req *dgClient.ListDataGraphsRequest) (*dgClient.ListDataGraphsResponse, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	resp := &dgClient.ListDataGraphsResponse{}
	if req.Page-1 < len(f.pages) {
		resp.Data = f.pages[req.Page-1]
	}
	if req.Page < len(f.pages) {
		resp.Paging.Next = "next"
	}
	return resp, nil
}

type harness struct {
	graphs   *fakeDataGraphs
	accounts *fakeAccounts
	retl     *fakeRETL
	sources  *fakeSources
	dests    *fakeDestinations
	provider *accounts.Provider
}

func newHarness() *harness {
	h := &harness{
		accounts: &fakeAccounts{list: []client.Account{{ID: "remote-acc", ExternalID: "wh"}}},
		retl:     &fakeRETL{},
		sources:  &fakeSources{},
		dests:    &fakeDestinations{},
		graphs:   &fakeDataGraphs{},
	}
	h.provider = accounts.NewProvider(h.accounts, &accounts.ReferrerClients{
		RETLSources: h.retl, Sources: h.sources, Destinations: h.dests, DataGraphs: h.graphs,
	})
	return h
}

func planRemoving(urns ...string) *planner.Plan {
	return &planner.Plan{Diff: &differ.Diff{RemovedResources: urns}}
}

func (h *harness) apiCalls() int {
	return h.accounts.calls + h.retl.calls + h.sources.calls + h.dests.calls + h.graphs.calls
}

func TestCheckPlan_NoAccountRemovedMakesNoAPICalls(t *testing.T) {
	h := newHarness()

	err := h.provider.CheckPlan(context.Background(), planRemoving("retl-source-table:t1", "destination:d1"))

	require.NoError(t, err)
	assert.Zero(t, h.apiCalls())
}

func TestCheckPlan_NilReferrersSkipsTheCheck(t *testing.T) {
	store := &fakeAccounts{}
	p := accounts.NewProvider(store, nil)

	require.NoError(t, p.CheckPlan(context.Background(), planRemoving("account:wh")))
	assert.Zero(t, store.calls)
}

func TestCheckPlan_NoReferrers(t *testing.T) {
	h := newHarness()
	h.retl.sources = []retl.RETLSource{{ID: "r1", Name: "other", AccountID: "another-acc"}}
	h.sources.sources = []client.Source{{ID: "s1", Name: "web", Config: json.RawMessage(`{"rudderAccountId":"another-acc"}`)}}

	err := h.provider.CheckPlan(context.Background(), planRemoving("account:wh"))

	require.NoError(t, err)
}

func TestCheckPlan_AccountNotInWorkspaceIsSkipped(t *testing.T) {
	h := newHarness()
	h.accounts.list = nil

	require.NoError(t, h.provider.CheckPlan(context.Background(), planRemoving("account:wh")))
	assert.Zero(t, h.retl.calls+h.sources.calls+h.dests.calls+h.graphs.calls)
}

func TestCheckPlan_FindsReferrerPerShape(t *testing.T) {
	cases := []struct {
		name  string
		setup func(h *harness)
		want  []string
	}{
		{
			name: "rETL source with an account id, created outside the CLI",
			setup: func(h *harness) {
				h.retl.sources = []retl.RETLSource{{ID: "r1", Name: "orders", AccountID: "remote-acc"}}
			},
			want: []string{`rETL source "orders" (r1) uses it through accountId`, "delete it in the workspace first"},
		},
		{
			name: "rETL source the CLI created names the flags",
			setup: func(h *harness) {
				h.retl.sources = []retl.RETLSource{{ID: "r1", Name: "orders", AccountID: "remote-acc", ExternalID: "orders-ext"}}
			},
			want: []string{`rETL source "orders" (r1)`, "the CLI created it", "RUDDERSTACK_CLI_EXPERIMENTAL=true"},
		},
		{
			name: "cloud source rudderAccountId",
			setup: func(h *harness) {
				h.sources.sources = []client.Source{{ID: "s1", Name: "stripe", Config: json.RawMessage(`{"rudderAccountId":"remote-acc"}`)}}
			},
			want: []string{`source "stripe" (s1) uses it through config.rudderAccountId`, "delete it in the workspace first"},
		},
		{
			name: "source config.accountId",
			setup: func(h *harness) {
				h.sources.sources = []client.Source{{ID: "s1", Name: "tbl", Config: json.RawMessage(`{"accountId":"remote-acc"}`)}}
			},
			want: []string{"config.accountId"},
		},
		{
			name: "source rudderDeleteAccountId",
			setup: func(h *harness) {
				h.sources.sources = []client.Source{{ID: "s1", Name: "x", Config: json.RawMessage(`{"rudderDeleteAccountId":"remote-acc"}`)}}
			},
			want: []string{"config.rudderDeleteAccountId"},
		},
		{
			name: "singer source credentials",
			setup: func(h *harness) {
				h.sources.sources = []client.Source{{ID: "s1", Name: "singer", Config: json.RawMessage(`{"config":{"credentials":{"someAccountId":"remote-acc","token":"t"}}}`)}}
			},
			want: []string{"config.config.credentials.someAccountId"},
		},
		{
			name: "destination rudderAccountId",
			setup: func(h *harness) {
				h.dests.dests = []client.Destination{{ID: "d1", Name: "sheet", Config: json.RawMessage(`{"rudderAccountId":"remote-acc"}`)}}
			},
			want: []string{`destination "sheet" (d1) uses it through config.rudderAccountId`},
		},
		{
			name: "destination rudderDeleteAccountId",
			setup: func(h *harness) {
				h.dests.dests = []client.Destination{{ID: "d1", Name: "sheet", Config: json.RawMessage(`{"rudderDeleteAccountId":"remote-acc"}`)}}
			},
			want: []string{"config.rudderDeleteAccountId"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness()
			tc.setup(h)

			err := h.provider.CheckPlan(context.Background(), planRemoving("account:wh"))

			require.Error(t, err)
			for _, want := range tc.want {
				assert.Contains(t, err.Error(), want)
			}
		})
	}
}

func TestCheckPlan_DataGraph(t *testing.T) {
	t.Run("found on a later page, created outside the CLI", func(t *testing.T) {
		h := newHarness()
		h.graphs.pages = [][]dgClient.DataGraph{
			{{ID: "dg-other", AccountID: "another-acc"}},
			{{ID: "dg-1", AccountID: "remote-acc"}},
		}

		err := h.provider.CheckPlan(context.Background(), planRemoving("account:wh"))

		require.Error(t, err)
		assert.Contains(t, err.Error(), `data graph "dg-1" (dg-1) uses it through accountId`)
		assert.Contains(t, err.Error(), "delete it in the workspace first")
		assert.NotContains(t, err.Error(), "dg-other")
	})

	t.Run("CLI managed names no rETL flags", func(t *testing.T) {
		h := newHarness()
		h.graphs.pages = [][]dgClient.DataGraph{{{ID: "dg-1", AccountID: "remote-acc", ExternalID: "graph"}}}

		err := h.provider.CheckPlan(context.Background(), planRemoving("account:wh"))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "the CLI manages it")
		assert.NotContains(t, err.Error(), "RUDDERSTACK_CLI_EXPERIMENTAL")
	})

	t.Run("none found", func(t *testing.T) {
		h := newHarness()
		h.graphs.pages = [][]dgClient.DataGraph{{{ID: "dg-1", AccountID: "another-acc"}}}

		require.NoError(t, h.provider.CheckPlan(context.Background(), planRemoving("account:wh")))
	})

	t.Run("deleted in the same plan does not block", func(t *testing.T) {
		h := newHarness()
		h.graphs.pages = [][]dgClient.DataGraph{{{ID: "dg-1", AccountID: "remote-acc", ExternalID: "graph"}}}

		require.NoError(t, h.provider.CheckPlan(context.Background(), planRemoving("account:wh", "data-graph:graph")))
	})
}

func TestCheckPlan_NamesEveryReferrer(t *testing.T) {
	h := newHarness()
	h.retl.sources = []retl.RETLSource{{ID: "r1", Name: "orders", AccountID: "remote-acc"}}
	h.dests.dests = []client.Destination{{ID: "d1", Name: "sheet", Config: json.RawMessage(`{"rudderAccountId":"remote-acc"}`)}}

	err := h.provider.CheckPlan(context.Background(), planRemoving("account:wh"))

	require.Error(t, err)
	assert.Contains(t, err.Error(), `"orders"`)
	assert.Contains(t, err.Error(), `"sheet"`)
}

// A source listed by both the rETL and the generic source API is reported once.
func TestCheckPlan_ReportsASourceOnce(t *testing.T) {
	h := newHarness()
	h.retl.sources = []retl.RETLSource{{ID: "r1", Name: "orders", AccountID: "remote-acc"}}
	h.sources.sources = []client.Source{{ID: "r1", Name: "orders", Config: json.RawMessage(`{"accountId":"remote-acc"}`)}}

	err := h.provider.CheckPlan(context.Background(), planRemoving("account:wh"))

	require.Error(t, err)
	assert.Equal(t, 1, strings.Count(err.Error(), `"orders"`))
}

func TestCheckPlan_ReferrerDeletedInTheSamePlanDoesNotBlock(t *testing.T) {
	h := newHarness()
	h.retl.sources = []retl.RETLSource{
		{ID: "r1", Name: "orders", AccountID: "remote-acc", ExternalID: "orders-ext"},
		{ID: "r2", Name: "users", AccountID: "remote-acc", ExternalID: "users-ext"},
	}

	err := h.provider.CheckPlan(context.Background(),
		planRemoving("account:wh", "retl-source-table:orders-ext"))

	require.Error(t, err, "users-ext is not deleted, so it still blocks")
	assert.NotContains(t, err.Error(), `"orders"`)
	assert.Contains(t, err.Error(), `"users"`)

	err = h.provider.CheckPlan(context.Background(),
		planRemoving("account:wh", "retl-source-table:orders-ext", "retl-source-sql-model:users-ext"))
	require.NoError(t, err)
}

func TestCheckPlan_APIErrorsSurface(t *testing.T) {
	boom := errors.New("boom")
	cases := []struct {
		name  string
		setup func(h *harness)
		want  string
	}{
		{"accounts", func(h *harness) { h.accounts.err = boom }, "listing accounts"},
		{"rETL sources", func(h *harness) { h.retl.err = boom }, "listing rETL sources"},
		{"sources", func(h *harness) { h.sources.err = boom }, "listing sources"},
		{"destinations", func(h *harness) { h.dests.err = boom }, "listing destinations"},
		{"data graphs", func(h *harness) { h.graphs.err = boom }, "listing data graphs"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness()
			tc.setup(h)

			err := h.provider.CheckPlan(context.Background(), planRemoving("account:wh"))

			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.want)
			assert.ErrorIs(t, err, boom)
		})
	}
}
