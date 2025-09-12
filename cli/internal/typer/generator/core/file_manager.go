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
}

// NewFileManager creates a new FileManager
func NewFileManager(baseDir string) *FileManager {
	return &FileManager{
		BaseDir: baseDir,
	}
}

// WriteFile writes a single file to the filesystem atomically
// It creates parent directories as needed and handles file writing safely
func (fm *FileManager) WriteFile(file File) error {
	if err := fm.validateFile(file); err != nil {
		return err
	}

	baseDir := fm.BaseDir
	if baseDir == "" {
		baseDir = "." // Use current working directory as default
	}
	fullPath := filepath.Join(baseDir, file.Path)

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
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("creating directory %s: %w", parentDir, err)
	}
	return nil
}

// writeFileAtomically writes file content atomically using a temporary file
func (fm *FileManager) writeFileAtomically(path, content string) error {
	name := filepath.Base(path)

	// Create temporary file in system temp directory to avoid leaving temp files in output dir
	tmpFile, err := os.CreateTemp("", name+".tmp.*")
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

	// Set proper file permissions (0644 - owner read/write, group/others read)
	if err := os.Chmod(path, 0644); err != nil {
		return fmt.Errorf("setting file permissions: %w", err)
	}

	return nil
}
