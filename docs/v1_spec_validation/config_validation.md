# V1 Config Validation for Properties and Custom Types

**Status:** Draft  
**Owner:** Data Catalog Validation  
**Date:** 2026-03-08

---

## 1. Overview

### Problem Statement

The shared config validator in `cli/internal/providers/datacatalog/rules/config/` currently validates the V0 config shape only:

- type-specific keys are hard-coded in camelCase (`minLength`, `itemTypes`, `exclusiveMinimum`)
- custom type references are recognized using the legacy reference format only
- the validator has no concept of spec version or config dialect

V1 config validation must support the following changes without duplicating the entire validator stack:

1. V1 config keys use snake_case instead of camelCase
2. V1 array custom type references should recognize the current custom type reference format: `#custom-type:<id>`
3. V1 key handling should be driven by caller-supplied aliasing rather than a dedicated version-specific validator fork

This spec defines a focused change in the shared `rules/config` package only. Wiring V1 property and custom type rules to consume the new behavior is intentionally deferred to follow-up PRs.

### Goals

1. Reuse the existing validator architecture and union/cross-field semantics
2. Add version-aware field normalization and custom-type-ref recognition
3. Keep V0 behavior unchanged
4. Allow V1 callers to supply key aliases for snake_case config input
5. Keep the implementation narrow enough that later rule wiring is straightforward

### Non-Goals

1. Wiring V1 property rules to the new config validator
2. Wiring V1 custom type rules to the new config validator
3. V1 semantic validation for properties or custom types
4. Validation of top-level V1 property fields such as `item_type` and `item_types`
5. Refactoring all validators into a new abstraction from scratch

---

## 2. Current State

### Existing Shared Validator

The current shared engine in `cli/internal/providers/datacatalog/rules/config/validator.go` already provides the correct high-level behavior:

- field-level union semantics across multiple types
- cross-field validation with deduplication
- validator overrides for context-specific object handling

This is the design that should be preserved.

### Current Limitations

The current implementation is V0-oriented in a few specific places:

1. Allowed-key maps are camelCase-oriented:
   - `StringTypeConfig`: `minLength`, `maxLength`
   - `IntegerTypeConfig`: `exclusiveMinimum`, `multipleOf`
   - `ArrayTypeConfig`: `itemTypes`, `minItems`, `uniqueItems`
2. Cross-field lookups are also camelCase-oriented
3. Custom type references in the shared package use the legacy matcher only
4. The validator cannot distinguish between V0 and V1 config input
5. The custom type object override currently accepts `additionalProperties`, which is not the correct V1 spelling

---

## 3. Design Decisions

### D-1: Introduce `ConfigKeyword` as the Shared Logical Keyword Layer

The shared validator flow should use `ConfigKeyword` values as the logical representation of supported config fields.

Examples:

- `KeywordMinLength`
- `KeywordItemTypes`
- `KeywordExclusiveMinimum`
- `KeywordAdditionalProperties`

This avoids leaking raw V0 camelCase field names into the new V1 aliasing API and gives the validator a version-independent keyword vocabulary.

### D-2: Add an Options-Driven Field Mapping Layer at the Boundary

Version support should be implemented by normalizing accepted raw input keys to shared `ConfigKeyword` values before invoking type validators.

Examples:

- V0 `minLength` -> `KeywordMinLength`
- V0 `itemTypes` -> `KeywordItemTypes`
- V1 `min_length` -> `KeywordMinLength`
- V1 `item_types` -> `KeywordItemTypes`
- V1 `multiple_of` -> `KeywordMultipleOf`
- V1 `additional_properties` -> `KeywordAdditionalProperties`

This field mapping should be configured through a `WithOptions` framework rather than a separate dialect interface.

### D-3: No Special Wrong-Spelling Handling in Shared Validation

This change should not introduce a special wrong-spelling path for V1 keys.

Examples:

- `min_length` can be aliased to the shared validator keyword for min length
- `minLength` is not handled specially by the shared package
- if `minLength` is not aliased, it should fall through to the existing unknown-key / not-applicable behavior

### D-4: V1 Custom Type Refs Use `#custom-type:<id>`

For this change, the current reference format is:

```text
#custom-type:<id>
```

Legacy refs such as `#/custom-types/<group>/<id>` do not need a special V1 error. If they appear inside V1 config, it is acceptable for them to fail through the existing invalid-type path.

### D-5: Scope is Limited to `config`

This work covers only config-object validation in the shared package.

It does **not** validate V1 top-level property fields such as:

- `item_type`
- `item_types`
- `type`
- `types`

Those concerns belong to property/custom-type V1 rule wiring and semantic validation, which are explicitly out of scope for this spec.

### D-6: Preserve Backward Compatibility for Existing Callers

Because V1 rule wiring is deferred, the shared package should expose the new behavior without forcing all current V0 call sites to change immediately.

Recommended approach:

- keep the existing `ValidateConfig(...)` entrypoint as a legacy wrapper
- add a new `ValidateConfigWithOptions(...)` entrypoint for future V1 callers
- make the legacy wrapper pass an explicit V0 alias preset instead of relying on implicit field-name fallback

This removes ambiguity about how V0 input continues to resolve and makes the no-regression behavior part of the API contract.

### D-7: Do Not Strengthen Existing Validation Semantics in This Change

This work is about version-aware key handling and custom-type-ref recognition. It should not silently tighten unrelated config validation behavior.

The following existing shared-validator semantics must remain unchanged:

1. `enum` validation is shallow:
   - value must be an array
   - duplicate entries are rejected
   - enum element types are **not** validated against the enclosing config type
2. `pattern` validation is shallow:
   - value must be a string
   - the string is **not** compiled as a regex in this validator layer
3. Unknown or invalid type names remain the responsibility of upstream syntax validation:
   - if no validator can be constructed for a type, the shared config validator should continue to defer rather than inventing a new type-level error

This is important because the current tests explicitly permit mixed-type enum arrays, and no current validator compiles regex patterns.

---

## 4. Proposed Architecture

### 4.1 New Options Framework

Introduce an options-driven validation entrypoint inside `cli/internal/providers/datacatalog/rules/config/`.

```go
type ConfigKeyword string

const (
	KeywordEnum                 ConfigKeyword = "enum"
	KeywordMinimum              ConfigKeyword = "minimum"
	KeywordMaximum              ConfigKeyword = "maximum"
	KeywordPattern              ConfigKeyword = "pattern"
	KeywordFormat               ConfigKeyword = "format"
	KeywordMinLength            ConfigKeyword = "min_length"
	KeywordMaxLength            ConfigKeyword = "max_length"
	KeywordExclusiveMinimum     ConfigKeyword = "exclusive_minimum"
	KeywordExclusiveMaximum     ConfigKeyword = "exclusive_maximum"
	KeywordMultipleOf           ConfigKeyword = "multiple_of"
	KeywordItemTypes            ConfigKeyword = "item_types"
	KeywordMinItems             ConfigKeyword = "min_items"
	KeywordMaxItems             ConfigKeyword = "max_items"
	KeywordUniqueItems          ConfigKeyword = "unique_items"
	KeywordAdditionalProperties ConfigKeyword = "additional_properties"
)

type ValidateConfigOption func(*validateConfigOptions)

func WithFieldAliases(aliases map[string]ConfigKeyword) ValidateConfigOption
func WithCustomTypeRefMatcher(fn func(string) bool) ValidateConfigOption
func WithValidatorOverrides(overrides map[string]TypeConfigValidator) ValidateConfigOption
```

Notes:

- `WithFieldAliases` configures accepted raw-to-keyword mappings such as `min_length -> KeywordMinLength`
- `WithCustomTypeRefMatcher` makes custom type handling version-aware without exposing regex assumptions in the validator flow
- `WithValidatorOverrides` preserves the current custom-type object override behavior in the same options framework
- alias presets should be explicit and complete for each supported input shape rather than relying on hidden default keyword lookup

The options framework carries behavior only. It does not supply custom error messages.

### 4.2 Entry Point Strategy

Retain the current public behavior for V0 callers and add a new options-aware path for future V1 use.

```go
func ValidateConfig(
	types []string,
	config map[string]any,
	reference string,
	validatorOverrides map[string]TypeConfigValidator,
) []rules.ValidationResult {
	return ValidateConfigWithOptions(
		types,
		config,
		reference,
		WithFieldAliases(v0FieldAliases),
		WithValidatorOverrides(validatorOverrides),
		WithCustomTypeRefMatcher(legacyCustomTypeRefMatcher),
	)
}

func ValidateConfigWithOptions(
	types []string,
	config map[string]any,
	reference string,
	opts ...ValidateConfigOption,
) []rules.ValidationResult
```

This keeps current V0 consumers stable and gives future V1 rule handlers an explicit API to opt into V1 behavior.

Recommended presets:

```go
var v0FieldAliases = map[string]ConfigKeyword{
	"enum":                 KeywordEnum,
	"minimum":              KeywordMinimum,
	"maximum":              KeywordMaximum,
	"pattern":              KeywordPattern,
	"format":               KeywordFormat,
	"minLength":            KeywordMinLength,
	"maxLength":            KeywordMaxLength,
	"exclusiveMinimum":     KeywordExclusiveMinimum,
	"exclusiveMaximum":     KeywordExclusiveMaximum,
	"multipleOf":           KeywordMultipleOf,
	"itemTypes":            KeywordItemTypes,
	"minItems":             KeywordMinItems,
	"maxItems":             KeywordMaxItems,
	"uniqueItems":          KeywordUniqueItems,
	"additionalProperties": KeywordAdditionalProperties,
}

var v1FieldAliases = map[string]ConfigKeyword{
	"enum":                  KeywordEnum,
	"minimum":               KeywordMinimum,
	"maximum":               KeywordMaximum,
	"pattern":               KeywordPattern,
	"format":                KeywordFormat,
	"min_length":            KeywordMinLength,
	"max_length":            KeywordMaxLength,
	"exclusive_minimum":     KeywordExclusiveMinimum,
	"exclusive_maximum":     KeywordExclusiveMaximum,
	"multiple_of":           KeywordMultipleOf,
	"item_types":            KeywordItemTypes,
	"min_items":             KeywordMinItems,
	"max_items":             KeywordMaxItems,
	"unique_items":          KeywordUniqueItems,
	"additional_properties": KeywordAdditionalProperties,
}
```

The shared package may expose these as reusable presets, or callers may compose them locally. The important part is that V0 compatibility remains explicit and testable.

### 4.3 Canonicalization Flow

Before invoking the existing validators:

1. Iterate over raw config keys
2. Resolve the raw key through the effective alias map supplied in options
3. If the raw key is not present in the effective alias map, leave it unresolved
4. Build a normalized config map keyed by `ConfigKeyword`
5. Run field-level validation using `ConfigKeyword`, but preserve raw references in error output
6. Run cross-field validation against the normalized config map

Suggested shape:

```go
type normalizedField struct {
	Raw       string
	Keyword   ConfigKeyword
	Value     any
	Resolved  bool
}

func normalizeConfig(
	config map[string]any,
	opts validateConfigOptions,
) (fields []normalizedField, normalized map[ConfigKeyword]any)
```

Important:

- references in validation errors should continue to point to the user-provided raw key
- cross-field errors may continue to point at the config object root, as they do today
- keys not present in the effective alias map should continue through the existing unknown-key / not-applicable flow
- unresolved keys should not be inserted into the normalized cross-field map

### 4.4 Validator Construction Must Become Options-Aware

The main validators should become keyword-oriented, and type resolution must stop depending on the legacy regex directly.

Recommended direction:

```go
func getDefaultValidatorForType(typeName string, opts validateConfigOptions) TypeConfigValidator {
	switch typeName {
	case "string":
		return &StringTypeConfig{}
	case "integer":
		return &IntegerTypeConfig{}
	case "number":
		return &NumberTypeConfig{}
	case "array":
		return &ArrayTypeConfig{isCustomTypeRef: opts.customTypeRefMatcher}
	case "object":
		return &ObjectTypeConfig{}
	case "boolean":
		return &BooleanTypeConfig{}
	case "null":
		return &NullTypeConfig{}
	default:
		if opts.customTypeRefMatcher != nil && opts.customTypeRefMatcher(typeName) {
			return &CustomTypeConfig{}
		}
		return nil
	}
}
```

The array validator should also use the configured matcher for `itemTypes` custom type reference checks.

```go
type ArrayTypeConfig struct {
	isCustomTypeRef func(string) bool
}
```

Recommended follow-through:

- `ValidateField(field ConfigKeyword, value any)`
- allowed-key sets keyed by `ConfigKeyword`
- cross-field validation maps keyed by `ConfigKeyword`

The other type validators may remain stateless aside from consuming `ConfigKeyword`.

---

## 5. V1 Option Set Behavior

### 5.1 Accepted V0 Aliases

The legacy `ValidateConfig(...)` wrapper must resolve V0 camelCase input through an explicit alias preset.

| Raw V0 key | Config keyword |
|------------|----------------|
| `enum` | `KeywordEnum` |
| `minimum` | `KeywordMinimum` |
| `maximum` | `KeywordMaximum` |
| `pattern` | `KeywordPattern` |
| `format` | `KeywordFormat` |
| `minLength` | `KeywordMinLength` |
| `maxLength` | `KeywordMaxLength` |
| `exclusiveMinimum` | `KeywordExclusiveMinimum` |
| `exclusiveMaximum` | `KeywordExclusiveMaximum` |
| `multipleOf` | `KeywordMultipleOf` |
| `itemTypes` | `KeywordItemTypes` |
| `minItems` | `KeywordMinItems` |
| `maxItems` | `KeywordMaxItems` |
| `uniqueItems` | `KeywordUniqueItems` |
| `additionalProperties` | `KeywordAdditionalProperties` |

This alias preset should be passed automatically by the legacy wrapper so existing V0 callers do not need to change.

### 5.2 Accepted V1 Aliases

The V1 configuration passed through `ValidateConfigWithOptions(...)` must accept the following raw-to-keyword mappings:

| Raw V1 key | Config keyword |
|------------|----------------|
| `enum` | `KeywordEnum` |
| `minimum` | `KeywordMinimum` |
| `maximum` | `KeywordMaximum` |
| `pattern` | `KeywordPattern` |
| `format` | `KeywordFormat` |
| `min_length` | `KeywordMinLength` |
| `max_length` | `KeywordMaxLength` |
| `exclusive_minimum` | `KeywordExclusiveMinimum` |
| `exclusive_maximum` | `KeywordExclusiveMaximum` |
| `multiple_of` | `KeywordMultipleOf` |
| `item_types` | `KeywordItemTypes` |
| `min_items` | `KeywordMinItems` |
| `max_items` | `KeywordMaxItems` |
| `unique_items` | `KeywordUniqueItems` |
| `additional_properties` | `KeywordAdditionalProperties` |

The V1 alias preset should be explicit and complete. This keeps the resolution model symmetrical with V0 and avoids implicit fallback behavior that is easy to misread or regress.

### 5.3 Unknown V1 Keys

If a raw V1 key is not present in the supplied alias map, it should continue through the existing unknown-key / not-applicable behavior.

Examples:

- `minLength` is not handled specially in the shared package
- if V1 aliases only include `min_length`, then `minLength` should fall through and be handled as an unknown / not-applicable key
- unknown keys must not gain a new dedicated message path in this work

### 5.4 Mixed Case Input

If both a valid snake_case key and an unaliased camelCase key are present in the same V1 config object:

- the snake_case key is validated normally
- the unaliased camelCase key flows through the existing unknown-key / not-applicable behavior

No special duplicate/conflict handling is required for this case.

---

## 6. Custom Type Object Override

The existing custom type object override must remain, but its V1-facing spelling changes through the aliasing layer.

Desired behavior:

- V0 custom type object config continues to accept `additionalProperties`
- V1 custom type object config accepts only `additional_properties`
- V1 `additionalProperties` is not aliased and therefore falls through the existing unknown-key / not-applicable behavior
- any other object config key continues to be rejected by the existing override semantics

This is achieved by mapping accepted V1 `additional_properties` to the shared validator keyword for additional properties.

The override implementation itself does not need a separate V1 version if canonicalization is done correctly.

---

## 7. Implementation Guidance

### Recommended Files to Add or Update

Likely changes for this work:

```text
cli/internal/providers/datacatalog/rules/config/
├── validator.go                  # new options-aware entrypoint, normalization flow
├── utils.go                      # remove direct legacy-only matcher dependency
├── array_validator.go            # options-aware custom type ref checks
├── customtype_validator.go       # no logic change, but type resolution path changes
├── options.go                    # new functional options plus explicit V0/V1 alias presets
└── validator_test.go             # new V1 and backward-compat coverage
```

### Recommended Non-Changes

The following should not be expanded in this PR:

1. property V1 rule wiring
2. custom type V1 rule wiring
3. semantic validation
4. refactoring each validator into a different interface
5. top-level property `item_type` / `item_types` validation

### Future Usage Example

This is not part of the current implementation, but the API should make later wiring simple:

```go
results := config.ValidateConfigWithOptions(
	types,
	property.Config,
	reference,
	config.WithFieldAliases(v1FieldAliases),
	config.WithCustomTypeRefMatcher(v1CustomTypeRefMatcher),
)
```

---

## 8. Edge Cases That Must Be Covered

The implementation must preserve the following behavior:

1. Empty config remains valid
2. V0 config behavior remains unchanged
3. V0 compatibility does not rely on implicit raw-key fallback; it is driven through an explicit V0 alias preset
4. Cross-field validation still uses normalized canonical keys
5. Deduplication behavior remains unchanged
6. V1 accepts unchanged keys such as `enum`, `minimum`, `maximum`, `pattern`, and `format`
7. V1 introduces no special wrong-spelling path in the shared package
8. V1 legacy custom type refs inside `item_types` may fail through the normal invalid-type path
9. V1 object config accepts only `additional_properties`
10. V1 unaliased camelCase `additionalProperties` falls through the existing unknown-key / not-applicable behavior
11. Raw user-facing references in field errors must use the original input key, not the normalized `ConfigKeyword`
12. `enum` remains array-and-duplicate validated only; enum element types are not newly type-checked in this work
13. `pattern` remains string-validated only; regex compilation is not introduced in this work
14. If no validator can be constructed for a type, shared config validation still defers to the surrounding spec/type syntax validators

---

## 9. Test Strategy

### Unit Tests

The implementation must add table-driven tests covering at least:

1. V0 wrapper behavior remains unchanged
2. Legacy wrapper passes an explicit V0 alias preset and does not rely on implicit raw-key fallback
3. V1 normalization accepts valid snake_case keys
4. V1 unchanged-key acceptance for `enum`, `minimum`, `maximum`, `pattern`, `format`
5. Unaliased camelCase keys fall through the existing unknown-key / not-applicable behavior
6. V1 array `item_types` accepts primitive types
7. V1 array `item_types` accepts `#custom-type:<id>`
8. V1 array `item_types` rejects invalid references through the normal invalid-type path
9. V1 custom type object config accepts `additional_properties`
10. V1 custom type object config with unaliased `additionalProperties` falls through the existing unknown-key / not-applicable behavior
11. Mixed valid snake_case + unaliased camelCase in the same config map
12. Cross-field validation still works after normalization:
    - `min_length > max_length`
    - `min_items > max_items`
    - `exclusive_minimum >= exclusive_maximum`
13. Validator override behavior still works with the options-aware entrypoint
14. V1 top-level custom type type detection works in the shared package:
    - `ValidateConfigWithOptions([]string{"#custom-type:Address"}, ...)` rejects config as not allowed
15. `enum` mixed element types remain valid where they are valid today
16. `pattern` invalid regex syntax is not rejected solely for being an invalid regex pattern
17. Unknown type names still cause the shared config validator to defer rather than producing a new type-level error

### Suggested Coverage Focus

The highest-value coverage should be on:

- keyword alias resolution logic
- options-aware validation flow
- array custom type reference handling
- custom type object override behavior

### Verification Commands

The implementation must be validated with:

```bash
go build ./...
make test
make lint
```

If the implementation adds or changes package-level tests, a focused coverage run for the changed package is also recommended:

```bash
go test ./cli/internal/providers/datacatalog/rules/config/... -cover
```

---

## 10. Acceptance Criteria

### Functional

- [ ] A focused V1-aware config validation path exists in `cli/internal/providers/datacatalog/rules/config/`
- [ ] The existing `ValidateConfig(...)` function remains usable for legacy callers and preserves current V0 behavior
- [ ] The legacy `ValidateConfig(...)` wrapper passes an explicit V0 field-alias preset
- [ ] A new `ValidateConfigWithOptions(...)` entrypoint exists for future V1 rule wiring
- [ ] `WithFieldAliases(...)` uses `ConfigKeyword` values rather than raw internal string names
- [ ] V0 camelCase config keys are resolved through explicit aliases rather than implicit fallback
- [ ] V1 accepts snake_case config keys and normalizes them to shared validator keywords
- [ ] V1 does not introduce a dedicated wrong-spelling error path in the shared package
- [ ] V1 array custom type references are recognized only in `#custom-type:<id>` format
- [ ] Legacy custom type refs inside V1 `item_types` are allowed to fail through the normal invalid-type path
- [ ] V1 custom type object config accepts only `additional_properties`
- [ ] V1 `additionalProperties` is not aliased and therefore falls through the existing unknown-key / not-applicable behavior
- [ ] Raw field references in validation results use the original input key provided by the user
- [ ] Existing union semantics, cross-field semantics, and deduplication behavior remain unchanged
- [ ] Existing shallow `enum` behavior remains unchanged
- [ ] Existing shallow `pattern` behavior remains unchanged
- [ ] Existing deferral behavior for unknown/invalid types remains unchanged

### Scope

- [ ] The implementation is limited to shared config validation logic
- [ ] No V1 semantic validation is introduced in this work
- [ ] No property/custom-type V1 rule wiring is introduced in this work
- [ ] No validation for top-level property `item_type` / `item_types` is introduced in this work

### Quality

- [ ] Test coverage for changed files or changed packages is greater than 85%
- [ ] New behavior is covered by table-driven unit tests
- [ ] `go build ./...` passes after the implementation is complete
- [ ] `make test` passes after the implementation is complete
- [ ] `make lint` passes after the implementation is complete

---

## 11. Risks and Follow-Ups

### Risks

1. If raw-key references are not preserved, V1 error paths will point to canonical camelCase names instead of the user input
2. If custom type reference matching remains hard-coded to the legacy regex anywhere in the shared path, V1 arrays will silently miss valid custom type refs
3. If the current `ValidateConfig` signature is replaced instead of wrapped, later rule-wiring deferral will cause unnecessary churn in existing V0 callers
4. If V0 compatibility depends on implicit fallback rather than an explicit alias preset, the `ConfigKeyword` abstraction is likely to be misread and regress later

### Follow-Ups

1. Wire V1 property config validation to the new options-aware API
2. Wire V1 custom type config validation to the new options-aware API
3. Add V1 semantic validation for config references in follow-up PRs
4. Review V0-to-V1 migration helpers separately to ensure config-key conversion remains aligned with the chosen V1 spellings
