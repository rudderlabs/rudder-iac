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
    Code        string               `json:"code,omitempty" mapstructure:"code" validate:"required_without=File,excluded_with=File"`
    File        string               `json:"file,omitempty" mapstructure:"file" validate:"required_without=Code,excluded_with=Code"`
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

> Validation operates on the raw YAML-decoded `TransformationSpec` before `extractTestsFromSpec()` applies `DefaultInputPath` / `DefaultOutputPath`. As a result, `tests[].input` and `tests[].output` are validated only when the user explicitly sets them in YAML; omitted fields are not defaulted as part of V1 syntactic validation.

### Tag-Based (handled by `rules.ValidateStruct()`)

| # | Validation | Tag | Description |
|---|-----------|-----|-------------|
| 1 | `id` required | `validate:"required"` on `ID` | Transformation must have a non-empty `id` |
| 2 | `name` required | `validate:"required"` on `Name` | Transformation must have a non-empty `name` |
| 3 | `language` required + oneof | `validate:"required,oneof=javascript python"` on `Language` | Language must be one of the allowed values |
| 4 | Tests dive | `validate:"omitempty,dive"` on `Tests` | Recursively validates each `TransformationTest` |
| 5 | Test `name` required | `validate:"required"` on `TransformationTest.Name` | Each test must have a non-empty `name` |
| 6 | `code`/`file` mutual exclusivity | `validate:"required_without=File,excluded_with=File"` on `Code`; `validate:"required_without=Code,excluded_with=Code"` on `File` | Exactly one of `code` or `file` must be specified — both missing and both present are invalid. Pattern mirrors `cli/internal/project/specs/metadata.go` (`LocalID`/`URN`) and `cli/internal/providers/retl/sqlmodel/model.go` (`SQL`/`File`) |

### Custom Logic (manual rule code)

| # | Validation | Description |
|---|-----------|-------------|
| 7 | Test name must not be whitespace-only | Test names must satisfy `strings.TrimSpace(name) != ""`; values like `"   "` are rejected even though they are non-empty |
| 8 | Test name pattern | Test names must match `^[A-Za-z0-9 _/\-]+$` (alphanumeric, spaces, underscores, slashes, hyphens) |
| 9 | Only spec-relative paths allowed | User-specified `file`, `input`, and `output` paths must be relative to the YAML spec file's directory. Absolute paths are rejected, and relative paths containing any `..` segment (e.g. `../sibling`) are rejected unconditionally |
| 10 | `file` must be a valid file path | When the user explicitly sets `file`, the resolved path must exist and be a regular file. The path is resolved against `filepath.Dir(ValidationContext.FilePath)` — the directory of the YAML spec file |
| 11 | Input/Output must be valid directories | When the user explicitly sets `input` or `output`, the resolved path must exist and be a directory. Relative paths are resolved against `filepath.Dir(ValidationContext.FilePath)` — the directory of the YAML spec file |
| 12 | JSON files in Input/Output must be valid JSON | When the user explicitly sets `input` or `output` and `.json` files exist at the resolved directory, each file must contain valid JSON. Any valid JSON document shape is accepted, including arrays, objects only |

---

## Semantic Validations to Add for V1

| # | Validation | Description |
|---|-----------|-------------|
| 1 | Imported transformation-library handles resolve | For each library handle successfully extracted from the transformation's resolved code, a `transformation-library` resource with matching `import_name` must exist in the resource graph. The semantic rule should read code from the loaded `*model.TransformationResource` in `ctx.Graph`, not from the raw spec, so both inline `code` and file-backed `file` specs are handled uniformly |

**Prerequisite:** `Provider.ResourceGraph()` must stop returning an error when an imported library handle is missing. Instead, it should skip adding that dependency edge and let the semantic rule emit the validation diagnostic.

**Dependency:** `transformation-library.import_name` uniqueness is validated in [transformation_library.md](/Users/abhimanyubabbar/workspace/go/src/github.com/rudderlabs/rudder-iac/docs/v1_spec_validation/transformation_library.md). This semantic rule assumes each imported handle maps to at most one library.

**Boundary:** This semantic rule validates only imports that were successfully extracted by the parser. Import extraction failures unrelated to missing libraries remain mid-pipeline failures and are not converted into semantic diagnostics by this spec.

---

## Acceptance Criteria

- [ ] `validate:` tags added to `TransformationSpec` and `TransformationTest` shared structs (safe since no V0.1 rules exist in new engine)
- [ ] `code`/`file` mutual exclusivity expressed via `required_without`/`excluded_with` tags on both `Code` and `File` fields (no custom logic needed)
- [ ] V1 syntactic rule uses `rules.ValidateStruct()` for tag-based validations (#1-#6)
- [ ] Custom logic implemented for whitespace-only test name rejection (#7), test name pattern (#8), spec-relative path enforcement for `file`/`input`/`output` (#9), `file` existence/type checks (#10), Input/Output directory existence checks (#11), and JSON file validity (#12)
- [ ] Test path validation runs on the raw YAML-decoded spec before default enrichment; omitted `input`/`output` fields are not validated via `DefaultInputPath` / `DefaultOutputPath`
- [ ] `cli/internal/validation/engine.go` updates `runValidationRules()` to set `ValidationContext.FilePath = path` for every per-spec rule invocation
- [ ] Relative `file`/`input`/`output` paths resolved against `filepath.Dir(ValidationContext.FilePath)`; absolute paths and `..` segments rejected; file existence checked via `os.Stat`; directory existence checked via `os.Stat`
- [ ] Semantic validation #1 is implemented as a V1 rule using the loaded transformation resource from `ctx.Graph` to inspect resolved code and verify imported library handles against graph libraries' `ImportName`
- [ ] `Provider.ResourceGraph()` no longer returns an error for unresolved library imports; it skips adding the missing dependency edge so semantic validation can report the issue
- [ ] Import extraction failures unrelated to missing libraries remain existing mid-pipeline failures; only successfully extracted handles participate in semantic import-resolution validation
- [ ] All validations are tested with unit tests
- [ ] Test coverage for changed files exceeds 85%

---

## Implementation Guide

### File Location

Follow the datacatalog pattern: rule files live directly under `rules/<resource-type>/` inside the provider root, **not** under `handlers/rules/`. Reference: `cli/internal/providers/datacatalog/rules/event/event_spec_valid.go`.

The existing stub at `cli/internal/providers/transformations/handlers/rules/transformation/transformation_spec_valid.go` must be **deleted** and replaced with a proper implementation at the path below.

```
cli/internal/providers/transformations/
└── rules/
    └── transformation/
        ├── transformation_spec_valid.go                  ← syntactic rule
        ├── transformation_spec_valid_test.go             ← syntactic unit tests
        ├── transformation_semantic_valid.go      ← semantic rule
        └── transformation_semantic_valid_test.go ← semantic unit tests
```

### `NewPathAwarePatternValidator` prerequisite

The path checks (#9–#12) need the spec file's absolute path, which lives in `ValidationContext.FilePath`. The standard `NewPatternValidator` only passes `(kind, version, metadata, spec)` to the validation function — it does **not** forward `FilePath`.

This requires an explicit engine change: `cli/internal/validation/engine.go` must update `runValidationRules()` so each per-spec rule receives `ValidationContext{FilePath: path, ...}`. Without that change, relative `file` / `input` / `output` resolution in the transformation rule cannot work correctly.

Before implementing this rule, add `NewPathAwarePatternValidator` to `cli/internal/provider/rules/pattern_validator.go`:

```go
// NewPathAwarePatternValidator is identical to NewPatternValidator but also forwards
// ctx.FilePath to fn, enabling path-relative directory resolution inside rule logic.
func NewPathAwarePatternValidator[T any](
    patterns []rules.MatchPattern,
    fn func(kind string, version string, filePath string, metadata map[string]any, spec T) []rules.ValidationResult,
) PatternValidator {
    return PatternValidator{
        patterns: patterns,
        validate: func(ctx *rules.ValidationContext) []rules.ValidationResult {
            spec, err := unmarshalSpec[T](ctx.Spec)
            if err != nil {
                return err
            }
            results := fn(ctx.Kind, ctx.Version, ctx.FilePath, ctx.Metadata, spec)
            prefixReferences(results)
            return results
        },
    }
}
```

> This spec includes the required engine change: in `engine.go`, `runValidationRules()` must pass `FilePath: path` when constructing `ValidationContext` for rule execution.

### Rule File Skeleton (`transformation_spec_valid.go`)

```go
package transformation

import (
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "regexp"
    "strings"

    prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
    "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
    "github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
    "github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/handlers/transformation"
    "github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

var testNamePattern = regexp.MustCompile(`^[A-Za-z0-9 _/\-]+$`)

func validateTransformationSpec(
    _ string,
    _ string,
    filePath string,
    _ map[string]any,
    spec specs.TransformationSpec,
) []rules.ValidationResult {
    validationErrors, err := rules.ValidateStruct(spec, "")
    if err != nil {
        return []rules.ValidationResult{{Reference: "/", Message: err.Error()}}
    }

    results := funcs.ParseValidationErrors(validationErrors, nil)

    // #9–#10: path checks for file
    if spec.File != "" {
        resolved, err := resolveSpecRelativePath(filePath, spec.File)
        if err != nil {
            results = append(results, rules.ValidationResult{
                Reference: "/file",
                Message:   err.Error(),
            })
        } else {
            results = append(results, validateSpecFile(resolved)...)
        }
    }

    for i, test := range spec.Tests {
        // #7: reject whitespace-only test names
        if test.Name != "" && strings.TrimSpace(test.Name) == "" {
            results = append(results, rules.ValidationResult{
                Reference: fmt.Sprintf("/tests/%d/name", i),
                Message:   "test name must not be blank or whitespace-only",
            })
            continue
        }

        // #8: test name pattern
        if test.Name != "" && !testNamePattern.MatchString(test.Name) {
            results = append(results, rules.ValidationResult{
                Reference: fmt.Sprintf("/tests/%d/name", i),
                Message:   `test name must match ^[A-Za-z0-9 _/\-]+$`,
            })
        }

        // #9–#12: directory checks for input and output
        for _, field := range []struct {
            name string
            path string
        }{
            {"input", test.Input},
            {"output", test.Output},
        } {
            if field.path == "" {
                continue
            }

            // #9: allow only spec-relative paths; reject absolute paths and ".."
            resolved, err := resolveSpecRelativePath(filePath, field.path)
            if err != nil {
                results = append(results, rules.ValidationResult{
                    Reference: fmt.Sprintf("/tests/%d/%s", i, field.name),
                    Message:   err.Error(),
                })
                continue
            }

            // #11: directory must exist
            dirResults := validateTestDirectory(i, field.name, resolved)
            results = append(results, dirResults...)
            if len(dirResults) > 0 {
                // skip JSON check when directory is absent/invalid
                continue
            }

            // #12: JSON files must be valid JSON
            results = append(results, validateTestJSONFiles(i, field.name, resolved)...)
        }
    }

    return results
}

// resolveSpecRelativePath resolves a spec-relative path against the spec file.
// Only relative paths are allowed; absolute paths are rejected.
// Relative paths are resolved against filepath.Dir(specFilePath).
// Any path whose cleaned form contains a ".." segment is rejected to prevent
// traversal outside the spec file's directory tree.
func resolveSpecRelativePath(specFilePath, path string) (string, error) {
    if filepath.IsAbs(path) {
        return "", fmt.Errorf("path must be relative to the spec file directory: %q", path)
    }

    // Block ".." traversal in relative paths
    cleaned := filepath.Clean(path)
    for _, segment := range strings.Split(cleaned, string(filepath.Separator)) {
        if segment == ".." {
            return "", fmt.Errorf("path must not contain '..' segments: %q", path)
        }
    }
    return filepath.Join(filepath.Dir(specFilePath), path), nil
}

// validateSpecFile checks that the resolved path exists and is a regular file.
func validateSpecFile(resolved string) []rules.ValidationResult {
    fi, err := os.Stat(resolved)
    if err != nil {
        return []rules.ValidationResult{{
            Reference: "/file",
            Message:   fmt.Sprintf("path %q does not exist or is not accessible", resolved),
        }}
    }
    if fi.IsDir() {
        return []rules.ValidationResult{{
            Reference: "/file",
            Message:   fmt.Sprintf("path %q must be a file", resolved),
        }}
    }
    return nil
}

// validateTestDirectory checks that the resolved path exists and is a directory.
func validateTestDirectory(testIdx int, field, resolved string) []rules.ValidationResult {
    fi, err := os.Stat(resolved)
    if err != nil {
        return []rules.ValidationResult{{
            Reference: fmt.Sprintf("/tests/%d/%s", testIdx, field),
            Message:   fmt.Sprintf("path %q does not exist or is not accessible", resolved),
        }}
    }
    if !fi.IsDir() {
        return []rules.ValidationResult{{
            Reference: fmt.Sprintf("/tests/%d/%s", testIdx, field),
            Message:   fmt.Sprintf("path %q must be a directory", resolved),
        }}
    }
    return nil
}

// validateTestJSONFiles checks that every .json file in dir contains valid JSON.
func validateTestJSONFiles(testIdx int, field, dir string) []rules.ValidationResult {
    entries, err := os.ReadDir(dir)
    if err != nil {
        return nil
    }

    var results []rules.ValidationResult
    for _, entry := range entries {
        if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
            continue
        }

        data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
        if err != nil {
            continue
        }

        if !json.Valid(data) {
            results = append(results, rules.ValidationResult{
                Reference: fmt.Sprintf("/tests/%d/%s/%s", testIdx, field, entry.Name()),
                Message:   "file must contain valid JSON",
            })
        }
    }
    return results
}

func NewTransformationSpecSyntaxValidRule() rules.Rule {
    return prules.NewTypedRule(
        "transformation/spec-syntax-valid",
        rules.Error,
        "transformation spec syntax must be valid",
        rules.Examples{},
        prules.NewPathAwarePatternValidator(
            prules.V1VersionPatterns(transformation.KindTransformation),
            validateTransformationSpec,
        ),
    )
}
```

### Semantic Rule Design (`transformation_semantic_valid.go`)

The semantic rule should:

1. Look up the loaded transformation resource from the graph using `resources.URN(spec.ID, transformation.HandlerMetadata.ResourceType)`.
2. Read resolved code from `resource.RawData().(*model.TransformationResource).Code` so inline `code` and file-backed `file` are treated the same.
3. Parse imported library handles using the existing transformation parser.
4. Build the set of available library handles by iterating graph resources of type `transformation-library` and reading each `*model.LibraryResource`.ImportName.
5. Emit a validation diagnostic for each missing imported handle.

Diagnostic references should point to `/file` when the spec uses `file`, otherwise `/code`.

`Provider.ResourceGraph()` must be updated so unresolved imports do not abort graph construction. When a handle is missing, skip `graph.AddDependency(...)` for that handle and let the semantic rule report the issue.

Import extraction failures unrelated to missing libraries should continue to fail in the existing mid-pipeline flow; this semantic rule only validates handles returned successfully by `ExtractImports()`.

### Wiring into `provider.go`

**Step 1 — Update the import** in `cli/internal/providers/transformations/provider.go`.

Replace the old `handlers/rules/transformation` import:

```go
// Remove this:
trules "github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/handlers/rules/transformation"

// Add this:
trules "github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/rules/transformation"
```

**Step 2 — `SyntacticRules()` and `SemanticRules()`** remain unchanged in shape; they already exist in `provider.go`. The wired result after the migration:

```go
// SyntacticRules aggregates syntactic rules from all handlers implementing RuleHandler.
func (p *Provider) SyntacticRules() []rules.Rule {
    return []rules.Rule{
        trules.NewTransformationSpecSyntaxValidRule(),
    }
}

// SemanticRules aggregates semantic rules from all handlers implementing RuleHandler.
func (p *Provider) SemanticRules() []rules.Rule {
    return []rules.Rule{
        trules.NewTransformationImportsSemanticValidRule(),
    }
}
```

> The datacatalog provider also carries a blank import (`_ ".../datacatalog/rules"`) to trigger the `init()` in `constants.go` for custom validator pattern registration. Transformations have no cross-reference patterns to register, so **no `constants.go` or blank import is needed** for this provider.

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

Add V1 spec validation rules for the `transformation` resource. These rules target `rudder/v1` specs, implementing 12 syntactic validations covering required fields, language constraints, code/file exclusivity via go validator tags, whitespace-aware test name checks, spec-relative path constraints for `file`/`input`/`output`, file/directory existence, and JSON file validity, plus 1 semantic validation for transformation-library import resolution.

---

## Changes

* Add `validate:` tags to `TransformationSpec` and `TransformationTest` shared structs, including `required_without`/`excluded_with` tags for `code`/`file` mutual exclusivity
* Add `NewPathAwarePatternValidator` to `cli/internal/provider/rules/pattern_validator.go` to thread `FilePath` into rule validation functions
* Add V1 syntactic rule using `rules.ValidateStruct()` for tag-based validations + custom logic for whitespace-aware test names, spec-relative path enforcement for `file`/`input`/`output`, file/directory existence, and JSON file validity
* Add V1 semantic rule for imported transformation-library handle resolution using the loaded transformation resource from the graph
* Update `Provider.ResourceGraph()` to skip unresolved transformation-library imports so semantic validation can report them
* Keep non-missing-library import extraction failures in the existing mid-pipeline failure path; semantic validation only covers successfully extracted handles

---

## Testing

* Unit tests for all syntactic validations
* Unit tests for the semantic import-resolution validation
* Table-driven tests covering valid and invalid V1 transformation specs
* Tests for code/file mutual exclusivity via tag validation
* Tests for relative-only path enforcement on `file` / `input` / `output` (`..` traversal and absolute paths rejected)
* Tests for `file` existence/type checks and Input/Output directory existence checks
* Tests for JSON file validity (valid JSON, malformed JSON, empty file, scalar JSON values)
* Tests confirming import extraction failures unrelated to missing libraries still fail in the existing mid-pipeline path
* Tests for unresolved transformation-library imports producing semantic diagnostics without aborting graph construction

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
