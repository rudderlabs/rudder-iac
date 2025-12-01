package specs

import (
	"encoding/json"
	"fmt"

	"github.com/go-viper/mapstructure/v2"
)

// Metadata represents the common metadata fields for all specs
type Metadata struct {
	Name   string                    `yaml:"name" json:"name,omitempty"`
	Import *WorkspacesImportMetadata `yaml:"import" json:"import,omitempty"`
}

// WorkspacesImportMetadata holds import spec metadata for a set of workspaces
type WorkspacesImportMetadata struct {
	Workspaces []WorkspaceImportMetadata `yaml:"workspaces" json:"workspaces,omitempty"`
}

// WorkspaceImportMetadata holds import spec metadata for a single workspace
type WorkspaceImportMetadata struct {
	WorkspaceID string      `yaml:"workspace_id" json:"workspace_id"`
	Resources   []ImportIds `yaml:"resources" json:"resources,omitempty"`
}

// ImportIds holds the local and remote IDs for a resource to be imported, as specified in import spec metadata
type ImportIds struct {
	LocalID  string `yaml:"local_id" json:"local_id"`
	RemoteID string `yaml:"remote_id" json:"remote_id"`
}

// Validate checks that all required fields are present in the Metadata
func (m *Metadata) Validate() error {
	if m.Import != nil {
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
	}

	return nil
}

// CommonMetadata decodes and returns the common Metadata from the Spec's Metadata map
// It will return an error if decoding fails, but will not further validate the metadata fields.
// If necessary, validation should be performed separately by calling [Metadata.Validate].
func (s *Spec) CommonMetadata() (Metadata, error) {
	var metadata Metadata
	if s.Metadata == nil {
		return metadata, nil
	}

	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		TagName: "yaml",
		Result:  &metadata,
	})

	if err != nil {
		return metadata, nil
	}

	err = decoder.Decode(s.Metadata)
	if err != nil {
		return metadata, fmt.Errorf("failed to decode metadata: %w", err)
	}

	return metadata, nil
}

// ToMap converts the Metadata struct to a map[string]any
// This is useful when creating specs that need metadata in map form
func (m *Metadata) ToMap() (map[string]any, error) {
	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}

	var result map[string]any
	err = json.Unmarshal(b, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}
