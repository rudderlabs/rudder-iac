package lister

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"sort"

	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
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
	case DetailedFormat:
		return printResourceDetails(rs)
	default:
		return fmt.Errorf("unknown output format: %s", l.Format)
	}
}

// printResourceDetails prints every field of each resource as aligned
// key/value lines. Unlike the table it needs no terminal: the table is a
// bubbles TUI and fails outright without a TTY, which a detail view read from
// a script or a pipe must not do.
//
// Identifying fields lead, in a fixed order, because a detail view is read
// top-down; everything else follows alphabetically so two runs of the same
// command print the same thing.
func printResourceDetails(rs []resources.ResourceData) error {
	lead := []string{"id", "externalId", "name", "kind", "enabled"}

	for i, r := range rs {
		if i > 0 {
			fmt.Println()
		}

		ordered := make([]string, 0, len(r))
		for _, key := range lead {
			if _, ok := r[key]; ok {
				ordered = append(ordered, key)
			}
		}
		rest := make([]string, 0, len(r))
		for key := range r {
			if !slices.Contains(lead, key) {
				rest = append(rest, key)
			}
		}
		sort.Strings(rest)
		ordered = append(ordered, rest...)

		width := 0
		for _, key := range ordered {
			if len(key) > width {
				width = len(key)
			}
		}

		for _, key := range ordered {
			fmt.Printf("%-*s  %s\n", width, key, detailValue(r[key]))
		}
	}
	return nil
}

// detailValue renders one field. Nested values go through JSON rather than %v
// so a config map reads as a config map and not as Go's map syntax.
func detailValue(v any) string {
	switch typed := v.(type) {
	case nil:
		return ""
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	}

	switch reflect.ValueOf(v).Kind() {
	case reflect.Map, reflect.Slice, reflect.Array, reflect.Struct, reflect.Ptr:
		if b, err := json.Marshal(v); err == nil {
			return string(b)
		}
	}
	return fmt.Sprintf("%v", v)
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
