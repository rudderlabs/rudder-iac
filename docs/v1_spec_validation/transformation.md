# Transformation - V1 Spec Validation

**Ticket**: [DEX-257 - V1 Spec Validation](https://linear.app/rudderstack/issue/DEX-257/v1-spec-validation)

---

## V0.1 Spec Validation Status

**No validation rules exist** in the new validation engine for transformation resources (any version). The transformations provider does not have a `rules/` directory and does not implement `RuleHandler`.

All existing validations live in the old handler-based framework:
- `cli/internal/providers/transformations/handlers/transformation/handler.go` (ValidateSpec, ValidateResource)

---

## Structural Differences: V0.1 vs V1

There is **no structural difference** between V0.1 and V1. The same `TransformationSpec` struct is used for both versions.

### Struct (`TransformationSpec` in `project/specs/transformation.go`)

```go
type TransformationSpec struct {
    ID          string               `json:"id" mapstructure:"id"`
    Name        string               `json:"name" mapstructure:"name"`
    Description string               `json:"description" mapstructure:"description"`
    Language    string               `json:"language" mapstructure:"language"`
    Code        string               `json:"code,omitempty" mapstructure:"code"`
    File        string               `json:"file,omitempty" mapstructure:"file"`
    Tests       []TransformationTest `json:"tests,omitempty" mapstructure:"tests"`
}

type TransformationTest struct {
    Name   string `json:"name" mapstructure:"name"`
    Input  string `json:"input,omitempty" mapstructure:"input"`
    Output string `json:"output,omitempty" mapstructure:"output"`
}
```

---

## Syntactic Validations to Add for V1

These rules must target `MatchKindVersion("transformation", "rudder/v1")` and decode into `TransformationSpec`.

| # | Validation | Description |
|---|-----------|-------------|
| 1 | `id` required | Transformation must have a non-empty `id` field |
| 2 | `name` required | Transformation must have a non-empty `name` field |
| 3 | `code` and `file` mutually exclusive, one required | Either `code` or `file` must be specified, but not both |
| 4 | `language` required | Transformation must have a non-empty `language` field |
| 5 | `language` in allowed values | Language must be one of: `javascript`, `python` |
| 6 | Each test must have `name` | Every test entry in the `tests` array must have a non-empty `name` |
| 7 | Test name pattern | Test names must match `^[A-Za-z0-9 _/\-]+$` (alphanumeric, spaces, underscores, slashes, hyphens) |
| 8 | Code syntax validation | When `code` is provided (or resolved from `file`), validate syntax using the appropriate parser (esbuild for JavaScript, Python parser for Python) |

---

## Semantic Validations to Add for V1

No semantic validations needed. Transformations have no cross-resource references.

---

## Acceptance Criteria

- [ ] All 8 syntactic validations listed above are implemented as V1 rules targeting `MatchKindVersion("transformation", "rudder/v1")`
- [ ] Code syntax validation integrates with the existing parser infrastructure (`parser.ValidateSyntax()`)
- [ ] All validations are tested with unit tests
- [ ] Test coverage for changed files exceeds 85%

---

## PR Process

**Branch**: `feat/dex-257-transformation-v1-spec-validation`

**Target Branch**: `main`

**PR Template** (from `.github/pull_request_template.md`):

```markdown
## 🔗 Ticket

[DEX-257](https://linear.app/rudderstack/issue/DEX-257/v1-spec-validation)

---

## Summary

Add V1 spec validation rules for the `transformation` resource. These rules target `rudder/v1` specs, implementing 8 syntactic validations including required fields, language constraints, code/file exclusivity, test name patterns, and code syntax validation.

---

## Changes

* Add V1 syntactic rule for transformation spec validation (required fields, language, code/file exclusivity)
* Add V1 syntactic rule for test name validation and code syntax validation

---

## Testing

* Unit tests for all syntactic validations
* Table-driven tests covering valid and invalid V1 transformation specs
* Tests for code syntax validation (valid/invalid JavaScript and Python)

---

## Risk / Impact

Low
V1 validation is new functionality; no rules existed previously for transformations.

---

## Checklist

* [ ] Ticket linked
* [ ] Tests added/updated
* [ ] No breaking changes (or documented)
```
