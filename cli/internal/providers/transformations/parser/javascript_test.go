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
			name: "CommonJS require",
			code: `const myLib = require('myLib');`,
			expected: []string{"myLib"},
		},
		{
			name: "CommonJS require with spaces",
			code: `const lib = require(  'myLib'  );`,
			expected: []string{"myLib"},
		},
		{
			name: "Multiple imports mixed",
			code: `
				import lib1 from 'lib1';
				import { func } from 'lib2';
				const lib3 = require('lib3');
			`,
			expected: []string{"lib1", "lib2", "lib3"},
		},
		{
			name: "Filter relative imports",
			code: `
				import myLib from 'myLib';
				import util from './util';
				import config from '../config';
				import abs from '/absolute/path';
			`,
			expected: []string{"myLib"},
		},
		{
			name: "Deduplicate imports",
			code: `
				import myLib from 'myLib';
				const lib = require('myLib');
			`,
			expected: []string{"myLib"},
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
			name: "Complex real-world example",
			code: `
				// Import validation library
				import {
					validateEmail,
					validatePhone
				} from 'validationHelpers';

				import * as utils from 'commonUtils';

				const crypto = require('cryptoLib');

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
