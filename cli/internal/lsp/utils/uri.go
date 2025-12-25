package utils

import (
	"fmt"
	"net/url"
	"path/filepath"
	"runtime"
	"strings"
)

// URIToPath converts a file:// URI to a local file system path
// Handles platform-specific path conversions and URL decoding
func URIToPath(uri string) (string, error) {
	if uri == "" {
		return "", fmt.Errorf("empty URI")
	}

	// Parse the URI
	u, err := url.Parse(uri)
	if err != nil {
		return "", fmt.Errorf("invalid URI: %w", err)
	}

	// Check if it's a file:// URI
	if u.Scheme != "file" {
		return "", fmt.Errorf("URI scheme must be 'file', got '%s'", u.Scheme)
	}

	// Get the path component (already URL-decoded by url.Parse)
	path := u.Path

	// Platform-specific handling
	if runtime.GOOS == "windows" {
		// On Windows, file URIs look like: file:///C:/path/to/file
		// The path will be "/C:/path/to/file", we need to remove the leading "/"
		if len(path) > 2 && path[0] == '/' && path[2] == ':' {
			path = path[1:] // Remove leading slash
		}
		// Convert forward slashes to backslashes
		path = filepath.FromSlash(path)
	}

	return path, nil
}

// PathToURI converts a local file system path to a file:// URI
// Handles platform-specific path conversions and URL encoding
func PathToURI(path string) string {
	if path == "" {
		return ""
	}

	// Convert to absolute path if not already
	absPath, err := filepath.Abs(path)
	if err != nil {
		// If we can't get absolute path, use the path as-is
		absPath = path
	}

	// Convert to forward slashes
	absPath = filepath.ToSlash(absPath)

	// Platform-specific handling
	if runtime.GOOS == "windows" {
		// On Windows, ensure we have a leading slash before the drive letter
		// C:/path -> /C:/path
		if len(absPath) > 1 && absPath[1] == ':' {
			absPath = "/" + absPath
		}
	} else {
		// On Unix, ensure we have a leading slash
		if !strings.HasPrefix(absPath, "/") {
			absPath = "/" + absPath
		}
	}

	// URL encode the path (handles spaces and special characters)
	// We use url.PathEscape which preserves forward slashes
	encodedPath := url.PathEscape(absPath)
	// url.PathEscape escapes slashes, but we need to unescape them for file URIs
	encodedPath = strings.ReplaceAll(encodedPath, "%2F", "/")

	return "file://" + encodedPath
}

// IsFileURI checks if the given URI is a file:// URI
func IsFileURI(uri string) bool {
	return strings.HasPrefix(uri, "file://")
}
