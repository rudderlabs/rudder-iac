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

func TestNewEnvResolver(t *testing.T) {
	t.Setenv("RUDDER_DB_HOST", "db.example.com")
	t.Setenv("RUDDER_DB_PORT", "5432")
	t.Setenv("RUDDER_EMPTY", "")
	t.Setenv("DB_HOST", "ignored")

	r := NewEnvResolver()

	t.Run("strips prefix and loads value", func(t *testing.T) {
		value, found := r.Resolve("DB_HOST")
		assert.Equal(t, "db.example.com", value)
		assert.True(t, found)
	})

	t.Run("loads multiple prefixed vars", func(t *testing.T) {
		value, found := r.Resolve("DB_PORT")
		assert.Equal(t, "5432", value)
		assert.True(t, found)
	})

	t.Run("ignores unprefixed vars", func(t *testing.T) {
		_, found := r.Resolve("ignored")
		assert.False(t, found)
	})

	t.Run("missing key returns not found", func(t *testing.T) {
		_, found := r.Resolve("MISSING")
		assert.False(t, found)
	})

	t.Run("empty value is preserved", func(t *testing.T) {
		value, found := r.Resolve("EMPTY")
		assert.Equal(t, "", value)
		assert.True(t, found)
	})
}
