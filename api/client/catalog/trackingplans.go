package catalog

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/rudderlabs/rudder-iac/api/client/apitask"
	"github.com/rudderlabs/rudder-iac/cli/pkg/tasker"
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
	ID           string    `json:"id"`
	ExternalID   string    `json:"externalId"`
	Name         string    `json:"name"`
	Description  *string   `json:"description,omitempty"`
	CreationType string    `json:"creationType"`
	Version      int       `json:"version"`
	WorkspaceID  string    `json:"workspaceId"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type TrackingPlanWithIdentifiers struct {
	TrackingPlan
	Events []*TrackingPlanEventPropertyIdentifiers `json:"events"`
}

type GetTrackingPlansResponse struct {
	TrackingPlans []TrackingPlan `json:"trackingPlans"`
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
		Schema     string         `json:"$schema"`
		Type       string         `json:"type"`
		Properties map[string]any `json:"properties"`
		Defs       map[string]any `json:"$defs,omitempty"`
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
	GetTrackingPlanWithSchemas(ctx context.Context, id string) (*TrackingPlanWithSchemas, error)
	GetTrackingPlan(ctx context.Context, id string) (*TrackingPlan, error)
	GetTrackingPlans(ctx context.Context, options ListOptions) ([]*TrackingPlan, error)
	GetTrackingPlanWithIdentifiers(ctx context.Context, id string, rebuildSchemas bool) (*TrackingPlanWithIdentifiers, error)
	GetTrackingPlansWithIdentifiers(ctx context.Context, options ListOptions) ([]*TrackingPlanWithIdentifiers, error)
	GetTrackingPlanEventSchema(ctx context.Context, id string, eventId string) (*TrackingPlanEventSchema, error)
	UpdateTrackingPlanEvents(ctx context.Context, id string, input []EventIdentifierDetail, rebuildSchemas bool) error
	SetTrackingPlanExternalId(ctx context.Context, id string, externalId string) error
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

func (c *RudderDataCatalog) GetTrackingPlans(ctx context.Context, options ListOptions) ([]*TrackingPlan, error) {
	resp, err := c.client.Do(ctx, "GET", fmt.Sprintf("v2/catalog/tracking-plans%s", options.ToQuery()), nil)
	if err != nil {
		return nil, fmt.Errorf("executing http request to fetch tracking plans: %w", err)
	}

	tps := struct {
		TrackingPlans []*TrackingPlan `json:"trackingPlans"`
	}{}

	if err := json.NewDecoder(bytes.NewReader(resp)).Decode(&tps); err != nil {
		return nil, fmt.Errorf("decoding tracking plans response: %w", err)
	}

	return tps.TrackingPlans, nil
}

func (c *RudderDataCatalog) GetTrackingPlan(ctx context.Context, id string) (*TrackingPlan, error) {
	resp, err := c.client.Do(ctx, "GET", fmt.Sprintf("v2/catalog/tracking-plans/%s", id), nil)
	if err != nil {
		return nil, fmt.Errorf("executing http request to fetch tracking plan skeleton: %w", err)
	}

	trackingPlan := TrackingPlan{}
	if err := json.NewDecoder(bytes.NewReader(resp)).Decode(&trackingPlan); err != nil {
		return nil, fmt.Errorf("decoding tracking plan response: %w", err)
	}

	return &trackingPlan, nil
}

func (c *RudderDataCatalog) GetTrackingPlanWithIdentifiers(ctx context.Context, id string, rebuildSchemas bool) (*TrackingPlanWithIdentifiers, error) {
	resp, err := c.client.Do(ctx, "GET", fmt.Sprintf("v2/catalog/tracking-plans/%s", id), nil)
	if err != nil {
		return nil, fmt.Errorf("executing http request to fetch tracking plan: %w", err)
	}

	trackingPlan := TrackingPlanWithIdentifiers{}
	if err := json.NewDecoder(bytes.NewReader(resp)).Decode(&trackingPlan); err != nil {
		return nil, fmt.Errorf("decoding tracking plan response: %w", err)
	}

	events, err := getAllResourcesPaginated[*TrackingPlanEventResponse](
		ctx,
		c.client,
		fmt.Sprintf("v2/catalog/tracking-plans/%s/events?rebuildSchemas=%t", id, rebuildSchemas),
		c.concurrency,
	)
	if err != nil {
		return nil, fmt.Errorf("fetching all events on tracking plan: %s: %w", id, err)
	}

	eventTasks := make([]tasker.Task, len(events))
	for i, event := range events {
		eventTasks[i] = apitask.NewAPIFetchTask[*TrackingPlanEventPropertyIdentifiers](
			c.client,
			fmt.Sprintf("v2/catalog/tracking-plans/%s/events/%s?format=properties&rebuildSchemas=%t",
				id,
				event.ID,
				rebuildSchemas,
			),
		)
	}

	results := tasker.NewResults[*TrackingPlanEventPropertyIdentifiers]()
	errs := tasker.RunTasks(
		ctx,
		eventTasks,
		c.concurrency,
		false,
		apitask.RunAPIFetchTask(ctx, results),
	)

	if len(errs) > 0 {
		return nil, fmt.Errorf("errors fetching events identifiers: %w", errors.Join(errs...))
	}

	eventSchemas := make([]*TrackingPlanEventPropertyIdentifiers, len(events))
	for i, key := range results.GetKeys() {
		event, ok := results.Get(key)
		if !ok {
			return nil, fmt.Errorf("event %s not found in results", key)
		}
		eventSchemas[i] = event
	}

	trackingPlan.Events = eventSchemas
	return &trackingPlan, nil
}

type TrackingPlanEventResponse struct {
	ID string `json:"id"`
}

func (c *RudderDataCatalog) GetTrackingPlanWithSchemas(ctx context.Context, id string) (*TrackingPlanWithSchemas, error) {
	resp, err := c.client.Do(ctx, "GET", fmt.Sprintf("v2/catalog/tracking-plans/%s", id), nil)
	if err != nil {
		return nil, fmt.Errorf("executing http request to fetch tracking plan: %w", err)
	}

	trackingPlan := TrackingPlanWithSchemas{}
	if err := json.NewDecoder(bytes.NewReader(resp)).Decode(&trackingPlan); err != nil {
		return nil, fmt.Errorf("decoding tracking plan response: %w", err)
	}

	events, err := getAllResourcesPaginated[*TrackingPlanEventResponse](ctx, c.client, fmt.Sprintf("v2/catalog/tracking-plans/%s/events", id), c.concurrency)
	if err != nil {
		return nil, fmt.Errorf("fetching all events on tracking plan: %s: %w", id, err)
	}

	trackingPlan.Events = make([]TrackingPlanEventSchema, len(events))
	for i, event := range events {
		schema, err := c.GetTrackingPlanEventSchema(ctx, id, event.ID)
		if err != nil {
			return nil, fmt.Errorf("fetching event schema: %s on tracking plan: %s: %w", event.ID, id, err)
		}
		trackingPlan.Events[i] = *schema
	}

	return &trackingPlan, nil
}

func (c *RudderDataCatalog) GetTrackingPlansWithIdentifiers(ctx context.Context, options ListOptions) ([]*TrackingPlanWithIdentifiers, error) {
	resp, err := c.client.Do(ctx, "GET", fmt.Sprintf("v2/catalog/tracking-plans%s", options.ToQuery()), nil)
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
		trackingPlan, err := c.GetTrackingPlanWithIdentifiers(ctx, trackingPlansResp.TrackingPlans[i].ID, options.RebuildSchemas)
		if err != nil {
			return nil, fmt.Errorf("fetching tracking plan %s: %w", trackingPlansResp.TrackingPlans[i].ID, err)
		}
		result[i] = trackingPlan
	}

	return result, nil
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

func (c *RudderDataCatalog) UpdateTrackingPlanEvents(ctx context.Context, id string, events []EventIdentifierDetail, rebuildSchemas bool) error {
	var (
		err   error
		batch []EventIdentifierDetail
	)

	for i := 0; i < len(events); i += c.eventUpdateBatchSize {
		batch = events[i:min(i+c.eventUpdateBatchSize, len(events))]

		_, err = c.updateTrackingPlanEventsBatch(ctx, id, batch, rebuildSchemas)
		if err != nil {
			return fmt.Errorf("updating batch of tracking plan events: %w", err)
		}
	}

	return nil
}

func (c *RudderDataCatalog) updateTrackingPlanEventsBatch(
	ctx context.Context,
	id string,
	events []EventIdentifierDetail,
	rebuildSchemas bool,
) (*TrackingPlan, error) {
	payload := struct {
		Events []EventIdentifierDetail `json:"events"`
	}{
		Events: events,
	}

	byt, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshalling input: %w", err)
	}

	resp, err := c.client.Do(ctx, "PUT", fmt.Sprintf("v2/catalog/tracking-plans/%s/events?rebuildSchemas=%t", id, rebuildSchemas), bytes.NewReader(byt))
	if err != nil {
		return nil, fmt.Errorf("executing http request: %w", err)
	}

	trackingPlan := TrackingPlan{}
	if err := json.NewDecoder(bytes.NewReader(resp)).Decode(&trackingPlan); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &trackingPlan, nil
}

func (c *RudderDataCatalog) SetTrackingPlanExternalId(ctx context.Context, id string, externalId string) error {
	payload := map[string]string{
		"externalId": externalId,
	}

	byt, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshalling payload: %w", err)
	}

	_, err = c.client.Do(ctx, "PUT", fmt.Sprintf("v2/catalog/tracking-plans/%s/external-id", id), bytes.NewReader(byt))
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}

	return nil
}
