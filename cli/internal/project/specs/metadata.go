package specs

import (
	"encoding/json"
	"fmt"

	"github.com/go-viper/mapstructure/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

// Metadata represents the common metadata fields for all specs
type Metadata struct {
	Name   string                    `yaml:"name" json:"name,omitempty" validate:"required"`
	Import *WorkspacesImportMetadata `yaml:"import" json:"import,omitempty"`
}

// WorkspacesImportMetadata holds import spec metadata for a set of workspaces
type WorkspacesImportMetadata struct {
	Workspaces []WorkspaceImportMetadata `yaml:"workspaces" json:"workspaces,omitempty" validate:"dive"`
}

// WorkspaceImportMetadata holds import spec metadata for a single workspace
type WorkspaceImportMetadata struct {
	WorkspaceID string      `yaml:"workspace_id" json:"workspace_id" validate:"required"`
	Resources   []ImportIds `yaml:"resources" json:"resources,omitempty" validate:"dive"`
}

// ImportIds holds the local and remote IDs for a resource to be imported, as specified in import spec metadata
type ImportIds struct {
	// Deprecated: Use URN instead for new providers.
	LocalID string `yaml:"local_id,omitempty" json:"local_id,omitempty" validate:"required"`
	// URN identifies the local resource (format: "resource-type:resource-id")
	URN      string `yaml:"urn,omitempty" json:"urn,omitempty"`
	RemoteID string `yaml:"remote_id" json:"remote_id" validate:"required"`
}

// Validate checks that ImportIds has valid field combinations
func (i *ImportIds) Validate() error {
	hasLocalID := i.LocalID != ""
	hasURN := i.URN != ""

	if hasLocalID && hasURN {
		return fmt.Errorf("urn and local_id are mutually exclusive")
	}
	if !hasLocalID && !hasURN {
		return fmt.Errorf("either urn or local_id must be set")
	}
	if i.RemoteID == "" {
		return fmt.Errorf("remote_id is required")
	}
	return nil
}

// Validate checks that all required fields are present in the Metadata
func (m *Metadata) Validate() error {
	if m.Import != nil {
		for idx, ws := range m.Import.Workspaces {
			if ws.WorkspaceID == "" {
				return fmt.Errorf("missing required field 'workspace_id' in import metadata, workspace index %d", idx)
			}
			for _, res := range ws.Resources {
				if err := res.Validate(); err != nil {
					return fmt.Errorf("invalid import resource in workspace '%s': %w", ws.WorkspaceID, err)
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

// MigrateImportMetadataToURN converts LocalID-based import metadata to URN format
func MigrateImportMetadataToURN(spec *Spec, resourceType string) error {
	metadata, err := spec.CommonMetadata()
	if err != nil {
		return err
	}

	if metadata.Import == nil {
		return nil
	}

	for wi, workspace := range metadata.Import.Workspaces {
		for ri, resource := range workspace.Resources {
			// Skip if already using URN
			if resource.URN != "" {
				continue
			}

			// Convert LocalID to URN
			if resource.LocalID != "" {
				urn := resources.URN(resource.LocalID, resourceType)
				metadata.Import.Workspaces[wi].Resources[ri].URN = urn
				// Clear LocalID for clean migration
				metadata.Import.Workspaces[wi].Resources[ri].LocalID = ""
			}
		}
	}

	// Update spec metadata
	metadataMap, err := metadata.ToMap()
	if err != nil {
		return err
	}
	spec.Metadata = metadataMap

	return nil
}
