package accounts

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/secret"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/differ"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockStore records the last request seen by each verb and returns canned data.
type mockStore struct {
	created        *client.CreateAccountRequest
	updated        *client.UpdateAccountRequest
	updatedID      string
	externalIDSet  [2]string // {id, externalID}
	createReturnID string
}

func (m *mockStore) Create(_ context.Context, req *client.CreateAccountRequest) (*client.Account, error) {
	m.created = req
	return &client.Account{ID: m.createReturnID}, nil
}
func (m *mockStore) Update(_ context.Context, id string, req *client.UpdateAccountRequest) (*client.Account, error) {
	m.updated, m.updatedID = req, id
	return &client.Account{ID: id}, nil
}
func (m *mockStore) Delete(context.Context, string) error { return nil }
func (m *mockStore) Get(context.Context, string) (*client.Account, error) {
	return &client.Account{ID: "remote-1"}, nil
}
func (m *mockStore) ListAll(context.Context, ...client.ListAccountsOption) ([]client.Account, error) {
	return nil, nil
}
func (m *mockStore) SetExternalID(_ context.Context, id, externalID string) error {
	m.externalIDSet = [2]string{id, externalID}
	return nil
}

func bqResource(id string) *AccountResource {
	cred := secret.New("svc-account-json")
	return &AccountResource{
		ID:                    id,
		Name:                  "name-" + id, // distinct from ID to prove they map separately
		AccountDefinitionName: "SOURCE_BIGQUERY",
		Config: map[string]any{
			"projectId":   "proj-123",
			"location":    "US",
			"credentials": &cred,
		},
	}
}

func TestCreate_SplitsConfigAndClaimsExternalIDInline(t *testing.T) {
	m := &mockStore{createReturnID: "remote-1"}
	h := &HandlerImpl{store: m}

	state, err := h.Create(context.Background(), bqResource("prod-bq"))
	require.NoError(t, err)
	assert.Equal(t, "remote-1", state.ID)

	// options carry non-secret keys; secret carries only credentials (revealed).
	var opts, sec map[string]any
	require.NoError(t, json.Unmarshal(m.created.Options, &opts))
	require.NoError(t, json.Unmarshal(m.created.Secret, &sec))
	assert.Equal(t, map[string]any{"projectId": "proj-123", "location": "US"}, opts)
	assert.Equal(t, map[string]any{"credentials": "svc-account-json"}, sec)

	assert.Equal(t, "SOURCE_BIGQUERY", m.created.AccountDefinitionName)
	assert.Equal(t, "name-prod-bq", m.created.Name, "display name maps from spec Name, not ID")
	// external id is claimed in the create call itself (spec ID)...
	assert.Equal(t, "prod-bq", m.created.ExternalID)
	// ...so there is no separate SetExternalID round trip.
	assert.Equal(t, [2]string{}, m.externalIDSet, "SetExternalID must not be called on create")
}

func pgResource(id string) *AccountResource {
	pw := secret.New("s3cr3t")
	return &AccountResource{
		ID:                    id,
		Name:                  "name-" + id,
		AccountDefinitionName: "SOURCE_POSTGRES",
		Config: map[string]any{
			"host":     "db.example.com",
			"dbname":   "analytics",
			"user":     "rudder",
			"port":     "5432",
			"sslMode":  "require",
			"password": &pw,
		},
	}
}

func TestCreate_SplitsPostgresConfig(t *testing.T) {
	m := &mockStore{createReturnID: "remote-1"}
	h := &HandlerImpl{store: m}

	_, err := h.Create(context.Background(), pgResource("prod-pg"))
	require.NoError(t, err)

	var opts, sec map[string]any
	require.NoError(t, json.Unmarshal(m.created.Options, &opts))
	require.NoError(t, json.Unmarshal(m.created.Secret, &sec))
	// Only password is a secret; everything else is a (non-secret) option. user is an option.
	assert.Equal(t, map[string]any{
		"host": "db.example.com", "dbname": "analytics", "user": "rudder", "port": "5432", "sslMode": "require",
	}, opts)
	assert.Equal(t, map[string]any{"password": "s3cr3t"}, sec)
	assert.Equal(t, "SOURCE_POSTGRES", m.created.AccountDefinitionName)
}

func sfKeyPairResource(id string) *AccountResource {
	pk := secret.New("dummy-snowflake-private-key")
	return &AccountResource{
		ID:                    id,
		Name:                  "name-" + id,
		AccountDefinitionName: "SOURCE_SNOWFLAKE",
		Config: map[string]any{
			"account":            "xy12345.eu-west-1",
			"dbname":             "ANALYTICS",
			"warehouse":          "COMPUTE_WH",
			"user":               "RUDDER",
			"authenticationType": "keyPair",
			"privateKey":         &pk,
		},
	}
}

func TestCreate_SplitsSnowflakeKeyPairConfig(t *testing.T) {
	m := &mockStore{createReturnID: "remote-1"}
	h := &HandlerImpl{store: m}

	_, err := h.Create(context.Background(), sfKeyPairResource("prod-sf"))
	require.NoError(t, err)

	var opts, sec map[string]any
	require.NoError(t, json.Unmarshal(m.created.Options, &opts))
	require.NoError(t, json.Unmarshal(m.created.Secret, &sec))
	// user and authenticationType are options; only privateKey is present in the secret
	// (password / privateKeyPassphrase are absent under keyPair auth).
	assert.Equal(t, map[string]any{
		"account": "xy12345.eu-west-1", "dbname": "ANALYTICS", "warehouse": "COMPUTE_WH", "user": "RUDDER", "authenticationType": "keyPair",
	}, opts)
	assert.Equal(t, map[string]any{"privateKey": "dummy-snowflake-private-key"}, sec)
	assert.Equal(t, "SOURCE_SNOWFLAKE", m.created.AccountDefinitionName)
}

func TestUpdate_RejectsDefinitionChange(t *testing.T) {
	h := &HandlerImpl{store: &mockStore{}}
	newData := bqResource("prod-bq")
	oldData := &AccountResource{AccountDefinitionName: "SOURCE_SNOWFLAKE"}

	_, err := h.Update(context.Background(), newData, oldData, &AccountState{ID: "remote-1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "account definition change is not supported")
}

func TestExtractResourcesFromSpec_UnsupportedDefinition(t *testing.T) {
	h := &HandlerImpl{store: &mockStore{}}
	_, err := h.ExtractResourcesFromSpec("f.yaml", &AccountSpec{
		ID: "x", AccountDefinitionName: "DESTINATION_SALESFORCE_OAUTH",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported account definition")
}

// Every definition an account spec may name has a type, so a source that
// references the account can always be checked against it.
func TestDefinitionType_CoversRegisteredDefinitions(t *testing.T) {
	for name := range registeredAccounts {
		_, ok := DefinitionType(name)
		assert.True(t, ok, "account definition %s has no type", name)
	}

	got, ok := DefinitionType("SOURCE_POSTGRES")
	assert.True(t, ok)
	assert.Equal(t, "postgres", got)

	_, ok = DefinitionType("DESTINATION_SALESFORCE_OAUTH")
	assert.False(t, ok)
}

func TestMapRemoteToState_SecretIsUnknown(t *testing.T) {
	h := &HandlerImpl{store: &mockStore{}}
	acc := &client.Account{ID: "remote-1", ExternalID: "prod-bq", Name: "Prod BQ", Options: json.RawMessage(`{"projectId":"p"}`)}
	acc.Definition.Name = "SOURCE_BIGQUERY"

	res, state, err := h.MapRemoteToState(&RemoteAccount{Account: acc}, nil)
	require.NoError(t, err)
	assert.Equal(t, "prod-bq", res.ID)
	assert.Equal(t, "Prod BQ", res.Name, "display name maps back from the remote")
	assert.Equal(t, "remote-1", state.ID)
	assert.Equal(t, "p", res.Config["projectId"])

	cred, ok := res.Config["credentials"].(*secret.String)
	require.True(t, ok, "credentials should be wrapped as *secret.String")
	assert.True(t, cred.IsUnknown(), "remote secret must be unknown so it always diffs")
}

func TestMapRemoteToState_SeedsOnlyTheAuthModesSecrets(t *testing.T) {
	h := &HandlerImpl{store: &mockStore{}}

	res, _, err := h.MapRemoteToState(sfRemote("snf", "password"), nil)
	require.NoError(t, err)

	wrapped, ok := res.Config["password"].(*secret.String)
	require.True(t, ok, "password should be wrapped as *secret.String")
	assert.True(t, wrapped.IsUnknown(), "password must be unknown so it always diffs")
	assert.NotContains(t, res.Config, "privateKey")
}

func bqRemote(externalID string, opts string) *RemoteAccount {
	acc := &client.Account{
		ID:         "remote-" + externalID,
		ExternalID: externalID,
		Name:       "name-" + externalID,
		Options:    json.RawMessage(opts),
	}
	acc.Definition.Name = "SOURCE_BIGQUERY"
	return &RemoteAccount{Account: acc}
}

// Export must tokenize the secret into a per-resource "{{ .VAR }}" reference the
// user fills via a var file — the API never returns the value, so a masked
// literal would be useless. Non-secret options pass through verbatim.
func TestToExportSpecMap_TokenizesSecret(t *testing.T) {
	h := &HandlerImpl{store: &mockStore{}}

	specMap, err := h.toExportSpecMap("prod-analytics-bq", bqRemote("prod-analytics-bq", `{"project":"acme","location":"US"}`))
	require.NoError(t, err)

	config := specMap["config"].(map[string]any)
	assert.Equal(t, "{{ .PROD_ANALYTICS_BQ_CREDENTIALS }}", config["credentials"], "secret must export as a var reference")
	assert.Equal(t, "acme", config["project"])
	assert.Equal(t, "US", config["location"])
	assert.Equal(t, "prod-analytics-bq", specMap["id"])
	assert.Equal(t, "name-prod-analytics-bq", specMap["name"])
	assert.Equal(t, "SOURCE_BIGQUERY", specMap["account_definition_name"])
}

// The whole exported spec — serialized as it would be written to disk — must
// never carry a raw secret. Even a value the API happened to echo back stays
// masked.
func TestFormatForExport_NeverLeaksSecret(t *testing.T) {
	h := &HandlerImpl{store: &mockStore{}}

	entities, entries, err := h.FormatForExport(
		map[string]*RemoteAccount{
			"prod-analytics-bq": bqRemote("prod-analytics-bq", `{"project":"acme","location":"US","credentials":"leaked-key-value"}`),
		}, nil, nil)
	require.NoError(t, err)
	require.Len(t, entities, 1)
	require.Len(t, entries, 1)

	assert.Equal(t, "accounts/prod-analytics-bq.yaml", entities[0].RelativePath)

	rendered, err := json.Marshal(entities[0].Content)
	require.NoError(t, err)
	assert.NotContains(t, string(rendered), "leaked-key-value", "raw secret must never reach an exported spec")
	assert.Contains(t, string(rendered), "{{ .PROD_ANALYTICS_BQ_CREDENTIALS }}")
}

// Narrowing to the auth mode decides which secrets are seeded, not which are
// masked: a secret of the other mode that the API echoed back is still a secret.
func TestFormatForExport_NeverLeaksOffModeSecret(t *testing.T) {
	h := &HandlerImpl{store: &mockStore{}}
	remote := sfRemote("snf", "keyPair")
	remote.Options = json.RawMessage(`{"account":"xy12345","authenticationType":"keyPair","password":"leaked-password"}`)

	entities, _, err := h.FormatForExport(map[string]*RemoteAccount{"snf": remote}, nil, nil)
	require.NoError(t, err)
	require.Len(t, entities, 1)

	rendered, err := json.Marshal(entities[0].Content)
	require.NoError(t, err)
	assert.NotContains(t, string(rendered), "leaked-password", "raw secret must never reach an exported spec")
	assert.Contains(t, string(rendered), "{{ .SNF_PASSWORD }}")
}

func TestMapRemoteToState_OffModeSecretIsUnknown(t *testing.T) {
	h := &HandlerImpl{store: &mockStore{}}
	remote := sfRemote("snf", "keyPair")
	remote.Options = json.RawMessage(`{"authenticationType":"keyPair","password":"leaked-password"}`)

	res, _, err := h.MapRemoteToState(remote, nil)
	require.NoError(t, err)

	wrapped, ok := res.Config["password"].(*secret.String)
	require.True(t, ok, "an echoed off-mode secret must still be wrapped as *secret.String")
	assert.True(t, wrapped.IsUnknown())
}

func TestToExportSpecMap_UnsupportedDefinition(t *testing.T) {
	h := &HandlerImpl{store: &mockStore{}}
	acc := &client.Account{ID: "remote-x", ExternalID: "x"}
	// A non-warehouse-source definition the accounts provider does not manage.
	acc.Definition.Name = "DESTINATION_SALESFORCE_OAUTH"

	_, err := h.toExportSpecMap("x", &RemoteAccount{Account: acc})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported definition")
}

// splitConfig partitions the top-level config by exact key, so a nested secret
// key would leave its container — holding the revealed plaintext — in the
// non-secret options payload. The split must refuse rather than leak.
func TestSplitConfig_RejectsNestedSecretKey(t *testing.T) {
	const definition = "SOURCE_NESTED_TEST"
	registeredAccounts[definition] = accountDefinition{Type: "webhook", SecretKeys: []string{"headers.to"}}
	t.Cleanup(func() { delete(registeredAccounts, definition) })

	s := secret.New("plaintext-that-must-not-leak")
	m := &mockStore{createReturnID: "remote-1"}
	h := &HandlerImpl{store: m}

	_, err := h.Create(context.Background(), &AccountResource{
		ID:                    "nested",
		Name:                  "nested",
		AccountDefinitionName: definition,
		Config: map[string]any{
			"headers": []any{map[string]any{"from": "X-Api-Key", "to": &s}},
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `nested secret key "headers.to"`)
	assert.Nil(t, m.created, "nothing may reach the API")
}

// The guard above is the backstop; this is the early warning. Every registered
// account secret key must be a top-level key until splitConfig and the seeding
// loops in MapRemoteToState/toExportSpecMap become path-aware.
func TestRegisteredAccountSecretKeys_AreFlat(t *testing.T) {
	for definition, def := range registeredAccounts {
		for _, key := range def.SecretKeys {
			assert.NotContains(t, key, ".",
				"definition %q: the accounts config split does not support nested secret keys yet", definition)
		}
	}
}

// sfOptions renders a Snowflake account's remote options. An empty authType
// omits the discriminator, which is how an older account reaches the CLI.
func sfOptions(authType string) string {
	const base = `{"account":"xy12345","dbname":"ANALYTICS","warehouse":"WH","user":"RUDDER"`
	if authType == "" {
		return base + "}"
	}
	return fmt.Sprintf(`%s,"authenticationType":%q}`, base, authType)
}

func sfRemote(externalID, authType string) *RemoteAccount {
	acc := &client.Account{
		ID:         "remote-" + externalID,
		ExternalID: externalID,
		Name:       "name-" + externalID,
		Options:    json.RawMessage(sfOptions(authType)),
	}
	acc.Definition.Name = "SOURCE_SNOWFLAKE"
	return &RemoteAccount{Account: acc}
}

func TestToExportSpecMap_NarrowsSecretsToAuthMode(t *testing.T) {
	base := map[string]any{
		"account": "xy12345", "dbname": "ANALYTICS", "warehouse": "WH", "user": "RUDDER",
	}
	withMode := func(mode string, secrets map[string]any) map[string]any {
		want := maps.Clone(base)
		want["authenticationType"] = mode
		maps.Copy(want, secrets)
		return want
	}

	for _, tc := range []struct {
		name string
		mode string
		want map[string]any
	}{
		{
			name: "keyPair exports only the key-pair secrets",
			mode: "keyPair",
			want: withMode("keyPair", map[string]any{
				"privateKey":           "{{ .SNF_PRIVATEKEY }}",
				"privateKeyPassphrase": "{{ .SNF_PRIVATEKEYPASSPHRASE }}",
			}),
		},
		{
			name: "password exports only the password",
			mode: "password",
			want: withMode("password", map[string]any{"password": "{{ .SNF_PASSWORD }}"}),
		},
		{
			name: "absent mode exports as an explicit password account",
			mode: "",
			want: withMode("password", map[string]any{"password": "{{ .SNF_PASSWORD }}"}),
		},
		{
			name: "mode outside the enum keeps the full set",
			mode: "oauth-someday",
			want: withMode("oauth-someday", map[string]any{
				"password":             "{{ .SNF_PASSWORD }}",
				"privateKey":           "{{ .SNF_PRIVATEKEY }}",
				"privateKeyPassphrase": "{{ .SNF_PRIVATEKEYPASSPHRASE }}",
			}),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := &HandlerImpl{store: &mockStore{}}

			specMap, err := h.toExportSpecMap("snf", sfRemote("snf", tc.mode))
			require.NoError(t, err)

			assert.Equal(t, tc.want, specMap["config"])
		})
	}
}

// An account that predates the discriminator imports with an explicit
// "password", which the account schema requires on every update. The first
// plan must add it as a real change once, after which only the always-unknown
// secret diffs.
func TestImportedAbsentModeAccount_AddsDiscriminatorOnce(t *testing.T) {
	h := &HandlerImpl{store: &mockStore{}}

	specMap, err := h.toExportSpecMap("snf", sfRemote("snf", ""))
	require.NoError(t, err)
	config := maps.Clone(specMap["config"].(map[string]any))
	config["password"] = "from-var-file"
	local, err := h.ExtractResourcesFromSpec("snf.yaml", &AccountSpec{
		ID: "snf", Name: "name-snf", AccountDefinitionName: "SOURCE_SNOWFLAKE", Config: config,
	})
	require.NoError(t, err)

	for _, tc := range []struct {
		name           string
		remoteMode     string
		wantDiffKeys   []string
		wantSecretOnly bool
	}{
		{name: "before the first apply", remoteMode: "", wantDiffKeys: []string{"authenticationType", "password"}},
		{name: "after the first apply", remoteMode: "password", wantDiffKeys: []string{"password"}, wantSecretOnly: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			remote, _, err := h.MapRemoteToState(sfRemote("snf", tc.remoteMode), nil)
			require.NoError(t, err)

			diffs, secretOnly := differ.CompareData(remote.Config, local["snf"].Config)

			assert.ElementsMatch(t, tc.wantDiffKeys, slices.Collect(maps.Keys(diffs)))
			assert.Equal(t, tc.wantSecretOnly, secretOnly)
		})
	}
}
