package internal

type Resource struct {
	URN          string
	ID           string
	Type         string
	Data         map[string]interface{}
	Dependencies []string

	ImportMetadata *ResourceImportInfo
}

type ResourceImportInfo struct {
	WorkspaceId string
	RemoteId    string
}

func (i *ResourceImportInfo) IsImport() bool {
	return i.RemoteId != ""
}
