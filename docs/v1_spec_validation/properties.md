# Properties - V1 Spec Validation

**Ticket**: [DEX-257 - V1 Spec Validation](https://linear.app/rudderstack/issue/DEX-257/v1-spec-validation)

---

## V0.1 Spec Validation Status

V0.1 validation rules **exist** in the new validation engine. Three rules are registered via `LegacyVersionPatterns("properties")` targeting `rudder/0.1` and `rudder/v0.1` only:

| Rule ID | Phase | Location |
|---------|-------|----------|
| `datacatalog/properties/spec-syntax-valid` | Syntactic | `cli/internal/providers/datacatalog/rules/property/property_spec_valid.go` |
| `datacatalog/properties/config-valid` | Syntactic | `cli/internal/providers/datacatalog/rules/property/property_config_valid.go` |
| `datacatalog/properties/semantic-valid` | Semantic | `cli/internal/providers/datacatalog/rules/property/property_semantic_valid.go` |

These rules decode into V0 structs (`PropertySpec` / `Property`) and do **not** match V1 specs (`rudder/v1`).

---

## Structural Differences: V0.1 vs V1

| Aspect | V0.1 (`Property`) | V1 (`PropertyV1`) |
|--------|--------------------|--------------------|
| Config field name | `propConfig` (camelCase JSON key) | `config` (snake_case JSON key) |
| Config keys | camelCase (`itemTypes`, `minLength`, `maxLength`, `exclusiveMinimum`) | snake_case (`item_types`, `min_length`, `max_length`, `exclusive_minimum`) |
| Multiple types | Comma-separated in `type` field (e.g. `"string,number"`) | Separate `types` array field (e.g. `["string", "number"]`) |
| Item types | Inside config as `itemTypes` | Top-level `item_type` (single) and `item_types` (array) fields |
| Validation tags on struct | Has `validate:"required"`, `validate:"gte=1,lte=65"` etc. | **Add matching tags** (see below) |
| Ref format for custom types | `#/custom-types/<group>/<id>` | `#custom-type:<id>` (use `pattern=custom_type_ref`) |

### V0.1 Struct (`Property` in `localcatalog/model.go`)

```go
type Property struct {
    LocalID     string                 `mapstructure:"id" json:"id" validate:"required"`
    Name        string                 `mapstructure:"name" json:"name" validate:"required,gte=1,lte=65"`
    Description string                 `mapstructure:"description,omitempty" json:"description" validate:"omitempty,gte=3,lte=2000"`
    Type        string                 `mapstructure:"type,omitempty" json:"type"`
    Config      map[string]interface{} `mapstructure:"propConfig,omitempty" json:"propConfig,omitempty"`
}
```

### V1 Struct — Current (`PropertyV1` in `localcatalog/model_v1.go`)

```go
type PropertyV1 struct {
    LocalID     string                 `mapstructure:"id" json:"id"`
    Name        string                 `mapstructure:"name" json:"name"`
    Description string                 `mapstructure:"description,omitempty" json:"description,omitempty"`
    Type        string                 `mapstructure:"type,omitempty" json:"type,omitempty"`
    Types       []string               `mapstructure:"types,omitempty" json:"types,omitempty"`
    ItemType    string                 `mapstructure:"item_type,omitempty" json:"item_type,omitempty"`
    ItemTypes   []string               `mapstructure:"item_types,omitempty" json:"item_types,omitempty"`
    Config      map[string]interface{} `mapstructure:"config,omitempty" json:"config,omitempty"`
}
```

### V1 Struct — Updated (tags to add)

```go
type PropertyV1 struct {
    LocalID     string                 `mapstructure:"id" json:"id" validate:"required"`
    Name        string                 `mapstructure:"name" json:"name" validate:"required,gte=1,lte=65"`
    Description string                 `mapstructure:"description,omitempty" json:"description,omitempty" validate:"omitempty,gte=3,lte=2000"`
    Type        string                 `mapstructure:"type,omitempty" json:"type,omitempty" validate:"excluded_with=Types"`
    Types       []string               `mapstructure:"types,omitempty" json:"types,omitempty" validate:"excluded_with=Type,dive,oneof=string number integer boolean null array object"`
    ItemType    string                 `mapstructure:"item_type,omitempty" json:"item_type,omitempty" validate:"excluded_with=ItemTypes"`
    ItemTypes   []string               `mapstructure:"item_types,omitempty" json:"item_types,omitempty" validate:"excluded_with=ItemType,dive,oneof=string number integer boolean null array object"`
    Config      map[string]interface{} `mapstructure:"config,omitempty" json:"config,omitempty"`
}
```

**Tag notes:**
- `dive,oneof=...` on `Types` and `ItemTypes` validates each array element against the allowed primitive types. This means custom-type refs (e.g. `#custom-type:<id>`) are **not allowed** in `types` or `item_types` arrays — use the singular `type` or `item_type` field for custom-type references instead.
- `excluded_with` on `Type`/`Types` and `ItemType`/`ItemTypes` enforces mutual exclusivity at the validator level, following the same pattern used in `retl/sqlmodel/model.go` (`SQL`/`File`) and `project/specs/metadata.go` (`LocalID`/`URN`).

Also update `PropertySpecV1` to enable recursive validation via `dive`:

```go
type PropertySpecV1 struct {
    Properties []PropertyV1 `mapstructure:"properties" json:"properties" validate:"dive"`
}
```

---

## Syntactic Validations to Add for V1

These rules must target `MatchKindVersion("properties", "rudder/v1")` and decode into `PropertySpecV1` / `PropertyV1`.

### Tag-Based (handled by `rules.ValidateStruct()`)

| # | Validation | Tag | Description |
|---|-----------|-----|-------------|
| 1 | `id` required | `validate:"required"` on `LocalID` | Property must have a non-empty `id` |
| 2 | `name` required + length | `validate:"required,gte=1,lte=65"` on `Name` | Property must have a non-empty `name`, 1-65 chars |
| 3 | `description` constraints | `validate:"omitempty,gte=3,lte=2000"` on `Description` | If present, 3-2000 chars |
| 4 | `types` array values in ValidTypes | `validate:"dive,oneof=string number integer boolean null array object"` on `Types` | Each value in `types` must be a valid primitive type |
| 5 | `item_types` array values in ValidTypes | `validate:"dive,oneof=string number integer boolean null array object"` on `ItemTypes` | Each value in `item_types` must be a valid primitive type |
| 6 | `item_type` and `item_types` mutually exclusive | `validate:"excluded_with=ItemTypes"` on `ItemType` / `validate:"excluded_with=ItemType"` on `ItemTypes` | Only one of `item_type` or `item_types` can be specified |
| 7 | `type` and `types` mutually exclusive | `validate:"excluded_with=Types"` on `Type` / `validate:"excluded_with=Type"` on `Types` | Only one of `type` (single) or `types` (array) can be specified |

### Custom Logic (manual rule code)

| # | Validation | Description |
|---|-----------|-------------|
| 8 | No leading/trailing whitespace in `name` | Property name must not have leading or trailing whitespace |
| 9 | No duplicate values in `types` array | The `types` array must not contain duplicate type values |
| 10 | `type` must be in ValidTypes or custom-type ref | Single `type` must be a valid primitive type or a custom-type reference (`#custom-type:<id>`). This also catches comma-separated values (e.g. `"string,integer"`) since they won't match any valid type or ref pattern. |
| 11 | `item_type` must be in ValidTypes or custom-type ref | `item_type` must be a valid primitive type or custom-type reference |

### Out of Scope (handled separately)

| # | Validation | Description |
|---|-----------|-------------|
| - | Config validation per type | Validate `config` fields based on the property type (snake_case keys: `min_length`, `max_length`, `pattern`, `format` for string; `minimum`, `maximum`, `exclusive_minimum`, `exclusive_maximum`, `multiple_of` for number/integer; `item_types`, `min_items`, `max_items`, `unique_items` for array; etc.). **Will be handled in a separate PR/task.** |
| - | Custom-type in `type` disallows `config` | When `type` references a custom type (`#custom-type:<id>`), `config` must be nil. **Will be handled as part of config validation in a separate PR/task.** |

---

## Semantic Validations to Add for V1

| # | Validation | Description |
|---|-----------|-------------|
| 1 | Custom-type refs in `type` resolve | If `type` is `#custom-type:<id>`, the referenced custom type must exist in the resource graph |
| 2 | Custom-type refs in `item_type` resolve | If `item_type` is a custom-type ref, it must exist in the graph |
| 3 | `(name, type, itemTypes)` uniqueness | No two properties may share the same combination of (name, type, itemTypes) |

**Note:** Custom-type refs in `types` and `item_types` arrays are now prevented at the syntactic level by the `dive,oneof=...` validator tag — only primitive types are allowed. Semantic checks #3 and #4 from the previous revision are no longer needed.

### Out of Scope (handled separately)

| # | Validation | Description |
|---|-----------|-------------|
| - | Config `item_types` custom-type refs resolve | Custom-type references within config `item_types` must resolve in the graph. **Will be handled in a separate PR/task alongside config validation.** |

Note: `id` uniqueness is handled by the project-level rule `project/duplicate-local-id` which is version-agnostic.

---

## Acceptance Criteria

- [ ] `validate:` tags added to `PropertyV1` and `PropertySpecV1` structs matching V0.1 style (including `dive,oneof` on `Types`/`ItemTypes`, `excluded_with` on `Type`/`Types` and `ItemType`/`ItemTypes`)
- [ ] V1 syntactic rule uses `rules.ValidateStruct()` for tag-based validations (#1-#7)
- [ ] Custom logic implemented for validations #8-#11 (whitespace, duplicates, type/item_type value checks) -- config validation and custom-type config constraint excluded, handled separately
- [ ] All 3 semantic validations listed above are implemented as V1 rules (config `item_types` ref resolution excluded, handled separately)
- [ ] All validations are tested with unit tests
- [ ] Test coverage for changed files exceeds 85%

---

## PR Process

**Branch**: `feat/dex-257-properties-v1-spec-validation`

**Target Branch**: `main`

**PR Template** (from `.github/pull_request_template.md`):

```markdown
## 🔗 Ticket

[DEX-257](https://linear.app/rudderstack/issue/DEX-257/v1-spec-validation)

---

## Summary

Add V1 spec validation rules for the `properties` resource in the datacatalog provider. These rules target `rudder/v1` specs and decode into `PropertySpecV1`/`PropertyV1` structs, implementing syntactic and semantic validations. Config validation is excluded and will be handled separately.

---

## Changes

* Add `validate:` tags to `PropertyV1` and `PropertySpecV1` structs
* Add V1 syntactic rule using `rules.ValidateStruct()` for tag-based validations + custom logic for type/types exclusivity
* Add V1 semantic rule for property reference resolution and uniqueness
* **Note**: Config validation (per-type config fields, snake_case keys) is out of scope and will be handled separately

---

## Testing

* Unit tests for all syntactic validations
* Unit tests for all semantic validations
* Table-driven tests covering valid and invalid V1 property specs

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
