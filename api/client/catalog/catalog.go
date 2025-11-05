package catalog

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/rudderlabs/rudder-iac/api/client"
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

// getAllResourcesWithPagination is a generic helper function that fetches all items from a paginated catalog API endpoint
func getAllResourcesWithPagination[T any](
	ctx context.Context,
	apiClient *client.Client,
	endpoint string,
) ([]T, error) {
	var allItems []T
	page := 1

	for {
		// Build URL with page parameter
		u, err := url.Parse(endpoint)
		if err != nil {
			return nil, fmt.Errorf("parsing URL: %w", err)
		}

		q := u.Query()
		q.Set("page", strconv.Itoa(page))
		u.RawQuery = q.Encode()

		resp, err := apiClient.Do(ctx, "GET", u.String(), nil)
		if err != nil {
			var apiErr *client.APIError

			if ok := errors.As(err, &apiErr); ok && apiErr.FeatureNotEnabled() {
				return nil, nil
			}
			return nil, fmt.Errorf("sending get request for page %d: %w", page, err)
		}

		var response PaginatedResponse[T]
		if err := json.NewDecoder(bytes.NewReader(resp)).Decode(&response); err != nil {
			return nil, fmt.Errorf("decoding response for page %d: %w", page, err)
		}

		// Append items from current page
		allItems = append(allItems, response.Data...)

		// Check if we've reached the last page
		if response.CurrentPage*response.PageSize >= response.Total {
			break
		}

		page++
	}

	return allItems, nil
}
