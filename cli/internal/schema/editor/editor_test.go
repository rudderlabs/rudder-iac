package editor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHeader(t *testing.T) {
	assert.Equal(t,
		"# yaml-language-server: $schema=./schemas/tracking-plan.schema.json",
		Header("./schemas/tracking-plan.schema.json"),
	)
}

func TestEnsureHeaderPrepends(t *testing.T) {
	content := []byte("version: rudder/v1\nkind: transformation\n")
	out := EnsureHeader(content, ".rudder/schemas/transformation.schema.json")

	assert.Equal(t,
		"# yaml-language-server: $schema=.rudder/schemas/transformation.schema.json\nversion: rudder/v1\nkind: transformation\n",
		string(out),
	)
}

func TestEnsureHeaderIdempotent(t *testing.T) {
	content := []byte("# yaml-language-server: $schema=old.json\nversion: rudder/v1\n")
	out := EnsureHeader(content, "new.json")
	assert.Equal(t, string(content), string(out))
}

func TestDefaultSchemaRef(t *testing.T) {
	assert.Equal(t, ".rudder/schemas/events.schema.json", DefaultSchemaRef("events"))
}

func TestFileName(t *testing.T) {
	assert.Equal(t, "properties.schema.json", FileName("properties"))
}
