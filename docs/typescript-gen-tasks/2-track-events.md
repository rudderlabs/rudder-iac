# 04: Track Events

**Priority:** P0 (Core)
**Depends on:** 01, 02
**Complexity:** Medium

---

## Goal
Generate track event interfaces and methods with function overloads matching JS SDK flexibility.

---

## What to Build

### 1. Properties Interface
```yaml
# Input
- type: "event_rule"
  event:
    $ref: "#/events/typer-test/some_track_event"
  properties:
    - $ref: "#/properties/typer-test/some-string"
      required: true
    - $ref: "#/properties/typer-test/some-boolean"
      required: false
```
```typescript
// Output
interface TrackSomeTrackEventProperties {
  someString: PropertySomeString;   // required - no ?
  someBoolean?: PropertySomeBoolean; // optional - has ?
}
```

### 2. Method with Overloads
Generate overloads so users can skip `options` and pass `callback` directly:

```typescript
/**
 * This is a track event for testing.
 */
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

  this.analytics.track(
    'Some Track Event',  // Original event name
    props,
    this.withRudderTyperContext(options),
    cb
  );
}
```

### 3. Empty Track Events
Events with no properties:
```typescript
trackSomeEmptyEvent(): void;
trackSomeEmptyEvent(callback: ApiCallback): void;
trackSomeEmptyEvent(options: ApiOptions, callback?: ApiCallback): void;
trackSomeEmptyEvent(
  optionsOrCallback?: ApiOptions | ApiCallback,
  callback?: ApiCallback
): void {
  // ... parameter detection ...
  this.analytics.track('Some Empty Event', {}, this.withRudderTyperContext(options), cb);
}
```

### 4. Allow Unplanned Properties
When `allow_unplanned: true`, properties are optional and can be anything:
```yaml
event:
  $ref: "#/events/typer-test/flexible_event"
  allow_unplanned: true
```
```typescript
type TrackFlexibleEventProperties = Record<string, any>;

trackFlexibleEvent(): void;
trackFlexibleEvent(callback: ApiCallback): void;
trackFlexibleEvent(props: TrackFlexibleEventProperties): void;
trackFlexibleEvent(props: TrackFlexibleEventProperties, callback: ApiCallback): void;
trackFlexibleEvent(props: TrackFlexibleEventProperties, options: ApiOptions, callback?: ApiCallback): void;
trackFlexibleEvent(
  propsOrCallback?: TrackFlexibleEventProperties | ApiCallback,
  optionsOrCallback?: ApiOptions | ApiCallback,
  callback?: ApiCallback
): void {
  let props: TrackFlexibleEventProperties = {};
  // ... more complex parameter detection ...
}
```

---

## Naming Convention

| Source | Pattern | Example |
|--------|---------|---------|
| Event name | `track{PascalCase}` | `trackUserSignedUp` |
| Interface | `Track{PascalCase}Properties` | `TrackUserSignedUpProperties` |

---

## Supported Call Patterns
All these must work:
```typescript
// Just properties
rudderTyper.trackSomeEvent({ someString: 'hello' });

// Properties + callback (skip options)
rudderTyper.trackSomeEvent({ someString: 'hello' }, () => console.log('done'));

// Properties + options
rudderTyper.trackSomeEvent({ someString: 'hello' }, { integrations: { All: false } });

// Full
rudderTyper.trackSomeEvent(
  { someString: 'hello' },
  { integrations: { All: false } },
  () => console.log('done')
);
```

---

## Key Rules
1. **Preserve event name:** Use original name in `analytics.track()` call
2. **JSDoc:** Include description from event definition
3. **Context:** Always use `this.withRudderTyperContext(options)`
4. **Parameter detection:** `typeof optionsOrCallback === 'function'`

---

## Acceptance Criteria
- [ ] Interface naming: `Track{EventName}Properties`
- [ ] Method naming: `track{eventName}`
- [ ] Required props have no `?`, optional props have `?`
- [ ] Function overloads support all call patterns
- [ ] Empty events work (no props parameter)
- [ ] `allow_unplanned` events use `Record<string, any>`
- [ ] Original event name preserved in track() call
- [ ] JSDoc comments from description
- [ ] Context helper called on all track() calls
