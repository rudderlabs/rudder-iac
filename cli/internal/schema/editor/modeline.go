package editor

import (
	"bytes"
	"strings"
)

const prefix = "# yaml-language-server: $schema="

func FileName(kind string) string {
	return kind + ".schema.json"
}

func EnsureModeline(content []byte, schemaURL string) []byte {
	if schemaURL == "" || hasModeline(content) {
		return content
	}
	return append(append([]byte(prefix+schemaURL), '\n'), content...)
}

func hasModeline(content []byte) bool {
	for _, line := range bytes.SplitN(content, []byte("\n"), 3) {
		trimmed := strings.TrimSpace(string(line))
		if trimmed == "" {
			continue
		}
		return strings.HasPrefix(trimmed, prefix)
	}
	return false
}
