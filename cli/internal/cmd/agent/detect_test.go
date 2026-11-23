package agent

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectProject(t *testing.T) {
	tests := []struct {
		name     string
		files    map[string]string
		expected ProjectInfo
		summary  string
	}{
		{
			name: "vite react typescript",
			files: map[string]string{
				"package.json": `{"dependencies":{"react":"^18"},"devDependencies":{"vite":"^5","typescript":"^5"}}`,
			},
			expected: ProjectInfo{
				Ecosystem:  "Node.js",
				Manifest:   "package.json",
				Frameworks: []string{"Vite", "React"},
				TypeScript: true,
			},
			summary: "Node.js (package.json); frameworks: Vite, React; TypeScript: yes",
		},
		{
			name: "next with tsconfig only",
			files: map[string]string{
				"package.json":  `{"dependencies":{"next":"14","react":"18"}}`,
				"tsconfig.json": `{}`,
			},
			expected: ProjectInfo{
				Ecosystem:  "Node.js",
				Manifest:   "package.json",
				Frameworks: []string{"Next.js", "React"},
				TypeScript: true,
			},
			summary: "Node.js (package.json); frameworks: Next.js, React; TypeScript: yes",
		},
		{
			name:     "plain javascript",
			files:    map[string]string{"package.json": `{"name":"x"}`},
			expected: ProjectInfo{Ecosystem: "Node.js", Manifest: "package.json"},
			summary:  "Node.js (package.json); TypeScript: no",
		},
		{
			name:     "unparseable package.json",
			files:    map[string]string{"package.json": `{nope`},
			expected: ProjectInfo{Ecosystem: "Node.js", Manifest: "package.json", Note: "package.json could not be parsed"},
			summary:  "Node.js (package.json); TypeScript: no; package.json could not be parsed",
		},
		{
			name:     "go module",
			files:    map[string]string{"go.mod": "module x\n"},
			expected: ProjectInfo{Ecosystem: "Go", Manifest: "go.mod"},
			summary:  "Go (go.mod)",
		},
		{
			name:     "python",
			files:    map[string]string{"requirements.txt": "flask\n"},
			expected: ProjectInfo{Ecosystem: "Python", Manifest: "requirements.txt"},
			summary:  "Python (requirements.txt)",
		},
		{
			name:     "swift package",
			files:    map[string]string{"Package.swift": ""},
			expected: ProjectInfo{Ecosystem: "Swift", Manifest: "Package.swift"},
			summary:  "Swift (Package.swift)",
		},
		{
			name:     "node wins over other manifests",
			files:    map[string]string{"package.json": `{}`, "go.mod": "module x\n"},
			expected: ProjectInfo{Ecosystem: "Node.js", Manifest: "package.json"},
			summary:  "Node.js (package.json); TypeScript: no",
		},
		{
			name:     "empty directory",
			files:    map[string]string{},
			expected: ProjectInfo{},
			summary:  "unknown (no recognised manifest in this directory)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for name, content := range tt.files {
				require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600))
			}

			got := DetectProject(dir)
			assert.Equal(t, tt.expected, got)
			assert.Equal(t, tt.summary, got.Summary())
		})
	}
}

func TestDetectProjectXcodeProject(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, "App.xcodeproj"), 0o755))

	assert.Equal(t, ProjectInfo{Ecosystem: "Swift or Objective-C (Xcode)", Manifest: "App.xcodeproj"}, DetectProject(dir))
}
