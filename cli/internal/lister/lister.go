package lister

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
)

// Filters represents a generic way to pass key-value filter criteria.
type Filters map[string]string

type Lister struct {
	Provider project.Provider
}

func (l *Lister) List(ctx context.Context, resourceType string, filters Filters) error {
	spinner := ui.NewSpinner(fmt.Sprintf("Fetching %s...", resourceType))
	spinner.Start()
	resources, err := l.Provider.List(ctx, resourceType, filters)
	spinner.Stop()
	if err != nil {
		return err
	}

	return printResources(resources)
}

func printResources(resources []resources.ResourceData) error {
	for _, r := range resources {
		b, err := json.Marshal(r)
		if err != nil {
			return err
		}
		fmt.Println(string(b))
	}
	return nil
}

func New(p project.Provider) *Lister {
	return &Lister{
		Provider: p,
	}
}
