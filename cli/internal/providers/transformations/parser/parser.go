package parser

import (
	"fmt"
	"strings"
)

// Parser extracts library imports from transformation code and validates syntax
type Parser interface {
	ExtractImports(code string) ([]string, error)
	ValidateSyntax(code string) error
}

// NewParser creates a parser for the given language
func NewParser(language string) (Parser, error) {
	switch strings.ToLower(language) {
	case "javascript":
		return &JavaScriptParser{}, nil
	case "python":
		return &PythonParser{}, nil
	default:
		return nil, fmt.Errorf("unsupported language: %s", language)
	}
}