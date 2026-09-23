package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

// ErrExternalIDAlreadyInUse signals a 409 from the set external ID endpoint.
// The backend uses 409 for two distinct conditions and distinguishes them only
// by message: the external ID is already claimed by another connection, or this
// connection already has one (externalId is set-once and cannot be changed).
// Callers that need to tell them apart must inspect the wrapped error.
var ErrExternalIDAlreadyInUse = fmt.Errorf("external ID already in use")

// ErrConnectionPairExists signals a 409 from create: a connection for this
// source/destination pair already exists, so the externalId claim was refused.
// The existing connection typically carries no externalId, so the remedy is to
// adopt it rather than to retry the create. As with the set endpoint, the
// backend's message is the only detail beyond the status, so it is wrapped.
var ErrConnectionPairExists = fmt.Errorf("connection already exists for this source and destination pair")

type Connection struct {
	ID            string     `json:"id,omitempty"`
	ExternalID    string     `json:"externalId,omitempty"`
	SourceID      string     `json:"sourceId"`
	DestinationID string     `json:"destinationId"`
	IsEnabled     bool       `json:"enabled"`
	CreatedAt     *time.Time `json:"createdAt,omitempty"`
	UpdatedAt     *time.Time `json:"updatedAt,omitempty"`
}

type connections struct {
	*service
}

type ConnectionsPage struct {
	APIPage
	Connections []Connection `json:"connections"`
}

type ListConnectionsOption func(*ListConnectionsOptions)

func WithConnectionsHasExternalID(hasExternalID bool) ListConnectionsOption {
	return func(o *ListConnectionsOptions) {
		o.HasExternalID = &hasExternalID
	}
}

type ListConnectionsOptions struct {
	HasExternalID *bool
}

func (s *connections) Next(ctx context.Context, paging Paging) (*ConnectionsPage, error) {
	page := &ConnectionsPage{}
	ok, err := s.service.next(ctx, paging, page)
	if !ok {
		page = nil
	}
	return page, err
}

func (s *connections) List(ctx context.Context, opts ...ListConnectionsOption) (*ConnectionsPage, error) {
	page := &ConnectionsPage{}
	if err := s.list(ctx, page, opts...); err != nil {
		return nil, err
	}

	return page, nil
}

func (s *connections) list(ctx context.Context, result interface{}, opts ...ListConnectionsOption) error {
	options := &ListConnectionsOptions{}
	for _, opt := range opts {
		opt(options)
	}

	path := s.basePath
	query := url.Values{}
	if options.HasExternalID != nil {
		query.Add("hasExternalId", strconv.FormatBool(*options.HasExternalID))
	}
	if len(query) > 0 {
		path = fmt.Sprintf("%s?%s", path, query.Encode())
	}

	_, err := s.next(ctx, Paging{Next: path}, result)
	return err
}

func (s *connections) Get(ctx context.Context, id string) (*Connection, error) {
	response := struct{ Connection *Connection }{}
	if err := s.get(ctx, id, &response); err != nil {
		return nil, err
	}

	return response.Connection, nil
}

func (s *connections) Create(ctx context.Context, connection *Connection) (*Connection, error) {
	// copy input and remove fields that should not be in request body without modifying input
	conn := *connection
	conn.ID = ""

	response := struct{ Connection *Connection }{}
	if err := s.create(ctx, &conn, &response); err != nil {
		// A create body carrying an externalId is refused with 409 when a live
		// connection already exists for the pair; map it so callers can offer
		// adoption instead of surfacing the backend's set-external-id advice.
		var apiError *APIError
		if errors.As(err, &apiError) && apiError.HTTPStatusCode == 409 {
			return nil, fmt.Errorf("%w: %w", ErrConnectionPairExists, err)
		}
		return nil, err
	}

	return response.Connection, nil
}

func (s *connections) Update(ctx context.Context, connection *Connection) (*Connection, error) {
	// copy input and remove fields that should not be in request body without
	// modifying input; externalId can only be set through the external-id
	// endpoint, the generic update rejects bodies carrying it
	conn := *connection
	conn.ID = ""
	conn.ExternalID = ""

	response := struct{ Connection *Connection }{}
	if err := s.update(ctx, connection.ID, &conn, &response); err != nil {
		return nil, err
	}

	return response.Connection, nil
}

func (s *connections) Delete(ctx context.Context, id string) error {
	return s.service.delete(ctx, id)
}

func (s *connections) SetExternalID(ctx context.Context, id, externalID string) error {
	body, err := json.Marshal(map[string]string{"externalId": externalID})
	if err != nil {
		return fmt.Errorf("marshalling external ID: %w", err)
	}

	path := fmt.Sprintf("%s/%s/external-id", s.basePath, id)
	if _, err := s.client.Do(ctx, "PUT", path, bytes.NewReader(body)); err != nil {
		var apiError *APIError
		if errors.As(err, &apiError) && apiError.HTTPStatusCode == 409 {
			return fmt.Errorf("%w: %w", ErrExternalIDAlreadyInUse, err)
		}
		return fmt.Errorf("setting external ID: %w", err)
	}

	return nil
}
