package accounts

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/secret"
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
	registeredAccountSecretKeys[definition] = []string{"headers.to"}
	t.Cleanup(func() { delete(registeredAccountSecretKeys, definition) })

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
	for definition, keys := range registeredAccountSecretKeys {
		for _, key := range keys {
			assert.NotContains(t, key, ".",
				"definition %q: the accounts config split does not support nested secret keys yet", definition)
		}
	}
}
