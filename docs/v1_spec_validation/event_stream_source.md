# Event-Stream Source - V1 Spec Validation

**Ticket**: [DEX-257 - V1 Spec Validation](https://linear.app/rudderstack/issue/DEX-257/v1-spec-validation)

---

## V0.1 Spec Validation Status

V0.1 validation rules **partially exist** in the new validation engine. Two rules are registered via `LegacyVersionPatterns("event-stream-source")` targeting `rudder/0.1` and `rudder/v0.1` only:

| Rule ID | Phase | Location |
|---------|-------|----------|
| `event-stream/source/spec-syntax-valid` | Syntactic | `cli/internal/providers/event-stream/rules/source/source_spec_valid.go` |
| `event-stream/source/semantic-valid` | Semantic | `cli/internal/providers/event-stream/rules/source/source_semantic_valid.go` |

Additionally, the handler (`source/handler.go`) performs several validation checks during `LoadSpec` and `Validate` that are **not ported** to the new engine for any version:
- Unknown fields rejection (`ErrorUnused: true`)
- `type` must be in allowed `sourceDefinitions`
- Governance field requirements when governance is present

These rules do **not** match V1 specs (`rudder/v1`).

---

## Structural Differences: V0.1 vs V1

The event-stream source spec structure is **identical** between V0.1 and V1. The same `SourceSpec` struct is used for both versions. The struct **already has `validate:` tags** on `SourceSpec` for core fields (`id`, `name`, `type`).

| Aspect | V0.1 | V1 |
|--------|------|----|
| Tracking plan ref format | `#/tp/<group>/<id>` (validated by `pattern=legacy_tracking_plan_ref`) | `#tracking-plan:<id>` (use `pattern=tracking_plan_ref`) |
| SourceSpec struct | Same | Same |
| Validation tags on SourceSpec | **Already present** (`required`, `oneof`) | **Already present** (shared struct) |
| TrackingPlanSpec.Ref tag | `pattern=legacy_tracking_plan_ref` | **Needs V1 handling** (see note below) |

### Struct (`SourceSpec` in `source/model.go`) — Already Has Tags

```go
type SourceSpec struct {
    LocalID          string                `json:"id"         mapstructure:"id"         validate:"required"`
    Name             string                `json:"name"       mapstructure:"name"       validate:"required"`
    SourceDefinition string                `json:"type"       mapstructure:"type"       validate:"required,oneof=java dotnet php flutter cordova rust react_native python ios android javascript go node ruby unity"`
    Enabled          *bool                 `json:"enabled"    mapstructure:"enabled"`
    Governance       *SourceGovernanceSpec `json:"governance" mapstructure:"governance"`
}

type SourceGovernanceSpec struct {
    TrackingPlan *TrackingPlanSpec `json:"validations" mapstructure:"validations"`
}

type TrackingPlanSpec struct {
    Ref    string                  `json:"tracking_plan" mapstructure:"tracking_plan" validate:"required,pattern=legacy_tracking_plan_ref"`
    Config *TrackingPlanConfigSpec `json:"config"        mapstructure:"config"        validate:"required"`
}
```

**Note on tracking plan ref pattern**: The existing `TrackingPlanSpec.Ref` tag uses `pattern=legacy_tracking_plan_ref` (V0.1 format `#/tp/<group>/<id>`). Since the struct is shared between V0.1 and V1, the tag cannot be changed without breaking V0.1. Two approaches:

1. **Preferred**: Keep existing tags for `SourceSpec` core fields (validations #1-#3 handled by `rules.ValidateStruct()`). For governance tracking plan ref validation, use **custom rule logic** that checks the V1 ref format (`#tracking-plan:<id>`) via the already-registered `tracking_plan_ref` pattern.
2. **Alternative**: Create a V1-specific governance struct (`TrackingPlanSpecV1`) with `validate:"required,pattern=tracking_plan_ref"` on the `Ref` field.

---

## Syntactic Validations to Add for V1

These rules must target `MatchKindVersion("event-stream-source", "rudder/v1")` and decode into `SourceSpec`.

### Tag-Based (handled by `rules.ValidateStruct()` via existing tags)

| # | Validation | Existing Tag | Description |
|---|-----------|--------------|-------------|
| 1 | `id` required | `validate:"required"` on `LocalID` | Source must have a non-empty `id` |
| 2 | `name` required | `validate:"required"` on `Name` | Source must have a non-empty `name` |
| 3 | `type` required + oneof | `validate:"required,oneof=java dotnet php..."` on `SourceDefinition` | Must be one of the allowed source definitions |

### Custom Logic (manual rule code)

| # | Validation | Description |
|---|-----------|-------------|
| 4 | Governance: `tracking_plan` required and format valid | If `governance.validations` is specified, `tracking_plan` ref must be present and match V1 format `#tracking-plan:<id>` (use `tracking_plan_ref` pattern in custom code since the struct tag uses legacy pattern) |
| 5 | Governance: `config` required when governance present | If `governance.validations` is specified, `config` must be present |

---

## Semantic Validations to Add for V1

| # | Validation | Description |
|---|-----------|-------------|
| 1 | Tracking plan URN exists in graph | The tracking plan reference (format `#tracking-plan:<id>`) must resolve to an existing resource in the graph |
| 2 | Referenced resource is a tracking plan | The resolved resource must be of type `tracking-plan` |
| 3 | Source `name` uniqueness | No two event-stream sources may share the same name |

Note: `id` uniqueness is handled by the project-level rule `project/duplicate-local-id` which is version-agnostic.

---

## Acceptance Criteria

- [ ] No struct tag changes needed on `SourceSpec` (existing tags already cover #1-#3)
- [ ] V1 syntactic rule uses `rules.ValidateStruct()` for tag-based validations (#1-#3)
- [ ] Custom logic implemented for governance tracking plan ref format validation (#4) using V1 `tracking_plan_ref` pattern, and governance config requirement (#5)
- [ ] All 3 semantic validations listed above are implemented as V1 rules
- [ ] Tracking plan ref validation uses V1 ref format (`#tracking-plan:<id>`)
- [ ] All validations are tested with unit tests
- [ ] Test coverage for changed files exceeds 85%

---

## PR Process

**Branch**: `feat/dex-257-event-stream-source-v1-spec-validation`

**Target Branch**: `main`

**PR Template** (from `.github/pull_request_template.md`):

```markdown
## 🔗 Ticket

[DEX-257](https://linear.app/rudderstack/issue/DEX-257/v1-spec-validation)

---

## Summary

Add V1 spec validation rules for the `event-stream-source` resource. These rules target `rudder/v1` specs, implementing 5 syntactic and 3 semantic validations including governance field validation and tracking plan reference resolution.

---

## Changes

* Add V1 syntactic rule using `rules.ValidateStruct()` for existing tag-based validations (id, name, type) + custom logic for governance tracking plan V1 ref format and config requirement
* Add V1 semantic rule for tracking plan reference resolution and source name uniqueness

---

## Testing

* Unit tests for all syntactic validations
* Unit tests for all semantic validations
* Table-driven tests covering valid and invalid V1 source specs

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
