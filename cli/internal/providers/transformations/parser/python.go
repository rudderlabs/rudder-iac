package parser

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// PythonParser extracts imports from Python code using regex
type PythonParser struct{}

// ValidateSyntax is not implemented for regex-based parser
// Python syntax validation requires a proper parser
func (p *PythonParser) ValidateSyntax(code string) error {
	return nil
}

// ExtractImports parses Python code and returns external library import names.
// Uses regex-based extraction after stripping comments.
// Handles multi-line imports with parentheses and backslash continuations.
func (p *PythonParser) ExtractImports(code string) ([]string, error) {
	// Sanitize code to avoid false positives
	cleanCode := sanitizeCode(code)

	// Normalize multi-line imports and split semicolon-separated statements
	normalizedCode := normalizeMultilineImports(cleanCode)

	// Extract imports using regex
	modules, err := extractImportStatements(normalizedCode)
	if err != nil {
		return nil, err
	}

	// Filter out base whitelist and deduplicate
	moduleSet := make(map[string]struct{})
	for _, module := range modules {
		topLevel := strings.Split(module, ".")[0]
		if !isPythonBuiltinModule(topLevel) {
			moduleSet[topLevel] = struct{}{}
		}
	}

	var externalImports []string
	for module := range moduleSet {
		externalImports = append(externalImports, module)
	}

	return externalImports, nil
}

var (
	tripleDoubleQuoteRe = regexp.MustCompile(`"""[\s\S]*?"""`)
	tripleSingleQuoteRe = regexp.MustCompile(`'''[\s\S]*?'''`)
	doubleQuoteStringRe = regexp.MustCompile(`"(?:[^"\\]|\\.)*"`)
	singleQuoteStringRe = regexp.MustCompile(`'(?:[^'\\]|\\.)*'`)
	singleLineCommentRe = regexp.MustCompile(`#[^\r\n]*`)
)

// sanitizeCode removes comments and string contents that could contain false import statements.
// Processing order matters: triple-quoted strings before single-quoted to handle nested quotes.
func sanitizeCode(code string) string {
	// Step 1: Remove triple-quoted strings (""" and ''')
	code = tripleDoubleQuoteRe.ReplaceAllString(code, "")
	code = tripleSingleQuoteRe.ReplaceAllString(code, "")

	// Step 2: Empty out single/double-quoted string contents (preserve quotes for structure)
	code = doubleQuoteStringRe.ReplaceAllString(code, "\"\"")
	code = singleQuoteStringRe.ReplaceAllString(code, "''")

	// Step 3: Remove single-line comments (#...)
	code = singleLineCommentRe.ReplaceAllString(code, "")

	return code
}

var (
	// Matches: from X import (\n...\n) - parentheses spanning multiple lines
	parenImportRe = regexp.MustCompile(`(?m)((?:from|import)\s+[^(]*\([^)]*)\n([^)]*\))`)

	// Matches: line ending with backslash continuation
	backslashContinueRe = regexp.MustCompile(`(?m)\\\n\s*`)
)

// normalizeMultilineImports joins multi-line imports into single lines.
// Handles: parentheses () and backslash \ continuations.
func normalizeMultilineImports(code string) string {
	// Join backslash-continued lines
	code = backslashContinueRe.ReplaceAllString(code, " ")

	// Join parentheses-continued imports (may need multiple passes for deeply nested)
	for parenImportRe.MatchString(code) {
		code = parenImportRe.ReplaceAllString(code, "$1 $2")
	}

	return code
}

// Regex patterns for Python imports
var (
	// import foo, bar, baz
	// import foo as f, bar as b
	simpleImportPattern = regexp.MustCompile(`^import\s+(.+)$`)

	// from foo import bar, baz
	// from foo.bar import baz as b
	// from foo import *
	// from foo import (bar, baz)
	fromImportPattern = regexp.MustCompile(`^from\s+(\S+)\s+import\s+`)
)

// extractImportStatements extracts module names from import statements
func extractImportStatements(code string) ([]string, error) {
	var modules []string

	for line := range strings.SplitSeq(code, "\n") {
		line = strings.TrimSpace(line)

		// Early skip: only process lines containing import statements
		if !strings.Contains(line, "import") {
			continue
		}

		// Split on semicolons to handle multiple statements on one line
		for stmt := range strings.SplitSeq(line, ";") {
			stmt = strings.TrimSpace(stmt)

			switch {
			case strings.HasPrefix(stmt, "from "):
				module, err := parseFromImport(stmt)
				if err != nil {
					return nil, err
				}
				if module != "" {
					modules = append(modules, module)
				}

			case strings.HasPrefix(stmt, "import "):
				parsed, err := parseSimpleImport(stmt)
				if err != nil {
					return nil, err
				}
				modules = append(modules, parsed...)
			}
		}
	}

	return modules, nil
}

// parseFromImport extracts the module name from "from X import Y" statement.
func parseFromImport(stmt string) (string, error) {
	matches := fromImportPattern.FindStringSubmatch(stmt)
	if matches == nil {
		return "", nil
	}
	module := matches[1]
	if isRelativeImport(module) {
		return "", fmt.Errorf("relative imports (from . or from ..) are not supported")
	}
	return module, nil
}

// parseSimpleImport extracts module names from "import X, Y as alias, Z" statement.
func parseSimpleImport(stmt string) ([]string, error) {
	matches := simpleImportPattern.FindStringSubmatch(stmt)
	if matches == nil {
		return nil, nil
	}

	importPart := strings.NewReplacer("(", "", ")", "").Replace(matches[1])

	var modules []string
	for part := range strings.SplitSeq(importPart, ",") {
		part = strings.TrimSpace(part)
		if idx := strings.Index(part, " as "); idx != -1 {
			part = part[:idx]
		}
		if part == "" {
			continue
		}
		if isRelativeImport(part) {
			return nil, fmt.Errorf("relative imports (from . or from ..) are not supported")
		}
		modules = append(modules, part)
	}
	return modules, nil
}

// isRelativeImport checks if a module path is a relative import
func isRelativeImport(module string) bool {
	return strings.HasPrefix(module, ".")
}

// pythonBuiltinModules contains modules that don't need to be tracked as external dependencies
var pythonBuiltinModules = []string{
	"ast", "base64", "collections", "datetime", "dateutil", "hashlib",
	"hmac", "json", "math", "random", "re", "requests", "string",
	"time", "uuid", "urllib", "utils", "copy", "_strptime", "typing",
}

func isPythonBuiltinModule(module string) bool {
	return slices.Contains(pythonBuiltinModules, module)
}
