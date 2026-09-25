// Package editor provides helpers for the yaml-language-server "$schema"
// modeline that associates a JSON Schema with a YAML spec file. It is a leaf
// package with no provider dependencies so it can be used from the writer
// without creating an import cycle with the schema generator.
package editor

import (
	"bytes"
	"strings"
)

// headerPrefix is the modeline yaml-language-server reads to associate a schema
// with a YAML file, giving editors inline completion and validation.
const (
	headerPrefix          = "# yaml-language-server: $schema="
	DefaultReleaseURLBase = "https://github.com/rudderlabs/rudder-iac/releases/download"
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

// VersionedReleaseURLBase returns the release-download base for version. An
// empty baseURL uses DefaultReleaseURLBase; overrides are trimmed before the
// version is appended.
func VersionedReleaseURLBase(baseURL, version string) string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	version = strings.Trim(strings.TrimSpace(version), "/")
	if baseURL == "" {
		baseURL = DefaultReleaseURLBase
		if version != "" && !strings.HasPrefix(version, "v") {
			version = "v" + version
		}
	}
	return baseURL + "/" + version
}

// SchemaURL appends a kind's schema filename to baseURL. Surrounding whitespace
// and trailing slashes are removed so custom bases compose into a stable URL.
func SchemaURL(baseURL, kind string) string {
	return strings.TrimRight(strings.TrimSpace(baseURL), "/") + "/" + FileName(kind)
}

// ReleaseURL returns the canonical versioned release URL for a kind's schema.
func ReleaseURL(version, kind string) string {
	return SchemaURL(VersionedReleaseURLBase("", version), kind)
}
