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
			expected: []string{"mylib", "another"},
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
			name: "Filter Python base whitelist modules",
			code: `
import json
import hashlib
from datetime import datetime
import mylib
from base64 import b64encode
import another_lib
`,
			expected: []string{"mylib", "another_lib"},
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

# Import standard library (should be filtered)
import json
from datetime import datetime

# Transform function
def transform_event(event):
    data = pandas.DataFrame(event)
    response = requests.get("https://api.example.com")
    return data.to_dict()
`,
			expected: []string{"pandas", "numpy"},
		},
		{
			name: "Import with as alias",
			code: `
import mylib as ml
from another import func as f
`,
			expected: []string{"mylib", "another"},
		},
		{
			name: "Star imports",
			code: `
from mylib import *
import another
`,
			expected: []string{"mylib", "another"},
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
			expected: []string{"mylib"},
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
			expected: []string{"mylib", "another_lib"},
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
			expected: []string{"mylib", "another_lib"},
		},
		{
			name: "Multi-line import with backslash",
			code: `
from mylib import func1, \
    func2, \
    func3
import another_lib
`,
			expected: []string{"mylib", "another_lib"},
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
			expected: []string{"mylib", "another", "third_lib"},
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
			expected: []string{"mylib"},
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

func TestPythonParser_isPythonBuiltinModule(t *testing.T) {
	tests := []struct {
		module   string
		expected bool
	}{
		{"ast", true},
		{"json", true},
		{"datetime", true},
		{"hashlib", true},
		{"requests", true},
		{"mylib", false},
		{"pandas", false},
		{"numpy", false},
		{"sklearn", false},
	}

	for _, tt := range tests {
		t.Run(tt.module, func(t *testing.T) {
			result := isPythonBuiltinModule(tt.module)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeCode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Remove single-line comment",
			input:    "import mylib  # this is a comment",
			expected: "import mylib  ",
		},
		{
			name:     "Remove comment on its own line",
			input:    "# comment\nimport mylib",
			expected: "\nimport mylib",
		},
		{
			name: "Remove triple double-quote docstring",
			input: `"""
This is a docstring
"""
import mylib`,
			expected: "\nimport mylib",
		},
		{
			name: "Remove triple single-quote docstring",
			input: `'''
This is a docstring
'''
import mylib`,
			expected: "\nimport mylib",
		},
		{
			name: "Remove inline triple double-quote",
			input: `x = """some string"""
import mylib`,
			expected: "x = \nimport mylib",
		},
		{
			name: "Remove inline triple single-quote",
			input: `x = '''some string'''
import mylib`,
			expected: "x = \nimport mylib",
		},
		{
			name: "Multiple comments removed",
			input: `# first comment
import lib1  # inline comment
# another comment
import lib2`,
			expected: "\nimport lib1  \n\nimport lib2",
		},
		{
			name: "Docstring with import inside should be removed",
			input: `"""
import fake_lib
from another import something
"""
import real_lib`,
			expected: "\nimport real_lib",
		},
		{
			name:     "No comments - unchanged",
			input:    "import mylib\nfrom another import func",
			expected: "import mylib\nfrom another import func",
		},
		{
			name:     "Double-quoted string content replaced",
			input:    `x = "import fake; something"`,
			expected: `x = ""`,
		},
		{
			name:     "Single-quoted string content replaced",
			input:    `x = 'import fake; something'`,
			expected: `x = ''`,
		},
		{
			name:     "Escaped quotes in string preserved",
			input:    `x = "hello \"world\""`,
			expected: `x = ""`,
		},
		{
			name: "Multi-line string variable assignment",
			input: `x = """
import fake_lib
from another import something
"""
import real_lib`,
			expected: "x = \nimport real_lib",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeCode(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNormalizeMultilineImports(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Single line import unchanged",
			input:    "import mylib\n",
			expected: "import mylib\n",
		},
		{
			name:     "Backslash continuation joined",
			input:    "from mylib import func1, \\\n    func2, \\\n    func3\n",
			expected: "from mylib import func1,  func2,  func3\n",
		},
		{
			name:     "Parentheses continuation joined",
			input:    "from mylib import (\n    func1,\n    func2\n)\n",
			expected: "from mylib import (     func1,     func2 )\n",
		},
		{
			name:     "Mixed imports",
			input:    "import simple\nfrom mylib import (\n    a,\n    b\n)\nimport another\n",
			expected: "import simple\nfrom mylib import (     a,     b )\nimport another\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeMultilineImports(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseFromImport(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    string
		expectedErr string
	}{
		{
			name:     "Simple from import",
			input:    "from mylib import func",
			expected: "mylib",
		},
		{
			name:     "From import with submodule",
			input:    "from mylib.sub import func",
			expected: "mylib.sub",
		},
		{
			name:     "From import with multiple items",
			input:    "from mylib import func1, func2",
			expected: "mylib",
		},
		{
			name:     "From import with star",
			input:    "from mylib import *",
			expected: "mylib",
		},
		{
			name:     "From import with parentheses",
			input:    "from mylib import (func1, func2)",
			expected: "mylib",
		},
		{
			name:     "Not a from import",
			input:    "import mylib",
			expected: "",
		},
		{
			name:        "Relative import with dot",
			input:       "from . import util",
			expectedErr: "relative imports",
		},
		{
			name:        "Relative import with double dot",
			input:       "from ..pkg import mod",
			expectedErr: "relative imports",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseFromImport(tt.input)
			if tt.expectedErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestParseSimpleImport(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    []string
		expectedErr string
	}{
		{
			name:     "Single import",
			input:    "import mylib",
			expected: []string{"mylib"},
		},
		{
			name:     "Multiple imports",
			input:    "import lib1, lib2, lib3",
			expected: []string{"lib1", "lib2", "lib3"},
		},
		{
			name:     "Import with alias",
			input:    "import mylib as ml",
			expected: []string{"mylib"},
		},
		{
			name:     "Multiple imports with aliases",
			input:    "import lib1 as l1, lib2 as l2",
			expected: []string{"lib1", "lib2"},
		},
		{
			name:     "Import with submodule",
			input:    "import mylib.submodule",
			expected: []string{"mylib.submodule"},
		},
		{
			name:     "Not an import statement",
			input:    "from mylib import func",
			expected: nil,
		},
		{
			name:        "Relative import",
			input:       "import .mylib",
			expectedErr: "relative imports",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseSimpleImport(tt.input)
			if tt.expectedErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
