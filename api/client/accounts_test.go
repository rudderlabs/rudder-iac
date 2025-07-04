package client_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/api/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientAccountsList(t *testing.T) {
	ctx := context.Background()

	calls := []testutils.Call{
		{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/v2/accounts", "")
			},
			ResponseStatus: 200,
			ResponseBody: `{
				"data": [{
					"id": "id-1",
					"name": "name-1",
					"definition": {
						"type": "type-1",
						"category": "category-1"
					}
				}, {
					"id": "id-2",
					"name": "name-2",
					"definition": {
						"type": "type-2",
						"category": "category-2"
					}
				}],
				"paging": {
					"total": 3,
					"next": "/v2/accounts?page=2"
				}
			}`,
		},
		{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/v2/accounts?page=2", "")
			},
			ResponseStatus: 200,
			ResponseBody: `{
				"data": [{
					"id": "id-3",
					"name": "name-3",
					"definition": {
						"type": "type-3",
						"category": "category-3"
					}
				}],
				"paging": {
					"total": 3
				}
			}`,
		},
	}

	httpClient := testutils.NewMockHTTPClient(t, calls...)

	c, err := client.New("some-access-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	page, err := c.Accounts.List(ctx)
	require.NoError(t, err)
	assert.NotNil(t, page)
	assert.Len(t, page.Accounts, 2)
	assert.Equal(t, "id-1", page.Accounts[0].ID)
	assert.Equal(t, "name-1", page.Accounts[0].Name)
	assert.Equal(t, "type-1", page.Accounts[0].Definition.Type)
	assert.Equal(t, "category-1", page.Accounts[0].Definition.Category)
	assert.Equal(t, "id-2", page.Accounts[1].ID)
	assert.Equal(t, "name-2", page.Accounts[1].Name)
	assert.Equal(t, "type-2", page.Accounts[1].Definition.Type)
	assert.Equal(t, "category-2", page.Accounts[1].Definition.Category)
	assert.Equal(t, 3, page.Paging.Total)
	assert.Equal(t, "/v2/accounts?page=2", page.Paging.Next)

	page, err = c.Accounts.Next(ctx, page.Paging)
	require.NoError(t, err)
	assert.NotNil(t, page)
	assert.Len(t, page.Accounts, 1)
	assert.Equal(t, "id-3", page.Accounts[0].ID)
	assert.Equal(t, "name-3", page.Accounts[0].Name)
	assert.Equal(t, "type-3", page.Accounts[0].Definition.Type)
	assert.Equal(t, "category-3", page.Accounts[0].Definition.Category)
	assert.Equal(t, 3, page.Paging.Total)
	assert.Equal(t, "", page.Paging.Next)

	page, err = c.Accounts.Next(ctx, page.Paging)
	require.NoError(t, err)
	assert.Nil(t, page)

	httpClient.AssertNumberOfCalls()
}

func TestClientAccountsGet(t *testing.T) {
	ctx := context.Background()

	calls := []testutils.Call{
		{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/v2/accounts/some-id", "")
			},
			ResponseStatus: 200,
			ResponseBody: `{
				"id": "some-id",
				"name": "some-name",
				"definition": {
					"type": "some-type",
					"category": "some-category"
				},
				"createdAt": "2020-01-01T01:01:01Z",
				"updatedAt": "2020-01-02T01:01:01Z"
			}`,
		},
	}

	httpClient := testutils.NewMockHTTPClient(t, calls...)

	c, err := client.New("some-access-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	account, err := c.Accounts.Get(ctx, "some-id")
	require.NoError(t, err)
	assert.NotNil(t, account)
	assert.Equal(t, "some-id", account.ID)
	assert.Equal(t, "some-name", account.Name)
	assert.Equal(t, "some-type", account.Definition.Type)
	assert.Equal(t, "some-category", account.Definition.Category)
	assert.Equal(t, time.Date(2020, 1, 1, 1, 1, 1, 0, time.UTC), *account.CreatedAt)
	assert.Equal(t, time.Date(2020, 1, 2, 1, 1, 1, 0, time.UTC), *account.UpdatedAt)

	httpClient.AssertNumberOfCalls()
}
