package destination_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/accounts"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/differ"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// adsTestDefinition mirrors the shape of the three real destinations that take
// an integration account: one required rudder_account_id plus ordinary keys.
func adsTestDefinition() *definitions.DestinationDefinition {
	return &definitions.DestinationDefinition{
		Type:        "ADS",
		DisplayName: "Test Ads",
		APIType:     "ADS",
		Version:     1,
		Properties: []converter.ConfigProperty{
			converter.Simple("rudderAccountId", "rudder_account_id"),
			converter.Simple("customerId", "customer_id"),
		},
		SecretKeys: []string{},
		NewConfig: func() any {
			return &struct {
				RudderAccountID string `mapstructure:"rudder_account_id" validate:"required"`
				CustomerID      string `mapstructure:"customer_id"`
			}{}
		},
		SourceTypes:     []string{"cloud"},
		ConnectionModes: map[string][]string{"cloud": {"cloud"}},
	}
}

func adsRegistry(t *testing.T) *definitions.Registry {
	t.Helper()
	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(adsTestDefinition()))
	return registry
}

func adsSpec(accountValue string) *destination.DestinationSpec {
	return &destination.DestinationSpec{
		ID:                "ads-prod",
		DisplayName:       "Ads Production",
		Type:              "ADS",
		Enabled:           true,
		DefinitionVersion: lo.ToPtr[int64](1),
		Config: map[string]any{
			"rudder_account_id": accountValue,
			"customer_id":       "cust-1",
		},
	}
}

func accountURN(id string) string {
	return resources.URN(id, accounts.AccountResourceType)
}

func TestAccountRef_SpecRefBecomesPropertyRef(t *testing.T) {
	t.Parallel()

	h := destination.NewHandler(nil, adsRegistry(t))
	extracted, err := h.Impl.ExtractResourcesFromSpec("x", adsSpec("#account:google-ads-prod"))
	require.NoError(t, err)

	ref, ok := extracted["ads-prod"].Config["rudder_account_id"].(*resources.PropertyRef)
	require.True(t, ok, "a #account: value must become a PropertyRef")
	assert.Equal(t, accountURN("google-ads-prod"), ref.URN)
	assert.Equal(t, "id", ref.Property)
	assert.False(t, ref.IsResolved)
	assert.Equal(t, "cust-1", extracted["ads-prod"].Config["customer_id"])
}

// 331 orgs have a literal account id in their specs today. That must keep
// working untouched.
func TestAccountRef_LiteralIDIsLeftAlone(t *testing.T) {
	t.Parallel()

	h := destination.NewHandler(nil, adsRegistry(t))
	extracted, err := h.Impl.ExtractResourcesFromSpec("x", adsSpec("2fS9kQwPzX7rTnV4yLbH8mJcD1e"))
	require.NoError(t, err)
	assert.Equal(t, "2fS9kQwPzX7rTnV4yLbH8mJcD1e", extracted["ads-prod"].Config["rudder_account_id"])
}

func TestAccountRef_MalformedRefIsRejected(t *testing.T) {
	t.Parallel()

	h := destination.NewHandler(nil, adsRegistry(t))
	for _, value := range []string{"#account:", "#transformation:tp-1", "#notakind:x", "#"} {
		_, err := h.Impl.ExtractResourcesFromSpec("x", adsSpec(value))
		require.Error(t, err, "value %q should be rejected", value)
		assert.Contains(t, err.Error(), "expected format #account:<id>")
	}
}

// The remote side has to produce the same ref the spec does, or a managed
// account diffs a ref against an id on every single apply.
func TestAccountRef_RemoteStateMatchesSpec(t *testing.T) {
	t.Parallel()

	h := destination.NewHandler(nil, adsRegistry(t))
	remote := &destination.RemoteDestination{Destination: &client.Destination{
		ID: "dst-1", ExternalID: "ads-prod", Name: "Ads Production",
		Type: "ADS", Version: 1, IsEnabled: true,
		Config: []byte(`{"rudderAccountId":"remote-acct-1","customerId":"cust-1"}`),
	}}
	resolver := urnResolver{accountURNByID: map[string]string{
		"remote-acct-1": accountURN("google-ads-prod"),
	}}

	fromRemote, _, err := h.Impl.MapRemoteToState(remote, resolver)
	require.NoError(t, err)

	fromSpec, err := h.Impl.ExtractResourcesFromSpec("x", adsSpec("#account:google-ads-prod"))
	require.NoError(t, err)

	remoteRef, ok := fromRemote.Config["rudder_account_id"].(*resources.PropertyRef)
	require.True(t, ok, "a managed remote account must map back to a PropertyRef")
	assert.Equal(t, accountURN("google-ads-prod"), remoteRef.URN)

	// The property that matters is that the differ sees no change. It reaches a
	// ref nested in the config map and compares it by URN, so this is the real
	// no-drift check rather than a struct comparison (which the Resolve closure
	// would fail on regardless).
	propertyDiffs, _ := differ.CompareData(
		resources.ResourceData{"config": fromRemote.Config},
		resources.ResourceData{"config": fromSpec["ads-prod"].Config},
	)
	assert.Empty(t, propertyDiffs, "a managed account must not diff between spec and remote")
}

// An account the CLI does not manage has no URN to point at, so both sides keep
// the literal id. That is equally drift-free.
func TestAccountRef_UnmanagedRemoteAccountStaysLiteral(t *testing.T) {
	t.Parallel()

	h := destination.NewHandler(nil, adsRegistry(t))
	remote := &destination.RemoteDestination{Destination: &client.Destination{
		ID: "dst-1", ExternalID: "ads-prod", Name: "Ads Production",
		Type: "ADS", Version: 1, IsEnabled: true,
		Config: []byte(`{"rudderAccountId":"remote-acct-1","customerId":"cust-1"}`),
	}}

	fromRemote, _, err := h.Impl.MapRemoteToState(remote, urnResolver{})
	require.NoError(t, err)
	assert.Equal(t, "remote-acct-1", fromRemote.Config["rudder_account_id"])
}

// End to end: extract the ref, let the apply framework resolve it against
// account state, then confirm the API payload carries the remote account id.
func TestAccountRef_ResolvedRefReachesTheAPIAsAnID(t *testing.T) {
	t.Parallel()

	var createBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		createBody = string(body)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"destination":{"id":"dst-1","externalId":"ads-prod",
			"name":"Ads Production","type":"ADS","enabled":true,
			"config":{"rudderAccountId":"remote-acct-1","customerId":"cust-1"}}}`))
	}))
	t.Cleanup(srv.Close)

	h := destination.NewHandler(newTestClient(t, srv.URL), adsRegistry(t))
	extracted, err := h.Impl.ExtractResourcesFromSpec("x", adsSpec("#account:google-ads-prod"))
	require.NoError(t, err)

	st := state.EmptyState()
	st.AddResource(&state.ResourceState{
		ID:        "google-ads-prod",
		Type:      accounts.AccountResourceType,
		OutputRaw: &accounts.AccountState{ID: "remote-acct-1"},
	})
	require.NoError(t, state.DereferenceByReflection(extracted["ads-prod"], st))

	_, err = h.Impl.Create(context.Background(), extracted["ads-prod"])
	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, json.Unmarshal([]byte(createBody), &payload))
	config := payload["config"].(map[string]any)
	assert.Equal(t, "remote-acct-1", config["rudderAccountId"],
		"the API must get the resolved account id, not the reference")
	assert.NotContains(t, createBody, "google-ads-prod")
	assert.NotContains(t, createBody, "urn")
}

// A ref the framework never resolved must fail loudly rather than send a
// nonsense account id to the API.
func TestAccountRef_UnresolvedRefFailsBeforeTheAPICall(t *testing.T) {
	t.Parallel()

	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	h := destination.NewHandler(newTestClient(t, srv.URL), adsRegistry(t))
	extracted, err := h.Impl.ExtractResourcesFromSpec("x", adsSpec("#account:google-ads-prod"))
	require.NoError(t, err)

	_, err = h.Impl.Create(context.Background(), extracted["ads-prod"])
	require.Error(t, err)
	assert.Contains(t, err.Error(), "was not resolved")
	assert.False(t, called, "no request should be sent with an unresolved reference")
}

// The graph's own config is read again after an apply, so flattening for the
// API must not mutate it.
func TestAccountRef_APIConversionLeavesTheGraphConfigAlone(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"destination":{"id":"dst-1","externalId":"ads-prod",
			"name":"Ads Production","type":"ADS","enabled":true,"config":{}}}`))
	}))
	t.Cleanup(srv.Close)

	h := destination.NewHandler(newTestClient(t, srv.URL), adsRegistry(t))
	extracted, err := h.Impl.ExtractResourcesFromSpec("x", adsSpec("#account:google-ads-prod"))
	require.NoError(t, err)

	st := state.EmptyState()
	st.AddResource(&state.ResourceState{
		ID:        "google-ads-prod",
		Type:      accounts.AccountResourceType,
		OutputRaw: &accounts.AccountState{ID: "remote-acct-1"},
	})
	require.NoError(t, state.DereferenceByReflection(extracted["ads-prod"], st))

	_, err = h.Impl.Create(context.Background(), extracted["ads-prod"])
	require.NoError(t, err)

	_, stillARef := extracted["ads-prod"].Config["rudder_account_id"].(*resources.PropertyRef)
	assert.True(t, stillARef, "the graph must still hold the reference after an apply")
}
