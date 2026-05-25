package resolver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewEnvResolverFromEnviron(t *testing.T) {
	tests := []struct {
		name      string
		environ   []string
		lookup    string
		wantValue string
		wantFound bool
	}{
		{
			name: "strips prefix from matching env vars",
			environ: []string{
				"RUDDER_DB_HOST=db.example.com",
				"RUDDER_DB_PORT=5432",
			},
			lookup:    "DB_PORT",
			wantValue: "5432",
			wantFound: true,
		},
		{
			name: "ignores env vars without matching prefix",
			environ: []string{
				"RUDDER_DB_HOST=db.example.com",
				"HOME=/home/user",
				"PATH=/usr/bin",
			},
			lookup:    "HOME",
			wantValue: "",
			wantFound: false,
		},
		{
			name: "case sensitive prefix matching",
			environ: []string{
				"RUDDER_DB_HOST=upper",
				"rudder_db_host=lower",
			},
			lookup:    "db_host",
			wantValue: "",
			wantFound: false,
		},
		{
			name: "empty value is preserved",
			environ: []string{
				"RUDDER_HOST=",
			},
			lookup:    "HOST",
			wantValue: "",
			wantFound: true,
		},
		{
			name:      "no matching env vars yields empty map",
			environ:   []string{"OTHER_VAR=value"},
			lookup:    "OTHER_VAR",
			wantValue: "",
			wantFound: false,
		},
		{
			name: "value containing equals sign",
			environ: []string{
				"RUDDER_CONN=host=db;port=5432",
			},
			lookup:    "CONN",
			wantValue: "host=db;port=5432",
			wantFound: true,
		},
		{
			name:      "entry without equals sign is skipped",
			environ:   []string{"RUDDER_BROKEN"},
			lookup:    "BROKEN",
			wantValue: "",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newEnvResolverFromEnviron(tt.environ)
			value, found := r.Resolve(tt.lookup)
			assert.Equal(t, tt.wantValue, value)
			assert.Equal(t, tt.wantFound, found)
		})
	}
}

func TestNewEnvResolver_LoadsRudderPrefixedVariables(t *testing.T) {
	t.Setenv("RUDDER_ABC", "hello")
	t.Setenv("ABC", "ignored")

	r := NewEnvResolver()
	value, found := r.Resolve("ABC")

	assert.Equal(t, "hello", value)
	assert.True(t, found)
}
