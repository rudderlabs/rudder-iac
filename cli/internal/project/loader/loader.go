package loader

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
)

var log = logger.New("loader")

const (
	ExtensionYAML = ".yaml"
	ExtensionYML  = ".yml"
)

// Loader is responsible for finding and loading project specification files.
type Loader struct {
}

// Load scans the configured location for YAML files (.yaml or .yml).
// It walks the directory tree recursively to discover them, and parses them into Spec objects.
// It returns a map of file paths to their corresponding Spec objects,
// or an error if any file operation or spec parsing fails.
func (l *Loader) Load(location string) (map[string]*specs.RawSpec, error) {
	var allRawSpecs map[string]*specs.RawSpec = make(map[string]*specs.RawSpec)

	log.Info("loading specs", "location", location)

	err := filepath.WalkDir(location, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walking path %s: %w", path, err)
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Check file extension
		ext := filepath.Ext(path)
		if ext != ExtensionYAML && ext != ExtensionYML {
			return nil
		}

		// Read file
		f, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("opening file: %w", err)
		}
		defer f.Close()

		data, err := io.ReadAll(f)
		if err != nil {
			return fmt.Errorf("reading file: %w", err)
		}

		allRawSpecs[path] = &specs.RawSpec{Data: data}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("loading specs: %w", err)
	}

	return allRawSpecs, nil
}
