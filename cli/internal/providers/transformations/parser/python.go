package parser

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/kluctl/go-embed-python/python"
)

// PythonParser extracts imports from Python code using embedded Python runtime
type PythonParser struct {
	ep   *python.EmbeddedPython
	once sync.Once
	err  error
}

// initEmbeddedPython initializes the embedded Python runtime lazily
func (p *PythonParser) initEmbeddedPython() error {
	p.once.Do(func() {
		ep, err := python.NewEmbeddedPython("python")
		if err != nil {
			p.err = fmt.Errorf("initializing embedded python: %w", err)
			return
		}
		p.ep = ep
	})
	return p.err
}

// ValidateSyntax validates Python code syntax using Python's ast.parse
func (p *PythonParser) ValidateSyntax(code string) error {
	if err := p.initEmbeddedPython(); err != nil {
		return err
	}

	// Python script to validate syntax
	script := `import ast
import sys
import json

code = sys.stdin.read()
try:
    ast.parse(code)
    print(json.dumps({"success": True}))
except SyntaxError as e:
    print(json.dumps({
        "success": False,
        "error": str(e),
        "line": e.lineno,
        "offset": e.offset,
        "text": e.text
    }))
`

	cmd, err := p.ep.PythonCmd("-c", script)
	if err != nil {
		return fmt.Errorf("creating python command: %w", err)
	}
	cmd.Stdin = strings.NewReader(code)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("running python validation: %w", err)
	}

	var result struct {
		Success bool   `json:"success"`
		Error   string `json:"error"`
		Line    int    `json:"line"`
		Offset  int    `json:"offset"`
		Text    string `json:"text"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return fmt.Errorf("parsing validation result: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("python syntax error: %s", result.Error)
	}

	return nil
}

// ExtractImports parses Python code and returns external library import names.
// Based on the reference implementation from the transformation service.
// Validates that all imports are in the MODULE_WHITELIST (BASE_MODULE_WHITELIST).
// Returns only external libraries that are not in the whitelist.
func (p *PythonParser) ExtractImports(code string) ([]string, error) {
	if err := p.initEmbeddedPython(); err != nil {
		return nil, err
	}

	// Build whitelist JSON from BASE_MODULE_WHITELIST constant
	whitelistJSON, err := json.Marshal(BASE_MODULE_WHITELIST)
	if err != nil {
		return nil, fmt.Errorf("marshaling whitelist: %w", err)
	}

	// Python script matching the reference parse_code_and_get_modules implementation
	// Note: validate_imports is set to False for extraction - validation happens separately
	// Whitelist is passed via stdin as the first line
	script := `import ast
import sys
import json

# Read whitelist from first line of stdin
whitelist_line = sys.stdin.readline()
MODULE_WHITELIST = set(json.loads(whitelist_line))

# Read the actual code
code = sys.stdin.read()

try:
    root = ast.parse(code)
except SyntaxError as e:
    print(json.dumps({
        "success": False,
        "error": str(e)
    }))
    sys.exit(0)

modules = {}
validate_imports = False  # Extraction mode - just collect imports

for node in ast.walk(root):
    if isinstance(node, ast.Import):
        module = []
    elif isinstance(node, ast.ImportFrom):
        module = node.module.split(".") if node.module else []
    else:
        continue

    for n in node.names:
        # Create result tuple (module, name, alias)
        name_parts = n.name.split(".")

        module_obj = {}
        name = None

        if len(module) > 0:
            module_obj["name"] = ".".join(module)
            name = module[0]
        else:
            module_obj["name"] = ".".join(name_parts)
            name = name_parts[0]

        # Validate imports against whitelist (when validate_imports is True)
        if validate_imports:
            if name not in MODULE_WHITELIST:
                print(json.dumps({
                    "success": False,
                    "error": f"Unpermitted import(s). Supported modules / packages for import: {sorted(MODULE_WHITELIST)}",
                    "unpermitted_import": name
                }))
                sys.exit(0)

        modules[module_obj["name"]] = module_obj

# Extract module names
module_values = list(modules.values())
print(json.dumps({
    "success": True,
    "modules": module_values
}))
`

	cmd, err := p.ep.PythonCmd("-c", script)
	if err != nil {
		return nil, fmt.Errorf("creating python command: %w", err)
	}
	cmd.Stdin = strings.NewReader(fmt.Sprintf("%s\n%s", string(whitelistJSON), code))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("running python import extraction: %w", err)
	}

	var result struct {
		Success           bool   `json:"success"`
		Error             string `json:"error"`
		Modules           []map[string]string `json:"modules"`
		UnpermittedImport string `json:"unpermitted_import"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("parsing import extraction result: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("%s", result.Error)
	}

	// Extract top-level module names and filter out BASE_MODULE_WHITELIST
	moduleSet := make(map[string]bool)
	for _, moduleObj := range result.Modules {
		if moduleName, ok := moduleObj["name"]; ok {
			nameParts := strings.Split(moduleName, ".")
			topLevel := nameParts[0]
			if !isPythonBaseWhitelist(topLevel) {
				moduleSet[topLevel] = true
			}
		}
	}

	// Convert set to slice
	var externalImports []string
	for module := range moduleSet {
		externalImports = append(externalImports, module)
	}

	return externalImports, nil
}

// BASE_MODULE_WHITELIST matches the whitelist from the reference implementation
// These are the modules that are allowed to be imported without being tracked as external dependencies
var BASE_MODULE_WHITELIST = []string{
	"ast", "base64", "collections", "datetime", "dateutil", "hashlib",
	"hmac", "json", "math", "random", "re", "requests", "string",
	"time", "uuid", "urllib", "utils", "copy", "_strptime", "typing",
}

// isPythonBaseWhitelist checks if module is part of the BASE_MODULE_WHITELIST
func isPythonBaseWhitelist(module string) bool {
	return slices.Contains(BASE_MODULE_WHITELIST, module)
}
