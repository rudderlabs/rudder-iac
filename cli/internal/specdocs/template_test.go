package specdocs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequiredByDefaults(t *testing.T) {
	t.Parallel()

	fields := map[string]Field{
		"api_version": {Key: "api_version", Type: "string", Default: "v2"},
		"enabled":     {Key: "enabled", Type: "boolean", Default: false},
		"region":      {Key: "region", Type: "string", Required: true},
		"block.mode":  {Key: "block.mode", Type: "string", Default: "fast", DefaultRequiresParent: true},
	}

	cases := []struct {
		name  string
		field Field
		want  bool
	}{
		{
			name:  "requiredIf met by a default",
			field: Field{Key: "id_type", RequiredIf: []Condition{{Key: "api_version", Value: "v2"}}},
			want:  true,
		},
		{
			name:  "requiredIf a default does not meet",
			field: Field{Key: "id_type", RequiredIf: []Condition{{Key: "api_version", Value: "v1"}}},
		},
		{
			name: "requiredIf with a condition key the caller fills",
			field: Field{Key: "id_type", RequiredIf: []Condition{
				{Key: "api_version", Value: "v2"},
				{Key: "region", Value: "eu"},
			}},
		},
		{
			name:  "requiredIf on a default that needs its parent",
			field: Field{Key: "id_type", RequiredIf: []Condition{{Key: "block.mode", Value: "fast"}}},
		},
		{
			name:  "requiredUnless no default meets",
			field: Field{Key: "token", RequiredUnless: []Condition{{Key: "enabled", Value: true}}},
			want:  true,
		},
		{
			name:  "requiredUnless a default meets",
			field: Field{Key: "token", RequiredUnless: []Condition{{Key: "enabled", Value: false}}},
		},
		{
			name: "requiredUnless with a condition key the caller fills",
			field: Field{Key: "token", RequiredUnless: []Condition{
				{Key: "enabled", Value: true},
				{Key: "region", Value: "eu"},
			}},
		},
		{
			name:  "a nested key",
			field: Field{Key: "auth.token", RequiredIf: []Condition{{Key: "api_version", Value: "v2"}}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, requiredByDefaults(tc.field, fields))
		})
	}
}
