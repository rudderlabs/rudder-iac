package importremote

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-viper/mapstructure/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"gopkg.in/yaml.v3"
)

const (
	SpecVersion = "rudder/v0.1"
)

type ImportIds struct {
	LocalID  string `yaml:"local_id" mapstructure:"local_id"`
	RemoteID string `yaml:"remote_id" mapstructure:"remote_id"`
}

type Metadata struct {
	Name   string                   `yaml:"name" mapstructure:"name"`
	Import WorkspacesImportMetadata `yaml:"import" mapstructure:"import"`
}

type WorkspacesImportMetadata struct {
	Workspaces []WorkspaceImportMetadata `yaml:"workspaces" mapstructure:"workspaces"`
}
type WorkspaceImportMetadata struct {
	WorkspaceID string      `yaml:"workspace_id" mapstructure:"workspace_id"`
	Resources   []ImportIds `yaml:"resources" mapstructure:"resources"`
}

type ImportData struct {
	ResourceData *resources.ResourceData
	Metadata     Metadata
	ResourceType string
}

type ImportArgs struct {
	RemoteID string
	LocalID  string
}

func Import(ctx context.Context, resources []ImportData, location string) error {
	for _, resourceData := range resources {
		metadata := make(map[string]interface{})
		err := mapstructure.Decode(resourceData.Metadata, &metadata)
		if err != nil {
			return err
		}

		spec := &specs.Spec{
			Version:  SpecVersion,
			Kind:     resourceData.ResourceType,
			Metadata: metadata,
			Spec:     *resourceData.ResourceData,
		}

		specYAML, err := encodeYaml(spec)
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

func encodeYaml(spec *specs.Spec) ([]byte, error) {
	var node yaml.Node
	err := node.Encode(spec)
	if err != nil {
		return nil, err
	}
	forceStringQuotes(&node)

	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)

	err = encoder.Encode(&node)
	if err != nil {
		return nil, err
	}
	err = encoder.Close()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Function to force quote only string values (not keys)
func forceStringQuotes(node *yaml.Node) {
	switch node.Kind {
	case yaml.DocumentNode, yaml.SequenceNode:
		// Process all children
		for _, child := range node.Content {
			forceStringQuotes(child)
		}
	case yaml.MappingNode:
		// For mapping nodes, skip keys (even indices) and only process values (odd indices)
		for i, child := range node.Content {
			if i%2 == 1 { // Only process values (odd indices)
				forceStringQuotes(child)
			} else {
				// Still need to recurse into keys in case they contain nested structures
				// but don't quote the key itself if it's a string
				if child.Kind != yaml.ScalarNode {
					forceStringQuotes(child)
				}
			}
		}
	case yaml.ScalarNode:
		// Only quote if it's a string
		if node.Tag == "!!str" {
			node.Style = yaml.DoubleQuotedStyle
		}
	}
}
