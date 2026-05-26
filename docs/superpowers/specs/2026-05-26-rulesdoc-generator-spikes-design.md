# RulesDoc Generator — Spike Comparison Design

**Date:** 2026-05-26
**Status:** Draft (revision 2)
**Author:** Siva Shanmukh Vetcha
**Related:**
- Notion: [Validations V1 — auto-generated docs publication](https://www.notion.so/rudderstacks/Validations-V1-auto-generated-docs-publication-d12f2b415dd0834cb9c88164acbf6b7f)
- Notion: [Yaml Validation Framework PRD](https://www.notion.so/rudderstacks/2d8f2b415dd080ac96a9d9e8353d113e)
- Notion: [CLI Validation Framework LLD](https://www.notion.so/rudderstacks/2e2f2b415dd080e4b9b1f586e8645064)
- Linear: [DEX-216 RuleDoc Generator (epic)](https://linear.app/rudderstack/issue/DEX-216)
- Linear: [DEX-269 Phase 1: docs package foundation](https://linear.app/rudderstack/issue/DEX-269)
- Linear: [DEX-270 Phase 2: registry extension + RuleProvider interface](https://linear.app/rudderstack/issue/DEX-270)
- GitHub PR: [#471 DEX-269 foundation (OPEN)](https://github.com/rudderlabs/rudder-iac/pull/471)
- GitHub PR: [#475 DEX-270 registry extension (OPEN, stacked on #471)](https://github.com/rudderlabs/rudder-iac/pull/475)

> **Naming convention used throughout this spec:** The resolved artifact's top-level type is `RulesDoc` and the per-rule type is `ResolvedRule` (rename introduced by PR #475). PR #471 currently calls them `DocumentedRules` / `DocumentedRule`; both spikes stack on the renamed versions from #475.

---

## 1. Goal

Re-evaluate the in-flight RuleDoc Generator design (DEX-269/270) by building **two sibling spike PRs** that demonstrate the same end-to-end pipeline using two different *authoring* strategies. The spikes are the evidence that lets us pick a direction with confidence before investing in full rollout across all rules.

The pipeline both spikes deliver:

> Registered validation rules → authored doc data → generated `RulesDoc` artifact → executable verification (invalid examples actually produce expected diagnostics) → serialized as JSON (LLMs) and YAML (Hugo / humans) → committed to `./docs/generated/`.

Hugo consumes the YAML downstream and renders markdown for `rudderstack.com/docs`. The publication/PR-to-docs-repo step is **out of scope for the spikes** because it does not differ between paths.

### Why not just finish DEX-269/270 as planned

The Notion project page ("auto-generated docs publication") describes the remaining work as "lightweight, docs already generated, just publishing + positioning." That framing is stale: the docs are not generated. DEX-269 lays a foundation but is biased toward one specific authoring strategy (YAML fragments). Before continuing to build on that bias, we want a direct comparison against a simpler in-Go alternative.

### Property that must hold

**Executable docs — docs cannot drift from code.** Each invalid example is verified at generation time by running it through the existing `ValidationEngine` and asserting that the produced diagnostics satisfy the authored `ExpectedDiagnostic`s. The verification uses **subset semantics** (every authored expectation must appear in produced output; extras are allowed — see §5). Generation fails if any expected diagnostic is missing. Both spikes must demonstrate this property; the spike that cannot is disqualified.

---

## 2. Architecture — the `Resolver` seam

Both spikes share the same pipeline. The only thing that differs is **where the authored doc data comes from**. That difference is encoded behind a single interface:

```
┌────────────────────────────────────────────────────────────────────┐
│            rudder-cli docs rules  (new top-level command)          │
│                                                                    │
│  1. Walk registry           → []rules.Rule (syntactic + semantic)  │
│  2. Resolve doc data        → per rule, get authored *ResolvedRule │
│                               (via the Resolver — only seam)       │
│  3. Build RulesDoc          → enrich with metadata from rule       │
│  4. Validate structurally   → checks 1-3 (struct tags, applies-to  │
│                               coverage, unique example IDs).       │
│                               Check 4 (completeness) commented out │
│                               for spike phase (see §6/§8).         │
│  5. Verify executably       → run engine on every invalid example, │
│                               subset-match produced diagnostics    │
│                               against authored ExpectedDiagnostics │
│  6. Emit artifact           → JSON + YAML to ./docs/generated/     │
└────────────────────────────────────────────────────────────────────┘
                                ▲
                                │  Resolver is the only seam between paths
                ┌───────────────┴────────────────┐
                │                                │
         Path A: YAMLResolver             Path B: ExamplesResolver
         loads <rule>.docs.yaml           type-asserts rule.(Documented)
         from filesystem                  and reads DocExamples()
```

Steps 1, 3, 4, 5, 6 are **identical** in both spikes. Step 2 is the only thing that differs.

### Resolver interface

Defined in `cli/internal/validation/docs/`:

```go
type Resolver interface {
    // ResolveFor returns the authored doc data for the given rule,
    // or (nil, nil) if no docs are authored for it.
    // Errors indicate authoring or load problems, not missing docs.
    ResolveFor(r rules.Rule) (*RuleDocEntry, error)
}
```

Both spikes implement this same interface. The generator depends only on the interface; nothing in steps 3-6 cares which path's resolver it received.

---

## 3. Layer-by-layer breakdown — versus PR #471 / #475

### Layer 1 — Resolved artifact types

**In PR #471 (+ #475 rename):** `RulesDoc`, `ResolvedRule`, `ToolMetadata` defined with `validate` tags in [types.go](cli/internal/validation/docs/types.go).

**Status:** Both spikes reuse these but **both** trim the same two fields (`ResolvedRule.Provider`, `ToolMetadata.GeneratedAt`) and add `json` tags alongside existing `yaml` tags. This is part of the **shared first commit** — see §6.1 / §7.1 which describe identical content.

### Layer 2 — Shared inner types

**In PR #471:** `MatchBehaviorEntry`, `MatchPatternDoc`, `ValidExample`, `InvalidExample`, `ExpectedDiagnostic`.

**Status:** Path-agnostic. Both paths reuse them in-place inside the `docs` package. Path B's `Documented` interface (§6) returns `[]MatchBehaviorEntry`.

### Layer 3 — `RulesDoc.Validate(expectedRuleIDs)`

**In PR #471:** 4 structural checks in [rules_doc.go](cli/internal/validation/docs/rules_doc.go) — `validateRuleStruct`, `validateAppliesToCoverage`, `validateUniqueExampleIDs`, `validateRegisteredCompleteness`.

**Status for spikes:**
- Checks 1-3 reused as-is.
- Check 4 (`validateRegisteredCompleteness`) is **commented out** in both spike PRs with a clear marker (`// TODO(spike DEX-XXX): re-enable after pilot phase`). It would otherwise fail every spike run because only 3 of ~13 registered rules will have docs. Restoration is one line.
- The parameter `registeredRuleIDs` is renamed to `expectedRuleIDs` to reflect that the call site decides what set to enforce — semantic clarification only, no behavioral change while the check is commented out.

Both spike PRs include this trivial edit; for Path B it rolls into the trim commit, while for Path A it's a single dedicated commit.

### Layer 4 — Authored wrapper `RuleDocEntry`

**In PR #471:** Defined in `types.go`. The per-rule unit that providers/loaders contribute (rule_id + match_behavior list).

**Status:** Path A uses it. Path B's trim commit removes it from the package — Path B's `ExamplesResolver` returns a `*ResolvedRule` directly (no wrapper needed; the resolved type has the same `MatchBehavior` shape).

### Layer 5 — `LoadRuleDocEntries` YAML loader

**In PR #471:** Reads `.yaml`/`.yml` from a directory via `fs.FS` in [rule_doc_entry.go](cli/internal/validation/docs/rule_doc_entry.go).

**Status:** Path A uses it (wrapped by `YAMLResolver`). Path B's trim commit removes it (and its tests).

### Layer 6 — Registry `AllSyntacticRules() / AllSemanticRules()`

**In PR #475:** Added to the `Registry` interface with defensive copies.

**Status:** Both spikes need to enumerate rules. **Both spikes stack on top of #475's registry-extension portion.** The `RuleProvider.RuleDocEntries()` portion of #475 (Layer 7) is dropped — §4 spells out the exact file-level split.

### Layer 7 — `RuleProvider.RuleDocEntries() []docs.RuleDocEntry`

**In PR #475:** Adds the method to `RuleProvider`; `CompositeProvider` aggregates; `EmptyProvider`/`MockProvider` return `nil`.

**Status:** Both spikes **drop this**. Path A's `YAMLResolver` reads YAML files directly without going through the provider interface, and Path B doesn't go through providers at all. Keeping docs concerns out of the apply-cycle interface surface is a substantive design improvement over DEX-270's original plan.

How "dropping it" is operationalized: §4 spells out splitting #475.

### Layer 8 — `Resolver` interface

**Not in #471/#475.** New. Defined in `cli/internal/validation/docs/`. Both spikes add identical code.

### Layer 9 — Generator, Verifier, Serializer, CLI command

**Not in #471/#475.** All new. Both spikes add identical code (the duplication is intentional — see §9 "Deliberate duplication").

---

## 4. Pre-spike work — what lands first

The two spike PRs both **stack on a common base**: PR #471 plus the registry-extension portion of #475. Concretely:

1. **PR #471 (DEX-269) merges as-is.** Lands the docs package foundation: resolved types, inner types, `RulesDoc.Validate`, authored `RuleDocEntry`, `LoadRuleDocEntries`. Additive, low-risk.

2. **PR #475 is split:**
    - **Land a new, separate PR containing only the registry extensions** (`AllSyntacticRules` / `AllSemanticRules` on `rules.Registry` + tests). This is `registry.go` and `registry_test.go` from #475's diff — clean carve-out.
    - **PR #475 is then either closed or kept open for record only.** The provider-interface portion (`RuleDocEntries()` on `RuleProvider`, `CompositeProvider` aggregation, `EmptyProvider`/`MockProvider` nil-defaults) is rejected as a design choice and does not merge.
    - Concrete files dropped from #475: `cli/internal/provider/provider.go`, `cli/internal/provider/composite.go`, `cli/internal/provider/composite_test.go`, `cli/internal/provider/emptyprovider.go`, `cli/internal/testutils/mockprovider.go`, and the registry-extension files now ship in a separate PR.

These two predecessor merges are the "common foundation." Both spike PRs branch from this state.

---

## 5. Verifier semantics — subset by default

For each authored `InvalidExample` on a rule, the Verifier:

1. Materializes the example's `Files` map into a tmpdir as a tiny RudderStack project.
2. Loads the project (using the existing project loader).
3. Runs `ValidationEngine.ValidateSyntax` + `ValidateSemantic`.
4. Subset-matches produced diagnostics against authored `ExpectedDiagnostic`s.

**Subset match definition:** every authored `ExpectedDiagnostic` must match at least one produced diagnostic. Match means: same `severity`, same `file`, and produced `message` contains the `MessageContains` substring. Extra produced diagnostics not in the authored list are **ignored** (no failure).

**Why `reference` is not in the match key:** the runtime `validation.Diagnostic` struct does not preserve the authored JSON-pointer reference — the engine resolves `ValidationResult.Reference` through the path indexer into a `Position` (line, column) before producing the diagnostic (see [engine.go:204-225](cli/internal/validation/engine.go)). The authored `ExpectedDiagnostic.Reference` field is retained in the artifact as **documentation-only metadata** (it tells human readers and LLMs *where* the rule fires), but the verifier doesn't match against it. A future enhancement could resolve the authored reference through the materialized example's path index to derive an expected `Position` and compare positions; out of scope for the spike.

**Strict mode (designed-in, deferred):** the CLI command names a `--strict-verify` flag in its help text but the flag is unimplemented in the spike. When implemented later, it switches to exact match (produced set must equal authored set). This seam preserves the option without committing the spike to it.

Why subset by default: project-level rules in the validation framework fire across multiple files. Exact match would cause unrelated rule additions to break existing example fixtures, creating maintenance churn without catching real drift in the rule those fixtures document.

---

## 6. Path A spike PR — YAML fragments

### Stacking

Branch: `feat/docs-rule-gen-spike-path-a` off `feat/dex-269-add-docs-foundation` (PR #471's branch) plus the registry-extension PR (§4 step 2).

### Commits

**Commit 1 — Shared prep** (identical content to Path B's commit 1 — §7.1). Eight items:
1. Rename `DocumentedRules` → `RulesDoc`, `DocumentedRule` → `ResolvedRule` (#475 rename direction).
2. Drop `ResolvedRule.Provider` field. **Why:** forces every rule to be owned by a provider; project-level rules don't belong to one. Rule ID prefix already encodes provenance.
3. Drop `ToolMetadata.GeneratedAt` field. **Why:** runtime timestamp makes the artifact non-reproducible — hostile to committed/PR-able output.
4. Add `json` struct tags alongside existing `yaml` tags on all serializable types in `types.go`. **Why:** the spike emits both YAML (Hugo) and JSON (LLM); both need snake_case keys.
5. Replace `rules.ValidateStruct(rule, "")` call in `docs/rules_doc.go` with direct `validator.New()` usage in the `docs` package. **Why:** the only `docs → rules` import in #471. Required for Path B (cycle), kept symmetric for Path A (removes unjustified cross-package coupling).
6. Comment out `validateRegisteredCompleteness` invocation in `RulesDoc.Validate` with `// TODO(spike DEX-371): re-enable after pilot phase` marker.
7. Rename parameter `registeredRuleIDs` → `expectedRuleIDs`.
8. Add `AllSyntacticRules()` / `AllSemanticRules()` methods to `rules.Registry` interface + `defaultRegistry` (defensive-copy returns). **Why:** the spike enumerates the full rule set; the registry-extension carve-out PR (§4) hasn't landed, so both spikes include this directly.

**Commit 2 — Resolver interface:** Add `Resolver` interface + `YAMLResolver` impl in `cli/internal/validation/docs/`. `YAMLResolver` wraps `LoadRuleDocEntries` (loads all fragments at construction) and indexes results by `RuleID`. Returns `*ResolvedRule` from `ResolveFor` — converting from `RuleDocEntry` by enriching with rule metadata.

**Commit 3 — Generator + Validator wiring:** `generator.go` walks registry (`AllSyntacticRules` ∪ `AllSemanticRules`), calls `resolver.ResolveFor` per rule, builds `RulesDoc`. Enriches each `ResolvedRule` with `Phase` (from which registry list it came from), `Severity`, `Description`, `AppliesTo` (from `rules.Rule`).

**Commit 4 — Verifier:** `verifier.go` — for each `InvalidExample`, materializes a tmpdir from `Files`, loads it as a project, runs `ValidationEngine.ValidateSyntax` + `ValidateSemantic`, subset-matches produced diagnostics against `ExpectedDiagnostic`s.

**Commit 5 — Serializer:** `serializer.go` — emits `RulesDoc` as YAML and JSON. No markdown rendering; Hugo handles that.

**Commit 6 — CLI command:** `cli/internal/cmd/docs/rules/` — new top-level subcommand `rudder-cli docs rules`. Flags: `--fragments-dir` (default `cli/internal/validation/docs/fragments/`), `--output-dir` (default `./docs/generated/`), `--strict-verify` (named in help text but unimplemented; emits a clear error when invoked). Wires into the root command tree under a new `docs` group.

**Commit 7 — Three pilot fragments authored** (data files, not Go):
- `cli/internal/validation/docs/fragments/datacatalog-categories-spec-syntax-valid.docs.yaml` — covers `(categories, rudder/v0.1)` and `(categories, rudder/v1)` in two `match_behavior` entries.
- `cli/internal/validation/docs/fragments/project-metadata-syntax-valid.docs.yaml` — covers all kinds via wildcard `MatchPatternDoc {kind: "*", version: "*"}`.
- `cli/internal/validation/docs/fragments/project-duplicate-urn.docs.yaml` — multi-file invalid example using `Files: {a.yaml: ..., b.yaml: ...}` map.

### What stays untouched

- No changes to any `rules.Rule` implementation.
- No changes to the `RuleProvider` interface.
- Existing rule tests untouched.
- The existing `rules.Rule.Examples() Examples` method (used by the engine to attach to diagnostics — see [engine.go:217-224](cli/internal/validation/engine.go) and [text.go](cli/internal/validation/renderer/text.go)) remains in place, unaffected. The docs pipeline ignores it; the renderer continues to consume it.

### Estimated diff shape (Path A)

| Area | Files added/modified | Lines (rough) |
|---|---|---|
| Shared prep commit (8 items) | ~4 modified | ~40 |
| Resolver + YAMLResolver | 2 added | ~80 |
| Generator + Verifier + Serializer | 3 added | ~250 |
| CLI command | 1-2 added | ~80 |
| 3 YAML fragment files | 3 added | ~150 (authored data) |
| Tests for new code | 4-5 added | ~400 |
| **Total** | ~16 files | ~1000 LoC |

Code (non-test, non-data) ≈ 450 LoC; under the 500 cap. Authored data ≈ 150 LoC; under the 200 cap.

---

## 7. Path B spike PR — separate `Documented` interface

### Stacking

Branch: `feat/docs-rule-gen-spike-path-b` off `feat/dex-269-add-docs-foundation` (PR #471's branch) plus the registry-extension PR (§4 step 2).

### Commit 1: Shared prep (identical content to Path A's commit 1 — §6.1)

Same 8-item list as Path A's commit 1; the full enumeration is in §6.1 and not repeated here. Key points specific to Path B:

- **The cycle-breaking item is load-bearing on Path B.** Replacing `rules.ValidateStruct(rule, "")` with `validator.New()` in `docs/rules_doc.go` dissolves the only `docs → rules` import. This is what allows the `Documented` interface (commit 3 below) to live in the `docs` package without creating a cycle when leaf rule packages import `docs.MatchBehaviorEntry`.
- **What survives in the docs package after the shared prep:** `types.go` (resolved types minus `Provider` and `GeneratedAt`, plus `json` tags), `rules_doc.go` (`RulesDoc.Validate` with checks 1-3, completeness commented out, `validator.New()` instantiated locally), `rule_doc_entry.go` (still present — Path B removes it in commit 2 below), `rules_doc_test.go`, `rule_doc_entry_test.go` (also still present — also removed in commit 2).
- The shared prep is conceptually identical content across both branches but each will have its own SHA due to different commit histories. Literal SHA identity would require a common ancestor commit — out of scope; not worth the coordination overhead.

### Commit 2: Path-B-only deletions

Path B doesn't need #471's YAML authoring surface. Removed in a dedicated commit so the "removed surface area" criterion in §9 reads as a single, isolated diff (rather than mixing path-specific deletions into the shared prep).

| Removal | Why |
|---|---|
| Remove `RuleDocEntry` type from `types.go` | Path B rules expose docs directly via the `Documented` interface (commit 3). No external authored wrapper needed. |
| Remove `LoadRuleDocEntries` function (entire `rule_doc_entry.go` file) | Path B has no on-disk YAML fragments to load. |
| Remove `rule_doc_entry_test.go` | Tests for the removed loader. |

After this commit: only Path-B-relevant surface remains in the `docs` package.

### Commit 3: `Documented` interface

Define an opt-in sibling interface in the docs package, with **no change** to `rules.Rule`:

```go
// In cli/internal/validation/docs/

// Documented is an optional interface for rules that author structured
// documentation data. The docs pipeline type-asserts rules to this
// interface; rules that don't implement it produce no doc entry.
//
// The existing rules.Rule.Examples() Examples method is unrelated:
// it returns plain strings attached to diagnostics at runtime and is
// consumed by the text renderer for end-user error output.
type Documented interface {
    DocExamples() []MatchBehaviorEntry
}
```

Critically, `rules` does **not** import `docs`. The `Documented` interface lives in `docs`. A rule implementation in (say) `cli/internal/providers/datacatalog/rules/category/` imports `docs` to gain access to `docs.MatchBehaviorEntry` when authoring its `DocExamples()` return — but `cli/internal/validation/rules/` (the framework package) does not.

This is why dissolving the `docs → rules` dep in commit 1 was important: it allows the leaf rule packages to freely import `docs` without creating a cycle with `rules`.

### Commit 4: `ExamplesResolver`

```go
// In cli/internal/validation/docs/

type ExamplesResolver struct{}

func (ExamplesResolver) ResolveFor(r rules.Rule) (*ResolvedRule, error) {
    d, ok := r.(Documented)
    if !ok {
        return nil, nil // no docs authored
    }
    behavior := d.DocExamples()
    if len(behavior) == 0 {
        return nil, nil
    }
    return &ResolvedRule{ MatchBehavior: behavior }, nil
}
```

The Generator (commit 5) enriches the returned `*ResolvedRule` with `RuleID`, `Phase`, `Severity`, `Description`, `AppliesTo` from `rules.Rule` — same as Path A.

### Commits 5-7: Generator + Verifier + Serializer + CLI

Same as Path A. Identical code (deliberate duplication — see §9). The CLI command is `rudder-cli docs rules` with `--output-dir` and `--strict-verify` flags only; no `--fragments-dir` flag (there is no fragments directory in Path B).

### Commit 8: Three pilot rules implement `Documented`

The three pilot rules' Go files are edited to add a `DocExamples()` method:

- [datacatalog/categories/spec-syntax-valid](cli/internal/providers/datacatalog/rules/category/category_spec_valid.go) — implement `DocExamples()` returning a structured `[]docs.MatchBehaviorEntry` literal covering v0.1 and v1. The existing `examples` package var (returned by the existing `Examples() Examples` method) stays untouched.
- [project/metadata-syntax-valid](cli/internal/project/rules/metadata_syntax_valid.go) — add `DocExamples()` method on `*MetadataSyntaxValidRule` returning structured match behavior with wildcard `*` `AppliesTo`. The existing `Examples()` method body stays untouched.
- [project/duplicate-urn](cli/internal/project/rules/duplicate_urn_rule.go) — add `DocExamples()` on `*duplicateURNRule` with multi-file invalid example using `Files: map[string]string{"a.yaml": ..., "b.yaml": ...}`. Rule ID is `project/duplicate-urn`.

YAML content within Go literals uses `heredoc.Doc` for readability — already a project convention (see the existing `Examples()` body in `metadata_syntax_valid.go`).

### What stays untouched

- The `rules.Rule` interface (no method added).
- The `RuleProvider` interface (no `RuleDocEntries()` method ever added).
- The validation engine itself.
- The existing `rules.Rule.Examples() Examples` method, its callers in `engine.go`, and the text renderer — Path B doesn't touch the runtime example-attachment behavior.
- Every rule **except** the three pilots — they're unchanged. The `Documented` interface is opt-in via type assertion; non-implementers are silently skipped by the resolver.
- `rules.ValidateStruct` — still used by 11+ semantic rule implementations across the providers (e.g., [category_spec_valid.go:31](cli/internal/providers/datacatalog/rules/category/category_spec_valid.go)). Only its use *inside* `docs/rules_doc.go` is replaced.

### Estimated diff shape (Path B)

| Area | Files added/modified | Lines (rough) |
|---|---|---|
| Commit 1 — Shared prep (8 items) | ~4 modified | ~40 |
| Commit 2 — Path-B-only deletions | 2 modified, 2 deleted | -170 net |
| `Documented` interface | 1 modified | ~10 |
| Resolver + ExamplesResolver | 2 added | ~50 |
| Generator + Verifier + Serializer | 3 added | ~250 |
| CLI command | 1-2 added | ~70 |
| 3 pilot rules edited (additive method) | 3 modified | ~150 (authored data) |
| Tests for new code | 4-5 added | ~400 |
| **Total** | ~19 files | ~800 LoC net |

Code (non-test, non-data) ≈ 420 LoC; under the 500 cap (trim deletes count net). Authored data ≈ 150 LoC; under the 200 cap.

---

## 8. Worked example — what the three pilot rules look like

This section is illustrative — exact wording locked during implementation.

### `datacatalog/categories/spec-syntax-valid` (per-version)

**Authored shape (both paths):**
- 2 `MatchBehaviorEntry`s — one for `(categories, rudder/v0.1)`, one for `(categories, rudder/v1)`.
- Each has 1 valid + 2 invalid examples.
- Each invalid example has 1 expected diagnostic (the kind of missing-field or wrong-type error the rule catches at that version).

Path A: single YAML file `datacatalog-categories-spec-syntax-valid.docs.yaml` with two `match_behavior` entries.

Path B: implement `func (r *categorySpecRule) DocExamples() []docs.MatchBehaviorEntry { return []docs.MatchBehaviorEntry{ {AppliesTo: ...v0.1...}, {AppliesTo: ...v1...} } }`.

### `project/metadata-syntax-valid` (applies-to-all)

**Authored shape:**
- 1 `MatchBehaviorEntry` with `AppliesTo: [{Kind: "*", Version: "*"}]`.
- 2 valid examples (basic, with import block).
- 2 invalid examples (missing name, missing workspace_id).

This pilot exists to prove both spikes handle wildcard match patterns correctly — the artifact must serialize `kind: "*"` cleanly.

### `project/duplicate-urn` (multi-file)

**Authored shape:**
- 1 `MatchBehaviorEntry` with wildcard `AppliesTo`.
- 1 valid example (2 files, distinct URNs).
- 1 invalid example with `Files: {a.yaml: ..., b.yaml: ...}` where both define the same URN. Expected diagnostic points at the second file's offending position.

This pilot exists to prove both spikes handle the cross-file `ProjectRule` case (the multi-spec validation path in [engine.go:88](cli/internal/validation/engine.go)).

---

## 9. Comparison criteria

When both spike PRs are open, the comparison reads:

| Dimension | What we look at | Why it matters |
|---|---|---|
| **Total non-test diff size** | Lines added in PR diff excluding `_test.go` files | First-order proxy for "code we have to maintain" |
| **Per-rule authoring cost** | Lines required to add docs for 1 new rule (pick a 4th rule and add it both ways post-merge of one path) | Most important number — this is what the team lives with going forward |
| **Authoring locality** | Does the rule's docs live near its `.go` file? Can a code reviewer see them in one PR diff view? | Affects review and refactor friction |
| **Refactor blast radius** | Rename a rule ID. How many files change in each path? | Maintenance health |
| **IDE / editor support** | Editing the YAML file vs editing Go literal — which has better syntax highlighting, validation, autocomplete? | Author ergonomics |
| **Schema enforcement timing** | Path A catches at load+validate-time (runtime); Path B catches at compile-time. When does each fail? | Compile-time is stricter but less flexible |
| **Verification failure ergonomics** | When an invalid example stops producing the expected diagnostic, what does the error message look like? Is the offending example easy to locate? | Day-2 ops |
| **Package coupling** | After Path B's trim commit dissolves `docs → rules`, leaf rule packages import `docs` for `MatchBehaviorEntry`. Is that coupling acceptable? Compare to Path A where leaf packages import nothing new | Architectural cost |
| **Migration cost** | What does it take to add docs for the remaining ~10 rules after the pilot? | Total cost-to-finish, not just the spike's cost |
| **Removed surface area** | Path B's trim removes `RuleDocEntry`, `LoadRuleDocEntries`, two struct fields, and one cross-package dep. Path A keeps all of those. Is the removed surface area a win or a loss? | Net design impact |

We do **not** measure: performance, test-coverage delta, or CI runtime. Those are second-order and roughly equivalent between paths.

---

## 10. Deliberate duplication

The Resolver interface, Generator, Verifier, Serializer, and CLI command code is **identical** between the two spike PRs. We deliberately do not extract this into a third shared PR.

Reasons:
- Spikes are throwaway-ish; we expect to discard one of them. Extracting shared infrastructure first commits us to keeping that direction before we've made the decision.
- Each spike PR is self-contained — reviewable end-to-end without context-switching.
- When the winner is picked, the loser's PR is closed, and the winner's PR can be cleaned up and merged through normal review. The "extract shared code" refactor happens only if and when a second consumer appears.

---

## 11. Out of scope

Explicitly **not** in either spike PR:

- **Hugo integration / docs-site PR pipeline.** A GitHub Action that takes the emitted YAML and opens a PR to the RudderStack docs repo. This step is identical for both paths and follows after a path is chosen.
- **Rolling docs out to the remaining rules.** Only the three pilot rules are authored. Other registered rules will have no doc entry — which is why `validateRegisteredCompleteness` is commented out for the spike (see §3 Layer 3, §6 commit 1).
- **Strict-verify mode.** The `--strict-verify` flag is named in the CLI help but unimplemented in both spikes (returns an error when invoked). Verifier runs in subset mode only.
- **CLI flag ergonomics.** The `rudder-cli docs rules` command is minimal — no `--severity`, no `--format` choices beyond emitting both JSON and YAML. Polish is post-spike.
- **LSP / IDE consumption** of the emitted JSON. Producing the JSON in a shape LLMs can consume is sufficient; wiring it into the LSP server is later work.
- **Customer-configurable rules** (PRD section 5.7). Out of scope for Validations V1 entirely.
- **Unifying or removing the existing `rules.Rule.Examples() Examples` method.** Path B adds a parallel `Documented` interface; the existing method stays untouched in both spikes. If Path B wins, a follow-on PR can decide whether to unify, deprecate, or leave alongside.

---

## 12. Open implementation questions

These do not block this design but are decided during implementation:

1. **YAML fragment discovery convention for Path A.** Filename format: `<rule-id-with-slashes-replaced-by-dashes>.docs.yaml`? Subdirectories mirroring rule ID namespaces? Locked during implementation; no design impact.
2. **CLI command path.** `cli/internal/cmd/docs/rules/` vs `cli/internal/cmd/docs-rules/`. Locked during implementation.
3. **Project loader reuse in Verifier.** The Verifier needs to load a tmpdir as a "project" to run the engine against. The existing project loader expects a directory of YAML files — exactly what the Verifier materializes. Reuse it; if its API is awkward for in-memory invocation, document the awkwardness rather than refactor in the spike.
4. **Spike-only naming for `--strict-verify`.** The flag is unimplemented but named. When invoked, it should return a clean error (e.g., "strict-verify mode not implemented in spike; tracked in DEX-XXX") rather than be hidden, so the design intent is discoverable.

---

## 13. Acceptance criteria for the spikes

Both spike PRs must, at minimum:

- [ ] `rudder-cli docs rules` runs cleanly with default flags and emits `docs/generated/rules.yaml` and `docs/generated/rules.json`.
- [ ] The artifact passes `RulesDoc.Validate` (zero errors) for the three pilot rules — where "passes" means checks 1-3 (struct validation, applies-to coverage, unique example IDs) succeed. Check 4 (completeness) is commented out for the spike (§3 Layer 3, §6 commit 1).
- [ ] The Verifier runs each authored invalid example through `ValidationEngine` and asserts every `ExpectedDiagnostic` matches at least one produced diagnostic (subset semantics, §5). Generation fails if any expectation is missing.
- [ ] All three pilot rules are authored in the path's chosen format with matching coverage (same number of valid/invalid examples and same `ExpectedDiagnostic`s as the sibling PR's authoring).
- [ ] `make test` and `make lint` are green.
- [ ] **Code cap**: non-test, non-authored-data diff is under 500 lines added. Excludes `_test.go` files, YAML fragment files (Path A), and the Go literal `[]docs.MatchBehaviorEntry` bodies inside pilot rules' `DocExamples()` methods (Path B). Measures only the spike's *infrastructure code* — Resolver, Generator, Verifier, Serializer, CLI command, package edits.
- [ ] **Authored-data cap**: per-spike authored data across all three pilot rules is under 200 lines. Symmetric across paths: counts YAML for Path A and Go literal data for Path B. This is what §9's "per-rule authoring cost" comparison measures, so it gets its own bound.

The two caps together are the objective version of "reviewable in a single sitting." Splitting code vs authored data prevents the comparison from being biased by whichever encoding is more verbose at the data level.

---

## 14. Decision after spikes

After both PRs are open:

1. Walk the comparison-criteria table (§9) row by row using each PR as evidence.
2. Pick a winner. Document the decision (with the table) in a short follow-on PR or Linear comment.
3. Close the losing PR. Land the winner.
4. Plan follow-on work: roll docs out to the remaining rules, restore `validateRegisteredCompleteness`, decide on strict-verify mode, build the Hugo PR pipeline, scope the docs-team handoff per PRD FR-029–FR-031.
