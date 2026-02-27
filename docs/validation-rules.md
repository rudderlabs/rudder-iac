# Validation Rules

> **Spec Version:** rudder/v1
> **Other Versions:** [v0.1](./validation-rules/v0.1.md)

This document describes all validation rules enforced by the `rudder-cli validate` command.

## Table of Contents

- [Global](#global)
  - [project/metadata-syntax-valid](#projectmetadata-syntax-valid)
  - [project/spec-syntax-valid](#projectspec-syntax-valid)
  - [project/spec-values-valid](#projectspec-values-valid)
  - [project/duplicate-local-id](#projectduplicate-local-id)
- [Properties](#properties)
  - [datacatalog/properties/spec-syntax-valid](#datacatalogpropertiesspec-syntax-valid)
  - [datacatalog/properties/config-valid](#datacatalogpropertiesconfig-valid)
  - [datacatalog/properties/semantic-valid](#datacatalogpropertiessemantic-valid)
- [Events](#events)
  - [datacatalog/events/spec-syntax-valid](#datacatalogeventsspec-syntax-valid)
- [Categories](#categories)
  - [datacatalog/categories/spec-syntax-valid](#datacatalogcategoriesspec-syntax-valid)
  - [datacatalog/categories/semantic-valid](#datacatalogcategoriessemantic-valid)
- [Custom Types](#custom-types)
  - [datacatalog/custom-types/spec-syntax-valid](#datacatalogcustom-typesspec-syntax-valid)
  - [datacatalog/custom-types/config-valid](#datacatalogcustom-typesconfig-valid)
  - [datacatalog/custom-types/semantic-valid](#datacatalogcustom-typessemantic-valid)
- [Tracking Plans](#tracking-plans)
  - [datacatalog/tracking-plans/spec-syntax-valid](#datacatalogtracking-plansspec-syntax-valid)
  - [datacatalog/tracking-plans/semantic-valid](#datacatalogtracking-planssemantic-valid)
- [Event Stream](#event-stream)
  - [event-stream/source/spec-syntax-valid](#event-streamsourcespec-syntax-valid)
  - [event-stream/source/semantic-valid](#event-streamsourcesemantic-valid)
- [RETL](#retl)
  - [retl/sqlmodel/spec-syntax-valid](#retlsqlmodelspec-syntax-valid)

---

## Global

### project/metadata-syntax-valid

![Severity: Error](https://img.shields.io/badge/severity-error-red)

**Description:** metadata syntax must be valid

**Applies to:** `*`

**Validation Phase:** Syntactic

**Available in:** [v1](./#projectmetadata-syntax-valid) | [v0.1](./validation-rules/v0.1.md#projectmetadata-syntax-valid)

**Checks Performed:**
- `name` field is required in metadata
- If `import` block is present:
  - `workspaces` array must have valid entries
  - Each workspace must have `workspace_id`
  - Each resource in workspace must have `local_id` and `remote_id`
  - Each `local_id` must exist as an external ID in the spec body

**Valid Examples:**

```yaml
metadata:
  name: my-project
```

```yaml
metadata:
  name: my-project
  import:
    workspaces:
      - workspace_id: ws-123
        resources:
          - local_id: src-local
            remote_id: src-remote-456
```

**Invalid Examples:**

```yaml
metadata: # name is missing
  import:
    workspaces:
      - workspace_id: ws-123
```

```yaml
metadata:
  name: my-project
  import:
    workspaces: # missing workspace_id
      - resources:
          - local_id: src-local
            remote_id: src-remote-456
```

---

### project/spec-syntax-valid

![Severity: Error](https://img.shields.io/badge/severity-error-red)

**Description:** spec syntax must be valid

**Applies to:** `*`

**Validation Phase:** Syntactic

**Available in:** [v1](./#projectspec-syntax-valid) | [v0.1](./validation-rules/v0.1.md#projectspec-syntax-valid)

**Checks Performed:**
- `kind` field is required
- `version` field is required
- `metadata` section is required
- `spec` section is required

**Valid Examples:**

```yaml
version: rudder/v1
kind: properties
metadata:
  name: my-properties
spec:
  properties:
    - name: MyTestProperty
      type: string
```

**Invalid Examples:**

```yaml
version: rudder/v1
kind: # missing kind
metadata:
  name: my-properties
spec:
  properties:
    - name: MyTestProperty
      type: string
```

```yaml
version: # missing version
kind: properties
metadata:
  name: my-properties
spec:
  properties:
    - name: MyTestProperty
      type: string
```

```yaml
version: rudder/v1
kind: properties
metadata: # missing metadata name
spec:
  properties:
    - name: MyTestProperty
      type: string
```

```yaml
version: rudder/v1
kind: properties
metadata:
  name: my-properties
# missing spec
```

---

### project/spec-values-valid

![Severity: Error](https://img.shields.io/badge/severity-error-red)

**Description:** spec kind and version must be valid and supported

**Applies to:** `*`

**Validation Phase:** Syntactic

**Available in:** [v1](./#projectspec-values-valid) | [v0.1](./validation-rules/v0.1.md#projectspec-values-valid)

**Checks Performed:**
- `version` must be one of: `rudder/v1`, `rudder/0.1`, `rudder/v0.1`
- `kind` must be a supported kind for the provider

**Valid Examples:**

```yaml
version: rudder/v1
kind: properties
metadata:
  name: my-properties
spec:
  properties:
    - name: MyTestProperty
      type: string
```

```yaml
version: rudder/0.1
kind: events
metadata:
  name: my-events
spec:
  events:
    - name: MyTestEvent
```

**Invalid Examples:**

```yaml
version: rudder/v2  # unsupported version
kind: properties
metadata:
  name: my-properties
spec:
  properties:
    - name: MyTestProperty
      type: string
```

```yaml
version: rudder/v1
kind: unsupported-kind  # unsupported kind
metadata:
  name: my-test
spec:
  data: []
```

---

### project/duplicate-local-id

![Severity: Error](https://img.shields.io/badge/severity-error-red)

**Description:** local IDs must be unique within each kind

**Applies to:** `*`

**Validation Phase:** Project (cross-file)

**Available in:** [v1](./#projectduplicate-local-id) | [v0.1](./validation-rules/v0.1.md#projectduplicate-local-id)

**Checks Performed:**
- Local IDs (the `id` field) must be unique within the same kind
- Duplicate IDs across different files are flagged in all locations

---

## Properties

### datacatalog/properties/spec-syntax-valid

![Severity: Error](https://img.shields.io/badge/severity-error-red)

**Description:** property spec syntax must be valid

**Applies to:** `properties`

**Validation Phase:** Syntactic

**Available in:** [v1](./#datacatalogpropertiesspec-syntax-valid) | [v0.1](./validation-rules/v0.1.md#datacatalogpropertiesspec-syntax-valid)

**Checks Performed:**
- Each property must have an `id` field (required)
- Each property must have a `name` field (required, 1-65 characters)
- `description` if provided must be 3-2000 characters
- `type` must be:
  - A valid primitive type: `string`, `number`, `integer`, `boolean`, `null`, `array`, `object`
  - A custom type reference: `#/custom-types/<group>/<id>`
  - A comma-separated union of unique primitive types (e.g., `string,number`)

**Valid Examples:**

```yaml
properties:
  - id: user_id
    name: User ID
    description: Unique identifier for the user
    type: string
  - id: email
    name: Email
    type: string
```

**Invalid Examples:**

```yaml
properties:
  - name: Missing ID
    type: string
```

```yaml
properties:
  - id: user_id
    # Missing required name field
    type: string
```

```yaml
properties:
  - id: user_id
    name: ""
    # Name cannot be empty
    type: string
```

```yaml
properties:
  - id: user_id
    name: This is a very long name that exceeds the maximum allowed length of sixty five characters for a property name
    # Name exceeds 65 characters
    type: string
```

---

### datacatalog/properties/config-valid

![Severity: Error](https://img.shields.io/badge/severity-error-red)

**Description:** property config must be valid for the given type

**Applies to:** `properties`

**Validation Phase:** Syntactic

**Available in:** [v1](./#datacatalogpropertiesconfig-valid) | [v0.1](./validation-rules/v0.1.md#datacatalogpropertiesconfig-valid)

**Checks Performed:**
- Config is only allowed for configurable types (not `object` or `null`)
- **String type:**
  - `format`: one of `date-time`, `date`, `time`, `email`, `uuid`, `hostname`, `ipv4`, `ipv6`
  - `minLength`: non-negative integer
  - `maxLength`: non-negative integer, must be >= `minLength`
  - `pattern`: string (regex pattern)
  - `enum`: array of strings
- **Integer type:**
  - `minimum`: integer
  - `maximum`: integer, must be >= `minimum`
  - `enum`: array of integers
- **Number type:**
  - `minimum`: number
  - `maximum`: number, must be >= `minimum`
  - `enum`: array of numbers
- **Array type:**
  - `itemTypes`: array of type strings or custom type references
  - `minItems`: non-negative integer
  - `maxItems`: non-negative integer, must be >= `minItems`
- **Boolean type:**
  - `enum`: array of booleans

**Valid Examples:**

```yaml
properties:
  - id: user_email
    name: UserEmail
    type: string
    propConfig:
      format: "email"
      minLength: 5
      maxLength: 100
```

```yaml
properties:
  - id: age
    name: Age
    type: integer
    propConfig:
      minimum: 0
      maximum: 120
```

```yaml
properties:
  - id: tags
    name: Tags
    type: array
    propConfig:
      itemTypes: ["string"]
      minItems: 1
      maxItems: 10
```

**Invalid Examples:**

```yaml
properties:
  - id: address
    name: Address
    type: object
    propConfig:
      # Config not allowed for object type
      properties: []
```

```yaml
properties:
  - id: email
    name: Email
    type: string
    propConfig:
      # Invalid format value
      format: invalid
```

```yaml
properties:
  - id: count
    name: Count
    type: integer
    propConfig:
      # minimum must be integer not float
      minimum: 1.5
```

---

### datacatalog/properties/semantic-valid

![Severity: Error](https://img.shields.io/badge/severity-error-red)

**Description:** property references must resolve to existing resources

**Applies to:** `properties`

**Validation Phase:** Semantic

**Available in:** [v1](./#datacatalogpropertiessemantic-valid) | [v0.1](./validation-rules/v0.1.md#datacatalogpropertiessemantic-valid)

**Checks Performed:**
- Custom type references in `type` field must exist in resource graph
- Custom type references in `propConfig.itemTypes` must exist in resource graph
- Property (name, type, itemTypes) combination must be unique across the catalog

---

## Events

### datacatalog/events/spec-syntax-valid

![Severity: Error](https://img.shields.io/badge/severity-error-red)

**Description:** event spec syntax must be valid

**Applies to:** `events`

**Validation Phase:** Syntactic

**Available in:** [v1](./#datacatalogeventsspec-syntax-valid) | [v0.1](./validation-rules/v0.1.md#datacatalogeventsspec-syntax-valid)

**Checks Performed:**
- Each event must have an `id` field (required)
- Each event must have an `event_type` field: `track`, `screen`, `identify`, `group`, `page`
- `description` if provided must be 3-2000 characters and start with a letter
- For `track` events: `name` is required (1-64 characters)
- For non-track events: `name` should be empty
- `category` reference must match pattern `#/categories/<group>/<id>`

**Valid Examples:**

```yaml
events:
  - id: page_viewed
    event_type: track
    name: Page Viewed
    description: User viewed a page
  - id: product_clicked
    name: Product Clicked
    event_type: track
```

**Invalid Examples:**

```yaml
events:
  - name: Missing ID
    event_type: track
```

```yaml
events:
  - id: missing_type
    # Missing required event_type field
```

---

## Categories

### datacatalog/categories/spec-syntax-valid

![Severity: Error](https://img.shields.io/badge/severity-error-red)

**Description:** category spec syntax must be valid

**Applies to:** `categories`

**Validation Phase:** Syntactic

**Available in:** [v1](./#datacatalogcategoriesspec-syntax-valid) | [v0.1](./validation-rules/v0.1.md#datacatalogcategoriesspec-syntax-valid)

**Checks Performed:**
- Each category must have an `id` field (required)
- Each category must have a `name` field (required)
- `name` must match display name pattern: start with letter/underscore, followed by 2-64 alphanumeric/space/comma/period/hyphen characters

**Valid Examples:**

```yaml
categories:
  - id: user_actions
    name: User Actions
  - id: system_events
    name: System Events
```

**Invalid Examples:**

```yaml
categories:
  - name: Missing ID
```

```yaml
categories:
  - id: missing_name
    # Missing required name field
```

---

### datacatalog/categories/semantic-valid

![Severity: Error](https://img.shields.io/badge/severity-error-red)

**Description:** category names must be unique across the catalog

**Applies to:** `categories`

**Validation Phase:** Semantic

**Available in:** [v1](./#datacatalogcategoriessemantic-valid) | [v0.1](./validation-rules/v0.1.md#datacatalogcategoriessemantic-valid)

**Checks Performed:**
- Category names must be globally unique in the catalog

---

## Custom Types

### datacatalog/custom-types/spec-syntax-valid

![Severity: Error](https://img.shields.io/badge/severity-error-red)

**Description:** custom type spec syntax must be valid

**Applies to:** `custom-types`

**Validation Phase:** Syntactic

**Available in:** [v1](./#datacatalogcustom-typesspec-syntax-valid) | [v0.1](./validation-rules/v0.1.md#datacatalogcustom-typesspec-syntax-valid)

**Checks Performed:**
- Each custom type must have an `id` field (required)
- Each custom type must have a `name` field (required, 2-65 characters)
- `name` must start with uppercase and contain only alphanumeric, underscores, or hyphens
- Each custom type must have a `type` field: `string`, `number`, `integer`, `boolean`, `array`, `object`, `null`
- `description` if provided must be 3-2000 characters and start with a letter
- `properties` array must have valid property references (`$ref` required)
- `variants` are only allowed for `object` type, max 1 variant

**Valid Examples:**

```yaml
types:
  - id: address
    name: Address
    description: Physical address structure
    type: object
  - id: user_status
    name: User Status
    type: string
```

**Invalid Examples:**

```yaml
types:
  - name: Missing ID
    type: string
```

```yaml
types:
  - id: user_status
    # Missing required name field
    type: string
```

```yaml
types:
  - id: status
    name: Status
    # Missing required type field
```

---

### datacatalog/custom-types/config-valid

![Severity: Error](https://img.shields.io/badge/severity-error-red)

**Description:** custom type config must be valid for the given type

**Applies to:** `custom-types`

**Validation Phase:** Syntactic

**Available in:** [v1](./#datacatalogcustom-typesconfig-valid) | [v0.1](./validation-rules/v0.1.md#datacatalogcustom-typesconfig-valid)

**Checks Performed:**
- Same config validation rules as properties apply
- Config keywords must be appropriate for the declared type

**Valid Examples:**

```yaml
types:
  - id: user_status
    name: UserStatus
    type: string
    config:
      enum: ["active", "inactive"]
      pattern: "^[a-z]+$"
```

```yaml
types:
  - id: age
    name: Age
    type: integer
    config:
      minimum: 0
      maximum: 120
```

```yaml
types:
  - id: tags
    name: Tags
    type: array
    config:
      itemTypes: ["string"]
      minItems: 1
```

**Invalid Examples:**

```yaml
types:
  - id: address
    name: Address
    type: object
    config:
      # Config not allowed for object type
      properties: []
```

```yaml
types:
  - id: status
    name: Status
    type: string
    config:
      # Invalid format value
      format: invalid
```

```yaml
types:
  - id: count
    name: Count
    type: integer
    config:
      # enum values must be integers
      enum: [1.5, 2.5]
```

---

### datacatalog/custom-types/semantic-valid

![Severity: Error](https://img.shields.io/badge/severity-error-red)

**Description:** custom type references must resolve to existing resources

**Applies to:** `custom-types`

**Validation Phase:** Semantic

**Available in:** [v1](./#datacatalogcustom-typessemantic-valid) | [v0.1](./validation-rules/v0.1.md#datacatalogcustom-typessemantic-valid)

**Checks Performed:**
- Property references (`$ref` in properties) must exist in resource graph
- Custom type references in `config.itemTypes` must exist in resource graph
- Variant discriminators must reference valid properties
- Custom type names must be globally unique in the catalog

---

## Tracking Plans

### datacatalog/tracking-plans/spec-syntax-valid

![Severity: Error](https://img.shields.io/badge/severity-error-red)

**Description:** tracking plan spec syntax must be valid

**Applies to:** `tp`

**Validation Phase:** Syntactic

**Available in:** [v1](./#datacatalogtracking-plansspec-syntax-valid) | [v0.1](./validation-rules/v0.1.md#datacatalogtracking-plansspec-syntax-valid)

**Checks Performed:**
- `display_name` is required and must match display name pattern
- `description` if provided must be 3-2000 characters and start with a letter
- Each rule must have an `id` field (required)
- Each rule must have an `event` reference (required)
- Event reference must match pattern `#/events/<group>/<id>` or `#event:<id>`
- Property references must match pattern `#/properties/<group>/<id>` or `#property:<id>`
- Variant `type` must be `discriminator`
- Variant discriminator cannot be empty
- Variant must have at least 1 case
- Property nesting depth must not exceed 3 levels

**Valid Examples:**

```yaml
id: test_tp
display_name: Test Tracking Plan
rules:
  - id: signup_rule
    type: event_rule
    event:
      $ref: "#/events/user-events/signup"
    variants:
      - type: discriminator
        discriminator: "#/properties/signup-props/signup_method"
        cases:
          - display_name: "Email Signup"
            match: ["email"]
            description: "User signed up via email"
            properties:
              - $ref: "#/properties/signup-props/email_address"
                required: true
              - $ref: "#/properties/signup-props/email_verified"
                required: false
        default:
          - $ref: "#/properties/common/user_id"
            required: true
```

**Invalid Examples:**

```yaml
id: test_tp
display_name: Test Tracking Plan
rules:
  - id: invalid_rule
    type: event_rule
    variants:
      - type: "wrong_type"  # Must be "discriminator"
        discriminator: "#/properties/props/field"
        cases:
          - display_name: "Case 1"
            properties:
              - $ref: "#/properties/props/prop1"
```

```yaml
id: test_tp
display_name: Test Tracking Plan
rules:
  - id: invalid_rule
    type: event_rule
    variants:
      - type: discriminator
        discriminator: ""  # Cannot be empty
        cases: []  # Must have at least 1 case
```

---

### datacatalog/tracking-plans/semantic-valid

![Severity: Error](https://img.shields.io/badge/severity-error-red)

**Description:** tracking plan references must resolve to existing resources

**Applies to:** `tp`

**Validation Phase:** Semantic

**Available in:** [v1](./#datacatalogtracking-planssemantic-valid) | [v0.1](./validation-rules/v0.1.md#datacatalogtracking-planssemantic-valid)

**Checks Performed:**
- Event references must exist in resource graph
- Property references must exist in resource graph
- Discriminator must reference a property in the rule's properties
- Nested properties only allowed for types supporting nesting (object, array with object items)
- `additionalProperties` only allowed for types that support it
- Tracking plan names must be globally unique in the project

---

## Event Stream

### event-stream/source/spec-syntax-valid

![Severity: Error](https://img.shields.io/badge/severity-error-red)

**Description:** event stream source spec syntax must be valid

**Applies to:** `source`

**Validation Phase:** Syntactic

**Available in:** [v1](./#event-streamsourcespec-syntax-valid) | [v0.1](./validation-rules/v0.1.md#event-streamsourcespec-syntax-valid)

**Checks Performed:**
- Source spec must conform to struct validation tags

---

### event-stream/source/semantic-valid

![Severity: Error](https://img.shields.io/badge/severity-error-red)

**Description:** event stream source references must resolve to existing resources

**Applies to:** `source`

**Validation Phase:** Semantic

**Available in:** [v1](./#event-streamsourcesemantic-valid) | [v0.1](./validation-rules/v0.1.md#event-streamsourcesemantic-valid)

**Checks Performed:**
- Tracking plan reference in governance section must exist in resource graph
- Reference must match pattern `#/tp/<group>/<id>` or `#tracking-plan:<id>`

---

## RETL

### retl/sqlmodel/spec-syntax-valid

![Severity: Error](https://img.shields.io/badge/severity-error-red)

**Description:** retl sql model spec syntax must be valid

**Applies to:** `sqlmodel`

**Validation Phase:** Syntactic

**Available in:** [v1](./#retlsqlmodelspec-syntax-valid) | [v0.1](./validation-rules/v0.1.md#retlsqlmodelspec-syntax-valid)

**Checks Performed:**
- SQL model spec must conform to struct validation tags

---

## Reference

### Supported Versions

| Version | Status |
|---------|--------|
| `rudder/v1` | Current (this document) |
| `rudder/0.1` | Legacy ([documentation](./validation-rules/v0.1.md)) |
| `rudder/v0.1` | Legacy (alias for 0.1) |

### Primitive Types

| Type | Description |
|------|-------------|
| `string` | Text values |
| `number` | Floating-point numbers |
| `integer` | Whole numbers |
| `boolean` | True/false values |
| `array` | Ordered lists |
| `object` | Key-value structures |
| `null` | Null/empty values |

### Config Keywords

| Type | Keywords |
|------|----------|
| `string` | `enum`, `minLength`, `maxLength`, `pattern`, `format` |
| `integer` | `enum`, `minimum`, `maximum` |
| `number` | `enum`, `minimum`, `maximum` |
| `array` | `itemTypes`, `minItems`, `maxItems` |
| `boolean` | `enum` |

### Format Values

`date-time`, `date`, `time`, `email`, `uuid`, `hostname`, `ipv4`, `ipv6`

### Event Types

`track`, `screen`, `identify`, `group`, `page`
