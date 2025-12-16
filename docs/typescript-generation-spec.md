# TypeScript Generation Specification

## Overview

This document defines the mapping rules for generating TypeScript code from RudderStack tracking plan YAML definitions. It serves as the specification for implementing TypeScript support in RudderTyper 2.0.

This specification follows the same conventions as the Kotlin generator for consistency across platforms.

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
type PropertySomeMultiType = any[] | boolean | number | Record<string, any> | string | null;
```

---

## Naming Conventions

| Source             | Target         | Convention         | Example                                              |
| ------------------ | -------------- | ------------------ | ---------------------------------------------------- |
| Event `name`       | Function name  | camelCase + prefix | `"Some Track Event"` → `trackSomeTrackEvent`         |
| Event `name`       | Interface name | PascalCase + prefix| `"Some Track Event"` → `TrackSomeTrackEventProperties` |
| Property `name`    | Type alias     | Property prefix    | `someString` → `PropertySomeString`                  |
| Custom type `name` | Type alias     | CustomType prefix  | `SomeStringType` → `CustomTypeSomeStringType`        |

---

## Property Types

For every property, generate a type alias. This enables reuse and consistent typing across the codebase.

**Input (properties.yaml):**

```yaml
- id: "some-string"
  name: "someString"
  type: "string"
  description: "some string property"
```

**Output (index.ts):**

```typescript
/** some string property */
type PropertySomeString = string;
```

**Usage in event properties:**

```typescript
interface TrackSomeTrackEventProperties {
  someString?: PropertySomeString;
}
```

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
trackSomeTrackEvent(
  props: TrackSomeTrackEventProperties,
  options?: ApiOptions,
  callback?: apiCallback
): void {
  this.analytics.track(
    'Some Track Event',
    props,
    this.withRudderTyperContext(options),
    callback
  );
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
/** some string property */
type PropertySomeString = string;

interface TrackSomeTrackEventProperties {
  someString: PropertySomeString; // required - no ?
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
interface TrackSomeTrackEventProperties {
  someString?: PropertySomeString; // optional - has ?
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
type PropertySomeArrayOfStrings = string[];
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
type PropertySomeArrayOfMultipleTypes = (boolean | number | string)[];
```

---

## Enums (Union Types)

Instead of using TypeScript enums with `S_` and `N_` prefixes, use union types that list the exact values. This provides better type safety and cleaner code.

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
type PropertySomeStringWithEnum = 'GET' | 'PUT' | 'POST' | 'DELETE' | 'PATCH';
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
type PropertySomeIntegerWithEnum = 200 | 201 | 400 | 500;
```

### Mixed Type Enum

When an enum contains both strings and numbers:

```typescript
type PropertyMixedEnum = 'GET' | 'POST' | 200 | 404;
```

---

## Custom Types

Custom types use a `CustomType` prefix and are defined as type aliases (not wrapped in an interface).

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
/** some string custom type */
type CustomTypeSomeStringType = string;
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
/** some object custom type */
interface CustomTypeSomeObjectType {
  someCustomString: PropertySomeCustomString; // required
  someInteger?: PropertySomeInteger;          // optional
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
type PropertySomeCustomString = CustomTypeSomeStringType;
```

---

## Nested Objects

Use inline nested object types instead of separate interfaces. This keeps the type definition local to the event rule and avoids name conflicts.

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
interface TrackSomeTrackEventProperties {
  someNestedObject?: {
    someNestedLevel1?: {
      someNestedLevel2?: {
        someString?: PropertySomeString;
        someInteger?: PropertySomeInteger;
      };
    };
  };
}
```

---

## Variants (Discriminated Unions)

TypeScript discriminated unions map directly from Kotlin sealed classes. Each variant case becomes an interface with a literal type for the discriminator property.

### Kotlin → TypeScript Mapping

| Kotlin Concept | TypeScript Equivalent |
|----------------|----------------------|
| Sealed class | Union type (`type T = A \| B \| C`) |
| Sealed subclass | Interface with literal discriminator |
| Abstract discriminator property | Discriminator with literal type per case |
| Default subclass | Interface with `Exclude<>` discriminator type |

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
interface CustomTypeSomeVariantTypeCase1 {
  someString: 'case_1';           // literal type
  someInteger: PropertySomeInteger; // required in this case
}

// Case 2: someString === "case_2_a" or "case_2_b"
interface CustomTypeSomeVariantTypeCase2A {
  someString: 'case_2_a';             // literal type
  someInteger?: PropertySomeInteger;  // optional (from base)
  someNumber: PropertySomeNumber;     // required in this case
}

interface CustomTypeSomeVariantTypeCase2B {
  someString: 'case_2_b';             // literal type
  someInteger?: PropertySomeInteger;  // optional (from base)
  someNumber: PropertySomeNumber;     // required in this case
}

// Default case: any string except the defined cases
interface CustomTypeSomeVariantTypeDefault {
  someString: Exclude<string, 'case_1' | 'case_2_a' | 'case_2_b'>; // excludes defined cases
  someInteger?: PropertySomeInteger;  // optional (from base)
  someBoolean?: PropertySomeBoolean;  // from default schema
}

// Union type combining all cases
type CustomTypeSomeVariantType =
  | CustomTypeSomeVariantTypeCase1
  | CustomTypeSomeVariantTypeCase2A
  | CustomTypeSomeVariantTypeCase2B
  | CustomTypeSomeVariantTypeDefault;
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
interface TrackDeviceEventPropertiesMobile {
  deviceType: 'mobile';
  osVersion: PropertyOsVersion;
  appVersion: PropertyAppVersion;
}

// Desktop device case
interface TrackDeviceEventPropertiesDesktop {
  deviceType: 'desktop';
  browser: PropertyBrowser;
}

// Default case
interface TrackDeviceEventPropertiesDefault {
  deviceType: Exclude<string, 'mobile' | 'desktop'>;
}

// Union type
type TrackDeviceEventProperties =
  | TrackDeviceEventPropertiesMobile
  | TrackDeviceEventPropertiesDesktop
  | TrackDeviceEventPropertiesDefault;

// Method uses union type
trackDeviceEvent(
  props: TrackDeviceEventProperties,
  options?: ApiOptions,
  callback?: apiCallback
): void {
  this.analytics.track('Device Event', props, this.withRudderTyperContext(options), callback);
}
```

### Naming Convention for Variant Interfaces

| Component | Pattern | Example |
|-----------|---------|---------|
| Case interface | `{TypeName}{CaseName}` | `CustomTypeSomeVariantTypeCase1` |
| Multi-match interface | `{TypeName}{MatchValue}` | `CustomTypeSomeVariantTypeCase2A` |
| Default interface | `{TypeName}Default` | `CustomTypeSomeVariantTypeDefault` |
| Union type | `{TypeName}` | `CustomTypeSomeVariantType` |

### Type Narrowing Usage

```typescript
function handleVariant(data: CustomTypeSomeVariantType) {
  // TypeScript narrows type based on discriminator
  if (data.someString === 'case_1') {
    // data is CustomTypeSomeVariantTypeCase1
    console.log(data.someInteger); // number (required)
  } else if (data.someString === 'case_2_a' || data.someString === 'case_2_b') {
    // data is CustomTypeSomeVariantTypeCase2A | CustomTypeSomeVariantTypeCase2B
    console.log(data.someNumber);  // number (required)
  } else {
    // data is CustomTypeSomeVariantTypeDefault
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
type PropertyStatusEnum = 'pending' | 'active' | 'completed';

interface TrackTaskEventPropertiesPending {
  status: 'pending';  // literal from union
  // ... pending-specific properties
}

interface TrackTaskEventPropertiesActive {
  status: 'active';   // literal from union
  // ... active-specific properties
}

interface TrackTaskEventPropertiesDefault {
  status: Exclude<PropertyStatusEnum, 'pending' | 'active'>;  // 'completed' only
  // ... default properties
}

type TrackTaskEventProperties =
  | TrackTaskEventPropertiesPending
  | TrackTaskEventPropertiesActive
  | TrackTaskEventPropertiesDefault;
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
interface TrackSomeTrackEventProperties {
  someString: PropertySomeString;   // required
  someInteger: PropertySomeInteger; // required
  someBoolean?: PropertySomeBoolean; // optional
}

// Method in RudderTyperAnalytics class
trackSomeTrackEvent(
  props: TrackSomeTrackEventProperties,
  options?: ApiOptions,
  callback?: apiCallback
): void {
  this.analytics.track('Some Track Event', props, this.withRudderTyperContext(options), callback);
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
// Method in RudderTyperAnalytics class
trackSomeEmptyTrackEvent(
  options?: ApiOptions,
  callback?: apiCallback
): void {
  this.analytics.track('Some Empty Track Event', {}, this.withRudderTyperContext(options), callback);
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
type TrackSomeEmptyTrackEventWithAdditionalPropertiesProperties = Record<string, any>;

// Method in RudderTyperAnalytics class
trackSomeEmptyTrackEventWithAdditionalProperties(
  props?: TrackSomeEmptyTrackEventWithAdditionalPropertiesProperties,
  options?: ApiOptions,
  callback?: apiCallback
): void {
  this.analytics.track('Some Empty Track Event With Additional Properties', props || {}, this.withRudderTyperContext(options), callback);
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

// 3. Callback type
type apiCallback = (data?: any) => void;

// 4. Custom Types (type aliases)
/** some string custom type */
type CustomTypeSomeStringType = string;

/** some object custom type */
interface CustomTypeSomeObjectType {
  someCustomString: PropertySomeCustomString;
  someInteger?: PropertySomeInteger;
}

// 5. Property Types (type aliases)
/** some string property */
type PropertySomeString = string;

/** some integer property */
type PropertySomeInteger = number;

// 6. Event Properties Interfaces
interface TrackSomeTrackEventProperties {
  someString: PropertySomeString;
  someNestedObject?: {
    someNestedLevel1?: {
      someNestedLevel2?: {
        someString?: PropertySomeString;
        someInteger?: PropertySomeInteger;
      };
    };
  };
}

interface IdentifyTraits {
  userName?: PropertyUserName;
  email?: PropertyEmail;
}

// 7. Configuration
export interface RudderTyperOptions {
  analytics?: RudderAnalytics;
  onViolation?: ViolationHandler;
}

// 8. RudderTyper Analytics Class
export class RudderTyperAnalytics {
  private analytics: RudderAnalytics;

  constructor(analytics: RudderAnalytics) {
    this.analytics = analytics;
  }

  // 9. Track functions
  trackSomeTrackEvent(
    props: TrackSomeTrackEventProperties,
    options?: ApiOptions,
    callback?: apiCallback
  ): void {
    this.analytics.track(
      'Some Track Event',
      props,
      this.withRudderTyperContext(options),
      callback
    );
  }

  // 10. Identify function
  identify(
    userId: string,
    traits?: IdentifyTraits,
    options?: ApiOptions,
    callback?: apiCallback
  ): void {
    this.analytics.identify(userId, traits, this.withRudderTyperContext(options), callback);
  }

  // 11. Context helper (private)
  private withRudderTyperContext(message: ApiOptions = {}): ApiOptions {
    return {
      ...message,
      context: {
        ...(message.context || {}),
        ruddertyper: {
          sdk: 'analytics.js',
          language: 'typescript',
          rudderTyperVersion: '2.0.0',
          trackingPlanId: 'tp_xxxxx',
          trackingPlanVersion: 1,
        },
      },
    };
  }
}
```

---

## RudderTyper Context

Every analytics call includes metadata for attribution.

```typescript
private withRudderTyperContext(message: ApiOptions = {}): ApiOptions {
  return {
    ...message,
    context: {
      ...(message.context || {}),
      ruddertyper: {
        sdk: 'analytics.js',
        language: 'typescript',
        rudderTyperVersion: '2.0.0',
        trackingPlanId: 'tp_xxxxx',
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
interface IdentifyTraits {
  userName?: PropertyUserName;
  email?: PropertyEmail;
}

identify(
  userId: string,
  traits?: IdentifyTraits,
  options?: ApiOptions,
  callback?: apiCallback,
): void { ... }
```

**Page Event:**

> **Note:** The JavaScript SDK uses `page()` for page view tracking. The `screen()` method is not supported in the JS SDK - it is only available in mobile SDKs (Kotlin, Swift, etc.) for screen view tracking.

```typescript
interface PageProperties {
  pageType?: PropertyPageType;
}

page(
  category?: string,
  name?: string,
  properties?: PageProperties,
  options?: ApiOptions,
  callback?: apiCallback,
): void { ... }
```

**Group Event:**

```typescript
interface GroupTraits {
  groupName?: PropertyGroupName;
}

group(
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
// Method name sanitized, original preserved in track call
trackProductPremiumClicked(props: TrackProductPremiumClickedProperties, ...): void {
  this.analytics.track('Product "Premium" Clicked', props, ...);
}
```

**Event name with dollar sign:**

```yaml
- name: "$Variable$String"
```

```typescript
trackVariableString(props: TrackVariableStringProperties, ...): void {
  this.analytics.track('$Variable$String', props, ...);
}
```

---

### Name Collision Handling

When two events sanitize to the same method name:

```yaml
- name: "eventWithNameCamelCase"
- name: "$eventWithNameCamelCase$!"
```

```typescript
trackEventWithNameCamelCase(props: TrackEventWithNameCamelCaseProperties, ...): void { ... }
trackEventWithNameCamelCase1(props: TrackEventWithNameCamelCaseProperties1, ...): void { ... }  // Suffix added
```

---

### Reserved Words

Property names that are TypeScript reserved words:

```yaml
- name: "class"
  type: "string"
```

```typescript
interface TrackSomeEventProperties {
  _class?: PropertyClass; // Prefixed with underscore
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
type PropertySomeNull = null;
```

---

### Empty Object Types

**With additional properties allowed:**

```typescript
type PropertySomeObject = Record<string, any>;
```

**Without additional properties:**

```typescript
type PropertySomeObject = Record<string, never>;  // Or {}
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
type PropertyItems = CustomTypeSomeObjectType[];
```

---

### Deeply Nested Objects (3+ levels)

Use inline types for all nesting levels:

```typescript
interface TrackEventProperties {
  level1?: {
    level2?: {
      level3?: {
        value?: PropertyValue;
      };
    };
  };
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
// No collision - different namespaces with prefixes
type PropertyUser = string;

interface TrackUserProperties {
  user?: PropertyUser;
}

// Method in RudderTyperAnalytics class
trackUser(props: TrackUserProperties, ...): void { ... }
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

- [ ] Class-based approach: `RudderTyperAnalytics` class with constructor accepting `RudderAnalytics` (named differently from SDK to avoid confusion)
- [ ] No global declaration (`declare global`) - use dependency injection via constructor
- [ ] Primitive types map correctly
- [ ] Required properties have no `?` modifier
- [ ] Optional properties have `?` modifier
- [ ] Enums generate as union types (not TypeScript enums with `S_`/`N_` prefixes)
- [ ] Custom types use `CustomType` prefix as type aliases
- [ ] Property types use `Property` prefix as type aliases
- [ ] Nested objects use inline types (not separate interfaces)
- [ ] Method names are camelCase with event type prefix (e.g., `trackSomeEvent`)
- [ ] Interface names are PascalCase with prefix (e.g., `TrackSomeEventProperties`)
- [ ] Event names preserved exactly in track calls
- [ ] RudderTyper context included in all calls via `this.withRudderTyperContext()`
- [ ] `allow_unplanned: true` results in `Record<string, any>`
- [ ] Identify/Page/Group events handled (Note: Screen is not supported in JS SDK)
- [ ] Method overloads for non-track events
- [ ] Special characters in names sanitized
- [ ] Name collisions resolved with numeric suffix
- [ ] Reserved words prefixed with underscore
- [ ] Null type supported
- [ ] Deeply nested objects use inline types
- [ ] Variants generate discriminated unions
- [ ] Variant cases use literal types for discriminator
- [ ] Variant default case uses `Exclude<>` type for discriminator
