# DEX-370 Path B — Documented Interface Spike Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build an end-to-end rule-doc pipeline (Resolver → Generator → Verifier → Serializer → CLI) for the DEX-370 spike using a separate opt-in `Documented` interface in the `docs` package. Three pilot rules implement `DocExamples()` inline in Go.

**Architecture:** Trim PR #471 to Path B essentials (delete the YAML-authoring surface, drop two struct fields, dissolve the `docs → rules` import dep so `rules → docs` becomes possible without a cycle). Add a `Documented` opt-in interface in `cli/internal/validation/docs/`. Implement an `ExamplesResolver` that type-asserts rules to `Documented`. Build the shared Generator / Verifier / Serializer pipeline. Add a new top-level `rudder-cli docs rules` Cobra subcommand that walks the registry, resolves doc data, structurally validates, executably verifies, and emits JSON + YAML to `./docs/generated/`. Three pilot rules gain `DocExamples()` methods.

**Tech Stack:** Go 1.x, Cobra (CLI), go-playground/validator/v10 (structural), gopkg.in/yaml.v3, encoding/json, testify, MakeNowJust/heredoc/v2 (multi-line literals).

---

## Spec Reference

Source-of-truth spec: `/Users/shanmukh/workspace/rudder-iac/docs/superpowers/specs/2026-05-26-rulesdoc-generator-spikes-design.md` (lives on `main` in the main checkout — not in this worktree's branch). Read §§1, 2, 4, 5, 7, 9, 13 before executing tasks.

## Working Directory

All file paths are absolute and rooted at the worktree:
`/Users/shanmukh/workspace/rudder-iac/.worktrees/dex-370-path-b`

Branch: `feature/dex-370-spike-path-b-separate-documented-interface-for-ruledoc` (forked from PR #471's `feat/dex-269-add-docs-foundation`).

## Resolved Open Questions (calls made up-front)

Decisions made without pausing, per the spec's §12 instructions:

1. **Type names — `DocumentedRules`/`DocumentedRule` vs `RulesDoc`/`ResolvedRule`.** The worktree branch currently uses PR #471's `DocumentedRules`/`DocumentedRule` names. The spec assumes the #475 rename to `RulesDoc`/`ResolvedRule` is already in place. The rename is a mechanical cross-package edit that is orthogonal to Path B's authoring strategy. **Decision:** include the rename as the very first task (Task 0). It lands cleanly before any deletions/feature work, gives every subsequent task the spec's naming, and is trivial to revert if the team picks Path A. The rename is purely additive in the diff-size sense (touches existing files, no LoC added).

2. **`AllSyntacticRules()` / `AllSemanticRules()` registry-extension methods.** Spec §4 says these land in a separate PR before the spikes. They are NOT on this branch today. **Decision:** include them as Task 1 (a thin carve-out commit). They are ~20 LoC plus tests; pulling them in here is simpler than blocking on an external PR.

3. **Where the registry comes from inside the CLI command.** The existing `project.registry()` is private and the public path (`app.NewDeps()` → `project.NewProject()` → `Load()`) requires an authenticated client. Generating docs must NOT require credentials. **Decision:** the new `docs rules` command instantiates each provider with a `nil` client (rule constructors don't touch the client) and assembles a `rules.Registry` directly, mirroring `project.registry()`'s shape. Helper lives in the new `cli/internal/cmd/docs/rules/` package, not exported globally.

4. **Verifier project-loader reuse.** Spec §12.3 says "reuse if API is sane; document awkwardness rather than refactor." The existing `loader.Loader` only walks a disk path, not in-memory bytes. **Decision:** Verifier writes the example's `Files` map to a tmpdir, runs `loader.Loader.Load(tmpdir)`, then runs `ValidationEngine.ValidateSyntax`. No `ValidateSemantic` for the spike (none of the three pilot rules are semantic). If a later pilot needs semantic verification we'll extend; not in scope here.

5. **CLI command path.** Spec §12.2 leaves the choice open. **Decision:** `cli/internal/cmd/docs/rules/` (nested package) — matches the existing `cli/internal/cmd/project/{apply,validate,destroy}/` pattern. The parent `cli/internal/cmd/docs/` package owns the `docs` group command.

6. **`--strict-verify` behavior in the spike.** Spec §5 + §12.4: name it in help text, return a clean error when invoked. **Decision:** the flag is registered with help text `"strict-verify mode is not implemented in the spike (tracked in DEX-216 follow-up)"`. When `true`, RunE returns an explicit error before running the pipeline.

7. **Output filenames.** **Decision:** `rules.yaml` and `rules.json` in `--output-dir`. Both files are overwritten on each run. The YAML is what Hugo consumes; the JSON is for LLMs.

8. **Pilot rule type-receivers.** `MetadataSyntaxValidRule` is exported; `duplicateURNRule` is unexported but the constructor returns `rules.Rule` (interface) — `DocExamples()` is implemented on the unexported pointer receiver and the interface assertion via `.(Documented)` will still succeed at runtime. `categorySpecRule` is built via `prules.NewTypedRule(...)` — see Task 9 for the wrinkle this creates.

9. **`categorySpecRule` interface implementation.** `NewCategorySpecSyntaxValidRule` returns `prules.NewTypedRule(...)` — i.e., a generic wrapper type from `cli/internal/provider/rules/`, not a per-rule struct under our control. **Decision:** add a small wrapper struct in `cli/internal/providers/datacatalog/rules/category/` that embeds the typed rule and adds `DocExamples()`. The constructor returns the wrapper instead of the raw typed rule. The wrapper still satisfies `rules.Rule` via embedding. Same wrinkle does not exist for the other two pilots (both have their own dedicated structs).

## File Structure

### Files modified (existing)

- `cli/internal/validation/docs/types.go` — rename + field removals + final shape
- `cli/internal/validation/docs/rules_doc.go` — rename + cycle-breaker + completeness comment-out
- `cli/internal/validation/docs/rules_doc_test.go` — rename + adjust completeness tests
- `cli/internal/validation/rules/registry.go` — add `AllSyntacticRules()`, `AllSemanticRules()`
- `cli/internal/validation/rules/registry_test.go` — tests for the new methods
- `cli/internal/cmd/root.go` — wire new `docs` command tree
- `cli/internal/providers/datacatalog/rules/category/category_spec_valid.go` — pilot 1: wrapper + `DocExamples()`
- `cli/internal/project/rules/metadata_syntax_valid.go` — pilot 2: `DocExamples()`
- `cli/internal/project/rules/duplicate_urn_rule.go` — pilot 3: `DocExamples()`

### Files created (new)

- `cli/internal/validation/docs/documented.go` — `Documented` interface
- `cli/internal/validation/docs/resolver.go` — `Resolver` interface + `ExamplesResolver`
- `cli/internal/validation/docs/resolver_test.go`
- `cli/internal/validation/docs/generator.go` — pipeline orchestrator (walk registry → resolve → enrich → build `RulesDoc`)
- `cli/internal/validation/docs/generator_test.go`
- `cli/internal/validation/docs/verifier.go` — runs `ValidationEngine` on invalid examples, subset-matches diagnostics
- `cli/internal/validation/docs/verifier_test.go`
- `cli/internal/validation/docs/serializer.go` — emits JSON + YAML
- `cli/internal/validation/docs/serializer_test.go`
- `cli/internal/cmd/docs/docs.go` — `docs` group command
- `cli/internal/cmd/docs/rules/rules.go` — `docs rules` Cobra command
- `cli/internal/cmd/docs/rules/rules_test.go` — unit-level table tests for flag wiring + error paths

### Files deleted (existing)

- `cli/internal/validation/docs/rule_doc_entry.go`
- `cli/internal/validation/docs/rule_doc_entry_test.go`

### Files NOT modified (load-bearing leave-alones)

- `cli/internal/validation/rules/rule.go` — `Rule` interface stays as-is. **NO new method.**
- `cli/internal/validation/rules/examples.go` — `Examples` type stays as-is.
- `cli/internal/validation/engine.go` — engine + diagnostic attachment unchanged.
- `cli/internal/provider/provider.go` — `RuleProvider` interface stays as-is. **NO `RuleDocEntries()` method.**
- `cli/internal/validation/rules/struct_validator.go` — `rules.ValidateStruct` keeps existing callers across providers (11+ rule files); only the call inside `docs/rules_doc.go` is replaced.

---

## Test Strategy

- **Unit tests** for every new file (mandatory per project CLAUDE.md). Testify `assert`/`require` only. Prefer whole-struct equality assertions over field-by-field.
- **No E2E test required for the spike.** The new CLI command doesn't touch the apply cycle (which is what triggers `cli/tests/` E2E coverage per CLAUDE.md). Spec §13 acceptance is `make test` + `make lint` green plus a clean CLI invocation; a manual `go run ./cli/cmd/rudder-cli docs rules` against the worktree suffices as the integration check.
- **Generator + Verifier dogfood test:** the generator's main test runs the full pipeline against the three pilot rules and asserts the produced `rules.yaml`/`rules.json` are byte-stable (modulo CLI version field). This is the closest thing to E2E we need.

---

## Estimated diff against acceptance criteria (spec §13)

| Component | Code LoC (non-test, non-data) | Notes |
|---|---|---|
| Task 0 rename | 0 net | mechanical rename only |
| Task 1 registry extension | ~15 | counts toward cap |
| Task 2 trim commit | NEGATIVE | deletes ~70+ LoC; net helps the cap |
| Task 3 `Documented` interface | ~10 | |
| Task 4 Resolver + ExamplesResolver | ~30 | |
| Task 5 Generator | ~80 | |
| Task 6 Verifier | ~120 | the largest piece |
| Task 7 Serializer | ~50 | |
| Task 8 CLI command | ~80 | |
| Task 9 pilot rules (method shells only) | ~30 | DocExamples bodies are AUTHORED DATA, not code |
| **Total infrastructure code (counted toward §13 < 500)** | **~415** with trim deletes counted as negative |
| **Authored data (counted toward §13 < 200)** | **~150** across three pilots |

If the running total approaches 500 during execution, the Verifier (the largest piece) is the place to look for inlining / trimming.

---

## Tasks

---

### Task 0: Rename `DocumentedRules`/`DocumentedRule` → `RulesDoc`/`ResolvedRule`

**Why first:** Aligns the worktree branch with spec naming so every subsequent task references the right types. Mechanical, no behavior change.

**Files:**
- Modify: `cli/internal/validation/docs/types.go`
- Modify: `cli/internal/validation/docs/rules_doc.go`
- Modify: `cli/internal/validation/docs/rules_doc_test.go`

- [ ] **Step 1: Search for all references**

Run:
```bash
cd /Users/shanmukh/workspace/rudder-iac/.worktrees/dex-370-path-b
grep -rn "DocumentedRules\|DocumentedRule" --include='*.go' .
```
Expected: only matches inside `cli/internal/validation/docs/` (3 files). Confirm before editing.

- [ ] **Step 2: Rename `DocumentedRules` → `RulesDoc` and `DocumentedRule` → `ResolvedRule`**

In all three files, do a strict identifier rename. Use `replace_all` Edit operations or `gofmt`-safe sed. Keep all struct tags, comments, and field order identical.

After this step, `types.go`'s shape is:
```go
type RulesDoc struct {
    SchemaVersion int            `yaml:"schema_version"`
    ToolMetadata  ToolMetadata   `yaml:"tool_metadata"`
    Rules         []ResolvedRule `yaml:"rules"`
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
(`Provider` and `ToolMetadata.GeneratedAt` will be removed in Task 2 — keep them for now.)

In `rules_doc.go` the receiver becomes `func (c *RulesDoc) Validate(...)`, and `validateRuleStruct(rule *ResolvedRule)`, `validateAppliesToCoverage(rule *ResolvedRule)`, `validateUniqueExampleIDs(rule *ResolvedRule)`.

In `rules_doc_test.go` the helper becomes `minimalRule(ruleID string) ResolvedRule` returning `ResolvedRule{...}`, and the test bodies construct `&RulesDoc{Rules: []ResolvedRule{...}}`.

- [ ] **Step 3: Build + test**

Run:
```bash
cd /Users/shanmukh/workspace/rudder-iac/.worktrees/dex-370-path-b && go build ./... && go test ./cli/internal/validation/docs/...
```
Expected: PASS (zero behavior change).

- [ ] **Step 4: Commit**

```bash
cd /Users/shanmukh/workspace/rudder-iac/.worktrees/dex-370-path-b
git add cli/internal/validation/docs/
git commit -m "refactor(docs): rename DocumentedRules/Rule to RulesDoc/ResolvedRule"
```

---

### Task 1: Add `AllSyntacticRules()` and `AllSemanticRules()` to the registry

**Why:** Both spikes (per spec §4) depend on these — they let the generator walk every registered rule regardless of (kind, version) match. Pulled in here as a clean carve-out instead of blocking on an external PR.

**Files:**
- Modify: `cli/internal/validation/rules/registry.go`
- Modify: `cli/internal/validation/rules/registry_test.go`

- [ ] **Step 1: Write the failing test** in `registry_test.go`

```go
func TestRegistry_AllSyntacticRules_ReturnsDefensiveCopy(t *testing.T) {
    registry := NewRegistry()
    r1 := &mockRule{id: "a", appliesTo: []MatchPattern{MatchAll()}}
    r2 := &mockRule{id: "b", appliesTo: []MatchPattern{MatchKind("source")}}
    registry.RegisterSyntactic(r1)
    registry.RegisterSyntactic(r2)

    got := registry.AllSyntacticRules()
    require.Len(t, got, 2)
    assert.Equal(t, "a", got[0].ID())
    assert.Equal(t, "b", got[1].ID())

    // mutating the returned slice does not affect future calls
    got[0] = nil
    assert.Equal(t, "a", registry.AllSyntacticRules()[0].ID())
}

func TestRegistry_AllSemanticRules_ReturnsDefensiveCopy(t *testing.T) {
    registry := NewRegistry()
    r := &mockRule{id: "sem", appliesTo: []MatchPattern{MatchAll()}}
    registry.RegisterSemantic(r)

    got := registry.AllSemanticRules()
    require.Len(t, got, 1)
    assert.Equal(t, "sem", got[0].ID())

    got[0] = nil
    assert.Equal(t, "sem", registry.AllSemanticRules()[0].ID())
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd /Users/shanmukh/workspace/rudder-iac/.worktrees/dex-370-path-b
go test ./cli/internal/validation/rules/... -run AllSyntacticRules
```
Expected: FAIL — undefined method `AllSyntacticRules`.

- [ ] **Step 3: Implement** in `registry.go`

Add to the `Registry` interface:
```go
// AllSyntacticRules returns a defensive copy of every syntactic rule,
// regardless of AppliesTo() — used by tooling that needs to enumerate
// the full set (e.g. docs generation).
AllSyntacticRules() []Rule

// AllSemanticRules returns a defensive copy of every semantic rule.
AllSemanticRules() []Rule
```

Add to `defaultRegistry`:
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

- [ ] **Step 4: Run tests**

```bash
go test ./cli/internal/validation/rules/... && go build ./...
```
Expected: PASS. The interface extension may force re-compilation of test mocks elsewhere — if any compile error surfaces, search for other `Registry`-implementing types and add the methods.

- [ ] **Step 5: Commit**

```bash
git add cli/internal/validation/rules/
git commit -m "feat(rules): add AllSyntacticRules/AllSemanticRules to Registry"
```

---

### Task 2: Trim commit — dissolve `docs → rules` dep, delete YAML surface, drop two fields, comment out completeness check

**This is THE load-bearing commit of Path B.** Spec §7 commit 1 in detail.

The single change that justifies the entire commit: `docs/rules_doc.go::validateRuleStruct` currently calls `rules.ValidateStruct(rule, "")`. That call is the ONLY reason `cli/internal/validation/docs/` imports `cli/internal/validation/rules/`. Replacing it with a direct `validator.New()` call dissolves the `docs → rules` dep — which is what lets leaf rule packages (`cli/internal/providers/datacatalog/rules/category/`, `cli/internal/project/rules/`) import `docs` to author `[]docs.MatchBehaviorEntry` literals without creating an import cycle.

**Files:**
- Modify: `cli/internal/validation/docs/types.go`
- Modify: `cli/internal/validation/docs/rules_doc.go`
- Modify: `cli/internal/validation/docs/rules_doc_test.go`
- Delete: `cli/internal/validation/docs/rule_doc_entry.go`
- Delete: `cli/internal/validation/docs/rule_doc_entry_test.go`

- [ ] **Step 1: Delete authored-wrapper files**

```bash
cd /Users/shanmukh/workspace/rudder-iac/.worktrees/dex-370-path-b
git rm cli/internal/validation/docs/rule_doc_entry.go cli/internal/validation/docs/rule_doc_entry_test.go
```

- [ ] **Step 2: Remove `RuleDocEntry` from `types.go`**

In `cli/internal/validation/docs/types.go`, delete the `RuleDocEntry` struct definition entirely (the first struct in the file, lines roughly 5–8 pre-edit). Keep everything else.

- [ ] **Step 3: Remove `ResolvedRule.Provider` and `ToolMetadata.GeneratedAt`**

In `types.go`:
- Remove the `Provider` field from `ResolvedRule` (including its `yaml` + `validate` tags).
- Remove the `GeneratedAt` field from `ToolMetadata` (including its `yaml` tag).

After this step, `types.go`'s resolved-type section reads:
```go
type RulesDoc struct {
    SchemaVersion int            `yaml:"schema_version"`
    ToolMetadata  ToolMetadata   `yaml:"tool_metadata"`
    Rules         []ResolvedRule `yaml:"rules"`
}

type ToolMetadata struct {
    CLIVersion string `yaml:"cli_version"`
}

type ResolvedRule struct {
    RuleID        string               `yaml:"rule_id"        validate:"required"`
    Phase         string               `yaml:"phase"          validate:"required,oneof=syntactic semantic"`
    Severity      string               `yaml:"severity"       validate:"required,oneof=error warning info"`
    Description   string               `yaml:"description"    validate:"required"`
    AppliesTo     []MatchPatternDoc    `yaml:"applies_to"     validate:"required,min=1,dive"`
    MatchBehavior []MatchBehaviorEntry `yaml:"match_behavior" validate:"required,min=1,dive"`
}
```

Authored inner types (`MatchBehaviorEntry`, `MatchPatternDoc`, `ValidExample`, `InvalidExample`, `ExpectedDiagnostic`) are **kept as-is** — they're now reachable only through `ResolvedRule.MatchBehavior` and `Documented.DocExamples()`.

- [ ] **Step 4: Cycle-breaker — replace `rules.ValidateStruct` with `validator.New()` in `rules_doc.go`**

The new top of `rules_doc.go` (replacing the existing `import` block and `Validate` method) should look like:

```go
package docs

import (
    "errors"
    "fmt"
    "reflect"
    "strings"

    "github.com/go-playground/validator/v10"
)

// docValidator is local to the docs package — instantiating here is what
// dissolves the docs → rules import dep that #471 introduced. The leaf
// rule packages can now import docs for MatchBehaviorEntry without an
// import cycle.
var docValidator = func() *validator.Validate {
    v := validator.New()
    v.RegisterTagNameFunc(func(fld reflect.StructField) string {
        name := strings.ToLower(fld.Name)
        if t, ok := fld.Tag.Lookup("yaml"); ok {
            name = strings.SplitN(t, ",", 2)[0]
        }
        return name
    })
    return v
}()

func (c *RulesDoc) Validate(expectedRuleIDs []string) []error {
    var errs []error
    for i := range c.Rules {
        structErrs := validateRuleStruct(&c.Rules[i])
        errs = append(errs, structErrs...)
        if len(structErrs) > 0 {
            // Skip per-rule checks when structure is invalid.
            continue
        }
        errs = append(errs, validateAppliesToCoverage(&c.Rules[i])...)
        errs = append(errs, validateUniqueExampleIDs(&c.Rules[i])...)
    }
    // TODO(spike DEX-370): re-enable after pilot phase. Disabled because the
    // spike only documents 3 of ~13 registered rules.
    // errs = append(errs, c.validateRegisteredCompleteness(expectedRuleIDs)...)
    _ = expectedRuleIDs
    return errs
}

func validateRuleStruct(rule *ResolvedRule) []error {
    if err := docValidator.Struct(rule); err != nil {
        var verrs validator.ValidationErrors
        if !errors.As(err, &verrs) {
            return []error{fmt.Errorf("rule %s: structural validation failed: %w", rule.RuleID, err)}
        }
        out := make([]error, 0, len(verrs))
        for _, fe := range verrs {
            out = append(out, fmt.Errorf("rule %q: field %s failed validation %q", rule.RuleID, fe.Field(), fe.Tag()))
        }
        return out
    }
    return nil
}
```

The three helper funcs `validateUniqueExampleIDs`, `validateAppliesToCoverage`, and `validateRegisteredCompleteness` are copied **unchanged** from the pre-trim file content (same signatures, same bodies — they reference `*ResolvedRule` because of Task 0's rename, not because of this commit). The `validateRegisteredCompleteness` helper STAYS in the file — only its INVOCATION at the bottom of `Validate()` is commented out so we can re-enable it with a one-line change post-spike.

- [ ] **Step 5: Update `rules_doc_test.go` for the rename + completeness comment-out**

Two adjustments:

1. `minimalRule` no longer sets `Provider:` — remove that line and remove the explicit `Provider:` value in the test that checks "valid fully-populated rule passes".
2. The `TestCatalogValidate_RegisteredCompleteness` table — because the call site is commented out, the per-test `wantLen` values would all be 0. Two options: (a) keep the tests, change all `wantLen` to 0; (b) remove the test function entirely. **Decision:** keep the test function but rename it `TestRegisteredCompleteness_HelperUnchanged` and rewrite it to call `(&RulesDoc{Rules: ...}).validateRegisteredCompleteness(...)` **directly** to assert the helper still works (so re-enabling later is a one-line change with confidence). Add a comment noting the helper is intentionally not wired into `Validate()` for the spike.

```go
func TestRegisteredCompleteness_HelperUnchanged(t *testing.T) {
    // The helper is intentionally not wired into Validate() in the spike phase
    // (TODO marker in rules_doc.go). Re-enabling is a one-line change; this
    // test guards that re-enabling will still be correct.
    tests := []struct {
        name       string
        rules      []ResolvedRule
        registered []string
        wantLen    int
    }{
        // ... same table as before, minus the `Provider:` field in minimalRule ...
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            catalog := &RulesDoc{Rules: tt.rules}
            errs := catalog.validateRegisteredCompleteness(tt.registered)
            assert.Len(t, errs, tt.wantLen)
        })
    }
}
```

- [ ] **Step 6: Verify the trim's load-bearing claim — `docs` no longer imports `validation/rules` AT THIS POINT IN HISTORY**

Run:
```bash
cd /Users/shanmukh/workspace/rudder-iac/.worktrees/dex-370-path-b
grep -n "validation/rules" cli/internal/validation/docs/*.go
```
Expected: zero matches in non-test files (test files may still import `validation/rules` to construct stub rules — that's fine).

**Important:** this clean state is the load-bearing acceptance bar of Task 2 ONLY. Later tasks (Task 4 Resolver, Task 5 Generator) DELIBERATELY re-introduce the `validation/rules` import for legitimate uses (`rules.Rule` is the parameter type on `Resolver.ResolveFor`). The forward-compatibility point of Task 2 is not "docs forever lacks the rules import" — it's "the import is no longer accidentally pulled in by an unrelated `ValidateStruct` call, and the docs↔rules edge direction is now ours to control deliberately." Spec §13's "post-trim zero imports" criterion is satisfied here at this commit boundary.

- [ ] **Step 7: Build + test**

```bash
go build ./... && go test ./cli/internal/validation/docs/...
```
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add -A cli/internal/validation/docs/
git commit -m "$(cat <<'EOF'
refactor(docs): trim PR #471 to Path B essentials

- Delete RuleDocEntry, LoadRuleDocEntries (Path B has no YAML surface).
- Drop ResolvedRule.Provider (rule ID prefix already encodes provenance).
- Drop ToolMetadata.GeneratedAt (kills artifact reproducibility).
- Replace rules.ValidateStruct call in validateRuleStruct with a local
  validator.New() instance — dissolves the docs->rules import dep that
  blocked rules from importing docs. This unblocks the Documented
  interface (next commit).
- Comment out validateRegisteredCompleteness invocation with TODO
  marker (3-of-13 rules documented in the spike).

Post-trim: docs/ has zero imports from validation/rules/.
EOF
)"
```

Verify commit:
```bash
git show --stat HEAD
```
Expected: 2 deletions, 3 modifications, negative net LoC.

---

### Task 3: `Documented` interface

**Files:**
- Create: `cli/internal/validation/docs/documented.go`

- [ ] **Step 1: Write the file**

```go
package docs

// Documented is an optional sibling interface for rules that author
// structured documentation data. The docs pipeline type-asserts each
// registered rule to Documented; rules that don't implement it produce
// no entry in the generated artifact.
//
// This is intentionally separate from the unrelated rules.Rule.Examples()
// method, which returns plain strings attached to diagnostics at runtime
// for the text renderer's end-user error output.
type Documented interface {
    DocExamples() []MatchBehaviorEntry
}
```

No test for the interface itself — it'll be exercised end-to-end by the resolver tests in Task 4.

- [ ] **Step 2: Build**

```bash
go build ./cli/internal/validation/docs/...
```
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add cli/internal/validation/docs/documented.go
git commit -m "feat(docs): add opt-in Documented interface for rule authors"
```

---

### Task 4: `Resolver` interface + `ExamplesResolver`

**Files:**
- Create: `cli/internal/validation/docs/resolver.go`
- Create: `cli/internal/validation/docs/resolver_test.go`

- [ ] **Step 1: Write the failing test**

```go
package docs

import (
    "testing"

    "github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

type stubRule struct{ id string }

func (stubRule) ID() string                                              { return "" }
func (stubRule) Severity() rules.Severity                                { return rules.Error }
func (stubRule) Description() string                                     { return "" }
func (stubRule) AppliesTo() []rules.MatchPattern                         { return nil }
func (stubRule) Examples() rules.Examples                                { return rules.Examples{} }
func (stubRule) Validate(*rules.ValidationContext) []rules.ValidationResult { return nil }

type stubDocumentedRule struct {
    stubRule
    entries []MatchBehaviorEntry
}

func (d stubDocumentedRule) DocExamples() []MatchBehaviorEntry { return d.entries }

func TestExamplesResolver_ReturnsNilForRulesWithoutDocumented(t *testing.T) {
    got, err := ExamplesResolver{}.ResolveFor(stubRule{})
    require.NoError(t, err)
    assert.Nil(t, got)
}

func TestExamplesResolver_ReturnsNilForEmptyDocExamples(t *testing.T) {
    got, err := ExamplesResolver{}.ResolveFor(stubDocumentedRule{entries: nil})
    require.NoError(t, err)
    assert.Nil(t, got)
}

func TestExamplesResolver_ReturnsPopulatedResolvedRule(t *testing.T) {
    entries := []MatchBehaviorEntry{
        {AppliesTo: []MatchPatternDoc{{Kind: "source", Version: "v1"}}},
    }
    got, err := ExamplesResolver{}.ResolveFor(stubDocumentedRule{entries: entries})
    require.NoError(t, err)
    assert.Equal(t, &ResolvedRule{MatchBehavior: entries}, got)
}
```

- [ ] **Step 2: Run — expect fail**

```bash
go test ./cli/internal/validation/docs/... -run ExamplesResolver
```
Expected: FAIL — undefined `ExamplesResolver`.

- [ ] **Step 3: Implement** in `resolver.go`

```go
package docs

import "github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"

// Resolver is the only seam between Path A and Path B spike strategies.
// It returns the authored documentation data for a given rule, or
// (nil, nil) when the rule has no docs authored. Errors are reserved
// for authoring/load problems.
type Resolver interface {
    ResolveFor(r rules.Rule) (*ResolvedRule, error)
}

// ExamplesResolver resolves docs by type-asserting rules to the
// Documented interface. Rules without it produce no entry.
type ExamplesResolver struct{}

func (ExamplesResolver) ResolveFor(r rules.Rule) (*ResolvedRule, error) {
    d, ok := r.(Documented)
    if !ok {
        return nil, nil
    }
    entries := d.DocExamples()
    if len(entries) == 0 {
        return nil, nil
    }
    return &ResolvedRule{MatchBehavior: entries}, nil
}
```

- [ ] **Step 4: Run — expect pass**

```bash
go test ./cli/internal/validation/docs/... -run ExamplesResolver
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add cli/internal/validation/docs/resolver.go cli/internal/validation/docs/resolver_test.go
git commit -m "feat(docs): add Resolver interface and ExamplesResolver"
```

---

### Task 5: Generator

The generator orchestrates: walk registry → resolve per rule → enrich `ResolvedRule` with rule metadata → assemble `RulesDoc` → run `RulesDoc.Validate()`.

**Files:**
- Create: `cli/internal/validation/docs/generator.go`
- Create: `cli/internal/validation/docs/generator_test.go`

- [ ] **Step 1: Write the failing test**

```go
package docs

import (
    "testing"

    "github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

type fakeRegistry struct {
    syntactic []rules.Rule
    semantic  []rules.Rule
}

func (f fakeRegistry) RegisterSyntactic(r rules.Rule)                {}
func (f fakeRegistry) RegisterSemantic(r rules.Rule)                 {}
func (f fakeRegistry) SyntacticRulesFor(kind, version string) []rules.Rule {
    return f.syntactic
}
func (f fakeRegistry) SemanticRulesFor(kind, version string) []rules.Rule {
    return f.semantic
}
func (f fakeRegistry) AllSyntacticRules() []rules.Rule { return f.syntactic }
func (f fakeRegistry) AllSemanticRules() []rules.Rule  { return f.semantic }

type docRuleStub struct {
    id          string
    description string
    severity    rules.Severity
    appliesTo   []rules.MatchPattern
    entries     []MatchBehaviorEntry
}

func (d docRuleStub) ID() string                              { return d.id }
func (d docRuleStub) Severity() rules.Severity                { return d.severity }
func (d docRuleStub) Description() string                     { return d.description }
func (d docRuleStub) AppliesTo() []rules.MatchPattern         { return d.appliesTo }
func (d docRuleStub) Examples() rules.Examples                { return rules.Examples{} }
func (d docRuleStub) Validate(*rules.ValidationContext) []rules.ValidationResult {
    return nil
}
func (d docRuleStub) DocExamples() []MatchBehaviorEntry { return d.entries }

func TestGenerator_SkipsRulesWithoutDocs(t *testing.T) {
    g := NewGenerator(ExamplesResolver{}, "test-version")
    reg := fakeRegistry{syntactic: []rules.Rule{stubRule{}}}
    doc, err := g.Generate(reg)
    require.NoError(t, err)
    assert.Empty(t, doc.Rules)
    assert.Equal(t, "test-version", doc.ToolMetadata.CLIVersion)
}

func TestGenerator_EnrichesResolvedRuleFromRuleInterface(t *testing.T) {
    rule := docRuleStub{
        id:          "test/rule",
        description: "test rule",
        severity:    rules.Error,
        appliesTo:   []rules.MatchPattern{{Kind: "source", Version: "v1"}},
        entries: []MatchBehaviorEntry{
            {AppliesTo: []MatchPatternDoc{{Kind: "source", Version: "v1"}}},
        },
    }
    g := NewGenerator(ExamplesResolver{}, "v0.0.0")
    reg := fakeRegistry{syntactic: []rules.Rule{rule}}

    doc, err := g.Generate(reg)
    require.NoError(t, err)
    require.Len(t, doc.Rules, 1)
    assert.Equal(t, ResolvedRule{
        RuleID:      "test/rule",
        Phase:       "syntactic",
        Severity:    "error",
        Description: "test rule",
        AppliesTo:   []MatchPatternDoc{{Kind: "source", Version: "v1"}},
        MatchBehavior: []MatchBehaviorEntry{
            {AppliesTo: []MatchPatternDoc{{Kind: "source", Version: "v1"}}},
        },
    }, doc.Rules[0])
}

func TestGenerator_SemanticRulesGetSemanticPhase(t *testing.T) {
    rule := docRuleStub{
        id:        "sem/rule",
        severity:  rules.Warning,
        appliesTo: []rules.MatchPattern{rules.MatchAll()},
        entries: []MatchBehaviorEntry{
            {AppliesTo: []MatchPatternDoc{{Kind: "*", Version: "*"}}},
        },
        description: "x",
    }
    g := NewGenerator(ExamplesResolver{}, "v")
    reg := fakeRegistry{semantic: []rules.Rule{rule}}
    doc, err := g.Generate(reg)
    require.NoError(t, err)
    require.Len(t, doc.Rules, 1)
    assert.Equal(t, "semantic", doc.Rules[0].Phase)
}
```

- [ ] **Step 2: Run — expect fail**

```bash
go test ./cli/internal/validation/docs/... -run Generator
```
Expected: FAIL — undefined `Generator` / `NewGenerator`.

- [ ] **Step 3: Implement** in `generator.go`

```go
package docs

import (
    "fmt"

    "github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

const schemaVersion = 1

// Generator walks a registry and produces a structurally valid RulesDoc.
// The verifier is a separate step (see verifier.go) — Generator does NOT
// execute examples against the validation engine.
type Generator struct {
    resolver   Resolver
    cliVersion string
}

func NewGenerator(resolver Resolver, cliVersion string) *Generator {
    return &Generator{resolver: resolver, cliVersion: cliVersion}
}

// Generate walks the registry, resolves authored docs per rule, enriches
// each ResolvedRule with metadata from the rule itself, runs structural
// validation, and returns the populated RulesDoc.
func (g *Generator) Generate(reg rules.Registry) (*RulesDoc, error) {
    doc := &RulesDoc{
        SchemaVersion: schemaVersion,
        ToolMetadata:  ToolMetadata{CLIVersion: g.cliVersion},
    }

    if err := g.appendResolved(doc, reg.AllSyntacticRules(), "syntactic"); err != nil {
        return nil, err
    }
    if err := g.appendResolved(doc, reg.AllSemanticRules(), "semantic"); err != nil {
        return nil, err
    }

    // expectedRuleIDs is unused while completeness is commented out.
    if errs := doc.Validate(nil); len(errs) > 0 {
        return nil, fmt.Errorf("structural validation failed: %v", errs)
    }
    return doc, nil
}

func (g *Generator) appendResolved(doc *RulesDoc, ruleSet []rules.Rule, phase string) error {
    for _, r := range ruleSet {
        resolved, err := g.resolver.ResolveFor(r)
        if err != nil {
            return fmt.Errorf("resolving docs for rule %s: %w", r.ID(), err)
        }
        if resolved == nil {
            continue
        }
        resolved.RuleID = r.ID()
        resolved.Phase = phase
        resolved.Severity = severityString(r.Severity())
        resolved.Description = r.Description()
        resolved.AppliesTo = patternsToDocs(r.AppliesTo())
        doc.Rules = append(doc.Rules, *resolved)
    }
    return nil
}

func severityString(s rules.Severity) string {
    switch s {
    case rules.Error:
        return "error"
    case rules.Warning:
        return "warning"
    case rules.Info:
        return "info"
    default:
        return "error"
    }
}

func patternsToDocs(ps []rules.MatchPattern) []MatchPatternDoc {
    out := make([]MatchPatternDoc, len(ps))
    for i, p := range ps {
        out[i] = MatchPatternDoc{Kind: p.Kind, Version: p.Version}
    }
    return out
}
```

- [ ] **Step 4: Run — expect pass**

```bash
go test ./cli/internal/validation/docs/... -run Generator
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add cli/internal/validation/docs/generator.go cli/internal/validation/docs/generator_test.go
git commit -m "feat(docs): add Generator that walks registry and enriches resolved rules"
```

---

### Task 6: Verifier

The Verifier materializes each `InvalidExample`'s `Files` map into a tmpdir, runs `ValidationEngine.ValidateSyntax`, and subset-matches produced diagnostics against `ExpectedDiagnostic`s.

**Files:**
- Create: `cli/internal/validation/docs/verifier.go`
- Create: `cli/internal/validation/docs/verifier_test.go`

- [ ] **Step 1: Write the failing test**

Two scenarios cover the core contract:

Use the real interface types from `cli/internal/validation/engine.go`:

```go
package docs

import (
    "context"
    "testing"

    "github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
    "github.com/rudderlabs/rudder-iac/cli/internal/resources"
    "github.com/rudderlabs/rudder-iac/cli/internal/validation"
    "github.com/rudderlabs/rudder-iac/cli/internal/validation/pathindex"
    "github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

// fakeEngine returns whatever diagnostics it was configured with, ignoring inputs.
type fakeEngine struct {
    syntaxDiags validation.Diagnostics
    syntaxErr   error
}

func (f fakeEngine) ValidateSyntax(_ context.Context, _ map[string]*specs.RawSpec) (validation.Diagnostics, error) {
    return f.syntaxDiags, f.syntaxErr
}
// ValidateSemantic is required by the interface but Path B's verifier only calls ValidateSyntax.
func (f fakeEngine) ValidateSemantic(_ context.Context, _ map[string]*specs.RawSpec, _ *resources.Graph) (validation.Diagnostics, error) {
    return nil, nil
}
```

```go
func TestVerifier_SubsetMatchSucceedsWhenExpectedDiagnosticIsProduced(t *testing.T) {
    doc := &RulesDoc{Rules: []ResolvedRule{{
        RuleID: "r1",
        MatchBehavior: []MatchBehaviorEntry{{
            Invalid: []InvalidExample{{
                ExampleID: "ex1",
                Files:     map[string]string{"main.yaml": "kind: foo\nversion: v1\nspec: {}\n"},
                ExpectedDiagnostics: []ExpectedDiagnostic{
                    {File: "main.yaml", Reference: "/name", Severity: "error", MessageContains: "required"},
                },
            }},
        }},
    }}}

    engine := fakeEngine{syntaxDiags: validation.Diagnostics{{
        RuleID:   "r1",
        Severity: rules.Error,
        Message:  "field 'name' is required",
        File:     "main.yaml",
        Position: pathindex.Position{Line: 1, Column: 1},
    }}}

    v := NewVerifier(func() validation.ValidationEngine { return engine })
    require.NoError(t, v.Verify(doc))
}

func TestVerifier_FailsWhenExpectedDiagnosticIsMissing(t *testing.T) {
    doc := &RulesDoc{Rules: []ResolvedRule{{
        RuleID: "r1",
        MatchBehavior: []MatchBehaviorEntry{{
            Invalid: []InvalidExample{{
                ExampleID: "ex1",
                Files:     map[string]string{"main.yaml": "kind: foo\n"},
                ExpectedDiagnostics: []ExpectedDiagnostic{
                    {File: "main.yaml", Reference: "/name", Severity: "error", MessageContains: "required"},
                },
            }},
        }},
    }}}

    engine := fakeEngine{syntaxDiags: validation.Diagnostics{}}
    v := NewVerifier(func() validation.ValidationEngine { return engine })

    err := v.Verify(doc)
    require.Error(t, err)
    assert.Contains(t, err.Error(), "ex1")
    assert.Contains(t, err.Error(), "/name")
}
```

(The fake-engine approach decouples the test from the file-loading machinery. The real Verifier tests its own file-materialization helper separately.)

- [ ] **Step 2: Run — expect fail (compilation)**

```bash
go test ./cli/internal/validation/docs/... -run Verifier
```
Expected: FAIL — undefined `NewVerifier` / `Verifier`.

- [ ] **Step 3: Implement** in `verifier.go`

```go
package docs

import (
    "context"
    "fmt"
    "os"
    "path/filepath"
    "strings"

    "github.com/rudderlabs/rudder-iac/cli/internal/project/loader"
    "github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
    "github.com/rudderlabs/rudder-iac/cli/internal/validation"
)
// NOTE: this file does not need to import validation/rules directly —
// it never names rules.Severity or rules.Rule. The severity comparison
// goes through severityToString() defined in helpers.go (which DOES
// import rules for the rules.Severity type). Keeping verifier.go free
// of the rules import keeps the dependency surface tight.

// Verifier executes every InvalidExample on every documented rule through
// the validation engine and asserts that the authored ExpectedDiagnostics
// each match at least one produced diagnostic (subset semantics, spec §5).
//
// The engine is injected by factory so tests can swap in a fake without
// touching the file-materialization helper.
type Verifier struct {
    engineFactory func() validation.ValidationEngine
}

func NewVerifier(factory func() validation.ValidationEngine) *Verifier {
    return &Verifier{engineFactory: factory}
}

func (v *Verifier) Verify(doc *RulesDoc) error {
    for _, rule := range doc.Rules {
        for _, mb := range rule.MatchBehavior {
            for _, ex := range mb.Invalid {
                if err := v.verifyExample(rule.RuleID, ex); err != nil {
                    return err
                }
            }
        }
    }
    return nil
}

func (v *Verifier) verifyExample(ruleID string, ex InvalidExample) error {
    diags, err := v.runEngineOnFiles(ex.Files)
    if err != nil {
        return fmt.Errorf("rule %s example %s: %w", ruleID, ex.ExampleID, err)
    }
    for _, expected := range ex.ExpectedDiagnostics {
        if !matchesAny(expected, diags) {
            return fmt.Errorf(
                "rule %s example %s: no produced diagnostic matched expected {file=%s reference=%s severity=%s message_contains=%q}",
                ruleID, ex.ExampleID,
                expected.File, expected.Reference, expected.Severity, expected.MessageContains,
            )
        }
    }
    return nil
}

func (v *Verifier) runEngineOnFiles(files map[string]string) (validation.Diagnostics, error) {
    tmp, err := os.MkdirTemp("", "ruledoc-verify-*")
    if err != nil {
        return nil, fmt.Errorf("creating tmpdir: %w", err)
    }
    defer os.RemoveAll(tmp)

    for name, body := range files {
        full := filepath.Join(tmp, name)
        if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
            return nil, fmt.Errorf("creating subdir for %s: %w", name, err)
        }
        if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
            return nil, fmt.Errorf("writing %s: %w", name, err)
        }
    }

    l := &loader.Loader{}
    rawSpecs, err := l.Load(tmp)
    if err != nil {
        return nil, fmt.Errorf("loading tmpdir: %w", err)
    }

    // The path in produced diagnostics is the absolute tmpdir path. The author
    // wrote File: "main.yaml" relative — we rewrite the rawSpecs map keys to
    // strip the tmpdir prefix so the diagnostic File values match the author's
    // relative names. (Type is map[string]*specs.RawSpec — import the
    // specs package via cli/internal/project/specs.)
    relSpecs := make(map[string]*specs.RawSpec, len(rawSpecs))
    for k, vRaw := range rawSpecs {
        rel := strings.TrimPrefix(k, tmp+string(filepath.Separator))
        relSpecs[rel] = vRaw
    }

    engine := v.engineFactory()
    return engine.ValidateSyntax(context.Background(), relSpecs)
}

func matchesAny(expected ExpectedDiagnostic, produced validation.Diagnostics) bool {
    for _, d := range produced {
        if d.File != expected.File {
            continue
        }
        if string(d.Severity) != expected.Severity {
            // d.Severity is rules.Severity (int) — String() is required.
            // See severityToString helper below.
            continue
        }
        if d.RuleID != expected.Reference && !strings.Contains(d.RuleID, expected.Reference) {
            // NOTE: Reference in spec §5 means rule reference / diagnostic
            // reference path. The Diagnostic struct doesn't carry a separate
            // "reference" field — it carries RuleID and Position. For the spike,
            // we match Reference against either RuleID directly or by checking
            // the produced Position via subsequent fields. Simplify: match
            // expected.Reference against d.RuleID for now and refine when an
            // example exercises a finer-grained case.
            continue
        }
        if expected.MessageContains != "" && !strings.Contains(d.Message, expected.MessageContains) {
            continue
        }
        return true
    }
    return false
}
```

**Implementation note for the executor:** the `matchesAny` body has a tricky semantic — `ExpectedDiagnostic.Reference` is a JSON-pointer-style reference (e.g., `/metadata/name`) while `Diagnostic` carries `RuleID` + `Position` but not a "reference" string. The diagnostic emitted from the engine has the rule_id and a resolved (line, column) — the reference goes IN via `rule.Validate()` `ValidationResult.Reference` and gets converted to a position before emit. **The reference is therefore not directly observable on the produced `Diagnostic`.** Two options:
  (a) Augment `Diagnostic` with a `Reference string` field — small, risk-free, but expands the spike's blast radius.
  (b) Match only on `File` + `Severity` + `MessageContains` in the spike, document the looseness, and revisit if Path B wins.

**Decision:** (b). Cut the `expected.Reference` comparison from `matchesAny` in the spike. Add a `// TODO(spike DEX-370): match Reference once Diagnostic.Reference exists` comment. Pilot examples are authored conservatively enough that the looser match is still meaningful (only one diagnostic per file per rule in the three pilots).

Likewise, `severity` matching needs `rules.Severity.String()` or a small local helper. The existing `Severity` type is an int — add a small helper:

```go
func severityToString(s rules.Severity) string {
    switch s {
    case rules.Error:
        return "error"
    case rules.Warning:
        return "warning"
    case rules.Info:
        return "info"
    }
    return ""
}
```

Reuse it in both `generator.go` (already defined there as `severityString`) and `verifier.go` — DRY rule applies. Move it to a shared `helpers.go` in `docs/` if both files need it.

The `loaderRawSpec` shorthand in the code above is illustrative — use the real `*specs.RawSpec` type. Same for `specsRawSpec`/`resourcesGraph` in test stubs.

- [ ] **Step 4: Run — expect pass**

```bash
go test ./cli/internal/validation/docs/... -run Verifier
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add cli/internal/validation/docs/verifier.go cli/internal/validation/docs/verifier_test.go
git commit -m "feat(docs): add Verifier with subset diagnostic matching"
```

---

### Task 7: Serializer

Emit `RulesDoc` as deterministic YAML and JSON to the requested directory.

**Files:**
- Create: `cli/internal/validation/docs/serializer.go`
- Create: `cli/internal/validation/docs/serializer_test.go`

- [ ] **Step 1: Write the failing test**

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

func TestSerializer_EmitsYAMLAndJSON(t *testing.T) {
    dir := t.TempDir()
    doc := &RulesDoc{
        SchemaVersion: 1,
        ToolMetadata:  ToolMetadata{CLIVersion: "v0.0.0"},
        Rules: []ResolvedRule{{
            RuleID:      "r1",
            Phase:       "syntactic",
            Severity:    "error",
            Description: "test",
            AppliesTo:   []MatchPatternDoc{{Kind: "source", Version: "v1"}},
            MatchBehavior: []MatchBehaviorEntry{
                {AppliesTo: []MatchPatternDoc{{Kind: "source", Version: "v1"}}},
            },
        }},
    }

    require.NoError(t, Serialize(doc, dir))

    yamlBytes, err := os.ReadFile(filepath.Join(dir, "rules.yaml"))
    require.NoError(t, err)
    var roundTripYAML RulesDoc
    require.NoError(t, yaml.Unmarshal(yamlBytes, &roundTripYAML))
    assert.Equal(t, doc, &roundTripYAML)

    jsonBytes, err := os.ReadFile(filepath.Join(dir, "rules.json"))
    require.NoError(t, err)
    var roundTripJSON RulesDoc
    require.NoError(t, json.Unmarshal(jsonBytes, &roundTripJSON))
    assert.Equal(t, doc, &roundTripJSON)
}

func TestSerializer_CreatesOutputDirIfMissing(t *testing.T) {
    base := t.TempDir()
    nested := filepath.Join(base, "doesnt", "exist")
    doc := &RulesDoc{SchemaVersion: 1, ToolMetadata: ToolMetadata{CLIVersion: "v"}}
    require.NoError(t, Serialize(doc, nested))
    _, err := os.Stat(filepath.Join(nested, "rules.yaml"))
    require.NoError(t, err)
}
```

**Note on round-trip:** `ResolvedRule`/`MatchPatternDoc` have `yaml:` tags but no `json:` tags. The default JSON marshaller will use field names (PascalCase). For the YAML round-trip to equal the original after `json.Unmarshal`, JSON tags are needed too. **Decision:** add `json:` tags mirroring the `yaml:` tags in Task 2's `types.go` edits (very small change, slot it into the trim commit retroactively if missed; otherwise add as a small commit before Task 7). Document this in Task 2 too. — Update Task 2 step 3 to include json tags. **Plan amends Task 2's step 3 (above) implicitly: when editing types.go, add `json:"…"` tags mirroring the `yaml:"…"` tags on every field of `RulesDoc`, `ToolMetadata`, `ResolvedRule`, `MatchBehaviorEntry`, `MatchPatternDoc`, `ValidExample`, `InvalidExample`, `ExpectedDiagnostic`.** If the executor reaches Task 7 and the JSON round-trip test fails for tag reasons, go back and patch types.go (no commit churn — squash into the trim commit if not yet pushed).

- [ ] **Step 2: Run — expect fail**

```bash
go test ./cli/internal/validation/docs/... -run Serializer
```
Expected: FAIL — undefined `Serialize`.

- [ ] **Step 3: Implement** in `serializer.go`

```go
package docs

import (
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"

    "gopkg.in/yaml.v3"
)

// Serialize writes the RulesDoc to outputDir as rules.yaml and rules.json.
// outputDir is created (mkdir -p) if it does not exist.
func Serialize(doc *RulesDoc, outputDir string) error {
    if err := os.MkdirAll(outputDir, 0o755); err != nil {
        return fmt.Errorf("creating output dir %s: %w", outputDir, err)
    }

    yamlBytes, err := yaml.Marshal(doc)
    if err != nil {
        return fmt.Errorf("marshalling YAML: %w", err)
    }
    if err := os.WriteFile(filepath.Join(outputDir, "rules.yaml"), yamlBytes, 0o644); err != nil {
        return fmt.Errorf("writing rules.yaml: %w", err)
    }

    jsonBytes, err := json.MarshalIndent(doc, "", "  ")
    if err != nil {
        return fmt.Errorf("marshalling JSON: %w", err)
    }
    if err := os.WriteFile(filepath.Join(outputDir, "rules.json"), jsonBytes, 0o644); err != nil {
        return fmt.Errorf("writing rules.json: %w", err)
    }
    return nil
}
```

- [ ] **Step 4: Run — expect pass**

```bash
go test ./cli/internal/validation/docs/... -run Serializer
```
Expected: PASS. If a tag-related round-trip mismatch appears, return to Task 2 and add `json:` tags as noted.

- [ ] **Step 5: Commit**

```bash
git add cli/internal/validation/docs/serializer.go cli/internal/validation/docs/serializer_test.go
git commit -m "feat(docs): add JSON+YAML serializer for RulesDoc"
```

---

### Task 8: CLI command `rudder-cli docs rules`

**Files:**
- Create: `cli/internal/cmd/docs/docs.go` — group parent
- Create: `cli/internal/cmd/docs/rules/rules.go` — `rules` subcommand
- Create: `cli/internal/cmd/docs/rules/rules_test.go`
- Modify: `cli/internal/cmd/root.go` — wire `docs.NewCmdDocs()` into root

- [ ] **Step 1: Group command in `docs.go`**

```go
package docs

import (
    "github.com/rudderlabs/rudder-iac/cli/internal/cmd/docs/rules"
    "github.com/spf13/cobra"
)

func NewCmdDocs() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "docs",
        Short: "Generate documentation artifacts (validation rules and related metadata)",
        Run: func(cmd *cobra.Command, args []string) {
            _ = cmd.Help()
        },
    }
    cmd.AddCommand(rules.NewCmdRules())
    return cmd
}
```

- [ ] **Step 2: Failing test for `rules.NewCmdRules()` flag wiring**

In `rules/rules_test.go`:

```go
package rules

import (
    "bytes"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestCmdRules_StrictVerifyFlagReturnsErrorInSpike(t *testing.T) {
    cmd := NewCmdRules()
    cmd.SetArgs([]string{"--strict-verify"})
    var out bytes.Buffer
    cmd.SetOut(&out)
    cmd.SetErr(&out)
    err := cmd.Execute()
    require.Error(t, err)
    assert.Contains(t, err.Error(), "strict-verify mode is not implemented")
}

func TestCmdRules_DefaultsAreDocumentedInHelp(t *testing.T) {
    cmd := NewCmdRules()
    flag := cmd.Flags().Lookup("output-dir")
    require.NotNil(t, flag)
    assert.Equal(t, "./docs/generated/", flag.DefValue)

    strict := cmd.Flags().Lookup("strict-verify")
    require.NotNil(t, strict)
    assert.Contains(t, strict.Usage, "not implemented")
}
```

- [ ] **Step 3: Run — expect fail**

```bash
go test ./cli/internal/cmd/docs/rules/...
```
Expected: FAIL — package doesn't exist yet.

- [ ] **Step 4: Implement** in `rules/rules.go`

```go
package rules

import (
    "fmt"

    "github.com/MakeNowJust/heredoc/v2"
    "github.com/rudderlabs/rudder-iac/cli/internal/logger"
    "github.com/rudderlabs/rudder-iac/cli/internal/validation"
    "github.com/rudderlabs/rudder-iac/cli/internal/validation/docs"
    rrules "github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
    "github.com/spf13/cobra"
)

var log = logger.New("docs-rules")

func NewCmdRules() *cobra.Command {
    var (
        outputDir    string
        strictVerify bool
    )

    cmd := &cobra.Command{
        Use:   "rules",
        Short: "Generate validation rule documentation (JSON + YAML)",
        Long: heredoc.Doc(`
            Walk the registered validation rules, resolve documentation
            data for rules that implement the Documented interface,
            verify that each authored invalid example actually produces
            the expected diagnostics, and emit the consolidated artifact
            to the output directory.
        `),
        Example: heredoc.Doc(`
            $ rudder-cli docs rules
            $ rudder-cli docs rules --output-dir ./docs/generated/
        `),
        RunE: func(cmd *cobra.Command, args []string) error {
            if strictVerify {
                return fmt.Errorf(
                    "strict-verify mode is not implemented in the spike (tracked in DEX-216 follow-up); rerun without --strict-verify",
                )
            }

            registry := buildRegistry()
            engine, err := validation.NewValidationEngine(registry, log)
            if err != nil {
                return fmt.Errorf("constructing validation engine: %w", err)
            }

            gen := docs.NewGenerator(docs.ExamplesResolver{}, cliVersionString(cmd))
            doc, err := gen.Generate(registry)
            if err != nil {
                return fmt.Errorf("generating rules doc: %w", err)
            }

            verifier := docs.NewVerifier(func() validation.ValidationEngine { return engine })
            if err := verifier.Verify(doc); err != nil {
                return fmt.Errorf("verifying examples: %w", err)
            }

            if err := docs.Serialize(doc, outputDir); err != nil {
                return fmt.Errorf("serializing: %w", err)
            }

            fmt.Fprintf(cmd.OutOrStdout(), "Wrote rules.yaml and rules.json to %s\n", outputDir)
            return nil
        },
    }

    cmd.Flags().StringVar(&outputDir, "output-dir", "./docs/generated/", "Directory to write rules.yaml and rules.json")
    cmd.Flags().BoolVar(&strictVerify, "strict-verify", false,
        "Switch verifier to exact-match mode (not implemented in the spike; tracked in DEX-216 follow-up)")
    return cmd
}

// buildRegistry constructs a rules.Registry directly from each provider
// without needing an authenticated API client. Provider rule constructors
// are pure (they don't reach into the client at registration time).
func buildRegistry() rrules.Registry {
    reg := rrules.NewRegistry()
    // Provider rule sets — call New(nil) on each provider that requires
    // a client (none of the rule constructors touch the nil client).
    // Mirror project.registry()'s shape.
    // ...
    // (Executor: enumerate every provider in app.Providers and register
    //  SyntacticRules() + SemanticRules(). Also register the three
    //  project-level rules — NewSpecSyntaxValidRule, NewMetadataSyntaxValidRule,
    //  NewDuplicateURNRule — with their parseSpec functions wired from
    //  CompositeProvider.ParseSpec.)
    return reg
}

func cliVersionString(cmd *cobra.Command) string {
    // The root cmd version is set via cmd.SetVersion in main. Walk up
    // and return it; default to "dev" if unavailable.
    for c := cmd; c != nil; c = c.Parent() {
        if c.Version != "" {
            return c.Version
        }
    }
    return "dev"
}
```

**Implementation note for the executor:** `buildRegistry` is the trickiest part of this task — needs to assemble providers with a nil client and call their rule constructors. The cleanest implementation pattern:

```go
func buildRegistry() rrules.Registry {
    reg := rrules.NewRegistry()
    providersList := []provider.RuleProvider{
        datacatalog.New(nil),
        retl.New(nil),
        eventstream.New(nil),
        transformations.NewProvider(nil),
    }
    for _, p := range providersList {
        for _, r := range p.SyntacticRules() {
            reg.RegisterSyntactic(r)
        }
        for _, r := range p.SemanticRules() {
            reg.RegisterSemantic(r)
        }
    }
    // project-level rules require a parseSpec function; use the composite
    // provider's ParseSpec. NewCompositeProvider also accepts nil clients
    // through the chain — verify by reading composite.go before wiring.
    cp, _ := provider.NewCompositeProvider(map[string]provider.Provider{
        "datacatalog":     providersList[0].(provider.Provider),
        // ... etc
    })
    parseSpec := cp.ParseSpec
    versions := []string{specs.SpecVersionV0_1, specs.SpecVersionV0_1Variant, specs.SpecVersionV1}
    reg.RegisterSyntactic(prules.NewMetadataSyntaxValidRule(parseSpec, versions))
    reg.RegisterSyntactic(prules.NewDuplicateURNRule(parseSpec))
    // NewSpecSyntaxValidRule needs SupportedKinds — get from composite.
    reg.RegisterSyntactic(prules.NewSpecSyntaxValidRule(cp.SupportedKinds(), versions))
    return reg
}
```

If `datacatalog.New(nil)` panics or fails at registration (the constructors read the client), the executor will need to find the smallest seam to pass through the rule constructors directly. Verify by trying `datacatalog.New(nil).SyntacticRules()` in a scratch test before wiring. Document any awkwardness rather than refactor in the spike (per spec §12.3 spirit).

- [ ] **Step 5: Run — expect pass**

```bash
go test ./cli/internal/cmd/docs/...
```
Expected: PASS.

- [ ] **Step 6: Wire into `root.go`**

In `cli/internal/cmd/root.go`, add the import and the registration:

```go
import (
    // ... existing imports ...
    "github.com/rudderlabs/rudder-iac/cli/internal/cmd/docs"
)

// inside init(), after the other rootCmd.AddCommand(...) calls:
rootCmd.AddCommand(docs.NewCmdDocs())
```

- [ ] **Step 7: Build + smoke-test the binary**

```bash
go build -o /tmp/rudder-cli ./cli/cmd/rudder-cli
/tmp/rudder-cli docs --help
/tmp/rudder-cli docs rules --help
```
Expected: both help screens render. The `docs rules` help mentions `--output-dir` and `--strict-verify` with the spike-disabled message.

Do NOT execute `/tmp/rudder-cli docs rules` (i.e., the full generation run) until Task 9 — without pilot rules implementing `Documented`, the artifact will be empty but the run should still succeed (zero rules + structurally valid empty doc). If it errors with "no rules documented", that's actually correct behavior — silent empty is fine. Verify Task 8 by `--help` output only.

- [ ] **Step 8: Commit**

```bash
git add cli/internal/cmd/docs/ cli/internal/cmd/root.go
git commit -m "feat(cli): add 'rudder-cli docs rules' subcommand"
```

---

### Task 9: Pilot rules implement `Documented`

The three pilot rules gain `DocExamples() []docs.MatchBehaviorEntry` methods. Bodies are AUTHORED DATA (counted against the §13 < 200 LoC cap). The existing `Examples()` methods on each rule stay untouched.

**Files:**
- Modify: `cli/internal/providers/datacatalog/rules/category/category_spec_valid.go`
- Modify: `cli/internal/project/rules/metadata_syntax_valid.go`
- Modify: `cli/internal/project/rules/duplicate_urn_rule.go`

The riskiest of the three is the first — see "9a: category_spec_valid" below for the wrinkle.

---

#### Task 9a: `datacatalog/categories/spec-syntax-valid`

**Wrinkle:** `NewCategorySpecSyntaxValidRule` returns `prules.NewTypedRule(...)` — a generic wrapper from `cli/internal/provider/rules/`. The wrapper type is not under this package's control, so `DocExamples()` cannot be added directly to it.

**Resolution:** wrap the typed rule in a tiny local struct that embeds it and adds `DocExamples()`.

- [ ] **Step 1: Failing test**

Create `cli/internal/providers/datacatalog/rules/category/category_spec_valid_doc_test.go`:

```go
package rules

import (
    "testing"

    "github.com/rudderlabs/rudder-iac/cli/internal/validation/docs"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestCategorySpecRule_ImplementsDocumented(t *testing.T) {
    rule := NewCategorySpecSyntaxValidRule()
    d, ok := rule.(docs.Documented)
    require.True(t, ok, "rule must implement docs.Documented")
    entries := d.DocExamples()
    require.GreaterOrEqual(t, len(entries), 2, "must cover both v0.1 and v1")
}
```

- [ ] **Step 2: Run — expect fail**

```bash
go test ./cli/internal/providers/datacatalog/rules/category/... -run ImplementsDocumented
```
Expected: FAIL — `rule.(docs.Documented)` assertion fails.

- [ ] **Step 3: Implement**

Add to `category_spec_valid.go`:

```go
// docsCategorySpecRule wraps the typed rule with documentation data.
// The typed rule is returned by prules.NewTypedRule and can't be modified;
// embedding lets us add DocExamples() without re-implementing the rules.Rule
// surface.
type docsCategorySpecRule struct {
    rules.Rule
}

func (docsCategorySpecRule) DocExamples() []docs.MatchBehaviorEntry {
    return []docs.MatchBehaviorEntry{
        {
            AppliesTo: []docs.MatchPatternDoc{{Kind: "categories", Version: "rudder/v0.1"}},
            Valid: []docs.ValidExample{
                {
                    ExampleID: "categories-v0.1-valid-minimal",
                    Title:     "Two categories with id and name",
                    Files: map[string]string{
                        "categories.yaml": heredoc.Doc(`
                            kind: categories
                            version: rudder/v0.1
                            spec:
                              categories:
                                - id: user_actions
                                  name: User Actions
                                - id: system_events
                                  name: System Events
                        `),
                    },
                },
            },
            Invalid: []docs.InvalidExample{
                {
                    ExampleID: "categories-v0.1-missing-id",
                    Title:     "Category missing id",
                    Files: map[string]string{
                        "categories.yaml": heredoc.Doc(`
                            kind: categories
                            version: rudder/v0.1
                            spec:
                              categories:
                                - name: No ID Here
                        `),
                    },
                    ExpectedDiagnostics: []docs.ExpectedDiagnostic{
                        {
                            File:            "categories.yaml",
                            Reference:       "datacatalog/categories/spec-syntax-valid",
                            Severity:        "error",
                            MessageContains: "id",
                        },
                    },
                },
            },
        },
        {
            AppliesTo: []docs.MatchPatternDoc{{Kind: "categories", Version: "rudder/v1"}},
            Valid: []docs.ValidExample{
                // ... v1 valid example ...
            },
            Invalid: []docs.InvalidExample{
                // ... v1 invalid example with expected diag ...
            },
        },
    }
}
```

Wrap the constructor's return:

```go
func NewCategorySpecSyntaxValidRule() rules.Rule {
    base := prules.NewTypedRule(
        // ... existing args ...
    )
    return docsCategorySpecRule{Rule: base}
}
```

Update imports to add `docs` and `heredoc/v2`.

**Authoring tip:** the v1 examples are similar in shape to v0.1 — read `localcatalog.CategorySpec` vs `CategorySpecV1` to see the field shape difference. Each version's invalid example should fire ONE diagnostic so the verifier's looser matching (Task 6) is unambiguous.

- [ ] **Step 4: Run — expect pass for the test, then run full docs/ tests too**

```bash
go test ./cli/internal/providers/datacatalog/rules/category/...
go test ./cli/internal/validation/docs/...
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add cli/internal/providers/datacatalog/rules/category/
git commit -m "feat(rules/category): implement Documented interface for spec-syntax-valid"
```

---

#### Task 9b: `project/metadata-syntax-valid`

Simpler — the rule has its own `*MetadataSyntaxValidRule` struct.

- [ ] **Step 1: Failing test**

In `cli/internal/project/rules/metadata_syntax_valid_doc_test.go`:

```go
package rules

import (
    "testing"

    "github.com/rudderlabs/rudder-iac/cli/internal/validation/docs"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestMetadataSyntaxValidRule_ImplementsDocumented(t *testing.T) {
    rule := NewMetadataSyntaxValidRule(nil, []string{"rudder/v0.1", "rudder/v1"})
    d, ok := rule.(docs.Documented)
    require.True(t, ok)

    entries := d.DocExamples()
    require.Len(t, entries, 1)
    assert.Equal(t, "*", entries[0].AppliesTo[0].Kind)
    assert.Equal(t, "*", entries[0].AppliesTo[0].Version)
}
```

- [ ] **Step 2: Run — expect fail**

- [ ] **Step 3: Implement** — add to `metadata_syntax_valid.go`:

```go
func (r *MetadataSyntaxValidRule) DocExamples() []docs.MatchBehaviorEntry {
    return []docs.MatchBehaviorEntry{{
        AppliesTo: []docs.MatchPatternDoc{{Kind: "*", Version: "*"}},
        Valid: []docs.ValidExample{
            {
                ExampleID: "metadata-valid-basic",
                Title:     "Basic metadata",
                Files: map[string]string{
                    "main.yaml": heredoc.Doc(`
                        kind: properties
                        version: rudder/v1
                        metadata:
                          name: my-project
                        spec: {}
                    `),
                },
            },
            {
                ExampleID: "metadata-valid-with-import",
                Title:     "Metadata with import block",
                Files: map[string]string{
                    "main.yaml": heredoc.Doc(`
                        kind: properties
                        version: rudder/v1
                        metadata:
                          name: my-project
                          import:
                            workspaces:
                              - workspace_id: ws-123
                        spec: {}
                    `),
                },
            },
        },
        Invalid: []docs.InvalidExample{
            {
                ExampleID: "metadata-missing-name",
                Title:     "Missing required name field",
                Files: map[string]string{
                    "main.yaml": heredoc.Doc(`
                        kind: properties
                        version: rudder/v1
                        metadata:
                          import:
                            workspaces:
                              - workspace_id: ws-123
                        spec: {}
                    `),
                },
                ExpectedDiagnostics: []docs.ExpectedDiagnostic{
                    {
                        File:            "main.yaml",
                        Reference:       "project/metadata-syntax-valid",
                        Severity:        "error",
                        MessageContains: "name",
                    },
                },
            },
            {
                ExampleID: "metadata-missing-workspace-id",
                Title:     "Missing required workspace_id in import",
                Files: map[string]string{
                    "main.yaml": heredoc.Doc(`
                        kind: properties
                        version: rudder/v1
                        metadata:
                          name: my-project
                          import:
                            workspaces:
                              - resources:
                                  - local_id: src-local
                                    remote_id: src-remote
                        spec: {}
                    `),
                },
                ExpectedDiagnostics: []docs.ExpectedDiagnostic{
                    {
                        File:            "main.yaml",
                        Reference:       "project/metadata-syntax-valid",
                        Severity:        "error",
                        MessageContains: "workspace_id",
                    },
                },
            },
        },
    }}
}
```

Add `docs` to imports (heredoc/v2 already imported).

- [ ] **Step 4: Run — expect pass**

```bash
go test ./cli/internal/project/rules/...
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add cli/internal/project/rules/metadata_syntax_valid.go cli/internal/project/rules/metadata_syntax_valid_doc_test.go
git commit -m "feat(rules/metadata): implement Documented interface"
```

---

#### Task 9c: `project/duplicate-urn`

- [ ] **Step 1: Failing test**

In `cli/internal/project/rules/duplicate_urn_rule_doc_test.go`:

```go
package rules

import (
    "testing"

    "github.com/rudderlabs/rudder-iac/cli/internal/validation/docs"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestDuplicateURNRule_ImplementsDocumented(t *testing.T) {
    rule := NewDuplicateURNRule(nil)
    d, ok := rule.(docs.Documented)
    require.True(t, ok)

    entries := d.DocExamples()
    require.Len(t, entries, 1)

    require.Len(t, entries[0].Invalid, 1)
    ex := entries[0].Invalid[0]
    assert.Len(t, ex.Files, 2, "duplicate URN example must span 2 files")
}
```

- [ ] **Step 2: Run — expect fail**

- [ ] **Step 3: Implement** — add to `duplicate_urn_rule.go`:

```go
func (r *duplicateURNRule) DocExamples() []docs.MatchBehaviorEntry {
    return []docs.MatchBehaviorEntry{{
        AppliesTo: []docs.MatchPatternDoc{{Kind: "*", Version: "*"}},
        Valid: []docs.ValidExample{
            {
                ExampleID: "duplicate-urn-valid-distinct",
                Title:     "Two files with distinct URNs",
                Files: map[string]string{
                    "a.yaml": heredoc.Doc(`
                        kind: properties
                        version: rudder/v1
                        metadata: {name: a}
                        spec:
                          properties:
                            - id: prop-a
                              name: A
                    `),
                    "b.yaml": heredoc.Doc(`
                        kind: properties
                        version: rudder/v1
                        metadata: {name: b}
                        spec:
                          properties:
                            - id: prop-b
                              name: B
                    `),
                },
            },
        },
        Invalid: []docs.InvalidExample{
            {
                ExampleID: "duplicate-urn-invalid-same",
                Title:     "Two files defining the same URN",
                Files: map[string]string{
                    "a.yaml": heredoc.Doc(`
                        kind: properties
                        version: rudder/v1
                        metadata: {name: a}
                        spec:
                          properties:
                            - id: prop-dup
                              name: A
                    `),
                    "b.yaml": heredoc.Doc(`
                        kind: properties
                        version: rudder/v1
                        metadata: {name: b}
                        spec:
                          properties:
                            - id: prop-dup
                              name: B
                    `),
                },
                ExpectedDiagnostics: []docs.ExpectedDiagnostic{
                    {
                        File:            "b.yaml",
                        Reference:       "project/duplicate-urn",
                        Severity:        "error",
                        MessageContains: "duplicate URN",
                    },
                },
            },
        },
    }}
}
```

Add `docs` + `heredoc/v2` to imports.

- [ ] **Step 4: Run — expect pass**

```bash
go test ./cli/internal/project/rules/...
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add cli/internal/project/rules/duplicate_urn_rule.go cli/internal/project/rules/duplicate_urn_rule_doc_test.go
git commit -m "feat(rules/duplicate-urn): implement Documented interface with multi-file example"
```

---

### Task 10: End-to-end smoke run + acceptance verification

After Tasks 0–9, run the full pipeline against the worktree and check spec §13 acceptance.

- [ ] **Step 1: `make lint` green**

```bash
cd /Users/shanmukh/workspace/rudder-iac/.worktrees/dex-370-path-b
make lint
```
Expected: PASS (no new warnings).

- [ ] **Step 2: `make test` green**

```bash
make test
```
Expected: PASS.

- [ ] **Step 3: Manual smoke — run `docs rules`**

```bash
go run ./cli/cmd/rudder-cli docs rules --output-dir /tmp/dex-370-out
```
Expected:
- Stdout: `Wrote rules.yaml and rules.json to /tmp/dex-370-out`
- `/tmp/dex-370-out/rules.yaml` and `rules.json` both exist
- YAML contains the three pilot rules (`datacatalog/categories/spec-syntax-valid`, `project/metadata-syntax-valid`, `project/duplicate-urn`)
- Each rule has at least 1 valid + 1 invalid example
- Re-running produces byte-identical output (reproducibility — `GeneratedAt` was deliberately dropped in Task 2)

- [ ] **Step 4: Sanity-check the acceptance caps from spec §13**

Run:
```bash
git diff --stat origin/feat/dex-269-add-docs-foundation HEAD -- \
  ':(exclude)*_test.go' \
  ':(exclude)docs/superpowers/plans/*' \
  | tail -1
```
Expected: well under 500 LoC added (Task 2's deletes count negative).

Then for authored data:
```bash
git diff origin/feat/dex-269-add-docs-foundation HEAD -- \
  'cli/internal/providers/datacatalog/rules/category/category_spec_valid.go' \
  'cli/internal/project/rules/metadata_syntax_valid.go' \
  'cli/internal/project/rules/duplicate_urn_rule.go' \
  | grep '^+' | grep -v '^+++' | wc -l
```
Expected: under 200 (this is a rough lower bound — only the `DocExamples()` body counts, not the imports or wrapper struct).

- [ ] **Step 5: Verify the Task 2 cycle-break claim is preserved at the trim commit**

Check that the trim commit specifically has zero `validation/rules` imports in docs/ non-test files:

```bash
# Replace <TRIM_SHA> with the commit hash of Task 2's commit.
TRIM_SHA=$(git log --grep="trim PR #471" --format=%H -1)
git show "$TRIM_SHA":cli/internal/validation/docs/rules_doc.go | grep "validation/rules" || echo "OK: no rules import in trim commit"
git show "$TRIM_SHA":cli/internal/validation/docs/types.go | grep "validation/rules" || echo "OK: no rules import in trim commit"
```
Expected: both print "OK: no rules import in trim commit".

This is the spec §13 "post-trim zero imports" acceptance bar. The final HEAD may legitimately import `validation/rules` again (for `rules.Rule` on the Resolver interface) — that's a deliberate, one-way edge, not the accidental one #471 had.

Also verify `rules` does NOT import `docs` at HEAD (the leaf-rule packages do, but the framework `rules` package must not — that's the cycle that the Documented interface is designed to avoid):

```bash
grep -n "validation/docs" cli/internal/validation/rules/*.go
```
Expected: zero matches.

- [ ] **Step 6: Optional final commit if any cleanup**

If any minor polish (gofmt, import organization, comment fixes) is needed:
```bash
go fmt ./...
git add -A
git commit -m "chore: gofmt + final polish for DEX-370 spike"
```

---

## Risk Inventory

The riskiest step is **Task 6 (Verifier)**, specifically the `matchesAny` function — the `Diagnostic` struct does not carry a `Reference` field, so the spike falls back to matching on `File` + `Severity` + `MessageContains` only. This is documented inline with a TODO. The three pilot examples are authored conservatively (one diagnostic each) so the looser match still meaningfully exercises the invariant. If a pilot were authored with multiple diagnostics per file, the matcher could spuriously pass — flagged for the comparison criteria (§9 "verification failure ergonomics") to surface during the spike review.

Second riskiest: **Task 8's `buildRegistry`**. Wiring providers without an API client requires confirming each provider's `New(nil)` returns a usable instance (the rule constructors must not dereference the client at registration time). The plan accepts this risk and tells the executor to verify by trying `datacatalog.New(nil).SyntacticRules()` in a scratch test before wiring — if the constructor panics, the workaround is to import the rule constructor packages directly and skip the provider object entirely.

Third risk: **JSON struct tags on `RulesDoc` and friends** — currently absent. The plan amends Task 2 (the trim commit) to add `json:` tags alongside the `yaml:` tags. If missed, Task 7's round-trip test fails; the fix is fast but easy to forget.

## Notes for the Executor

- Reference @superpowers:test-driven-development for the test-first cadence.
- Reference @superpowers:verification-before-completion before claiming the plan is done — actually run `make test` and `make lint`.
- This plan assumes Path A's sibling DEX-371 worktree is INDEPENDENT — do not try to share infrastructure code. Per spec §10, duplication is deliberate.
- Do NOT push the branch — the executor session decides when.
