package catalog

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/api/client/apitask"
	"github.com/rudderlabs/rudder-iac/cli/pkg/tasker"
)

// PaginatedResponse defines the generic structure for all paginated API responses
type PaginatedResponse[T any] struct {
	Data        []T `json:"data"`
	Total       int `json:"total"`
	CurrentPage int `json:"currentPage"`
	PageSize    int `json:"pageSize"`
}

type DataCatalog interface {
	EventStore
	PropertyStore
	TrackingPlanStore
	StateStore
	CustomTypeStore
	CategoryStore
}

type ListOptions struct {
	HasExternalID *bool
}

func (o *ListOptions) ToQuery() string {
	query := ""
	if o.HasExternalID != nil {
		query += fmt.Sprintf("?hasExternalId=%t", *o.HasExternalID)
	}
	return query
}

type RudderDataCatalog struct {
	client *client.Client
}

func NewRudderDataCatalog(client *client.Client) DataCatalog {
	return &RudderDataCatalog{
		client: client,
	}
}

func IsCatalogNotFoundError(err error) bool {
	var apiErr *client.APIError

	if ok := errors.As(err, &apiErr); !ok {
		return false
	}
	return apiErr.HTTPStatusCode == 400 && strings.Contains(apiErr.Message, "not found")
}

func IsCatalogAlreadyExistsError(err error) bool {
	var apiErr *client.APIError

	if ok := errors.As(err, &apiErr); !ok {
		return false
	}
	return apiErr.HTTPStatusCode == 400 && strings.Contains(apiErr.Message, "already exists")
}

func getAllResourcesPaginated[T any](ctx context.Context, apiClient *client.Client, endpoint string) ([]T, error) {
	firstPage, err := getFirstPage[T](ctx, apiClient, endpoint)
	if err != nil {
		return nil, fmt.Errorf("getting first page: %w", err)
	}

	totalPages := int(math.Ceil(float64(firstPage.Total) / float64(firstPage.PageSize)))
	fmt.Printf("Endpoint: %s Total pages: %d\n", endpoint, totalPages)

	if totalPages <= 1 {
		return firstPage.Data, nil
	}

	tasks := make([]tasker.Task, totalPages)

	results := apitask.NewResults[PaginatedResponse[T]]()
	for i := 1; i <= totalPages; i++ {
		u, err := url.Parse(endpoint)
		if err != nil {
			return nil, fmt.Errorf("parsing URL: %w", err)
		}
		q := u.Query()
		q.Set("page", strconv.Itoa(i))
		u.RawQuery = q.Encode()

		tasks[i-1] = apitask.NewAPIFetchTask[PaginatedResponse[T]](apiClient, u.String())
	}

	errs := tasker.RunTasks(
		ctx,
		tasks,
		10,
		false,
		apitask.RunAPIFetchTask(ctx, results),
	)

	if len(errs) > 0 {
		return nil, fmt.Errorf("errors fetching paginated resources: %w", errors.Join(errs...))
	}

	allItems := make([]T, 0, firstPage.Total)
	for _, key := range results.GetKeys() {
		item, ok := results.Get(key)
		if !ok {
			return nil, fmt.Errorf("item %s not found in results", key)
		}

		allItems = append(allItems, item.Data...)
	}

	return allItems, nil
}

func getFirstPage[T any](ctx context.Context, apiClient *client.Client, endpoint string) (PaginatedResponse[T], error) {
	toReturn := PaginatedResponse[T]{}

	resp, err := apiClient.Do(ctx, "GET", endpoint, nil)
	if err != nil {
		var apiErr *client.APIError
		if ok := errors.As(err, &apiErr); ok && apiErr.FeatureNotEnabled() {
			return toReturn, nil
		}
		return toReturn, fmt.Errorf("sending get request: %w", err)
	}

	if err := json.NewDecoder(bytes.NewReader(resp)).Decode(&toReturn); err != nil {
		return toReturn, fmt.Errorf("decoding response: %w", err)
	}

	return toReturn, nil
}
