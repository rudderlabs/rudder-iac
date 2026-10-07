package client

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type Source struct {
	ID                   string          `json:"id,omitempty"`
	Name                 string          `json:"name"`
	Type                 string          `json:"type"`
	WriteKey             string          `json:"writeKey,omitempty"`
	IsEnabled            bool            `json:"enabled"`
	Config               json.RawMessage `json:"config"`
	CreatedAt            *time.Time      `json:"createdAt,omitempty"`
	UpdatedAt            *time.Time      `json:"updatedAt,omitempty"`
	GeoEnrichmentEnabled *bool           `json:"geoEnrichmentEnabled,omitempty"`
	Transient            *bool           `json:"transient,omitempty"`
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

// GetAll walks every page of sources. Unlike the event-stream source listing,
// each Source carries its raw Config.
func (s *sources) GetAll(ctx context.Context) ([]Source, error) {
	var all []Source

	page, err := s.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing sources: %w", err)
	}

	for page != nil {
		all = append(all, page.Sources...)
		page, err = s.Next(ctx, page.Paging)
		if err != nil {
			return nil, fmt.Errorf("fetching next sources page: %w", err)
		}
	}
	return all, nil
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
