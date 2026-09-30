// Package editor provides helpers for the yaml-language-server "$schema"
// modeline that associates a JSON Schema with a YAML spec file. It is a leaf
// package with no provider dependencies so it can be used from the writer
// without creating an import cycle with the schema generator.
package editor

import (
	"bytes"
	"os"
	"strings"
)

// headerPrefix is the modeline yaml-language-server reads to associate a schema
// with a YAML file, giving editors inline completion and validation.
const (
	headerPrefix         = "# yaml-language-server: $schema="
	SchemaBaseURLEnv     = "RUDDERSTACK_CLI_SCHEMA_BASE_URL"
	DefaultSchemaURLBase = "https://www.rudderstack.com/docs/schemas/rudder-cli/v1"
)

// Header returns the yaml-language-server modeline pointing at schemaRef (a path
// or URL). Editors with the YAML extension use it to validate the file.
func Header(schemaRef string) string {
	return headerPrefix + schemaRef
}

// EnsureHeader prepends the yaml-language-server modeline to content if it is
// not already present. Existing headers (of any schema ref) are left untouched
// so re-formatting a file is idempotent.
func EnsureHeader(content []byte, schemaRef string) []byte {
	if HasHeader(content) {
		return content
	}
	header := append([]byte(Header(schemaRef)), '\n')
	return append(header, content...)
}

// HasHeader reports whether content already begins with a yaml-language-server
// modeline (allowing leading blank lines).
func HasHeader(content []byte) bool {
	for _, line := range bytes.SplitN(content, []byte("\n"), 3) {
		trimmed := strings.TrimSpace(string(line))
		if trimmed == "" {
			continue
		}
		return strings.HasPrefix(trimmed, headerPrefix)
	}
	return false
}

// FileName returns the conventional schema file name for a kind.
func FileName(kind string) string {
	return kind + ".schema.json"
}

// URLBase returns the canonical schema URL root unless an explicit option or
// environment override is present. The namespace is versioned by the spec
// contract, not the CLI binary version.
func URLBase(baseURL string) string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		baseURL = strings.TrimRight(strings.TrimSpace(os.Getenv(SchemaBaseURLEnv)), "/")
	}
	if baseURL == "" {
		baseURL = DefaultSchemaURLBase
	}
	return baseURL
}

// SchemaURL appends a kind's schema filename to baseURL. Surrounding whitespace
// and trailing slashes are removed so custom bases compose into a stable URL.
func SchemaURL(baseURL, kind string) string {
	return strings.TrimRight(strings.TrimSpace(baseURL), "/") + "/" + FileName(kind)
}

// URL returns the canonical public URL for a kind's schema.
func URL(kind string) string {
	return SchemaURL(DefaultSchemaURLBase, kind)
}
