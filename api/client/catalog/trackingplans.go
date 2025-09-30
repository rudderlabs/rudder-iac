package catalog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type TrackingPlanCreate struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	ExternalID  string `json:"externalId,omitempty"`
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
	CategoryId      *string                      `json:"categoryId,omitempty"`
	EventType       string                       `json:"eventType"`
	IdentitySection string                       `json:"identitySection"`
	Rules           TrackingPlanUpsertEventRules `json:"rules"`
}

type TrackingPlan struct {
	ID           string              `json:"id"`
	ExternalId   string              `json:"externalId"`
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

type TrackingPlanWithIdentifiers struct {
	ID           string                                 `json:"id"`
	ExternalID   string                                 `json:"externalId"`
	Name         string                                 `json:"name"`
	Description  *string                                `json:"description,omitempty"`
	CreationType string                                 `json:"creationType"`
	Version      int                                    `json:"version"`
	WorkspaceID  string                                 `json:"workspaceId"`
	CreatedAt    time.Time                              `json:"createdAt"`
	UpdatedAt    time.Time                              `json:"updatedAt"`
	Events       []TrackingPlanEventPropertyIdentifiers `json:"events"`
}

type TrackingPlanWithoutEvents struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Description  *string   `json:"description,omitempty"`
	Version      int       `json:"version"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
	CreationType string    `json:"creationType"`
	WorkspaceID  string    `json:"workspaceId"`
}

type GetTrackingPlansResponse struct {
	TrackingPlans []TrackingPlanWithoutEvents `json:"trackingPlans"`
}

type TrackingPlanWithSchemas struct {
	ID           string                    `json:"id"`
	Name         string                    `json:"name"`
	Description  *string                   `json:"description,omitempty"`
	CreationType string                    `json:"creationType"`
	Version      int                       `json:"version"`
	WorkspaceID  string                    `json:"workspaceId"`
	CreatedAt    time.Time                 `json:"createdAt"`
	UpdatedAt    time.Time                 `json:"updatedAt"`
	Events       []TrackingPlanEventSchema `json:"events"`
}

type TrackingPlanEventPropertyIdentifiers struct {
	ID                   string                       `json:"id"`
	ExternalID           string                       `json:"externalId"`
	Name                 string                       `json:"name"`
	Description          string                       `json:"description"`
	EventType            string                       `json:"eventType"`
	CategoryId           *string                      `json:"categoryId"` // Can be null
	WorkspaceId          string                       `json:"workspaceId"`
	CreatedBy            string                       `json:"createdBy"`
	UpdatedBy            *string                      `json:"updatedBy"` // Can be null
	CreatedAt            time.Time                    `json:"createdAt"`
	UpdatedAt            time.Time                    `json:"updatedAt"`
	IdentitySection      string                       `json:"identitySection"`
	AdditionalProperties bool                         `json:"additionalProperties"`
	Properties           []*TrackingPlanEventProperty `json:"properties,omitempty"`
	Variants             []Variant                    `json:"variants,omitempty"`
}

type TrackingPlanEventProperty struct {
	ID                   string                       `json:"id"`
	Name                 string                       `json:"name"`
	Required             bool                         `json:"required"`
	AdditionalProperties bool                         `json:"additionalProperties"`
	Properties           []*TrackingPlanEventProperty `json:"properties,omitempty"`
}

type TrackingPlanEventSchema struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	EventType       string    `json:"eventType"`
	CategoryId      *string   `json:"categoryId"` // Can be null
	WorkspaceId     string    `json:"workspaceId"`
	CreatedBy       string    `json:"createdBy"`
	UpdatedBy       *string   `json:"updatedBy"` // Can be null
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
	IdentitySection string    `json:"identitySection"`
	Rules           struct {
		Schema     string                 `json:"$schema"`
		Type       string                 `json:"type"`
		Properties map[string]interface{} `json:"properties"`
	} `json:"rules"`
}

const (
	IdentitySectionProperties    = "properties"
	IdentitySectionTraits        = "traits"
	IdentitySectionContextTraits = "context.traits"
)

type TrackingPlanEventsUpdate struct {
	Events []EventIdentifierDetail `json:"events"`
}

type EventIdentifierDetail struct {
	ID                   string                     `json:"id"`
	Properties           []PropertyIdentifierDetail `json:"properties"`
	AdditionalProperties bool                       `json:"additionalProperties"`
	IdentitySection      string                     `json:"identitySection"`
	Variants             []Variant                  `json:"variants,omitempty"`
}

type PropertyIdentifierDetail struct {
	ID                   string                     `json:"id"`
	Required             bool                       `json:"required"`
	AdditionalProperties bool                       `json:"additionalProperties"`
	Metadata             map[string]any             `json:"metadata,omitempty"`
	Properties           []PropertyIdentifierDetail `json:"properties,omitempty"`
}

type TrackingPlanStore interface {
	CreateTrackingPlan(ctx context.Context, input TrackingPlanCreate) (*TrackingPlan, error)
	UpsertTrackingPlan(ctx context.Context, id string, input TrackingPlanUpsertEvent) (*TrackingPlan, error)
	UpdateTrackingPlan(ctx context.Context, id string, name, description string) (*TrackingPlan, error)
	DeleteTrackingPlan(ctx context.Context, id string) error
	DeleteTrackingPlanEvent(ctx context.Context, trackingPlanId string, eventId string) error
	GetTrackingPlan(ctx context.Context, id string) (*TrackingPlanWithIdentifiers, error)
	GetTrackingPlans(ctx context.Context) ([]*TrackingPlanWithIdentifiers, error)
	GetTrackingPlanEventSchema(ctx context.Context, id string, eventId string) (*TrackingPlanEventSchema, error)
	GetTrackingPlanEventWithIdentifiers(ctx context.Context, id, eventId string) (*TrackingPlanEventPropertyIdentifiers, error)
	UpdateTrackingPlanEvent(ctx context.Context, id string, input EventIdentifierDetail) (*TrackingPlan, error)
}

// TODO: Make this create idempotent so that we can call it multiple times without error
func (c *RudderDataCatalog) CreateTrackingPlan(ctx context.Context, input TrackingPlanCreate) (*TrackingPlan, error) {
	byt, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("marshalling input: %w", err)
	}

	resp, err := c.client.Do(ctx, "POST", "v2/catalog/tracking-plans", bytes.NewReader(byt))
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

	resp, err := c.client.Do(ctx, "PUT", fmt.Sprintf("v2/catalog/tracking-plans/%s", id), bytes.NewReader(byt))
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

	resp, err := c.client.Do(ctx, "PATCH", fmt.Sprintf("v2/catalog/tracking-plans/%s/events", id), bytes.NewReader(byt))
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
	_, err := c.client.Do(ctx, "DELETE", fmt.Sprintf("v2/catalog/tracking-plans/%s", id), nil)
	if err != nil {
		return fmt.Errorf("sending delete request: %w", err)
	}

	return nil
}

func (c *RudderDataCatalog) DeleteTrackingPlanEvent(ctx context.Context, trackingPlanId string, eventId string) error {
	_, err := c.client.Do(ctx, "DELETE", fmt.Sprintf("v2/catalog/tracking-plans/%s/events/%s", trackingPlanId, eventId), nil)
	if err != nil {
		return fmt.Errorf("sending delete request: %w", err)
	}
	return nil
}

func (c *RudderDataCatalog) GetTrackingPlan(ctx context.Context, id string) (*TrackingPlanWithIdentifiers, error) {
	resp, err := c.client.Do(ctx, "GET", fmt.Sprintf("v2/catalog/tracking-plans/%s", id), nil)
	if err != nil {
		return nil, fmt.Errorf("executing http request to fetch tracking plan: %w", err)
	}

	trackingPlan := TrackingPlanWithIdentifiers{}
	if err := json.NewDecoder(bytes.NewReader(resp)).Decode(&trackingPlan); err != nil {
		return nil, fmt.Errorf("decoding tracking plan response: %w", err)
	}

	var events struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}

	eventsResp, err := c.client.Do(ctx, "GET", fmt.Sprintf("v2/catalog/tracking-plans/%s/events", id), nil)
	if err != nil {
		return nil, fmt.Errorf("executing http request to fetch events on tracking plan: %w", err)
	}

	if err := json.NewDecoder(bytes.NewReader(eventsResp)).Decode(&events); err != nil {
		return nil, fmt.Errorf("decoding events response: %w, response: %s", err, string(eventsResp))
	}

	trackingPlan.Events = make([]TrackingPlanEventPropertyIdentifiers, len(events.Data))
	for i, event := range events.Data {
		schema, err := c.GetTrackingPlanEventWithIdentifiers(ctx, id, event.ID)
		if err != nil {
			return nil, fmt.Errorf("fetching event schema: %s on tracking plan: %s: %w", event.ID, id, err)
		}
		trackingPlan.Events[i] = *schema
	}

	return &trackingPlan, nil
}

func (c *RudderDataCatalog) GetTrackingPlans(ctx context.Context) ([]*TrackingPlanWithIdentifiers, error) {
	resp, err := c.client.Do(ctx, "GET", "v2/catalog/tracking-plans", nil)
	if err != nil {
		return nil, fmt.Errorf("executing http request to fetch tracking plans: %w", err)
	}

	var trackingPlansResp GetTrackingPlansResponse
	if err := json.NewDecoder(bytes.NewReader(resp)).Decode(&trackingPlansResp); err != nil {
		return nil, fmt.Errorf("decoding tracking plans response: %w", err)
	}

	// Convert to slice of pointers and fetch full details
	result := make([]*TrackingPlanWithIdentifiers, len(trackingPlansResp.TrackingPlans))
	for i := range trackingPlansResp.TrackingPlans {
		// Get full tracking plan details with events
		trackingPlan, err := c.GetTrackingPlan(ctx, trackingPlansResp.TrackingPlans[i].ID)
		if err != nil {
			return nil, fmt.Errorf("fetching tracking plan %s: %w", trackingPlansResp.TrackingPlans[i].ID, err)
		}
		result[i] = trackingPlan
	}

	return result, nil
}

func (c *RudderDataCatalog) GetTrackingPlanEventWithIdentifiers(ctx context.Context, id, eventId string) (*TrackingPlanEventPropertyIdentifiers, error) {
	resp, err := c.client.Do(ctx, "GET", fmt.Sprintf("v2/catalog/tracking-plans/%s/events/%s?format=properties", id, eventId), nil)
	if err != nil {
		return nil, fmt.Errorf("executing http request: %w", err)
	}

	eventWithProps := TrackingPlanEventPropertyIdentifiers{}
	if err := json.NewDecoder(bytes.NewReader(resp)).Decode(&eventWithProps); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &eventWithProps, nil
}

func (c *RudderDataCatalog) GetTrackingPlanEventSchema(ctx context.Context, id string, eventId string) (*TrackingPlanEventSchema, error) {
	resp, err := c.client.Do(ctx, "GET", fmt.Sprintf("v2/catalog/tracking-plans/%s/events/%s?format=schema", id, eventId), nil)
	if err != nil {
		return nil, fmt.Errorf("executing http request: %w", err)
	}

	schema := TrackingPlanEventSchema{}
	if err := json.NewDecoder(bytes.NewReader(resp)).Decode(&schema); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &schema, nil
}

func (c *RudderDataCatalog) UpdateTrackingPlanEvent(ctx context.Context, id string, input EventIdentifierDetail) (*TrackingPlan, error) {
	byt, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("marshalling input: %w", err)
	}

	resp, err := c.client.Do(ctx, "PUT", fmt.Sprintf("v2/catalog/tracking-plans/%s/events", id), bytes.NewReader(byt))
	if err != nil {
		return nil, fmt.Errorf("executing http request: %w", err)
	}

	trackingPlan := TrackingPlan{}
	if err := json.NewDecoder(bytes.NewReader(resp)).Decode(&trackingPlan); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &trackingPlan, nil
}
