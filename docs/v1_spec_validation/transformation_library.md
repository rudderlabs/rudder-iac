# Transformation Library - V1 Spec Validation

**Ticket**: [DEX-257 - V1 Spec Validation](https://linear.app/rudderstack/issue/DEX-257/v1-spec-validation)

---

## V0.1 Spec Validation Status

**No validation rules exist** in the new validation engine for transformation library resources (any version). The transformations provider does not have a `rules/` directory and does not implement `RuleHandler`.

All existing validations live in the old handler-based framework:
- `cli/internal/providers/transformations/handlers/library/handler.go` (ValidateSpec, ValidateResource)

---

## Structural Differences: V0.1 vs V1

There is **no structural difference** between V0.1 and V1. The same `TransformationLibrarySpec` struct is used for both versions. Since no V0.1 rules exist in the new validation engine for transformation libraries, adding `validate:` tags to the shared struct is safe.

### Struct — Current (`TransformationLibrarySpec` in `project/specs/transformation.go`)

```go
type TransformationLibrarySpec struct {
    ID          string `json:"id" mapstructure:"id"`
    Name        string `json:"name" mapstructure:"name"`
    Description string `json:"description" mapstructure:"description"`
    Language    string `json:"language" mapstructure:"language"`
    Code        string `json:"code,omitempty" mapstructure:"code"`
    File        string `json:"file,omitempty" mapstructure:"file"`
    ImportName  string `json:"import_name" mapstructure:"import_name"`
}
```

### Struct — Updated (tags to add)

```go
type TransformationLibrarySpec struct {
    ID          string `json:"id" mapstructure:"id" validate:"required"`
    Name        string `json:"name" mapstructure:"name" validate:"required"`
    Description string `json:"description" mapstructure:"description"`
    Language    string `json:"language" mapstructure:"language" validate:"required,oneof=javascript python"`
    Code        string `json:"code,omitempty" mapstructure:"code" validate:"required_without=File,excluded_with=File"`
    File        string `json:"file,omitempty" mapstructure:"file" validate:"required_without=Code,excluded_with=Code"`
    ImportName  string `json:"import_name" mapstructure:"import_name" validate:"required"`
}
```

---

## Syntactic Validations to Add for V1

These rules must target `MatchKindVersion("transformation-library", "rudder/v1")` and decode into `TransformationLibrarySpec`.

> Validation operates on the raw YAML-decoded `TransformationLibrarySpec`. When `file` is set, it must be validated as a spec-relative path using the same rules as the transformation resource: relative to the YAML spec file's directory, with absolute paths and `..` traversal rejected.

### Tag-Based (handled by `rules.ValidateStruct()`)

| # | Validation | Tag | Description |
|---|-----------|-----|-------------|
| 1 | `id` required | `validate:"required"` on `ID` | Library must have a non-empty `id` |
| 2 | `name` required | `validate:"required"` on `Name` | Library must have a non-empty `name` |
| 3 | `import_name` required | `validate:"required"` on `ImportName` | Library must have a non-empty `import_name` |
| 4 | `language` required + oneof | `validate:"required,oneof=javascript python"` on `Language` | Language must be one of the allowed values |
| 5 | `code`/`file` mutual exclusivity | `validate:"required_without=File,excluded_with=File"` on `Code`; `validate:"required_without=Code,excluded_with=Code"` on `File` | Exactly one of `code` or `file` must be specified — both missing and both present are invalid. Pattern mirrors `cli/internal/project/specs/metadata.go` (`LocalID`/`URN`) and `cli/internal/providers/retl/sqlmodel/model.go` (`SQL`/`File`) |

### Custom Logic (manual rule code)

| # | Validation | Description |
|---|-----------|-------------|
| 6 | `import_name` must equal camelCase of `name` | The `import_name` field must be the camelCase transformation of the `name` field |
| 7 | Only spec-relative `file` paths allowed | User-specified `file` paths must be relative to the YAML spec file's directory. Absolute paths are rejected, and relative paths containing any `..` segment are rejected unconditionally |
| 8 | `file` must be a valid file path | When the user explicitly sets `file`, the resolved path must exist and be a regular file. The path is resolved against `filepath.Dir(ValidationContext.FilePath)` — the directory of the YAML spec file |
| 9 | Code syntax validation | When `code` is provided (or resolved from `file`), validate syntax using the appropriate parser (esbuild for JavaScript, Python parser for Python) |

---

## Semantic Validations to Add for V1

| # | Validation | Description |
|---|-----------|-------------|
| 1 | `import_name` uniqueness | No two `transformation-library` resources may share the same `import_name`. This must be implemented as a cross-resource validation because duplicates can occur across different spec files |

**Note:** The transformation semantic validation for imported library handles assumes this invariant, so `import_name` uniqueness must be enforced here rather than in the transformation spec.

---

## Acceptance Criteria

- [ ] `validate:` tags added to `TransformationLibrarySpec` shared struct (safe since no V0.1 rules exist in new engine)
- [ ] `code`/`file` mutual exclusivity expressed via `required_without`/`excluded_with` tags on both `Code` and `File` fields (no custom logic needed)
- [ ] V1 syntactic rule uses `rules.ValidateStruct()` for tag-based validations (#1-#5)
- [ ] Custom logic implemented for import_name camelCase check (#6), spec-relative `file` path enforcement (#7), `file` existence/type checks (#8), and code syntax validation (#9)
- [ ] `import_name` camelCase validation matches the existing handler logic
- [ ] Relative `file` paths resolved against `filepath.Dir(ValidationContext.FilePath)`; absolute paths and `..` segments rejected; file existence checked via `os.Stat`
- [ ] Code syntax validation integrates with the existing parser infrastructure (`parser.ValidateSyntax()`)
- [ ] Semantic validation #1 is implemented to reject duplicate `transformation-library.import_name` values across all loaded library specs
- [ ] All validations are tested with unit tests
- [ ] Test coverage for changed files exceeds 85%

### `NewPathAwarePatternValidator` prerequisite

`file` path validation (#7–#8) needs the spec file's absolute path, which lives in `ValidationContext.FilePath`. The standard `NewPatternValidator` does not forward `FilePath`, so the library rule must use `NewPathAwarePatternValidator` and relies on the same `engine.go` change already described in [transformation.md](/Users/abhimanyubabbar/workspace/go/src/github.com/rudderlabs/rudder-iac/docs/v1_spec_validation/transformation.md).

---

## PR Process

**Branch**: `feat/dex-257-transformation-library-v1-spec-validation`

**Target Branch**: `main`

**PR Template** (from `.github/pull_request_template.md`):

```markdown
## 🔗 Ticket

[DEX-257](https://linear.app/rudderstack/issue/DEX-257/v1-spec-validation)

---

## Summary

Add V1 spec validation rules for the `transformation-library` resource. These rules target `rudder/v1` specs, implementing 9 syntactic validations including required fields, language constraints, code/file exclusivity via go validator tags, import_name camelCase validation, spec-relative `file` path validation, and code syntax validation, plus 1 semantic validation for `import_name` uniqueness.

---

## Changes

* Add `validate:` tags to `TransformationLibrarySpec` shared struct, including `required_without`/`excluded_with` tags for `code`/`file` mutual exclusivity
* Add V1 syntactic rule using `rules.ValidateStruct()` for tag-based validations + custom logic for import_name camelCase, spec-relative `file` path enforcement, file existence/type checks, and code syntax validation
* Add V1 semantic rule to enforce `import_name` uniqueness across all transformation libraries

---

## Testing

* Unit tests for all syntactic validations
* Unit tests for the semantic `import_name` uniqueness validation
* Table-driven tests covering valid and invalid V1 library specs
* Tests for code/file mutual exclusivity via tag validation
* Tests for import_name camelCase validation
* Tests for relative-only `file` path enforcement (`..` traversal and absolute paths rejected)
* Tests for `file` existence/type checks
* Tests for code syntax validation (valid/invalid JavaScript and Python)
* Tests for duplicate `import_name` values across multiple library specs

---

## Risk / Impact

Low
V1 validation is new functionality; no rules existed previously for transformation libraries.

---

## Checklist

* [ ] Ticket linked
* [ ] Tests added/updated
* [ ] No breaking changes (or documented)
```
