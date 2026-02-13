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
			name: "Ignore comments",
			code: `
# import fake_lib
import real_lib
"""
import another_fake
"""
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
			name: "Invalid syntax - missing colon",
			code: `def func()
    pass`,
			expectedErr: "expected ':'",
		},
		{
			name:        "Invalid syntax - incomplete statement",
			code:        `import`,
			expectedErr: "invalid syntax",
		},
		{
			name: "Invalid syntax - bad indentation",
			code: `def func():
pass`,
			expectedErr: "expected an indented block",
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

func TestPythonParser_ValidateSyntax(t *testing.T) {
	parser := &PythonParser{}

	tests := []struct {
		name    string
		code    string
		wantErr bool
	}{
		{
			name:    "Valid Python - simple statement",
			code:    `x = 1`,
			wantErr: false,
		},
		{
			name: "Valid Python - function definition",
			code: `def transform_event(event):
    return event`,
			wantErr: false,
		},
		{
			name: "Valid Python - complex code",
			code: `
import json

def transform_event(event):
    """Transform event data"""
    data = json.loads(event)
    return data
`,
			wantErr: false,
		},
		{
			name: "Valid Python - class definition",
			code: `class MyClass:
    def __init__(self):
        self.value = 42`,
			wantErr: false,
		},
		{
			name:    "Valid Python - list comprehension",
			code:    `result = [x * 2 for x in range(10)]`,
			wantErr: false,
		},
		{
			name:    "Valid Python - dictionary",
			code:    `data = {"key": "value", "number": 123}`,
			wantErr: false,
		},
		{
			name: "Invalid Python - missing colon",
			code: `def func()
    pass`,
			wantErr: true,
		},
		{
			name: "Invalid Python - bad indentation",
			code: `def func():
pass`,
			wantErr: true,
		},
		{
			name:    "Invalid Python - incomplete statement",
			code:    `x = `,
			wantErr: true,
		},
		{
			name:    "Invalid Python - mismatched brackets",
			code:    `data = [1, 2, 3`,
			wantErr: true,
		},
		{
			name: "Invalid Python - invalid syntax",
			code: `def func():
    return
  invalid_indent`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := parser.ValidateSyntax(tt.code)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "syntax error")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPythonParser_isPythonBaseWhitelist(t *testing.T) {
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
			result := isPythonBaseWhitelist(tt.module)
			assert.Equal(t, tt.expected, result)
		})
	}
}
