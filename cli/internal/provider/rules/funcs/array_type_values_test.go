package funcs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatchesAllowedType(t *testing.T) {
	t.Parallel()

	allowedStringBoolInteger := []string{"string", "bool", "integer"}

	tests := []struct {
		name    string
		value   any
		allowed []string
		want    bool
	}{
		// Valid types
		{name: "string value", value: "hello", allowed: allowedStringBoolInteger, want: true},
		{name: "empty string", value: "", allowed: allowedStringBoolInteger, want: true},
		{name: "bool true", value: true, allowed: allowedStringBoolInteger, want: true},
		{name: "bool false", value: false, allowed: allowedStringBoolInteger, want: true},
		{name: "whole number float64", value: float64(42), allowed: allowedStringBoolInteger, want: true},
		{name: "zero float64", value: float64(0), allowed: allowedStringBoolInteger, want: true},
		{name: "negative whole number", value: float64(-5), allowed: allowedStringBoolInteger, want: true},

		// Invalid types
		{name: "float with decimals", value: 3.14, allowed: allowedStringBoolInteger, want: false},
		{name: "nil value", value: nil, allowed: allowedStringBoolInteger, want: false},
		{name: "slice value", value: []any{"nested"}, allowed: allowedStringBoolInteger, want: false},
		{name: "map value", value: map[string]any{"key": "val"}, allowed: allowedStringBoolInteger, want: false},
		{name: "int value (not float64)", value: 42, allowed: allowedStringBoolInteger, want: false},

		// Subset of allowed types
		{name: "string only allows string", value: "hello", allowed: []string{"string"}, want: true},
		{name: "string only rejects bool", value: true, allowed: []string{"string"}, want: false},
		{name: "bool only allows bool", value: true, allowed: []string{"bool"}, want: true},
		{name: "bool only rejects string", value: "hello", allowed: []string{"bool"}, want: false},
		{name: "integer only allows whole float64", value: float64(1), allowed: []string{"integer"}, want: true},
		{name: "integer only rejects decimal", value: 1.5, allowed: []string{"integer"}, want: false},

		// Empty allowed types
		{name: "empty allowed rejects string", value: "hello", allowed: []string{}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesAllowedType(tt.value, tt.allowed)
			assert.Equal(t, tt.want, got)
		})
	}
}
