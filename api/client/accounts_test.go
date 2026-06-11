package client_test

import (
	"context"
	"encoding/json"
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
						"name": "def-1",
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
	assert.Equal(t, "def-1", page.Accounts[0].Definition.Name)
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

func TestClientAccountsListAll(t *testing.T) {
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

	accounts, err := c.Accounts.ListAll(ctx)
	require.NoError(t, err)
	assert.NotNil(t, accounts)
	assert.Len(t, accounts, 3)
	assert.Equal(t, "id-1", accounts[0].ID)
	assert.Equal(t, "name-1", accounts[0].Name)
	assert.Equal(t, "type-1", accounts[0].Definition.Type)
	assert.Equal(t, "category-1", accounts[0].Definition.Category)
	assert.Equal(t, "id-2", accounts[1].ID)
	assert.Equal(t, "name-2", accounts[1].Name)
	assert.Equal(t, "type-2", accounts[1].Definition.Type)
	assert.Equal(t, "category-2", accounts[1].Definition.Category)
	assert.Equal(t, "id-3", accounts[2].ID)
	assert.Equal(t, "name-3", accounts[2].Name)
	assert.Equal(t, "type-3", accounts[2].Definition.Type)
	assert.Equal(t, "category-3", accounts[2].Definition.Category)

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
					"name": "some-definition-name",
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
	assert.Equal(t, "some-definition-name", account.Definition.Name)
	assert.Equal(t, "some-type", account.Definition.Type)
	assert.Equal(t, "some-category", account.Definition.Category)
	assert.Equal(t, time.Date(2020, 1, 1, 1, 1, 1, 0, time.UTC), *account.CreatedAt)
	assert.Equal(t, time.Date(2020, 1, 2, 1, 1, 1, 0, time.UTC), *account.UpdatedAt)

	httpClient.AssertNumberOfCalls()
}

func TestClientAccountsCreate(t *testing.T) {
	ctx := context.Background()

	calls := []testutils.Call{
		{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(t, req, "POST", "https://api.rudderstack.com/v2/accounts", `{
					"accountDefinitionName": "BigQuery",
					"name": "some-name",
					"options": { "key1": "val1" },
					"secret": { "token": "shh" }
				}`)
			},
			ResponseStatus: 201,
			ResponseBody: `{
				"id": "some-id",
				"name": "some-name",
				"definition": {
					"name": "BigQuery",
					"type": "some-type",
					"category": "some-category"
				},
				"options": { "key1": "val1" },
				"createdAt": "2020-01-01T01:01:01Z",
				"updatedAt": "2020-01-02T01:01:01Z"
			}`,
		},
	}

	httpClient := testutils.NewMockHTTPClient(t, calls...)

	c, err := client.New("some-access-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	account, err := c.Accounts.Create(ctx, &client.CreateAccountRequest{
		AccountDefinitionName: "BigQuery",
		Name:                  "some-name",
		Options:               json.RawMessage([]byte(`{ "key1": "val1" }`)),
		Secret:                json.RawMessage([]byte(`{ "token": "shh" }`)),
	})
	require.NoError(t, err)
	assert.NotNil(t, account)
	assert.Equal(t, "some-id", account.ID)
	assert.Equal(t, "some-name", account.Name)
	assert.Equal(t, "BigQuery", account.Definition.Name)
	assert.Equal(t, "some-type", account.Definition.Type)
	assert.Equal(t, "some-category", account.Definition.Category)
	assert.Equal(t, time.Date(2020, 1, 1, 1, 1, 1, 0, time.UTC), *account.CreatedAt)
	assert.Equal(t, time.Date(2020, 1, 2, 1, 1, 1, 0, time.UTC), *account.UpdatedAt)

	httpClient.AssertNumberOfCalls()
}

func TestClientAccountsUpdate(t *testing.T) {
	ctx := context.Background()

	calls := []testutils.Call{
		{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(t, req, "PUT", "https://api.rudderstack.com/v2/accounts/some-id", `{
					"name": "new-name",
					"options": { "key1": "val1" },
					"secret": { "token": "shh" }
				}`)
			},
			ResponseStatus: 200,
			ResponseBody: `{
				"id": "some-id",
				"name": "new-name",
				"definition": {
					"name": "BigQuery",
					"type": "some-type",
					"category": "some-category"
				},
				"options": { "key1": "val1" },
				"createdAt": "2020-01-01T01:01:01Z",
				"updatedAt": "2020-01-02T01:01:01Z"
			}`,
		},
	}

	httpClient := testutils.NewMockHTTPClient(t, calls...)

	c, err := client.New("some-access-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	account, err := c.Accounts.Update(ctx, "some-id", &client.UpdateAccountRequest{
		Name:    "new-name",
		Options: json.RawMessage([]byte(`{ "key1": "val1" }`)),
		Secret:  json.RawMessage([]byte(`{ "token": "shh" }`)),
	})
	require.NoError(t, err)
	assert.NotNil(t, account)
	assert.Equal(t, "some-id", account.ID)
	assert.Equal(t, "new-name", account.Name)
	assert.Equal(t, "BigQuery", account.Definition.Name)
	assert.Equal(t, "some-type", account.Definition.Type)
	assert.Equal(t, "some-category", account.Definition.Category)
	assert.Equal(t, time.Date(2020, 1, 1, 1, 1, 1, 0, time.UTC), *account.CreatedAt)
	assert.Equal(t, time.Date(2020, 1, 2, 1, 1, 1, 0, time.UTC), *account.UpdatedAt)

	httpClient.AssertNumberOfCalls()
}

func TestClientAccountsDelete(t *testing.T) {
	ctx := context.Background()

	calls := []testutils.Call{
		{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(t, req, "DELETE", "https://api.rudderstack.com/v2/accounts/some-id", "")
			},
			ResponseStatus: 204,
			ResponseBody:   "",
		},
	}

	httpClient := testutils.NewMockHTTPClient(t, calls...)

	c, err := client.New("some-access-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	err = c.Accounts.Delete(ctx, "some-id")
	require.NoError(t, err)

	httpClient.AssertNumberOfCalls()
}
