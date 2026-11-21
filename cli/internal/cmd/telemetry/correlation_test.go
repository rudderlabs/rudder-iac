package telemetry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSessionID(t *testing.T) {
	start := time.Date(2026, 11, 21, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		gap      time.Duration
		wantSame bool
	}{
		{name: "the next command a second later", gap: time.Second, wantSame: true},
		{name: "a command at the idle limit", gap: sessionIdle, wantSame: true},
		{name: "a command after the idle limit", gap: sessionIdle + time.Second, wantSame: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "telemetry.json")

			first, _ := correlationIDs(path, ".", start)
			second, _ := correlationIDs(path, ".", start.Add(tt.gap))

			require.NotEmpty(t, first)
			assert.Equal(t, tt.wantSame, first == second)
		})
	}

	t.Run("every command extends the session", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "telemetry.json")

		first, _ := correlationIDs(path, ".", start)
		correlationIDs(path, ".", start.Add(20*time.Minute))
		third, _ := correlationIDs(path, ".", start.Add(40*time.Minute))

		assert.Equal(t, first, third)
	})

	t.Run("a damaged state file starts a new session", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "telemetry.json")

		first, _ := correlationIDs(path, ".", start)
		require.NoError(t, os.WriteFile(path, []byte("{"), 0o600))
		second, _ := correlationIDs(path, ".", start.Add(time.Second))

		assert.NotEmpty(t, second)
		assert.NotEqual(t, first, second)
	})
}

func TestProjectID(t *testing.T) {
	// repo is a git checkout with specs in a subdirectory; bare has no checkout
	// around it.
	repo, bare := t.TempDir(), t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(repo, ".git"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(repo, "specs", "prod"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(bare, "specs"), 0o755))
	spec := filepath.Join(repo, "specs", "prod", "sources.yaml")
	require.NoError(t, os.WriteFile(spec, nil, 0o600))

	tests := []struct {
		name     string
		cwd      string
		a, b     string
		wantSame bool
	}{
		{name: "no location is the working directory", cwd: repo, a: "", b: ".", wantSame: true},
		{name: "a relative and an absolute location", cwd: repo, a: "specs", b: filepath.Join(repo, "specs"), wantSame: true},
		{name: "a subdirectory of a checkout is the checkout", cwd: repo, a: "specs/prod", b: ".", wantSame: true},
		{name: "a spec file is the checkout holding it", cwd: repo, a: spec, b: ".", wantSame: true},
		{name: "a location that does not exist yet", cwd: repo, a: "specs/new", b: ".", wantSame: true},
		{name: "two checkouts", cwd: repo, a: ".", b: bare, wantSame: false},
		{name: "outside a checkout a subdirectory is its own project", cwd: bare, a: "specs", b: ".", wantSame: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Chdir(tt.cwd)
			path := filepath.Join(t.TempDir(), "telemetry.json")
			now := time.Now()

			_, a := correlationIDs(path, tt.a, now)
			_, b := correlationIDs(path, tt.b, now)

			assert.Len(t, a, 64)
			assert.Equal(t, tt.wantSame, a == b)
		})
	}

	t.Run("another machine hashes the same path to another id", func(t *testing.T) {
		now := time.Now()

		_, here := correlationIDs(filepath.Join(t.TempDir(), "telemetry.json"), repo, now)
		_, there := correlationIDs(filepath.Join(t.TempDir(), "telemetry.json"), repo, now)

		assert.NotEqual(t, here, there)
	})
}

// TestCommandPropsCarryNoPath fails if a path a command passes reaches the event.
func TestCommandPropsCarryNoPath(t *testing.T) {
	clearCIEnv(t)

	project := filepath.Join(t.TempDir(), "acme-tracking")
	require.NoError(t, os.MkdirAll(filepath.Join(project, "models"), 0o755))
	t.Chdir(project)

	tests := []struct {
		name      string
		extras    []KV
		forbidden []string
	}{
		{name: "an absolute location", extras: []KV{{K: "location", V: project}}},
		{name: "a relative location", extras: []KV{{K: "location", V: "./models"}}, forbidden: []string{"models"}},
		{name: "no location"},
		{
			name:      "a location and a SQL location",
			extras:    []KV{{K: "location", V: "."}, {K: "sqlLocation", V: "models/orders.sql"}},
			forbidden: []string{"models", "orders"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := filepath.Join(t.TempDir(), "telemetry.json")
			original := stateFile
			stateFile = func() string { return state }
			t.Cleanup(func() { stateFile = original })

			props := commandProps("apply", nil, append(tt.extras, KV{K: "dryRun", V: true})...)

			assert.NotEmpty(t, props["session_id"])
			assert.NotEmpty(t, props["project_id"])
			assert.Equal(t, true, props["dryRun"])

			sent, err := json.Marshal(props)
			require.NoError(t, err)
			for _, word := range append(tt.forbidden, "acme-tracking", "/", `\`) {
				assert.NotContains(t, string(sent), word)
			}
		})
	}
}
