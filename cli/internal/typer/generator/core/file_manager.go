package core

import (
	"fmt"
	"os"
	"path/filepath"
)

// WriteFile writes a single file to the filesystem
// It creates parent directories as needed and handles file writing atomically
func WriteFile(outputDir string, file File) error {
	if outputDir == "" {
		return fmt.Errorf("output directory cannot be empty")
	}
	if file.Path == "" {
		return fmt.Errorf("file path cannot be empty")
	}
	// Construct the full file path
	fullPath := filepath.Join(outputDir, file.Path)

	// Create parent directories if they don't exist
	parentDir := filepath.Dir(fullPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("creating parent directory %s: %w", parentDir, err)
	}
	// Write file atomically using a temporary file
	if err := writeFileAtomically(fullPath, file.Content); err != nil {
		return fmt.Errorf("writing file content: %w", err)
	}
	return nil
}

// writeFileAtomically writes file content atomically using a temporary file
// This ensures that if the write fails, the original file is not corrupted
func writeFileAtomically(path, content string) error {
	// Create a temporary file in the same directory
	dir := filepath.Dir(path)
	name := filepath.Base(path)

	tmpFile, err := os.CreateTemp(dir, name+".tmp")
	if err != nil {
		return fmt.Errorf("creating temporary file: %w", err)
	}
	defer func() {
		tmpFile.Close()
		// Clean up temp file if it still exists
		os.Remove(tmpFile.Name())
	}()

	// Write content to temporary file
	if _, err := tmpFile.WriteString(content); err != nil {
		return fmt.Errorf("writing to temporary file: %w", err)
	}
	// Close the temporary file before renaming
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("closing temporary file: %w", err)
	}
	// Atomically rename the temporary file to the target file
	if err := os.Rename(tmpFile.Name(), path); err != nil {
		return fmt.Errorf("renaming temporary file to target: %w", err)
	}
	return nil
}
