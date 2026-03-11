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

// pythonBuiltinModules contains Python stdlib and builtin modules that should be filtered
// from import extraction since they are available in the RudderStack transformation runtime.
var pythonBuiltinModules = map[string]struct{}{
	"ast":         {},
	"base64":      {},
	"collections": {},
	"datetime":    {},
	"dateutil":    {},
	"hashlib":     {},
	"hmac":        {},
	"json":        {},
	"math":        {},
	"random":      {},
	"re":          {},
	"requests":    {},
	"string":      {},
	"time":        {},
	"uuid":        {},
	"urllib":      {},
	"utils":       {},
	"copy":        {},
	"_strptime":   {},
	"typing":      {},
}

// ValidateSyntax is a no-op in this phase; full Python syntax validation
// requires embedding a Python interpreter or an equivalent AST library.
func (p *PythonParser) ValidateSyntax(code string) error {
	return nil
}

// ExtractImports parses Python source and returns the deduplicated, sorted list of
// top-level module names referenced by import statements, excluding Python builtin modules.
//
// Relative imports are rejected with an error because transformations must use
// absolute imports. Builtin/stdlib modules are filtered automatically.
func (p *PythonParser) ExtractImports(code string) ([]string, error) {
	candidates := scanImportCandidates(code)

	moduleSet := make(map[string]struct{})
	for _, c := range candidates {
		stmt, err := importParser.ParseString("", c.text)
		if err != nil {
			// Candidate contains "import" but is not a valid import statement
			// (e.g. "importlib.load_module()" slipped through the scanner filter).
			// Skip rather than hard-fail.
			continue
		}

		modules, err := extractModules(stmt)
		if err != nil {
			return nil, err
		}
		for _, m := range modules {
			// Filter out builtin modules
			if _, isBuiltin := pythonBuiltinModules[m]; !isBuiltin {
				moduleSet[m] = struct{}{}
			}
		}
	}

	result := make([]string, 0, len(moduleSet))
	for m := range moduleSet {
		result = append(result, m)
	}
	sort.Strings(result)

	return result, nil
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
		// Malformed statement — skip.
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
		// Relative simple imports (import .foo) are not valid Python syntax, but guard
		// for safety since the scanner operates on text, not a verified AST.
		if strings.HasPrefix(top, ".") {
			return nil, fmt.Errorf("relative imports (from . or from ..) are not supported")
		}
		modules = append(modules, top)
	}
	return modules, nil
}
