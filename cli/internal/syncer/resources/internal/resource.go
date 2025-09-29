package internal

type Resource struct {
	URN          string
	ID           string
	Type         string
	Data         map[string]interface{}
	Dependencies []string

	ImportMetadata *ResourceImportMetadata
	FileMetadata   *ResourceFileMetadata
}

type ResourceFileMetadata struct {
	FilePath    string
	MetadataRef string
}

type ResourceImportMetadata struct {
	WorkspaceId string
	RemoteId    string
}
