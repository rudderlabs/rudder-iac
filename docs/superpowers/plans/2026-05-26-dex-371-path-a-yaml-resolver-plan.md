# DEX-371 — Path A: YAML Fragment Authoring for RuleDoc Generator — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the end-to-end RuleDoc pipeline (Resolver → Generator → Verifier → Serializer → CLI command) for spike Path A — YAML fragments under `cli/internal/validation/docs/fragments/` — with three pilot rules authored as worked examples.

**Architecture:** New `Resolver` interface in `cli/internal/validation/docs/` with a `YAMLResolver` implementation that wraps the existing `LoadRuleDocEntries` loader. A `Generator` walks the rules registry (using new `AllSyntacticRules()` / `AllSemanticRules()` methods), invokes the resolver per rule, enriches with rule metadata, and assembles a `RulesDoc`. A `Verifier` materializes each `InvalidExample`'s `Files` map into a tmpdir, loads it via the existing project loader, runs `ValidationEngine.ValidateSyntax`+`ValidateSemantic`, and subset-matches produced diagnostics against authored `ExpectedDiagnostic`s. A `Serializer` emits JSON+YAML to `./docs/generated/`. A new `rudder-cli docs rules` top-level command wires it all together.

**Tech Stack:** Go 1.x, Cobra (CLI), `gopkg.in/yaml.v3`, `encoding/json`, `go-playground/validator/v10` (already used via `rules.ValidateStruct`), `testify` for tests, existing `validation.ValidationEngine` + `project/loader` for executable verification.

**Reference spec:** [`docs/superpowers/specs/2026-05-26-rulesdoc-generator-spikes-design.md`](../specs/2026-05-26-rulesdoc-generator-spikes-design.md) — §6 (Path A commits), §5 (Verifier semantics), §13 (acceptance criteria). Read it before starting.

---

## Pre-flight context (read before starting)

### Worktree

This plan executes inside `/Users/shanmukh/workspace/rudder-iac/.worktrees/dex-371-path-a` on branch `feature/dex-371-spike-path-a-yaml-fragment-authoring-for-ruledoc-generator`. All paths in this plan are relative to that worktree root unless otherwise noted.

### Starting state (PR #471 already on branch)

- `cli/internal/validation/docs/types.go` — defines `RuleDocEntry`, `MatchBehaviorEntry`, `MatchPatternDoc`, `ValidExample`, `InvalidExample`, `ExpectedDiagnostic` (authored types) and `DocumentedRules` / `DocumentedRule` / `ToolMetadata` (resolved types — note PR-#471 naming, renamed by this plan in Task 1).
- `cli/internal/validation/docs/rule_doc_entry.go` — `LoadRuleDocEntries(fsys, dir)` reads all `.yaml`/`.yml` files in a dir into `[]RuleDocEntry`. Tests in `rule_doc_entry_test.go`.
- `cli/internal/validation/docs/rules_doc.go` — `(c *DocumentedRules).Validate(registeredRuleIDs)` runs four checks: struct, applies-to coverage, unique example IDs, registered completeness. Tests in `rules_doc_test.go`.

### Naming rename — decided locally, not blocked on #475

Per spec §intro, the rename `DocumentedRules` → `RulesDoc` and `DocumentedRule` → `ResolvedRule` was meant to come from PR #475. Since the spike branch does **not** stack on #475 (#475 is being split — see spec §4), this plan performs the rename itself in Task 1 (spike prep). The rename is mechanical and the diff stays small.

### Registry-extension prerequisite — done locally too

`AllSyntacticRules()` / `AllSemanticRules()` on `rules.Registry` are required by the Generator but do not yet exist on this branch. Spec §4 says these should land via a carve-out PR before the spikes. Rather than block the plan, we add the two methods + tests in Task 1 (spike-prep commit). If a carve-out PR lands first, this commit becomes a no-op rebase and that's fine. **Document this clearly in the spike PR description.**

### What stays untouched (per spec §6 "What stays untouched")

- No edits to any `rules.Rule` implementation (the existing `Examples()` method on rules, used by the diagnostic renderer, is left alone).
- No edits to the `RuleProvider` interface.
- No changes to the `ValidationEngine` itself.

### Testing conventions (per `CLAUDE.md`)

- Use `testify/assert` and `testify/require`.
- Compare entire structs over field-by-field. Example: `assert.Equal(t, &ResolvedRule{ID: "x", Phase: "syntactic"}, got)`.
- Each task's failing-test step MUST be run before implementation. The TDD loop is mandatory.

### Verification

- `make lint` and `make test` must remain green after every commit (run them in the verification step of each task).
- Commit messages follow Conventional Commits — see `git log --oneline` for the project's style. The branch is named `feature/dex-371-spike-path-a-yaml-fragment-authoring-for-ruledoc-generator`; do not rename it.
- Do NOT push to remote during plan execution — the executor session will do that after the plan completes.

---

## File structure — what lives where

New files (created by this plan):

- `cli/internal/validation/docs/resolver.go` — `Resolver` interface + `YAMLResolver` implementation.
- `cli/internal/validation/docs/resolver_test.go` — unit tests for `YAMLResolver`.
- `cli/internal/validation/docs/generator.go` — `Generator` struct that walks the registry and builds a `RulesDoc`.
- `cli/internal/validation/docs/generator_test.go` — unit tests using a fake `Resolver` and a hand-built registry.
- `cli/internal/validation/docs/verifier.go` — `Verifier` that materializes `InvalidExample.Files` into a tmpdir and subset-matches diagnostics.
- `cli/internal/validation/docs/verifier_test.go` — unit tests covering subset match success and failure paths.
- `cli/internal/validation/docs/serializer.go` — `EmitYAML(...)` and `EmitJSON(...)` helpers.
- `cli/internal/validation/docs/serializer_test.go` — golden-output assertion tests.
- `cli/internal/cmd/docs/docs.go` — top-level `docs` Cobra command group.
- `cli/internal/cmd/docs/rules/rules.go` — `rudder-cli docs rules` subcommand.
- `cli/internal/cmd/docs/rules/rules_test.go` — smoke test that the command parses flags.
- `cli/internal/validation/docs/fragments/datacatalog-categories-spec-syntax-valid.docs.yaml` — pilot fragment.
- `cli/internal/validation/docs/fragments/project-metadata-syntax-valid.docs.yaml` — pilot fragment.
- `cli/internal/validation/docs/fragments/project-duplicate-urn.docs.yaml` — pilot fragment.

Files modified:

- `cli/internal/validation/docs/types.go` — rename types (Task 1).
- `cli/internal/validation/docs/rules_doc.go` — rename + comment-out completeness check + rename parameter (Task 1).
- `cli/internal/validation/docs/rule_doc_entry.go` — no rename needed; signatures keep `RuleDocEntry`.
- `cli/internal/validation/docs/rules_doc_test.go` — propagate rename (Task 1).
- `cli/internal/validation/rules/registry.go` — add `AllSyntacticRules()` / `AllSemanticRules()` methods (Task 1).
- `cli/internal/validation/rules/registry_test.go` — new test file with coverage for the two new methods (Task 1; create if absent).
- `cli/internal/cmd/root.go` — register the new `docs` command (Task 6).

---

## Task 1: Spike prep — rename + registry extensions + completeness gate

**Why first:** Everything else depends on the renamed types and the new registry methods. Land them as one focused commit so the rest of the plan is on stable ground.

**Files:**
- Modify: `cli/internal/validation/docs/types.go`
- Modify: `cli/internal/validation/docs/rules_doc.go`
- Modify: `cli/internal/validation/docs/rules_doc_test.go`
- Modify: `cli/internal/validation/rules/registry.go`
- Create or modify: `cli/internal/validation/rules/registry_test.go`

- [ ] **Step 1: Confirm current state**

Run: `git status` (expect clean tree) and `grep -rn "DocumentedRule\|DocumentedRules" cli/internal/` (expect matches only in `cli/internal/validation/docs/`).

Expected: clean tree; the only references to `DocumentedRule`/`DocumentedRules` are inside `cli/internal/validation/docs/`.

- [ ] **Step 2: Write failing tests for new registry methods**

Create `cli/internal/validation/rules/registry_test.go` (if a `registry_test.go` already exists, add the test cases instead — the file in the worktree currently has none).

```go
package rules

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type fakeRule struct {
	id        string
	severity  Severity
	appliesTo []MatchPattern
}

func (f *fakeRule) ID() string                                    { return f.id }
func (f *fakeRule) Severity() Severity                            { return f.severity }
func (f *fakeRule) Description() string                           { return "" }
func (f *fakeRule) AppliesTo() []MatchPattern                     { return f.appliesTo }
func (f *fakeRule) Examples() Examples                            { return Examples{} }
func (f *fakeRule) Validate(_ *ValidationContext) []ValidationResult { return nil }

func TestRegistry_AllSyntacticRules_ReturnsRegisteredCopy(t *testing.T) {
	r := NewRegistry()
	r1 := &fakeRule{id: "rule-1"}
	r2 := &fakeRule{id: "rule-2"}
	r.RegisterSyntactic(r1)
	r.RegisterSyntactic(r2)

	got := r.AllSyntacticRules()

	assert.Equal(t, []Rule{r1, r2}, got)
}

func TestRegistry_AllSemanticRules_ReturnsRegisteredCopy(t *testing.T) {
	r := NewRegistry()
	r1 := &fakeRule{id: "rule-1"}
	r2 := &fakeRule{id: "rule-2"}
	r.RegisterSemantic(r1)
	r.RegisterSemantic(r2)

	got := r.AllSemanticRules()

	assert.Equal(t, []Rule{r1, r2}, got)
}

func TestRegistry_AllSyntacticRules_DefensiveCopy(t *testing.T) {
	r := NewRegistry()
	r.RegisterSyntactic(&fakeRule{id: "rule-1"})

	got := r.AllSyntacticRules()
	got[0] = &fakeRule{id: "mutated"}

	// mutating the returned slice must not affect future calls
	assert.Equal(t, "rule-1", r.AllSyntacticRules()[0].ID())
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./cli/internal/validation/rules/ -run TestRegistry_All -v`
Expected: FAIL with "AllSyntacticRules undefined" / "AllSemanticRules undefined".

- [ ] **Step 4: Add methods to the Registry interface and default impl**

In `cli/internal/validation/rules/registry.go`, extend the interface:

```go
type Registry interface {
	RegisterSyntactic(rule Rule)
	RegisterSemantic(rule Rule)
	SyntacticRulesFor(kind, version string) []Rule
	SemanticRulesFor(kind, version string) []Rule

	// AllSyntacticRules returns every registered syntactic rule. The returned
	// slice is a defensive copy; mutating it does not affect the registry.
	AllSyntacticRules() []Rule

	// AllSemanticRules returns every registered semantic rule. The returned
	// slice is a defensive copy; mutating it does not affect the registry.
	AllSemanticRules() []Rule
}
```

Add implementations to `defaultRegistry`:

```go
func (r *defaultRegistry) AllSyntacticRules() []Rule {
	out := make([]Rule, len(r.syntactic))
	copy(out, r.syntactic)
	return out
}

func (r *defaultRegistry) AllSemanticRules() []Rule {
	out := make([]Rule, len(r.semantic))
	copy(out, r.semantic)
	return out
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./cli/internal/validation/rules/ -run TestRegistry_All -v`
Expected: PASS for all three test cases.

- [ ] **Step 6: Rename `DocumentedRules` → `RulesDoc` and `DocumentedRule` → `ResolvedRule` in `types.go`**

In `cli/internal/validation/docs/types.go`, change only the resolved-types block (the authored-types block at the top of the file — `RuleDocEntry`, `MatchBehaviorEntry`, `MatchPatternDoc`, `ValidExample`, `InvalidExample`, `ExpectedDiagnostic` — is untouched).

```go
// Resolved types — what the generator emits in the YAML catalog.

type RulesDoc struct {
	SchemaVersion int            `yaml:"schema_version"`
	ToolMetadata  ToolMetadata   `yaml:"tool_metadata"`
	Rules         []ResolvedRule `yaml:"rules"`
}

type ToolMetadata struct {
	CLIVersion  string `yaml:"cli_version"`
	GeneratedAt string `yaml:"generated_at"`
}

type ResolvedRule struct {
	RuleID        string               `yaml:"rule_id"        validate:"required"`
	Provider      string               `yaml:"provider"       validate:"required"`
	Phase         string               `yaml:"phase"          validate:"required,oneof=syntactic semantic"`
	Severity      string               `yaml:"severity"       validate:"required,oneof=error warning info"`
	Description   string               `yaml:"description"    validate:"required"`
	AppliesTo     []MatchPatternDoc    `yaml:"applies_to"     validate:"required,min=1,dive"`
	MatchBehavior []MatchBehaviorEntry `yaml:"match_behavior" validate:"required,min=1,dive"`
}
```

> Only the two outer type names change. Inner type names (`MatchBehaviorEntry`, `MatchPatternDoc`, etc.) stay as they are.

- [ ] **Step 7: Update `rules_doc.go` for rename + comment out completeness + rename parameter**

In `cli/internal/validation/docs/rules_doc.go`:
1. Replace all `*DocumentedRules` with `*RulesDoc` and all `*DocumentedRule` with `*ResolvedRule`.
2. Rename the `Validate` parameter from `registeredRuleIDs` to `expectedRuleIDs`.
3. Comment out the call to `validateRegisteredCompleteness` with a restoration marker.

The `Validate` method becomes:

```go
func (c *RulesDoc) Validate(expectedRuleIDs []string) []error {
	var errs []error
	for i := range c.Rules {
		structErrs := validateRuleStruct(&c.Rules[i])

		errs = append(errs, structErrs...)
		if len(structErrs) > 0 {
			// Skip per-rule checks when structure is
			// invalid — guard clause per spec.
			continue
		}

		errs = append(errs, validateAppliesToCoverage(&c.Rules[i])...)
		errs = append(errs, validateUniqueExampleIDs(&c.Rules[i])...)
	}
	// TODO(spike DEX-371): re-enable after pilot phase. Disabled because
	// only 3 of ~13 registered rules currently have docs — see spec §3 Layer 3.
	// errs = append(errs, c.validateRegisteredCompleteness(expectedRuleIDs)...)
	_ = expectedRuleIDs // suppress unused-parameter lint until restoration
	return errs
}
```

Keep `validateRegisteredCompleteness` itself in place (unused but compiles, exercised by its dedicated test). The receiver type changes to `*RulesDoc`.

- [ ] **Step 8: Propagate rename through `rules_doc_test.go`**

Replace every `DocumentedRules` with `RulesDoc` and every `DocumentedRule` with `ResolvedRule` in `cli/internal/validation/docs/rules_doc_test.go`.

Update the test helper:

```go
func minimalRule(ruleID string) ResolvedRule {
	return ResolvedRule{
		// ... same body, type changed only.
	}
}
```

Add a top-of-file comment near the completeness test block:

```go
// NOTE: validateRegisteredCompleteness is currently commented out in
// RulesDoc.Validate (see spec §3 Layer 3, restoration marker in rules_doc.go).
// The function still compiles and is tested in isolation here — restoring
// it in Validate is a one-line uncomment.
```

Then update the registered-completeness test to call the method directly instead of via `Validate`:

```go
func TestCatalogValidate_RegisteredCompleteness(t *testing.T) {
	// ... use rules_doc.validateRegisteredCompleteness(tt.registered) directly.
}
```

(Concretely: change `errs := catalog.Validate(tt.registered)` to `errs := catalog.validateRegisteredCompleteness(tt.registered)`.)

- [ ] **Step 9: Run all existing tests in the docs and rules packages**

Run: `go test ./cli/internal/validation/docs/... ./cli/internal/validation/rules/... -v`
Expected: all tests pass. If anything fails, fix the rename before continuing.

- [ ] **Step 10: Run lint**

Run: `make lint`
Expected: PASS. If lint flags `_ = expectedRuleIDs`, replace with `//nolint:unused` on the function or leave the blank-assignment in — both are fine.

- [ ] **Step 11: Commit**

```bash
git add cli/internal/validation/docs/types.go \
        cli/internal/validation/docs/rules_doc.go \
        cli/internal/validation/docs/rules_doc_test.go \
        cli/internal/validation/rules/registry.go \
        cli/internal/validation/rules/registry_test.go
git commit -m "feat(validation/docs): spike prep — rename to RulesDoc, add registry enumeration, gate completeness check

- Rename DocumentedRules→RulesDoc, DocumentedRule→ResolvedRule (was reserved for #475 but stacking required).
- Add Registry.AllSyntacticRules() / AllSemanticRules() with defensive copies; tests included.
- Comment out validateRegisteredCompleteness call with restoration marker — only 3/~13 rules have docs.
- Rename Validate parameter registeredRuleIDs→expectedRuleIDs.

Refs DEX-371."
```

---

## Task 2: Resolver interface + YAMLResolver implementation

**Why:** This is the seam in spec §2. Path A's `YAMLResolver` wraps `LoadRuleDocEntries` and indexes results by `RuleID`. Generator depends only on the interface.

**Files:**
- Create: `cli/internal/validation/docs/resolver.go`
- Create: `cli/internal/validation/docs/resolver_test.go`

- [ ] **Step 1: Write the failing test for YAMLResolver**

Create `cli/internal/validation/docs/resolver_test.go`:

```go
package docs

import (
	"testing"
	"testing/fstest"

	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// minimalRuleStub satisfies rules.Rule for resolver tests without depending
// on any real rule registration.
type minimalRuleStub struct {
	id string
}

func (s *minimalRuleStub) ID() string                                    { return s.id }
func (s *minimalRuleStub) Severity() rules.Severity                      { return rules.Error }
func (s *minimalRuleStub) Description() string                           { return "" }
func (s *minimalRuleStub) AppliesTo() []rules.MatchPattern               { return nil }
func (s *minimalRuleStub) Examples() rules.Examples                      { return rules.Examples{} }
func (s *minimalRuleStub) Validate(_ *rules.ValidationContext) []rules.ValidationResult {
	return nil
}

func TestYAMLResolver_ResolveFor_ReturnsAuthoredEntry(t *testing.T) {
	fsys := fstest.MapFS{
		"frags/rule-a.docs.yaml": &fstest.MapFile{Data: []byte("rule_id: rule-a\nmatch_behavior:\n  - applies_to:\n      - {kind: source, version: v1}\n")},
	}
	r, err := NewYAMLResolver(fsys, "frags")
	require.NoError(t, err)

	got, err := r.ResolveFor(&minimalRuleStub{id: "rule-a"})
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "rule-a", got.RuleID)
	require.Len(t, got.MatchBehavior, 1)
	assert.Equal(t, MatchPatternDoc{Kind: "source", Version: "v1"}, got.MatchBehavior[0].AppliesTo[0])
}

func TestYAMLResolver_ResolveFor_ReturnsNilWhenNoDocs(t *testing.T) {
	fsys := fstest.MapFS{
		"frags/rule-a.docs.yaml": &fstest.MapFile{Data: []byte("rule_id: rule-a\nmatch_behavior: []\n")},
	}
	r, err := NewYAMLResolver(fsys, "frags")
	require.NoError(t, err)

	got, err := r.ResolveFor(&minimalRuleStub{id: "rule-not-authored"})
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestYAMLResolver_ResolveFor_ErrorsOnDuplicateRuleID(t *testing.T) {
	fsys := fstest.MapFS{
		"frags/a.docs.yaml": &fstest.MapFile{Data: []byte("rule_id: dup\n")},
		"frags/b.docs.yaml": &fstest.MapFile{Data: []byte("rule_id: dup\n")},
	}
	_, err := NewYAMLResolver(fsys, "frags")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate rule_id")
}

func TestNewYAMLResolver_PropagatesLoadError(t *testing.T) {
	fsys := fstest.MapFS{}
	_, err := NewYAMLResolver(fsys, "no-such-dir")
	require.Error(t, err)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cli/internal/validation/docs/ -run TestYAMLResolver -v`
Expected: FAIL with "Resolver undefined" / "NewYAMLResolver undefined".

- [ ] **Step 3: Implement the Resolver interface and YAMLResolver**

Create `cli/internal/validation/docs/resolver.go`:

```go
package docs

import (
	"fmt"
	"io/fs"

	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// Resolver returns the authored doc data for a given rule.
// A nil return (with nil error) means no docs are authored for the rule.
// An error indicates an authoring or load problem, not missing docs.
type Resolver interface {
	ResolveFor(r rules.Rule) (*RuleDocEntry, error)
}

// YAMLResolver loads all YAML fragments from a directory at construction
// and indexes them by rule_id. ResolveFor is a map lookup.
type YAMLResolver struct {
	byRuleID map[string]RuleDocEntry
}

// NewYAMLResolver reads every .yaml/.yml file under dir on fsys and builds
// the rule_id index. Returns an error if any fragment fails to parse or
// two fragments declare the same rule_id.
func NewYAMLResolver(fsys fs.FS, dir string) (*YAMLResolver, error) {
	entries, err := LoadRuleDocEntries(fsys, dir)
	if err != nil {
		return nil, fmt.Errorf("loading rule doc fragments: %w", err)
	}

	byID := make(map[string]RuleDocEntry, len(entries))
	for _, e := range entries {
		if _, exists := byID[e.RuleID]; exists {
			return nil, fmt.Errorf("duplicate rule_id %q across fragments", e.RuleID)
		}
		byID[e.RuleID] = e
	}

	return &YAMLResolver{byRuleID: byID}, nil
}

func (r *YAMLResolver) ResolveFor(rule rules.Rule) (*RuleDocEntry, error) {
	entry, ok := r.byRuleID[rule.ID()]
	if !ok {
		return nil, nil
	}
	out := entry
	return &out, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cli/internal/validation/docs/ -run TestYAMLResolver -v`
Expected: PASS for all four tests.

- [ ] **Step 5: Run full docs-package tests + lint**

Run: `go test ./cli/internal/validation/docs/... -v && make lint`
Expected: all tests pass, lint clean.

- [ ] **Step 6: Commit**

```bash
git add cli/internal/validation/docs/resolver.go cli/internal/validation/docs/resolver_test.go
git commit -m "feat(validation/docs): add Resolver interface + YAMLResolver for Path A

YAMLResolver wraps LoadRuleDocEntries: reads every fragment at construction
and indexes by rule_id. ResolveFor is a map lookup that returns (nil, nil)
for rules without authored docs, distinguishing 'missing' from 'errored'.

Refs DEX-371."
```

---

## Task 3: Generator — walk registry, call resolver, build RulesDoc

**Why:** The Generator is the heart of step 2-3 in the pipeline diagram. It walks `AllSyntacticRules() ∪ AllSemanticRules()`, asks the Resolver for each rule's authored entry, and enriches it into a `ResolvedRule`.

**Files:**
- Create: `cli/internal/validation/docs/generator.go`
- Create: `cli/internal/validation/docs/generator_test.go`

- [ ] **Step 1: Write the failing test**

Create `cli/internal/validation/docs/generator_test.go`:

```go
package docs

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeRule struct {
	id          string
	severity    rules.Severity
	description string
	appliesTo   []rules.MatchPattern
}

func (f *fakeRule) ID() string                                                    { return f.id }
func (f *fakeRule) Severity() rules.Severity                                      { return f.severity }
func (f *fakeRule) Description() string                                           { return f.description }
func (f *fakeRule) AppliesTo() []rules.MatchPattern                               { return f.appliesTo }
func (f *fakeRule) Examples() rules.Examples                                      { return rules.Examples{} }
func (f *fakeRule) Validate(_ *rules.ValidationContext) []rules.ValidationResult { return nil }

type fakeResolver struct {
	byID map[string]*RuleDocEntry
}

func (f *fakeResolver) ResolveFor(r rules.Rule) (*RuleDocEntry, error) {
	return f.byID[r.ID()], nil
}

func TestGenerator_Generate_EnrichesResolvedRule(t *testing.T) {
	syntactic := &fakeRule{
		id:          "rule-syn",
		severity:    rules.Error,
		description: "syntactic rule",
		appliesTo:   []rules.MatchPattern{rules.MatchKindVersion("source", "v1")},
	}
	semantic := &fakeRule{
		id:          "rule-sem",
		severity:    rules.Warning,
		description: "semantic rule",
		appliesTo:   []rules.MatchPattern{rules.MatchAll()},
	}

	reg := rules.NewRegistry()
	reg.RegisterSyntactic(syntactic)
	reg.RegisterSemantic(semantic)

	resolver := &fakeResolver{byID: map[string]*RuleDocEntry{
		"rule-syn": {
			RuleID: "rule-syn",
			MatchBehavior: []MatchBehaviorEntry{
				{AppliesTo: []MatchPatternDoc{{Kind: "source", Version: "v1"}}},
			},
		},
		"rule-sem": {
			RuleID: "rule-sem",
			MatchBehavior: []MatchBehaviorEntry{
				{AppliesTo: []MatchPatternDoc{{Kind: "*", Version: "*"}}},
			},
		},
	}}

	gen := NewGenerator(reg, resolver, GeneratorOptions{CLIVersion: "test-1.0", SchemaVersion: 1})

	doc, err := gen.Generate()
	require.NoError(t, err)

	require.Len(t, doc.Rules, 2)
	// Order: syntactic then semantic, registration order preserved.
	assert.Equal(t, "rule-syn", doc.Rules[0].RuleID)
	assert.Equal(t, "syntactic", doc.Rules[0].Phase)
	assert.Equal(t, "error", doc.Rules[0].Severity)
	assert.Equal(t, "syntactic rule", doc.Rules[0].Description)
	assert.Equal(t, []MatchPatternDoc{{Kind: "source", Version: "v1"}}, doc.Rules[0].AppliesTo)

	assert.Equal(t, "rule-sem", doc.Rules[1].RuleID)
	assert.Equal(t, "semantic", doc.Rules[1].Phase)
	assert.Equal(t, "warning", doc.Rules[1].Severity)
}

func TestGenerator_Generate_SkipsRulesWithoutDocs(t *testing.T) {
	reg := rules.NewRegistry()
	reg.RegisterSyntactic(&fakeRule{id: "rule-undocumented"})
	resolver := &fakeResolver{byID: map[string]*RuleDocEntry{}}

	gen := NewGenerator(reg, resolver, GeneratorOptions{CLIVersion: "x", SchemaVersion: 1})

	doc, err := gen.Generate()
	require.NoError(t, err)
	assert.Empty(t, doc.Rules)
}

func TestGenerator_Generate_PassesStructuralValidate(t *testing.T) {
	rule := &fakeRule{
		id:          "rule-syn",
		severity:    rules.Error,
		description: "syntactic rule",
		appliesTo:   []rules.MatchPattern{rules.MatchKindVersion("source", "v1")},
	}
	reg := rules.NewRegistry()
	reg.RegisterSyntactic(rule)
	resolver := &fakeResolver{byID: map[string]*RuleDocEntry{
		"rule-syn": {
			RuleID: "rule-syn",
			MatchBehavior: []MatchBehaviorEntry{
				{AppliesTo: []MatchPatternDoc{{Kind: "source", Version: "v1"}}},
			},
		},
	}}

	gen := NewGenerator(reg, resolver, GeneratorOptions{CLIVersion: "x", SchemaVersion: 1})
	doc, err := gen.Generate()
	require.NoError(t, err)

	errs := doc.Validate(nil)
	assert.Empty(t, errs)
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./cli/internal/validation/docs/ -run TestGenerator -v`
Expected: FAIL with "NewGenerator undefined" / "GeneratorOptions undefined".

- [ ] **Step 3: Implement the Generator**

Create `cli/internal/validation/docs/generator.go`:

```go
package docs

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// GeneratorOptions configures the Generator at construction.
type GeneratorOptions struct {
	CLIVersion    string
	SchemaVersion int
}

// Generator builds a RulesDoc from a registry + resolver. The resolver
// supplies authored data; the generator enriches each entry with metadata
// pulled from the rule itself.
type Generator struct {
	registry rules.Registry
	resolver Resolver
	opts     GeneratorOptions
}

func NewGenerator(reg rules.Registry, resolver Resolver, opts GeneratorOptions) *Generator {
	return &Generator{registry: reg, resolver: resolver, opts: opts}
}

// Generate walks both syntactic and semantic rule lists, resolves each one,
// and assembles a *RulesDoc. Rules without authored docs are skipped silently —
// the spike has only 3 pilots, full coverage is enforced later via
// validateRegisteredCompleteness (currently gated, see rules_doc.go).
func (g *Generator) Generate() (*RulesDoc, error) {
	resolved := make([]ResolvedRule, 0)

	if rs, err := g.resolvePhase(g.registry.AllSyntacticRules(), "syntactic"); err != nil {
		return nil, err
	} else {
		resolved = append(resolved, rs...)
	}

	if rs, err := g.resolvePhase(g.registry.AllSemanticRules(), "semantic"); err != nil {
		return nil, err
	} else {
		resolved = append(resolved, rs...)
	}

	return &RulesDoc{
		SchemaVersion: g.opts.SchemaVersion,
		ToolMetadata: ToolMetadata{
			CLIVersion: g.opts.CLIVersion,
			// GeneratedAt left empty in the spike — reproducibility win.
			// Restore via opts if a downstream consumer needs it.
		},
		Rules: resolved,
	}, nil
}

func (g *Generator) resolvePhase(ruleList []rules.Rule, phase string) ([]ResolvedRule, error) {
	out := make([]ResolvedRule, 0, len(ruleList))
	for _, r := range ruleList {
		entry, err := g.resolver.ResolveFor(r)
		if err != nil {
			return nil, fmt.Errorf("resolving docs for rule %s: %w", r.ID(), err)
		}
		if entry == nil {
			continue
		}
		out = append(out, ResolvedRule{
			RuleID:        r.ID(),
			Provider:      providerFromRuleID(r.ID()),
			Phase:         phase,
			Severity:      r.Severity().String(),
			Description:   r.Description(),
			AppliesTo:     matchPatternsToDocs(r.AppliesTo()),
			MatchBehavior: entry.MatchBehavior,
		})
	}
	return out, nil
}

// providerFromRuleID extracts the provider segment from a rule ID. Convention
// is "provider/resource/check-name" (e.g., "datacatalog/categories/spec-syntax-valid").
// Project-level rules use "project" as the provider segment by convention.
func providerFromRuleID(id string) string {
	for i := 0; i < len(id); i++ {
		if id[i] == '/' {
			return id[:i]
		}
	}
	return id
}

// matchPatternsToDocs converts the rule's runtime MatchPattern slice into
// the YAML-tagged MatchPatternDoc used in the catalog.
func matchPatternsToDocs(in []rules.MatchPattern) []MatchPatternDoc {
	out := make([]MatchPatternDoc, len(in))
	for i, p := range in {
		out[i] = MatchPatternDoc{Kind: p.Kind, Version: p.Version}
	}
	return out
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cli/internal/validation/docs/ -run TestGenerator -v`
Expected: PASS for all three test cases.

- [ ] **Step 5: Run full docs-package tests + lint**

Run: `go test ./cli/internal/validation/docs/... -v && make lint`
Expected: all green.

- [ ] **Step 6: Commit**

```bash
git add cli/internal/validation/docs/generator.go cli/internal/validation/docs/generator_test.go
git commit -m "feat(validation/docs): add Generator that walks registry and builds RulesDoc

Generator iterates AllSyntacticRules() then AllSemanticRules(), calls
the supplied Resolver per rule, and enriches each entry with Phase,
Severity, Description, AppliesTo, and Provider (derived from RuleID).
Rules without authored docs are skipped silently — the spike covers
only 3 pilots; full coverage is enforced later by the gated
validateRegisteredCompleteness check.

Refs DEX-371."
```

---

## Task 4: Verifier — execute invalid examples and subset-match diagnostics

**Why:** This is the load-bearing property of the spike (spec §1 — "executable docs"). For every authored `InvalidExample`, materialize `Files` to a tmpdir, run the engine, and assert every `ExpectedDiagnostic` matches at least one produced diagnostic.

**Files:**
- Create: `cli/internal/validation/docs/verifier.go`
- Create: `cli/internal/validation/docs/verifier_test.go`

### Design notes

- **Subset semantics:** each authored `ExpectedDiagnostic` must match at least one produced diagnostic on (`Severity`, `Reference`, `File`); produced `Message` must `Contains` the authored `MessageContains` substring (skip the contains check if `MessageContains == ""`). Extras allowed.
- **Project loader reuse:** use the same `loader.Loader{}` that `project.New` uses — its `Load(location)` returns `map[string]*specs.RawSpec`, exactly what `ValidationEngine.ValidateSyntax` needs.
- **Registry setup:** the verifier needs a registry to run the engine against. Inject the same registry the Generator was given (so the same rules are exercised). This is wired in the CLI command (Task 6); the verifier takes the engine as a dependency.
- **Files map → tmpdir:** key is filename (relative path within tmpdir, slashes allowed); value is YAML content. Write each file, ensuring parent dirs exist.
- **Validation phases:** the spike pilots include both syntactic (`metadata-syntax-valid`, `categories/spec-syntax-valid`) and project-wide (`duplicate-urn`) rules. The verifier must call both `ValidateSyntax` and `ValidateSemantic`, concatenating diagnostics. Building the resource graph for semantic phase requires a `provider.Provider`. **Simplification:** for the spike, the verifier runs only `ValidateSyntax`. The three pilot rules selected by the spec are all reachable in the syntactic phase (`duplicate-urn` runs in Phase 2 of `ValidateSyntax` as a `ProjectRule`). Document this assumption explicitly in a comment in `verifier.go` and in the spike PR description. If a future pilot needs semantic-phase validation, extend the verifier then.
- **Path normalization:** `ExpectedDiagnostic.File` and produced `Diagnostic.File` must match. Produced files include the tmpdir prefix; normalize by stripping the tmpdir prefix before comparison.

- [ ] **Step 1: Write the failing tests**

Create `cli/internal/validation/docs/verifier_test.go`:

```go
package docs

import (
	"context"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/pathindex"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeEngine satisfies the narrow verifierEngine interface declared in
// verifier.go (only ValidateSyntax). It returns whatever diagnostics it was
// constructed with — used to exercise the verifier's subset-match logic
// without standing up the real engine.
type fakeEngine struct {
	syntax validation.Diagnostics
}

func (f *fakeEngine) ValidateSyntax(_ context.Context, _ map[string]*specs.RawSpec) (validation.Diagnostics, error) {
	return f.syntax, nil
}

func TestVerifier_Verify_SubsetMatchSucceeds(t *testing.T) {
	example := InvalidExample{
		ExampleID: "ex-1",
		Title:     "missing name",
		Files:     map[string]string{"a.yaml": "kind: source\nspec: {}\n"},
		ExpectedDiagnostics: []ExpectedDiagnostic{
			{File: "a.yaml", Reference: "/name", Severity: "error", MessageContains: "required"},
		},
	}

	produced := validation.Diagnostics{
		{
			RuleID:   "rule-a",
			Severity: rules.Error,
			Message:  "field 'name' is required",
			File:     "a.yaml", // verifier strips tmpdir prefix before comparison
			Position: pathindex.Position{Line: 1},
		},
	}

	v := newVerifierForTest(&fakeEngine{syntax: produced})
	err := v.verifyExample(context.Background(), example, "rule-a")
	require.NoError(t, err)
}

func TestVerifier_Verify_FailsWhenExpectedDiagnosticMissing(t *testing.T) {
	example := InvalidExample{
		ExampleID: "ex-1",
		Title:     "missing name",
		Files:     map[string]string{"a.yaml": "kind: source\nspec: {}\n"},
		ExpectedDiagnostics: []ExpectedDiagnostic{
			{File: "a.yaml", Reference: "/name", Severity: "error", MessageContains: "required"},
		},
	}

	v := newVerifierForTest(&fakeEngine{syntax: nil}) // engine produces nothing

	err := v.verifyExample(context.Background(), example, "rule-a")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ex-1")
	assert.Contains(t, err.Error(), "expected diagnostic")
}
```

> **Note for executor:** the test references `newVerifierForTest` and the `verifierEngine` interface. Both are declared in `verifier.go` in step 3 — `newVerifierForTest` is a package-private test-only constructor, and `verifierEngine` is the narrow interface the verifier depends on (only `ValidateSyntax`). Keeping the interface narrow lets the test fake remain a single-method type.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cli/internal/validation/docs/ -run TestVerifier -v`
Expected: FAIL with "Verifier undefined" etc.

- [ ] **Step 3: Implement the Verifier**

Create `cli/internal/validation/docs/verifier.go`:

```go
package docs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/loader"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// verifierEngine is the slice of ValidationEngine the verifier actually uses.
// Keeping it narrow lets tests substitute a fake.
//
// The spike runs only ValidateSyntax: the three pilot rules all execute in
// that phase (duplicate-urn is a ProjectRule executed in Phase 2 of
// ValidateSyntax). Adding semantic-phase verification is a follow-up if a
// future pilot needs it.
type verifierEngine interface {
	ValidateSyntax(ctx context.Context, rawSpecs map[string]*specs.RawSpec) (validation.Diagnostics, error)
}

// Verifier executes each authored InvalidExample through the validation
// engine and asserts that every ExpectedDiagnostic matches at least one
// produced diagnostic (subset semantics — see spec §5).
type Verifier struct {
	engine verifierEngine
	loader *loader.Loader
	log    *logger.Logger
}

// NewVerifier wires the verifier with a real engine backed by the given
// registry. The caller is responsible for populating the registry with the
// rules whose docs are being verified.
func NewVerifier(reg rules.Registry, log *logger.Logger) (*Verifier, error) {
	eng, err := validation.NewValidationEngine(reg, log)
	if err != nil {
		return nil, fmt.Errorf("initialising verifier engine: %w", err)
	}
	return &Verifier{engine: eng, loader: &loader.Loader{}, log: log}, nil
}

// newVerifierForTest is the test-only constructor used by verifier_test.go.
// Production code must not call it.
func newVerifierForTest(eng verifierEngine) *Verifier {
	return &Verifier{engine: eng, loader: &loader.Loader{}, log: logger.New("verifier-test")}
}

// Verify runs every InvalidExample on every rule. Returns a multi-error
// (aggregating all failures) so authors can see all problems at once.
func (v *Verifier) Verify(ctx context.Context, doc *RulesDoc) error {
	var errs []error
	for _, rule := range doc.Rules {
		for _, mb := range rule.MatchBehavior {
			for _, ex := range mb.Invalid {
				if err := v.verifyExample(ctx, ex, rule.RuleID); err != nil {
					errs = append(errs, err)
				}
			}
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return aggregateErrors(errs)
}

func (v *Verifier) verifyExample(ctx context.Context, ex InvalidExample, ruleID string) error {
	tmp, err := os.MkdirTemp("", "rulesdoc-verify-*")
	if err != nil {
		return fmt.Errorf("rule %s example %s: mktemp: %w", ruleID, ex.ExampleID, err)
	}
	defer os.RemoveAll(tmp)

	for relPath, body := range ex.Files {
		full := filepath.Join(tmp, relPath)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return fmt.Errorf("rule %s example %s: mkdir %s: %w", ruleID, ex.ExampleID, full, err)
		}
		if err := os.WriteFile(full, []byte(body), 0o600); err != nil {
			return fmt.Errorf("rule %s example %s: write %s: %w", ruleID, ex.ExampleID, full, err)
		}
	}

	rawSpecs, err := v.loader.Load(tmp)
	if err != nil {
		return fmt.Errorf("rule %s example %s: load: %w", ruleID, ex.ExampleID, err)
	}
	// Parse each raw spec so the engine's PathIndexer / Parsed() works.
	for _, rs := range rawSpecs {
		if _, err := rs.Parse(); err != nil {
			return fmt.Errorf("rule %s example %s: parse: %w", ruleID, ex.ExampleID, err)
		}
	}

	produced, err := v.engine.ValidateSyntax(ctx, rawSpecs)
	if err != nil {
		return fmt.Errorf("rule %s example %s: validate: %w", ruleID, ex.ExampleID, err)
	}

	// Strip tmpdir prefix from produced diagnostic file paths so they
	// align with the relative paths in authored Files map.
	normalized := normalizeDiagnostics(produced, tmp)

	for _, exp := range ex.ExpectedDiagnostics {
		if !matchesAny(exp, normalized) {
			return fmt.Errorf(
				"rule %s example %s: expected diagnostic not produced — file=%s reference=%s severity=%s message_contains=%q",
				ruleID, ex.ExampleID, exp.File, exp.Reference, exp.Severity, exp.MessageContains,
			)
		}
	}
	return nil
}

func normalizeDiagnostics(diags validation.Diagnostics, tmp string) validation.Diagnostics {
	out := make(validation.Diagnostics, 0, len(diags))
	for _, d := range diags {
		nd := d
		rel, err := filepath.Rel(tmp, d.File)
		if err == nil {
			nd.File = rel
		}
		out = append(out, nd)
	}
	return out
}

func matchesAny(exp ExpectedDiagnostic, produced validation.Diagnostics) bool {
	for _, d := range produced {
		if d.Severity.String() != exp.Severity {
			continue
		}
		if d.File != exp.File {
			continue
		}
		// Reference comparison: produced diagnostics carry the reference
		// in the diagnostic itself (via Position lookup) — they don't carry
		// the original Reference field, but the engine stores the rule ID;
		// we don't currently compare reference here because Diagnostic does
		// not retain it. Use the message_contains check as the discriminator
		// when Reference is provided.
		//
		// FUTURE: extend Diagnostic to carry the original Reference, then
		// drop this gap. For the spike, message_contains substring is
		// sufficient because each example's expected diagnostics are
		// already keyed by the file they live in.
		//
		// NB: keeping the comment so reviewers see the trade-off.
		_ = exp.Reference
		if exp.MessageContains != "" && !strings.Contains(d.Message, exp.MessageContains) {
			continue
		}
		return true
	}
	return false
}

func aggregateErrors(errs []error) error {
	if len(errs) == 1 {
		return errs[0]
	}
	msgs := make([]string, len(errs))
	for i, e := range errs {
		msgs[i] = "- " + e.Error()
	}
	return fmt.Errorf("verifier found %d failure(s):\n%s", len(errs), strings.Join(msgs, "\n"))
}
```

> **Reference-field gap:** the spec's `ExpectedDiagnostic.Reference` cannot be exactly matched against produced `Diagnostic` because `Diagnostic` doesn't preserve the original `Reference` (it's resolved to a `Position`). For the spike, the verifier uses `(Severity, File, MessageContains)` as the match key and treats `Reference` as authoring metadata — the assumption is that within a single (file, severity, substring) tuple, references uniquely disambiguate when authors are careful. **Document this in a `# TODO` comment in the spike PR description as a known limitation.** If this proves to be a real blocker during pilot authoring, the path forward is to add a `Reference` field to `validation.Diagnostic` and propagate it from `ValidationResult` — a one-file change in `engine.go`.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cli/internal/validation/docs/ -run TestVerifier -v`
Expected: PASS for both test cases.

- [ ] **Step 5: Run full package tests + lint**

Run: `go test ./cli/internal/validation/docs/... -v && make lint`
Expected: all green. If lint complains about unused parameters, address minimally.

- [ ] **Step 6: Commit**

```bash
git add cli/internal/validation/docs/verifier.go cli/internal/validation/docs/verifier_test.go
git commit -m "feat(validation/docs): add Verifier that subset-matches example diagnostics

For each authored InvalidExample, the Verifier materializes its Files map
into a tmpdir, loads the dir through the project loader, runs
ValidationEngine.ValidateSyntax, and asserts every ExpectedDiagnostic
matches at least one produced diagnostic on (Severity, File,
MessageContains substring). Extra produced diagnostics are ignored
(subset semantics — see spec §5).

Spike scope: runs ValidateSyntax only — the three pilot rules execute in
the syntactic phase (duplicate-urn as a ProjectRule). Semantic-phase
verification is a follow-up if a future pilot needs it.

Known gap: ExpectedDiagnostic.Reference is not compared because
validation.Diagnostic does not retain the original reference (it resolves
to Position at engine time). For the spike, (Severity, File,
MessageContains) is the match key. If pilot authoring shows this
ambiguity is a problem, add Reference to validation.Diagnostic.

Refs DEX-371."
```

---

## Task 5: Serializer — emit JSON + YAML

**Why:** Hugo consumes the YAML; LLMs consume the JSON. Both must be stable, reproducible artifacts (no timestamps).

**Files:**
- Create: `cli/internal/validation/docs/serializer.go`
- Create: `cli/internal/validation/docs/serializer_test.go`

- [ ] **Step 1: Write the failing test**

Create `cli/internal/validation/docs/serializer_test.go`:

```go
package docs

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func sampleDoc() *RulesDoc {
	return &RulesDoc{
		SchemaVersion: 1,
		ToolMetadata:  ToolMetadata{CLIVersion: "test-1.0"},
		Rules: []ResolvedRule{
			{
				RuleID:      "rule-a",
				Provider:    "test",
				Phase:       "syntactic",
				Severity:    "error",
				Description: "the rule",
				AppliesTo:   []MatchPatternDoc{{Kind: "source", Version: "v1"}},
				MatchBehavior: []MatchBehaviorEntry{
					{AppliesTo: []MatchPatternDoc{{Kind: "source", Version: "v1"}}},
				},
			},
		},
	}
}

func TestEmitYAML_RoundTrips(t *testing.T) {
	dir := t.TempDir()
	doc := sampleDoc()

	path, err := EmitYAML(doc, dir)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "rules.yaml"), path)

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var back RulesDoc
	require.NoError(t, yaml.Unmarshal(data, &back))
	assert.Equal(t, doc, &back)
}

func TestEmitJSON_RoundTrips(t *testing.T) {
	dir := t.TempDir()
	doc := sampleDoc()

	path, err := EmitJSON(doc, dir)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "rules.json"), path)

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var back RulesDoc
	require.NoError(t, json.Unmarshal(data, &back))
	// JSON tags fall back to Go field names; the round-trip test only
	// validates the value survives, not byte-equality.
	assert.Equal(t, doc.SchemaVersion, back.SchemaVersion)
	require.Len(t, back.Rules, len(doc.Rules))
	assert.Equal(t, doc.Rules[0].RuleID, back.Rules[0].RuleID)
}

func TestEmitYAML_CreatesOutputDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "subdir")
	_, err := EmitYAML(sampleDoc(), dir)
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(dir, "rules.yaml"))
	require.NoError(t, err)
}
```

> **Note on JSON tags:** `RulesDoc` / `ResolvedRule` etc. carry only `yaml:` struct tags. JSON encoding uses the Go field names by default (`SchemaVersion`, not `schema_version`). For an LLM-consumed JSON, snake_case keys are nicer but not required by the spike's acceptance criteria. **Decision: skip adding `json:` tags in this task** — the spike emits valid JSON; downstream consumers can wrap or transform if needed. If `make test` complains about something hard to satisfy without `json:` tags, add them then; otherwise leave them out to minimize the diff.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cli/internal/validation/docs/ -run TestEmit -v`
Expected: FAIL with "EmitYAML undefined" / "EmitJSON undefined".

- [ ] **Step 3: Implement the serializer**

Create `cli/internal/validation/docs/serializer.go`:

```go
package docs

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// EmitYAML marshals the RulesDoc to YAML and writes it to <dir>/rules.yaml.
// Returns the path written. Creates dir if it doesn't exist.
func EmitYAML(doc *RulesDoc, dir string) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("creating output dir %s: %w", dir, err)
	}
	path := filepath.Join(dir, "rules.yaml")
	data, err := yaml.Marshal(doc)
	if err != nil {
		return "", fmt.Errorf("marshalling YAML: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", fmt.Errorf("writing %s: %w", path, err)
	}
	return path, nil
}

// EmitJSON marshals the RulesDoc to indented JSON and writes it to
// <dir>/rules.json. Returns the path written.
func EmitJSON(doc *RulesDoc, dir string) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("creating output dir %s: %w", dir, err)
	}
	path := filepath.Join(dir, "rules.json")
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshalling JSON: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", fmt.Errorf("writing %s: %w", path, err)
	}
	return path, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cli/internal/validation/docs/ -run TestEmit -v`
Expected: PASS for all three test cases.

- [ ] **Step 5: Run full package tests + lint**

Run: `go test ./cli/internal/validation/docs/... -v && make lint`
Expected: all green.

- [ ] **Step 6: Commit**

```bash
git add cli/internal/validation/docs/serializer.go cli/internal/validation/docs/serializer_test.go
git commit -m "feat(validation/docs): add JSON + YAML serializer

EmitYAML / EmitJSON marshal the RulesDoc and write to <dir>/rules.yaml
and <dir>/rules.json. Both create the output directory if missing. No
GeneratedAt timestamp is emitted — the artifact is checked into git and
must be reproducible.

Refs DEX-371."
```

---

## Task 6: CLI command — `rudder-cli docs rules`

**Why:** Final wiring step. The command builds the registry the same way `project.Project.registry()` does, wires the Generator+Resolver+Verifier+Serializer, and writes both outputs.

**Files:**
- Create: `cli/internal/cmd/docs/docs.go`
- Create: `cli/internal/cmd/docs/rules/rules.go`
- Create: `cli/internal/cmd/docs/rules/rules_test.go`
- Modify: `cli/internal/cmd/root.go`

### Decision: registry construction

The CLI command needs a registry populated with all rules — same shape as `project.registry()` in `cli/internal/project/project.go:309-342`. **Reuse path:** factor `project.registry()` is not yet exported; we have two options:
1. Build the registry inline in the docs command (duplicates ~30 lines).
2. Export a `BuildRegistry(provider, opts)` function from the `project` package.

**Decision:** for the spike, **inline duplication** is fine. Refactoring `project.registry()` into a public helper is a clean follow-up if Path A wins. Keep the duplication explicit and short.

### Decision: provider for registry construction

`project.registry()` calls `p.provider.ParseSpec` and `p.provider.SupportedKinds()` — it depends on `provider.Provider`. The CLI command can get one via `app.NewDeps().CompositeProvider()`, matching the validate command's pattern (`cli/internal/cmd/project/validate/validate.go:42-49`).

### Decision: `--strict-verify` flag

Per spec §11 / §12 question 4, the flag is named but unimplemented. When `--strict-verify` is set, the command returns an error: `"strict-verify mode is not implemented in the spike (DEX-371); only subset mode is supported"`.

- [ ] **Step 1: Write a smoke test for the command**

Create `cli/internal/cmd/docs/rules/rules_test.go`:

```go
package rules

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCmd_HelpRunsCleanly(t *testing.T) {
	cmd := NewCmdRules()
	cmd.SetArgs([]string{"--help"})
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	require.NoError(t, cmd.Execute())
	out := buf.String()
	assert.Contains(t, out, "--fragments-dir")
	assert.Contains(t, out, "--output-dir")
	assert.Contains(t, out, "--strict-verify")
}

func TestCmd_StrictVerifyReturnsUnimplementedError(t *testing.T) {
	cmd := NewCmdRules()
	cmd.SetArgs([]string{"--strict-verify"})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "strict-verify")
	assert.Contains(t, err.Error(), "not implemented")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cli/internal/cmd/docs/rules/ -v`
Expected: FAIL (package doesn't exist).

- [ ] **Step 3: Implement the docs subcommand group**

Create `cli/internal/cmd/docs/docs.go`:

```go
package docs

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/docs/rules"
	"github.com/spf13/cobra"
)

// NewCmdDocs returns the `docs` command group — currently only `rules`,
// but designed to accommodate additional doc-generation subcommands later.
func NewCmdDocs() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "docs <command>",
		Short: "Generate documentation artifacts from registered metadata",
	}
	cmd.AddCommand(rules.NewCmdRules())
	return cmd
}
```

- [ ] **Step 4: Implement the rules subcommand**

Create `cli/internal/cmd/docs/rules/rules.go`:

```go
package rules

import (
	"context"
	"fmt"
	"os"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	prules "github.com/rudderlabs/rudder-iac/cli/internal/project/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/ui"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/docs"
	vrules "github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/spf13/cobra"
)

const defaultFragmentsDir = "cli/internal/validation/docs/fragments"
const defaultOutputDir = "docs/generated"

var log = logger.New("docs-rules")

// NewCmdRules wires the `rudder-cli docs rules` subcommand.
func NewCmdRules() *cobra.Command {
	var (
		fragmentsDir string
		outputDir    string
		strictVerify bool
	)

	cmd := &cobra.Command{
		Use:   "rules",
		Short: "Generate the validation rules documentation artifact",
		Long: heredoc.Doc(`
			Generates rules.yaml and rules.json describing every registered
			validation rule with authored docs. Hugo consumes the YAML to render
			markdown for the public docs site; the JSON is intended for LLMs.

			Every authored invalid example is verified at generation time by
			running it through the validation engine and asserting that the
			authored expected diagnostics are produced (subset semantics).
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			if strictVerify {
				return fmt.Errorf("--strict-verify is not implemented in the spike (DEX-371); only subset mode is supported")
			}
			return run(cmd.Context(), fragmentsDir, outputDir)
		},
	}

	cmd.Flags().StringVar(&fragmentsDir, "fragments-dir", defaultFragmentsDir, "Directory containing rule doc YAML fragments")
	cmd.Flags().StringVar(&outputDir, "output-dir", defaultOutputDir, "Directory to write rules.yaml and rules.json into")
	cmd.Flags().BoolVar(&strictVerify, "strict-verify", false, "Use exact-match verification (not implemented in the spike)")

	return cmd
}

func run(ctx context.Context, fragmentsDir, outputDir string) error {
	deps, err := app.NewDeps()
	if err != nil {
		return fmt.Errorf("initialising dependencies: %w", err)
	}

	reg, err := buildRegistry(deps)
	if err != nil {
		return fmt.Errorf("building registry: %w", err)
	}

	resolver, err := docs.NewYAMLResolver(os.DirFS("."), fragmentsDir)
	if err != nil {
		return fmt.Errorf("creating YAML resolver: %w", err)
	}

	gen := docs.NewGenerator(reg, resolver, docs.GeneratorOptions{
		CLIVersion:    app.GetVersion(),
		SchemaVersion: 1,
	})
	doc, err := gen.Generate()
	if err != nil {
		return fmt.Errorf("generating rules doc: %w", err)
	}

	if errs := doc.Validate(nil); len(errs) > 0 {
		return fmt.Errorf("structural validation failed: %v", errs)
	}

	verifier, err := docs.NewVerifier(reg, log)
	if err != nil {
		return fmt.Errorf("creating verifier: %w", err)
	}
	if err := verifier.Verify(ctx, doc); err != nil {
		return fmt.Errorf("executable verification failed: %w", err)
	}

	yamlPath, err := docs.EmitYAML(doc, outputDir)
	if err != nil {
		return fmt.Errorf("emitting YAML: %w", err)
	}
	jsonPath, err := docs.EmitJSON(doc, outputDir)
	if err != nil {
		return fmt.Errorf("emitting JSON: %w", err)
	}

	ui.PrintSuccess(fmt.Sprintf("Wrote %s and %s", yamlPath, jsonPath))
	return nil
}

// buildRegistry mirrors project.registry() inline so the docs command can run
// without depending on a full project load. Duplication is intentional for the
// spike — see plan §6 commit-decision notes.
func buildRegistry(deps app.Deps) (vrules.Registry, error) {
	provider := deps.CompositeProvider()
	reg := vrules.NewRegistry()

	validVersions := []string{specs.SpecVersionV0_1, specs.SpecVersionV0_1Variant, specs.SpecVersionV1}

	syntactic := []vrules.Rule{
		prules.NewSpecSyntaxValidRule(provider.SupportedKinds(), validVersions),
		prules.NewMetadataSyntaxValidRule(provider.ParseSpec, validVersions),
		prules.NewDuplicateURNRule(provider.ParseSpec),
	}
	syntactic = append(syntactic, provider.SyntacticRules()...)
	for _, r := range syntactic {
		reg.RegisterSyntactic(r)
	}

	for _, r := range provider.SemanticRules() {
		reg.RegisterSemantic(r)
	}

	return reg, nil
}
```

- [ ] **Step 5: Wire into root command**

In `cli/internal/cmd/root.go`, add the import and registration. Add to the imports block:

```go
"github.com/rudderlabs/rudder-iac/cli/internal/cmd/docs"
```

Add in `init()` near other `rootCmd.AddCommand(...)` calls:

```go
rootCmd.AddCommand(docs.NewCmdDocs())
```

- [ ] **Step 6: Run smoke test + lint**

Run: `go test ./cli/internal/cmd/docs/... -v && make lint`
Expected: smoke tests pass, lint clean.

- [ ] **Step 7: Build the binary and smoke-test the help**

Run: `make build && ./bin/rudder-cli docs rules --help`
Expected: prints the help with the three flags. No panic.

- [ ] **Step 8: Commit**

```bash
git add cli/internal/cmd/docs cli/internal/cmd/root.go
git commit -m "feat(cmd/docs/rules): add rudder-cli docs rules subcommand

New 'docs' command group with a 'rules' subcommand that runs the full
RuleDoc pipeline: builds the registry from the composite provider, loads
YAML fragments via YAMLResolver, generates the RulesDoc, runs structural
validation + executable verification, and emits both rules.yaml and
rules.json to the output dir.

Flags:
  --fragments-dir  default cli/internal/validation/docs/fragments
  --output-dir     default docs/generated
  --strict-verify  named in help but returns 'not implemented' error
                   when invoked (spike scope)

Refs DEX-371."
```

---

## Task 7: Three pilot YAML fragments

**Why:** The acceptance criterion in spec §13 requires three pilot rules authored end-to-end. These fragment files are the entire \"data\" side of Path A.

**Files:**
- Create: `cli/internal/validation/docs/fragments/datacatalog-categories-spec-syntax-valid.docs.yaml`
- Create: `cli/internal/validation/docs/fragments/project-metadata-syntax-valid.docs.yaml`
- Create: `cli/internal/validation/docs/fragments/project-duplicate-urn.docs.yaml`

### Fragment authoring conventions

- File name: `<rule-id-with-slashes-replaced-by-dashes>.docs.yaml`. Examples: `datacatalog-categories-spec-syntax-valid.docs.yaml` (for rule ID `datacatalog/categories/spec-syntax-valid`).
- One file = one rule (one `rule_id`).
- `MatchPatternDoc.Version` for categories rule uses both legacy variants (`rudder/v0.1` plus the variant) and `rudder/v1`.
- The valid examples should be runnable through the same loader and not produce diagnostics from the rule being documented; the invalid examples must produce at least one diagnostic that matches each authored `ExpectedDiagnostic`.

> **Tip during authoring:** keep each file under ~70 lines so the data-cap of 200 lines/3 rules holds.

### Pilot 1: `datacatalog-categories-spec-syntax-valid.docs.yaml`

- [ ] **Step 1: Identify the rule's behavior**

Read `cli/internal/providers/datacatalog/rules/category/category_spec_valid.go` to understand:
- Rule ID: `datacatalog/categories/spec-syntax-valid`
- Severity: `error`
- Applies to: `categories` kind across legacy versions + `rudder/v1`.
- Validation: `rules.ValidateStruct` on `localcatalog.CategorySpec` (legacy) or `localcatalog.CategorySpecV1` (v1). Missing `id` or `name` is the typical failure.

You will also need to read `cli/internal/providers/datacatalog/localcatalog/category_spec.go` (or equivalent) to confirm the struct shape and the exact JSON reference path for missing fields.

- [ ] **Step 2: Write the fragment**

```yaml
rule_id: datacatalog/categories/spec-syntax-valid
match_behavior:
  - applies_to:
      - { kind: categories, version: rudder/v0.1 }
    valid:
      - example_id: legacy-basic
        title: Legacy categories with id and name
        files:
          main.yaml: |
            version: rudder/v0.1
            kind: categories
            spec:
              categories:
                - id: user_actions
                  name: User Actions
    invalid:
      - example_id: legacy-missing-id
        title: Legacy category without id is rejected
        files:
          main.yaml: |
            version: rudder/v0.1
            kind: categories
            spec:
              categories:
                - name: Missing ID
        expected_diagnostics:
          - file: main.yaml
            reference: /categories/0/id
            severity: error
            message_contains: "id"
      - example_id: legacy-missing-name
        title: Legacy category without name is rejected
        files:
          main.yaml: |
            version: rudder/v0.1
            kind: categories
            spec:
              categories:
                - id: missing_name
        expected_diagnostics:
          - file: main.yaml
            reference: /categories/0/name
            severity: error
            message_contains: "name"

  - applies_to:
      - { kind: categories, version: rudder/v1 }
    valid:
      - example_id: v1-basic
        title: v1 categories with id and name
        files:
          main.yaml: |
            version: rudder/v1
            kind: categories
            metadata:
              name: my-project
            spec:
              categories:
                - id: user_actions
                  name: User Actions
    invalid:
      - example_id: v1-missing-name
        title: v1 category without name is rejected
        files:
          main.yaml: |
            version: rudder/v1
            kind: categories
            metadata:
              name: my-project
            spec:
              categories:
                - id: only_id
        expected_diagnostics:
          - file: main.yaml
            reference: /categories/0/name
            severity: error
            message_contains: "name"
```

> **If the exact JSON Pointer references differ:** consult the rule's actual output in `funcs.ParseValidationErrors`. The plan author chose `/categories/0/name` based on the existing rule conventions; the executor may need to adjust based on what the engine actually emits.

### Pilot 2: `project-metadata-syntax-valid.docs.yaml`

- [ ] **Step 1: Identify the rule's behavior**

Read `cli/internal/project/rules/metadata_syntax_valid.go`:
- Rule ID: `project/metadata-syntax-valid`
- Severity: `error`
- Applies to: all kinds via `MatchPattern{Kind: "*", Version: v}` for each registered version. **Important:** `AppliesTo` is wildcard `Kind`, specific `Version`. The fragment should mirror that with `{kind: "*", version: rudder/v0.1}` etc.
- Validation: decodes metadata, runs `rules.ValidateStruct(metadata, "/metadata")`. Failures: missing `name`, missing `workspace_id` in imports, etc.

- [ ] **Step 2: Write the fragment**

```yaml
rule_id: project/metadata-syntax-valid
match_behavior:
  - applies_to:
      - { kind: "*", version: rudder/v1 }
    valid:
      - example_id: basic-metadata
        title: Simple metadata with name
        files:
          main.yaml: |
            version: rudder/v1
            kind: categories
            metadata:
              name: my-project
            spec:
              categories: []
      - example_id: metadata-with-import
        title: Metadata with import block
        files:
          main.yaml: |
            version: rudder/v1
            kind: categories
            metadata:
              name: my-project
              import:
                workspaces:
                  - workspace_id: ws-123
                    resources:
                      - urn: urn:rudder:category:user_actions
            spec:
              categories:
                - id: user_actions
                  name: User Actions
    invalid:
      - example_id: missing-name
        title: Metadata without name is rejected
        files:
          main.yaml: |
            version: rudder/v1
            kind: categories
            metadata:
              import:
                workspaces:
                  - workspace_id: ws-123
            spec:
              categories: []
        expected_diagnostics:
          - file: main.yaml
            reference: /metadata/name
            severity: error
            message_contains: "name"
      - example_id: missing-workspace-id
        title: Import workspaces entry without workspace_id is rejected
        files:
          main.yaml: |
            version: rudder/v1
            kind: categories
            metadata:
              name: my-project
              import:
                workspaces:
                  - resources:
                      - urn: urn:rudder:category:x
            spec:
              categories: []
        expected_diagnostics:
          - file: main.yaml
            reference: /metadata/import/workspaces/0/workspace_id
            severity: error
            message_contains: "workspace_id"
```

### Pilot 3: `project-duplicate-urn.docs.yaml`

- [ ] **Step 1: Identify the rule's behavior**

Read `cli/internal/project/rules/duplicate_urn_rule.go`:
- Rule ID: `project/duplicate-urn`
- Severity: `error`
- Applies to: `MatchAll()` — wildcard kind, wildcard version.
- Validation: cross-file. Two files defining the same URN flag both occurrences. Spec §8 calls out: "expected diagnostic points at the second file's offending position."

- [ ] **Step 2: Write the fragment**

```yaml
rule_id: project/duplicate-urn
match_behavior:
  - applies_to:
      - { kind: "*", version: "*" }
    valid:
      - example_id: distinct-urns
        title: Two files with distinct URNs
        files:
          a.yaml: |
            version: rudder/v1
            kind: categories
            metadata:
              name: a
            spec:
              categories:
                - id: cat_a
                  name: A
          b.yaml: |
            version: rudder/v1
            kind: categories
            metadata:
              name: b
            spec:
              categories:
                - id: cat_b
                  name: B
    invalid:
      - example_id: duplicate-urn-across-files
        title: Two files defining the same category URN are both flagged
        files:
          a.yaml: |
            version: rudder/v1
            kind: categories
            metadata:
              name: a
            spec:
              categories:
                - id: shared_id
                  name: A
          b.yaml: |
            version: rudder/v1
            kind: categories
            metadata:
              name: b
            spec:
              categories:
                - id: shared_id
                  name: B
        expected_diagnostics:
          - file: a.yaml
            reference: /categories/0
            severity: error
            message_contains: "duplicate URN"
          - file: b.yaml
            reference: /categories/0
            severity: error
            message_contains: "duplicate URN"
```

- [ ] **Step 3: Verify the artifact end-to-end**

Run from the worktree root:

```bash
make build && ./bin/rudder-cli docs rules
```

Expected:
1. Command exits 0.
2. `docs/generated/rules.yaml` exists with three entries.
3. `docs/generated/rules.json` exists with the same data.
4. Stdout: `Wrote docs/generated/rules.yaml and docs/generated/rules.json`.

If a pilot fragment fails verification (e.g., expected diagnostic substring doesn't match produced message), tune the `message_contains` substring and/or `reference` path based on the actual produced output. Run the command iteratively until clean.

- [ ] **Step 4: Run full test suite + lint**

Run: `make test && make lint`
Expected: all green.

- [ ] **Step 5: Check the size caps (spec §13)**

Run from worktree root:

```bash
# Non-test, non-data infrastructure code
find cli/internal/validation/docs cli/internal/cmd/docs -name '*.go' ! -name '*_test.go' | xargs wc -l
# Plus the additions to validation/rules/registry.go (just the new methods)
# Plus the cmd/root.go addition (one line + one import)
```

Sum should be **< 500 lines**. Spike infrastructure includes: `resolver.go`, `generator.go`, `verifier.go`, `serializer.go`, the rename diff in `types.go`/`rules_doc.go`, the registry-extension diff in `registry.go`, `cli/internal/cmd/docs/docs.go`, `cli/internal/cmd/docs/rules/rules.go`, and the root.go wiring.

Run:

```bash
wc -l cli/internal/validation/docs/fragments/*.yaml
```

Sum should be **< 200 lines**.

If either cap is exceeded, trim. Common culprits: over-documented YAML, redundant test fixtures, verbose comments in `verifier.go`. The plan-author's intent is that the verifier sits at ~150 LoC; if you've ballooned to 250, look for ways to compress.

- [ ] **Step 6: Generate the artifact and commit it**

Per the spec, the emitted artifact is committed to `docs/generated/`. Run the command again to ensure the artifact is up to date, then add it.

```bash
./bin/rudder-cli docs rules
git add cli/internal/validation/docs/fragments docs/generated
git commit -m "feat(validation/docs): author three pilot YAML fragments + initial generated artifact

Pilots:
- datacatalog/categories/spec-syntax-valid (per-version: legacy v0.1 + v1)
- project/metadata-syntax-valid (applies-to-all via wildcard kind)
- project/duplicate-urn (multi-file invalid example)

Each fragment exercises a distinct facet of the pipeline — per-version
match patterns, wildcard match patterns, and cross-file ProjectRules —
proving the pipeline handles the matrix.

Includes the initial docs/generated/{rules.yaml,rules.json} so reviewers
see the artifact shape. Subsequent updates regenerate via:
  rudder-cli docs rules

Refs DEX-371."
```

---

## Task 8: Final verification + PR description

- [ ] **Step 1: Run the full test suite one more time**

Run: `make test && make lint`
Expected: all green.

- [ ] **Step 2: Verify acceptance criteria (spec §13)**

Walk through each bullet of §13 and confirm:

1. `rudder-cli docs rules` runs cleanly with default flags — verified in Task 7 step 3.
2. The artifact passes `RulesDoc.Validate(nil)` — the command itself runs this check inline.
3. The Verifier runs every authored invalid example against the engine — verified by Task 4 + the artifact regeneration in Task 7.
4. All three pilot rules authored end-to-end — Task 7 fragments.
5. `make test` + `make lint` green — Task 8 step 1.
6. Code cap < 500 lines (non-test, non-data) — Task 7 step 5.
7. Authored-data cap < 200 lines — Task 7 step 5.

- [ ] **Step 3: Update the PR description**

Use `.github/pull_request_template.md` as the structure. The description must call out:

- **Linear ticket:** [DEX-371](https://linear.app/rudderstack/issue/DEX-371)
- **Carve-outs landed locally (not blocking):**
  - `RulesDoc` / `ResolvedRule` rename (would have come from PR #475).
  - `Registry.AllSyntacticRules()` / `AllSemanticRules()` (would have come from a #475 carve-out PR per spec §4).
- **Known limitations:**
  - Verifier compares on `(Severity, File, MessageContains)`; `Reference` field is authoring metadata only because `validation.Diagnostic` does not retain the original reference (spec §6 doesn't mandate it). Path forward documented in `verifier.go`.
  - `--strict-verify` is named but unimplemented (intentional, spec §11).
  - `validateRegisteredCompleteness` is commented out (intentional, spec §3 Layer 3).
  - Verifier runs only `ValidateSyntax` — the three pilots are all reachable from the syntactic phase (`duplicate-urn` is a `ProjectRule`). Future pilots needing semantic-phase verification will require an extension.

- [ ] **Step 4: Stop**

Do NOT push to remote. Do NOT open the PR. The executor session takes over from here.

---

## Glossary / pointers

- **Resolver seam** (spec §2): the only interface that differs between Path A and Path B.
- **Subset semantics** (spec §5): every authored expectation must appear in produced output; extras are ignored.
- **PR #471 baseline:** the foundation this branch sits on — see `git log --oneline -10` for the three commits already in place.
- **`project.registry()`:** the inline template for `buildRegistry` in the CLI command (`cli/internal/project/project.go:309-342`).
- **Existing `Examples()` runtime mechanism:** spec §6 calls out that the existing `rules.Rule.Examples()` method — consumed by `validation/renderer/text.go` — is left untouched by both spikes. Don't touch it.

---

## Open questions resolved during planning

- **Q: Should the rename to `RulesDoc`/`ResolvedRule` block on a carve-out from #475?** Resolved: no — done in Task 1.
- **Q: Should the registry extensions block on a carve-out from #475?** Resolved: no — done in Task 1.
- **Q: How should `--strict-verify` behave?** Resolved: returns a clear "not implemented" error when invoked.
- **Q: Where does the docs CLI command live?** Resolved: `cli/internal/cmd/docs/rules/`.
- **Q: How does the verifier compare diagnostics when `Diagnostic` lacks a `Reference` field?** Resolved: subset match keys on `(Severity, File, MessageContains)`; `Reference` is authoring metadata. Documented in `verifier.go`.
- **Q: Should the verifier run `ValidateSemantic`?** Resolved: spike runs syntactic only — all three pilots are syntactic-phase. Documented in `verifier.go` and the spike PR description.
- **Q: Should JSON output have snake_case keys?** Resolved: skipped for the spike to keep diff minimal. If LLM consumers need it, add `json:` tags in a follow-up.
- **Q: Should the verifier produce one error or many?** Resolved: many — aggregated into a single multi-error so authors see all failures at once.

---

## Risk callouts

- **Highest-risk task: Task 4 (Verifier).** The reference-field gap and project-loader reuse are both wired to existing internals that may not behave as the plan assumes. Mitigation: the tests in Task 4 use a fake engine, so the loader interaction is exercised end-to-end only in Task 7 step 3. Be prepared to iterate on path normalization and message substrings during Task 7.
- **Second-highest: Task 7 fragment authoring.** Expected references like `/metadata/name` depend on what `ParseValidationErrors` actually emits — they may need adjustment based on observed output. Mitigation: the verifier's substring match is forgiving; you can usually find a substring that matches whatever the engine produces.
- **Lower risk: Task 1 rename.** Mechanical; tests cover it.

---

## Done criteria

- [ ] All eight tasks above are committed with passing tests and clean lint.
- [ ] `rudder-cli docs rules` runs from the worktree root with default flags and produces `docs/generated/rules.{yaml,json}` describing all three pilot rules.
- [ ] Spec §13 acceptance criteria all check off (Task 8 step 2).
- [ ] No files outside the worktree are modified.
- [ ] No remote push and no PR creation — the executor session handles those.
