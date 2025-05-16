package importer

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
	"github.com/rudderlabs/rudder-iac/cli/internal/workspace"
)

type ImportProvider interface {
	List(ctx context.Context, resourceType string) ([]*workspace.Resource, error)
	Template(ctx context.Context, resource *workspace.Resource) ([]byte, error)
	ImportState(ctx context.Context, resource *workspace.Resource) (*state.ResourceState, error)
	PutResourceState(ctx context.Context, urn string, state *state.ResourceState) error
}

type Importer struct {
	outputDir string
	provider  ImportProvider
}

type ImportData struct {
	LocalID  string
	Resource *workspace.Resource
}

func New(outputDir string, provider ImportProvider) *Importer {
	return &Importer{
		outputDir: outputDir,
		provider:  provider,
	}
}

func (i *Importer) Import(ctx context.Context, resourceType string) error {
	selectedResource, err := i.selectResource(ctx, resourceType)
	if err != nil {
		return fmt.Errorf("failed to select resource: %w", err)
	}

	localID := i.sanitizeResourceName(selectedResource.Name)
	path, content, err := i.generateFile(ctx, localID, selectedResource)
	if err != nil {
		return fmt.Errorf("failed to generate file for resource %s: %w", selectedResource.ID, err)
	}

	if err := i.writeFile(path, content); err != nil {
		return fmt.Errorf("failed to write file %s: %w", path, err)
	}

	ui.PrintSuccess(fmt.Sprintf("Created file: %s", path))

	if err := i.setState(ctx, localID, selectedResource); err != nil {
		return fmt.Errorf("failed to set state for resource %s: %w", selectedResource.ID, err)
	}

	ui.PrintSuccess(fmt.Sprintf("Set state for resource %s", resources.URN(localID, selectedResource.Type)))

	return nil
}

func (i *Importer) sanitizeResourceName(name string) string {
	// Convert to lowercase first
	name = strings.ToLower(name)

	// Replace non-alphanumeric characters with underscores
	reg := regexp.MustCompile("[^a-z0-9]+")
	return reg.ReplaceAllString(name, "_")
}

func (i *Importer) selectResource(ctx context.Context, resourceType string) (*workspace.Resource, error) {
	resourceList, err := i.provider.List(ctx, resourceType)
	if err != nil {
		return nil, fmt.Errorf("failed to list resources: %w", err)
	}

	if len(resourceList) == 0 {
		return nil, fmt.Errorf("no resources of type %s found", resourceType)
	}

	options := make([]ui.Option, len(resourceList))
	for i, resource := range resourceList {
		options[i] = ui.Option{
			ID:      resource.ID,
			Display: resource.Name,
		}
	}

	selectedID, err := ui.Select("Select a resource to import:", options)
	if err != nil {
		return nil, fmt.Errorf("failed to select resource: %w", err)
	}

	for _, resource := range resourceList {
		if resource.ID == selectedID {
			return resource, nil
		}
	}

	return nil, fmt.Errorf("selected resource not found")
}

func (i *Importer) generateFile(ctx context.Context, localID string, resource *workspace.Resource) (string, []byte, error) {
	importData := &ImportData{
		LocalID:  localID,
		Resource: resource,
	}

	path := fmt.Sprintf("%s.yaml", localID)
	template, err := i.provider.Template(ctx, resource)
	if err != nil {
		return "", nil, fmt.Errorf("failed to retrieve file template for resource %s: %w", resource.ID, err)
	}

	content, err := generateFromTemplate(template, importData)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate files for resource %s: %w", resource.ID, err)
	}

	return path, content, err
}

func (i *Importer) writeFile(path string, content []byte) error {
	fullPath := filepath.Join(i.outputDir, path)

	// Create parent directories if they don't exist
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory for %s: %w", path, err)
	}

	// Write the file
	if err := os.WriteFile(fullPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", path, err)
	}

	return nil
}

func (i *Importer) setState(ctx context.Context, localID string, resource *workspace.Resource) error {
	stateData, err := i.provider.ImportState(ctx, resource)
	if err != nil {
		return fmt.Errorf("failed to compute state %s: %w", stateData, err)
	}

	// Override the ID to use the local ID while keeping all other state
	stateData.ID = localID

	urn := resources.URN(localID, resource.Type)
	fmt.Println("Setting state for resource:", urn)
	// serialize the state to a string or JSON format
	resourceStateJSON, _ := json.Marshal(stateData)
	fmt.Println("Resource state:", string(resourceStateJSON))
	if err := i.provider.PutResourceState(ctx, urn, stateData); err != nil {
		return fmt.Errorf("failed to commit state for resource %s: %w", resource.ID, err)
	}

	return nil
}
