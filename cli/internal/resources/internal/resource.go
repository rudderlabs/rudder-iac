package internal

type Resource struct {
	URN  string
	ID   string
	Type string
	// map serialization is deprecated and will be gradually phased out in favor of structured Data
	Data         map[string]any
	RawData      any
	Dependencies []string

	ImportMetadata     *ResourceImportMetadata
	FileMetadata       *ResourceFileMetadata
	AdditionalMetadata map[string]any
}

type ResourceFileMetadata struct {
	MetadataRef string
}

type ResourceImportMetadata struct {
	WorkspaceId string
	RemoteId    string
}
