package docs

import (
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"

	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/builtin"
	"github.com/rudderlabs/rudder-iac/cli/internal/specdocs"
)

func newCmdDestination() *cobra.Command {
	var (
		source  string
		id      string
		version int64
	)

	cmd := &cobra.Command{
		Use:   "destination [<type>]",
		Short: "Print the config a destination type accepts",
		Long: heredoc.Doc(`
			Prints a destination type's config keys, allowed values, defaults, secret keys and connection
			modes, read from the definitions compiled into this rudder-cli, so it matches what validate and
			apply accept. Without a type it lists every type. With --source and --id it also names each
			secret's variable and prints a spec to start from.
		`),
		Example: heredoc.Doc(`
			$ rudder-cli docs destination webhook --source javascript --id storefront-webhook --json
		`),
		Args: cobra.MaximumNArgs(1),
	}
	resolveFormat := formatFlags(cmd)

	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		var destType string
		if len(args) == 1 {
			destType = args[0]
		}
		format, formatErr := resolveFormat()
		registry, registryErr := builtin.NewRegistry(true)

		defer func() {
			// Only registered types and known source types are reported, so
			// free text a user typed never reaches telemetry; the ID never does.
			reportedType := ""
			if destType != "" {
				reportedType = unknownValue
				if registry != nil && registry.IsSupported(destType) {
					reportedType = destType
				}
			}
			reportedSource := source
			if source != "" && !specdocs.IsSourceType(source) {
				reportedSource = unknownValue
			}
			telemetry.TrackCommand(telemetry.CommandName(cmd), err, []telemetry.KV{
				{K: "type", V: reportedType},
				{K: "format", V: reportedFormat(format)},
				{K: "source", V: reportedSource},
				{K: "listing", V: destType == ""},
				{K: "has_id", V: id != ""},
			}...)
		}()

		if registryErr != nil {
			return fmt.Errorf("building destination registry: %w", registryErr)
		}
		if formatErr != nil {
			return formatErr
		}

		var (
			out        = cmd.OutOrStdout()
			cliVersion = cmd.Root().Version
			docs       = specdocs.NewDestinations(registry, builtin.IsVerified)
		)

		if destType == "" {
			if id != "" || version != 0 {
				return fail(out, format, cliVersion, &specdocs.Error{
					Code:        specdocs.CodeMissingType,
					Message:     "--id and --definition-version need a destination type",
					Suggestions: []string{},
					ValidTypes:  registry.SupportedTypes(),
				})
			}
			list, err := docs.List(specdocs.ListOptions{CLIVersion: cliVersion, Source: source})
			if err != nil {
				return fail(out, format, cliVersion, err)
			}
			if format == formatJSON {
				return specdocs.WriteJSON(out, list)
			}
			return specdocs.RenderDestinationListMarkdown(out, list)
		}

		doc, err := docs.Describe(specdocs.DescribeOptions{
			CLIVersion: cliVersion,
			Type:       destType,
			Version:    version,
			Source:     source,
			ID:         id,
		})
		if err != nil {
			return fail(out, format, cliVersion, err)
		}
		if format == formatJSON {
			return specdocs.WriteJSON(out, doc)
		}
		return specdocs.RenderDestinationMarkdown(out, doc)
	}

	cmd.Flags().StringVar(&source, "source", "", "Event stream source spec type to narrow the output to, such as javascript")
	cmd.Flags().StringVar(&id, "id", "", "Destination spec ID to name secret variables after")
	cmd.Flags().Int64Var(&version, "definition-version", 0, "Definition version to describe (default: the latest)")
	return cmd
}
