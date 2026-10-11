package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ProjectInfo is what init saw in the working directory.
type ProjectInfo struct {
	// Ecosystem is empty when no known manifest was found.
	Ecosystem  string
	Manifest   string
	Frameworks []string
	TypeScript bool
	// Note carries a non-fatal problem, such as a package.json that does not parse.
	Note string
}

// Summary renders the one-line description init prints.
func (p ProjectInfo) Summary() string {
	if p.Ecosystem == "" {
		return "unknown (no recognised manifest in this directory)"
	}

	parts := []string{fmt.Sprintf("%s (%s)", p.Ecosystem, p.Manifest)}
	if len(p.Frameworks) > 0 {
		parts = append(parts, "frameworks: "+strings.Join(p.Frameworks, ", "))
	}
	if p.Ecosystem == ecosystemNode {
		parts = append(parts, "TypeScript: "+yesNo(p.TypeScript))
	}
	if p.Note != "" {
		parts = append(parts, p.Note)
	}
	return strings.Join(parts, "; ")
}

func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

const ecosystemNode = "Node.js"

// nodeFrameworks maps a package.json dependency to a display name. The order
// is the order they print in.
var nodeFrameworks = []struct{ dep, name string }{
	{"next", "Next.js"},
	{"nuxt", "Nuxt"},
	{"@remix-run/react", "Remix"},
	{"@sveltejs/kit", "SvelteKit"},
	{"@angular/core", "Angular"},
	{"expo", "Expo"},
	{"react-native", "React Native"},
	{"vite", "Vite"},
	{"react", "React"},
	{"vue", "Vue"},
	{"svelte", "Svelte"},
	{"express", "Express"},
}

// otherManifests lists the files that identify a non-Node ecosystem. The first
// match wins, and only Node projects are inspected any further.
var otherManifests = []struct{ file, ecosystem string }{
	{"go.mod", "Go"},
	{"pyproject.toml", "Python"},
	{"requirements.txt", "Python"},
	{"setup.py", "Python"},
	{"Gemfile", "Ruby"},
	{"pom.xml", "Java"},
	{"build.gradle", "Java or Kotlin"},
	{"build.gradle.kts", "Java or Kotlin"},
	{"Package.swift", "Swift"},
	{"pubspec.yaml", "Dart or Flutter"},
	{"Cargo.toml", "Rust"},
	{"composer.json", "PHP"},
}

// otherGlobs identifies ecosystems whose manifest name is not fixed.
var otherGlobs = []struct{ pattern, ecosystem string }{
	{"*.xcodeproj", "Swift or Objective-C (Xcode)"},
	{"*.csproj", ".NET"},
	{"*.sln", ".NET"},
}

// DetectProject inspects dir without modifying it.
func DetectProject(dir string) ProjectInfo {
	if info, ok := detectNode(dir); ok {
		return info
	}

	for _, m := range otherManifests {
		if fileExists(filepath.Join(dir, m.file)) {
			return ProjectInfo{Ecosystem: m.ecosystem, Manifest: m.file}
		}
	}

	for _, g := range otherGlobs {
		matches, err := filepath.Glob(filepath.Join(dir, g.pattern))
		if err != nil || len(matches) == 0 {
			continue
		}
		return ProjectInfo{Ecosystem: g.ecosystem, Manifest: filepath.Base(matches[0])}
	}

	return ProjectInfo{}
}

func detectNode(dir string) (ProjectInfo, bool) {
	raw, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return ProjectInfo{}, false
	}

	info := ProjectInfo{
		Ecosystem:  ecosystemNode,
		Manifest:   "package.json",
		TypeScript: fileExists(filepath.Join(dir, "tsconfig.json")),
	}

	var pkg struct {
		Dependencies    map[string]json.RawMessage `json:"dependencies"`
		DevDependencies map[string]json.RawMessage `json:"devDependencies"`
	}
	if err := json.Unmarshal(raw, &pkg); err != nil {
		info.Note = "package.json could not be parsed"
		return info, true
	}

	has := func(dep string) bool {
		if _, ok := pkg.Dependencies[dep]; ok {
			return true
		}
		_, ok := pkg.DevDependencies[dep]
		return ok
	}

	for _, f := range nodeFrameworks {
		if has(f.dep) {
			info.Frameworks = append(info.Frameworks, f.name)
		}
	}
	if has("typescript") {
		info.TypeScript = true
	}

	return info, true
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
