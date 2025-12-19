# Task 9: Other Events

**Depends on:** Task 1-4

---

## Goal

identify, group, page, alias (same patterns as track).

---

## 1. Identify Events

```yaml
# Input
- type: "event_rule"
  event:
    $ref: "#/events/typer-test/user_identify"
    event_type: "identify"
  properties:
    - $ref: "#/properties/typer-test/email"
      required: true
    - $ref: "#/properties/typer-test/plan"
      required: false
```

```typescript
// Output
interface IdentifyUserIdentifyTraits {
  email: PropertyEmail;
  plan?: PropertyPlan;
}

identify(userId: string): void;
identify(userId: string, traits: IdentifyUserIdentifyTraits): void;
identify(userId: string, traits: IdentifyUserIdentifyTraits, callback: ApiCallback): void;
identify(userId: string, traits: IdentifyUserIdentifyTraits, options: ApiOptions, callback?: ApiCallback): void;
identify(
  userId: string,
  traits?: IdentifyUserIdentifyTraits,
  optionsOrCallback?: ApiOptions | ApiCallback,
  callback?: ApiCallback
): void {
  // ... parameter detection ...
  this.analytics.identify(userId, traits, this.withRudderTyperContext(options), cb);
}
```

**Naming:** `Identify{EventName}Traits` (uses `Traits` suffix, not `Properties`)

---

## 2. Group Events

```typescript
interface GroupCompanyGroupTraits {
  companyName: PropertyCompanyName;
  industry?: PropertyIndustry;
}

group(groupId: string): void;
group(groupId: string, traits: GroupCompanyGroupTraits): void;
group(groupId: string, traits: GroupCompanyGroupTraits, callback: ApiCallback): void;
group(groupId: string, traits: GroupCompanyGroupTraits, options: ApiOptions, callback?: ApiCallback): void;
group(
  groupId: string,
  traits?: GroupCompanyGroupTraits,
  optionsOrCallback?: ApiOptions | ApiCallback,
  callback?: ApiCallback
): void {
  // ... parameter detection ...
  this.analytics.group(groupId, traits, this.withRudderTyperContext(options), cb);
}
```

**Naming:** `Group{EventName}Traits`

---

## 3. Alias Events
Alias has no properties/traits, just `to` and `from`:

```typescript
alias(to: string): void;
alias(to: string, from: string): void;
alias(to: string, from: string, callback: ApiCallback): void;
alias(to: string, from: string, options: ApiOptions, callback?: ApiCallback): void;
alias(
  to: string,
  from?: string,
  optionsOrCallback?: ApiOptions | ApiCallback,
  callback?: ApiCallback
): void {
  // ... parameter detection ...
  this.analytics.alias(to, from, this.withRudderTyperContext(options), cb);
}
```

---

## 4. Page Events (Complex)
Page has many overload combinations matching JS SDK:

```typescript
interface PageProductPageViewedProperties {
  productId: PropertyProductId;
  productName?: PropertyProductName;
}

// All these overloads:
pageProductPageViewed(): void;
pageProductPageViewed(callback: ApiCallback): void;
pageProductPageViewed(properties: PageProductPageViewedProperties): void;
pageProductPageViewed(properties: PageProductPageViewedProperties, callback: ApiCallback): void;
pageProductPageViewed(properties: PageProductPageViewedProperties, options: ApiOptions): void;
pageProductPageViewed(properties: PageProductPageViewedProperties, options: ApiOptions, callback: ApiCallback): void;
pageProductPageViewed(name: string): void;
pageProductPageViewed(name: string, properties: PageProductPageViewedProperties): void;
pageProductPageViewed(name: string, properties: PageProductPageViewedProperties, callback: ApiCallback): void;
pageProductPageViewed(name: string, properties: PageProductPageViewedProperties, options: ApiOptions): void;
pageProductPageViewed(name: string, properties: PageProductPageViewedProperties, options: ApiOptions, callback: ApiCallback): void;
pageProductPageViewed(category: string, name: string): void;
pageProductPageViewed(category: string, name: string, properties: PageProductPageViewedProperties): void;
// ... more combinations ...

// Implementation with complex parameter detection
pageProductPageViewed(
  categoryOrNameOrPropertiesOrCallback?: string | PageProductPageViewedProperties | ApiCallback,
  nameOrPropertiesOrOptionsOrCallback?: string | PageProductPageViewedProperties | ApiOptions | ApiCallback,
  propertiesOrOptionsOrCallback?: PageProductPageViewedProperties | ApiOptions | ApiCallback,
  optionsOrCallback?: ApiOptions | ApiCallback,
  callback?: ApiCallback
): void {
  // Complex parameter detection logic
  // See spec for full implementation
}
```

**Naming:** `Page{EventName}Properties`

---

## 5. Screen Events
**Important:** Screen is NOT supported in JS SDK. Options:
1. Skip with warning comment
2. Convert to page with comment

```typescript
// Option 1: Skip
// Screen events not supported in JS SDK. Use page() instead.

// Option 2: Convert
/**
 * Note: Screen events converted to page() for JS SDK compatibility.
 * Original event type was 'screen'.
 */
pageAppHomeViewed(...): void { ... }
```

---

## Summary Table

| Event Type | Method | Interface | Suffix |
|------------|--------|-----------|--------|
| identify | `identify` | `Identify{Name}Traits` | `Traits` |
| group | `group` | `Group{Name}Traits` | `Traits` |
| alias | `alias` | N/A | N/A |
| page | `page{Name}` | `Page{Name}Properties` | `Properties` |
| screen | Skip or convert | - | - |

---

## Acceptance Criteria
- [ ] Identify: generates traits interface + method with userId first
- [ ] Group: generates traits interface + method with groupId first
- [ ] Alias: generates method with to/from parameters
- [ ] Page: generates all JS SDK overload combinations
- [ ] Screen: handled (skip with warning or convert to page)
- [ ] All use correct naming suffixes (Traits vs Properties)
- [ ] All use `withRudderTyperContext()`
- [ ] All support callback in different positions
