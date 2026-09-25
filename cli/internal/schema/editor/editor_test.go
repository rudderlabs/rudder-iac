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

func TestURL(t *testing.T) {
	assert.Equal(t,
		"https://www.rudderstack.com/docs/schemas/rudder-cli/v1/source.schema.json",
		URL("source"),
	)
}

func TestURLBase(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		envURL  string
		want    string
	}{
		{
			name: "default docs URL",
			want: DefaultSchemaURLBase,
		},
		{
			name:   "environment override",
			envURL: " https://schemas.example.test/env/// ",
			want:   "https://schemas.example.test/env",
		},
		{
			name:    "explicit override wins",
			baseURL: "  https://schemas.example.test/releases///  ",
			envURL:  "https://schemas.example.test/env",
			want:    "https://schemas.example.test/releases",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(SchemaBaseURLEnv, tt.envURL)
			assert.Equal(t, tt.want, URLBase(tt.baseURL))
		})
	}
}

func TestSchemaURLTrimsBase(t *testing.T) {
	assert.Equal(t,
		"https://schemas.example.test/releases/source.schema.json",
		SchemaURL("  https://schemas.example.test/releases///  ", "source"),
	)
}
