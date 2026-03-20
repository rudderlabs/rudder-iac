package docs

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func minimalRule(ruleID string) DocumentedRule {
	return DocumentedRule{
		RuleID:      ruleID,
		Provider:    "test-provider",
		Phase:       "syntactic",
		Severity:    "error",
		Description: "test description",
		AppliesTo: []MatchPatternDoc{
			{Kind: "source", Version: "v1"},
		},
		MatchBehavior: []MatchBehaviorEntry{
			{
				AppliesTo: []MatchPatternDoc{
					{Kind: "source", Version: "v1"},
				},
			},
		},
	}
}

func TestCatalogValidate_StructuralValidation(t *testing.T) {
	tests := []struct {
		name      string
		rule      DocumentedRule
		wantEmpty bool
		wantField string
	}{
		{
			name:      "valid fully-populated rule passes",
			rule:      minimalRule("rule-A"),
			wantEmpty: true,
		},
		{
			name:      "empty rule_id is rejected",
			rule:      func() DocumentedRule { r := minimalRule(""); return r }(),
			wantField: "rule_id",
		},
		{
			name:      "invalid phase value is rejected",
			rule:      func() DocumentedRule { r := minimalRule("rule-A"); r.Phase = "invalid-phase"; return r }(),
			wantField: "phase",
		},
		{
			name:      "empty applies_to slice is rejected",
			rule:      func() DocumentedRule { r := minimalRule("rule-A"); r.AppliesTo = []MatchPatternDoc{}; return r }(),
			wantField: "applies_to",
		},
		{
			name: "invalid ExpectedDiagnostic severity is rejected",
			rule: func() DocumentedRule {
				r := minimalRule("rule-A")
				r.MatchBehavior[0].Invalid = []InvalidExample{
					{
						ExampleID: "ex-1",
						Title:     "bad example",
						Files:     map[string]string{"main.yaml": "content"},
						ExpectedDiagnostics: []ExpectedDiagnostic{
							{File: "main.yaml", Reference: "/name", Severity: "critical"},
						},
					},
				}
				return r
			}(),
			wantField: "severity",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			catalog := &DocumentedRules{Rules: []DocumentedRule{tt.rule}}
			registered := []string{}
			if tt.rule.RuleID != "" {
				registered = []string{tt.rule.RuleID}
			}
			errs := catalog.Validate(registered)

			if tt.wantEmpty {
				assert.Empty(t, errs)
				return
			}
			require.NotEmpty(t, errs)
			assert.Contains(t, errs[0].Error(), tt.wantField)
		})
	}
}

func TestCatalogValidate_UniqueExampleIDs(t *testing.T) {
	tests := []struct {
		name    string
		rule    DocumentedRule
		wantLen int
	}{
		{
			name:    "all unique example_ids across match_behavior produces no errors",
			rule:    minimalRule("rule-A"),
			wantLen: 0,
		},
		{
			name: "duplicate example_id across valid and invalid in same entry is flagged",
			rule: func() DocumentedRule {
				r := minimalRule("rule-A")
				r.MatchBehavior = append(r.MatchBehavior, MatchBehaviorEntry{
					AppliesTo: []MatchPatternDoc{{Kind: "source", Version: "v2"}},
					Valid: []ValidExample{
						{ExampleID: "ex-shared", Title: "valid example", Files: map[string]string{"a.yaml": "x"}},
					},
					Invalid: []InvalidExample{
						{
							ExampleID: "ex-shared",
							Title:     "invalid example",
							Files:     map[string]string{"a.yaml": "x"},
							ExpectedDiagnostics: []ExpectedDiagnostic{
								{File: "a.yaml", Reference: "/name", Severity: "error"},
							},
						},
					},
				})
				return r
			}(),
			wantLen: 1,
		},
		{
			name: "duplicate example_id across match_behavior entries is flagged",
			rule: func() DocumentedRule {
				r := minimalRule("rule-A")
				mbWithDup := func(kind, version string) MatchBehaviorEntry {
					return MatchBehaviorEntry{
						AppliesTo: []MatchPatternDoc{{Kind: kind, Version: version}},
						Valid: []ValidExample{
							{ExampleID: "ex-dup", Title: "example", Files: map[string]string{"a.yaml": "x"}},
						},
					}
				}
				r.MatchBehavior = append(r.MatchBehavior,
					mbWithDup("source", "v2"),
					mbWithDup("source", "v3"),
				)
				return r
			}(),
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			catalog := &DocumentedRules{Rules: []DocumentedRule{tt.rule}}
			errs := catalog.Validate([]string{tt.rule.RuleID})
			assert.Len(t, errs, tt.wantLen)
		})
	}
}

func TestCatalogValidate_AppliesToCoverage(t *testing.T) {
	tests := []struct {
		name    string
		rule    DocumentedRule
		wantLen int
	}{
		{
			name:    "all top-level pairs present in match_behavior produces no errors",
			rule:    minimalRule("rule-A"),
			wantLen: 0,
		},
		{
			name: "one top-level pair absent from match_behavior is flagged",
			rule: func() DocumentedRule {
				r := minimalRule("rule-A")
				r.AppliesTo = append(r.AppliesTo, MatchPatternDoc{Kind: "destination", Version: "v1"})
				return r
			}(),
			wantLen: 1,
		},
		{
			name: "multiple uncovered pairs each produce an error",
			rule: func() DocumentedRule {
				r := minimalRule("rule-A")
				r.AppliesTo = append(r.AppliesTo,
					MatchPatternDoc{Kind: "destination", Version: "v1"},
					MatchPatternDoc{Kind: "connection", Version: "v2"},
				)
				return r
			}(),
			wantLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			catalog := &DocumentedRules{Rules: []DocumentedRule{tt.rule}}
			errs := catalog.Validate([]string{tt.rule.RuleID})
			assert.Len(t, errs, tt.wantLen)
		})
	}
}

func TestCatalogValidate_RegisteredCompleteness(t *testing.T) {
	tests := []struct {
		name       string
		rules      []DocumentedRule
		registered []string
		wantLen    int
	}{
		{
			name:       "perfect 1:1 mapping produces no errors",
			rules:      []DocumentedRule{minimalRule("rule-A")},
			registered: []string{"rule-A"},
			wantLen:    0,
		},
		{
			name:       "registered rule with no catalog entry is flagged",
			rules:      []DocumentedRule{},
			registered: []string{"rule-missing"},
			wantLen:    1,
		},
		{
			name:       "catalog entry without registration is flagged as orphan",
			rules:      []DocumentedRule{minimalRule("rule-orphan")},
			registered: []string{},
			wantLen:    1,
		},
		{
			name:       "both missing and orphan produce two errors",
			rules:      []DocumentedRule{minimalRule("rule-orphan")},
			registered: []string{"rule-missing"},
			wantLen:    2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			catalog := &DocumentedRules{Rules: tt.rules}
			errs := catalog.Validate(tt.registered)
			assert.Len(t, errs, tt.wantLen)
		})
	}
}
