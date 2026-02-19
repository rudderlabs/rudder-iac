package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPythonParser_ExtractImports(t *testing.T) {
	parser := &PythonParser{}

	tests := []struct {
		name     string
		code     string
		expected []string
	}{
		{
			name:     "Simple import statement",
			code:     `import mylib`,
			expected: []string{"mylib"},
		},
		{
			name:     "Multiple imports on one line",
			code:     `import mylib1, mylib2, mylib3`,
			expected: []string{"mylib1", "mylib2", "mylib3"},
		},
		{
			name:     "From import statement",
			code:     `from mylib import func1, func2`,
			expected: []string{"mylib"},
		},
		{
			name: "Multiple import statements",
			code: `
import mylib1
from mylib2 import func
import mylib3
`,
			expected: []string{"mylib1", "mylib2", "mylib3"},
		},
		{
			name: "Mixed import styles",
			code: `
import mylib1
from mylib2 import something
from mylib3.submodule import another
`,
			expected: []string{"mylib1", "mylib2", "mylib3"},
		},
		{
			name: "Sub-module imports return top-level",
			code: `
import mylib.submodule.deep
from another.lib.sub import func
`,
			expected: []string{"another", "mylib"},
		},
		{
			name: "Ignore single-line comments",
			code: `
# import fake_lib
import real_lib
# from fake_lib2 import something
from real_lib2 import func
`,
			expected: []string{"real_lib", "real_lib2"},
		},
		{
			name: "Ignore multi-line string comments",
			code: `
"""
import fake_lib
from another_fake import something
"""
import real_lib
'''
import yet_another_fake
'''
from real_lib2 import func
`,
			expected: []string{"real_lib", "real_lib2"},
		},
		{
			name: "No imports",
			code: `def transform_event(event):
    return event`,
			expected: []string{},
		},
		{
			name: "Stdlib modules pass through (filtering is provider responsibility)",
			code: `
import json
import hashlib
from datetime import datetime
import mylib
from base64 import b64encode
import another_lib
`,
			expected: []string{"another_lib", "base64", "datetime", "hashlib", "json", "mylib"},
		},
		{
			name: "Deduplicate imports",
			code: `
import mylib
from mylib import func1
from mylib import func2
import mylib
`,
			expected: []string{"mylib"},
		},
		{
			name: "Complex real-world example",
			code: `
# Import external libraries
import pandas
from numpy import array
import requests

# Import standard library
import json
from datetime import datetime

# Transform function
def transform_event(event):
    data = pandas.DataFrame(event)
    response = requests.get("https://api.example.com")
    return data.to_dict()
`,
			expected: []string{"datetime", "json", "numpy", "pandas", "requests"},
		},
		{
			name: "Import with as alias",
			code: `
import mylib as ml
from another import func as f
`,
			expected: []string{"another", "mylib"},
		},
		{
			name: "Star imports",
			code: `
from mylib import *
import another
`,
			expected: []string{"another", "mylib"},
		},
		{
			name: "Multiple from imports from same module",
			code: `
from mylib import func1
from mylib import func2
from mylib.submodule import func3
`,
			expected: []string{"mylib"},
		},
		{
			name: "Constants and variables not included in imports",
			code: `
import json
import mylib

# Constants should not be extracted as imports
BASE_MODULE_WHITELIST = ["ast", "json", "datetime"]
HIDDEN_IMPORTABLES = {"config", "utils"}
MY_CONSTANT = "some_value"

# Variables should not be extracted
some_var = "test"
another_var = {"key": "value"}

def transform(event):
    return mylib.process(event)
`,
			expected: []string{"json", "mylib"},
		},
		{
			name: "Function and class names not included in imports",
			code: `
from mylib import transform
import another_lib

# These should not be extracted as imports
def my_function():
    pass

class MyClass:
    pass

MODULE_WHITELIST = set()
validate_imports = True
`,
			expected: []string{"another_lib", "mylib"},
		},
		{
			name: "Multi-line import with parentheses",
			code: `
from mylib import (
    func1,
    func2,
    func3
)
import another_lib
`,
			expected: []string{"another_lib", "mylib"},
		},
		{
			name: "Multi-line import with backslash",
			code: `
from mylib import func1, \
    func2, \
    func3
import another_lib
`,
			expected: []string{"another_lib", "mylib"},
		},
		{
			name: "Complex multi-line imports",
			code: `
from mylib import (
    ClassA,
    ClassB,
    function_c,
)
from another import (item1, item2)
import third_lib
`,
			expected: []string{"another", "mylib", "third_lib"},
		},
		{
			name: "Import inside inline comment should be ignored",
			code: `
import real_lib  # import fake_lib
from real_lib2 import func  # from fake import something
`,
			expected: []string{"real_lib", "real_lib2"},
		},
		{
			name: "String containing import should be ignored",
			code: `
import real_lib
code = "import fake_lib"
template = 'from fake import something'
`,
			expected: []string{"real_lib"},
		},
		{
			name:     "Semicolon-separated imports",
			code:     `import json; import mylib`,
			expected: []string{"json", "mylib"},
		},
		{
			name:     "Semicolon inside string should not split",
			code:     `var a = "abc; import parse"`,
			expected: []string{},
		},
		{
			name:     "Import followed by statement with semicolon",
			code:     `import mylib; x = 1`,
			expected: []string{"mylib"},
		},
		{
			name:     "Multiple semicolon-separated imports",
			code:     `import lib1; import lib2; from lib3 import func`,
			expected: []string{"lib1", "lib2", "lib3"},
		},
		{
			name: "Prefixed string literals do not produce false imports",
			code: `
import real_lib
x = r"import fake_lib"
y = f"from {module} import something"
z = b"import bytes_fake"
`,
			expected: []string{"real_lib"},
		},
		{
			name: "Function-scope import",
			code: `
def transform(event):
    import mylib
    return mylib.process(event)
`,
			expected: []string{"mylib"},
		},
		{
			name: "One-line suite import",
			code: `
if True: import mylib
`,
			expected: []string{"mylib"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.ExtractImports(tt.code)
			require.NoError(t, err)
			assert.ElementsMatch(t, tt.expected, result)
		})
	}
}

func TestPythonParser_ExtractImports_Errors(t *testing.T) {
	parser := &PythonParser{}

	tests := []struct {
		name        string
		code        string
		expectedErr string
	}{
		{
			name:        "Relative import with single dot",
			code:        `from . import util`,
			expectedErr: "relative imports (from . or from ..) are not supported",
		},
		{
			name:        "Relative import with double dot",
			code:        `from .. import config`,
			expectedErr: "relative imports (from . or from ..) are not supported",
		},
		{
			name:        "Relative import with submodule",
			code:        `from ..pkg import mod`,
			expectedErr: "relative imports (from . or from ..) are not supported",
		},
		{
			name:        "Relative import with .module",
			code:        `from .utils import helper`,
			expectedErr: "relative imports (from . or from ..) are not supported",
		},
		{
			name: "Mixed valid and relative imports",
			code: `
import mylib
from . import util
`,
			expectedErr: "relative imports (from . or from ..) are not supported",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.ExtractImports(tt.code)
			require.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}

// --- Scanner unit tests ---

func TestScanImportCandidates(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantText []string // expected candidate .text values
	}{
		{
			name:     "Simple import",
			input:    "import mylib",
			wantText: []string{"import mylib"},
		},
		{
			name:     "Import inside triple-double-quote string is skipped",
			input:    "\"\"\"\nimport fake\n\"\"\"\nimport real",
			wantText: []string{"import real"},
		},
		{
			name:     "Import inside triple-single-quote string is skipped",
			input:    "'''\nimport fake\n'''\nimport real",
			wantText: []string{"import real"},
		},
		{
			name:     "Import in single-line comment is skipped",
			input:    "# import fake\nimport real",
			wantText: []string{"import real"},
		},
		{
			name:     "Import in double-quoted string is skipped",
			input:    `x = "import fake"` + "\nimport real",
			wantText: []string{"import real"},
		},
		{
			name:     "Import in single-quoted string is skipped",
			input:    `x = 'import fake'` + "\nimport real",
			wantText: []string{"import real"},
		},
		{
			name:     "Semicolon splits statements",
			input:    "import lib1; import lib2",
			wantText: []string{"import lib1", "import lib2"},
		},
		{
			name:     "Backslash continuation joined",
			input:    "from mylib import func1, \\\n    func2",
			wantText: []string{"from mylib import func1,      func2"},
		},
		{
			name:     "Parentheses continuation joined",
			input:    "from mylib import (\n    func1,\n    func2\n)",
			wantText: []string{"from mylib import (     func1,     func2 )"},
		},
		{
			name:     "Prefixed string r-string skipped",
			input:    `x = r"import fake"` + "\nimport real",
			wantText: []string{"import real"},
		},
		{
			name:     "Prefixed string f-string skipped",
			input:    "x = f\"import fake\"\nimport real",
			wantText: []string{"import real"},
		},
		{
			name:     "Non-import lines not emitted",
			input:    "x = 1\ny = 2\nz = x + y",
			wantText: []string{},
		},
		// Prefixed triple-quoted strings (Python 3.11)
		{
			// Single prefix + triple: r/f/b + """ or '''
			name:     "r-prefixed triple-quote string skipped",
			input:    "x = r\"\"\"\nimport fake\n\"\"\"\nimport real",
			wantText: []string{"import real"},
		},
		{
			// Valid double prefix + triple: rb, br, rf, fr
			name:     "rb-prefixed triple-quote string skipped",
			input:    "x = rb\"\"\"\nimport fake\n\"\"\"\nimport real",
			wantText: []string{"import real"},
		},
		{
			// Invalid double prefix (bu) must not match — "buffer" is an identifier
			name:     "identifier starting with prefix chars not mistaken for string",
			input:    "buffer = 42\nimport real",
			wantText: []string{"import real"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := scanImportCandidates(tt.input)
			texts := make([]string, len(got))
			for i, c := range got {
				texts[i] = c.text
			}
			assert.Equal(t, tt.wantText, texts)
		})
	}
}

// --- Grammar unit tests ---

func TestParseImportStatement(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantFrom    bool
		wantSimple  bool
		expectedErr bool
	}{
		{
			name:       "Simple import",
			input:      "import mylib",
			wantSimple: true,
		},
		{
			name:       "Simple import with alias",
			input:      "import mylib as ml",
			wantSimple: true,
		},
		{
			name:       "Multiple simple imports",
			input:      "import lib1, lib2, lib3",
			wantSimple: true,
		},
		{
			name:     "From import",
			input:    "from mylib import func",
			wantFrom: true,
		},
		{
			name:     "From import with submodule",
			input:    "from mylib.sub import func",
			wantFrom: true,
		},
		{
			name:     "From import star",
			input:    "from mylib import *",
			wantFrom: true,
		},
		{
			name:     "From import parenthesized",
			input:    "from mylib import ( func1, func2 )",
			wantFrom: true,
		},
		{
			name:        "Invalid statement",
			input:       "def foo(): pass",
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stmt, err := importParser.ParseString("", tt.input)
			if tt.expectedErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantFrom, stmt.From != nil)
			assert.Equal(t, tt.wantSimple, stmt.Simple != nil)
		})
	}
}
