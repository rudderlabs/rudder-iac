package models

import (
	"time"
)

// SchemasResponse represents the API response structure
type SchemasResponse struct {
	Results     []Schema `json:"results"`
	CurrentPage int      `json:"currentPage"`
	HasNext     bool     `json:"hasNext,omitempty"`
}

// Schema represents an individual event schema
type Schema struct {
	UID             string                 `json:"uid"`
	WriteKey        string                 `json:"writeKey"`
	EventType       string                 `json:"eventType"`
	EventIdentifier string                 `json:"eventIdentifier"`
	Schema          map[string]interface{} `json:"schema"`
	CreatedAt       time.Time              `json:"createdAt"`
	LastSeen        time.Time              `json:"lastSeen"`
	Count           int                    `json:"count"`
}

// SchemasFile represents the output file structure
type SchemasFile struct {
	Schemas []Schema `json:"schemas"`
}
