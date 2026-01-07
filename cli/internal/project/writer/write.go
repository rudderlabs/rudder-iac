package writer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/formatter"
)

// FormattableEntity represents an importable entity with content and path.
// The path extension is used to determine the formatter to use (e.g .yaml for YAML files).
type FormattableEntity struct {
	Content      any
	RelativePath string
}

// Write is a helper function to write the files based on the formattable entities
// using a list of available formatters.
func Write(_ context.Context, baseDir string, formatters formatter.Formatters, data []FormattableEntity) error {

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

		err = writeFile(path, content)
		if err != nil {
			return fmt.Errorf("writing %s: %w", path, err)
		}
	}

	return nil
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
func OverwriteFile(formatters formatter.Formatters, entity FormattableEntity) error {
	formatted, err := formatters.Format(entity.Content, filepath.Ext(entity.RelativePath))
	if err != nil {
		return fmt.Errorf("formatting %s: %w", entity.RelativePath, err)
	}

	if err := os.WriteFile(entity.RelativePath, formatted, 0644); err != nil {
		return fmt.Errorf("writing file %s: %w", entity.RelativePath, err)
	}

	return nil
}
