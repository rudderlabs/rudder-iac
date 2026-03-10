package property

import (
	"strings"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func assertHasValidationError(t *testing.T, results []rules.ValidationResult, reference, messageContains string) {
	t.Helper()

	for _, result := range results {
		if result.Reference == reference && strings.Contains(result.Message, messageContains) {
			return
		}
	}

	t.Fatalf("expected validation error at %q containing %q, got: %+v", reference, messageContains, results)
}

func TestPropertySpecSyntaxValidRuleV1_ValidSpecs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		spec localcatalog.PropertySpecV1
	}{
		{
			name: "minimal valid property",
			spec: localcatalog.PropertySpecV1{
				Properties: []localcatalog.PropertyV1{
					{
						LocalID: "user_id",
						Name:    "User ID",
						Type:    "string",
					},
				},
			},
		},
		{
			name: "uses types and item_types arrays",
			spec: localcatalog.PropertySpecV1{
				Properties: []localcatalog.PropertyV1{
					{
						LocalID:   "metadata",
						Name:      "Metadata",
						Types:     []string{"object", "null"},
						ItemTypes: []string{"string", "number"},
					},
				},
			},
		},
		{
			name: "uses custom type references in type and item_type",
			spec: localcatalog.PropertySpecV1{
				Properties: []localcatalog.PropertyV1{
					{
						LocalID: "address",
						Name:    "Address",
						Type:    "#custom-type:Address",
					},
					{
						LocalID:  "addresses",
						Name:     "Addresses",
						Type:     "array",
						ItemType: "#custom-type:Address",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			results := validatePropertySpecV1(localcatalog.KindProperties, specs.SpecVersionV1, nil, tt.spec)
			assert.Empty(t, results)
		})
	}
}

func TestPropertySpecSyntaxValidRuleV1_TagValidations(t *testing.T) {
	t.Parallel()

	t.Run("required fields", func(t *testing.T) {
		t.Parallel()

		spec := localcatalog.PropertySpecV1{
			Properties: []localcatalog.PropertyV1{
				{},
			},
		}

		results := validatePropertySpecV1(localcatalog.KindProperties, specs.SpecVersionV1, nil, spec)
		require.Len(t, results, 2)
		assertHasValidationError(t, results, "/properties/0/id", "'id' is required")
		assertHasValidationError(t, results, "/properties/0/name", "'name' is required")
	})

	t.Run("type and types are mutually exclusive", func(t *testing.T) {
		t.Parallel()

		spec := localcatalog.PropertySpecV1{
			Properties: []localcatalog.PropertyV1{
				{
					LocalID: "mixed",
					Name:    "Mixed",
					Type:    "string",
					Types:   []string{"number"},
				},
			},
		}

		results := validatePropertySpecV1(localcatalog.KindProperties, specs.SpecVersionV1, nil, spec)
		require.Len(t, results, 2)
		assertHasValidationError(t, results, "/properties/0/type", "'type' and 'types' cannot be specified together")
		assertHasValidationError(t, results, "/properties/0/types", "'types' and 'type' cannot be specified together")
	})

	t.Run("item_type and item_types are mutually exclusive", func(t *testing.T) {
		t.Parallel()

		spec := localcatalog.PropertySpecV1{
			Properties: []localcatalog.PropertyV1{
				{
					LocalID:   "items",
					Name:      "Items",
					Type:      "array",
					ItemType:  "string",
					ItemTypes: []string{"number"},
				},
			},
		}

		results := validatePropertySpecV1(localcatalog.KindProperties, specs.SpecVersionV1, nil, spec)
		require.Len(t, results, 2)
		assertHasValidationError(t, results, "/properties/0/item_type", "'item_type' and 'item_types' cannot be specified together")
		assertHasValidationError(t, results, "/properties/0/item_types", "'item_types' and 'item_type' cannot be specified together")
	})

	t.Run("types and item_types only allow primitive values", func(t *testing.T) {
		t.Parallel()

		spec := localcatalog.PropertySpecV1{
			Properties: []localcatalog.PropertyV1{
				{
					LocalID:   "details",
					Name:      "Details",
					Types:     []string{"string", "#custom-type:Address"},
					ItemTypes: []string{"integer", "not-valid"},
				},
			},
		}

		results := validatePropertySpecV1(localcatalog.KindProperties, specs.SpecVersionV1, nil, spec)
		require.Len(t, results, 2)
		assertHasValidationError(t, results, "/properties/0/types/1", "must be one of")
		assertHasValidationError(t, results, "/properties/0/item_types/1", "must be one of")
	})
}

func TestPropertySpecSyntaxValidRuleV1_CustomValidations(t *testing.T) {
	t.Parallel()

	t.Run("name cannot have leading or trailing whitespace", func(t *testing.T) {
		t.Parallel()

		spec := localcatalog.PropertySpecV1{
			Properties: []localcatalog.PropertyV1{
				{
					LocalID: "email",
					Name:    " Email ",
					Type:    "string",
				},
			},
		}

		results := validatePropertySpecV1(localcatalog.KindProperties, specs.SpecVersionV1, nil, spec)
		require.Len(t, results, 1)
		assertHasValidationError(t, results, "/properties/0/name", "must not have leading or trailing whitespace")
	})

	t.Run("types cannot contain duplicates", func(t *testing.T) {
		t.Parallel()

		spec := localcatalog.PropertySpecV1{
			Properties: []localcatalog.PropertyV1{
				{
					LocalID: "status",
					Name:    "Status",
					Types:   []string{"string", "string"},
				},
			},
		}

		results := validatePropertySpecV1(localcatalog.KindProperties, specs.SpecVersionV1, nil, spec)
		require.Len(t, results, 1)
		assertHasValidationError(t, results, "/properties/0/types", "must be unique one of")
	})

	t.Run("item_types cannot contain duplicates", func(t *testing.T) {
		t.Parallel()

		spec := localcatalog.PropertySpecV1{
			Properties: []localcatalog.PropertyV1{
				{
					LocalID:   "tags",
					Name:     "Tags",
					Type:     "array",
					ItemTypes: []string{"string", "number", "string"},
				},
			},
		}

		results := validatePropertySpecV1(localcatalog.KindProperties, specs.SpecVersionV1, nil, spec)
		require.Len(t, results, 1)
		assertHasValidationError(t, results, "/properties/0/item_types", "must be unique one of")
	})

	t.Run("invalid single type values are rejected", func(t *testing.T) {
		t.Parallel()

		spec := localcatalog.PropertySpecV1{
			Properties: []localcatalog.PropertyV1{
				{
					LocalID: "invalid",
					Name:    "Invalid",
					Type:    "string,integer",
				},
			},
		}

		results := validatePropertySpecV1(localcatalog.KindProperties, specs.SpecVersionV1, nil, spec)
		require.Len(t, results, 1)
		assertHasValidationError(t, results, "/properties/0/type", "or of pattern #custom-type:<id>")
	})

	t.Run("invalid custom type references are rejected for type and item_type", func(t *testing.T) {
		t.Parallel()

		spec := localcatalog.PropertySpecV1{
			Properties: []localcatalog.PropertyV1{
				{
					LocalID:  "invalid_ref",
					Name:     "Invalid Ref",
					Type:     "#custom-types:Address",
					ItemType: "#custom-types:Address",
				},
			},
		}

		results := validatePropertySpecV1(localcatalog.KindProperties, specs.SpecVersionV1, nil, spec)
		require.Len(t, results, 2)
		assertHasValidationError(t, results, "/properties/0/type", "or of pattern #custom-type:<id>")
		assertHasValidationError(t, results, "/properties/0/item_type", "or of pattern #custom-type:<id>")
	})
}
