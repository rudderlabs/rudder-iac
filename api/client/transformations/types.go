package transformations

// Transformation represents a transformation resource from the API
type Transformation struct {
	ID          string   `json:"id"`
	VersionID   string   `json:"versionId"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Code        string   `json:"code"`
	Language    string   `json:"language"`
	Imports     []string `json:"imports"`
	WorkspaceID string   `json:"workspaceId"`
	ExternalID  string   `json:"externalId,omitempty"`
}

// TransformationLibrary represents a transformation library resource from the API
type TransformationLibrary struct {
	ID          string `json:"id"`
	VersionID   string `json:"versionId"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Code        string `json:"code"`
	Language    string `json:"language"`
	HandleName  string `json:"handleName"`
	WorkspaceID string `json:"workspaceId"`
	ExternalID  string `json:"externalId,omitempty"`
}

// CreateTransformationRequest is the request body for creating/updating transformations
type CreateTransformationRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Code        string  `json:"code"`
	Language    string  `json:"language"`
	TestEvents  []any   `json:"testEvents,omitempty"`
	ExternalID  *string `json:"externalId,omitempty"`
}

// CreateLibraryRequest is the request body for creating/updating libraries
type CreateLibraryRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Code        string  `json:"code"`
	Language    string  `json:"language"`
	ExternalID  *string `json:"externalId,omitempty"`
}

// BatchPublishRequest is the request body for batch publishing transformations and libraries
type BatchPublishRequest struct {
	Transformations []BatchPublishTransformation `json:"transformations,omitempty"`
	Libraries       []BatchPublishLibrary        `json:"libraries,omitempty"`
}

type BatchPublishTransformation struct {
	VersionID string `json:"versionId"`
	TestInput []any  `json:"testInput,omitempty"`
}

type BatchPublishLibrary struct {
	VersionID string `json:"versionId"`
}

// BatchPublishResponse is the response from batch publish
type BatchPublishResponse struct {
	Published bool `json:"published"`
}
