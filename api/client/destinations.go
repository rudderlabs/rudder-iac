package client

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"time"
)

type Destination struct {
	ID          string          `json:"id,omitempty"`
	Name        string          `json:"name"`
	Type        string          `json:"type"`
	Version     int             `json:"version,omitempty"`
	VersionInfo *VersionInfo    `json:"versionInfo,omitempty"`
	IsEnabled   bool            `json:"enabled"`
	Config      json.RawMessage `json:"config"`
	CreatedAt   *time.Time      `json:"createdAt,omitempty"`
	UpdatedAt   *time.Time      `json:"updatedAt,omitempty"`
}

type VersionInfo struct {
	Status           string  `json:"status,omitempty"`
	Action           string  `json:"action,omitempty"`
	RetirementDate   *string `json:"retirementDate,omitempty"`
	MigrationDocsURL *string `json:"migrationDocsURL,omitempty"`
}

type destinations struct {
	*service
}

type DestinationsPage struct {
	APIPage
	Destinations []Destination `json:"destinations"`
}

type DestinationTransformation struct {
	DestinationID    string    `json:"destinationId"`
	TransformationID string    `json:"transformationId"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

func (s *destinations) transformationPath(destinationID string) string {
	return strings.Join([]string{s.basePath, destinationID, "transformation"}, "/")
}

func (s *destinations) ConnectTransformation(ctx context.Context, destinationID, transformationID string) (*DestinationTransformation, error) {
	body, err := json.Marshal(map[string]string{"transformationId": transformationID})
	if err != nil {
		return nil, err
	}

	res, err := s.client.Do(ctx, "PUT", s.transformationPath(destinationID), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	result := &DestinationTransformation{}
	if err = json.Unmarshal(res, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *destinations) DisconnectTransformation(ctx context.Context, destinationID string) (*DestinationTransformation, error) {
	res, err := s.client.Do(ctx, "DELETE", s.transformationPath(destinationID), nil)
	if err != nil {
		return nil, err
	}

	result := &DestinationTransformation{}
	if err = json.Unmarshal(res, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *destinations) GetTransformation(ctx context.Context, destinationID string) (*DestinationTransformation, error) {
	res, err := s.client.Do(ctx, "GET", s.transformationPath(destinationID), nil)
	if err != nil {
		return nil, err
	}

	result := &DestinationTransformation{}
	if err = json.Unmarshal(res, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *destinations) Next(ctx context.Context, paging Paging) (*DestinationsPage, error) {
	page := &DestinationsPage{}
	ok, err := s.service.next(ctx, paging, page)
	if !ok {
		page = nil
	}
	return page, err
}

func (s *destinations) List(ctx context.Context) (*DestinationsPage, error) {
	page := &DestinationsPage{}
	if err := s.list(ctx, page); err != nil {
		return nil, err
	}

	return page, nil
}

func (s *destinations) Get(ctx context.Context, id string) (*Destination, error) {
	response := struct{ Destination *Destination }{}
	if err := s.get(ctx, id, &response); err != nil {
		return nil, err
	}

	return response.Destination, nil
}

func (s *destinations) Create(ctx context.Context, destination *Destination) (*Destination, error) {
	// copy input and remove fields that should not be in request body without modifying input
	dst := *destination
	dst.ID = ""

	response := struct{ Destination *Destination }{}
	if err := s.create(ctx, &dst, &response); err != nil {
		return nil, err
	}

	return response.Destination, nil
}

func (s *destinations) Update(ctx context.Context, destination *Destination) (*Destination, error) {
	// copy input and remove ID from request body without modifying input
	dst := *destination
	dst.ID = ""

	response := struct{ Destination *Destination }{}
	if err := s.update(ctx, destination.ID, &dst, &response); err != nil {
		return nil, err
	}

	return response.Destination, nil
}

func (s *destinations) Delete(ctx context.Context, id string) error {
	return s.service.delete(ctx, id)
}
