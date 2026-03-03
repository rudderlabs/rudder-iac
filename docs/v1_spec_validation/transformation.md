# Transformation - V1 Spec Validation

**Ticket**: [DEX-257 - V1 Spec Validation](https://linear.app/rudderstack/issue/DEX-257/v1-spec-validation)

---

## V0.1 Spec Validation Status

**No validation rules exist** in the new validation engine for transformation resources (any version). The transformations provider does not have a `rules/` directory and does not implement `RuleHandler`.

All existing validations live in the old handler-based framework:
- `cli/internal/providers/transformations/handlers/transformation/handler.go` (ValidateSpec, ValidateResource)

---

## Structural Differences: V0.1 vs V1

There is **no structural difference** between V0.1 and V1. The same `TransformationSpec` struct is used for both versions. Since no V0.1 rules exist in the new validation engine for transformations, adding `validate:` tags to the shared structs is safe.

### Struct — Current (`TransformationSpec` in `project/specs/transformation.go`)

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

### Struct — Updated (tags to add)

```go
type TransformationSpec struct {
    ID          string               `json:"id" mapstructure:"id" validate:"required"`
    Name        string               `json:"name" mapstructure:"name" validate:"required"`
    Description string               `json:"description" mapstructure:"description"`
    Language    string               `json:"language" mapstructure:"language" validate:"required,oneof=javascript python"`
    Code        string               `json:"code,omitempty" mapstructure:"code"`
    File        string               `json:"file,omitempty" mapstructure:"file"`
    Tests       []TransformationTest `json:"tests,omitempty" mapstructure:"tests" validate:"omitempty,dive"`
}

type TransformationTest struct {
    Name   string `json:"name" mapstructure:"name" validate:"required"`
    Input  string `json:"input,omitempty" mapstructure:"input"`
    Output string `json:"output,omitempty" mapstructure:"output"`
}
```

---

## Syntactic Validations to Add for V1

These rules must target `MatchKindVersion("transformation", "rudder/v1")` and decode into `TransformationSpec`.

### Tag-Based (handled by `rules.ValidateStruct()`)

| # | Validation | Tag | Description |
|---|-----------|-----|-------------|
| 1 | `id` required | `validate:"required"` on `ID` | Transformation must have a non-empty `id` |
| 2 | `name` required | `validate:"required"` on `Name` | Transformation must have a non-empty `name` |
| 3 | `language` required + oneof | `validate:"required,oneof=javascript python"` on `Language` | Language must be one of the allowed values |
| 4 | Tests dive | `validate:"omitempty,dive"` on `Tests` | Recursively validates each `TransformationTest` |
| 5 | Test `name` required | `validate:"required"` on `TransformationTest.Name` | Each test must have a non-empty name |

### Custom Logic (manual rule code)

| # | Validation | Description |
|---|-----------|-------------|
| 6 | `code` and `file` mutually exclusive, one required | Either `code` or `file` must be specified, but not both |
| 7 | Test name pattern | Test names must match `^[A-Za-z0-9 _/\-]+$` (alphanumeric, spaces, underscores, slashes, hyphens) |
| 8 | Code syntax validation | When `code` is provided (or resolved from `file`), validate syntax using the appropriate parser (esbuild for JavaScript, Python parser for Python) |

---

## Semantic Validations to Add for V1

No semantic validations needed. Transformations have no cross-resource references.

---

## Acceptance Criteria

- [ ] `validate:` tags added to `TransformationSpec` and `TransformationTest` shared structs (safe since no V0.1 rules exist in new engine)
- [ ] V1 syntactic rule uses `rules.ValidateStruct()` for tag-based validations (#1-#5)
- [ ] Custom logic implemented for code/file exclusivity (#6), test name pattern (#7), and code syntax validation (#8)
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

* Add `validate:` tags to `TransformationSpec` and `TransformationTest` shared structs
* Add V1 syntactic rule using `rules.ValidateStruct()` for tag-based validations + custom logic for code/file exclusivity, test name pattern, and code syntax validation

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
