package writer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/formatter"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/schema/editor"
)

// FormattableEntity represents an importable entity with content and path.
// The path extension is used to determine the formatter to use (e.g .yaml for YAML files).
type FormattableEntity struct {
	Content      any
	RelativePath string
}

// Option configures file rendering behavior.
type Option func(*options)

type options struct {
	schemaModeline        bool
	schemaModelineBaseURL string
}

// WithSchemaModeline prepends a yaml-language-server modeline to YAML entities
// whose content is a *specs.Spec. baseURL is the versioned schema directory used
// to build the per-kind schema URL.
func WithSchemaModeline(baseURL string) Option {
	return func(opts *options) {
		opts.schemaModeline = true
		opts.schemaModelineBaseURL = baseURL
	}
}

// Write is a helper function to write the files based on the formattable entities
// using a list of available formatters.
func Write(_ context.Context, baseDir string, formatters formatter.Formatters, data []FormattableEntity, writerOpts ...Option) error {
	var opts options
	for _, apply := range writerOpts {
		apply(&opts)
	}

	for _, datum := range data {
		path := filepath.Join(baseDir, datum.RelativePath)

		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return fmt.Errorf("creating directory %s: %w", filepath.Dir(path), err)
		}

		content, err := formatters.Format(
			datum.Content,
			filepath.Ext(path))
		if err != nil {
			return fmt.Errorf("formatting %s: %w", path, err)
		}
		content = addSchemaModeline(content, filepath.Ext(path), datum.Content, opts)

		err = writeFile(path, content)
		if err != nil {
			return fmt.Errorf("writing %s: %w", path, err)
		}
	}

	return nil
}

func addSchemaModeline(content []byte, ext string, value any, opts options) []byte {
	if !opts.schemaModeline || (ext != ".yaml" && ext != ".yml") {
		return content
	}
	spec, ok := value.(*specs.Spec)
	if !ok || spec == nil {
		return content
	}
	return editor.EnsureHeader(content, editor.SchemaURL(opts.schemaModelineBaseURL, spec.Kind))
}

// writeFile writes content to a file, but fails if the file already exists.
// This prevents accidental overwriting of existing files.
func writeFile(path string, content []byte) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("opening file %s: %w", path, err)
	}
	defer file.Close()

	_, err = file.Write(content)
	if err != nil {
		return fmt.Errorf("writing to file %s: %w", path, err)
	}
	return nil
}

// OverwriteFile writes a FormattableEntity to a file, overwriting it if it already exists.
// This is used for operations like migration where we intentionally want to replace existing files.
func OverwriteFile(formatters formatter.Formatters, entity FormattableEntity, writerOpts ...Option) error {
	var opts options
	for _, apply := range writerOpts {
		apply(&opts)
	}

	ext := filepath.Ext(entity.RelativePath)
	formatted, err := formatters.Format(entity.Content, ext)
	if err != nil {
		return fmt.Errorf("formatting %s: %w", entity.RelativePath, err)
	}
	formatted = addSchemaModeline(formatted, ext, entity.Content, opts)

	if err := os.WriteFile(entity.RelativePath, formatted, 0644); err != nil {
		return fmt.Errorf("writing file %s: %w", entity.RelativePath, err)
	}

	return nil
}
