package importer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rudderlabs/rudder-iac/cli/internal/importremote"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/formatter"
)

// Write is a helper function to write the files based on the formattable entities
// using a list of available formatters.
func Write(ctx context.Context, baseDir string, formatters formatter.Formatters, data []importremote.FormattableEntity) error {

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

		err = os.WriteFile(path, content, 0644)
		if err != nil {
			return fmt.Errorf("writing %s: %w", path, err)
		}
	}

	return nil
}
