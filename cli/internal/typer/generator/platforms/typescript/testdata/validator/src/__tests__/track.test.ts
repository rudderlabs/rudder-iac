import { RudderAnalytics } from "@rudderstack/analytics-js/bundled";
import { beforeEach, describe, expect, it } from "vitest";
import { RudderTyper } from "../RudderTyper/RudderTyper.ts";
import {
  TEST_CONFIG_BE_URL,
  TEST_DATA_PLANE_URL,
  TEST_WRITE_KEY,
  interceptor,
} from "./eventInterceptor.ts";

describe("RudderTyper.track", () => {
  let typer: RudderTyper;

  beforeEach(async () => {
    const analytics = new RudderAnalytics();
    analytics.load(TEST_WRITE_KEY, TEST_DATA_PLANE_URL, {
      configUrl: TEST_CONFIG_BE_URL,
      logLevel: "ERROR",
      queueOptions: { maxItems: 100, batch: { enabled: false } },
      sessions: { autoTrack: false },
      uaChTrackLevel: "none",
    });
    await new Promise<void>((resolve) => analytics.ready(() => resolve()));
    typer = new RudderTyper(() => analytics);
  });

  // ---- trackUserSignedUp (comprehensive) ----

  it("trackUserSignedUp dispatches every supported property type", async () => {
    typer.trackUserSignedUp({
      active: true,
      profile: {
        email: "newuser@example.com",
        firstName: "Bob",
        lastName: "Williams",
      },
      age: 28.5,
      arrayOfAny: [
        "string item",
        42,
        true,
        { nested: "object", count: 100 },
      ],
      contacts: [
        "contact1@example.com",
        "contact2@example.com",
        "support@company.org",
      ],
      deviceType: "mobile",
      propertyOfAny: {
        custom_field_1: "value1",
        custom_field_2: 999,
        nested_object: { deep_field: "deep_value" },
      },
      tags: ["premium", "early-adopter", "beta-tester", "verified"],
      untypedArray: [
        3.14159,
        "mixed",
        false,
        { id: 123, name: "test" },
      ],
      untypedField: {
        arbitrary_key: "arbitrary_value",
        number: 42.5,
      },
    });

    const [event] = await interceptor.waitForEvents(1);

    expect(event.type).toBe("track");
    expect(event.event).toBe("User Signed Up");
    expect(event.properties).toEqual({
      active: true,
      profile: {
        email: "newuser@example.com",
        firstName: "Bob",
        lastName: "Williams",
      },
      age: 28.5,
      arrayOfAny: [
        "string item",
        42,
        true,
        { nested: "object", count: 100 },
      ],
      contacts: [
        "contact1@example.com",
        "contact2@example.com",
        "support@company.org",
      ],
      deviceType: "mobile",
      propertyOfAny: {
        custom_field_1: "value1",
        custom_field_2: 999,
        nested_object: { deep_field: "deep_value" },
      },
      tags: ["premium", "early-adopter", "beta-tester", "verified"],
      untypedArray: [
        3.14159,
        "mixed",
        false,
        { id: 123, name: "test" },
      ],
      untypedField: {
        arbitrary_key: "arbitrary_value",
        number: 42.5,
      },
    });
  });

  // ---- trackUserSignedUp (minimal — required-only) ----

  it("trackUserSignedUp dispatches only the required properties when optionals are omitted", async () => {
    typer.trackUserSignedUp({
      active: false,
      profile: { email: "minimal@example.com", firstName: "Charlie" },
    });

    const [event] = await interceptor.waitForEvents(1);

    expect(event.type).toBe("track");
    expect(event.event).toBe("User Signed Up");
    expect(event.properties).toEqual({
      active: false,
      profile: { email: "minimal@example.com", firstName: "Charlie" },
    });
  });

  // ---- trackUserSignedUp (every device_type enum case) ----

  it("trackUserSignedUp preserves each PropertyDeviceType literal on the wire", async () => {
    typer.trackUserSignedUp({
      active: true,
      profile: {
        email: "tablet.user@example.com",
        firstName: "Diana",
        lastName: "Martinez",
      },
      deviceType: "tablet",
    });
    typer.trackUserSignedUp({
      active: true,
      profile: { email: "desktop.user@example.com", firstName: "Edward" },
      deviceType: "desktop",
    });
    typer.trackUserSignedUp({
      active: true,
      profile: { email: "tv.user@example.com", firstName: "Fiona", lastName: "Chen" },
      age: 45.0,
      deviceType: "smartTV",
      tags: ["smart-home", "entertainment"],
    });
    typer.trackUserSignedUp({
      active: true,
      profile: { email: "iot@example.com", firstName: "George" },
      deviceType: "IoT-Device",
    });

    const events = await interceptor.waitForEvents(4);

    expect(events.map((e) => e.type)).toEqual(["track", "track", "track", "track"]);
    expect(events.map((e) => e.properties)).toEqual([
      {
        active: true,
        profile: {
          email: "tablet.user@example.com",
          firstName: "Diana",
          lastName: "Martinez",
        },
        deviceType: "tablet",
      },
      {
        active: true,
        profile: { email: "desktop.user@example.com", firstName: "Edward" },
        deviceType: "desktop",
      },
      {
        active: true,
        profile: {
          email: "tv.user@example.com",
          firstName: "Fiona",
          lastName: "Chen",
        },
        age: 45.0,
        deviceType: "smartTV",
        tags: ["smart-home", "entertainment"],
      },
      {
        active: true,
        profile: { email: "iot@example.com", firstName: "George" },
        deviceType: "IoT-Device",
      },
    ]);
  });

  // ---- trackUserSignedUp (hoisted nested context, array of custom-type objects, variant base-only) ----

  it("trackUserSignedUp passes hoisted nested objects, custom-type arrays, and variant-base fields", async () => {
    typer.trackUserSignedUp({
      active: true,
      profile: { email: "nest@example.com", firstName: "Nina" },
      addresses: [
        { street: "1 Main St", city: "Springfield", postalCode: "12345" },
        { street: "2 Elm Rd", city: "Shelbyville" },
      ],
      profileList: [
        { email: "a@example.com", firstName: "Alice" },
        { email: "b@example.com", firstName: "Bob", lastName: "Brown" },
      ],
      context: {
        ipAddress: "203.0.113.42",
        nestedContext: {
          favoriteColors: ["red", "blue", "green"],
          profile: {
            email: "deep@example.com",
            firstName: "Deep",
            lastName: "Stack",
          },
        },
      },
      featureConfig: { featureFlag: "beta" },
      userAccess: { active: true, email: "access@example.com" },
      status: "active",
      priority: 3,
      rating: 4.5,
      enabled: true,
    });

    const [event] = await interceptor.waitForEvents(1);

    expect(event.type).toBe("track");
    expect(event.event).toBe("User Signed Up");
    expect(event.properties).toEqual({
      active: true,
      profile: { email: "nest@example.com", firstName: "Nina" },
      addresses: [
        { street: "1 Main St", city: "Springfield", postalCode: "12345" },
        { street: "2 Elm Rd", city: "Shelbyville" },
      ],
      profileList: [
        { email: "a@example.com", firstName: "Alice" },
        { email: "b@example.com", firstName: "Bob", lastName: "Brown" },
      ],
      context: {
        ipAddress: "203.0.113.42",
        nestedContext: {
          favoriteColors: ["red", "blue", "green"],
          profile: {
            email: "deep@example.com",
            firstName: "Deep",
            lastName: "Stack",
          },
        },
      },
      featureConfig: { featureFlag: "beta" },
      userAccess: { active: true, email: "access@example.com" },
      status: "active",
      priority: 3,
      rating: 4.5,
      enabled: true,
    });
  });

  // ---- trackProductPremiumClicked (event name with quotes + statusCode enum) ----

  it("trackProductPremiumClicked dispatches the literal event name and statusCode enum", async () => {
    typer.trackProductPremiumClicked({
      specialField: "value with \"quotes\" and \\path",
      statusCode: "404: Not Found",
    });

    const [event] = await interceptor.waitForEvents(1);

    expect(event.type).toBe("track");
    expect(event.event).toBe('Product "Premium" Clicked');
    expect(event.properties).toEqual({
      specialField: "value with \"quotes\" and \\path",
      statusCode: "404: Not Found",
    });
  });

  // ---- trackVariableString ($Variable$String / dollarField enum) ----

  it("trackVariableString dispatches the $-tokenized event name and dollarField enum", async () => {
    typer.trackVariableString({ dollarField: "$variable_name" });

    const [event] = await interceptor.waitForEvents(1);

    expect(event.type).toBe("track");
    expect(event.event).toBe("$Variable$String");
    expect(event.properties).toEqual({ dollarField: "$variable_name" });
  });

  // ---- trackEventWithNameCamelCase ($eventWithNameCamelCase$!) ----

  it("trackEventWithNameCamelCase dispatches the punctuated event name with optional field absent", async () => {
    typer.trackEventWithNameCamelCase({});

    const [event] = await interceptor.waitForEvents(1);

    expect(event.type).toBe("track");
    expect(event.event).toBe("$eventWithNameCamelCase$!");
    expect(event.properties).toEqual({});
  });

  // ---- trackEventWithNameCamelCase1 (collision counterpart — eventWithNameCamelCase) ----

  it("trackEventWithNameCamelCase1 dispatches the collision-counterpart event name", async () => {
    typer.trackEventWithNameCamelCase1({ active: true });

    const [event] = await interceptor.waitForEvents(1);

    expect(event.type).toBe("track");
    expect(event.event).toBe("eventWithNameCamelCase");
    expect(event.properties).toEqual({ active: true });
  });

  // ---- trackEmptyEventNoAdditionalProps (closed empty schema) ----

  it("trackEmptyEventNoAdditionalProps dispatches the event with no properties", async () => {
    typer.trackEmptyEventNoAdditionalProps();

    const [event] = await interceptor.waitForEvents(1);

    expect(event.type).toBe("track");
    expect(event.event).toBe("Empty Event No Additional Props");
    expect(event.properties).toEqual({});
  });

  // ---- trackEmptyEventWithAdditionalProps (open empty schema) ----

  it("trackEmptyEventWithAdditionalProps passes arbitrary properties through verbatim", async () => {
    typer.trackEmptyEventWithAdditionalProps({
      arbitrary: "value",
      count: 7,
      nested: { ok: true },
    });

    const [event] = await interceptor.waitForEvents(1);

    expect(event.type).toBe("track");
    expect(event.event).toBe("Empty Event With Additional Props");
    expect(event.properties).toEqual({
      arbitrary: "value",
      count: 7,
      nested: { ok: true },
    });
  });

  // ---- ruddertyper context injection ----

  it("merges the ruddertyper context into the dispatched track event", async () => {
    typer.trackEmptyEventNoAdditionalProps();

    const [event] = await interceptor.waitForEvents(1);

    const ctx = (event.context ?? {}) as Record<string, unknown>;
    expect(ctx.ruddertyper).toEqual({
      platform: "typescript",
      rudderCLIVersion: "1.0.0",
      trackingPlanId: "plan_12345",
      trackingPlanVersion: 13,
    });
  });
});
