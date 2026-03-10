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
| 7 | Test name pattern | Test names must match `^[A-Za-z0-9 _/\-]+$` (alphanumeric, spaces, underscores, slashes, hyphens) |
| 8 | `..` traversal blocked | Relative `input`/`output` paths containing any `..` segment (e.g. `../sibling`) are rejected unconditionally to prevent directory traversal outside the project folder |
| 9 | Input/Output must be a valid directory | When `input` or `output` is set, the resolved path must exist and be a directory. Absolute paths are checked as-is; relative paths are resolved against `filepath.Dir(ValidationContext.FilePath)` — the directory of the YAML spec file |
| 10 | JSON files in Input/Output must be valid JSON | When `.json` files exist at the resolved `input` or `output` directory, each file must contain valid JSON |

---

## Semantic Validations to Add for V1

No semantic validations needed. Transformations have no cross-resource references.

---

## Acceptance Criteria

- [ ] `validate:` tags added to `TransformationSpec` and `TransformationTest` shared structs (safe since no V0.1 rules exist in new engine)
- [ ] `code`/`file` mutual exclusivity expressed via `required_without`/`excluded_with` tags on both `Code` and `File` fields (no custom logic needed)
- [ ] V1 syntactic rule uses `rules.ValidateStruct()` for tag-based validations (#1-#6)
- [ ] Custom logic implemented for test name pattern (#7), `..` traversal blocking (#8), Input/Output directory existence (#9), and JSON file validity (#10)
- [ ] Relative `input`/`output` paths resolved against `filepath.Dir(ValidationContext.FilePath)`; `..` segments rejected; directory existence checked via `os.Stat`
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
        ├── transformation_spec_valid.go        ← syntactic rule (implement here)
        └── transformation_spec_valid_test.go   ← unit tests
```

### `NewPathAwarePatternValidator` prerequisite

The directory checks (#8–#10) need the spec file's absolute path, which lives in `ValidationContext.FilePath`. The standard `NewPatternValidator` only passes `(kind, version, metadata, spec)` to the validation function — it does **not** forward `FilePath`.

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

> `ValidationContext.FilePath` is populated for every rule invocation because `engine.go` (`runValidationRules`) now sets `FilePath: path` in the context.

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

    for i, test := range spec.Tests {
        // #7: test name pattern
        if test.Name != "" && !testNamePattern.MatchString(test.Name) {
            results = append(results, rules.ValidationResult{
                Reference: fmt.Sprintf("/tests/%d/name", i),
                Message:   `test name must match ^[A-Za-z0-9 _/\-]+$`,
            })
        }

        // #8–#10: directory checks for input and output
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

            // #8: block ".." traversal in relative paths
            resolved, err := resolveTestDir(filePath, field.path)
            if err != nil {
                results = append(results, rules.ValidationResult{
                    Reference: fmt.Sprintf("/tests/%d/%s", i, field.name),
                    Message:   err.Error(),
                })
                continue
            }

            // #9: directory must exist
            dirResults := validateTestDirectory(i, field.name, resolved)
            results = append(results, dirResults...)
            if len(dirResults) > 0 {
                // skip JSON check when directory is absent/invalid
                continue
            }

            // #10: JSON files must be valid JSON
            results = append(results, validateTestJSONFiles(i, field.name, resolved)...)
        }
    }

    return results
}

// resolveTestDir resolves the test directory path relative to the spec file.
// Relative paths are resolved against filepath.Dir(specFilePath).
// Any path whose cleaned form contains a ".." segment is rejected to prevent
// traversal outside the project directory.
func resolveTestDir(specFilePath, dir string) (string, error) {
    if !filepath.IsAbs(dir) {
        // Block ".." traversal in relative paths
        cleaned := filepath.Clean(dir)
        for _, segment := range strings.Split(cleaned, string(filepath.Separator)) {
            if segment == ".." {
                return "", fmt.Errorf("path must not contain '..' segments: %q", dir)
            }
        }
        dir = filepath.Join(filepath.Dir(specFilePath), dir)
    }
    return dir, nil
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
// Transformations have no cross-resource references, so this remains empty.
func (p *Provider) SemanticRules() []rules.Rule {
    return []rules.Rule{}
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

Add V1 spec validation rules for the `transformation` resource. These rules target `rudder/v1` specs, implementing 10 syntactic validations including required fields, language constraints, code/file exclusivity via go validator tags, test name patterns, Input/Output directory existence with `..` traversal blocking, and JSON file validity.

---

## Changes

* Add `validate:` tags to `TransformationSpec` and `TransformationTest` shared structs, including `required_without`/`excluded_with` tags for `code`/`file` mutual exclusivity
* Add `NewPathAwarePatternValidator` to `cli/internal/provider/rules/pattern_validator.go` to thread `FilePath` into rule validation functions
* Add V1 syntactic rule using `rules.ValidateStruct()` for tag-based validations + custom logic for test name pattern, `..` traversal blocking, Input/Output directory existence, and JSON file validity

---

## Testing

* Unit tests for all syntactic validations
* Table-driven tests covering valid and invalid V1 transformation specs
* Tests for code/file mutual exclusivity via tag validation
* Tests for `..` traversal rejection in relative paths
* Tests for Input/Output directory existence (absolute path, valid relative path, non-existent path, file instead of directory)
* Tests for JSON file validity (valid JSON, malformed JSON, empty file)

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
