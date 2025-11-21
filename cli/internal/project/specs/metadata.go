package specs

import (
	"fmt"

	"github.com/go-viper/mapstructure/v2"
)

// Metadata represents the common metadata fields for all specs
type Metadata struct {
	Name   string                   `yaml:"name" mapstructure:"name"`
	Import WorkspacesImportMetadata `yaml:"import" mapstructure:"import"`
}

// WorkspacesImportMetadata holds import spec metadata for a set of workspaces
type WorkspacesImportMetadata struct {
	Workspaces []WorkspaceImportMetadata `yaml:"workspaces" mapstructure:"workspaces"`
}

// WorkspaceImportMetadata holds import spec metadata for a single workspace
type WorkspaceImportMetadata struct {
	WorkspaceID string      `yaml:"workspace_id" mapstructure:"workspace_id"`
	Resources   []ImportIds `yaml:"resources" mapstructure:"resources"`
}

// ImportIds holds the local and remote IDs for a resource to be imported, as specified in import spec metadata
type ImportIds struct {
	LocalID  string `yaml:"local_id" mapstructure:"local_id"`
	RemoteID string `yaml:"remote_id" mapstructure:"remote_id"`
}

// Validate checks that all required fields are present in the Metadata
func (m *Metadata) Validate() error {
	for idx, ws := range m.Import.Workspaces {
		if ws.WorkspaceID == "" {
			return fmt.Errorf("missing required field 'workspace_id' in import metadata, workspace index %d", idx)
		}
		for _, res := range ws.Resources {
			if res.LocalID == "" {
				return fmt.Errorf("missing required field 'local_id' in import metadata for workspace '%s'", ws.WorkspaceID)
			}
			if res.RemoteID == "" {
				return fmt.Errorf("missing required field 'remote_id' in import metadata for workspace '%s', local_id '%s'", ws.WorkspaceID, res.LocalID)
			}
		}
	}

	return nil
}

// CommonMetadata decodes and returns the common Metadata from the Spec
// It will return an error if decoding fails or if required fields are missing
func (s *Spec) CommonMetadata() (Metadata, error) {
	var metadata Metadata
	err := mapstructure.Decode(s.Metadata, &metadata)
	if err != nil {
		return metadata, fmt.Errorf("failed to decode metadata: %w", err)
	}

	err = metadata.Validate()
	if err != nil {
		return metadata, fmt.Errorf("invalid spec metadata: %w", err)
	}

	return metadata, nil
}

// ToMap converts the Metadata struct to a map[string]any
// This is useful when creating specs that need metadata in map form
func (m *Metadata) ToMap() (map[string]any, error) {
	result := make(map[string]any)

	if m.Name != "" {
		result["name"] = m.Name
	}

	if len(m.Import.Workspaces) > 0 {
		importMap := make(map[string]any)
		workspaces := make([]any, 0, len(m.Import.Workspaces))

		for _, ws := range m.Import.Workspaces {
			wsMap := make(map[string]any)
			wsMap["workspace_id"] = ws.WorkspaceID

			resources := make([]any, 0, len(ws.Resources))

			if len(ws.Resources) >= 0 {
				for _, res := range ws.Resources {
					resMap := make(map[string]any)
					resMap["local_id"] = res.LocalID
					resMap["remote_id"] = res.RemoteID
					resources = append(resources, resMap)
				}
				wsMap["resources"] = resources
			}

			workspaces = append(workspaces, wsMap)
		}

		importMap["workspaces"] = workspaces
		result["import"] = importMap
	}

	return result, nil
}
