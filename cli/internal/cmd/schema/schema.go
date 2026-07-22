package schema

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/schema"
	"github.com/spf13/cobra"
)

// NewCmdSchema builds the `rudder-cli schema` command. It emits Draft 2020-12
// JSON Schema for spec kinds, generated from the typed spec structs so the
// schema never drifts from what the CLI actually parses.
func NewCmdSchema() *cobra.Command {
	var outDir string

	cmd := &cobra.Command{
		Use:   "schema [kind]",
		Short: "Print or write JSON Schema for spec kinds",
		Long: heredoc.Doc(`
			Generate JSON Schema (Draft 2020-12) for RudderStack CLI spec kinds.

			The schema is generated from the typed spec structs the CLI parses,
			so it always matches the current spec format. Point your editor at a
			written schema file with a header comment to get inline completion
			and validation:

			    # yaml-language-server: $schema=./schemas/tracking-plan.schema.json

			With no kind, the available kinds are listed. Pass a kind to print
			its schema to stdout, or --out to write every kind (plus a combined
			root schema) to a directory.
		`),
		Example: heredoc.Doc(`
			# List the kinds that have a schema
			$ rudder-cli schema

			# Print one kind's schema
			$ rudder-cli schema tracking-plan

			# Write all schemas to a directory
			$ rudder-cli schema --out ./schemas
		`),
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()

			if outDir != "" {
				return writeAll(cmd, outDir)
			}

			if len(args) == 0 {
				fmt.Fprintln(out, "Available spec kinds:")
				for _, kind := range schema.Kinds() {
					fmt.Fprintf(out, "  %s\n", kind)
				}
				fmt.Fprintln(out, "\nRun 'rudder-cli schema <kind>' to print a schema, or --out <dir> to write all.")
				return nil
			}

			b, err := schema.MarshalKind(args[0])
			if err != nil {
				return fmt.Errorf("generating schema for %q: %w", args[0], err)
			}
			fmt.Fprintln(out, string(b))
			return nil
		},
	}

	cmd.Flags().StringVar(&outDir, "out", "", "write all schemas (one file per kind, plus a combined root schema) to this directory")

	return cmd
}

func writeAll(cmd *cobra.Command, dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	written := make([]string, 0, len(schema.Kinds())+1)
	for _, kind := range schema.Kinds() {
		b, err := schema.MarshalKind(kind)
		if err != nil {
			return fmt.Errorf("generating schema for %q: %w", kind, err)
		}
		path := filepath.Join(dir, schema.FileName(kind))
		if err := os.WriteFile(path, append(b, '\n'), 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", path, err)
		}
		written = append(written, path)
	}

	root, err := schema.MarshalRoot()
	if err != nil {
		return fmt.Errorf("generating root schema: %w", err)
	}
	rootPath := filepath.Join(dir, schema.RootFileName)
	if err := os.WriteFile(rootPath, append(root, '\n'), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", rootPath, err)
	}
	written = append(written, rootPath)

	fmt.Fprintf(cmd.OutOrStdout(), "Wrote %d schema files to %s:\n%s\n",
		len(written), dir, "  "+strings.Join(written, "\n  "))
	return nil
}
