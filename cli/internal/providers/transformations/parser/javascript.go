package parser

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
)

// JavaScriptParser extracts imports from JavaScript code using esbuild
type JavaScriptParser struct{}

// ValidateSyntax validates JavaScript code syntax using esbuild
func (p *JavaScriptParser) ValidateSyntax(code string) error {
	result := api.Transform(code, api.TransformOptions{
		Loader: api.LoaderJS,
	})

	if len(result.Errors) > 0 {
		// Collect all error messages
		var errorMsgs []string
		for _, err := range result.Errors {
			errorMsgs = append(errorMsgs, err.Text)
		}
		return fmt.Errorf("javascript syntax error: %s", strings.Join(errorMsgs, "; "))
	}

	return nil
}

// ExtractImports parses JavaScript code and returns external library import names.
// Returns an error if the code contains:
// - require() syntax (CommonJS not supported)
// - Relative imports (./file, ../file)
// - Absolute imports (/path)
// Filters out RudderStack built-in libraries (@rs/<library>/<version>)
func (p *JavaScriptParser) ExtractImports(code string) ([]string, error) {
	result := api.Transform(code, api.TransformOptions{
		Loader: api.LoaderJS,
		Format: api.FormatESModule,
	})

	if len(result.Errors) > 0 {
		var errorMsgs []string
		for _, err := range result.Errors {
			errorMsgs = append(errorMsgs, err.Text)
		}
		return nil, fmt.Errorf("extracting imports: %s", strings.Join(errorMsgs, "; "))
	}

	transformedCode := string(result.Code)

	// Validate no require() in transformed code
	if err := validateNoRequire(transformedCode); err != nil {
		return nil, err
	}

	// Extract all import statements
	imports := extractAllImports(transformedCode)

	// Validate and filter imports
	libraryImports := make([]string, 0)
	for imp := range imports {
		if isRelativeOrAbsoluteImport(imp) {
			return nil, fmt.Errorf("relative imports (./file, ../file) and absolute imports (/path) are not supported")
		}
		if isExternalLibraryImport(imp) {
			libraryImports = append(libraryImports, imp)
		}
	}

	return libraryImports, nil
}

// validateNoRequire checks if transformed code contains require() calls
func validateNoRequire(code string) error {
	i := 0
	for i < len(code) {
		idx := strings.Index(code[i:], "require")
		if idx == -1 {
			break
		}

		pos := i + idx
		i = pos + 7 // Move past "require"

		// Check if followed by ( with optional whitespace
		for j := pos + 7; j < len(code); j++ {
			ch := code[j]
			if ch == ' ' || ch == '\t' || ch == '\n' {
				continue
			}
			if ch == '(' {
				return fmt.Errorf("require() syntax is not supported")
			}
			break
		}
	}
	return nil
}

// extractAllImports extracts all import paths from esbuild-transformed code
func extractAllImports(code string) map[string]bool {
	imports := make(map[string]bool)

	i := 0
	for i < len(code) {
		idx := strings.Index(code[i:], "import")
		if idx == -1 {
			break
		}

		pos := i + idx
		i = pos + 1

		// Extract module name from import statement
		remaining := code[pos+6:] // Skip "import"

		// Look for "from" keyword
		fromIdx := strings.Index(remaining, "from")
		if fromIdx != -1 {
			remaining = remaining[fromIdx+4:]
		}

		if moduleName := extractQuotedString(remaining); moduleName != "" {
			imports[moduleName] = true
		}
	}

	return imports
}

// extractQuotedString extracts the first quoted string from text
func extractQuotedString(text string) string {
	// Skip whitespace and opening parenthesis
	start := 0
	for start < len(text) && (text[start] == ' ' || text[start] == '\t' || text[start] == '\n' || text[start] == '(') {
		start++
	}

	if start >= len(text) {
		return ""
	}

	// Find opening quote
	quote := text[start]
	if quote != '"' && quote != '\'' && quote != '`' {
		return ""
	}

	// Find closing quote
	for i := start + 1; i < len(text); i++ {
		if text[i] == quote {
			return text[start+1 : i]
		}
	}

	return ""
}

// isRelativeOrAbsoluteImport checks if import is relative (./file, ../file) or absolute (/path)
func isRelativeOrAbsoluteImport(path string) bool {
	return strings.HasPrefix(path, "./") || strings.HasPrefix(path, "../") || strings.HasPrefix(path, "/")
}

// rudderStackLibraryPattern matches RudderStack built-in library imports
// Pattern: @rs/<library>/v<digits>
// Examples: @rs/hash/v1, @rs/utils/v2, @rs/crypto/v10
var rudderStackLibraryPattern = regexp.MustCompile(`^@rs/[^/]+/v\d+$`)

func isRudderStackLibrary(path string) bool {
	return rudderStackLibraryPattern.MatchString(path)
}

// isExternalLibraryImport checks if import is an external library (not RudderStack built-in)
func isExternalLibraryImport(path string) bool {
	return path != "" && !isRudderStackLibrary(path)
}
