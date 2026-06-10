package rules

import (
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"unicode"

	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	dgModel "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

var tableRefPattern = regexp.MustCompile(`^[^.]+\.[^.]+\.[^.]+$`)

var examples = rules.Examples{
	Valid: []string{
		`id: my-data-graph
account_id: wh-account-123
models:
  - id: user
    display_name: User
    type: entity
    table: db.schema.users
    primary_id: user_id`,
	},
	Invalid: []string{
		`id: my-data-graph
# Missing required account_id`,
		`id: my-data-graph
account_id: wh-123
models:
  - id: user
    display_name: User
    type: invalid
    table: db.schema.users`,
	},
}

var validateDataGraphSpec = func(_ string, _ string, _ map[string]any, spec dgModel.DataGraphSpec) []rules.ValidationResult {
	validationErrors, err := rules.ValidateStruct(spec, "")
	if err != nil {
		return []rules.ValidationResult{{
			Reference: "",
			Message:   err.Error(),
		}}
	}

	results := funcs.ParseValidationErrors(validationErrors, reflect.TypeOf(dgModel.DataGraphSpec{}))

	// Custom validation not expressible via struct tags
	for i, model := range spec.Models {
		// Table format: must be 3-part reference (catalog.schema.table)
		if model.Table != "" && !tableRefPattern.MatchString(model.Table) {
			results = append(results, rules.ValidationResult{
				Reference: fmt.Sprintf("/models/%d/table", i),
				Message:   "'table' must be a 3-part reference in the format catalog.schema.table",
			})
		}

		// Conditional required fields based on model type
		switch model.Type {
		case "entity":
			if model.PrimaryID == "" {
				results = append(results, rules.ValidationResult{
					Reference: fmt.Sprintf("/models/%d/primary_id", i),
					Message:   "'primary_id' is required for entity models",
				})
			}
			if model.Timestamp != "" {
				results = append(results, rules.ValidationResult{
					Reference: fmt.Sprintf("/models/%d/timestamp", i),
					Message:   "'timestamp' is not allowed on entity models",
				})
			}
		case "event":
			if model.Timestamp == "" {
				results = append(results, rules.ValidationResult{
					Reference: fmt.Sprintf("/models/%d/timestamp", i),
					Message:   "'timestamp' is required for event models",
				})
			}
			if model.PrimaryID != "" {
				results = append(results, rules.ValidationResult{
					Reference: fmt.Sprintf("/models/%d/primary_id", i),
					Message:   "'primary_id' is not allowed on event models",
				})
			}
		}

		results = append(results, validateModelColumns(i, model.Columns)...)
	}

	return results
}

// validateModelColumns enforces the per-column constraints that can't be
// expressed via struct tags: trimmed values, no control characters in
// display_name / description, the "at least one of display_name or description"
// rule, in-model uniqueness of `name`, and case-insensitive uniqueness of
// `display_name` (description has no uniqueness rule). max=255 is handled
// upstream by struct-tag validation.
func validateModelColumns(modelIdx int, columns []dgModel.ColumnMetadataYAML) []rules.ValidationResult {
	if len(columns) == 0 {
		return nil
	}

	var (
		results       []rules.ValidationResult
		nameToIdx     = map[string][]int{}
		dispNameToIdx = map[string][]int{} // lowercased display name -> indexes
		// preserve original display name spelling per lowercase key for messages
		dispNameOriginals = map[string][]string{}
	)

	for j, col := range columns {
		base := fmt.Sprintf("/models/%d/columns/%d", modelIdx, j)

		if col.Name != "" && hasLeadingOrTrailingWhitespace(col.Name) {
			results = append(results, rules.ValidationResult{
				Reference: base + "/name",
				Message:   "'name' must not have leading or trailing whitespace",
			})
		}

		if col.DisplayName == "" && col.Description == "" {
			results = append(results, rules.ValidationResult{
				Reference: base,
				Message:   "each column must set at least one of 'display_name' or 'description'",
			})
		}

		if col.DisplayName != "" {
			if hasLeadingOrTrailingWhitespace(col.DisplayName) {
				results = append(results, rules.ValidationResult{
					Reference: base + "/display_name",
					Message:   "'display_name' must not have leading or trailing whitespace",
				})
			}
			if hasControlCharacters(col.DisplayName) {
				results = append(results, rules.ValidationResult{
					Reference: base + "/display_name",
					Message:   "'display_name' must not contain control characters (tab, newline, carriage return)",
				})
			}
		}

		if col.Description != "" {
			if hasLeadingOrTrailingWhitespace(col.Description) {
				results = append(results, rules.ValidationResult{
					Reference: base + "/description",
					Message:   "'description' must not have leading or trailing whitespace",
				})
			}
			if hasControlCharacters(col.Description) {
				results = append(results, rules.ValidationResult{
					Reference: base + "/description",
					Message:   "'description' must not contain control characters (tab, newline, carriage return)",
				})
			}
		}

		if col.Name != "" {
			nameToIdx[col.Name] = append(nameToIdx[col.Name], j)
		}
		if col.DisplayName != "" {
			key := strings.ToLower(col.DisplayName)
			dispNameToIdx[key] = append(dispNameToIdx[key], j)
			dispNameOriginals[key] = append(dispNameOriginals[key], col.DisplayName)
		}
	}

	results = append(results, collectDuplicateColumnNames(modelIdx, nameToIdx)...)
	results = append(results, collectDuplicateColumnDisplayNames(modelIdx, columns, dispNameToIdx)...)

	return results
}

func collectDuplicateColumnNames(
	modelIdx int,
	nameToIdx map[string][]int,
) []rules.ValidationResult {
	keys := sortedStringKeys(nameToIdx)

	var results []rules.ValidationResult
	for _, name := range keys {
		idxs := nameToIdx[name]
		if len(idxs) < 2 {
			continue
		}
		for _, j := range idxs {
			results = append(results, rules.ValidationResult{
				Reference: fmt.Sprintf("/models/%d/columns/%d/name", modelIdx, j),
				Message:   fmt.Sprintf("duplicate column name %q within model (also at indexes %s)", name, formatPeerIndexes(idxs, j)),
			})
		}
	}
	return results
}

func collectDuplicateColumnDisplayNames(
	modelIdx int,
	columns []dgModel.ColumnMetadataYAML,
	dispNameToIdx map[string][]int,
) []rules.ValidationResult {
	keys := sortedStringKeys(dispNameToIdx)

	var results []rules.ValidationResult
	for _, key := range keys {
		idxs := dispNameToIdx[key]
		if len(idxs) < 2 {
			continue
		}
		// Build a name list to make the error actionable: surface every column
		// name that collides on this display name (case-insensitive).
		conflictingNames := make([]string, 0, len(idxs))
		for _, j := range idxs {
			conflictingNames = append(conflictingNames, fmt.Sprintf("%q", columns[j].Name))
		}
		joined := strings.Join(conflictingNames, ", ")
		for _, j := range idxs {
			results = append(results, rules.ValidationResult{
				Reference: fmt.Sprintf("/models/%d/columns/%d/display_name", modelIdx, j),
				Message: fmt.Sprintf(
					"duplicate column display name %q (case-insensitive) across columns: %s",
					columns[j].DisplayName, joined,
				),
			})
		}
	}
	return results
}

func formatPeerIndexes(all []int, self int) string {
	peers := make([]string, 0, len(all)-1)
	for _, i := range all {
		if i == self {
			continue
		}
		peers = append(peers, fmt.Sprintf("%d", i))
	}
	return strings.Join(peers, ", ")
}

func sortedStringKeys(m map[string][]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func hasLeadingOrTrailingWhitespace(s string) bool {
	return strings.TrimSpace(s) != s
}

func hasControlCharacters(s string) bool {
	for _, r := range s {
		if r == '\t' || r == '\n' || r == '\r' {
			return true
		}
		// Reject other ASCII control characters too for parity with the
		// server's "no control characters" rule.
		if unicode.IsControl(r) {
			return true
		}
	}
	return false
}

func NewDataGraphSpecSyntaxValidRule() rules.Rule {
	return prules.NewTypedRule(
		"datagraph/data-graph/spec-syntax-valid",
		rules.Error,
		"data graph spec syntax must be valid",
		examples,
		prules.NewPatternValidator(
			prules.V1VersionPatterns("data-graph"),
			validateDataGraphSpec,
		),
	)
}
