package parser

import (
	"encoding/json"
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
		return fmt.Errorf("javascript syntax error: \n\t%s", strings.Join(errorMsgs, "\n\t"))
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
	// Use esbuild's Build with Metafile to get structured import data
	result := api.Build(api.BuildOptions{
		Stdin: &api.StdinOptions{
			Contents:   code,
			Loader:     api.LoaderJS,
			ResolveDir: ".",
		},
		Format:   api.FormatESModule, // Needed to track require() calls
		Bundle:   false,              // Don't actually bundle, just analyze
		Metafile: true,               // Enable metadata output
		Write:    false,              // Don't write to disk
		LogLevel: api.LogLevelSilent,
	})

	if len(result.Errors) > 0 {
		var errorMsgs []string
		for _, err := range result.Errors {
			errorMsgs = append(errorMsgs, err.Text)
		}
		return nil, fmt.Errorf("extracting imports: \n\t%s", strings.Join(errorMsgs, "\n\t"))
	}

	// Parse the metafile JSON to get imports from outputs
	var meta struct {
		Outputs map[string]struct {
			Imports []struct {
				Path     string `json:"path"`
				Kind     string `json:"kind"`     // "import-statement", "require-call"
				External bool   `json:"external"` // true for external libraries
			} `json:"imports"`
		} `json:"outputs"`
	}

	if err := json.Unmarshal([]byte(result.Metafile), &meta); err != nil {
		return nil, fmt.Errorf("parsing metafile: %w", err)
	}

	// Extract and validate imports
	var libraryImports []string
	seen := make(map[string]bool)

	for _, output := range meta.Outputs {
		for _, imp := range output.Imports {
			// Check for require() - not supported
			if imp.Kind == "require-call" {
				return nil, fmt.Errorf("require() syntax is not supported")
			}

			// Check for dynamic imports - skip them (not extracting, but also not erroring)
			if imp.Kind == "dynamic-import" {
				return nil, fmt.Errorf("dynamic imports are not supported")
			}

			// Check for relative/absolute imports
			if isRelativeOrAbsoluteImport(imp.Path) {
				return nil, fmt.Errorf("relative imports (./file, ../file) and absolute imports (/path) are not supported")
			}

			// Filter external libraries and deduplicate
			if isExternalLibraryImport(imp.Path) && !seen[imp.Path] {
				libraryImports = append(libraryImports, imp.Path)
				seen[imp.Path] = true
			}
		}
	}

	return libraryImports, nil
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
