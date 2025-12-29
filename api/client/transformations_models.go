package client

// Transformation represents a transformation in the API
type Transformation struct {
	ID          string   `json:"id,omitempty"`
	VersionID   string   `json:"versionId,omitempty"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Code        string   `json:"code"`
	Language    string   `json:"language"`
	Imports     []string `json:"imports,omitempty"`
	WorkspaceID string   `json:"workspaceId,omitempty"`
	ExternalID  string   `json:"externalId,omitempty"`
	CreatedAt   string   `json:"createdAt,omitempty"`
	CreatedBy   string   `json:"createdBy,omitempty"`
}

// TransformationLibrary represents a transformation library in the API
type TransformationLibrary struct {
	ID          string `json:"id,omitempty"`
	VersionID   string `json:"versionId,omitempty"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Code        string `json:"code"`
	Language    string `json:"language"`
	HandleName  string `json:"importName"`
	WorkspaceID string `json:"workspaceId,omitempty"`
	ExternalID  string `json:"externalId,omitempty"`
	CreatedAt   string `json:"createdAt,omitempty"`
	CreatedBy   string `json:"createdBy,omitempty"`
}

// CreateTransformationRequest is the request body for creating/updating a transformation
type CreateTransformationRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Code        string `json:"code"`
	Language    string `json:"language"`
	TestEvents  []any  `json:"testEvents,omitempty"`
	ExternalID  string `json:"externalId,omitempty"`
}

// CreateLibraryRequest is the request body for creating/updating a transformation library
type CreateLibraryRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Code        string `json:"code"`
	Language    string `json:"language"`
	ImportName  string `json:"importName,omitempty"`
	ExternalID  string `json:"externalId,omitempty"`
}

// TransformationVersionInput represents a transformation version for batch publishing
type TransformationVersionInput struct {
	VersionID string `json:"versionId"`
	TestInput []any  `json:"testInput,omitempty"`
}

// LibraryVersionInput represents a library version for batch publishing
type LibraryVersionInput struct {
	VersionID string `json:"versionId"`
}

// BatchPublishRequest is the request body for batch publishing transformations and libraries
type BatchPublishRequest struct {
	Transformations []TransformationVersionInput `json:"transformations"`
	Libraries       []LibraryVersionInput        `json:"libraries"`
}

// BatchPublishResponse is the response from the batch publish endpoint
type BatchPublishResponse struct {
	Published bool `json:"published"`
}

// TransformationsListResponse wraps the list of transformations from the API
type TransformationsListResponse struct {
	Transformations []Transformation `json:"transformations"`
}

// LibrariesListResponse wraps the list of libraries from the API
type LibrariesListResponse struct {
	Libraries []TransformationLibrary `json:"libraries"`
}
