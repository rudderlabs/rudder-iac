package parser

import (
	"fmt"
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

// ExtractImports parses JavaScript code and returns library import names using esbuild
func (p *JavaScriptParser) ExtractImports(code string) ([]string, error) {
	// Transform the code to extract dependencies
	// Using Transform with JSX loader to handle modern JS syntax
	result := api.Transform(code, api.TransformOptions{
		Loader: api.LoaderJSX,
		Format: api.FormatESModule,
	})

	if len(result.Errors) > 0 {
		var errorMsgs []string
		for _, err := range result.Errors {
			errorMsgs = append(errorMsgs, err.Text)
		}
		return nil, fmt.Errorf("extracting imports: %s", strings.Join(errorMsgs, "; "))
	}

	// Parse the transformed output to extract import paths
	// esbuild preserves imports in the transformed code
	imports := extractImportsFromTransformedCode(string(result.Code))

	// Convert map to slice, filtering library imports only
	libraryImports := make([]string, 0, len(imports))
	for imp := range imports {
		if isLibraryImport(imp) {
			libraryImports = append(libraryImports, imp)
		}
	}

	return libraryImports, nil
}

// extractImportsFromTransformedCode extracts import paths from esbuild-transformed code
// esbuild preserves import/require statements in the output, making them easy to find
func extractImportsFromTransformedCode(code string) map[string]bool {
	imports := make(map[string]bool)

	// Scan through the code looking for import/require statements
	// esbuild's output is well-formatted, making this reliable
	i := 0
	for i < len(code) {
		// Look for "import" or "require"
		importIdx := strings.Index(code[i:], "import")
		requireIdx := strings.Index(code[i:], "require")

		nextImport := -1
		isImport := false

		if importIdx != -1 && (requireIdx == -1 || importIdx < requireIdx) {
			nextImport = i + importIdx
			isImport = true
		} else if requireIdx != -1 {
			nextImport = i + requireIdx
			isImport = false
		} else {
			break // No more imports/requires
		}

		i = nextImport + 1

		// Extract the module name
		var remaining string
		if isImport {
			// For imports, look for "from" keyword
			remaining = code[nextImport+6:] // Skip "import"
			fromIdx := strings.Index(remaining, "from")
			if fromIdx != -1 {
				remaining = remaining[fromIdx+4:]
			}
		} else {
			// For requires, start right after "require"
			remaining = code[nextImport+7:] // Skip "require"
		}

		// Extract quoted string
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

// isLibraryImport checks if the import path is a library (not relative/absolute path)
func isLibraryImport(path string) bool {
	// Filter out relative imports (./file, ../file) and absolute paths (/path)
	return path != "" && !strings.HasPrefix(path, ".") && !strings.HasPrefix(path, "/")
}
