package lister

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
)

// OutputFormat determines how the lister should format its output.
type OutputFormat string

const (
	// JSONFormat outputs each resource as a JSON object on a new line.
	JSONFormat OutputFormat = "json"
	// TableFormat outputs resources in a human-readable table.
	TableFormat OutputFormat = "table"
	// DetailedFormat outputs each resource with all its details.
	DetailedFormat OutputFormat = "details"
)

// Filters represents a generic way to pass key-value filter criteria.
type Filters map[string]string

type Lister struct {
	Provider     ListProvider
	Format       OutputFormat
	ColumnWidths map[string]int
}

type ListOption func(*Lister)

func WithFormat(format OutputFormat) ListOption {
	return func(l *Lister) {
		l.Format = format
	}
}

func WithColumnWidths(widths map[string]int) ListOption {
	return func(l *Lister) {
		l.ColumnWidths = widths
	}
}

type ListProvider interface {
	List(ctx context.Context, resourceType string, filters Filters) ([]resources.ResourceData, error)
}

func (l *Lister) List(ctx context.Context, resourceType string, filters Filters) error {
	var rs []resources.ResourceData
	var err error

	if l.Format != JSONFormat {
		spinner := ui.NewSpinner(fmt.Sprintf("Fetching %s...", resourceType))
		spinner.Start()
		rs, err = l.Provider.List(ctx, resourceType, filters)
		spinner.Stop()
	} else {
		rs, err = l.Provider.List(ctx, resourceType, filters)
	}

	if err != nil {
		return err
	}

	switch l.Format {
	case JSONFormat:
		return printResourcesAsJSON(rs)
	case TableFormat:
		return printTableWithDetails(rs, l.ColumnWidths)
	default:
		return fmt.Errorf("unknown output format: %s", l.Format)
	}
}

func printResourcesAsJSON(resources []resources.ResourceData) error {
	for _, r := range resources {
		b, err := json.Marshal(r)
		if err != nil {
			return err
		}
		fmt.Println(string(b))
	}
	return nil
}

func New(p ListProvider, opts ...ListOption) *Lister {
	l := &Lister{
		Provider:     p,
		Format:       TableFormat,
		ColumnWidths: map[string]int{},
	}
	for _, opt := range opts {
		opt(l)
	}
	return l
}
