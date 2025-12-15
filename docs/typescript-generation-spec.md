# TypeScript Generation Specification

## Overview

This document defines the mapping rules for generating TypeScript code from RudderStack tracking plan YAML definitions. It serves as the specification for implementing TypeScript support in RudderTyper 2.0.

---

## Test Data Location

```
cli/internal/typer/plan/testdata/project/
├── tracking-plan.yaml    # Main tracking plan with event rules
├── events.yaml           # Event definitions
├── properties.yaml       # Property definitions
├── custom-types.yaml     # Custom type definitions
├── categories.yaml       # Category definitions
```

---

## Input Files Structure

### tracking-plan.yaml

Defines event rules linking events to properties.

### events.yaml

Defines events with type and name.

### properties.yaml

Defines properties with types and configurations.

### custom-types.yaml

Defines reusable custom types.

---

## Type Mapping

### Primitive Types

| YAML Type | TypeScript            |
| --------- | --------------------- |
| `string`  | `string`              |
| `integer` | `number`              |
| `number`  | `number`              |
| `boolean` | `boolean`             |
| `array`   | `any[]` or `T[]`      |
| `object`  | `Record<string, any>` |
| `null`    | `null`                |

### Multi-Type Properties

**Input (properties.yaml):**

```yaml
- id: "some-multi-type"
  name: "someMultiType"
  type: "string,integer,number,boolean,object,array,null"
```

**Output (index.ts):**

```typescript
someMultiType?: any[] | boolean | number | Record<string, any> | string | null;
```

---

## Naming Conventions

| Source             | Target         | Convention         | Example                                 |
| ------------------ | -------------- | ------------------ | --------------------------------------- |
| Event `name`       | Function name  | camelCase          | `"Some Track Event"` → `someTrackEvent` |
| Event `name`       | Interface name | PascalCase         | `"Some Track Event"` → `SomeTrackEvent` |
| Property `name`    | Property name  | As-is (camelCase)  | `someString` → `someString`             |
| Custom type `name` | Type name      | As-is (PascalCase) | `SomeStringType` → `SomeStringType`     |

---

## Events

**Input (events.yaml):**

```yaml
- id: "some_track_event"
  name: "Some Track Event"
  event_type: "track"
  description: "This is a track event for testing."
```

**Output (index.ts):**

```typescript
/**
 * This is a track event for testing.
 */
export function someTrackEvent(
  props: SomeTrackEvent,
  options?: ApiOptions,
  callback?: apiCallback
): void {
  const a = analytics();
  if (a) {
    a.track(
      "Some Track Event",
      props || {},
      withRudderTyperContext(options),
      callback
    );
  }
}
```

---

## Properties

### Basic Property

**Input (properties.yaml):**

```yaml
- id: "some-string"
  name: "someString"
  type: "string"
  description: "some string property"
```

**Usage in tracking-plan.yaml:**

```yaml
properties:
  - $ref: "#/properties/typer-test/some-string"
    required: true
```

**Output (index.ts):**

```typescript
export interface SomeTrackEvent {
  someString: string; // required - no ?
}
```

### Optional Property

**Input (tracking-plan.yaml):**

```yaml
properties:
  - $ref: "#/properties/typer-test/some-string"
    required: false # or omitted
```

**Output (index.ts):**

```typescript
export interface SomeTrackEvent {
  someString?: string; // optional - has ?
}
```

---

## Arrays

### Simple Array

**Input (properties.yaml):**

```yaml
- id: "some-array-of-strings"
  name: "someArrayOfStrings"
  type: "array"
  propConfig:
    itemTypes:
      - "string"
```

**Output (index.ts):**

```typescript
someArrayOfStrings?: string[];
```

### Array with Multiple Item Types

**Input (properties.yaml):**

```yaml
- id: "some-array-of-multiple-types"
  name: "someArrayOfMultipleTypes"
  type: "array"
  propConfig:
    itemTypes:
      - "string"
      - "integer"
      - "number"
      - "boolean"
```

**Output (index.ts):**

```typescript
someArrayOfMultipleTypes?: (boolean | number | string)[];
```

---

## Enums

### String Enum

**Input (properties.yaml):**

```yaml
- id: "some-string-with-enum"
  name: "someStringWithEnum"
  type: "string"
  propConfig:
    enum:
      - "GET"
      - "PUT"
      - "POST"
      - "DELETE"
      - "PATCH"
```

**Output (index.ts):**

```typescript
export enum Somestringwithenum_String {
  S_GET = "GET",
  S_PUT = "PUT",
  S_POST = "POST",
  S_DELETE = "DELETE",
  S_PATCH = "PATCH",
}
```

### Integer Enum

**Input (properties.yaml):**

```yaml
- id: "some-integer-with-enum"
  name: "someIntegerWithEnum"
  type: "integer"
  propConfig:
    enum:
      - 200
      - 201
      - 400
      - 500
```

**Output (index.ts):**

```typescript
export enum Someintegerwithenum_Integer {
  N_200 = 200,
  N_201 = 201,
  N_400 = 400,
  N_500 = 500,
}
```

---

## Custom Types

### Primitive Custom Type

**Input (custom-types.yaml):**

```yaml
- id: "some-string-type"
  name: "SomeStringType"
  type: "string"
  description: "some string custom type"
```

**Output (index.ts):**

```typescript
export interface CustomTypeDefs {
  SomeStringType?: string;
}
```

### Object Custom Type

**Input (custom-types.yaml):**

```yaml
- id: "some-object-type"
  name: "SomeObjectType"
  type: "object"
  description: "some object custom type"
  properties:
    - $ref: "#/properties/typer-test/some-custom-string"
      required: true
    - $ref: "#/properties/typer-test/some-integer"
      required: false
```

**Output (index.ts):**

```typescript
export interface CustomTypeDefsSomeObjectType {
  someCustomString: CustomTypeDefs["SomeStringType"]; // required
  someInteger?: number; // optional
}

export interface CustomTypeDefs {
  SomeObjectType?: CustomTypeDefsSomeObjectType;
}
```

### Referencing Custom Types in Properties

**Input (properties.yaml):**

```yaml
- id: "some-custom-string"
  name: "someCustomString"
  type: "#/custom-types/typer-test/some-string-type"
```

**Output (index.ts):**

```typescript
someCustomString?: CustomTypeDefs['SomeStringType'];
```

---

## Nested Objects

**Input (tracking-plan.yaml):**

```yaml
properties:
  - $ref: "#/properties/typer-test/some-nested-object"
    properties:
      - $ref: "#/properties/typer-test/some-nested-level-1"
        properties:
          - $ref: "#/properties/typer-test/some-nested-level-2"
            properties:
              - $ref: "#/properties/typer-test/some-string"
              - $ref: "#/properties/typer-test/some-integer"
```

**Output (index.ts):**

```typescript
export interface SomeTrackEventSomeNestedLevel2 {
  someString?: string;
  someInteger?: number;
}

export interface SomeTrackEventSomeNestedLevel1 {
  someNestedLevel2?: SomeTrackEventSomeNestedLevel2;
}

export interface SomeTrackEventSomeNestedObject {
  someNestedLevel1?: SomeTrackEventSomeNestedLevel1;
}
```

**Naming Pattern:** `{EventName}{PropertyPath}`

---

## Variants (Discriminated Unions)

TypeScript discriminated unions map directly from Kotlin sealed classes. Each variant case becomes an interface with a literal type for the discriminator property.

### Kotlin → TypeScript Mapping

| Kotlin Concept | TypeScript Equivalent |
|----------------|----------------------|
| Sealed class | Union type (`type T = A \| B \| C`) |
| Sealed subclass | Interface with literal discriminator |
| Abstract discriminator property | Discriminator with literal type per case |
| Default subclass | Interface with `string` discriminator type |

### Custom Type with Variants

**Input (custom-types.yaml):**

```yaml
- id: "some-variant-type"
  name: "SomeVariantType"
  type: "object"
  properties:
    - $ref: "#/properties/typer-test/some-string"
      required: true
    - $ref: "#/properties/typer-test/some-integer"
      required: false
  variants:
    - type: "discriminator"
      discriminator: "#/properties/typer-test/some-string"
      cases:
        - display_name: "Case 1"
          match:
            - "case_1"
          properties:
            - $ref: "#/properties/typer-test/some-integer"
              required: true
        - display_name: "Case 2"
          match:
            - "case_2_a"
            - "case_2_b"
          properties:
            - $ref: "#/properties/typer-test/some-number"
              required: true
      default:
        - $ref: "#/properties/typer-test/some-boolean"
          required: false
```

**Output (index.ts):**

```typescript
// Case 1: someString === "case_1"
export interface SomeVariantTypeCase1 {
  someString: "case_1";           // literal type
  someInteger: number;            // required in this case
}

// Case 2: someString === "case_2_a" or "case_2_b"
export interface SomeVariantTypeCase2A {
  someString: "case_2_a";         // literal type
  someInteger?: number;           // optional (from base)
  someNumber: number;             // required in this case
}

export interface SomeVariantTypeCase2B {
  someString: "case_2_b";         // literal type
  someInteger?: number;           // optional (from base)
  someNumber: number;             // required in this case
}

// Default case: any other someString value
export interface SomeVariantTypeDefault {
  someString: string;             // any string (catches all other values)
  someInteger?: number;           // optional (from base)
  someBoolean?: boolean;          // from default schema
}

// Union type combining all cases
export type SomeVariantType =
  | SomeVariantTypeCase1
  | SomeVariantTypeCase2A
  | SomeVariantTypeCase2B
  | SomeVariantTypeDefault;

// Reference in CustomTypeDefs
export interface CustomTypeDefs {
  SomeVariantType?: SomeVariantType;
}
```

### Event Rule with Variants

**Input (tracking-plan.yaml):**

```yaml
- type: "event_rule"
  id: "device_event_rule"
  event:
    $ref: "#/events/typer-test/device_event"
  properties:
    - $ref: "#/properties/typer-test/device-type"
      required: true
  variants:
    - type: "discriminator"
      discriminator: "#/properties/typer-test/device-type"
      cases:
        - display_name: "Mobile"
          match:
            - "mobile"
          properties:
            - $ref: "#/properties/typer-test/os-version"
              required: true
            - $ref: "#/properties/typer-test/app-version"
              required: true
        - display_name: "Desktop"
          match:
            - "desktop"
          properties:
            - $ref: "#/properties/typer-test/browser"
              required: true
```

**Output (index.ts):**

```typescript
// Mobile device case
export interface DeviceEventMobile {
  deviceType: "mobile";
  osVersion: string;
  appVersion: string;
}

// Desktop device case
export interface DeviceEventDesktop {
  deviceType: "desktop";
  browser: string;
}

// Default case
export interface DeviceEventDefault {
  deviceType: string;
}

// Union type
export type DeviceEvent =
  | DeviceEventMobile
  | DeviceEventDesktop
  | DeviceEventDefault;

// Function uses union type
export function deviceEvent(
  props: DeviceEvent,
  options?: ApiOptions,
  callback?: apiCallback
): void {
  // ...
}
```

### Naming Convention for Variant Interfaces

| Component | Pattern | Example |
|-----------|---------|---------|
| Case interface | `{TypeName}{CaseName}` | `SomeVariantTypeCase1` |
| Multi-match interface | `{TypeName}{MatchValue}` | `SomeVariantTypeCase2A`, `SomeVariantTypeCase2B` |
| Default interface | `{TypeName}Default` | `SomeVariantTypeDefault` |
| Union type | `{TypeName}` | `SomeVariantType` |

### Type Narrowing Usage

```typescript
function handleVariant(data: SomeVariantType) {
  // TypeScript narrows type based on discriminator
  if (data.someString === "case_1") {
    // data is SomeVariantTypeCase1
    console.log(data.someInteger); // number (required)
  } else if (data.someString === "case_2_a" || data.someString === "case_2_b") {
    // data is SomeVariantTypeCase2A | SomeVariantTypeCase2B
    console.log(data.someNumber);  // number (required)
  } else {
    // data is SomeVariantTypeDefault
    console.log(data.someBoolean); // boolean | undefined
  }
}
```

### Enum Discriminator

When the discriminator property has an enum type:

**Input:**

```yaml
variants:
  - type: "discriminator"
    discriminator: "#/properties/typer-test/status-enum"  # enum: ["pending", "active", "completed"]
    cases:
      - match: ["pending"]
        properties: [...]
      - match: ["active"]
        properties: [...]
```

**Output:**

```typescript
export enum Status_String {
  S_PENDING = "pending",
  S_ACTIVE = "active",
  S_COMPLETED = "completed",
}

export interface TaskPending {
  status: Status_String.S_PENDING;  // enum literal
  // ... pending-specific properties
}

export interface TaskActive {
  status: Status_String.S_ACTIVE;   // enum literal
  // ... active-specific properties
}

export interface TaskDefault {
  status: Status_String;            // full enum type for default
  // ... default properties
}

export type Task = TaskPending | TaskActive | TaskDefault;
```

---

## Event Rules

### Track Event with Properties

**Input (tracking-plan.yaml):**

```yaml
- type: "event_rule"
  id: "some_track_event_rule"
  event:
    $ref: "#/events/typer-test/some_track_event"
    allow_unplanned: false
  properties:
    - $ref: "#/properties/typer-test/some-string"
      required: true
    - $ref: "#/properties/typer-test/some-integer"
      required: true
    - $ref: "#/properties/typer-test/some-boolean"
      required: false
```

**Output (index.ts):**

```typescript
export interface SomeTrackEvent {
  someString: string; // required
  someInteger: number; // required
  someBoolean?: boolean; // optional
}

export function someTrackEvent(
  props: SomeTrackEvent,
  options?: ApiOptions,
  callback?: apiCallback
): void {
  // ...
}
```

### Empty Track Event

**Input (tracking-plan.yaml):**

```yaml
- type: "event_rule"
  id: "some_empty_track_event_rule"
  event:
    $ref: "#/events/typer-test/some_empty_track_event"
    allow_unplanned: false
```

**Output (index.ts):**

```typescript
export function someEmptyTrackEvent(
  props?: Record<string, any>,
  options?: ApiOptions,
  callback?: apiCallback
): void {
  // ...
}
```

### Event with Additional Properties Allowed

**Input (tracking-plan.yaml):**

```yaml
- type: "event_rule"
  event:
    $ref: "#/events/typer-test/some_empty_track_event_with_additional_properties"
    allow_unplanned: true # allows additional properties
```

**Output (index.ts):**

```typescript
export function someEmptyTrackEventWithAdditionalProperties(
  props?: Record<string, any>, // generic - accepts any properties
  options?: ApiOptions,
  callback?: apiCallback
): void {
  // ...
}
```

---

## Generated File Structure

```typescript
// 1. Auto-generated header
/**
 * This client was automatically generated by RudderTyper. ** Do Not Edit **
 */

// 2. Imports
import type { RudderAnalytics, ApiOptions, ApiObject } from '@rudderstack/analytics-js';

// 3. Global declaration
declare global {
  interface Window {
    rudderanalytics: RudderAnalytics | undefined;
  }
}

// 4. Callback type
type apiCallback = (data?: any) => void;

// 5. Enums
export enum Somestringwithenum_String { ... }
export enum Someintegerwithenum_Integer { ... }

// 6. Nested interfaces (deepest first)
export interface SomeTrackEventSomeNestedLevel2 { ... }
export interface SomeTrackEventSomeNestedLevel1 { ... }

// 7. Event interfaces
export interface SomeTrackEvent { ... }

// 8. Custom type interfaces
export interface CustomTypeDefsSomeObjectType { ... }
export interface CustomTypeDefs { ... }

// 9. Configuration
export interface RudderTyperOptions { ... }
export function setRudderTyperOptions(options: RudderTyperOptions) { ... }

// 10. Context helper
function withRudderTyperContext(message: ApiOptions = {}): ApiOptions { ... }

// 11. Track functions
export function someTrackEvent(...) { ... }
export function someEmptyTrackEvent(...) { ... }

// 12. Client API
const clientAPI = { setRudderTyperOptions, someTrackEvent, ... };

// 13. Proxy export
export const RudderTyperAnalytics = new Proxy<typeof clientAPI>(clientAPI, { ... });
```

---

## RudderTyper Context

Every analytics call includes metadata for attribution.

```typescript
function withRudderTyperContext(message: ApiOptions = {}): ApiOptions {
  return {
    ...message,
    context: {
      ...(message.context || {}),
      ruddertyper: {
        sdk: "analytics.js",
        language: "typescript",
        rudderTyperVersion: "2.0.0",
        trackingPlanId: "tp_xxxxx",
        trackingPlanVersion: 1,
      },
    },
  };
}
```

---

## Edge Cases

### Other Event Types (Identify, Page, Group)

**Identify Event:**

```yaml
# events.yaml
- id: "identify_event"
  event_type: "identify"
```

**Expected Output:**

```typescript
export interface IdentifyTraits {
  userName?: string;
  email?: string;
}

export function identify(
  userId: string,
  traits?: IdentifyTraits,
  options?: ApiOptions,
  callback?: apiCallback,
): void { ... }
```

**Page Event:**

> **Note:** The JavaScript SDK uses `page()` for page view tracking. The `screen()` method is not supported in the JS SDK - it is only available in mobile SDKs (Kotlin, Swift, etc.) for screen view tracking.

```typescript
export function page(
  category?: string,
  name?: string,
  properties?: PageProperties,
  options?: ApiOptions,
  callback?: apiCallback,
): void { ... }
```

**Group Event:**

```typescript
export function group(
  groupId: string,
  traits?: GroupTraits,
  options?: ApiOptions,
  callback?: apiCallback,
): void { ... }
```

---

### Special Characters in Names

**Event name with quotes:**

```yaml
- name: 'Product "Premium" Clicked'
```

```typescript
// Function name sanitized, original preserved in track call
export function productPremiumClicked(...): void {
  a.track('Product "Premium" Clicked', ...);
}
```

**Event name with dollar sign:**

```yaml
- name: "$Variable$String"
```

```typescript
export function variableString(...): void {
  a.track('$Variable$String', ...);
}
```

---

### Name Collision Handling

When two events sanitize to the same function name:

```yaml
- name: "eventWithNameCamelCase"
- name: "$eventWithNameCamelCase$!"
```

```typescript
export function eventWithNameCamelCase(...): void { ... }
export function eventWithNameCamelCase2(...): void { ... }  // Suffix added
```

---

### Reserved Words

Property names that are TypeScript reserved words:

```yaml
- name: "class"
  type: "string"
```

```typescript
export interface SomeEvent {
  _class?: string; // Prefixed with underscore
}
```

**Reserved Words List:**
`break`, `case`, `catch`, `class`, `const`, `continue`, `default`, `delete`, `do`, `else`, `enum`, `export`, `extends`, `false`, `finally`, `for`, `function`, `if`, `import`, `in`, `instanceof`, `new`, `null`, `return`, `super`, `switch`, `this`, `throw`, `true`, `try`, `typeof`, `var`, `void`, `while`, `with`, `let`, `static`, `yield`, `any`, `boolean`, `number`, `string`, `symbol`, `type`, `async`, `await`

---

### Null Type

```yaml
- name: "someNull"
  type: "null"
```

```typescript
someNull?: null;
```

---

### Empty Object Types

**With additional properties allowed:**

```typescript
someObject?: Record<string, any>;
```

**Without additional properties:**

```typescript
someObject?: Record<string, never>;  // Or {}
```

---

### Array of Custom Type Objects

```yaml
- name: "items"
  type: "array"
  propConfig:
    itemTypes:
      - "#/custom-types/typer-test/some-object-type"
```

```typescript
items?: CustomTypeDefs['SomeObjectType'][];
```

---

### Deeply Nested Objects (3+ levels)

Each level gets its own interface:

```typescript
export interface EventLevel3 {
  value?: string;
}
export interface EventLevel2 {
  level3?: EventLevel3;
}
export interface EventLevel1 {
  level2?: EventLevel2;
}
export interface EventNested {
  level1?: EventLevel1;
}
```

---

### Same Name in Different Namespaces

Event and property with same name:

```yaml
# Event named "User"
- name: "User"
  event_type: "track"

# Property named "user"
- name: "user"
  type: "string"
```

```typescript
// No collision - different namespaces
export interface User {
  user?: string;
}
export function user(...): void { ... }
```

---

## Function Overloads

For non-track events, generate multiple function signatures:

### Identify

```typescript
// With userId
identify(userId: string, traits?: IdentifyTraits, options?: ApiOptions, callback?: apiCallback): void;
// Without userId (anonymous)
identify(traits?: IdentifyTraits, options?: ApiOptions, callback?: apiCallback): void;
```

### Page

> **Note:** The JS SDK does not support `screen()`. Use `page()` for page view tracking.

```typescript
// Full signature
page(category: string, name: string, properties?: PageProperties, options?: ApiOptions, callback?: apiCallback): void;
// Without category
page(name: string, properties?: PageProperties, options?: ApiOptions, callback?: apiCallback): void;
// Properties only
page(properties?: PageProperties, options?: ApiOptions, callback?: apiCallback): void;
```

### Group

```typescript
// With groupId
group(groupId: string, traits?: GroupTraits, options?: ApiOptions, callback?: apiCallback): void;
// Without groupId
group(traits?: GroupTraits, options?: ApiOptions, callback?: apiCallback): void;
```

---

## Validation Checklist

- [ ] Primitive types map correctly
- [ ] Required properties have no `?` modifier
- [ ] Optional properties have `?` modifier
- [ ] Enums generate with correct prefix (`S_` for strings, `N_` for numbers)
- [ ] Custom types reference via `CustomTypeDefs['TypeName']`
- [ ] Nested objects generate separate interfaces with hierarchical names
- [ ] Function names are camelCase
- [ ] Interface names are PascalCase
- [ ] Event names preserved exactly in track calls
- [ ] RudderTyper context included in all calls
- [ ] `allow_unplanned: true` results in `Record<string, any>`
- [ ] Identify/Page/Group events handled (Note: Screen is not supported in JS SDK)
- [ ] Function overloads for non-track events
- [ ] Special characters in names sanitized
- [ ] Name collisions resolved with numeric suffix
- [ ] Reserved words prefixed with underscore
- [ ] Null type supported
- [ ] Deeply nested objects work correctly
- [ ] Variants generate discriminated unions (not flat interfaces)
- [ ] Variant cases use literal types for discriminator
- [ ] Variant default case uses base type for discriminator
- [ ] Enum discriminators use enum literal types
