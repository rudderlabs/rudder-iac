package client

import (
	"context"
	"encoding/json"
	"time"
)

type Source struct {
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name"`
	Type      string          `json:"type"`
	WriteKey  string          `json:"writeKey,omitempty"`
	IsEnabled bool            `json:"enabled"`
	Config    json.RawMessage `json:"config"`
	CreatedAt *time.Time      `json:"createdAt,omitempty"`
	UpdatedAt *time.Time      `json:"updatedAt,omitempty"`
}

type sources struct {
	*service
}

type SourcesPage struct {
	APIPage
	Sources []Source `json:"sources"`
}

func (s *sources) Next(ctx context.Context, paging Paging) (*SourcesPage, error) {
	page := &SourcesPage{}
	ok, err := s.service.next(ctx, paging, page)
	if !ok {
		page = nil
	}
	return page, err
}

func (s *sources) List(ctx context.Context) (*SourcesPage, error) {
	page := &SourcesPage{}
	if err := s.list(ctx, page); err != nil {
		return nil, err
	}

	return page, nil
}

func (s *sources) Get(ctx context.Context, id string) (*Source, error) {
	response := struct{ Source *Source }{}
	if err := s.get(ctx, id, &response); err != nil {
		return nil, err
	}

	return response.Source, nil
}

func (s *sources) Create(ctx context.Context, source *Source) (*Source, error) {
	// copy input and remove fields that should not be in request body without modifying input
	src := *source
	src.ID = ""
	src.WriteKey = ""

	response := struct{ Source *Source }{}
	if err := s.create(ctx, &src, &response); err != nil {
		return nil, err
	}

	return response.Source, nil
}

func (s *sources) Update(ctx context.Context, source *Source) (*Source, error) {
	// copy input and remove ID from request body without modifying input
	src := *source
	src.ID = ""

	response := struct{ Source *Source }{}
	if err := s.update(ctx, source.ID, &src, &response); err != nil {
		return nil, err
	}

	return response.Source, nil
}

func (s *sources) Delete(ctx context.Context, id string) error {
	return s.service.delete(ctx, id)
}
