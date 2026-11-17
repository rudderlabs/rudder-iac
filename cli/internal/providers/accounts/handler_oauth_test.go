package accounts

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// oauthDefinition is the one OAuth account definition the CLI registers.
const oauthDefinition = "DESTINATION_GOOGLE_ADWORDS_OFFLINE_CONVERSIONS_OAUTH"

func oauthRemote(externalID string) *RemoteAccount {
	a := &client.Account{
		ID:          "remote-oauth-1",
		WorkspaceID: "ws-1",
		ExternalID:  externalID,
		Name:        "Google Ads (prod)",
		// The API echoes options back; an OAuth definition declares none, but a
		// stray key must not become a diff the CLI can never resolve.
		Options: json.RawMessage(`{"someRemoteKey":"value"}`),
	}
	a.Definition.Name = oauthDefinition
	a.Definition.Type = "google_adwords_offline_conversions"
	a.Definition.Category = "destination"
	return &RemoteAccount{Account: a}
}

func oauthResource(id string) *AccountResource {
	return &AccountResource{
		ID:                    id,
		Name:                  "Google Ads (prod)",
		AccountDefinitionName: oauthDefinition,
		Config:                map[string]any{},
	}
}

func TestOAuthSpecNeedsNoConfigBlock(t *testing.T) {
	t.Parallel()
	h := &HandlerImpl{store: &mockStore{}}

	got, err := h.ExtractResourcesFromSpec("accounts/google-ads.yaml", &AccountSpec{
		ID:                    "google-ads-prod",
		Name:                  "Google Ads (prod)",
		AccountDefinitionName: oauthDefinition,
		// No Config at all: the definition declares no option fields and the
		// credentials live in RudderStack's auth service.
	})
	require.NoError(t, err)
	assert.Equal(t, map[string]*AccountResource{
		"google-ads-prod": oauthResource("google-ads-prod"),
	}, got)
}

func TestOAuthCreateIsRefusedWithTheWayForward(t *testing.T) {
	t.Parallel()
	m := &mockStore{createReturnID: "remote-1"}
	h := &HandlerImpl{store: m}

	_, err := h.Create(context.Background(), oauthResource("google-ads-prod"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be created by rudder-cli")
	assert.Contains(t, err.Error(), "rudder-cli import workspace")
	assert.Nil(t, m.created, "the API refuses OAuth creation, so no request should be sent")
}

func TestOAuthUpdateIsRefusedAndNamesTheRemoteName(t *testing.T) {
	t.Parallel()
	m := &mockStore{}
	h := &HandlerImpl{store: m}

	oldData := oauthResource("google-ads-prod")
	newData := oauthResource("google-ads-prod")
	newData.Name = "Renamed in the spec"

	_, err := h.Update(context.Background(), newData, oldData, &AccountState{ID: "remote-oauth-1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be changed by rudder-cli")
	// The message has to say what to restore, since the spec is the only side
	// the user can fix.
	assert.Contains(t, err.Error(), "Google Ads (prod)")
	assert.Nil(t, m.updated, "the API refuses OAuth edits, so no request should be sent")
}

// Adopting an OAuth account is the external-id claim alone: the Update the
// non-OAuth path runs would be refused by the API, and its full-replace
// semantics would write an empty secret over a live token pair.
func TestOAuthImportClaimsExternalIDWithoutUpdating(t *testing.T) {
	t.Parallel()
	m := &mockStore{getAccount: oauthRemote("").Account}
	h := &HandlerImpl{store: m}

	state, err := h.Import(context.Background(), oauthResource("google-ads-prod"), "remote-oauth-1")
	require.NoError(t, err)
	assert.Equal(t, &AccountState{ID: "remote-oauth-1"}, state)
	assert.Equal(t, [2]string{"remote-oauth-1", "google-ads-prod"}, m.externalIDSet)
	assert.Nil(t, m.updated, "import must not push an update to an OAuth account")
}

// A managed OAuth account must produce no diff, ever: there is nothing the CLI
// can write, so any diff would provoke an Update the API refuses.
func TestOAuthRemoteStateMatchesSpecExactly(t *testing.T) {
	t.Parallel()
	h := &HandlerImpl{store: &mockStore{}}

	fromRemote, state, err := h.MapRemoteToState(oauthRemote("google-ads-prod"), nil)
	require.NoError(t, err)
	assert.Equal(t, &AccountState{ID: "remote-oauth-1"}, state)

	fromSpec, err := h.ExtractResourcesFromSpec("accounts/google-ads.yaml", &AccountSpec{
		ID:                    "google-ads-prod",
		Name:                  "Google Ads (prod)",
		AccountDefinitionName: oauthDefinition,
	})
	require.NoError(t, err)

	assert.Equal(t, fromSpec["google-ads-prod"], fromRemote,
		"spec and remote sides must be identical or the account diffs on every apply")
}

func TestOAuthExportWritesNoConfigBlock(t *testing.T) {
	t.Parallel()
	h := &HandlerImpl{store: &mockStore{}}

	spec, err := h.toExportSpecMap("google-ads-prod", oauthRemote(""))
	require.NoError(t, err)
	assert.Equal(t, map[string]any{
		"id":                      "google-ads-prod",
		"name":                    "Google Ads (prod)",
		"account_definition_name": oauthDefinition,
	}, spec)
}

// Delete is the one write the API does allow on an OAuth account.
func TestOAuthDeleteIsAllowed(t *testing.T) {
	t.Parallel()
	m := &mockStore{}
	h := &HandlerImpl{store: m}

	require.NoError(t, h.Delete(context.Background(), "google-ads-prod",
		oauthResource("google-ads-prod"), &AccountState{ID: "remote-oauth-1"}))
}

func TestOAuthConfigIsRejectedAtValidate(t *testing.T) {
	t.Parallel()

	results := validateAccountSpec("", "", nil, AccountSpec{
		ID:                    "google-ads-prod",
		Name:                  "Google Ads (prod)",
		AccountDefinitionName: oauthDefinition,
		Config:                map[string]any{"customerId": "123"},
	})
	require.Len(t, results, 1)
	assert.Equal(t, "/config", results[0].Reference)
	assert.Contains(t, results[0].Message, "'config' is not allowed")
}

func TestOAuthSpecWithoutConfigPassesValidate(t *testing.T) {
	t.Parallel()

	assert.Empty(t, validateAccountSpec("", "", nil, AccountSpec{
		ID:                    "google-ads-prod",
		Name:                  "Google Ads (prod)",
		AccountDefinitionName: oauthDefinition,
	}))
}

// Dropping validate:"required" from Config must not stop a warehouse account
// with no config from being caught; the per-definition check reports the
// specific keys instead.
func TestNonOAuthSpecWithoutConfigStillFailsValidate(t *testing.T) {
	t.Parallel()

	results := validateAccountSpec("", "", nil, AccountSpec{
		ID:                    "prod-pg",
		Name:                  "Postgres",
		AccountDefinitionName: "SOURCE_POSTGRES",
	})
	require.NotEmpty(t, results)
	var refs []string
	for _, r := range results {
		refs = append(refs, r.Reference)
	}
	assert.Contains(t, refs, "/config/host")
	assert.Contains(t, refs, "/config/password")
}

func TestOAuthDefinitionIsRecognised(t *testing.T) {
	t.Parallel()

	assert.True(t, isOAuthDefinition(oauthDefinition))
	assert.False(t, isOAuthDefinition("SOURCE_POSTGRES"))
	assert.False(t, isOAuthDefinition("DESTINATION_LINKEDIN_ADS_OAUTH"),
		"linkedin_ads has no account definition in integrations-config")

	// An OAuth definition must declare no secret keys: MapRemoteToState seeds
	// each one as present-and-unknown, which would put the account in the
	// plan's always-re-applied section and re-PUT an empty secret every run.
	keys, ok := secretKeys(oauthDefinition)
	require.True(t, ok)
	assert.Empty(t, keys)
}
