// Package docs holds the `docs` command group, which prints reference
// documentation read from the definitions compiled into this binary.
package docs

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"

	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/cmderrors"
	"github.com/rudderlabs/rudder-iac/cli/internal/specdocs"
)

const (
	formatMarkdown = "md"
	formatJSON     = "json"

	// unknownValue is what telemetry reports in place of a value the user
	// typed that the CLI does not recognise.
	unknownValue = "unknown"
)

func NewCmdDocs() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "docs",
		Short: "Print reference documentation generated from this rudder-cli",
		Long: "Prints reference documentation generated from the code compiled into this rudder-cli, " +
			"so it describes exactly what this version accepts. It needs no login and makes no API call.",
		Example: heredoc.Doc(`
			$ rudder-cli docs source
			$ rudder-cli docs destination --source javascript
			$ rudder-cli docs destination webhook --source javascript --id storefront-webhook --json
		`),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(newCmdDestination())
	cmd.AddCommand(newCmdSource())
	return cmd
}

// formatFlags registers --format and its -j/--json shorthand, which validate
// also offers, and returns a func that resolves the chosen format. The func,
// not cobra, rejects --json with a --format other than json, so that error
// comes as the JSON document --json asks for.
func formatFlags(cmd *cobra.Command) func() (string, error) {
	var (
		format string
		asJSON bool
	)
	cmd.Flags().StringVar(&format, "format", formatMarkdown, "Output format: md or json")
	cmd.Flags().BoolVarP(&asJSON, "json", "j", false, "Output JSON, the same as --format json")

	return func() (string, error) {
		switch {
		case asJSON && cmd.Flags().Changed("format") && format != formatJSON:
			return formatJSON, invalidArguments(fmt.Sprintf("--json and --format %q ask for different formats; pass one of them", format))
		case asJSON:
			return formatJSON, nil
		case format != formatMarkdown && format != formatJSON:
			return format, invalidArguments(fmt.Sprintf("--format must be md or json, got %q", format))
		default:
			return format, nil
		}
	}
}

func invalidArguments(message string) *specdocs.Error {
	return &specdocs.Error{Code: specdocs.CodeInvalidArguments, Message: message, Suggestions: []string{}}
}

func reportedFormat(format string) string {
	if format == formatMarkdown || format == formatJSON {
		return format
	}
	return unknownValue
}

// fail reports a specdocs input error. Under JSON the error document goes to
// stdout, where the caller already reads, and stderr stays quiet; otherwise
// the root prints the message.
func fail(w io.Writer, format, cliVersion string, err error) error {
	var docErr *specdocs.Error
	if !errors.As(err, &docErr) {
		return err
	}
	if format != formatJSON {
		return errors.New(plainMessage(docErr))
	}
	if writeErr := specdocs.WriteJSON(w, specdocs.NewErrorDoc(cliVersion, docErr)); writeErr != nil {
		return writeErr
	}
	return &cmderrors.SilentError{Err: docErr}
}

func plainMessage(e *specdocs.Error) string {
	message := e.Message + "."
	if len(e.Suggestions) > 0 {
		message = e.Message + "; did you mean " + strings.Join(e.Suggestions, ", ") + "?"
	}
	switch e.Code {
	case specdocs.CodeUnknownType, specdocs.CodeMissingType:
		if len(e.ValidTypes) > 0 {
			message += " Valid destination types: " + strings.Join(e.ValidTypes, ", ") + "."
		}
		message += " Run `rudder-cli docs destination` for their display names."
	case specdocs.CodeUnknownSource, specdocs.CodeUnsupportedSource:
		if len(e.ValidTypes) > 0 {
			message += " Valid source types: " + strings.Join(e.ValidTypes, ", ") + "."
		}
	case specdocs.CodeUnknownDefinitionVersion:
		versions := make([]string, 0, len(e.ValidVersions))
		for _, v := range e.ValidVersions {
			versions = append(versions, fmt.Sprint(v))
		}
		message += " Registered versions: " + strings.Join(versions, ", ") + "."
	}
	return message
}
