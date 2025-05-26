package loader

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/pkg/logger"
)

var log = logger.New("loader")

// Loader is responsible for finding and loading project specification files.
type Loader struct {
}

// Load scans the configured location for YAML files (.yaml or .yml).
// It walks the directory tree recursively to discover them, and parses them into Spec objects.
// It returns a map of file paths to their corresponding Spec objects,
// or an error if any file operation or spec parsing fails.
func (l *Loader) Load(location string) (map[string]*specs.Spec, error) {
	var allSpecs map[string]*specs.Spec = make(map[string]*specs.Spec)

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
		if ext != ".yaml" && ext != ".yml" {
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

		// Parse spec
		spec, err := specs.New(data)
		if err != nil {
			return fmt.Errorf("parsing spec file %s: %w", path, err)
		}

		allSpecs[path] = spec
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("loading specs: %w", err)
	}

	return allSpecs, nil
}
