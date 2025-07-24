package importutils

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"gopkg.in/yaml.v3"
)

const (
	CLIVersion = "rudder/v0.1"
)

type ImportData struct {
	ResourceData *resources.ResourceData
	Metadata     map[string]interface{}
}

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
		spec := &specs.Spec{
			Version:  CLIVersion,
			Kind:     resourceType,
			Metadata: resourceData.Metadata,
			Spec:     *resourceData.ResourceData,
		}

		name, ok := resourceData.Metadata["name"].(string)
		if !ok {
			return fmt.Errorf("name is required in metadata")
		}
		// write spec to file
		specYAML, err := yaml.Marshal(spec)
		if err != nil {
			return err
		}
		specPath := filepath.Join(location, fmt.Sprintf("%s.yaml", name))
		err = os.WriteFile(specPath, specYAML, 0644)
		if err != nil {
			return err
		}
	}

	return nil
}
