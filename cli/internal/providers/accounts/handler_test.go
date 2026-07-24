package accounts

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/secret"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// enableVarSubstitution turns on the experimental gate that makes exported
// secrets serialize as "{{ .VAR }}" references instead of masked literals.
func enableVarSubstitution(t *testing.T) {
	t.Helper()
	prevExp, prevFlag := viper.Get("experimental"), viper.Get("flags.enableVarSubstitution")
	viper.Set("experimental", true)
	viper.Set("flags.enableVarSubstitution", true)
	t.Cleanup(func() {
		viper.Set("experimental", prevExp)
		viper.Set("flags.enableVarSubstitution", prevFlag)
	})
}

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
		AccountDefinitionName: "SOURCE_BIGQUERY",
		Config: map[string]any{
			"projectId":   "proj-123",
			"location":    "US",
			"credentials": &cred,
		},
	}
}

func TestCreate_SplitsConfigAndClaimsExternalID(t *testing.T) {
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
	assert.Equal(t, "prod-bq", m.created.Name)
	// external id claimed = spec id, on the returned remote id.
	assert.Equal(t, [2]string{"remote-1", "prod-bq"}, m.externalIDSet)
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
	acc := &client.Account{ID: "remote-1", ExternalID: "prod-bq", Options: json.RawMessage(`{"projectId":"p"}`)}
	acc.Definition.Name = "SOURCE_BIGQUERY"

	res, state, err := h.MapRemoteToState(&RemoteAccount{Account: acc}, nil)
	require.NoError(t, err)
	assert.Equal(t, "prod-bq", res.ID)
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
		Options:    json.RawMessage(opts),
	}
	acc.Definition.Name = "SOURCE_BIGQUERY"
	return &RemoteAccount{Account: acc}
}

// Export must tokenize the secret into a per-resource "{{ .VAR }}" reference the
// user fills via a var file — the API never returns the value, so a masked
// literal would be useless. Non-secret options pass through verbatim.
func TestToExportSpecMap_TokenizesSecret(t *testing.T) {
	enableVarSubstitution(t)
	h := &HandlerImpl{store: &mockStore{}}

	specMap, err := h.toExportSpecMap("prod-analytics-bq", bqRemote("prod-analytics-bq", `{"project":"acme","location":"US"}`))
	require.NoError(t, err)

	config := specMap["config"].(map[string]any)
	assert.Equal(t, "{{ .PROD_ANALYTICS_BQ_CREDENTIALS }}", config["credentials"], "secret must export as a var reference")
	assert.Equal(t, "acme", config["project"])
	assert.Equal(t, "US", config["location"])
	assert.Equal(t, "prod-analytics-bq", specMap["id"])
	assert.Equal(t, "SOURCE_BIGQUERY", specMap["account_definition_name"])
}

// The whole exported spec — serialized as it would be written to disk — must
// never carry a raw secret. Even a value the API happened to echo back stays
// masked.
func TestFormatForExport_NeverLeaksSecret(t *testing.T) {
	enableVarSubstitution(t)
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
	acc.Definition.Name = "SOURCE_SNOWFLAKE"

	_, err := h.toExportSpecMap("x", &RemoteAccount{Account: acc})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported definition")
}
