package schema

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	schemapkg "github.com/rudderlabs/rudder-iac/cli/internal/schema"
)

type generateFunc func() (schemapkg.Set, error)

// NewCmdSchema creates the schema generation command.
func NewCmdSchema() *cobra.Command {
	return newCmdSchema(app.GenerateSchemas)
}

func newCmdSchema(generate generateFunc) *cobra.Command {
	var outDir string
	var overwrite bool

	cmd := &cobra.Command{
		Use:   "schema [kind]",
		Short: "Generate JSON Schemas for RudderStack specs",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			schemas, err := generate()
			if err != nil {
				return fmt.Errorf("generating schemas: %w", err)
			}

			if outDir != "" {
				if len(args) != 0 {
					return errors.New("a schema kind cannot be combined with --out; omit the kind to write all schemas")
				}
				return writeSchemas(outDir, schemas, overwrite)
			}
			if overwrite {
				return errors.New("--overwrite requires --out")
			}

			if len(args) == 0 {
				for _, kind := range schemapkg.Kinds(schemas) {
					if _, err := fmt.Fprintln(cmd.OutOrStdout(), kind); err != nil {
						return fmt.Errorf("writing schema kinds: %w", err)
					}
				}
				return nil
			}

			data, err := schemapkg.MarshalKind(schemas, args[0])
			if err != nil {
				return err
			}
			if _, err := fmt.Fprintln(cmd.OutOrStdout(), string(data)); err != nil {
				return fmt.Errorf("writing schema: %w", err)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&outDir, "out", "", "write all schemas to this directory")
	cmd.Flags().BoolVar(&overwrite, "overwrite", false, "overwrite existing schema files")
	return cmd
}

type artifact struct {
	path string
	data []byte
}

func writeSchemas(outDir string, schemas schemapkg.Set, overwrite bool) error {
	artifacts := make([]artifact, 0, len(schemas)+1)
	for _, kind := range schemapkg.Kinds(schemas) {
		data, err := schemapkg.MarshalKind(schemas, kind)
		if err != nil {
			return fmt.Errorf("marshaling schema for %q: %w", kind, err)
		}
		artifacts = append(artifacts, artifact{
			path: filepath.Join(outDir, schemapkg.FileName(kind)),
			data: data,
		})
	}
	root, err := schemapkg.MarshalRoot(schemas)
	if err != nil {
		return fmt.Errorf("marshaling root schema: %w", err)
	}
	artifacts = append(artifacts, artifact{
		path: filepath.Join(outDir, schemapkg.RootFileName),
		data: root,
	})

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("creating schema output directory: %w", err)
	}
	if !overwrite {
		for _, artifact := range artifacts {
			if _, err := os.Stat(artifact.path); err == nil {
				return fmt.Errorf("schema artifact already exists: %s (use --overwrite to replace it)", artifact.path)
			} else if !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("checking schema artifact %s: %w", artifact.path, err)
			}
		}
	}

	for _, artifact := range artifacts {
		if err := os.WriteFile(artifact.path, append(artifact.data, '\n'), 0o644); err != nil {
			return fmt.Errorf("writing schema artifact %s: %w", artifact.path, err)
		}
	}
	return nil
}
