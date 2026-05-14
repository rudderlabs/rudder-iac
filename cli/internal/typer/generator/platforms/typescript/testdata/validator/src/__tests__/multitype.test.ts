import { RudderAnalytics } from "@rudderstack/analytics-js/bundled";
import { beforeEach, describe, expect, it } from "vitest";
import { RudderTyper } from "../RudderTyper/RudderTyper.ts";
import {
  TEST_CONFIG_BE_URL,
  TEST_DATA_PLANE_URL,
  TEST_WRITE_KEY,
  interceptor,
} from "./eventInterceptor.ts";

describe("RudderTyper.track (multi-type properties)", () => {
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
    typer = new RudderTyper(analytics);
  });

  // ---- multiTypeField: string | number | boolean ----

  it("multiTypeField serializes each variant type correctly", async () => {
    typer.trackUserSignedUp({
      active: true,
      profile: { email: "string@example.com", firstName: "String" },
      multiTypeField: "hello world",
    });
    typer.trackUserSignedUp({
      active: true,
      profile: { email: "number@example.com", firstName: "Number" },
      multiTypeField: 42,
    });
    typer.trackUserSignedUp({
      active: true,
      profile: { email: "bool@example.com", firstName: "Bool" },
      multiTypeField: false,
    });

    const events = await interceptor.waitForEvents(3);

    expect(events.map((e) => e.properties)).toEqual([
      {
        active: true,
        profile: { email: "string@example.com", firstName: "String" },
        multiTypeField: "hello world",
      },
      {
        active: true,
        profile: { email: "number@example.com", firstName: "Number" },
        multiTypeField: 42,
      },
      {
        active: true,
        profile: { email: "bool@example.com", firstName: "Bool" },
        multiTypeField: false,
      },
    ]);
  });

  // ---- multiTypeArray: Array<string | number> ----

  it("multiTypeArray serializes mixed string and number items", async () => {
    typer.trackUserSignedUp({
      active: true,
      profile: { email: "array@example.com", firstName: "Array" },
      multiTypeArray: ["alpha", 1, "beta", 2, "gamma", 3],
    });

    const [event] = await interceptor.waitForEvents(1);

    expect(event.type).toBe("track");
    expect(event.properties).toEqual({
      active: true,
      profile: { email: "array@example.com", firstName: "Array" },
      multiTypeArray: ["alpha", 1, "beta", 2, "gamma", 3],
    });
  });

  // ---- multiTypeWithNull: string | number | null ----

  it("multiTypeWithNull serializes string, number, and null variants", async () => {
    typer.trackUserSignedUp({
      active: true,
      profile: { email: "mtn@example.com", firstName: "MTN" },
      multiTypeWithNull: "not-null",
    });
    typer.trackUserSignedUp({
      active: true,
      profile: { email: "mtn@example.com", firstName: "MTN" },
      multiTypeWithNull: 99,
    });
    typer.trackUserSignedUp({
      active: true,
      profile: { email: "mtn@example.com", firstName: "MTN" },
      multiTypeWithNull: null,
    });

    const events = await interceptor.waitForEvents(3);

    // The SDK strips null top-level property values from the payload,
    // so the null variant results in the property being absent.
    expect(events.map((e) => e.properties)).toEqual([
      {
        active: true,
        profile: { email: "mtn@example.com", firstName: "MTN" },
        multiTypeWithNull: "not-null",
      },
      {
        active: true,
        profile: { email: "mtn@example.com", firstName: "MTN" },
        multiTypeWithNull: 99,
      },
      {
        active: true,
        profile: { email: "mtn@example.com", firstName: "MTN" },
      },
    ]);
  });

  // ---- stringOrNull / numberOrNull ----

  it("stringOrNull and numberOrNull serialize both value and null variants", async () => {
    typer.trackUserSignedUp({
      active: true,
      profile: { email: "sn@example.com", firstName: "SN" },
      stringOrNull: "present",
      numberOrNull: 3.14,
    });
    typer.trackUserSignedUp({
      active: true,
      profile: { email: "sn@example.com", firstName: "SN" },
      stringOrNull: null,
      numberOrNull: null,
    });

    const events = await interceptor.waitForEvents(2);

    // The SDK strips null top-level property values from the payload.
    expect(events.map((e) => e.properties)).toEqual([
      {
        active: true,
        profile: { email: "sn@example.com", firstName: "SN" },
        stringOrNull: "present",
        numberOrNull: 3.14,
      },
      {
        active: true,
        profile: { email: "sn@example.com", firstName: "SN" },
      },
    ]);
  });

  // ---- arrayWithNullItems: Array<string | null> ----

  it("arrayWithNullItems serializes an array mixing string and null items", async () => {
    typer.trackUserSignedUp({
      active: true,
      profile: { email: "arr@example.com", firstName: "Arr" },
      arrayWithNullItems: ["one", null, "three", null],
    });

    const [event] = await interceptor.waitForEvents(1);

    expect(event.type).toBe("track");
    expect(event.properties).toEqual({
      active: true,
      profile: { email: "arr@example.com", firstName: "Arr" },
      arrayWithNullItems: ["one", null, "three", null],
    });
  });

  // ---- combined: multi-type fields alongside nested objects ----

  it("multi-type properties coexist with nested object and array properties", async () => {
    typer.trackUserSignedUp({
      active: true,
      profile: { email: "combo@example.com", firstName: "Combo" },
      multiTypeField: "text",
      multiTypeArray: [100, "mixed"],
      numberOrNull: 2.718,
      context: {
        ipAddress: "10.0.0.1",
        nestedContext: {
          profile: { email: "nested@example.com", firstName: "Nested" },
        },
      },
    });

    const [event] = await interceptor.waitForEvents(1);

    expect(event.type).toBe("track");
    expect(event.properties).toEqual({
      active: true,
      profile: { email: "combo@example.com", firstName: "Combo" },
      multiTypeField: "text",
      multiTypeArray: [100, "mixed"],
      numberOrNull: 2.718,
      context: {
        ipAddress: "10.0.0.1",
        nestedContext: {
          profile: { email: "nested@example.com", firstName: "Nested" },
        },
      },
    });
  });
});
