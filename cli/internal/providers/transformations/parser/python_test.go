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
			name: "All base whitelist modules filtered",
			code: `
import ast
import base64
import collections
import datetime
import hashlib
import json
import math
import random
import re
import string
import time
import uuid
import copy
import typing
`,
			expected: []string{},
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
			name: "Semicolon-separated imports",
			code: `import json; import mylib`,
			expected: []string{"mylib"},
		},
		{
			name: "Import followed by statement with semicolon",
			code: `import mylib; x = 1`,
			expected: []string{"mylib"},
		},
		{
			name: "Multiple semicolon-separated imports",
			code: `import lib1; import lib2; from lib3 import func`,
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
