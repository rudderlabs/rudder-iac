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

// GetProvider is the narrowest interface the get command needs from a single
// provider. It deliberately excludes provider.UnmanagedRemoteResourceLoader so
// that resourceops.SupportsUnmanaged can distinguish managed-only providers at
// runtime and print a degraded-mode note.
//
// Production providers (implementing the full provider.Provider interface) satisfy
// this interface automatically. The single-resource path additionally requires
// provider.Exporter; runSingle type-asserts to it and returns an informative error
// if the provider does not export (which cannot happen with real providers).
type GetProvider interface {
	provider.ManagedRemoteResourceLoader
}

// Composite is the minimal seam the get command needs from the composite
// provider. See NewCompositeShim to adapt a *provider.CompositeProvider.
type Composite interface {
	ProviderForType(resourceType string) (GetProvider, error)
	SupportedTypes() []string
}

// typeRouter is satisfied by *provider.CompositeProvider, which exposes
// ProviderForType returning the full provider.Provider.
type typeRouter interface {
	ProviderForType(string) (provider.Provider, error)
	SupportedTypes() []string
}

// compositeShim adapts a typeRouter (e.g. *provider.CompositeProvider) to the
// Composite seam by narrowing the ProviderForType return type to GetProvider.
// Since provider.Provider embeds ManagedRemoteResourceLoader, the value itself
// satisfies GetProvider — only the declared return type differs.
type compositeShim struct{ r typeRouter }

func (s *compositeShim) ProviderForType(t string) (GetProvider, error) {
	return s.r.ProviderForType(t)
}

func (s *compositeShim) SupportedTypes() []string { return s.r.SupportedTypes() }

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

			router, ok := d.CompositeProvider().(typeRouter)
			if !ok {
				return fmt.Errorf("internal error: composite provider does not support per-type routing")
			}

			err = RunGet(cmd.Context(), cmd.OutOrStdout(), &compositeShim{r: router}, args, opts)
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

// runList fetches all rows, applies any selector filter, and renders them.
func runList(ctx context.Context, out io.Writer, prov GetProvider, resourceType string, scope resourceops.Scope, opts GetOptions) error {
	rows, err := resourceops.ListRows(ctx, prov, resourceType, scope)
	if err != nil {
		return fmt.Errorf("listing %s: %w", resourceType, err)
	}

	rows, err = filterRows(rows, opts.Selector)
	if err != nil {
		return err
	}

	switch opts.Output {
	case "json":
		return renderRowsJSON(out, rows)
	default:
		return renderRowsTable(out, rows)
	}
}

// supportedSelectorKeys lists the row columns that -l/--selector can filter on.
// These map directly to the fields of resourceops.Row.
var supportedSelectorKeys = []string{"external-id", "remote-id", "name", "managed"}

// filterRows returns only those rows matching ALL entries in selector (AND semantics).
// String comparisons are case-insensitive. "managed" accepts "true" or "false".
//
// This is a v1 selector that operates over the four listed columns (external-id,
// remote-id, name, managed). Filtering on arbitrary resource fields (full
// lister.Filters) is a documented follow-up.
//
// An unknown key causes an error that lists the supported keys so the caller
// can fix the command rather than silently getting an unfiltered result.
func filterRows(rows []resourceops.Row, selector map[string]string) ([]resourceops.Row, error) {
	if len(selector) == 0 {
		return rows, nil
	}

	// Validate all keys up-front so we never silently ignore unknown keys.
	for k := range selector {
		if !isSupportedSelectorKey(k) {
			return nil, fmt.Errorf("unknown selector key %q; supported keys: %s",
				k, strings.Join(supportedSelectorKeys, ", "))
		}
	}

	out := rows[:0:0] // reuse backing array but start empty
	for _, row := range rows {
		if rowMatchesSelector(row, selector) {
			out = append(out, row)
		}
	}
	return out, nil
}

// isSupportedSelectorKey reports whether key is in supportedSelectorKeys.
func isSupportedSelectorKey(key string) bool {
	for _, k := range supportedSelectorKeys {
		if strings.EqualFold(k, key) {
			return true
		}
	}
	return false
}

// rowMatchesSelector reports whether row satisfies every entry in selector.
func rowMatchesSelector(row resourceops.Row, selector map[string]string) bool {
	for k, v := range selector {
		switch strings.ToLower(k) {
		case "external-id":
			if !strings.EqualFold(row.ExternalID, v) {
				return false
			}
		case "remote-id":
			if !strings.EqualFold(row.RemoteID, v) {
				return false
			}
		case "name":
			if !strings.EqualFold(row.Name, v) {
				return false
			}
		case "managed":
			want := strings.EqualFold(v, "true")
			if row.Managed != want {
				return false
			}
		}
	}
	return true
}

// runSingle fetches and renders a single resource by its id.
// yaml/json output formats require the provider to also implement provider.Provider
// (Exporter + full spec materialization); table output only needs ManagedRemoteResourceLoader.
func runSingle(ctx context.Context, out io.Writer, prov GetProvider, resourceType, id string, opts GetOptions) error {
	switch opts.Output {
	case "yaml", "json":
		// Full spec materialization requires the provider to export.
		fullProv, ok := prov.(provider.Provider)
		if !ok {
			return fmt.Errorf("provider for %q does not support yaml/json single-resource output", resourceType)
		}
		var s string
		var err error
		if opts.Output == "yaml" {
			s, err = resourceops.SpecYAML(ctx, fullProv, resourceType, id)
		} else {
			s, err = resourceops.SpecJSON(ctx, fullProv, resourceType, id)
		}
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
