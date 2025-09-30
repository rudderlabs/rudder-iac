package source

type CreateSourceRequest struct {
	ExternalID string `json:"externalId"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Enabled    bool   `json:"enabled"`
}

type UpdateSourceRequest struct {
	Name    string `json:"name,omitempty"`
	Enabled bool   `json:"enabled,omitempty"`
}

type EventStreamSource struct {
	ID          string `json:"id"`
	ExternalID  string `json:"externalId"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Enabled     bool   `json:"enabled"`
}

type eventStreamSources struct {
	Sources []EventStreamSource `json:"sources"`
}
