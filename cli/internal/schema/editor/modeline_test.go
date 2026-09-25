package editor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnsureModeline(t *testing.T) {
	content := []byte("version: rudder/v1\n")
	assert.Equal(t, "# yaml-language-server: $schema=https://example.com/source.schema.json\nversion: rudder/v1\n", string(EnsureModeline(content, "https://example.com/source.schema.json")))
}

func TestEnsureModelineDisabled(t *testing.T) {
	content := []byte("version: rudder/v1\n")
	assert.Equal(t, content, EnsureModeline(content, ""))
}

func TestEnsureModelinePreservesExisting(t *testing.T) {
	content := []byte("# yaml-language-server: $schema=custom.json\nversion: rudder/v1\n")
	assert.Equal(t, content, EnsureModeline(content, "https://example.com/source.schema.json"))
}
