package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

// paginatedResponse defines the generic structure for all paginated API responses
type paginatedResponse[T any] struct {
	Data        []T `json:"data"`
	Total       int `json:"total"`
	CurrentPage int `json:"currentPage"`
	PageSize    int `json:"pageSize"`
}

// GetAllResourcesWithPagination is a generic helper function that fetches all items from a paginated API endpoint
func GetAllResourcesWithPagination[T any](
	ctx context.Context,
	client *Client,
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

		resp, err := client.Do(ctx, "GET", u.String(), nil)
		if err != nil {
			return nil, fmt.Errorf("sending get request for page %d: %w", page, err)
		}

		var response paginatedResponse[T]
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