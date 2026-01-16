package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJavaScriptParser_ExtractImports(t *testing.T) {
	parser := &JavaScriptParser{}

	tests := []struct {
		name     string
		code     string
		expected []string
	}{
		{
			name: "ES6 default import",
			code: `import myLib from 'myLib';`,
			expected: []string{"myLib"},
		},
		{
			name: "ES6 named import",
			code: `import { func1, func2 } from 'myLib';`,
			expected: []string{"myLib"},
		},
		{
			name: "ES6 multi-line import",
			code: `import {
				add,
				subtract,
				multiply
			} from 'mathLib';`,
			expected: []string{"mathLib"},
		},
		{
			name: "ES6 multi-line with spaces",
			code: `import {
				add
			} from "someLib";`,
			expected: []string{"someLib"},
		},
		{
			name: "Multiple ES6 imports",
			code: `
				import lib1 from 'lib1';
				import { func } from 'lib2';
				import * as lib3 from 'lib3';
			`,
			expected: []string{"lib1", "lib2", "lib3"},
		},
		{
			name: "Ignore single-line comments",
			code: `
				// import fake from 'fake';
				import real from 'real';
			`,
			expected: []string{"real"},
		},
		{
			name: "Ignore multi-line comments",
			code: `
				/*
				import fake from 'fake';
				*/
				import real from 'real';
			`,
			expected: []string{"real"},
		},
		{
			name: "Multi-line import with comments",
			code: `
				import {
					// function add
					add,
					/* function subtract */
					subtract
				} from 'mathLib';
			`,
			expected: []string{"mathLib"},
		},
		{
			name: "No imports",
			code: `export function transformEvent(event) { return event; }`,
			expected: []string{},
		},
		{
			name: "Import with newlines and tabs",
			code: "import {\n\tadd,\n\tsubtract\n} from 'mathLib';",
			expected: []string{"mathLib"},
		},
		{
			name: "Namespace import",
			code: `import * as lib from 'myLib';`,
			expected: []string{"myLib"},
		},
		{
			name: "Import without from (side effects)",
			code: `import 'polyfill';`,
			expected: []string{"polyfill"},
		},
		{
			name: "Filter RudderStack built-in libraries",
			code: `
				import { sha1 } from '@rs/hash/v1';
				import { formatDate } from '@rs/utils/v2';
				import myLib from 'myLib';
			`,
			expected: []string{"myLib"},
		},
		{
			name: "Complex example without require",
			code: `
				// Import validation library
				import {
					validateEmail, // validate email
					validatePhone
				} from 'validationHelpers';
				import * as utils from 'commonUtils';
				import * as crypto from 'cryptoLib';
				// import { add } from 'mathLib';
				// Transform function
				export function transformEvent(event) {
					const email = utils.sanitize(event.email);
					return crypto.hash(email);
				}
			`,
			expected: []string{"validationHelpers", "commonUtils", "cryptoLib"},
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

func TestJavaScriptParser_ExtractImports_Errors(t *testing.T) {
	parser := &JavaScriptParser{}

	tests := []struct {
		name        string
		code        string
		expectedErr string
	}{
		{
			name: "CommonJS require not supported",
			code: `const myLib = require('myLib');`,
			expectedErr: "require() syntax is not supported",
		},
		{
			name: "CommonJS require with spaces",
			code: `const lib = require(  'myLib'  );`,
			expectedErr: "require() syntax is not supported",
		},
		{
			name: "Mixed imports and requires",
			code: `
				import lib1 from 'lib1';
				const lib3 = require('lib3');
			`,
			expectedErr: "require() syntax is not supported",
		},
		{
			name: "Relative import with ./",
			code: `import util from './util';`,
			expectedErr: "relative imports (./file, ../file) and absolute imports (/path) are not supported",
		},
		{
			name: "Relative import with ../",
			code: `import config from '../config';`,
			expectedErr: "relative imports (./file, ../file) and absolute imports (/path) are not supported",
		},
		{
			name: "Absolute import",
			code: `import abs from '/absolute/path';`,
			expectedErr: "relative imports (./file, ../file) and absolute imports (/path) are not supported",
		},
		{
			name: "Mixed valid and relative imports",
			code: `
				import myLib from 'myLib';
				import util from './util';
			`,
			expectedErr: "relative imports (./file, ../file) and absolute imports (/path) are not supported",
		},
		{
			name: "Multiple relative imports",
			code: `
				import util from './util';
				import config from '../config';
			`,
			expectedErr: "relative imports (./file, ../file) and absolute imports (/path) are not supported",
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

func TestJavaScriptParser_ValidateSyntax(t *testing.T) {
	parser := &JavaScriptParser{}

	tests := []struct {
		name    string
		code    string
		wantErr bool
	}{
		{
			name:    "Valid JavaScript",
			code:    `const x = 1; console.log(x);`,
			wantErr: false,
		},
		{
			name:    "Valid ES6 code",
			code:    `const add = (a, b) => a + b; export default add;`,
			wantErr: false,
		},
		{
			name:    "Invalid JavaScript - missing bracket",
			code:    `function test() { console.log("test");`,
			wantErr: true,
		},
		{
			name:    "Invalid JavaScript - syntax error",
			code:    `const x = ;`,
			wantErr: true,
		},
		{
			name:    "Valid complex code with imports",
			code:    `import lib from 'myLib'; const result = lib.process();`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := parser.ValidateSyntax(tt.code)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}