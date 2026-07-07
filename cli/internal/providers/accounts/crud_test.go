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

type setExtCall struct{ id, externalID string }

type mockStore struct {
	createReq   *client.CreateAccountRequest
	createResp  *client.Account
	updateID    string
	updateReq   *client.UpdateAccountRequest
	updateResp  *client.Account
	deleteID    string
	setExtCalls []setExtCall
	listOpts    *client.ListAccountsOptions
	listResp    []client.Account
	err         error
}

func (m *mockStore) Create(_ context.Context, req *client.CreateAccountRequest) (*client.Account, error) {
	m.createReq = req
	return m.createResp, m.err
}

func (m *mockStore) Update(_ context.Context, id string, req *client.UpdateAccountRequest) (*client.Account, error) {
	m.updateID, m.updateReq = id, req
	return m.updateResp, m.err
}

func (m *mockStore) Delete(_ context.Context, id string) error {
	m.deleteID = id
	return m.err
}

func (m *mockStore) Get(_ context.Context, _ string) (*client.Account, error) {
	return nil, m.err
}

func (m *mockStore) ListAll(_ context.Context, opts ...client.ListAccountsOption) ([]client.Account, error) {
	o := &client.ListAccountsOptions{}
	for _, opt := range opts {
		opt(o)
	}
	m.listOpts = o
	return m.listResp, m.err
}

func (m *mockStore) SetExternalID(_ context.Context, id, externalID string) error {
	m.setExtCalls = append(m.setExtCalls, setExtCall{id, externalID})
	return m.err
}

func bqAccount(id, externalID, definition string) client.Account {
	a := client.Account{ID: id, ExternalID: externalID}
	a.Definition.Name = definition
	return a
}

func TestCreateSplitsConfigAndClaimsExternalID(t *testing.T) {
	m := &mockStore{createResp: &client.Account{ID: "remote-uuid"}}
	h := &HandlerImpl{store: m}
	cred := secret.New("bq-creds")

	st, err := h.Create(context.Background(), &AccountResource{
		ID:                    "lovable-prod-bq",
		AccountDefinitionName: "SOURCE_BIGQUERY",
		Options:               map[string]any{"project": "p1"},
		Credentials:           &cred,
	})
	require.NoError(t, err)
	assert.Equal(t, "remote-uuid", st.ID)

	assert.Equal(t, "SOURCE_BIGQUERY", m.createReq.AccountDefinitionName)
	assert.Equal(t, "lovable-prod-bq", m.createReq.Name)
	assert.JSONEq(t, `{"project":"p1"}`, string(m.createReq.Options))
	assert.JSONEq(t, `{"credentials":"bq-creds"}`, string(m.createReq.Secret))

	require.Len(t, m.setExtCalls, 1)
	assert.Equal(t, setExtCall{id: "remote-uuid", externalID: "lovable-prod-bq"}, m.setExtCalls[0])
}

func TestUpdateRejectsImmutableDefinition(t *testing.T) {
	h := &HandlerImpl{store: &mockStore{}}
	cred := secret.New("x")

	_, err := h.Update(context.Background(),
		&AccountResource{ID: "a", AccountDefinitionName: "SOURCE_SNOWFLAKE", Credentials: &cred},
		&AccountResource{ID: "a", AccountDefinitionName: "SOURCE_BIGQUERY"},
		&AccountState{ID: "remote"},
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "immutable")
}

func TestUpdateFullReplaces(t *testing.T) {
	m := &mockStore{updateResp: &client.Account{ID: "remote"}}
	h := &HandlerImpl{store: m}
	cred := secret.New("newcreds")

	st, err := h.Update(context.Background(),
		&AccountResource{ID: "a", AccountDefinitionName: "SOURCE_BIGQUERY", Options: map[string]any{"project": "p2"}, Credentials: &cred},
		&AccountResource{ID: "a", AccountDefinitionName: "SOURCE_BIGQUERY"},
		&AccountState{ID: "remote"},
	)
	require.NoError(t, err)
	assert.Equal(t, "remote", st.ID)
	assert.Equal(t, "remote", m.updateID)
	assert.Equal(t, "a", m.updateReq.Name)
	assert.JSONEq(t, `{"project":"p2"}`, string(m.updateReq.Options))
	assert.JSONEq(t, `{"credentials":"newcreds"}`, string(m.updateReq.Secret))
}

func TestDeleteUsesRemoteID(t *testing.T) {
	m := &mockStore{}
	h := &HandlerImpl{store: m}

	require.NoError(t, h.Delete(context.Background(), "a", &AccountResource{}, &AccountState{ID: "remote"}))
	assert.Equal(t, "remote", m.deleteID)
}

func TestLoadRemoteResourcesFiltersAndRequestsManaged(t *testing.T) {
	m := &mockStore{listResp: []client.Account{
		bqAccount("1", "ext1", "SOURCE_BIGQUERY"),
		bqAccount("2", "ext2", "SOURCE_SNOWFLAKE"),
	}}
	h := &HandlerImpl{store: m}

	got, err := h.LoadRemoteResources(context.Background())
	require.NoError(t, err)
	require.Len(t, got, 1, "only the BigQuery account is supported")
	assert.Equal(t, "1", got[0].ID)

	require.NotNil(t, m.listOpts.HasExternalID)
	assert.True(t, *m.listOpts.HasExternalID, "managed load requests hasExternalId=true")
}

func TestLoadImportableResourcesRequestsUnmanaged(t *testing.T) {
	m := &mockStore{}
	h := &HandlerImpl{store: m}

	_, err := h.LoadImportableResources(context.Background())
	require.NoError(t, err)
	require.NotNil(t, m.listOpts.HasExternalID)
	assert.False(t, *m.listOpts.HasExternalID, "importable load requests hasExternalId=false")
}

func TestMapRemoteToStateKeysByExternalIDWithUnknownSecret(t *testing.T) {
	a := bqAccount("remote-uuid", "lovable-prod-bq", "SOURCE_BIGQUERY")
	a.Options = json.RawMessage(`{"project":"p1"}`)
	h := &HandlerImpl{}

	res, st, err := h.MapRemoteToState(&RemoteAccount{Account: &a}, nil)
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, "lovable-prod-bq", res.ID)
	assert.Equal(t, "SOURCE_BIGQUERY", res.AccountDefinitionName)
	assert.Equal(t, "p1", res.Options["project"])
	require.NotNil(t, res.Credentials)
	assert.True(t, res.Credentials.IsUnknown(), "secret must be unknown (always-diff)")
	assert.Equal(t, "remote-uuid", st.ID)
}

func TestMapRemoteToStateSkipsUnmanaged(t *testing.T) {
	a := bqAccount("x", "", "SOURCE_BIGQUERY")
	h := &HandlerImpl{}

	res, st, err := h.MapRemoteToState(&RemoteAccount{Account: &a}, nil)
	require.NoError(t, err)
	assert.Nil(t, res)
	assert.Nil(t, st)
}
