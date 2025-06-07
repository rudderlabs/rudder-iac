package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/rudderlabs/rudder-iac/cli/pkg/schema/models"
)

// SchemaClient handles API communication for schema fetching
type SchemaClient struct {
	httpClient *http.Client
	baseURL    string
	apiToken   string
}

// NewSchemaClient creates a new schema API client
func NewSchemaClient(baseURL, apiToken string) *SchemaClient {
	return &SchemaClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL:  baseURL,
		apiToken: apiToken,
	}
}

// FetchAllSchemas fetches all schemas with pagination support
func (c *SchemaClient) FetchAllSchemas(writeKey string) ([]models.Schema, error) {
	var allSchemas []models.Schema
	page := 1

	for {
		schemas, hasNext, err := c.fetchSchemasPage(page, writeKey)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch schemas page %d: %w", page, err)
		}

		allSchemas = append(allSchemas, schemas...)

		if !hasNext {
			break
		}
		page++
	}

	return allSchemas, nil
}

// fetchSchemasPage fetches a single page of schemas
func (c *SchemaClient) fetchSchemasPage(page int, writeKey string) ([]models.Schema, bool, error) {
	// Build URL with query parameters
	u, err := url.Parse(c.baseURL + "/v2/schemas")
	if err != nil {
		return nil, false, fmt.Errorf("invalid base URL: %w", err)
	}

	query := u.Query()
	query.Set("page", strconv.Itoa(page))
	if writeKey != "" {
		query.Set("writeKey", writeKey)
	}
	u.RawQuery = query.Encode()

	// Create request
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, false, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authorization header
	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	req.Header.Set("Content-Type", "application/json")

	// Make request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, false, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, false, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var response models.SchemasResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, false, fmt.Errorf("failed to decode response: %w", err)
	}

	return response.Results, response.HasNext, nil
}
