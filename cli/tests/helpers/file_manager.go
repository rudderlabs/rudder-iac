package helpers

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	resourceKeyToFileNameRegex = regexp.MustCompile(`[:\s\\/\*\?"<>\|]+`)
)

// StateFileManager manages reading state files from a base directory
type SnapshotFileManager struct {
	baseDir string
}

// NewStateFileManager creates a new StateFileManager instance
// It validates that the baseDir exists and is a directory
func NewSnapshotFileManager(baseDir string) (*SnapshotFileManager, error) {
	info, err := os.Stat(baseDir)
	if err != nil {
		return nil, fmt.Errorf("error accessing directory %s: %w", baseDir, err)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("path is not a directory: %s", baseDir)
	}

	return &SnapshotFileManager{
		baseDir: baseDir,
	}, nil
}

// resourceKeyToFileName converts a resource key to a filesystem-safe filename
// It replaces colons and other invalid characters with underscores, converts to lowercase,
// and removes leading/trailing underscores.
func (s *SnapshotFileManager) resourceURNToFileName(urn string) string {
	filename := strings.ToLower(urn)
	filename = resourceKeyToFileNameRegex.ReplaceAllString(filename, "_")
	filename = strings.Trim(filename, "_")

	return filename
}

// LoadExpectedState loads and parses a JSON state file for the given resource
func (sfm *SnapshotFileManager) LoadExpectedState(resourceURN string) (map[string]any, error) {
	fullPath := filepath.Join(
		sfm.baseDir,
		sfm.resourceURNToFileName(resourceURN))

	if _, err := os.Stat(fullPath); err != nil {
		return nil, fmt.Errorf("accessing state file for resource '%s': %w", resourceURN, err)
	}

	content, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("reading state file for resource '%s': %w", resourceURN, err)
	}

	var state map[string]any
	if err := json.Unmarshal(content, &state); err != nil {
		return nil, fmt.Errorf("parsing JSON for resource '%s': %w", resourceURN, err)
	}

	return state, nil
}

// LoadExpectedVersion reads the version file and returns the version string
// Returns "0.0.0" with ok=false when file doesn't exist to provide a sensible default
func (sfm *SnapshotFileManager) LoadExpectedVersion() (string, bool) {
	versionPath := filepath.Join(sfm.baseDir, "version")

	content, err := os.ReadFile(versionPath)
	if err != nil {
		return "0.0.0", false
	}

	version := strings.TrimSpace(string(content))
	return version, true
}

// ListResources returns a list of all resource names in the base directory
// Excludes version file and non-JSON files to focus only on actual resource state files
func (sfm *SnapshotFileManager) ListResources() ([]string, error) {
	entries, err := os.ReadDir(sfm.baseDir)
	if err != nil {
		return nil, fmt.Errorf("reading directory %s: %w", sfm.baseDir, err)
	}

	var resources []string
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == "version" {
			continue
		}
		resources = append(resources, entry.Name())
	}

	return resources, nil
}
