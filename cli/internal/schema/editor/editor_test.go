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

func TestFileName(t *testing.T) {
	assert.Equal(t, "properties.schema.json", FileName("properties"))
}

func TestReleaseURL(t *testing.T) {
	assert.Equal(t,
		"https://github.com/rudderlabs/rudder-iac/releases/download/v1.2.3/source.schema.json",
		ReleaseURL("v1.2.3", "source"),
	)
}

func TestVersionedReleaseURLBase(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		version string
		want    string
	}{
		{
			name:    "default release URL normalizes GoReleaser version to tag",
			version: "1.2.3",
			want:    "https://github.com/rudderlabs/rudder-iac/releases/download/v1.2.3",
		},
		{
			name:    "default release URL keeps prefixed tag",
			version: "v1.2.3",
			want:    "https://github.com/rudderlabs/rudder-iac/releases/download/v1.2.3",
		},
		{
			name:    "custom URL preserves version convention",
			baseURL: "  https://schemas.example.test/releases///  ",
			version: "1.2.3",
			want:    "https://schemas.example.test/releases/1.2.3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, VersionedReleaseURLBase(tt.baseURL, tt.version))
		})
	}
}

func TestSchemaURLTrimsBase(t *testing.T) {
	assert.Equal(t,
		"https://schemas.example.test/releases/source.schema.json",
		SchemaURL("  https://schemas.example.test/releases///  ", "source"),
	)
}
