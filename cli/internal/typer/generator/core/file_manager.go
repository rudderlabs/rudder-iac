package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileManager provides atomic file operations for code generation
type FileManager struct {
	// BaseDir is the base directory for all file operations
	BaseDir string
	// FileMode is the permission mode for created files (default: 0644)
	FileMode os.FileMode
	// DirMode is the permission mode for created directories (default: 0755)
	DirMode os.FileMode
}

// NewFileManager creates a new FileManager with default settings
func NewFileManager(baseDir string) *FileManager {
	return &FileManager{
		BaseDir:  baseDir,
		FileMode: 0644,
		DirMode:  0755,
	}
}

// WriteFile writes a single file to the filesystem atomically
// It creates parent directories as needed and handles file writing safely
func (fm *FileManager) WriteFile(file File) error {
	if err := fm.validateFile(file); err != nil {
		return err
	}

	fullPath := filepath.Join(fm.BaseDir, file.Path)

	// Create parent directories if they don't exist
	if err := fm.ensureParentDir(fullPath); err != nil {
		return fmt.Errorf("creating parent directory: %w", err)
	}

	// Write file atomically
	if err := fm.writeFileAtomically(fullPath, file.Content); err != nil {
		return fmt.Errorf("writing file %s: %w", file.Path, err)
	}

	return nil
}

// WriteFiles writes multiple files atomically as a batch operation
func (fm *FileManager) WriteFiles(files []File) error {
	if len(files) == 0 {
		return nil
	}

	// Validate all files first
	for i, file := range files {
		if err := fm.validateFile(file); err != nil {
			return fmt.Errorf("file %d: %w", i, err)
		}
	}

	// Write all files
	for _, file := range files {
		if err := fm.WriteFile(file); err != nil {
			return err
		}
	}

	return nil
}

// validateFile validates file input parameters
func (fm *FileManager) validateFile(file File) error {
	if fm.BaseDir == "" {
		return fmt.Errorf("base directory cannot be empty")
	}
	if file.Path == "" {
		return fmt.Errorf("file path cannot be empty")
	}
	if strings.Contains(file.Path, "..") {
		return fmt.Errorf("file path cannot contain '..' for security reasons")
	}
	if filepath.IsAbs(file.Path) {
		return fmt.Errorf("file path must be relative, got absolute path: %s", file.Path)
	}
	return nil
}

// ensureParentDir creates parent directories if they don't exist
func (fm *FileManager) ensureParentDir(fullPath string) error {
	parentDir := filepath.Dir(fullPath)
	if err := os.MkdirAll(parentDir, fm.DirMode); err != nil {
		return fmt.Errorf("creating directory %s: %w", parentDir, err)
	}
	return nil
}

// writeFileAtomically writes file content atomically using a temporary file
func (fm *FileManager) writeFileAtomically(path, content string) error {
	dir := filepath.Dir(path)
	name := filepath.Base(path)

	// Create temporary file in the same directory
	tmpFile, err := os.CreateTemp(dir, name+".tmp.*")
	if err != nil {
		return fmt.Errorf("creating temporary file: %w", err)
	}

	// Ensure cleanup of temporary file
	defer func() {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
	}()

	// Write content to temporary file
	if _, err := tmpFile.WriteString(content); err != nil {
		return fmt.Errorf("writing to temporary file: %w", err)
	}

	// Sync to ensure data is written to disk
	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("syncing temporary file: %w", err)
	}

	// Close the temporary file before renaming
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("closing temporary file: %w", err)
	}

	// Atomically rename the temporary file to the target file
	if err := os.Rename(tmpFile.Name(), path); err != nil {
		return fmt.Errorf("renaming temporary file to target: %w", err)
	}

	// Set proper file permissions
	if err := os.Chmod(path, fm.FileMode); err != nil {
		return fmt.Errorf("setting file permissions: %w", err)
	}

	return nil
}

// WriteFile is a convenience function that uses the default FileManager behavior
// Deprecated: Use FileManager.WriteFile for better control and testability
func WriteFile(outputDir string, file File) error {
	fm := NewFileManager(outputDir)
	return fm.WriteFile(file)
}
