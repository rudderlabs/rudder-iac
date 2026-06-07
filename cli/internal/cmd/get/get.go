package get

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/resourceops"
	"github.com/spf13/cobra"
)

// Composite is the minimal seam the get command needs from the composite
// provider. *provider.CompositeProvider satisfies both methods.
type Composite interface {
	ProviderForType(resourceType string) (provider.Provider, error)
	SupportedTypes() []string
}

// GetOptions holds the parsed flag values for the get command.
type GetOptions struct {
	Output    string
	Managed   bool
	Unmanaged bool
	Selector  map[string]string
}

// NewCmdGet returns the top-level `get` cobra command.
func NewCmdGet() *cobra.Command {
	var (
		output    string
		managed   bool
		unmanaged bool
		selector  []string
	)

	cmd := &cobra.Command{
		Use:   "get <type> [<id>]",
		Short: "Get or list resources by type",
		Long: `Get or list remote resources of a given type.

Examples:
  # List all event-stream sources (table by default)
  rudder-cli get event-stream-source

  # List only managed sources as JSON
  rudder-cli get event-stream-source --managed -o json

  # Print a single source as YAML
  rudder-cli get event-stream-source my-source -o yaml`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := GetOptions{
				Output:    output,
				Managed:   managed,
				Unmanaged: unmanaged,
				Selector:  parseSelector(selector),
			}

			var err error
			defer func() {
				telemetry.TrackCommand("get", err,
					telemetry.KV{K: "type", V: firstArg(args)},
					telemetry.KV{K: "output", V: output},
					telemetry.KV{K: "managed", V: managed},
					telemetry.KV{K: "unmanaged", V: unmanaged},
				)
			}()

			d, err := app.NewDeps()
			if err != nil {
				return err
			}

			cp, ok := d.CompositeProvider().(Composite)
			if !ok {
				return fmt.Errorf("internal error: composite provider does not implement the required interface")
			}

			err = RunGet(cmd.Context(), cmd.OutOrStdout(), cp, args, opts)
			return err
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "table", "Output format: table, yaml, or json")
	cmd.Flags().BoolVar(&managed, "managed", false, "Show only managed resources")
	cmd.Flags().BoolVar(&unmanaged, "unmanaged", false, "Show only unmanaged resources")
	cmd.Flags().StringArrayVarP(&selector, "selector", "l", nil, "Filter by label (key=value, repeatable)")

	return cmd
}

// RunGet is the testable core. It validates inputs, dispatches to list or single-
// resource paths, and writes all output to out.
func RunGet(ctx context.Context, out io.Writer, cp Composite, args []string, opts GetOptions) error {
	if opts.Managed && opts.Unmanaged {
		return fmt.Errorf("--managed and --unmanaged are mutually exclusive")
	}

	resourceType := args[0]

	if err := validateType(cp, resourceType); err != nil {
		return err
	}

	prov, err := cp.ProviderForType(resourceType)
	if err != nil {
		return fmt.Errorf("resolving provider for %q: %w", resourceType, err)
	}

	scope := toScope(opts)

	// Warn when unmanaged is requested but the provider can't enumerate them.
	if scope == resourceops.ScopeUnmanaged || scope == resourceops.ScopeAll {
		if !resourceops.SupportsUnmanaged(prov) {
			_, _ = fmt.Fprintf(out, "note: provider for %q does not support listing unmanaged resources; only managed resources will be shown\n", resourceType)
		}
	}

	if len(args) == 2 {
		return runSingle(ctx, out, prov, resourceType, args[1], opts)
	}

	return runList(ctx, out, prov, resourceType, scope, opts)
}

// runList fetches all rows and renders them.
func runList(ctx context.Context, out io.Writer, prov provider.Provider, resourceType string, scope resourceops.Scope, opts GetOptions) error {
	rows, err := resourceops.ListRows(ctx, prov, resourceType, scope)
	if err != nil {
		return fmt.Errorf("listing %s: %w", resourceType, err)
	}

	switch opts.Output {
	case "json":
		return renderRowsJSON(out, rows)
	default:
		return renderRowsTable(out, rows)
	}
}

// runSingle fetches and renders a single resource by its id.
func runSingle(ctx context.Context, out io.Writer, prov provider.Provider, resourceType, id string, opts GetOptions) error {
	switch opts.Output {
	case "yaml":
		s, err := resourceops.SpecYAML(ctx, prov, resourceType, id)
		if err != nil {
			return err
		}
		_, err = fmt.Fprint(out, s)
		return err

	case "json":
		s, err := resourceops.SpecJSON(ctx, prov, resourceType, id)
		if err != nil {
			return err
		}
		_, err = fmt.Fprint(out, s)
		return err

	default:
		// Table: list all rows then filter to the matching row.
		rows, err := resourceops.ListRows(ctx, prov, resourceType, resourceops.ScopeAll)
		if err != nil {
			return fmt.Errorf("fetching %s: %w", resourceType, err)
		}
		var match *resourceops.Row
		for i := range rows {
			if rows[i].ExternalID == id || rows[i].RemoteID == id {
				match = &rows[i]
				break
			}
		}
		if match == nil {
			return fmt.Errorf("%s %q: %w", resourceType, id, resourceops.ErrResourceNotFound)
		}
		return renderRowsTable(out, []resourceops.Row{*match})
	}
}

// renderRowsTable writes rows as a tab-separated table to out so that test
// code can capture the output (unlike ui.PrintTable which writes to stdout).
func renderRowsTable(out io.Writer, rows []resourceops.Row) error {
	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "EXTERNAL-ID\tREMOTE-ID\tNAME\tMANAGED")
	for _, r := range rows {
		managed := "no"
		if r.Managed {
			managed = "yes"
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", r.ExternalID, r.RemoteID, r.Name, managed)
	}
	return w.Flush()
}

// rowView is the JSON-serializable view of a Row.
type rowView struct {
	ExternalID string `json:"external_id"`
	RemoteID   string `json:"remote_id"`
	Name       string `json:"name"`
	Managed    bool   `json:"managed"`
}

// renderRowsJSON marshals rows to indented JSON.
func renderRowsJSON(out io.Writer, rows []resourceops.Row) error {
	views := make([]rowView, len(rows))
	for i, r := range rows {
		views[i] = rowView{
			ExternalID: r.ExternalID,
			RemoteID:   r.RemoteID,
			Name:       r.Name,
			Managed:    r.Managed,
		}
	}
	b, err := json.MarshalIndent(views, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding JSON: %w", err)
	}
	_, err = out.Write(b)
	return err
}

// validateType returns an error listing valid types when resourceType is not
// registered with cp.
func validateType(cp Composite, resourceType string) error {
	if cp == nil {
		return fmt.Errorf("no provider configured")
	}
	for _, t := range cp.SupportedTypes() {
		if t == resourceType {
			return nil
		}
	}
	return fmt.Errorf("unknown resource type %q; valid types: %s",
		resourceType, strings.Join(cp.SupportedTypes(), ", "))
}

// toScope converts the managed/unmanaged flag pair into a resourceops.Scope.
func toScope(opts GetOptions) resourceops.Scope {
	switch {
	case opts.Managed:
		return resourceops.ScopeManaged
	case opts.Unmanaged:
		return resourceops.ScopeUnmanaged
	default:
		return resourceops.ScopeAll
	}
}

func parseSelector(pairs []string) map[string]string {
	if len(pairs) == 0 {
		return nil
	}
	m := make(map[string]string, len(pairs))
	for _, pair := range pairs {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) == 2 {
			m[kv[0]] = kv[1]
		}
	}
	return m
}

func firstArg(args []string) string {
	if len(args) > 0 {
		return args[0]
	}
	return ""
}
