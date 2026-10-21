// Package docs holds the `docs` command group, which prints reference
// documentation read from the definitions compiled into this binary.
package docs

import (
	"errors"
	"fmt"
	"io"
	"strings"

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
// also offers, and returns a func that resolves the chosen format.
func formatFlags(cmd *cobra.Command) func() (string, error) {
	var (
		format string
		asJSON bool
	)
	cmd.Flags().StringVar(&format, "format", formatMarkdown, "Output format: md or json")
	cmd.Flags().BoolVarP(&asJSON, "json", "j", false, "Output JSON, the same as --format json")
	cmd.MarkFlagsMutuallyExclusive("format", "json")

	return func() (string, error) {
		if asJSON {
			return formatJSON, nil
		}
		if format != formatMarkdown && format != formatJSON {
			return format, fmt.Errorf("--format must be md or json, got %q", format)
		}
		return format, nil
	}
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
	case specdocs.CodeUnknownType:
		message += " Run `rudder-cli docs destination` to list the types."
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
