# 03: Variants (Discriminated Unions)

**Priority:** P2 (Advanced)
**Depends on:** 01, 02
**Complexity:** High

---

## Goal
Generate TypeScript discriminated unions from variant definitions.

---

## What to Build

### 1. Variant Case Interfaces
Each case gets its own interface with a **literal type** for the discriminator:

```yaml
# Input
variants:
  - type: "discriminator"
    discriminator: "#/properties/typer-test/some-string"
    cases:
      - display_name: "Case 1"
        match: ["case_1"]
        properties:
          - $ref: "#/properties/typer-test/some-integer"
            required: true
```
```typescript
// Output
interface CustomTypeSomeVariantTypeCase1 {
  someString: 'case_1';  // LITERAL type, not string
  someInteger: PropertySomeInteger;
}
```

### 2. Multi-Match Cases
When a case has multiple match values, generate separate interface per value:

```yaml
match:
  - "case_2_a"
  - "case_2_b"
```
```typescript
interface CustomTypeSomeVariantTypeCase2A {
  someString: 'case_2_a';
  someNumber: PropertySomeNumber;
}
interface CustomTypeSomeVariantTypeCase2B {
  someString: 'case_2_b';
  someNumber: PropertySomeNumber;
}
```

### 3. Default Case with Exclude
Default case uses `Exclude<>` to cover all other values:

```yaml
default:
  - $ref: "#/properties/typer-test/some-boolean"
    required: false
```
```typescript
interface CustomTypeSomeVariantTypeDefault {
  someString: Exclude<string, 'case_1' | 'case_2_a' | 'case_2_b'>;
  someBoolean?: PropertySomeBoolean;
}
```

### 4. Union Type
Combine all cases into a union:

```typescript
type CustomTypeSomeVariantType =
  | CustomTypeSomeVariantTypeCase1
  | CustomTypeSomeVariantTypeCase2A
  | CustomTypeSomeVariantTypeCase2B
  | CustomTypeSomeVariantTypeDefault;
```

### 5. Event Rule Variants
Variants also work in event rules:

```yaml
# Input (tracking-plan.yaml)
- type: "event_rule"
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
          match: ["mobile"]
          properties:
            - $ref: "#/properties/typer-test/os-version"
              required: true
        - display_name: "Desktop"
          match: ["desktop"]
          properties:
            - $ref: "#/properties/typer-test/browser"
              required: true
```
```typescript
// Output
interface TrackDeviceEventPropertiesMobile {
  deviceType: 'mobile';
  osVersion: PropertyOsVersion;
}
interface TrackDeviceEventPropertiesDesktop {
  deviceType: 'desktop';
  browser: PropertyBrowser;
}
interface TrackDeviceEventPropertiesDefault {
  deviceType: Exclude<string, 'mobile' | 'desktop'>;
}
type TrackDeviceEventProperties =
  | TrackDeviceEventPropertiesMobile
  | TrackDeviceEventPropertiesDesktop
  | TrackDeviceEventPropertiesDefault;
```

### 6. Enum Discriminator
When discriminator is an enum type, use the enum type in Exclude:

```typescript
type PropertyStatus = 'pending' | 'active' | 'completed';

// Default case excludes from enum type
interface PropertiesDefault {
  status: Exclude<PropertyStatus, 'pending' | 'active'>;  // only 'completed'
}
```

---

## Naming Convention

| Component | Pattern | Example |
|-----------|---------|---------|
| Case interface | `{TypeName}{DisplayName}` | `CustomTypeSomeVariantTypeCase1` |
| Multi-match | `{TypeName}{MatchValue}` | `CustomTypeSomeVariantTypeCase2A` |
| Default | `{TypeName}Default` | `CustomTypeSomeVariantTypeDefault` |
| Union | `{TypeName}` | `CustomTypeSomeVariantType` |

---

## Why This Matters
TypeScript type narrowing works with discriminated unions:
```typescript
function handle(data: CustomTypeSomeVariantType) {
  if (data.someString === 'case_1') {
    // TS knows: data is CustomTypeSomeVariantTypeCase1
    console.log(data.someInteger);  // number, required
  }
}
```

---

## Acceptance Criteria
- [ ] Case interfaces have literal discriminator types
- [ ] Multi-match variants generate separate interfaces
- [ ] Default case uses `Exclude<>` with all match values
- [ ] Union type combines all cases
- [ ] Event rule variants work correctly
- [ ] Enum discriminators use correct Exclude type
- [ ] Base properties from parent included in all cases
- [ ] TypeScript type narrowing works with output
