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
	// Step 1: Remove comments to avoid false positives
	cleanCode := removeComments(code)

	// Step 2: Normalize multi-line imports and split semicolon-separated statements
	normalizedCode := normalizeMultilineImports(cleanCode)

	// Step 3: Extract imports using regex
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

	// Convert set to slice
	var externalImports []string
	for module := range moduleSet {
		externalImports = append(externalImports, module)
	}

	return externalImports, nil
}

// removeComments strips Python comments from code
// Handles: # single-line comments, """ and ''' multi-line strings/docstrings
func removeComments(code string) string {
	var result strings.Builder
	lines := strings.Split(code, "\n")

	inMultilineString := false
	multilineDelimiter := ""

	for _, line := range lines {
		if inMultilineString {
			// Check if this line ends the multi-line string
			if idx := strings.Index(line, multilineDelimiter); idx != -1 {
				inMultilineString = false
				// Keep content after the closing delimiter
				line = line[idx+3:]
			} else {
				// Skip entire line - still in multi-line string
				result.WriteString("\n")
				continue
			}
		}

		// Process the line for comments and multi-line strings
		processedLine := processLine(line, &inMultilineString, &multilineDelimiter)
		result.WriteString(processedLine)
		result.WriteString("\n")
	}

	return result.String()
}

// processLine handles single-line comment removal and multi-line string detection
func processLine(line string, inMultilineString *bool, multilineDelimiter *string) string {
	var result strings.Builder
	i := 0
	inString := false
	stringChar := byte(0)

	for i < len(line) {
		// Check for multi-line string start (""" or ''')
		if !inString && i+2 < len(line) {
			if (line[i] == '"' && line[i+1] == '"' && line[i+2] == '"') ||
				(line[i] == '\'' && line[i+1] == '\'' && line[i+2] == '\'') {
				delimiter := line[i : i+3]
				// Check if it closes on the same line
				closeIdx := strings.Index(line[i+3:], delimiter)
				if closeIdx != -1 {
					// Multi-line string opens and closes on same line - skip it
					i = i + 3 + closeIdx + 3
					continue
				}
				// Starts multi-line string that continues to next line
				*inMultilineString = true
				*multilineDelimiter = delimiter
				return result.String()
			}
		}

		// Check for single-line string
		if !inString && (line[i] == '"' || line[i] == '\'') {
			inString = true
			stringChar = line[i]
			result.WriteByte(line[i])
			i++
			continue
		}

		// Check for end of single-line string
		if inString && line[i] == stringChar {
			// Check it's not escaped
			escaped := false
			j := i - 1
			for j >= 0 && line[j] == '\\' {
				escaped = !escaped
				j--
			}
			if !escaped {
				inString = false
			}
			result.WriteByte(line[i])
			i++
			continue
		}

		// Check for comment (only if not in string)
		if !inString && line[i] == '#' {
			// Rest of line is comment - stop processing
			return result.String()
		}

		result.WriteByte(line[i])
		i++
	}

	return result.String()
}

// normalizeMultilineImports joins multi-line imports into single lines
// Handles: parentheses () and backslash \ continuations
func normalizeMultilineImports(code string) string {
	var result strings.Builder
	lines := strings.Split(code, "\n")

	i := 0
	for i < len(lines) {
		line := strings.TrimSpace(lines[i])

		// Check if this is an import statement with parentheses
		if (strings.HasPrefix(line, "import ") || strings.HasPrefix(line, "from ")) &&
			strings.Contains(line, "(") && !strings.Contains(line, ")") {
			// Multi-line import with parentheses - collect until closing )
			var combined strings.Builder
			combined.WriteString(line)
			i++
			for i < len(lines) {
				nextLine := strings.TrimSpace(lines[i])
				combined.WriteString(" ")
				combined.WriteString(nextLine)
				if strings.Contains(nextLine, ")") {
					break
				}
				i++
			}
			result.WriteString(combined.String())
			result.WriteString("\n")
			i++
			continue
		}

		// Check for backslash continuation
		if (strings.HasPrefix(line, "import ") || strings.HasPrefix(line, "from ")) &&
			strings.HasSuffix(line, "\\") {
			var combined strings.Builder
			combined.WriteString(strings.TrimSuffix(line, "\\"))
			i++
			for i < len(lines) {
				nextLine := strings.TrimSpace(lines[i])
				combined.WriteString(" ")
				if before, ok := strings.CutSuffix(nextLine, "\\"); ok  {
					combined.WriteString(before)
					i++
				} else {
					combined.WriteString(nextLine)
					break
				}
			}
			result.WriteString(combined.String())
			result.WriteString("\n")
			i++
			continue
		}

		result.WriteString(line)
		result.WriteString("\n")
		i++
	}

	return result.String()
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
	lines := strings.SplitSeq(code, "\n")

	for line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Split on semicolons to handle multiple statements on one line
		for stmt := range strings.SplitSeq(line, ";") {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}

			// Check for "from X import Y" pattern
			if matches := fromImportPattern.FindStringSubmatch(stmt); matches != nil {
				module := matches[1]
				// Check for relative imports
				if isRelativeImport(module) {
					return nil, fmt.Errorf("relative imports (from . or from ..) are not supported")
				}
				modules = append(modules, module)
				continue
			}

			// Check for "import X, Y, Z" pattern
			if matches := simpleImportPattern.FindStringSubmatch(stmt); matches != nil {
				// Parse the imports (handles "import a, b as alias, c")
				importPart := matches[1]
				importPart = strings.ReplaceAll(importPart, "(", "")
				importPart = strings.ReplaceAll(importPart, ")", "")

				parts := strings.SplitSeq(importPart, ",")
				for part := range parts {
					part = strings.TrimSpace(part)
					// Handle "module as alias" - extract just the module name
					if asIdx := strings.Index(part, " as "); asIdx != -1 {
						part = part[:asIdx]
					}
					part = strings.TrimSpace(part)
					if part != "" {
						// Check for relative imports
						if isRelativeImport(part) {
							return nil, fmt.Errorf("relative imports (from . or from ..) are not supported")
						}
						modules = append(modules, part)
					}
				}
			}
		}
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
