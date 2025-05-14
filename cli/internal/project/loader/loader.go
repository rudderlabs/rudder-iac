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

type Loader struct {
	// Location is the root directory of the project
	Location string
}

func New(location string) *Loader {
	if location == "" {
		location = "."
	}

	return &Loader{
		Location: location,
	}
}

func (l *Loader) Load() (map[string]*specs.Spec, error) {
	var allSpecs map[string]*specs.Spec = make(map[string]*specs.Spec)

	log.Info("loading specs", "location", l.Location)

	err := filepath.WalkDir(l.Location, func(path string, d os.DirEntry, err error) error {
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
