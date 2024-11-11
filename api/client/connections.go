package client

import (
	"context"
	"time"
)

type Connection struct {
	ID            string     `json:"id,omitempty"`
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

func (s *connections) Next(ctx context.Context, paging Paging) (*ConnectionsPage, error) {
	page := &ConnectionsPage{}
	ok, err := s.service.next(ctx, paging, page)
	if !ok {
		page = nil
	}
	return page, err
}

func (s *connections) List(ctx context.Context) (*ConnectionsPage, error) {
	page := &ConnectionsPage{}
	if err := s.list(ctx, page); err != nil {
		return nil, err
	}

	return page, nil
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
		return nil, err
	}

	return response.Connection, nil
}

func (s *connections) Update(ctx context.Context, connection *Connection) (*Connection, error) {
	// copy input and remove ID from request body without modifying input
	conn := *connection
	conn.ID = ""

	response := struct{ Connection *Connection }{}
	if err := s.update(ctx, connection.ID, &conn, &response); err != nil {
		return nil, err
	}

	return response.Connection, nil
}

func (s *connections) Delete(ctx context.Context, id string) error {
	return s.service.delete(ctx, id)
}
