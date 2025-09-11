# File Manager Component

The File Manager component provides atomic file operations for code generation in the RudderStack CLI.

## Overview

The `FileManager` ensures that file writes are atomic and safe, preventing partial writes that could corrupt generated code files. It handles:

- **Atomic file writes** using temporary files and atomic rename operations
- **Directory creation** with proper permissions
- **Security validation** preventing path traversal attacks
- **Batch operations** for writing multiple files efficiently
- **Customizable permissions** for files and directories

## Usage

### Basic Usage

```go
package main

import (
    "github.com/rudderlabs/rudder-iac/cli/internal/typer/generator/core"
)

func main() {
    // Create a file manager
    fm := core.NewFileManager("/path/to/output")

    // Create a file
    file := core.File{
        Path:    "example.txt",
        Content: "Hello, World!",
    }

    // Write the file atomically
    err := fm.WriteFile(file)
    if err != nil {
        panic(err)
    }
}
```

### Batch Operations

```go
// Write multiple files at once
files := []core.File{
    {Path: "file1.txt", Content: "Content 1"},
    {Path: "dir/file2.txt", Content: "Content 2"},
    {Path: "dir/nested/file3.txt", Content: "Content 3"},
}

err := fm.WriteFiles(files)
if err != nil {
    panic(err)
}
```

### Custom Permissions

```go
// Create file manager with custom permissions
fm := &core.FileManager{
    BaseDir:  "/path/to/output",
    FileMode: 0600, // Owner read/write only
    DirMode:  0700, // Owner access only
}
```

### Legacy Support

```go
// Legacy function still available (deprecated)
err := core.WriteFile("/path/to/output", file)
```

## Security Features

- **Path traversal protection**: Rejects paths containing `..`
- **Absolute path protection**: Only allows relative paths
- **Input validation**: Validates all input parameters
- **Atomic operations**: Prevents partial writes

## Architecture

The component follows these principles:

1. **Atomic writes**: Uses temporary files and atomic rename operations
2. **Fail-safe**: If any operation fails, no partial state is left behind
3. **Validation first**: All inputs are validated before any file system operations
4. **Clean separation**: File management is separate from code generation logic

## Testing

The component includes comprehensive tests covering:

- Basic file writing operations
- Nested directory creation
- Atomic operation guarantees
- Security validation
- Error handling scenarios
- Large file handling
- Custom permission settings

Run tests with:

```bash
go test ./cli/internal/typer/generator/core/
```

## Integration

This component is designed to be used by code generators in the RudderStack CLI. It provides a clean, safe interface for writing generated files to disk while ensuring data integrity and security.
