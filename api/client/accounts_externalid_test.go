package client_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/api/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccountExternalIDMarshalsAsExternalId(t *testing.T) {
	b, err := json.Marshal(client.Account{ExternalID: "ext-1"})
	require.NoError(t, err)
	assert.Contains(t, string(b), `"externalId":"ext-1"`)
}

func TestClientAccountsListWithHasExternalID(t *testing.T) {
	ctx := context.Background()
	calls := []testutils.Call{
		{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(t, req, "GET", "", "") &&
					assert.Equal(t, "https://api.rudderstack.com/v2/accounts?hasExternalId=true", req.URL.String())
			},
			ResponseStatus: 200,
			ResponseBody:   `{"data": [{"id": "id-1", "externalId": "ext-1"}], "paging": {"total": 1}}`,
		},
	}

	httpClient := testutils.NewMockHTTPClient(t, calls...)
	c, err := client.New("some-access-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	page, err := c.Accounts.List(ctx, client.WithHasExternalID(true))
	require.NoError(t, err)
	require.Len(t, page.Accounts, 1)
	assert.Equal(t, "ext-1", page.Accounts[0].ExternalID)

	httpClient.AssertNumberOfCalls()
}

func TestClientAccountsSetExternalID(t *testing.T) {
	ctx := context.Background()
	calls := []testutils.Call{
		{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(t, req, "PUT", "https://api.rudderstack.com/v2/accounts/some-id/external-id", `{"externalId":"ext-1"}`)
			},
			ResponseStatus: 200,
			ResponseBody:   `{"id": "some-id", "externalId": "ext-1"}`,
		},
	}

	httpClient := testutils.NewMockHTTPClient(t, calls...)
	c, err := client.New("some-access-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	err = c.Accounts.SetExternalID(ctx, "some-id", "ext-1")
	require.NoError(t, err)

	httpClient.AssertNumberOfCalls()
}

func TestClientAccountsGetReturnsExternalID(t *testing.T) {
	ctx := context.Background()
	calls := []testutils.Call{
		{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/v2/accounts/some-id", "")
			},
			ResponseStatus: 200,
			ResponseBody:   `{"id": "some-id", "externalId": "ext-1"}`,
		},
	}

	httpClient := testutils.NewMockHTTPClient(t, calls...)
	c, err := client.New("some-access-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	account, err := c.Accounts.Get(ctx, "some-id")
	require.NoError(t, err)
	assert.Equal(t, "ext-1", account.ExternalID)

	httpClient.AssertNumberOfCalls()
}
