package parser

import (
	"fmt"
	"sort"
	"strings"
)

// PythonParser extracts imports from Python source code using a two-phase pipeline:
// a character-level scanner (phase 1) that safely skips strings and comments, followed
// by a participle grammar parser (phase 2) that produces typed ImportStatement AST nodes.
type PythonParser struct{}

// ValidateSyntax is a no-op in this phase; full Python syntax validation
// requires embedding a Python interpreter or an equivalent AST library.
func (p *PythonParser) ValidateSyntax(code string) error {
	return nil
}

// ExtractImports parses Python source and returns the deduplicated, sorted list of
// top-level external module names referenced by import statements.
//
// Relative imports are rejected with an error because transformations must use
// absolute imports. Builtin/stdlib filtering is delegated to the provider layer.
func (p *PythonParser) ExtractImports(code string) ([]string, error) {
	candidates := scanImportCandidates(code)

	moduleSet := make(map[string]struct{})
	for _, c := range candidates {
		text := trimSuitePrefix(c.text)
		stmt, err := importParser.ParseString("", text)
		if err != nil {
			// Not a valid import statement (e.g. "import" appears in an identifier
			// or assignment). Skip rather than hard-fail.
			continue
		}

		modules, err := extractModules(stmt)
		if err != nil {
			return nil, err
		}
		for _, m := range modules {
			moduleSet[m] = struct{}{}
		}
	}

	result := make([]string, 0, len(moduleSet))
	for m := range moduleSet {
		result = append(result, m)
	}
	sort.Strings(result)

	return result, nil
}

// trimSuitePrefix strips the compound-statement header that precedes an import
// in a one-line suite (e.g. "if True: import x" → "import x").
// Python allows any simple statement after a colon on the same line.
func trimSuitePrefix(s string) string {
	if strings.HasPrefix(s, "import ") || strings.HasPrefix(s, "from ") {
		return s
	}
	if idx := strings.LastIndex(s, ":"); idx != -1 {
		rest := strings.TrimSpace(s[idx+1:])
		if strings.HasPrefix(rest, "import ") || strings.HasPrefix(rest, "from ") {
			return rest
		}
	}
	return s
}

// extractModules pulls top-level module names out of a parsed ImportStatement.
// Returns an error when a relative import is detected.
func extractModules(stmt *ImportStatement) ([]string, error) {
	if stmt.From != nil {
		return extractFromImportModules(stmt.From)
	}
	if stmt.Simple != nil {
		return extractSimpleImportModules(stmt.Simple)
	}
	return nil, nil
}

func extractFromImportModules(fi *FromImport) ([]string, error) {
	if len(fi.Dots) > 0 {
		return nil, fmt.Errorf("relative imports (from . or from ..) are not supported")
	}
	if fi.Module == nil {
		// "from import ..." — malformed, skip
		return nil, nil
	}
	// Return only the top-level package name (e.g. "mylib" from "mylib.sub").
	return []string{fi.Module.Parts[0]}, nil
}

func extractSimpleImportModules(si *SimpleImport) ([]string, error) {
	modules := make([]string, 0, len(si.Modules))
	for _, ma := range si.Modules {
		if len(ma.Path.Parts) == 0 {
			continue
		}
		top := ma.Path.Parts[0]
		// Relative simple imports (import .foo) are not valid Python syntax, but
		// guard here for safety — the scanner won't emit them, but belt-and-suspenders.
		if strings.HasPrefix(top, ".") {
			return nil, fmt.Errorf("relative imports (from . or from ..) are not supported")
		}
		modules = append(modules, top)
	}
	return modules, nil
}
