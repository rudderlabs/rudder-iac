package schema

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	schemapkg "github.com/rudderlabs/rudder-iac/cli/internal/schema"
	"github.com/spf13/cobra"
)

func NewCmdSchema() *cobra.Command {
	var outDir string

	cmd := &cobra.Command{
		Use:   "schema [kind]",
		Short: "Generate JSON Schema for spec kinds",
		Long: heredoc.Doc(`
			Generate Draft 2020-12 JSON Schema for supported RudderStack spec kinds.

			With no arguments, this command lists available kinds. Pass a kind to
			print its schema, or use --out to write all kind schemas and the combined
			root schema to a directory.
		`),
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			catalog, err := app.GenerateSchemas()
			if err != nil {
				return fmt.Errorf("generating schemas: %w", err)
			}
			if outDir != "" {
				if len(args) != 0 {
					return fmt.Errorf("kind cannot be used with --out")
				}
				return writeAll(cmd, catalog, outDir)
			}
			if len(args) == 0 {
				for _, kind := range catalog.Kinds() {
					fmt.Fprintln(cmd.OutOrStdout(), kind)
				}
				return nil
			}

			data, err := catalog.MarshalKind(args[0])
			if err != nil {
				return fmt.Errorf("generating schema for %q: %w", args[0], err)
			}
			_, err = cmd.OutOrStdout().Write(data)
			return err
		},
	}
	cmd.Flags().StringVar(&outDir, "out", "", "write all schemas to a directory")
	return cmd
}

func writeAll(cmd *cobra.Command, catalog *schemapkg.Catalog, dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	written := make([]string, 0, len(catalog.Kinds())+1)
	for _, kind := range catalog.Kinds() {
		data, err := catalog.MarshalKind(kind)
		if err != nil {
			return fmt.Errorf("generating schema for %q: %w", kind, err)
		}
		path := filepath.Join(dir, schemapkg.FileName(kind))
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", path, err)
		}
		written = append(written, path)
	}

	data, err := catalog.MarshalRoot()
	if err != nil {
		return fmt.Errorf("generating root schema: %w", err)
	}
	path := filepath.Join(dir, schemapkg.RootFileName)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	written = append(written, path)

	fmt.Fprintf(cmd.OutOrStdout(), "Wrote %d schema files to %s:\n  %s\n", len(written), dir, strings.Join(written, "\n  "))
	return nil
}
