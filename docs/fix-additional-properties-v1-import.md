# Fix: Convert `additionalProperties` to `additional_properties` in V1 custom type import

## Context

When importing custom types from upstream for V1 specs, the `additionalProperties` config key (camelCase from the API) is not being converted to `additional_properties` (snake_case for V1). This happens because `additionalProperties` is missing from the `SupportedV0ConfigKeys` list used by `ConvertConfigKeysToSnakeCase()`.

**Impact**: Imported V1 custom type specs contain `additionalProperties` (camelCase) in their config, which V1 validation rejects as an unsupported field (since `V1FieldAliases` only maps `"additional_properties"`).

## Changes

### 1. Add `additionalProperties` to `SupportedV0ConfigKeys`

**File**: `cli/internal/providers/datacatalog/localcatalog/model_v1.go` (line 82-97)

Add `"additionalProperties"` to the `SupportedV0ConfigKeys` slice. This enables `ConvertConfigKeysToSnakeCase()` to convert it to `additional_properties` via the `SnakeCase` namer.

### 2. Add test case for V1 custom type import with `additionalProperties`

**File**: `cli/internal/providers/datacatalog/importremote/model/customtype_test.go`

Add a test in `TestCustomTypeForExport` that:
- Provides upstream `catalog.CustomType` with `Config: {"additionalProperties": true}` and `Type: "object"`
- Asserts V1 export output contains `"additional_properties": true` (snake_case) in the config map

### 3. Update existing `TestConvertConfigKeysToSnakeCase` test

**File**: `cli/internal/providers/datacatalog/localcatalog/model_v1_test.go`

The "converts all camelCase keys" subtest already covers all existing `SupportedV0ConfigKeys`. Add `"additionalProperties": true` to the input map and assert `result["additional_properties"] == true` + `result` does not contain `"additionalProperties"`.

## Key Files

- `cli/internal/providers/datacatalog/localcatalog/model_v1.go` — `SupportedV0ConfigKeys` list + `ConvertConfigKeysToSnakeCase()`
- `cli/internal/providers/datacatalog/importremote/model/customtype.go` — Import flow (V0 copies config as-is, V1 goes through FromV0)
- `cli/internal/providers/datacatalog/importremote/model/customtype_test.go` — Import tests
- `cli/internal/providers/datacatalog/rules/config/options.go` — `V1FieldAliases` maps `"additional_properties"` to `KeywordAdditionalProperties`

## Verification

1. `go test ./cli/internal/providers/datacatalog/localcatalog/...` — unit tests for config conversion
2. `go test ./cli/internal/providers/datacatalog/importremote/model/...` — import/export tests
3. `make test` — full unit test suite
4. `make lint` — lint check
