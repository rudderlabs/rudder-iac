package resolver

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeVarFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "vars.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
	return path
}

func TestNewFileResolver(t *testing.T) {
	tests := []struct {
		name    string
		content string
		noFile  bool
		wantErr error
	}{
		{
			name:    "missing file returns ErrNotFound",
			noFile:  true,
			wantErr: ErrNotFound,
		},
		{
			name:    "invalid YAML returns ErrIllegalArgument",
			content: "{{not valid yaml",
			wantErr: ErrIllegalArgument,
		},
		{
			name:    "nested map rejected",
			content: "DB:\n  HOST: localhost\n  PORT: 5432",
			wantErr: ErrIllegalArgument,
		},
		{
			name:    "nested array rejected",
			content: "HOSTS:\n  - a\n  - b",
			wantErr: ErrIllegalArgument,
		},
		{
			name:    "nil value (bare key) rejected",
			content: "EMPTY_KEY:",
			wantErr: ErrIllegalArgument,
		},
		{
			name:    "explicit null value rejected",
			content: "EMPTY_KEY: null",
			wantErr: ErrIllegalArgument,
		},
		{
			name:    "valid flat file succeeds",
			content: "DB_HOST: localhost\nDB_PORT: 5432",
		},
		{
			name:    "empty file succeeds",
			content: "",
		},
		{
			name:    "explicit empty string succeeds",
			content: `EMPTY: ""`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var path string
			if tt.noFile {
				path = filepath.Join(t.TempDir(), "nonexistent.yaml")
			} else {
				path = writeVarFile(t, tt.content)
			}

			_, err := NewFileResolver(path)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFileResolver_Resolve(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		lookup    string
		wantValue string
		wantFound bool
	}{
		{
			name:      "string value found",
			content:   "DB_HOST: db.staging.example.com",
			lookup:    "DB_HOST",
			wantValue: "db.staging.example.com",
			wantFound: true,
		},
		{
			name:      "integer converted to string",
			content:   "DB_PORT: 5432",
			lookup:    "DB_PORT",
			wantValue: "5432",
			wantFound: true,
		},
		{
			name:      "boolean converted to string",
			content:   "ENABLED: true",
			lookup:    "ENABLED",
			wantValue: "true",
			wantFound: true,
		},
		{
			name:      "float converted to string",
			content:   "RATE: 3.14",
			lookup:    "RATE",
			wantValue: "3.14",
			wantFound: true,
		},
		{
			name:      "explicit empty string value preserved",
			content:   `EMPTY_KEY: ""`,
			lookup:    "EMPTY_KEY",
			wantValue: "",
			wantFound: true,
		},
		{
			name:      "variable not in file returns not found",
			content:   "DB_HOST: localhost",
			lookup:    "MISSING",
			wantValue: "",
			wantFound: false,
		},
		{
			name:      "empty file resolves nothing",
			content:   "",
			lookup:    "ANY",
			wantValue: "",
			wantFound: false,
		},
		{
			name:      "comments and empty lines handled",
			content:   "# database config\n\nDB_HOST: localhost\n# port below\nDB_PORT: 5432",
			lookup:    "DB_HOST",
			wantValue: "localhost",
			wantFound: true,
		},
		{
			name:      "multiple variables in same file",
			content:   "DB_HOST: localhost\nDB_PORT: 5432\nDB_NAME: analytics",
			lookup:    "DB_NAME",
			wantValue: "analytics",
			wantFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeVarFile(t, tt.content)
			r, err := NewFileResolver(path)
			require.NoError(t, err)

			value, found := r.Resolve(tt.lookup)
			assert.Equal(t, tt.wantValue, value)
			assert.Equal(t, tt.wantFound, found)
		})
	}
}
