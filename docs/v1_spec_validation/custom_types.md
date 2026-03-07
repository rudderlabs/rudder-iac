# Custom Types - V1 Spec Validation

**Ticket**: [DEX-257 - V1 Spec Validation](https://linear.app/rudderstack/issue/DEX-257/v1-spec-validation)

---

## V0.1 Spec Validation Status

V0.1 validation rules **exist** in the new validation engine. Three rules are registered via `LegacyVersionPatterns("custom-types")` targeting `rudder/0.1` and `rudder/v0.1` only:

| Rule ID | Phase | Location |
|---------|-------|----------|
| `datacatalog/custom-types/spec-syntax-valid` | Syntactic | `cli/internal/providers/datacatalog/rules/customtype/customtype_spec_valid.go` |
| `datacatalog/custom-types/config-valid` | Syntactic | `cli/internal/providers/datacatalog/rules/customtype/customtype_config_valid.go` |
| `datacatalog/custom-types/semantic-valid` | Semantic | `cli/internal/providers/datacatalog/rules/customtype/customtype_semantic_valid.go` |

These rules decode into V0 structs (`CustomTypeSpec` / `CustomType`) and do **not** match V1 specs (`rudder/v1`).

### V0.1 implementation note: primitive type pattern (customtype_spec_valid.go)

The `primitive_type` validation pattern used for the custom type `type` field is **derived from a single source of truth** rather than hardcoded:

- **Location:** `cli/internal/providers/datacatalog/rules/customtype/customtype_spec_valid.go`
- **Pattern:** `customTypeTypeRegexPattern = fmt.Sprintf("^(%s)$", strings.Join(catalogRules.ValidPrimitiveTypes, "|"))`
- **Source of truth:** `catalogRules.ValidPrimitiveTypes` in `cli/internal/providers/datacatalog/rules/constants.go`

This keeps the allowed primitive types (string, number, integer, boolean, array, object, null) in one place; the regex and error message stay in sync with `ValidPrimitiveTypes`. When adding V1 syntactic validation, the same `primitive_type` pattern (and `ValidPrimitiveTypes`) should be reused so V0.1 and V1 do not diverge.

---

## Structural Differences: V0.1 vs V1

| Aspect | V0.1 (`CustomType`) | V1 (`CustomTypeV1`) |
|--------|----------------------|----------------------|
| Property reference field | `$ref` (`json:"$ref"`) | `property` (`json:"property"`) |
| Property ref format | `#/properties/<group>/<id>` (validated by `pattern=legacy_property_ref`) | `#property:<id>` |
| Variant property refs | `PropertyReference` with `$ref` field | `PropertyReferenceV1` with `property` field |
| Config keys | camelCase (`itemTypes`, `minLength`) | snake_case (`item_types`, `min_length`) |
| Validation tags | Has `validate:"required"`, `validate:"pattern=custom_type_name"` etc. | **Add matching tags** (see below) |

### V0.1 Struct (`CustomType` in `localcatalog/model.go`)

```go
type CustomType struct {
    LocalID     string               `mapstructure:"id" json:"id" validate:"required"`
    Name        string               `mapstructure:"name" json:"name" validate:"required,gte=2,lte=65,pattern=custom_type_name"`
    Description string               `mapstructure:"description,omitempty" json:"description,omitempty" validate:"omitempty,gte=3,lte=2000,pattern=letter_start"`
    Type        string               `mapstructure:"type" json:"type" validate:"required,pattern=primitive_type"`
    Config      map[string]any       `mapstructure:"config,omitempty" json:"config,omitempty"`
    Properties  []CustomTypeProperty `mapstructure:"properties,omitempty" json:"properties,omitempty" validate:"omitempty,dive"`
    Variants    Variants             `mapstructure:"variants,omitempty" json:"variants,omitempty" validate:"excluded_unless=Type object,omitempty,max=1,dive"`
}

type CustomTypeProperty struct {
    Ref      string `mapstructure:"$ref" json:"$ref" validate:"required,pattern=legacy_property_ref"`
    Required bool   `mapstructure:"required" json:"required"`
}
```

### V1 Struct — Current (`CustomTypeV1` in `localcatalog/model_v1.go`)

```go
type CustomTypeV1 struct {
    LocalID     string                 `mapstructure:"id" json:"id"`
    Name        string                 `mapstructure:"name" json:"name"`
    Description string                 `mapstructure:"description,omitempty" json:"description,omitempty"`
    Type        string                 `mapstructure:"type" json:"type"`
    Config      map[string]any         `mapstructure:"config,omitempty" json:"config,omitempty"`
    Properties  []CustomTypePropertyV1 `mapstructure:"properties,omitempty" json:"properties,omitempty"`
    Variants    VariantsV1             `mapstructure:"variants,omitempty" json:"variants,omitempty"`
}

type CustomTypePropertyV1 struct {
    Property string `mapstructure:"property" json:"property"`
    Required bool   `mapstructure:"required" json:"required"`
}
```

### V1 Struct — Updated (tags to add)

```go
type CustomTypeV1 struct {
    LocalID     string                 `mapstructure:"id" json:"id" validate:"required"`
    Name        string                 `mapstructure:"name" json:"name" validate:"required,gte=2,lte=65,pattern=custom_type_name"`
    Description string                 `mapstructure:"description,omitempty" json:"description,omitempty" validate:"omitempty,gte=3,lte=2000,pattern=letter_start"`
    Type        string                 `mapstructure:"type" json:"type" validate:"required,pattern=primitive_type"`
    Config      map[string]any         `mapstructure:"config,omitempty" json:"config,omitempty"`
    Properties  []CustomTypePropertyV1 `mapstructure:"properties,omitempty" json:"properties,omitempty" validate:"omitempty,dive"`
    Variants    VariantsV1             `mapstructure:"variants,omitempty" json:"variants,omitempty" validate:"excluded_unless=Type object,omitempty,max=1,dive"`
}

type CustomTypePropertyV1 struct {
    Property string `mapstructure:"property" json:"property" validate:"required,pattern=property_ref"`
    Required bool   `mapstructure:"required" json:"required"`
}
```

Also update `CustomTypeSpecV1`:

```go
type CustomTypeSpecV1 struct {
    Types []CustomTypeV1 `json:"types" validate:"required,dive"`
}
```

Also add tags to `VariantV1` and related structs in `localcatalog/variants.go`:

```go
type VariantV1 struct {
    Type          string              `json:"type" validate:"required,eq=discriminator"`
    Discriminator string              `json:"discriminator" validate:"required,pattern=property_ref"`
    Cases         []VariantCaseV1     `json:"cases" validate:"required,min=1,dive"`
    Default       DefaultPropertiesV1 `json:"default"`
}

type VariantCaseV1 struct {
    DisplayName string                `json:"display_name" validate:"required"`
    Match       []any                 `json:"match" validate:"required,min=1,array_item_types=string bool integer"`
    Description string                `json:"description"`
    Properties  []PropertyReferenceV1 `json:"properties" validate:"required,min=1,dive"`
}

type PropertyReferenceV1 struct {
    Property string `json:"property" validate:"required,pattern=property_ref"`
    Required bool   `json:"required"`
}
```

The `custom_type_name`, `primitive_type`, `property_ref`, and `letter_start` patterns are already registered.

---

## Syntactic Validations to Add for V1

These rules must target `MatchKindVersion("custom-types", "rudder/v1")` and decode into `CustomTypeSpecV1` / `CustomTypeV1`.

### Tag-Based (handled by `rules.ValidateStruct()`)

| # | Validation | Tag | Description |
|---|-----------|-----|-------------|
| 1 | `id` required | `validate:"required"` on `LocalID` | Custom type must have non-empty `id` |
| 2 | `name` required + pattern | `validate:"required,gte=2,lte=65,pattern=custom_type_name"` on `Name` | 2-65 chars, starts with capital letter |
| 3 | `description` constraints | `validate:"omitempty,gte=3,lte=2000,pattern=letter_start"` on `Description` | If present, 3-2000 chars starting with a letter |
| 4 | `type` required + primitive | `validate:"required,pattern=primitive_type"` on `Type` | Must be one of: string, number, integer, boolean, null, array, object |
| 5 | `properties` dive | `validate:"omitempty,dive"` on `Properties` | Recursively validates each `CustomTypePropertyV1` |
| 6 | `property` field required + format | `validate:"required,pattern=property_ref"` on `CustomTypePropertyV1.Property` | Must match `#properties:<id>` |
| 7 | `variants` conditional + dive | `validate:"excluded_unless=Type object,omitempty,max=1,dive"` on `Variants` | Only when type is object, max 1, dives into `VariantV1` |
| 8 | Variant structure | Tags on `VariantV1`, `VariantCaseV1`, `PropertyReferenceV1` | discriminator required, cases min=1, case display_name/match/properties required |

**Note:** The rule "when type is object, config must be nil/empty" is **not** custom logic in the syntactic rule. It is handled by the existing **config-valid** rule (shared `config` package): `ObjectTypeConfig.ConfigAllowed()` returns false, so object + non-empty config is rejected there. No extra manual check is needed for V1.

### Out of Scope (handled separately)

| # | Validation | Description |
|---|-----------|-------------|
| - | Config validation per type | Config fields validated based on type (snake_case keys: `min_length`/`max_length`/`format`/`pattern` for string; `minimum`/`maximum`/`exclusive_minimum`/`exclusive_maximum`/`multiple_of` for number/integer; `item_types`/`min_items`/`max_items`/`unique_items` for array). **Will be handled in a separate PR/task.** |

---

## Semantic Validations to Add for V1

| # | Validation | Description |
|---|-----------|-------------|
| 1 | Property refs resolve | Each `property` field in `properties` array (format `#property:<id>`) must resolve to an existing property in the graph |
| 2 | Variant discriminator ref resolves with type constraint | Discriminator (format `#property:<id>`) must resolve to a property with type string, integer, or boolean |
| 3 | Variant case/default property refs resolve | All property references in variant cases and default must resolve in the graph |
| 4 | `name` uniqueness | No two custom types may share the same name |

### Out of Scope (handled separately)

| # | Validation | Description |
|---|-----------|-------------|
| - | Config `item_types` custom-type refs resolve | Custom-type references in config `item_types` (snake_case) must exist in the graph. **Will be handled in a separate PR/task alongside config validation.** |

Note: `id` uniqueness is handled by the project-level rule `project/duplicate-local-id` which is version-agnostic.

---

## Acceptance Criteria

- [ ] `validate:` tags added to `CustomTypeV1`, `CustomTypePropertyV1`, `CustomTypeSpecV1`, `VariantV1`, `VariantCaseV1`, and `PropertyReferenceV1` structs matching V0.1 style (with V1 ref patterns)
- [ ] V1 syntactic rule uses `rules.ValidateStruct()` for tag-based validations (#1-#8)
- [ ] Object-type config exclusion: rely on config-valid rule (no custom logic in syntactic rule)
- [ ] All 4 semantic validations listed above are implemented as V1 rules (config `item_types` ref resolution excluded, handled separately)
- [ ] Property references use V1 format (`#properties:<id>` via `property` field, not `$ref`)
- [ ] All validations are tested with unit tests
- [ ] Test coverage for changed files exceeds 85%

---

## PR Process

**Branch**: `feat/dex-257-custom-types-v1-spec-validation`

**Target Branch**: `main`

**PR Template** (from `.github/pull_request_template.md`):

```markdown
## 🔗 Ticket

[DEX-257](https://linear.app/rudderstack/issue/DEX-257/v1-spec-validation)

---

## Summary

Add V1 spec validation rules for the `custom-types` resource in the datacatalog provider. These rules target `rudder/v1` specs and decode into `CustomTypeSpecV1`/`CustomTypeV1` structs. Config validation is excluded and will be handled separately.

---

## Changes

* Add `validate:` tags to `CustomTypeV1`, `CustomTypePropertyV1`, `CustomTypeSpecV1`, `VariantV1`, `VariantCaseV1`, `PropertyReferenceV1` structs
* Add V1 syntactic rule using `rules.ValidateStruct()` for tag-based validations (object-config exclusion handled by config-valid rule)
* Add V1 semantic rule for property reference resolution, variant discriminator validation, and name uniqueness
* **Note**: Per-type config validation (snake_case keys) is out of scope and will be handled separately

---

## Testing

* Unit tests for all syntactic validations
* Unit tests for all semantic validations
* Table-driven tests covering valid and invalid V1 custom type specs

---

## Risk / Impact

Low
V1 validation is new functionality; does not affect existing V0.1 validation paths.

---

## Checklist

* [ ] Ticket linked
* [ ] Tests added/updated
* [ ] No breaking changes (or documented)
```
