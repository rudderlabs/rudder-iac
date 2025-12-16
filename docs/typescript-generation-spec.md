# TypeScript Generation Specification

## Overview

This document defines the mapping rules for generating TypeScript code from RudderStack tracking plan YAML definitions. It serves as the specification for implementing TypeScript support in RudderTyper 2.0.

This specification follows the same conventions as the Kotlin generator for consistency across platforms.

---

## Table of Contents

- [Quick Reference](#quick-reference)
- [Test Data Location](#test-data-location)
- [Input Files Structure](#input-files-structure)
- [Type Mapping](#type-mapping)
- [Naming Conventions](#naming-conventions)
- [Property Types](#property-types)
- [Events](#events)
- [Properties](#properties)
- [Arrays](#arrays)
- [Enums (Union Types)](#enums-union-types)
- [Custom Types](#custom-types)
- [Nested Objects](#nested-objects)
- [Variants (Discriminated Unions)](#variants-discriminated-unions)
- [Event Rules](#event-rules)
- [Generated File Structure](#generated-file-structure)
- [RudderTyper Context](#ruddertyper-context)
- [Event Types](#event-types)
  - [Track Events](#track-events)
  - [Identify Events](#identify-events)
  - [Page Events](#page-events)
  - [Group Events](#group-events)
  - [Alias Events](#alias-events)
  - [Screen Events (Mobile Only)](#screen-events-mobile-only)
- [Edge Cases](#edge-cases)
- [Function Overloads](#function-overloads)
- [Validation Checklist](#validation-checklist)

---

## Quick Reference

### Architecture
| Rule | Example |
|------|---------|
| Class-based, not global | `export class RudderTyperAnalytics { constructor(analytics: RudderAnalytics) }` |
| Dependency injection | Pass `RudderAnalytics` instance to constructor |
| Context helper | `this.withRudderTyperContext(options)` on all calls |

### Naming Conventions
| Source | Pattern | Example |
|--------|---------|---------|
| Track event | `track{EventName}` | `trackUserSignedUp` |
| Page event | `page{EventName}` | `pageProductViewed` |
| Identify event | `identify` | `identify` |
| Group event | `group` | `group` |
| Alias event | `alias` | `alias` |
| Track properties interface | `Track{EventName}Properties` | `TrackUserSignedUpProperties` |
| Page properties interface | `Page{EventName}Properties` | `PageProductViewedProperties` |
| Identify traits interface | `Identify{EventName}Traits` | `IdentifyUserTraits` |
| Group traits interface | `Group{EventName}Traits` | `GroupCompanyTraits` |
| Property type alias | `Property{PropertyName}` | `PropertySomeString` |
| Custom type alias | `CustomType{TypeName}` | `CustomTypeSomeStringType` |

### Type Mappings
| YAML | TypeScript |
|------|------------|
| `string` | `string` |
| `integer` | `number` |
| `number` | `number` |
| `boolean` | `boolean` |
| `array` | `any[]` or `T[]` |
| `object` | `Record<string, any>` |
| `null` | `null` |

### Key Rules
| Rule | Do | Don't |
|------|-----|-------|
| Enums | `'GET' \| 'POST'` (union types) | `enum { S_GET = 'GET' }` |
| Nested objects | Inline types | Separate interfaces |
| Variant default | `Exclude<string, 'a' \| 'b'>` | Just `string` |
| Required props | `propName: Type` | `propName?: Type` |
| Optional props | `propName?: Type` | `propName: Type` |
| Function overloads | Support `(props, callback)` pattern | Only `(props, options, callback)` |

### Event Type Support (JS SDK)
| Event Type | Supported | Notes |
|------------|-----------|-------|
| `track` | ✅ | `track(event, properties, options, callback)` |
| `identify` | ✅ | `identify(userId, traits, options, callback)` |
| `page` | ✅ | `page(category, name, properties, options, callback)` |
| `group` | ✅ | `group(groupId, traits, options, callback)` |
| `alias` | ✅ | `alias(to, from, options, callback)` |
| `screen` | ❌ | Mobile SDK only - use `page` for web |

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

### Event Type Prefixes

| Event Type | Method Prefix | Interface Prefix | Example Method | Example Interface |
|------------|---------------|------------------|----------------|-------------------|
| `track`    | `track`       | `Track`          | `trackUserSignedUp` | `TrackUserSignedUpProperties` |
| `identify` | `identify`    | `Identify`       | `identify` | `IdentifyUserTraits` |
| `page`     | `page`        | `Page`           | `pageProductViewed` | `PageProductViewedProperties` |
| `group`    | `group`       | `Group`          | `group` | `GroupCompanyTraits` |
| `alias`    | `alias`       | N/A              | `alias` | N/A (no properties) |

### General Naming Rules

| Source             | Target         | Convention         | Example                                              |
| ------------------ | -------------- | ------------------ | ---------------------------------------------------- |
| Event `name`       | Function name  | camelCase + prefix | `"Some Track Event"` → `trackSomeTrackEvent`         |
| Event `name`       | Interface name | PascalCase + prefix| `"Some Track Event"` → `TrackSomeTrackEventProperties` |
| Property `name`    | Type alias     | Property prefix    | `someString` → `PropertySomeString`                  |
| Custom type `name` | Type alias     | CustomType prefix  | `SomeStringType` → `CustomTypeSomeStringType`        |

### Interface Suffixes by Event Type

| Event Type | Suffix       | Example                          |
|------------|--------------|----------------------------------|
| `track`    | `Properties` | `TrackUserSignedUpProperties`    |
| `identify` | `Traits`     | `IdentifyUserTraits`             |
| `page`     | `Properties` | `PageProductViewedProperties`    |
| `group`    | `Traits`     | `GroupCompanyTraits`             |

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

Generated methods include function overloads to support alternate invocations matching the JS SDK's flexibility:

```typescript
/**
 * This is a track event for testing.
 */
// Overload signatures for alternate invocations
trackSomeTrackEvent(props: TrackSomeTrackEventProperties): void;
trackSomeTrackEvent(props: TrackSomeTrackEventProperties, callback: ApiCallback): void;
trackSomeTrackEvent(props: TrackSomeTrackEventProperties, options: ApiOptions, callback?: ApiCallback): void;
// Implementation signature
trackSomeTrackEvent(
  props: TrackSomeTrackEventProperties,
  optionsOrCallback?: ApiOptions | ApiCallback,
  callback?: ApiCallback
): void {
  let options: ApiOptions | undefined;
  let cb: ApiCallback | undefined;

  if (typeof optionsOrCallback === 'function') {
    cb = optionsOrCallback;
  } else {
    options = optionsOrCallback;
    cb = callback;
  }

  this.analytics.track(
    'Some Track Event',
    props,
    this.withRudderTyperContext(options),
    cb
  );
}
```

This enables all these invocation patterns:
```typescript
// Just properties
rudderTyper.trackSomeTrackEvent({ someString: 'hello', ... });

// Properties + callback (skip options)
rudderTyper.trackSomeTrackEvent({ someString: 'hello', ... }, () => console.log('done'));

// Properties + options
rudderTyper.trackSomeTrackEvent({ someString: 'hello', ... }, { integrations: { All: false } });

// Full: properties + options + callback
rudderTyper.trackSomeTrackEvent({ someString: 'hello', ... }, { integrations: { All: false } }, () => console.log('done'));
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
  callback?: ApiCallback
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
  callback?: ApiCallback
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
// Overload signatures
trackSomeEmptyTrackEvent(): void;
trackSomeEmptyTrackEvent(callback: ApiCallback): void;
trackSomeEmptyTrackEvent(options: ApiOptions, callback?: ApiCallback): void;
// Implementation
trackSomeEmptyTrackEvent(
  optionsOrCallback?: ApiOptions | ApiCallback,
  callback?: ApiCallback
): void {
  let options: ApiOptions | undefined;
  let cb: ApiCallback | undefined;

  if (typeof optionsOrCallback === 'function') {
    cb = optionsOrCallback;
  } else {
    options = optionsOrCallback;
    cb = callback;
  }

  this.analytics.track('Some Empty Track Event', {}, this.withRudderTyperContext(options), cb);
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

// Overload signatures - properties are optional
trackSomeEmptyTrackEventWithAdditionalProperties(): void;
trackSomeEmptyTrackEventWithAdditionalProperties(callback: ApiCallback): void;
trackSomeEmptyTrackEventWithAdditionalProperties(
  props: TrackSomeEmptyTrackEventWithAdditionalPropertiesProperties,
): void;
trackSomeEmptyTrackEventWithAdditionalProperties(
  props: TrackSomeEmptyTrackEventWithAdditionalPropertiesProperties,
  callback: ApiCallback,
): void;
trackSomeEmptyTrackEventWithAdditionalProperties(
  props: TrackSomeEmptyTrackEventWithAdditionalPropertiesProperties,
  options: ApiOptions,
  callback?: ApiCallback,
): void;
// Implementation
trackSomeEmptyTrackEventWithAdditionalProperties(
  propsOrCallback?: TrackSomeEmptyTrackEventWithAdditionalPropertiesProperties | ApiCallback,
  optionsOrCallback?: ApiOptions | ApiCallback,
  callback?: ApiCallback
): void {
  let props: TrackSomeEmptyTrackEventWithAdditionalPropertiesProperties = {};
  let options: ApiOptions | undefined;
  let cb: ApiCallback | undefined;

  if (typeof propsOrCallback === 'function') {
    cb = propsOrCallback;
  } else if (propsOrCallback !== undefined) {
    props = propsOrCallback;
    if (typeof optionsOrCallback === 'function') {
      cb = optionsOrCallback;
    } else {
      options = optionsOrCallback;
      cb = callback;
    }
  }

  this.analytics.track(
    'Some Empty Track Event With Additional Properties',
    props,
    this.withRudderTyperContext(options),
    cb
  );
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
import type {
  RudderAnalytics,
  ApiOptions,
  ApiObject,
  ApiCallback,
} from '@rudderstack/analytics-js';

// 3. Custom Types (type aliases)
/** some string custom type */
type CustomTypeSomeStringType = string;

/** some object custom type */
interface CustomTypeSomeObjectType {
  someCustomString: PropertySomeCustomString;
  someInteger?: PropertySomeInteger;
}

// 4. Property Types (type aliases)
/** some string property */
type PropertySomeString = string;

/** some integer property */
type PropertySomeInteger = number;

// 5. Event Properties Interfaces
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

// 6. Configuration (optional - for future violation handling)
// export interface RudderTyperOptions {
//   analytics?: RudderAnalytics;
//   onViolation?: ViolationHandler;
// }

// 7. RudderTyper Analytics Class
export class RudderTyperAnalytics {
  private analytics: RudderAnalytics;

  constructor(analytics: RudderAnalytics) {
    this.analytics = analytics;
  }

  // 8. Track functions with overloads for alternate invocations
  trackSomeTrackEvent(props: TrackSomeTrackEventProperties): void;
  trackSomeTrackEvent(props: TrackSomeTrackEventProperties, callback: ApiCallback): void;
  trackSomeTrackEvent(props: TrackSomeTrackEventProperties, options: ApiOptions, callback?: ApiCallback): void;
  trackSomeTrackEvent(
    props: TrackSomeTrackEventProperties,
    optionsOrCallback?: ApiOptions | ApiCallback,
    callback?: ApiCallback
  ): void {
    let options: ApiOptions | undefined;
    let cb: ApiCallback | undefined;

    if (typeof optionsOrCallback === 'function') {
      cb = optionsOrCallback;
    } else {
      options = optionsOrCallback;
      cb = callback;
    }

    this.analytics.track('Some Track Event', props, this.withRudderTyperContext(options), cb);
  }

  // 9. Identify function with overloads
  identify(userId: string, traits?: IdentifyTraits): void;
  identify(userId: string, traits: IdentifyTraits, callback: ApiCallback): void;
  identify(userId: string, traits: IdentifyTraits, options: ApiOptions, callback?: ApiCallback): void;
  identify(
    userId: string,
    traits?: IdentifyTraits,
    optionsOrCallback?: ApiOptions | ApiCallback,
    callback?: ApiCallback
  ): void {
    let options: ApiOptions | undefined;
    let cb: ApiCallback | undefined;

    if (typeof optionsOrCallback === 'function') {
      cb = optionsOrCallback;
    } else {
      options = optionsOrCallback;
      cb = callback;
    }

    this.analytics.identify(userId, traits, this.withRudderTyperContext(options), cb);
  }

  // 10. Context helper (private)
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

// 11. Exported Types
export type {
  // Custom Types
  CustomTypeSomeStringType,
  CustomTypeSomeObjectType,
  // Property Types
  PropertySomeString,
  PropertySomeInteger,
  // Event Properties
  TrackSomeTrackEventProperties,
  IdentifyTraits,
};
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

## Event Types

RudderTyper generates type-safe methods for all supported event types. Each event type has its own method signature with function overloads to support alternate invocations matching the JS SDK's flexibility.

> **Reference:** See [JS SDK Supported APIs](https://www.rudderstack.com/docs/sources/event-streams/sdks/rudderstack-javascript-sdk/supported-api/) for all supported invocation patterns.

---

### Track Events

Track events are covered in the [Events](#events) and [Event Rules](#event-rules) sections above. They support:
- Events with required properties
- Events with no properties (empty)
- Events with optional properties (allow_unplanned)
- Events with variants (discriminated unions)

**Method Naming:** `track{EventName}` (e.g., `trackUserSignedUp`)

**Interface Naming:** `Track{EventName}Properties` (e.g., `TrackUserSignedUpProperties`)

---

### Identify Events

Identify events associate a user with their actions and record traits about them.

**Input (events.yaml):**

```yaml
- id: "user_identify"
  name: "User Identify"
  event_type: "identify"
  description: "Identifies a user with their traits."
```

**Input (tracking-plan.yaml):**

```yaml
- type: "event_rule"
  id: "user_identify_rule"
  event:
    $ref: "#/events/typer-test/user_identify"
  properties:
    - $ref: "#/properties/typer-test/user-name"
      required: false
    - $ref: "#/properties/typer-test/email"
      required: true
    - $ref: "#/properties/typer-test/plan"
      required: false
```

**Output (index.ts):**

```typescript
/** Traits for User Identify */
interface IdentifyUserIdentifyTraits {
  userName?: PropertyUserName;
  email: PropertyEmail;
  plan?: PropertyPlan;
}

/**
 * Identifies a user with their traits.
 */
// Overload signatures
identify(userId: string): void;
identify(userId: string, traits: IdentifyUserIdentifyTraits): void;
identify(userId: string, traits: IdentifyUserIdentifyTraits, callback: ApiCallback): void;
identify(userId: string, traits: IdentifyUserIdentifyTraits, options: ApiOptions, callback?: ApiCallback): void;
// Implementation
identify(
  userId: string,
  traits?: IdentifyUserIdentifyTraits,
  optionsOrCallback?: ApiOptions | ApiCallback,
  callback?: ApiCallback,
): void {
  let options: ApiOptions | undefined;
  let cb: ApiCallback | undefined;

  if (typeof optionsOrCallback === 'function') {
    cb = optionsOrCallback;
  } else {
    options = optionsOrCallback;
    cb = callback;
  }

  this.analytics.identify(userId, traits, this.withRudderTyperContext(options), cb);
}
```

**Usage Examples:**

```typescript
// Just userId
rudderTyper.identify('user-123');

// userId + traits
rudderTyper.identify('user-123', { email: 'user@example.com' });

// userId + traits + callback
rudderTyper.identify('user-123', { email: 'user@example.com' }, () => console.log('identified'));

// Full: userId + traits + options + callback
rudderTyper.identify('user-123', { email: 'user@example.com' }, { integrations: { All: false } }, () => {});
```

---

### Page Events

Page events record page views in web applications.

> **Note:** The JavaScript SDK uses `page()` for page view tracking. The `screen()` method is **not supported** in the JS SDK - it is only available in mobile SDKs (Kotlin, Swift, etc.) for screen view tracking.

**JS SDK Supported Invocations:**
```javascript
// Default invocation
page([category], [name], [properties], [apiOptions], [callback]);

// Alternate invocations
page(name, [properties], [apiOptions], [callback]);
page(properties, [apiOptions], [callback]);
page([callback]);
```

**Input (events.yaml):**

```yaml
- id: "product_page_viewed"
  name: "Product Page Viewed"
  event_type: "page"
  description: "User viewed a product page."
```

**Input (tracking-plan.yaml):**

```yaml
- type: "event_rule"
  id: "product_page_rule"
  event:
    $ref: "#/events/typer-test/product_page_viewed"
  properties:
    - $ref: "#/properties/typer-test/product-id"
      required: true
    - $ref: "#/properties/typer-test/product-name"
      required: false
    - $ref: "#/properties/typer-test/price"
      required: false
```

**Output (index.ts):**

```typescript
/** Properties for Product Page Viewed */
interface PageProductPageViewedProperties {
  productId: PropertyProductId;
  productName?: PropertyProductName;
  price?: PropertyPrice;
}

/**
 * User viewed a product page.
 */
// Overload signatures - matching ALL JS SDK alternate invocations

// page([callback])
pageProductPageViewed(): void;
pageProductPageViewed(callback: ApiCallback): void;

// page(properties, [apiOptions], [callback])
pageProductPageViewed(properties: PageProductPageViewedProperties): void;
pageProductPageViewed(properties: PageProductPageViewedProperties, callback: ApiCallback): void;
pageProductPageViewed(properties: PageProductPageViewedProperties, options: ApiOptions): void;
pageProductPageViewed(properties: PageProductPageViewedProperties, options: ApiOptions, callback: ApiCallback): void;

// page(name, [properties], [apiOptions], [callback])
pageProductPageViewed(name: string): void;
pageProductPageViewed(name: string, properties: PageProductPageViewedProperties): void;
pageProductPageViewed(name: string, properties: PageProductPageViewedProperties, callback: ApiCallback): void;
pageProductPageViewed(name: string, properties: PageProductPageViewedProperties, options: ApiOptions): void;
pageProductPageViewed(name: string, properties: PageProductPageViewedProperties, options: ApiOptions, callback: ApiCallback): void;

// page([category], [name], [properties], [apiOptions], [callback])
pageProductPageViewed(category: string, name: string): void;
pageProductPageViewed(category: string, name: string, properties: PageProductPageViewedProperties): void;
pageProductPageViewed(category: string, name: string, properties: PageProductPageViewedProperties, callback: ApiCallback): void;
pageProductPageViewed(category: string, name: string, properties: PageProductPageViewedProperties, options: ApiOptions): void;
pageProductPageViewed(category: string, name: string, properties: PageProductPageViewedProperties, options: ApiOptions, callback: ApiCallback): void;

// Implementation with parameter detection
pageProductPageViewed(
  categoryOrNameOrPropertiesOrCallback?: string | PageProductPageViewedProperties | ApiCallback,
  nameOrPropertiesOrOptionsOrCallback?: string | PageProductPageViewedProperties | ApiOptions | ApiCallback,
  propertiesOrOptionsOrCallback?: PageProductPageViewedProperties | ApiOptions | ApiCallback,
  optionsOrCallback?: ApiOptions | ApiCallback,
  callback?: ApiCallback,
): void {
  let category: string | undefined;
  let name: string | undefined;
  let properties: PageProductPageViewedProperties | undefined;
  let options: ApiOptions | undefined;
  let cb: ApiCallback | undefined;

  // Parameter detection logic matching JS SDK behavior
  if (typeof categoryOrNameOrPropertiesOrCallback === 'function') {
    // page(callback)
    cb = categoryOrNameOrPropertiesOrCallback;
  } else if (typeof categoryOrNameOrPropertiesOrCallback === 'object') {
    // page(properties, ...)
    properties = categoryOrNameOrPropertiesOrCallback;
    if (typeof nameOrPropertiesOrOptionsOrCallback === 'function') {
      cb = nameOrPropertiesOrOptionsOrCallback;
    } else if (typeof nameOrPropertiesOrOptionsOrCallback === 'object') {
      options = nameOrPropertiesOrOptionsOrCallback as ApiOptions;
      cb = propertiesOrOptionsOrCallback as ApiCallback;
    }
  } else if (typeof categoryOrNameOrPropertiesOrCallback === 'string') {
    if (typeof nameOrPropertiesOrOptionsOrCallback === 'string') {
      // page(category, name, ...)
      category = categoryOrNameOrPropertiesOrCallback;
      name = nameOrPropertiesOrOptionsOrCallback;
      if (typeof propertiesOrOptionsOrCallback === 'object') {
        properties = propertiesOrOptionsOrCallback as PageProductPageViewedProperties;
        if (typeof optionsOrCallback === 'function') {
          cb = optionsOrCallback;
        } else if (optionsOrCallback !== undefined) {
          options = optionsOrCallback;
          cb = callback;
        }
      } else if (typeof propertiesOrOptionsOrCallback === 'function') {
        cb = propertiesOrOptionsOrCallback;
      }
    } else {
      // page(name, ...)
      name = categoryOrNameOrPropertiesOrCallback;
      if (typeof nameOrPropertiesOrOptionsOrCallback === 'object') {
        properties = nameOrPropertiesOrOptionsOrCallback as PageProductPageViewedProperties;
        if (typeof propertiesOrOptionsOrCallback === 'function') {
          cb = propertiesOrOptionsOrCallback;
        } else if (typeof propertiesOrOptionsOrCallback === 'object') {
          options = propertiesOrOptionsOrCallback as ApiOptions;
          cb = optionsOrCallback as ApiCallback;
        }
      } else if (typeof nameOrPropertiesOrOptionsOrCallback === 'function') {
        cb = nameOrPropertiesOrOptionsOrCallback;
      }
    }
  }

  this.analytics.page(category, name, properties, this.withRudderTyperContext(options), cb);
}
```

**Usage Examples:**

```typescript
// page([callback])
rudderTyper.pageProductPageViewed();
rudderTyper.pageProductPageViewed(() => console.log('done'));

// page(properties, [apiOptions], [callback])
rudderTyper.pageProductPageViewed({ productId: 'prod-123' });
rudderTyper.pageProductPageViewed({ productId: 'prod-123' }, () => console.log('done'));
rudderTyper.pageProductPageViewed({ productId: 'prod-123' }, { integrations: { All: false } });
rudderTyper.pageProductPageViewed({ productId: 'prod-123' }, { integrations: { All: false } }, () => {});

// page(name, [properties], [apiOptions], [callback])
rudderTyper.pageProductPageViewed('Product Detail');
rudderTyper.pageProductPageViewed('Product Detail', { productId: 'prod-123' });
rudderTyper.pageProductPageViewed('Product Detail', { productId: 'prod-123' }, () => {});
rudderTyper.pageProductPageViewed('Product Detail', { productId: 'prod-123' }, { integrations: {} });
rudderTyper.pageProductPageViewed('Product Detail', { productId: 'prod-123' }, { integrations: {} }, () => {});

// page([category], [name], [properties], [apiOptions], [callback])
rudderTyper.pageProductPageViewed('Products', 'Product Detail');
rudderTyper.pageProductPageViewed('Products', 'Product Detail', { productId: 'prod-123' });
rudderTyper.pageProductPageViewed('Products', 'Product Detail', { productId: 'prod-123' }, () => {});
rudderTyper.pageProductPageViewed('Products', 'Product Detail', { productId: 'prod-123' }, { integrations: {} });
rudderTyper.pageProductPageViewed('Products', 'Product Detail', { productId: 'prod-123' }, { integrations: {} }, () => {});
```

---

### Group Events

Group events associate a user with a group (company, organization, account, etc.).

**Input (events.yaml):**

```yaml
- id: "company_group"
  name: "Company Group"
  event_type: "group"
  description: "Associates user with their company."
```

**Input (tracking-plan.yaml):**

```yaml
- type: "event_rule"
  id: "company_group_rule"
  event:
    $ref: "#/events/typer-test/company_group"
  properties:
    - $ref: "#/properties/typer-test/company-name"
      required: true
    - $ref: "#/properties/typer-test/industry"
      required: false
    - $ref: "#/properties/typer-test/employee-count"
      required: false
```

**Output (index.ts):**

```typescript
/** Traits for Company Group */
interface GroupCompanyGroupTraits {
  companyName: PropertyCompanyName;
  industry?: PropertyIndustry;
  employeeCount?: PropertyEmployeeCount;
}

/**
 * Associates user with their company.
 */
// Overload signatures
group(groupId: string): void;
group(groupId: string, traits: GroupCompanyGroupTraits): void;
group(groupId: string, traits: GroupCompanyGroupTraits, callback: ApiCallback): void;
group(groupId: string, traits: GroupCompanyGroupTraits, options: ApiOptions, callback?: ApiCallback): void;
// Implementation
group(
  groupId: string,
  traits?: GroupCompanyGroupTraits,
  optionsOrCallback?: ApiOptions | ApiCallback,
  callback?: ApiCallback,
): void {
  let options: ApiOptions | undefined;
  let cb: ApiCallback | undefined;

  if (typeof optionsOrCallback === 'function') {
    cb = optionsOrCallback;
  } else {
    options = optionsOrCallback;
    cb = callback;
  }

  this.analytics.group(groupId, traits, this.withRudderTyperContext(options), cb);
}
```

**Usage Examples:**

```typescript
// Just groupId
rudderTyper.group('company-456');

// groupId + traits
rudderTyper.group('company-456', { companyName: 'Acme Inc' });

// groupId + traits + callback
rudderTyper.group('company-456', { companyName: 'Acme Inc' }, () => console.log('grouped'));

// Full: groupId + traits + options + callback
rudderTyper.group('company-456', { companyName: 'Acme Inc', industry: 'Tech' }, { integrations: {} }, () => {});
```

---

### Alias Events

Alias events merge two user identities, connecting a known userId to a previous anonymous id.

**Input (events.yaml):**

```yaml
- id: "user_alias"
  name: "User Alias"
  event_type: "alias"
  description: "Merges user identities."
```

**Output (index.ts):**

```typescript
/**
 * Merges user identities.
 */
// Overload signatures
alias(to: string): void;
alias(to: string, from: string): void;
alias(to: string, from: string, callback: ApiCallback): void;
alias(to: string, from: string, options: ApiOptions, callback?: ApiCallback): void;
// Implementation
alias(
  to: string,
  from?: string,
  optionsOrCallback?: ApiOptions | ApiCallback,
  callback?: ApiCallback,
): void {
  let options: ApiOptions | undefined;
  let cb: ApiCallback | undefined;

  if (typeof optionsOrCallback === 'function') {
    cb = optionsOrCallback;
  } else {
    options = optionsOrCallback;
    cb = callback;
  }

  this.analytics.alias(to, from, this.withRudderTyperContext(options), cb);
}
```

**Usage Examples:**

```typescript
// Just new userId (from is automatically set to current anonymousId)
rudderTyper.alias('new-user-id');

// Explicit from and to
rudderTyper.alias('new-user-id', 'old-anonymous-id');

// With callback
rudderTyper.alias('new-user-id', 'old-anonymous-id', () => console.log('aliased'));

// Full with options
rudderTyper.alias('new-user-id', 'old-anonymous-id', { integrations: { All: true } }, () => {});
```

---

### Screen Events (Mobile Only)

> **Important:** The `screen()` method is **NOT supported** in the JavaScript SDK. Screen events are only available in mobile SDKs:
> - Kotlin/Android SDK
> - Swift/iOS SDK
> - React Native SDK
> - Flutter SDK
>
> For web applications, use `page()` events instead.

If a tracking plan contains screen events and TypeScript generation is requested, RudderTyper should either:
1. Skip screen events with a warning, or
2. Generate them as page events with a comment noting the conversion

---

## Edge Cases

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

All generated methods include function overloads to support alternate invocations matching the JS SDK's flexibility. This allows users to skip optional parameters like `options` and pass `callback` directly.

### Track Events

**JS SDK Supported Invocations:**
```javascript
track(event, [properties], [apiOptions], [callback]);
track(event, properties, callback);
track(event, callback);
track(event, properties);
track(event);
```

For track events with **required properties**:
```typescript
trackSomeEvent(props: TrackSomeEventProperties): void;
trackSomeEvent(props: TrackSomeEventProperties, callback: ApiCallback): void;
trackSomeEvent(props: TrackSomeEventProperties, options: ApiOptions, callback?: ApiCallback): void;
// Implementation handles parameter detection
trackSomeEvent(
  props: TrackSomeEventProperties,
  optionsOrCallback?: ApiOptions | ApiCallback,
  callback?: ApiCallback
): void { ... }
```

For track events with **no properties**:
```typescript
trackSomeEmptyEvent(): void;
trackSomeEmptyEvent(callback: ApiCallback): void;
trackSomeEmptyEvent(options: ApiOptions, callback?: ApiCallback): void;
trackSomeEmptyEvent(
  optionsOrCallback?: ApiOptions | ApiCallback,
  callback?: ApiCallback
): void { ... }
```

For track events with **optional properties** (allow_unplanned):
```typescript
trackSomeEvent(): void;
trackSomeEvent(callback: ApiCallback): void;
trackSomeEvent(props: TrackSomeEventProperties): void;
trackSomeEvent(props: TrackSomeEventProperties, callback: ApiCallback): void;
trackSomeEvent(props: TrackSomeEventProperties, options: ApiOptions, callback?: ApiCallback): void;
trackSomeEvent(
  propsOrCallback?: TrackSomeEventProperties | ApiCallback,
  optionsOrCallback?: ApiOptions | ApiCallback,
  callback?: ApiCallback
): void { ... }
```

### Identify

**JS SDK Supported Invocations:**
```javascript
identify([userId], [traits], [apiOptions], [callback]);
identify(userId, traits, callback);
identify(userId, callback);
identify(userId, traits);
identify(userId);
identify(traits, apiOptions, callback);
identify(traits, callback);
identify(traits);
```

```typescript
identify(userId: string): void;
identify(userId: string, traits: IdentifyTraits): void;
identify(userId: string, traits: IdentifyTraits, callback: ApiCallback): void;
identify(userId: string, traits: IdentifyTraits, options: ApiOptions): void;
identify(userId: string, traits: IdentifyTraits, options: ApiOptions, callback: ApiCallback): void;
identify(
  userId: string,
  traits?: IdentifyTraits,
  optionsOrCallback?: ApiOptions | ApiCallback,
  callback?: ApiCallback
): void { ... }
```

### Page

> **Note:** The JS SDK does not support `screen()`. Use `page()` for page view tracking.

**JS SDK Supported Invocations:**
```javascript
page([category], [name], [properties], [apiOptions], [callback]);
page(name, [properties], [apiOptions], [callback]);
page(properties, [apiOptions], [callback]);
page([callback]);
```

```typescript
// page([callback])
pageEventName(): void;
pageEventName(callback: ApiCallback): void;

// page(properties, [apiOptions], [callback])
pageEventName(properties: PageEventNameProperties): void;
pageEventName(properties: PageEventNameProperties, callback: ApiCallback): void;
pageEventName(properties: PageEventNameProperties, options: ApiOptions): void;
pageEventName(properties: PageEventNameProperties, options: ApiOptions, callback: ApiCallback): void;

// page(name, [properties], [apiOptions], [callback])
pageEventName(name: string): void;
pageEventName(name: string, properties: PageEventNameProperties): void;
pageEventName(name: string, properties: PageEventNameProperties, callback: ApiCallback): void;
pageEventName(name: string, properties: PageEventNameProperties, options: ApiOptions): void;
pageEventName(name: string, properties: PageEventNameProperties, options: ApiOptions, callback: ApiCallback): void;

// page([category], [name], [properties], [apiOptions], [callback])
pageEventName(category: string, name: string): void;
pageEventName(category: string, name: string, properties: PageEventNameProperties): void;
pageEventName(category: string, name: string, properties: PageEventNameProperties, callback: ApiCallback): void;
pageEventName(category: string, name: string, properties: PageEventNameProperties, options: ApiOptions): void;
pageEventName(category: string, name: string, properties: PageEventNameProperties, options: ApiOptions, callback: ApiCallback): void;

// Implementation with parameter detection
pageEventName(
  categoryOrNameOrPropertiesOrCallback?: string | PageEventNameProperties | ApiCallback,
  nameOrPropertiesOrOptionsOrCallback?: string | PageEventNameProperties | ApiOptions | ApiCallback,
  propertiesOrOptionsOrCallback?: PageEventNameProperties | ApiOptions | ApiCallback,
  optionsOrCallback?: ApiOptions | ApiCallback,
  callback?: ApiCallback,
): void { ... }
```

### Group

**JS SDK Supported Invocations:**
```javascript
group(groupId, [traits], [apiOptions], [callback]);
group(groupId, traits, callback);
group(groupId, callback);
group(groupId, traits);
group(groupId);
group(traits, apiOptions, callback);
group(traits, callback);
```

```typescript
group(groupId: string): void;
group(groupId: string, traits: GroupTraits): void;
group(groupId: string, traits: GroupTraits, callback: ApiCallback): void;
group(groupId: string, traits: GroupTraits, options: ApiOptions): void;
group(groupId: string, traits: GroupTraits, options: ApiOptions, callback: ApiCallback): void;
group(
  groupId: string,
  traits?: GroupTraits,
  optionsOrCallback?: ApiOptions | ApiCallback,
  callback?: ApiCallback
): void { ... }
```

### Alias

**JS SDK Supported Invocations:**
```javascript
alias(to, [from], [apiOptions], [callback]);
alias(to, from, callback);
alias(to, callback);
alias(to, from);
alias(to);
```

```typescript
alias(to: string): void;
alias(to: string, from: string): void;
alias(to: string, from: string, callback: ApiCallback): void;
alias(to: string, from: string, options: ApiOptions): void;
alias(to: string, from: string, options: ApiOptions, callback: ApiCallback): void;
alias(
  to: string,
  from?: string,
  optionsOrCallback?: ApiOptions | ApiCallback,
  callback?: ApiCallback
): void { ... }
```

---

## Validation Checklist

### Architecture
- [ ] Class-based approach: `RudderTyperAnalytics` class with constructor accepting `RudderAnalytics`
- [ ] No global declaration (`declare global`) - use dependency injection via constructor
- [ ] RudderTyper context included in all calls via `this.withRudderTyperContext()`
- [ ] **Function overloads for all methods** to support alternate invocations (props, callback) pattern

### Type System
- [ ] Primitive types map correctly (string, number, boolean, null)
- [ ] Required properties have no `?` modifier
- [ ] Optional properties have `?` modifier
- [ ] Enums generate as union types (not TypeScript enums with `S_`/`N_` prefixes)
- [ ] Custom types use `CustomType` prefix as type aliases
- [ ] Property types use `Property` prefix as type aliases
- [ ] Nested objects use inline types (not separate interfaces)
- [ ] `allow_unplanned: true` results in `Record<string, any>`
- [ ] Null type supported
- [ ] Deeply nested objects use inline types

### Naming Conventions
- [ ] Method names are camelCase with event type prefix (e.g., `trackSomeEvent`, `pageProductViewed`)
- [ ] Interface names are PascalCase with prefix (e.g., `TrackSomeEventProperties`, `IdentifyUserTraits`)
- [ ] Track/Page use `Properties` suffix, Identify/Group use `Traits` suffix
- [ ] Event names preserved exactly in SDK calls
- [ ] Special characters in names sanitized
- [ ] Name collisions resolved with numeric suffix
- [ ] Reserved words prefixed with underscore

### Event Types
- [ ] **Track events** - `trackEventName(props, options?, callback?)` with typed properties
- [ ] **Identify events** - `identify(userId, traits?, options?, callback?)` with typed traits
- [ ] **Page events** - `pageEventName(category?, name?, properties?, options?, callback?)` with typed properties
- [ ] **Group events** - `group(groupId, traits?, options?, callback?)` with typed traits
- [ ] **Alias events** - `alias(to, from?, options?, callback?)`
- [ ] **Screen events** - NOT supported in JS SDK (skip or convert to page with warning)

### Variants
- [ ] Variants generate discriminated unions
- [ ] Variant cases use literal types for discriminator
- [ ] Variant default case uses `Exclude<>` type for discriminator

### Exports
- [ ] Export `RudderTyperAnalytics` class
- [ ] Export all custom types (`CustomType*`)
- [ ] Export all property types (`Property*`)
- [ ] Export all event property/trait interfaces (`Track*Properties`, `Identify*Traits`, etc.)
