package transformations

// Transformation represents a transformation resource from the API
type Transformation struct {
	ID          string   `json:"id"`
	VersionID   string   `json:"versionId"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
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
	Description string `json:"description"`
	Code        string `json:"code"`
	Language    string `json:"language"`
	ImportName  string `json:"importName"`
	WorkspaceID string `json:"workspaceId"`
	ExternalID  string `json:"externalId,omitempty"`
}

// CreateTransformationRequest is the request body for creating a transformation
type CreateTransformationRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Code        string `json:"code"`
	Language    string `json:"language"`
	ExternalID  string `json:"externalId"`
}

// UpdateTransformationRequest is the request body for updating a transformation
type UpdateTransformationRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Code        string `json:"code"`
	Language    string `json:"language"`
}

// CreateLibraryRequest is the request body for creating a library
type CreateLibraryRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Code        string `json:"code"`
	Language    string `json:"language"`
	ExternalID  string `json:"externalId"`
}

// UpdateLibraryRequest is the request body for updating a library
type UpdateLibraryRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Code        string `json:"code"`
	Language    string `json:"language"`
}

// BatchPublishRequest is the request body for batch publishing transformations and libraries
type BatchPublishRequest struct {
	Transformations []BatchPublishTransformation `json:"transformations,omitempty"`
	Libraries       []BatchPublishLibrary        `json:"libraries,omitempty"`
}

// BatchPublishTransformation represents a transformation to publish
type BatchPublishTransformation struct {
	VersionID string `json:"versionId"`
	TestInput []any  `json:"testInput,omitempty"`
}

// BatchPublishLibrary represents a library to publish
type BatchPublishLibrary struct {
	VersionID string `json:"versionId"`
}

type SetExternalIDRequest struct {
	ExternalID string `json:"externalId"`
}
