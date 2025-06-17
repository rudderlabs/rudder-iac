package jsonpath

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tidwall/gjson"
)

// ProcessorResult holds the result of JSONPath processing
type ProcessorResult struct {
	Value interface{}
	Error error
}

// Processor handles JSONPath operations on schema data
type Processor struct {
	jsonPath   string
	skipFailed bool
}

// NewProcessor creates a new JSONPath processor
func NewProcessor(jsonPath string, skipFailed bool) *Processor {
	return &Processor{
		jsonPath:   jsonPath,
		skipFailed: skipFailed,
	}
}

// IsRootPath checks if the JSONPath expression refers to the root
func (p *Processor) IsRootPath() bool {
	return p.jsonPath == "" || p.jsonPath == "$" || p.jsonPath == "$."
}

// normalizeJSONPath converts standard JSONPath syntax to gjson syntax
func (p *Processor) normalizeJSONPath() string {
	path := p.jsonPath

	// Remove leading $ if present (gjson doesn't use $)
	if strings.HasPrefix(path, "$.") {
		path = strings.TrimPrefix(path, "$.")
	} else if path == "$" {
		path = ""
	}

	return path
}

// parseGjsonResult converts a gjson.Result to interface{}
func parseGjsonResult(result gjson.Result) (interface{}, error) {
	// Handle different result types
	switch result.Type {
	case gjson.String:
		return result.String(), nil
	case gjson.Number:
		return result.Num, nil
	case gjson.True, gjson.False:
		return result.Bool(), nil
	case gjson.JSON:
		// For objects and arrays, unmarshal the JSON
		var value interface{}
		if err := json.Unmarshal([]byte(result.Raw), &value); err != nil {
			return nil, err
		}
		return value, nil
	case gjson.Null:
		return nil, nil
	default:
		return result.Value(), nil
	}
}

// ProcessSchema applies JSONPath extraction to a schema
func (p *Processor) ProcessSchema(schema map[string]interface{}) ProcessorResult {
	if p.IsRootPath() {
		// Root path - return original schema
		return ProcessorResult{Value: schema, Error: nil}
	}

	// Convert schema to JSON for gjson processing
	jsonBytes, err := json.Marshal(schema)
	if err != nil {
		return ProcessorResult{Value: nil, Error: fmt.Errorf("failed to marshal schema to JSON: %w", err)}
	}

	// Normalize the JSONPath for gjson
	normalizedPath := p.normalizeJSONPath()

	// Apply JSONPath
	result := gjson.GetBytes(jsonBytes, normalizedPath)

	if !result.Exists() {
		return ProcessorResult{Value: nil, Error: fmt.Errorf("JSONPath '%s' returned no results", p.jsonPath)}
	}

	// Convert result back to interface{}
	value, err := parseGjsonResult(result)
	if err != nil {
		return ProcessorResult{Value: nil, Error: fmt.Errorf("failed to parse JSONPath result: %w", err)}
	}

	return ProcessorResult{Value: value, Error: nil}
}

// ShouldSkipOnError returns true if schemas should be skipped when JSONPath fails
func (p *Processor) ShouldSkipOnError() bool {
	return p.skipFailed
}
