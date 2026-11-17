package destination_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/writer"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/accounts"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/differ"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/planner"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// adsTestDefinition mirrors the shape of the three real destinations that take
// an integration account: one required rudder_account_id plus ordinary keys.
// The type is the real google_adwords_offline_conversions, because whether
// references are supported is a property of the type (accounts.SupportsReference).
func adsTestDefinition() *definitions.DestinationDefinition {
	return &definitions.DestinationDefinition{
		Type:        "google_adwords_offline_conversions",
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
		Type:              "google_adwords_offline_conversions",
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
// A capable type whose account nobody has adopted: the account is absent from
// the remote collection, so the literal stays on the remote side and matches the
// literal a spec carries. This is the state of every pipeline today, so the
// lookup miss must not fail the mapping.
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

// --- per-definition capability: import writes the form the type supports ---

// nonAdoptableRegistry mirrors linkedin_ads and bingads_offline_conversions:
// the key exists, but the CLI cannot adopt an account for the type, so there is
// no account spec to reference.
func nonAdoptableRegistry(t *testing.T) *definitions.Registry {
	t.Helper()
	def := adsTestDefinition()
	def.Type = "LINKEDIN_LIKE"
	def.APIType = "LINKEDIN_LIKE"
	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(def))
	return registry
}

func TestAccountRef_CapabilityIsPerDefinition(t *testing.T) {
	t.Parallel()

	assert.True(t, accounts.SupportsReference("google_adwords_offline_conversions"))
	assert.False(t, accounts.SupportsReference("linkedin_ads"))
	assert.False(t, accounts.SupportsReference("bingads_offline_conversions"))
	// A RETL warehouse account's type collides with a destination type name;
	// those accounts are not a destination's rudder_account_id.
	assert.False(t, accounts.SupportsReference("postgres"))
}

// A reference on a type that cannot support one is refused at load, rather than
// accepted into a config where nothing can ever resolve it.
func TestAccountRef_RefusedOnATypeThatCannotAdopt(t *testing.T) {
	t.Parallel()

	h := destination.NewHandler(nil, nonAdoptableRegistry(t))
	spec := adsSpec("#account:google-ads-prod")
	spec.Type = "LINKEDIN_LIKE"

	_, err := h.Impl.ExtractResourcesFromSpec("x", spec)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "takes the account's ID, not a reference")
}

// Remote state for such a type keeps the literal, so a literal spec matches.
func TestAccountRef_NonAdoptableTypeKeepsTheLiteralOnBothSides(t *testing.T) {
	t.Parallel()

	h := destination.NewHandler(nil, nonAdoptableRegistry(t))
	remote := &destination.RemoteDestination{Destination: &client.Destination{
		ID: "dst-1", ExternalID: "ads-prod", Name: "Ads Production",
		Type: "LINKEDIN_LIKE", Version: 1, IsEnabled: true,
		Config: []byte(`{"rudderAccountId":"remote-acct-1","customerId":"cust-1"}`),
	}}
	// Even with the account managed, the type cannot reference it.
	resolver := urnResolver{accountURNByID: map[string]string{
		"remote-acct-1": accountURN("google-ads-prod"),
	}}

	fromRemote, _, err := h.Impl.MapRemoteToState(remote, resolver)
	require.NoError(t, err)
	assert.Equal(t, "remote-acct-1", fromRemote.Config["rudder_account_id"])
}

// --- the drift case that import used to produce ---

// Export writes a reference for a capable type, so the spec import writes
// matches the form remote state is mapped in.
func TestAccountRef_ExportWritesAReferenceForACapableType(t *testing.T) {
	t.Parallel()

	h := destination.NewHandler(nil, adsRegistry(t))
	remote := &destination.RemoteDestination{Destination: &client.Destination{
		ID: "dst-1", ExternalID: "ads-prod", Name: "Ads Production", WorkspaceID: "ws-1",
		Type: "ADS", Version: 1, IsEnabled: true,
		Config: []byte(`{"rudderAccountId":"remote-acct-1","customerId":"cust-1"}`),
	}}

	entities, _, err := h.Impl.FormatForExport(
		map[string]*destination.RemoteDestination{"ads-prod": remote},
		nil,
		accountRefResolver{byRemoteID: map[string]string{"remote-acct-1": "#account:google-ads-prod"}},
	)
	require.NoError(t, err)
	require.Len(t, entities, 1)

	config := exportedConfig(t, entities[0])
	assert.Equal(t, "#account:google-ads-prod", config["rudder_account_id"],
		"an imported spec must name the account spec written beside it")
}

// An account that resolves to nothing has no reference to write. For a capable
// type that means a definition the CLI does not recognise, such as an account
// from the legacy role-keyed OAuth flow: an unadopted account of a registered
// definition is in the importable set and does resolve.
func TestAccountRef_ExportKeepsTheLiteralWhenNothingResolves(t *testing.T) {
	t.Parallel()

	h := destination.NewHandler(nil, adsRegistry(t))
	remote := &destination.RemoteDestination{Destination: &client.Destination{
		ID: "dst-1", ExternalID: "ads-prod", Name: "Ads Production", WorkspaceID: "ws-1",
		Type: "ADS", Version: 1, IsEnabled: true,
		Config: []byte(`{"rudderAccountId":"remote-acct-1","customerId":"cust-1"}`),
	}}

	entities, _, err := h.Impl.FormatForExport(
		map[string]*destination.RemoteDestination{"ads-prod": remote},
		nil,
		accountRefResolver{},
	)
	require.NoError(t, err)
	require.Len(t, entities, 1)
	assert.Equal(t, "remote-acct-1", exportedConfig(t, entities[0])["rudder_account_id"])
}

type accountRefResolver struct {
	byRemoteID map[string]string
}

func (r accountRefResolver) ResolveToReference(entityType string, remoteID string) (string, error) {
	if entityType != accounts.AccountResourceType {
		return "", errors.New("unexpected entity type " + entityType)
	}
	if ref, ok := r.byRemoteID[remoteID]; ok {
		return ref, nil
	}
	return "", errors.New("resource not present in resources collection")
}

func exportedConfig(t *testing.T, entity writer.FormattableEntity) map[string]any {
	t.Helper()
	spec, ok := entity.Content.(*specs.Spec)
	require.True(t, ok, "export content should be a spec, got %T", entity.Content)
	config, ok := spec.Spec["config"].(map[string]any)
	require.True(t, ok, "spec has no config map")
	return config
}

// --- the residual case: a literal left in a spec whose account is adopted ---

func destinationOp(t *testing.T, dest *destination.DestinationResource) *planner.Operation {
	t.Helper()
	return &planner.Operation{
		Type: planner.Update,
		Resource: resources.NewResource(
			dest.ID,
			destination.DestinationResourceType,
			resources.ResourceData{},
			[]string{},
			resources.WithRawData(dest),
		),
	}
}

func stateWithAdoptedAccount(externalID, remoteID string) *state.State {
	st := state.EmptyState()
	st.AddResource(&state.ResourceState{
		ID:        externalID,
		Type:      accounts.AccountResourceType,
		OutputRaw: &accounts.AccountState{ID: remoteID},
	})
	return st
}

// Remote state maps an adopted account as a reference, so a literal in the spec
// is not equal to it. Left alone that rewrites the destination with an identical
// config on every apply, which is what import used to set up.
func TestCheckPlan_RefusesALiteralNamingAnAdoptedAccount(t *testing.T) {
	t.Parallel()

	p := destination.NewProvider(nil, adsRegistry(t))
	plan := &planner.Plan{Operations: []*planner.Operation{destinationOp(t, &destination.DestinationResource{
		ID:                "ads-prod",
		Type:              "google_adwords_offline_conversions",
		DefinitionVersion: 1,
		Config:            map[string]any{"rudder_account_id": "remote-acct-1"},
	})}}

	err := p.CheckPlan(context.Background(), plan, stateWithAdoptedAccount("google-ads-prod", "remote-acct-1"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "#account:google-ads-prod")
	assert.Contains(t, err.Error(), "every apply")
}

// The reference form is the one import writes and the one that compares equal.
func TestCheckPlan_AllowsTheReferenceForm(t *testing.T) {
	t.Parallel()

	p := destination.NewProvider(nil, adsRegistry(t))
	plan := &planner.Plan{Operations: []*planner.Operation{destinationOp(t, &destination.DestinationResource{
		ID:                "ads-prod",
		Type:              "google_adwords_offline_conversions",
		DefinitionVersion: 1,
		Config: map[string]any{
			"rudder_account_id": &resources.PropertyRef{URN: accountURN("google-ads-prod"), Property: "id"},
		},
	})}}

	assert.NoError(t, p.CheckPlan(context.Background(), plan,
		stateWithAdoptedAccount("google-ads-prod", "remote-acct-1")))
}

// A literal naming an account the project does not manage is the documented,
// back-compatible form that 331 orgs use. It must stay allowed.
func TestCheckPlan_AllowsALiteralForAnUnmanagedAccount(t *testing.T) {
	t.Parallel()

	p := destination.NewProvider(nil, adsRegistry(t))
	plan := &planner.Plan{Operations: []*planner.Operation{destinationOp(t, &destination.DestinationResource{
		ID:                "ads-prod",
		Type:              "google_adwords_offline_conversions",
		DefinitionVersion: 1,
		Config:            map[string]any{"rudder_account_id": "some-other-account"},
	})}}

	assert.NoError(t, p.CheckPlan(context.Background(), plan,
		stateWithAdoptedAccount("google-ads-prod", "remote-acct-1")))
}

// A type that cannot adopt an account can only ever use a literal, so the same
// literal must not be refused there.
func TestCheckPlan_AllowsALiteralOnATypeThatCannotAdopt(t *testing.T) {
	t.Parallel()

	p := destination.NewProvider(nil, nonAdoptableRegistry(t))
	plan := &planner.Plan{Operations: []*planner.Operation{destinationOp(t, &destination.DestinationResource{
		ID:                "linkedin-prod",
		Type:              "LINKEDIN_LIKE",
		DefinitionVersion: 1,
		Config:            map[string]any{"rudder_account_id": "remote-acct-1"},
	})}}

	assert.NoError(t, p.CheckPlan(context.Background(), plan,
		stateWithAdoptedAccount("google-ads-prod", "remote-acct-1")))
}
