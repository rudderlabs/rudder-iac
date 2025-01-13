package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type PropertyCreate struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Type        string                 `json:"type"`
	Config      map[string]interface{} `json:"propConfig,omitempty"`
}

type Property struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Type        string                 `json:"type"`
	WorkspaceId string                 `json:"workspaceId"`
	Config      map[string]interface{} `json:"propConfig"`
	CreatedAt   time.Time              `json:"createdAt"`
	UpdatedAt   time.Time              `json:"updatedAt"`
	CreatedBy   string                 `json:"createdBy"`
	UpdatedBy   string                 `json:"updatedBy"`
}

type EventCreate struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	EventType   string  `json:"eventType"`
	CategoryId  *string `json:"categoryId"`
}

type Event struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	EventType   string    `json:"eventType"`
	CategoryId  *string   `json:"categoryId"`
	WorkspaceId string    `json:"workspaceId"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type TrackingPlanCreate struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type TrackingPlanUpsertEventRules struct {
	Type       string `json:"type"`
	Properties struct {
		Properties *TrackingPlanUpsertEventProperties            `json:"properties,omitempty"`
		Traits     *TrackingPlanUpsertEventProperties            `json:"traits,omitempty"`
		Context    *TrackingPlanUpsertEventContextTraitsIdentity `json:"context,omitempty"`
	} `json:"properties"`
}

type TrackingPlanUpsertEventProperties struct {
	Type                 string                 `json:"type"`
	AdditionalProperties bool                   `json:"additionalProperties"`
	Properties           map[string]interface{} `json:"properties"`
	Required             []string               `json:"required"`
}

type TrackingPlanUpsertEventContextTraitsIdentity struct {
	Properties struct {
		Traits TrackingPlanUpsertEventProperties `json:"traits,omitempty"`
	} `json:"properties"`
}

type TrackingPlanUpsertEvent struct {
	Name            string                       `json:"name"`
	Description     string                       `json:"description"`
	EventType       string                       `json:"eventType"`
	IdentitySection string                       `json:"identitySection"`
	Rules           TrackingPlanUpsertEventRules `json:"rules"`
}

type TrackingPlan struct {
	ID           string              `json:"id"`
	Name         string              `json:"name"`
	Description  *string             `json:"description,omitempty"`
	CreationType string              `json:"creationType"`
	Version      int                 `json:"version"`
	WorkspaceID  string              `json:"workspaceId"`
	CreatedAt    time.Time           `json:"createdAt"`
	UpdatedAt    time.Time           `json:"updatedAt"`
	Events       []TrackingPlanEvent `json:"events"`
}

type TrackingPlanEvent struct {
	ID             string `json:"id"`
	TrackingPlanID string `json:"trackingPlanId"`
	EventID        string `json:"eventId"`
	SchemaID       string `json:"schemaId"`
}

type TrackingPlanStore interface {
	CreateTrackingPlan(ctx context.Context, input TrackingPlanCreate) (*TrackingPlan, error)
	UpsertTrackingPlan(ctx context.Context, id string, input TrackingPlanUpsertEvent) (*TrackingPlan, error)
	UpdateTrackingPlan(ctx context.Context, id string, name, description string) (*TrackingPlan, error)
	DeleteTrackingPlan(ctx context.Context, id string) error
	DeleteTrackingPlanEvent(ctx context.Context, trackingPlanId string, eventId string) error
}

type EventStore interface {
	CreateEvent(ctx context.Context, input EventCreate) (*Event, error)
	UpdateEvent(ctx context.Context, id string, input *Event) (*Event, error)
	DeleteEvent(ctx context.Context, id string) error
}

type PropertyStore interface {
	CreateProperty(ctx context.Context, input PropertyCreate) (*Property, error)
	UpdateProperty(ctx context.Context, id string, input *Property) (*Property, error)
	DeleteProperty(ctx context.Context, id string) error
}

type DataCatalog interface {
	EventStore
	PropertyStore
	TrackingPlanStore
}

type RudderDataCatalog struct {
	client *Client
}

func NewRudderDataCatalog(client *Client) DataCatalog {

	return &RudderDataCatalog{
		client: client,
	}
}

func (c *RudderDataCatalog) DeleteEvent(ctx context.Context, id string) error {
	_, err := c.client.Do(ctx, "DELETE", fmt.Sprintf("catalog/events/%s", id), nil)
	if err != nil {
		return fmt.Errorf("sending delete request: %w", err)
	}

	return nil
}

func (c *RudderDataCatalog) CreateEvent(ctx context.Context, input EventCreate) (*Event, error) {
	body, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("marshalling input: %w", err)
	}

	resp, err := c.client.Do(ctx, "POST", "catalog/events", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}

	var event Event
	if err := json.NewDecoder(bytes.NewReader(resp)).Decode(&event); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &event, nil
}

func (c *RudderDataCatalog) UpdateEvent(ctx context.Context, id string, input *Event) (*Event, error) {
	byt, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("marshalling input: %w", err)
	}

	resp, err := c.client.Do(ctx, "PUT", fmt.Sprintf("catalog/events/%s", id), bytes.NewReader(byt))
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}

	var event Event
	if err := json.NewDecoder(bytes.NewReader(resp)).Decode(&event); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &event, nil
}

func (c *RudderDataCatalog) DeleteProperty(ctx context.Context, id string) error {
	_, err := c.client.Do(ctx, "DELETE", fmt.Sprintf("catalog/properties/%s", id), nil)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}
	return nil
}

func (c *RudderDataCatalog) UpdateProperty(ctx context.Context, id string, new *Property) (*Property, error) {
	byt, err := json.Marshal(new)
	if err != nil {
		return nil, fmt.Errorf("marshalling input: %w", err)
	}

	resp, err := c.client.Do(ctx, "PUT", fmt.Sprintf("catalog/properties/%s", id), bytes.NewReader(byt))
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}

	var property Property
	if err := json.NewDecoder(bytes.NewReader(resp)).Decode(&property); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &property, nil
}

func (c *RudderDataCatalog) CreateProperty(ctx context.Context, input PropertyCreate) (*Property, error) {
	byt, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("marshalling input: %w", err)
	}

	resp, err := c.client.Do(ctx, "POST", "catalog/properties", bytes.NewReader(byt))
	if err != nil {
		return nil, fmt.Errorf("executing http request: %w", err)
	}

	property := Property{} // Create a holder for response object
	if err := json.NewDecoder(bytes.NewReader(resp)).Decode(&property); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &property, nil
}

// TODO: Make this create idempotent so that we can call it multiple times without error
func (c *RudderDataCatalog) CreateTrackingPlan(ctx context.Context, input TrackingPlanCreate) (*TrackingPlan, error) {
	byt, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("marshalling input: %w", err)
	}

	resp, err := c.client.Do(ctx, "POST", "catalog/tracking-plans", bytes.NewReader(byt))
	if err != nil {
		return nil, fmt.Errorf("executing http request: %w", err)
	}

	trackingPlan := TrackingPlan{} // Create a holder for response object
	if err := json.NewDecoder(bytes.NewReader(resp)).Decode(&trackingPlan); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &trackingPlan, nil
}

func (c *RudderDataCatalog) UpdateTrackingPlan(ctx context.Context, id string, name, description string) (*TrackingPlan, error) {
	byt, err := json.Marshal(struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}{
		Name:        name,
		Description: description,
	})

	if err != nil {
		return nil, fmt.Errorf("marshalling input: %w", err)
	}

	resp, err := c.client.Do(ctx, "PUT", fmt.Sprintf("catalog/tracking-plans/%s", id), bytes.NewReader(byt))
	if err != nil {
		return nil, fmt.Errorf("executing http request: %w", err)
	}
	trackingPlan := TrackingPlan{} // Create a holder for response object
	if err := json.NewDecoder(bytes.NewReader(resp)).Decode(&trackingPlan); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &trackingPlan, nil
}

func (c *RudderDataCatalog) UpsertTrackingPlan(ctx context.Context, id string, event TrackingPlanUpsertEvent) (*TrackingPlan, error) {
	byt, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("marshalling input: %w", err)
	}

	resp, err := c.client.Do(ctx, "PATCH", fmt.Sprintf("catalog/tracking-plans/%s/events", id), bytes.NewReader(byt))
	if err != nil {
		return nil, fmt.Errorf("executing http request: %w", err)
	}

	tp := TrackingPlan{}
	if err := json.NewDecoder(bytes.NewReader(resp)).Decode(&tp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &tp, nil
}

func (c *RudderDataCatalog) DeleteTrackingPlan(ctx context.Context, id string) error {
	_, err := c.client.Do(ctx, "DELETE", fmt.Sprintf("catalog/tracking-plans/%s", id), nil)
	if err != nil {
		return fmt.Errorf("sending delete request: %w", err)
	}

	return nil
}

func (c *RudderDataCatalog) DeleteTrackingPlanEvent(ctx context.Context, trackingPlanId string, eventId string) error {
	_, err := c.client.Do(ctx, "DELETE", fmt.Sprintf("catalog/tracking-plans/%s/events/%s", trackingPlanId, eventId), nil)
	if err != nil {
		return fmt.Errorf("sending delete request: %w", err)
	}
	return nil
}

func IsCatalogNotFoundError(err error) bool {
	var apiErr *APIError

	if ok := errors.As(err, &apiErr); !ok {
		return false
	}
	return apiErr.HTTPStatusCode == 400 && strings.Contains(apiErr.Message, "not found")
}
