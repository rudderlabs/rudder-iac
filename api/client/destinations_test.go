package client_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/api/internal/testutils"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientDestinationsList(t *testing.T) {
	ctx := context.Background()

	calls := []testutils.Call{
		{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/v2/destinations", "")
			},
			ResponseStatus: 200,
			ResponseBody: `{
				"destinations": [{
					"id": "id-1",
					"type": "type-1",
					"name": "name-1",
					"config": {"key":"val-1"},
					"transformation": {"id":"trans-1"}
				},  {
					"id": "id-2",
					"type": "type-2",
					"name": "name-2",
					"config": {"key":"val-2"}
				}],
				"paging": {
					"total": 3,
					"next": "/destinations?page=2"
				}
			}`,
		},
		{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/v2/destinations?page=2", "")
			},
			ResponseStatus: 200,
			ResponseBody: `{
				"destinations": [{
					"id": "id-3",
					"type": "type-3",
					"name": "name-3",
					"config": {"key":"val-3"}
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

	page, err := c.Destinations.List(ctx)
	require.NoError(t, err)
	assert.NotNil(t, page)
	assert.Len(t, page.Destinations, 2)
	assert.Equal(t, client.Destination{
		ID:             "id-1",
		Type:           "type-1",
		Name:           "name-1",
		Config:         []byte(`{"key":"val-1"}`),
		Transformation: &client.DestinationTransformationLink{ID: "trans-1"},
	}, page.Destinations[0])
	assert.Equal(t, client.Destination{ID: "id-2", Type: "type-2", Name: "name-2", Config: []byte(`{"key":"val-2"}`)}, page.Destinations[1])
	assert.Nil(t, page.Destinations[1].Transformation)
	assert.Equal(t, 3, page.Paging.Total)
	assert.Equal(t, "/destinations?page=2", page.Paging.Next)

	page, err = c.Destinations.Next(ctx, page.Paging)
	require.NoError(t, err)
	assert.NotNil(t, page)
	assert.Len(t, page.Destinations, 1)
	assert.Equal(t, client.Destination{ID: "id-3", Type: "type-3", Name: "name-3", Config: []byte(`{"key":"val-3"}`)}, page.Destinations[0])
	assert.Equal(t, 3, page.Paging.Total)
	assert.Equal(t, "", page.Paging.Next)

	page, err = c.Destinations.Next(ctx, page.Paging)
	require.NoError(t, err)
	assert.Nil(t, page)

	httpClient.AssertNumberOfCalls()
}

func TestClientDestinationsGet(t *testing.T) {
	ctx := context.Background()

	calls := []testutils.Call{
		{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/v2/destinations/some-id", "")
			},
			ResponseStatus: 200,
			ResponseBody: `{
				"destination": {
					"id": "some-id",
					"name": "some-name",
					"type": "some-type",
					"config": {"key1": "val1"},
					"version": 2,
					"versionInfo": {
						"status": "deprecated",
						"action": "upgrade",
						"retirementDate": "2026-12-31",
						"migrationDocsURL": "https://docs.example.com/destinations/migration"
					},
					"createdAt": "2020-01-01T01:01:01Z",
					"updatedAt": "2020-01-02T01:01:01Z"	
				}
			}`,
		},
	}

	httpClient := testutils.NewMockHTTPClient(t, calls...)

	c, err := client.New("some-access-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	destination, err := c.Destinations.Get(ctx, "some-id")
	require.NoError(t, err)
	assert.NotNil(t, destination)
	assert.Equal(t, &client.Destination{
		ID:      "some-id",
		Name:    "some-name",
		Type:    "some-type",
		Version: 2,
		VersionInfo: &client.VersionInfo{
			Status:           "deprecated",
			Action:           "upgrade",
			RetirementDate:   lo.ToPtr("2026-12-31"),
			MigrationDocsURL: lo.ToPtr("https://docs.example.com/destinations/migration"),
		},
		Config:    []byte(`{"key1": "val1"}`),
		CreatedAt: lo.ToPtr(time.Date(2020, 1, 1, 1, 1, 1, 0, time.UTC)),
		UpdatedAt: lo.ToPtr(time.Date(2020, 1, 2, 1, 1, 1, 0, time.UTC)),
	}, destination)

	httpClient.AssertNumberOfCalls()
}

func TestClientDestinationsGetWithoutVersionInfo(t *testing.T) {
	ctx := context.Background()

	calls := []testutils.Call{
		{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/v2/destinations/some-id", "")
			},
			ResponseStatus: 200,
			ResponseBody: `{
				"destination": {
					"id": "some-id",
					"name": "some-name",
					"type": "some-type",
					"config": {"key1": "val1"}
				}
			}`,
		},
	}

	httpClient := testutils.NewMockHTTPClient(t, calls...)

	c, err := client.New("some-access-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	destination, err := c.Destinations.Get(ctx, "some-id")
	require.NoError(t, err)
	assert.Equal(t, &client.Destination{
		ID:     "some-id",
		Name:   "some-name",
		Type:   "some-type",
		Config: []byte(`{"key1": "val1"}`),
	}, destination)
	assert.Nil(t, destination.VersionInfo)

	httpClient.AssertNumberOfCalls()
}

func TestClientDestinationsCreate(t *testing.T) {
	ctx := context.Background()

	calls := []testutils.Call{
		{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(t, req, "POST", "https://api.rudderstack.com/v2/destinations", `{
					"name": "some-name",
					"type": "some-type",
					"version": 2,
					"enabled": true,
					"config": { "key1": "val1" }
				}`)
			},
			ResponseStatus: 200,
			ResponseBody: `{
				"destination": {
					"id": "some-id",
					"name": "some-name",
					"type": "some-type",
					"config": {
						"key1": "val1"
					},
					"createdAt": "2020-01-01T01:01:01Z",
					"updatedAt": "2020-01-02T01:01:01Z"
				}
			}`,
		},
	}

	httpClient := testutils.NewMockHTTPClient(t, calls...)

	c, err := client.New("some-access-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	destination, err := c.Destinations.Create(ctx, &client.Destination{
		Name:      "some-name",
		Type:      "some-type",
		Version:   2,
		IsEnabled: true,
		Config: json.RawMessage([]byte(`{
			"key1": "val1"
		}`)),
	})
	require.NoError(t, err)
	assert.NotNil(t, destination)
	assert.Equal(t, "some-id", destination.ID)
	assert.Equal(t, "some-name", destination.Name)
	assert.Equal(t, "some-type", destination.Type)
	assert.Equal(t, time.Date(2020, 1, 1, 1, 1, 1, 0, time.UTC), *destination.CreatedAt)
	assert.Equal(t, time.Date(2020, 1, 2, 1, 1, 1, 0, time.UTC), *destination.UpdatedAt)

	httpClient.AssertNumberOfCalls()
}

func TestClientDestinationsCreateOmitsEmptyVersion(t *testing.T) {
	ctx := context.Background()

	calls := []testutils.Call{
		{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(t, req, "POST", "https://api.rudderstack.com/v2/destinations", `{
					"name": "some-name",
					"type": "some-type",
					"enabled": true,
					"config": { "key1": "val1" }
				}`)
			},
			ResponseStatus: 200,
			ResponseBody: `{
				"destination": {
					"id": "some-id",
					"name": "some-name",
					"type": "some-type",
					"config": {
						"key1": "val1"
					}
				}
			}`,
		},
	}

	httpClient := testutils.NewMockHTTPClient(t, calls...)

	c, err := client.New("some-access-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	destination, err := c.Destinations.Create(ctx, &client.Destination{
		Name:      "some-name",
		Type:      "some-type",
		IsEnabled: true,
		Config: json.RawMessage([]byte(`{
			"key1": "val1"
		}`)),
	})
	require.NoError(t, err)
	assert.Equal(t, "some-id", destination.ID)

	httpClient.AssertNumberOfCalls()
}

func TestClientDestinationsUpdate(t *testing.T) {
	ctx := context.Background()

	calls := []testutils.Call{
		{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(t, req, "PUT", "https://api.rudderstack.com/v2/destinations/some-id", `{
					"name": "some-name",
					"type": "some-type",
					"version": 2,
					"enabled": true,
					"config": { "key1": "val1" }
				}`)
			},
			ResponseStatus: 200,
			ResponseBody: `{
				"destination": {
					"id": "some-id",
					"name": "some-name",
					"type": "some-type",
					"config": {
						"key1": "val1"
					},
					"createdAt": "2020-01-01T01:01:01Z",
					"updatedAt": "2020-01-02T01:01:01Z"
				}
			}`,
		},
	}

	httpClient := testutils.NewMockHTTPClient(t, calls...)

	c, err := client.New("some-access-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	destination, err := c.Destinations.Update(ctx, &client.Destination{
		ID:        "some-id",
		Name:      "some-name",
		Type:      "some-type",
		Version:   2,
		IsEnabled: true,
		Config: json.RawMessage([]byte(`{
			"key1": "val1"
		}`)),
	})
	require.NoError(t, err)
	assert.NotNil(t, destination)
	assert.Equal(t, "some-id", destination.ID)
	assert.Equal(t, "some-name", destination.Name)
	assert.Equal(t, "some-type", destination.Type)
	assert.Equal(t, time.Date(2020, 1, 1, 1, 1, 1, 0, time.UTC), *destination.CreatedAt)
	assert.Equal(t, time.Date(2020, 1, 2, 1, 1, 1, 0, time.UTC), *destination.UpdatedAt)

	httpClient.AssertNumberOfCalls()
}

func TestClientDestinationsUpdateOmitsEmptyVersion(t *testing.T) {
	ctx := context.Background()

	calls := []testutils.Call{
		{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(t, req, "PUT", "https://api.rudderstack.com/v2/destinations/some-id", `{
					"name": "some-name",
					"type": "some-type",
					"enabled": true,
					"config": { "key1": "val1" }
				}`)
			},
			ResponseStatus: 200,
			ResponseBody: `{
				"destination": {
					"id": "some-id",
					"name": "some-name",
					"type": "some-type",
					"config": {
						"key1": "val1"
					}
				}
			}`,
		},
	}

	httpClient := testutils.NewMockHTTPClient(t, calls...)

	c, err := client.New("some-access-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	destination, err := c.Destinations.Update(ctx, &client.Destination{
		ID:        "some-id",
		Name:      "some-name",
		Type:      "some-type",
		IsEnabled: true,
		Config: json.RawMessage([]byte(`{
			"key1": "val1"
		}`)),
	})
	require.NoError(t, err)
	assert.Equal(t, "some-id", destination.ID)

	httpClient.AssertNumberOfCalls()
}

func TestClientDestinations_ConnectTransformation(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(t, req, "PUT", "https://api.rudderstack.com/v2/destinations/some-destination-id/transformation", `{
					"transformationId": "some-transformation-id"
				}`)
			},
			ResponseStatus: 200,
			ResponseBody: `{
				"destinationId": "some-destination-id",
				"transformationId": "some-transformation-id",
				"createdAt": "2020-01-01T01:01:01Z",
				"updatedAt": "2020-01-02T01:01:01Z"
			}`,
		})

		c, err := client.New("some-access-token", client.WithHTTPClient(httpClient))
		require.NoError(t, err)

		result, err := c.Destinations.ConnectTransformation(ctx, "some-destination-id", "some-transformation-id")
		require.NoError(t, err)
		assert.Equal(t, &client.DestinationTransformation{
			DestinationID:    "some-destination-id",
			TransformationID: "some-transformation-id",
			CreatedAt:        time.Date(2020, 1, 1, 1, 1, 1, 0, time.UTC),
			UpdatedAt:        time.Date(2020, 1, 2, 1, 1, 1, 0, time.UTC),
		}, result)

		httpClient.AssertNumberOfCalls()
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(t, req, "PUT", "https://api.rudderstack.com/v2/destinations/some-destination-id/transformation", `{
					"transformationId": "some-transformation-id"
				}`)
			},
			ResponseStatus: 404,
			ResponseBody:   `{"message": "not found"}`,
		})

		c, err := client.New("some-access-token", client.WithHTTPClient(httpClient))
		require.NoError(t, err)

		_, err = c.Destinations.ConnectTransformation(ctx, "some-destination-id", "some-transformation-id")
		require.Error(t, err)

		var apiErr *client.APIError
		require.True(t, errors.As(err, &apiErr))
		assert.Equal(t, 404, apiErr.HTTPStatusCode)

		httpClient.AssertNumberOfCalls()
	})
}

func TestClientDestinations_DisconnectTransformation(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(
					t,
					req,
					"DELETE",
					"https://api.rudderstack.com/v2/destinations/some-destination-id/transformation",
					"",
				)
			},
			ResponseStatus: 200,
			ResponseBody: `{
				"destinationId": "some-destination-id",
				"transformationId": "some-transformation-id",
				"createdAt": "2020-01-01T01:01:01Z",
				"updatedAt": "2020-01-02T01:01:01Z"
			}`,
		})

		c, err := client.New("some-access-token", client.WithHTTPClient(httpClient))
		require.NoError(t, err)

		result, err := c.Destinations.DisconnectTransformation(ctx, "some-destination-id")
		require.NoError(t, err)
		assert.Equal(t, &client.DestinationTransformation{
			DestinationID:    "some-destination-id",
			TransformationID: "some-transformation-id",
			CreatedAt:        time.Date(2020, 1, 1, 1, 1, 1, 0, time.UTC),
			UpdatedAt:        time.Date(2020, 1, 2, 1, 1, 1, 0, time.UTC),
		}, result)

		httpClient.AssertNumberOfCalls()
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(
					t,
					req,
					"DELETE",
					"https://api.rudderstack.com/v2/destinations/some-destination-id/transformation",
					"",
				)
			},
			ResponseStatus: 404,
			ResponseBody:   `{"message": "not found"}`,
		})

		c, err := client.New("some-access-token", client.WithHTTPClient(httpClient))
		require.NoError(t, err)

		_, err = c.Destinations.DisconnectTransformation(ctx, "some-destination-id")
		require.Error(t, err)

		var apiErr *client.APIError
		require.True(t, errors.As(err, &apiErr))
		assert.Equal(t, 404, apiErr.HTTPStatusCode)

		httpClient.AssertNumberOfCalls()
	})
}

func TestClientDestinations_GetTransformation(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(
					t,
					req,
					"GET",
					"https://api.rudderstack.com/v2/destinations/some-destination-id/transformation",
					"",
				)
			},
			ResponseStatus: 200,
			ResponseBody: `{
				"destinationId": "some-destination-id",
				"transformationId": "some-transformation-id",
				"createdAt": "2020-01-01T01:01:01Z",
				"updatedAt": "2020-01-02T01:01:01Z"
			}`,
		})

		c, err := client.New("some-access-token", client.WithHTTPClient(httpClient))
		require.NoError(t, err)

		result, err := c.Destinations.GetTransformation(ctx, "some-destination-id")
		require.NoError(t, err)
		assert.Equal(t, &client.DestinationTransformation{
			DestinationID:    "some-destination-id",
			TransformationID: "some-transformation-id",
			CreatedAt:        time.Date(2020, 1, 1, 1, 1, 1, 0, time.UTC),
			UpdatedAt:        time.Date(2020, 1, 2, 1, 1, 1, 0, time.UTC),
		}, result)

		httpClient.AssertNumberOfCalls()
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		httpClient := testutils.NewMockHTTPClient(t, testutils.Call{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/v2/destinations/some-destination-id/transformation", "")
			},
			ResponseStatus: 404,
			ResponseBody:   `{"message": "not found"}`,
		})

		c, err := client.New("some-access-token", client.WithHTTPClient(httpClient))
		require.NoError(t, err)

		_, err = c.Destinations.GetTransformation(ctx, "some-destination-id")
		require.Error(t, err)

		var apiErr *client.APIError
		require.True(t, errors.As(err, &apiErr))
		assert.Equal(t, 404, apiErr.HTTPStatusCode)

		httpClient.AssertNumberOfCalls()
	})
}
func TestClientDestinations_GetAll(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name          string
		calls         []testutils.Call
		want          []client.Destination
		wantErrSubstr string
	}{
		{
			name: "multi page",
			calls: []testutils.Call{
				{
					Validate: func(req *http.Request) bool {
						return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/v2/destinations", "")
					},
					ResponseStatus: 200,
					ResponseBody: `{
						"destinations": [{
							"id": "id-1",
							"type": "type-1",
							"name": "name-1",
							"config": {"key":"val-1"}
						},  {
							"id": "id-2",
							"type": "type-2",
							"name": "name-2",
							"config": {"key":"val-2"}
						}],
						"paging": {
							"total": 3,
							"next": "/destinations?page=2"
						}
					}`,
				},
				{
					Validate: func(req *http.Request) bool {
						return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/v2/destinations?page=2", "")
					},
					ResponseStatus: 200,
					ResponseBody: `{
						"destinations": [{
							"id": "id-3",
							"type": "type-3",
							"name": "name-3",
							"config": {"key":"val-3"}
						}],
						"paging": {
							"total": 3
						}
					}`,
				},
			},
			want: []client.Destination{
				{ID: "id-1", Type: "type-1", Name: "name-1", Config: []byte(`{"key":"val-1"}`)},
				{ID: "id-2", Type: "type-2", Name: "name-2", Config: []byte(`{"key":"val-2"}`)},
				{ID: "id-3", Type: "type-3", Name: "name-3", Config: []byte(`{"key":"val-3"}`)},
			},
		},
		{
			name: "single page",
			calls: []testutils.Call{
				{
					Validate: func(req *http.Request) bool {
						return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/v2/destinations", "")
					},
					ResponseStatus: 200,
					ResponseBody: `{
						"destinations": [{
							"id": "id-1",
							"type": "type-1",
							"name": "name-1",
							"config": {"key":"val-1"}
						}],
						"paging": {
							"total": 1
						}
					}`,
				},
			},
			want: []client.Destination{
				{ID: "id-1", Type: "type-1", Name: "name-1", Config: []byte(`{"key":"val-1"}`)},
			},
		},
		{
			name: "list error",
			calls: []testutils.Call{
				{
					Validate: func(req *http.Request) bool {
						return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/v2/destinations", "")
					},
					ResponseStatus: 500,
					ResponseBody:   `{"error":"Internal Server Error"}`,
				},
			},
			wantErrSubstr: "listing destinations",
		},
		{
			name: "next page error",
			calls: []testutils.Call{
				{
					Validate: func(req *http.Request) bool {
						return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/v2/destinations", "")
					},
					ResponseStatus: 200,
					ResponseBody: `{
						"destinations": [{
							"id": "id-1",
							"type": "type-1",
							"name": "name-1",
							"config": {"key":"val-1"}
						}],
						"paging": {
							"total": 2,
							"next": "/destinations?page=2"
						}
					}`,
				},
				{
					Validate: func(req *http.Request) bool {
						return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/v2/destinations?page=2", "")
					},
					ResponseStatus: 500,
					ResponseBody:   `{"error":"Internal Server Error"}`,
				},
			},
			wantErrSubstr: "fetching next destinations page",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			httpClient := testutils.NewMockHTTPClient(t, tc.calls...)

			c, err := client.New("some-access-token", client.WithHTTPClient(httpClient))
			require.NoError(t, err)

			got, err := c.Destinations.GetAll(ctx)
			if tc.wantErrSubstr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErrSubstr)
				assert.Nil(t, got)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.want, got)
			}

			httpClient.AssertNumberOfCalls()
		})
	}
}

func TestClientDestinationsGetExternalID(t *testing.T) {
	ctx := context.Background()

	calls := []testutils.Call{
		{
			Validate: func(req *http.Request) bool {
				return testutils.ValidateRequest(t, req, "GET", "https://api.rudderstack.com/v2/destinations/some-id", "")
			},
			ResponseStatus: 200,
			ResponseBody: `{
				"destination": {
					"id": "some-id",
					"externalId": "ga4-production",
					"name": "Production GA4",
					"type": "GA4",
					"config": {"apiSecret":"secret-value"},
					"enabled": true
				}
			}`,
		},
	}

	httpClient := testutils.NewMockHTTPClient(t, calls...)

	c, err := client.New("some-access-token", client.WithHTTPClient(httpClient))
	require.NoError(t, err)

	destination, err := c.Destinations.Get(ctx, "some-id")
	require.NoError(t, err)
	require.NotNil(t, destination)
	assert.Equal(t, "ga4-production", destination.ExternalID)
	assert.Equal(t, &client.Destination{
		ID:         "some-id",
		ExternalID: "ga4-production",
		Name:       "Production GA4",
		Type:       "GA4",
		IsEnabled:  true,
		Config:     []byte(`{"apiSecret":"secret-value"}`),
	}, destination)

	httpClient.AssertNumberOfCalls()
}
