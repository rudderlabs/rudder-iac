package importremote

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-viper/mapstructure/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"gopkg.in/yaml.v3"
)

const (
	CLIVersion = "rudder/v0.1"
)

type ImportIds struct {
	LocalID  string `yaml:"local_id" mapstructure:"local_id"`
	RemoteID string `yaml:"remote_id" mapstructure:"remote_id"`
}

type ImportMetadata struct {
	WorkspaceID string      `yaml:"workspace_id" mapstructure:"workspace_id"`
	Name        string      `yaml:"name" mapstructure:"name"`
	ImportIds   []ImportIds `yaml:"import_ids" mapstructure:"import_ids"`
}

type ImportData struct {
	ResourceData *resources.ResourceData
	Metadata     ImportMetadata

type ImportProvider interface {
	Import(ctx context.Context, resourceType string, args ImportArgs) ([]ImportData, error)
}

type ImportArgs struct {
	RemoteID    string
	LocalID     string
	WorkspaceID string
}

func Import(ctx context.Context, resourceType string, resources []ImportData, location string) error {
	for _, resourceData := range resources {
		metadata := make(map[string]interface{})
		err := mapstructure.Decode(resourceData.Metadata, &metadata)
		if err != nil {
			return err
		}
		spec := &specs.Spec{
			Version:  CLIVersion,
			Kind:     resourceType,
			Metadata: metadata,
			Spec:     *resourceData.ResourceData,
		}
		// write spec to file
		specYAML, err := yaml.Marshal(spec)
		if err != nil {
			return err
		}
		specPath := filepath.Join(location, fmt.Sprintf("%s.yaml", resourceData.Metadata.Name))
		err = os.WriteFile(specPath, specYAML, 0644)
		if err != nil {
			return err
		}
	}

	return nil
}
