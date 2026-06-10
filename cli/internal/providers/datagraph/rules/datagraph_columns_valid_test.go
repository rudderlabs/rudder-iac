package rules

import (
	"strings"
	"testing"

	"github.com/go-viper/mapstructure/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	dgModel "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// TestDataGraphSpecSyntaxValid_ColumnsValid covers positive cases for the
// optional `columns:` block on entity and event models.
func TestDataGraphSpecSyntaxValid_ColumnsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		spec dgModel.DataGraphSpec
	}{
		{
			name: "entity model with single column",
			spec: dgModel.DataGraphSpec{
				ID:        "my-dg",
				AccountID: "wh-123",
				Models: []dgModel.ModelSpec{
					{
						ID:          "user",
						DisplayName: "User",
						Type:        "entity",
						Table:       "db.schema.users",
						PrimaryID:   "id",
						Columns: []dgModel.ColumnMetadataYAML{
							{Name: "id", DisplayName: "User ID"},
						},
					},
				},
			},
		},
		{
			name: "entity model with multiple columns",
			spec: dgModel.DataGraphSpec{
				ID:        "my-dg",
				AccountID: "wh-123",
				Models: []dgModel.ModelSpec{
					{
						ID:          "user",
						DisplayName: "User",
						Type:        "entity",
						Table:       "db.schema.users",
						PrimaryID:   "id",
						Columns: []dgModel.ColumnMetadataYAML{
							{Name: "id", DisplayName: "User ID"},
							{Name: "email_address", DisplayName: "Email"},
						},
					},
				},
			},
		},
		{
			name: "event model with columns",
			spec: dgModel.DataGraphSpec{
				ID:        "my-dg",
				AccountID: "wh-123",
				Models: []dgModel.ModelSpec{
					{
						ID:          "purchase",
						DisplayName: "Purchase",
						Type:        "event",
						Table:       "db.schema.purchases",
						Timestamp:   "purchased_at",
						Columns: []dgModel.ColumnMetadataYAML{
							{Name: "amount_usd", DisplayName: "Amount (USD)"},
						},
					},
				},
			},
		},
		{
			name: "column with description only (no display_name)",
			spec: dgModel.DataGraphSpec{
				ID:        "my-dg",
				AccountID: "wh-123",
				Models: []dgModel.ModelSpec{
					{
						ID:          "user",
						DisplayName: "User",
						Type:        "entity",
						Table:       "db.schema.users",
						PrimaryID:   "id",
						Columns: []dgModel.ColumnMetadataYAML{
							{Name: "notes", Description: "Free-form notes"},
						},
					},
				},
			},
		},
		{
			name: "column with both display_name and description",
			spec: dgModel.DataGraphSpec{
				ID:        "my-dg",
				AccountID: "wh-123",
				Models: []dgModel.ModelSpec{
					{
						ID:          "user",
						DisplayName: "User",
						Type:        "entity",
						Table:       "db.schema.users",
						PrimaryID:   "id",
						Columns: []dgModel.ColumnMetadataYAML{
							{Name: "id", DisplayName: "User ID", Description: "Primary identifier"},
						},
					},
				},
			},
		},
		{
			name: "two description-only columns do not collide (no description uniqueness)",
			spec: dgModel.DataGraphSpec{
				ID:        "my-dg",
				AccountID: "wh-123",
				Models: []dgModel.ModelSpec{
					{
						ID:          "user",
						DisplayName: "User",
						Type:        "entity",
						Table:       "db.schema.users",
						PrimaryID:   "id",
						Columns: []dgModel.ColumnMetadataYAML{
							{Name: "notes", Description: "Same note"},
							{Name: "memo", Description: "Same note"},
						},
					},
				},
			},
		},
		{
			name: "model without columns is valid",
			spec: dgModel.DataGraphSpec{
				ID:        "my-dg",
				AccountID: "wh-123",
				Models: []dgModel.ModelSpec{
					{
						ID:          "user",
						DisplayName: "User",
						Type:        "entity",
						Table:       "db.schema.users",
						PrimaryID:   "id",
					},
				},
			},
		},
		{
			name: "model with empty columns slice is valid",
			spec: dgModel.DataGraphSpec{
				ID:        "my-dg",
				AccountID: "wh-123",
				Models: []dgModel.ModelSpec{
					{
						ID:          "user",
						DisplayName: "User",
						Type:        "entity",
						Table:       "db.schema.users",
						PrimaryID:   "id",
						Columns:     []dgModel.ColumnMetadataYAML{},
					},
				},
			},
		},
		{
			name: "display_name at maximum length (255 chars) is valid",
			spec: dgModel.DataGraphSpec{
				ID:        "my-dg",
				AccountID: "wh-123",
				Models: []dgModel.ModelSpec{
					{
						ID:          "user",
						DisplayName: "User",
						Type:        "entity",
						Table:       "db.schema.users",
						PrimaryID:   "id",
						Columns: []dgModel.ColumnMetadataYAML{
							{Name: "id", DisplayName: strings.Repeat("a", 255)},
						},
					},
				},
			},
		},
		{
			name: "name at maximum length (255 chars) is valid",
			spec: dgModel.DataGraphSpec{
				ID:        "my-dg",
				AccountID: "wh-123",
				Models: []dgModel.ModelSpec{
					{
						ID:          "user",
						DisplayName: "User",
						Type:        "entity",
						Table:       "db.schema.users",
						PrimaryID:   "id",
						Columns: []dgModel.ColumnMetadataYAML{
							{Name: strings.Repeat("a", 255), DisplayName: "Long Name"},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := validateDataGraphSpec("data-graph", specs.SpecVersionV1, nil, tt.spec)
			assert.Empty(t, results, "Valid spec should not produce errors, got: %v", results)
		})
	}
}

// TestDataGraphSpecSyntaxValid_ColumnsInvalid covers per-column validation
// failures: missing fields, oversize, control characters, trimming, and
// in-model duplicates by name or case-insensitive display_name.
func TestDataGraphSpecSyntaxValid_ColumnsInvalid(t *testing.T) {
	t.Parallel()

	baseModel := func(cols ...dgModel.ColumnMetadataYAML) dgModel.ModelSpec {
		return dgModel.ModelSpec{
			ID:          "user",
			DisplayName: "User",
			Type:        "entity",
			Table:       "db.schema.users",
			PrimaryID:   "id",
			Columns:     cols,
		}
	}

	tests := []struct {
		name         string
		spec         dgModel.DataGraphSpec
		expectedRefs []string
		// substrings each expected message must contain (order matches expectedRefs)
		expectedMsgSubstrings []string
	}{
		{
			name: "column missing name",
			spec: dgModel.DataGraphSpec{
				ID:        "my-dg",
				AccountID: "wh-123",
				Models: []dgModel.ModelSpec{
					baseModel(dgModel.ColumnMetadataYAML{DisplayName: "User ID"}),
				},
			},
			expectedRefs:          []string{"/models/0/columns/0/name"},
			expectedMsgSubstrings: []string{"'name' is required"},
		},
		{
			name: "column with neither display_name nor description",
			spec: dgModel.DataGraphSpec{
				ID:        "my-dg",
				AccountID: "wh-123",
				Models: []dgModel.ModelSpec{
					baseModel(dgModel.ColumnMetadataYAML{Name: "id"}),
				},
			},
			expectedRefs:          []string{"/models/0/columns/0"},
			expectedMsgSubstrings: []string{"at least one of 'display_name' or 'description'"},
		},
		{
			name: "column missing name (with description set)",
			spec: dgModel.DataGraphSpec{
				ID:        "my-dg",
				AccountID: "wh-123",
				Models: []dgModel.ModelSpec{
					baseModel(dgModel.ColumnMetadataYAML{Description: "some note"}),
				},
			},
			expectedRefs:          []string{"/models/0/columns/0/name"},
			expectedMsgSubstrings: []string{"'name' is required"},
		},
		{
			name: "column name leading whitespace",
			spec: dgModel.DataGraphSpec{
				ID:        "my-dg",
				AccountID: "wh-123",
				Models: []dgModel.ModelSpec{
					baseModel(dgModel.ColumnMetadataYAML{Name: " id", DisplayName: "User ID"}),
				},
			},
			expectedRefs:          []string{"/models/0/columns/0/name"},
			expectedMsgSubstrings: []string{"leading or trailing whitespace"},
		},
		{
			name: "column name trailing whitespace",
			spec: dgModel.DataGraphSpec{
				ID:        "my-dg",
				AccountID: "wh-123",
				Models: []dgModel.ModelSpec{
					baseModel(dgModel.ColumnMetadataYAML{Name: "id ", DisplayName: "User ID"}),
				},
			},
			expectedRefs:          []string{"/models/0/columns/0/name"},
			expectedMsgSubstrings: []string{"leading or trailing whitespace"},
		},
		{
			name: "column name too long",
			spec: dgModel.DataGraphSpec{
				ID:        "my-dg",
				AccountID: "wh-123",
				Models: []dgModel.ModelSpec{
					baseModel(dgModel.ColumnMetadataYAML{
						Name:        strings.Repeat("a", 256),
						DisplayName: "X",
					}),
				},
			},
			expectedRefs:          []string{"/models/0/columns/0/name"},
			expectedMsgSubstrings: []string{"length must be less than or equal to 255"},
		},
		{
			name: "column display_name too long",
			spec: dgModel.DataGraphSpec{
				ID:        "my-dg",
				AccountID: "wh-123",
				Models: []dgModel.ModelSpec{
					baseModel(dgModel.ColumnMetadataYAML{
						Name:        "id",
						DisplayName: strings.Repeat("a", 256),
					}),
				},
			},
			expectedRefs:          []string{"/models/0/columns/0/display_name"},
			expectedMsgSubstrings: []string{"length must be less than or equal to 255"},
		},
		{
			name: "column display_name with newline is rejected",
			spec: dgModel.DataGraphSpec{
				ID:        "my-dg",
				AccountID: "wh-123",
				Models: []dgModel.ModelSpec{
					baseModel(dgModel.ColumnMetadataYAML{
						Name:        "id",
						DisplayName: "User\nID",
					}),
				},
			},
			expectedRefs:          []string{"/models/0/columns/0/display_name"},
			expectedMsgSubstrings: []string{"control characters"},
		},
		{
			name: "column description with leading whitespace is rejected",
			spec: dgModel.DataGraphSpec{
				ID:        "my-dg",
				AccountID: "wh-123",
				Models: []dgModel.ModelSpec{
					baseModel(dgModel.ColumnMetadataYAML{
						Name:        "id",
						Description: " padded note",
					}),
				},
			},
			expectedRefs:          []string{"/models/0/columns/0/description"},
			expectedMsgSubstrings: []string{"leading or trailing whitespace"},
		},
		{
			name: "column description with newline is rejected",
			spec: dgModel.DataGraphSpec{
				ID:        "my-dg",
				AccountID: "wh-123",
				Models: []dgModel.ModelSpec{
					baseModel(dgModel.ColumnMetadataYAML{
						Name:        "id",
						Description: "line\nbreak",
					}),
				},
			},
			expectedRefs:          []string{"/models/0/columns/0/description"},
			expectedMsgSubstrings: []string{"control characters"},
		},
		{
			name: "column display_name with tab is rejected",
			spec: dgModel.DataGraphSpec{
				ID:        "my-dg",
				AccountID: "wh-123",
				Models: []dgModel.ModelSpec{
					baseModel(dgModel.ColumnMetadataYAML{
						Name:        "id",
						DisplayName: "User\tID",
					}),
				},
			},
			expectedRefs:          []string{"/models/0/columns/0/display_name"},
			expectedMsgSubstrings: []string{"control characters"},
		},
		{
			name: "column display_name with carriage return is rejected",
			spec: dgModel.DataGraphSpec{
				ID:        "my-dg",
				AccountID: "wh-123",
				Models: []dgModel.ModelSpec{
					baseModel(dgModel.ColumnMetadataYAML{
						Name:        "id",
						DisplayName: "User\rID",
					}),
				},
			},
			expectedRefs:          []string{"/models/0/columns/0/display_name"},
			expectedMsgSubstrings: []string{"control characters"},
		},
		{
			name: "column display_name with leading whitespace",
			spec: dgModel.DataGraphSpec{
				ID:        "my-dg",
				AccountID: "wh-123",
				Models: []dgModel.ModelSpec{
					baseModel(dgModel.ColumnMetadataYAML{
						Name:        "id",
						DisplayName: " User",
					}),
				},
			},
			expectedRefs:          []string{"/models/0/columns/0/display_name"},
			expectedMsgSubstrings: []string{"leading or trailing whitespace"},
		},
		{
			name: "column display_name with trailing whitespace",
			spec: dgModel.DataGraphSpec{
				ID:        "my-dg",
				AccountID: "wh-123",
				Models: []dgModel.ModelSpec{
					baseModel(dgModel.ColumnMetadataYAML{
						Name:        "id",
						DisplayName: "User ",
					}),
				},
			},
			expectedRefs:          []string{"/models/0/columns/0/display_name"},
			expectedMsgSubstrings: []string{"leading or trailing whitespace"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := validateDataGraphSpec("data-graph", specs.SpecVersionV1, nil, tt.spec)
			require.Len(t, results, len(tt.expectedRefs), "unexpected error count; got: %v", results)

			gotRefs := extractRefs(results)
			gotMsgs := extractMsgs(results)
			assert.ElementsMatch(t, tt.expectedRefs, gotRefs, "refs mismatch; got: %v", results)
			// pair up by reference for substring assertions
			for i, wantRef := range tt.expectedRefs {
				idx := -1
				for j, r := range gotRefs {
					if r == wantRef {
						idx = j
						break
					}
				}
				require.GreaterOrEqual(t, idx, 0, "missing expected ref %q in %v", wantRef, gotRefs)
				assert.Contains(t, gotMsgs[idx], tt.expectedMsgSubstrings[i],
					"message at %s did not contain %q; got: %q", wantRef, tt.expectedMsgSubstrings[i], gotMsgs[idx])
			}
		})
	}
}

func TestDataGraphSpecSyntaxValid_ColumnsDuplicateName(t *testing.T) {
	t.Parallel()

	spec := dgModel.DataGraphSpec{
		ID:        "my-dg",
		AccountID: "wh-123",
		Models: []dgModel.ModelSpec{
			{
				ID:          "user",
				DisplayName: "User",
				Type:        "entity",
				Table:       "db.schema.users",
				PrimaryID:   "id",
				Columns: []dgModel.ColumnMetadataYAML{
					{Name: "id", DisplayName: "User ID"},
					{Name: "id", DisplayName: "Identifier"},
				},
			},
		},
	}

	results := validateDataGraphSpec("data-graph", specs.SpecVersionV1, nil, spec)

	// Both entries should be flagged so the error is actionable from either side.
	require.Len(t, results, 2)
	refs := extractRefs(results)
	assert.ElementsMatch(t,
		[]string{"/models/0/columns/0/name", "/models/0/columns/1/name"},
		refs,
	)
	for _, r := range results {
		assert.Contains(t, r.Message, "duplicate column name")
		assert.Contains(t, r.Message, `"id"`)
	}
}

func TestDataGraphSpecSyntaxValid_ColumnsDuplicateDisplayNameCI(t *testing.T) {
	t.Parallel()

	spec := dgModel.DataGraphSpec{
		ID:        "my-dg",
		AccountID: "wh-123",
		Models: []dgModel.ModelSpec{
			{
				ID:          "user",
				DisplayName: "User",
				Type:        "entity",
				Table:       "db.schema.users",
				PrimaryID:   "id",
				Columns: []dgModel.ColumnMetadataYAML{
					{Name: "id", DisplayName: "User ID"},
					{Name: "uid", DisplayName: "user id"},
				},
			},
		},
	}

	results := validateDataGraphSpec("data-graph", specs.SpecVersionV1, nil, spec)

	require.Len(t, results, 2)
	refs := extractRefs(results)
	assert.ElementsMatch(t,
		[]string{"/models/0/columns/0/display_name", "/models/0/columns/1/display_name"},
		refs,
	)
	// The error must name both column names so users can find them quickly.
	for _, r := range results {
		assert.Contains(t, r.Message, "duplicate column display name")
		assert.Contains(t, r.Message, `"id"`)
		assert.Contains(t, r.Message, `"uid"`)
	}
}

// TestModelSpec_ColumnsYAMLRoundTrip verifies that the yaml struct tags on
// ModelSpec + ColumnMetadataYAML produce the expected snake_case keys:
// parse → re-marshal → parse again yields an equivalent struct. Scoped to
// ModelSpec because DataGraphSpec is parsed via mapstructure (covered by
// TestModelSpec_ColumnsMapstructure).
func TestModelSpec_ColumnsYAMLRoundTrip(t *testing.T) {
	t.Parallel()

	input := []byte(`id: user
display_name: User
type: entity
table: db.schema.users
primary_id: id
columns:
  - name: id
    display_name: User ID
  - name: email_address
    display_name: Email
`)

	var parsed dgModel.ModelSpec
	require.NoError(t, yaml.Unmarshal(input, &parsed))

	expected := dgModel.ModelSpec{
		ID:          "user",
		DisplayName: "User",
		Type:        "entity",
		Table:       "db.schema.users",
		PrimaryID:   "id",
		Columns: []dgModel.ColumnMetadataYAML{
			{Name: "id", DisplayName: "User ID"},
			{Name: "email_address", DisplayName: "Email"},
		},
	}
	assert.Equal(t, expected, parsed)

	// Round-trip: marshal back and parse again
	roundTripBytes, err := yaml.Marshal(parsed)
	require.NoError(t, err)

	var reparsed dgModel.ModelSpec
	require.NoError(t, yaml.Unmarshal(roundTripBytes, &reparsed))
	assert.Equal(t, parsed, reparsed)
}

// TestModelSpec_ColumnsMapstructure verifies that the mapstructure tags on
// ColumnMetadataYAML (and the new Columns field on ModelSpec) match the yaml
// keys, so the spec → DataGraphSpec decode path used by the provider works.
func TestModelSpec_ColumnsMapstructure(t *testing.T) {
	t.Parallel()

	specMap := map[string]any{
		"id":         "my-dg",
		"account_id": "wh-123",
		"models": []any{
			map[string]any{
				"id":           "user",
				"display_name": "User",
				"type":         "entity",
				"table":        "db.schema.users",
				"primary_id":   "id",
				"columns": []any{
					map[string]any{
						"name":         "id",
						"display_name": "User ID",
					},
					map[string]any{
						"name":         "email_address",
						"display_name": "Email",
					},
				},
			},
		},
	}

	var decoded dgModel.DataGraphSpec
	require.NoError(t, mapstructure.Decode(specMap, &decoded))

	expected := dgModel.DataGraphSpec{
		ID:        "my-dg",
		AccountID: "wh-123",
		Models: []dgModel.ModelSpec{
			{
				ID:          "user",
				DisplayName: "User",
				Type:        "entity",
				Table:       "db.schema.users",
				PrimaryID:   "id",
				Columns: []dgModel.ColumnMetadataYAML{
					{Name: "id", DisplayName: "User ID"},
					{Name: "email_address", DisplayName: "Email"},
				},
			},
		},
	}
	assert.Equal(t, expected, decoded)
}
