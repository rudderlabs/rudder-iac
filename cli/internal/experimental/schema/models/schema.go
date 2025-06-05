package models

// Schema represents a single schema entry in the schemas file
type Schema struct {
	UID             string                 `json:"uid"`
	WriteKey        string                 `json:"writeKey"`
	EventType       string                 `json:"eventType"`
	EventIdentifier string                 `json:"eventIdentifier"`
	Schema          map[string]interface{} `json:"schema"`
	CreatedAt       string                 `json:"createdAt"`
	LastSeen        string                 `json:"lastSeen"`
	Count           int                    `json:"count"`
}

// SchemasFile represents the entire schemas JSON file structure
type SchemasFile struct {
	Schemas []Schema `json:"schemas"`
} 