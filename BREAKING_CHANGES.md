# Breaking Changes

This document lists breaking changes between releases of `rudder-cli`. It is separate from [CHANGELOG.md](CHANGELOG.md) and covers changes that require users to update their YAML spec files or workflows.

---

## Upcoming (unreleased)

### Spec format upgrade: `rudder/0.1` → `rudder/v1`

`rudder/v1` is a comprehensive redesign of the original `rudder/0.1` spec format. It improves consistency across resource kinds, replaces ambiguous field names, adopts snake_case conventions, and moves to compact URN-style references throughout. All changes below must be applied when upgrading from `rudder/0.1` to `rudder/v1`.

To migrate automatically, run:

```bash
$ rudder-cli migrate --location <path-to-project>
```

This rewrites spec files in place — **commit or back up your project before running.**


Below sections enumerate the changes in the spec format one by one providing the reasoning for the change:

### 1. Version field

**Scope:** spec

**Why:** The field-name, convention, and reference changes in `rudder/v1` break the existing spec contract. The version is bumped to signal the new contract, so tooling and validation can distinguish it from legacy `rudder/0.1`.

All spec files must update the `version` field.

**Before:**

```yaml
version: rudder/0.1
# or
version: rudder/v0.1
```

**After:**

```yaml
version: rudder/v1
```

### 2. Kind rename: tracking plans

**Scope:** spec

**Why:** `tp` was an opaque abbreviation; `tracking-plan` is self-documenting and consistent with how the resource is named everywhere else.

The tracking plan kind changes from `tp` to `tracking-plan`.

**Before:**

```yaml
version: rudder/0.1
kind: tp
```

**After:**

```yaml
version: rudder/v1
kind: tracking-plan
```

### 3. Reference format: path-based → compact URN

**Scope:** spec

**Why:** Path-based refs were verbose and order-dependent. Compact URNs are shorter and encode the resource type, making references unambiguous and easier to read.

All cross-resource references change from path-based (`#/<kind>/<group>/<id>`) to compact URN format (`#<resource-type>:<id>`).

| Resource | Before (v0.1) | After (v1) |
|----------|--------------|------------|
| Property | `#/properties/group/prop_id` | `#property:prop_id` |
| Event | `#/events/group/event_id` | `#event:event_id` |
| Category | `#/categories/group/cat_id` | `#category:cat_id` |
| Custom type | `#/custom-types/group/type_id` | `#custom-type:type_id` |
| Tracking plan | `#/tp/group/tp_id` | `#tracking-plan:tp_id` |

**Before (event referencing a category):**

```yaml
version: rudder/0.1
kind: events
spec:
  events:
    - id: api_tracking
      name: API Tracking
      event_type: track
      category: "#/categories/app_categories/user_actions"
```

**After:**

```yaml
version: rudder/v1
kind: events
spec:
  events:
    - id: api_tracking
      name: API Tracking
      event_type: track
      category: "#category:user_actions"
```

### 4. Property spec changes

**Scope:** spec

**Why:** `propConfig` is renamed to `config` for brevity and consistency with other resource field naming. Config keys move to snake_case to align with YAML ecosystem conventions used throughout the rest of the spec. `type` becomes `types` (an array) to eliminate fragile comma-separated string parsing. Array item types are hoisted to top-level to reduce nesting and make type information more discoverable.

Three changes to property definitions:

**a) `propConfig` renamed to `config`**

```yaml
# Before (v0.1)
propConfig:
  enum: ["GET", "PUT", "POST"]

# After (v1)
config:
  enum: ["GET", "PUT", "POST"]
```

**b) Config keys: camelCase → snake_case**

```yaml
# Before (v0.1)
propConfig:
  minimum: 0
  maximum: 10
  multipleOf: 2
  minLength: 10
  maxLength: 64

# After (v1)
config:
  minimum: 0
  maximum: 10
  multiple_of: 2
  min_length: 10
  max_length: 64
```

**c) Multi-type properties: `type` → `types`**

Properties with multiple types use a `types` array instead of a comma-separated `type` string.

```yaml
# Before (v0.1)
- id: status_code
  type: "integer,null"

# After (v1)
- id: status_code
  types:
    - integer
    - "null"
```

**d) Array item types: hoisted from config to top-level**

```yaml
# Before (v0.1)
- id: tag_list
  type: array
  propConfig:
    itemTypes:
      - string

# After (v1)
- id: tag_list
  type: array
  item_type: string

# Multiple item types
- id: user_scores
  type: array
  item_types:
    - integer
    - number
```

**Full before/after example:**

```yaml
# Before (v0.1)
version: rudder/v0.1
kind: properties
metadata:
  name: api_tracking
spec:
  properties:
    - id: api_method
      name: API Method
      type: string
      description: "http method of the api called"
      propConfig:
        enum: ["GET", "PUT", "POST", "DELETE", "PATCH"]
    - id: http_retry_count
      name: HTTP Retry Count
      type: integer
      description: "Number of times to retry the API call"
      propConfig:
        minimum: 0
        maximum: 10
        multipleOf: 2
    - id: password
      name: Password
      type: string
      propConfig:
        minLength: 10
        maxLength: 64
```

```yaml
# After (v1)
version: rudder/v1
kind: properties
metadata:
  name: api_tracking
spec:
  properties:
    - id: api_method
      name: API Method
      type: string
      description: "http method of the api called"
      config:
        enum: ["GET", "PUT", "POST", "DELETE", "PATCH"]
    - id: http_retry_count
      name: HTTP Retry Count
      type: integer
      description: "Number of times to retry the API call"
      config:
        minimum: 0
        maximum: 10
        multiple_of: 2
    - id: password
      name: Password
      type: string
      config:
        min_length: 10
        max_length: 64
```

### 5. Custom type changes

**Scope:** spec

**Why:** `$ref` is a JSON Schema artifact that carries no semantic meaning in this context; `property` is clearer. Config keys follow the same snake_case convention as properties.

**a) `$ref` renamed to `property` in type properties**

```yaml
# Before (v0.1)
spec:
  types:
    - id: login
      type: object
      properties:
        - $ref: "#/properties/api_tracking/username"
          required: true

# After (v1)
spec:
  types:
    - id: login
      type: object
      properties:
        - property: "#property:username"
          required: true
```

**b) Config keys follow the same camelCase → snake_case convention as properties**

```yaml
# Before (v0.1)
config:
  minLength: 10
  maxLength: 255

# After (v1)
config:
  min_length: 10
  max_length: 255
```

### 6. Tracking plan rule changes

**Scope:** spec

**Why:** The event object wrapper was redundant — `event` now holds a direct reference string, consistent with how other references are expressed. `allow_unplanned` is renamed to `additional_properties` and moved to rule level for clarity. Rule property references follow the same `$ref` → `property` rename as custom types, using compact URNs.

Two structural changes to tracking plan rules:

**a) `event` changes from object to direct reference string**

```yaml
# Before (v0.1)
rules:
  - type: event_rule
    id: login
    event:
      $ref: "#/events/api_tracking/api_tracking"
      allow_unplanned: false
      identity_section: properties

# After (v1)
rules:
  - type: event_rule
    id: login
    event: "#event:api_tracking"
    additional_properties: false
    identity_section: properties
```

- `event.$ref` → `event` (direct string)
- `event.allow_unplanned` → `additional_properties` (moved to rule level, renamed)
- `event.identity_section` → `identity_section` (moved to rule level)

**b) `$ref` renamed to `property` in rule properties**

```yaml
# Before (v0.1)
properties:
  - $ref: "#/properties/api_tracking/username"
    required: true
  - $ref: "#/properties/api_tracking/password"
    required: true

# After (v1)
properties:
  - property: "#property:username"
    required: true
  - property: "#property:password"
    required: true
```

**Full before/after example:**

```yaml
# Before (v0.1)
version: rudder/v0.1
kind: tp
metadata:
  name: api_tracking
spec:
  id: api_tracking
  display_name: API Tracking
  rules:
    - type: event_rule
      id: login
      event:
        $ref: "#/events/api_tracking/api_tracking"
        allow_unplanned: false
      properties:
        - $ref: "#/properties/api_tracking/username"
          required: true
        - $ref: "#/properties/api_tracking/password"
          required: true
```

```yaml
# After (v1)
version: rudder/v1
kind: tracking-plan
metadata:
  name: api_tracking
spec:
  id: api_tracking
  display_name: API Tracking
  rules:
    - type: event_rule
      id: login
      event: "#event:api_tracking"
      additional_properties: false
      properties:
        - property: "#property:username"
          required: true
        - property: "#property:password"
          required: true
```

### 7. Variant changes

**Scope:** spec

**Why:** Variants appear in both custom types and tracking plan rules. The `default` field is restructured from an array-of-objects pattern to an explicit `properties` wrapper that matches the shape used in rules. All `$ref` fields within variant discriminators and cases follow the same rename to `property` with compact URN references.

Two changes to variant definitions:

**a) Variant `default` restructured from array to object**

```yaml
# Before (v0.1)
variants:
  - type: discriminator
    default:
      - $ref: "#/properties/group/prop_a"
        required: true

# After (v1)
variants:
  - type: discriminator
    default:
      properties:
        - property: "#property:prop_a"
          required: true
```

**b) Variant discriminator and case properties: `$ref` → `property` with compact URN**

```yaml
# Before (v0.1)
variants:
  - type: discriminator
    discriminator: "#/properties/api_tracking/api_method"
    cases:
      - display_name: "Create Entity"
        match: ["POST"]
        properties:
          - $ref: "#/properties/api_tracking/user_agent"
            required: true

# After (v1)
variants:
  - type: discriminator
    discriminator: "#property:api_method"
    cases:
      - display_name: "Create Entity"
        match: ["POST"]
        properties:
          - property: "#property:user_agent"
            required: true
```

### 8. Import metadata: `local_id` → `urn`

**Scope:** spec, import

**Why:** The `urn` field encodes the resource type alongside the local ID, eliminating ambiguity when multiple resource types share the same ID values.

The `local_id` field in import metadata is replaced by a `urn` field that includes the resource type.

**Before:**

```yaml
metadata:
  name: test_props
  import:
    workspaces:
      - workspace_id: ws-123
        resources:
          - local_id: prop1
            remote_id: remote-prop-1
```

**After:**

```yaml
metadata:
  name: test_props
  import:
    workspaces:
      - workspace_id: ws-123
        resources:
          - urn: property:prop1
            remote_id: remote-prop-1
```

URN format is `<resource-type>:<local-id>`. Resource types include: `property`, `event`, `category`, `custom-type`, `tracking-plan`, `event-stream-source`, `retl-source-sql-model`.

### 9. Event stream source: tracking plan reference

**Scope:** spec, event-stream

**Why:** Consistency with the unified compact URN format adopted for all cross-resource references in v1.

The tracking plan reference in event stream source governance uses compact URN format.

**Before:**

```yaml
version: rudder/0.1
kind: event-stream-source
spec:
  id: test-source
  name: Test Source
  type: javascript
  governance:
    validations:
      tracking_plan: "#/tp/group/tp-abc123"
```

**After:**

```yaml
version: rudder/v1
kind: event-stream-source
spec:
  id: test-source
  name: Test Source
  type: javascript
  governance:
    validations:
      tracking_plan: "#tracking-plan:tp-abc123"
```

---

<!-- Convention for future entries:
     - Add new entries under ## Upcoming (unreleased), newest at the top of that section.
     - Each entry is an H3: ### <Action: what changed>
     - Required fields: Scope, Migration. Why is optional but encouraged.
     - At release time: move Upcoming entries into ## vX.Y.Z — YYYY-MM-DD.
     - Add "Deprecated in: vX.Y.Z" backreference on Removal entries.
     - Use horizontal rules (---) between H2 version blocks only.
-->
