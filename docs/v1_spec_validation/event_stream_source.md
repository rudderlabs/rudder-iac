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

The event-stream source spec structure is **identical** between V0.1 and V1. The same `SourceSpec` struct is used for both versions. The only difference is the tracking plan reference format in the governance section.

| Aspect | V0.1 | V1 |
|--------|------|----|
| Tracking plan ref format | `#/tp/<group>/<id>` (validated by `pattern=legacy_tracking_plan_ref`) | `#tracking-plan:<id>` |
| SourceSpec struct | Same | Same |

### Struct (`SourceSpec` in `source/model.go`)

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

---

## Syntactic Validations to Add for V1

These rules must target `MatchKindVersion("event-stream-source", "rudder/v1")` and decode into `SourceSpec`.

| # | Validation | Description |
|---|-----------|-------------|
| 1 | `id` required | Source must have a non-empty `id` field |
| 2 | `name` required | Source must have a non-empty `name` field |
| 3 | `type` required and in allowed definitions | `type` must be one of: java, dotnet, php, flutter, cordova, rust, react_native, python, ios, android, javascript, go, node, ruby, unity |
| 4 | Governance: `tracking_plan` required when governance present | If `governance.validations` is specified, `tracking_plan` ref must be present |
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

- [ ] All 5 syntactic validations listed above are implemented as V1 rules targeting `MatchKindVersion("event-stream-source", "rudder/v1")`
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

* Add V1 syntactic rule for source spec validation (required fields, type in allowed definitions, governance field requirements)
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
