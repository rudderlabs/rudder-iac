import { RudderAnalytics } from "@rudderstack/analytics-js/bundled";
import { beforeEach, describe, expect, it } from "vitest";
import { RudderTyper } from "../RudderTyper/RudderTyper.ts";
import {
  TEST_CONFIG_BE_URL,
  TEST_DATA_PLANE_URL,
  TEST_WRITE_KEY,
  interceptor,
} from "./eventInterceptor.ts";

describe("RudderTyper.track — variant discriminated unions", () => {
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

  // ---- trackEventWithVariants (each event-level variant case) ----

  it("trackEventWithVariants dispatches the mobile variant with discriminator and case-specific fields", async () => {
    typer.trackEventWithVariants({
      deviceType: "mobile",
      profile: {
        email: "mobile.user@example.com",
        firstName: "Hannah",
        lastName: "Smith",
      },
      tags: ["mobile", "app-user"],
    });

    const [event] = await interceptor.waitForEvents(1);

    expect(event.type).toBe("track");
    expect(event.event).toBe("Event With Variants");
    expect(event.properties).toEqual({
      deviceType: "mobile",
      profile: {
        email: "mobile.user@example.com",
        firstName: "Hannah",
        lastName: "Smith",
      },
      tags: ["mobile", "app-user"],
    });
  });

  it("trackEventWithVariants dispatches the desktop variant with discriminator and case-specific fields", async () => {
    typer.trackEventWithVariants({
      deviceType: "desktop",
      firstName: "Ian",
      lastName: "Walker",
      profile: {
        email: "desktop.user@example.com",
        firstName: "Ian",
      },
    });

    const [event] = await interceptor.waitForEvents(1);

    expect(event.type).toBe("track");
    expect(event.event).toBe("Event With Variants");
    expect(event.properties).toEqual({
      deviceType: "desktop",
      firstName: "Ian",
      lastName: "Walker",
      profile: {
        email: "desktop.user@example.com",
        firstName: "Ian",
      },
    });
  });

  it("trackEventWithVariants dispatches the default variant for an unmatched device type", async () => {
    typer.trackEventWithVariants({
      deviceType: "tablet",
      profile: {
        email: "tablet.user@example.com",
        firstName: "Janet",
      },
      untypedField: { custom: "data", count: 42 },
    });

    const [event] = await interceptor.waitForEvents(1);

    expect(event.type).toBe("track");
    expect(event.event).toBe("Event With Variants");
    expect(event.properties).toEqual({
      deviceType: "tablet",
      profile: {
        email: "tablet.user@example.com",
        firstName: "Janet",
      },
      untypedField: { custom: "data", count: 42 },
    });
  });

  // ---- CustomTypeFeatureConfig (each variant case as a property) ----

  it("trackUserSignedUp serializes each CustomTypeFeatureConfig variant with correct discriminator", async () => {
    typer.trackUserSignedUp({
      active: true,
      profile: {
        email: "feature.enabled@example.com",
        firstName: "Premium",
        lastName: "User",
      },
      featureConfig: { featureFlag: true, age: 30 },
    });

    typer.trackUserSignedUp({
      active: true,
      profile: {
        email: "feature.disabled@example.com",
        firstName: "Free",
        lastName: "User",
      },
      featureConfig: { featureFlag: false, firstName: "some-name" },
    });

    typer.trackUserSignedUp({
      active: true,
      profile: {
        email: "feature.beta@example.com",
        firstName: "Beta",
        lastName: "Tester",
      },
      featureConfig: {
        featureFlag: "beta",
        tags: ["beta-user", "early-access", "experimental"],
      },
    });

    typer.trackUserSignedUp({
      active: true,
      profile: {
        email: "feature.alpha@example.com",
        firstName: "Alpha",
        lastName: "User",
      },
      featureConfig: { featureFlag: "alpha" },
    });

    const events = await interceptor.waitForEvents(4);

    expect(events.map((e) => e.type)).toEqual(["track", "track", "track", "track"]);
    expect(events.map((e) => e.properties)).toEqual([
      {
        active: true,
        profile: {
          email: "feature.enabled@example.com",
          firstName: "Premium",
          lastName: "User",
        },
        featureConfig: { featureFlag: true, age: 30 },
      },
      {
        active: true,
        profile: {
          email: "feature.disabled@example.com",
          firstName: "Free",
          lastName: "User",
        },
        featureConfig: { featureFlag: false, firstName: "some-name" },
      },
      {
        active: true,
        profile: {
          email: "feature.beta@example.com",
          firstName: "Beta",
          lastName: "Tester",
        },
        featureConfig: {
          featureFlag: "beta",
          tags: ["beta-user", "early-access", "experimental"],
        },
      },
      {
        active: true,
        profile: {
          email: "feature.alpha@example.com",
          firstName: "Alpha",
          lastName: "User",
        },
        featureConfig: { featureFlag: "alpha" },
      },
    ]);
  });

  // ---- CustomTypeUserAccess (each variant case as a property) ----

  it("trackUserSignedUp serializes each CustomTypeUserAccess variant with correct discriminator", async () => {
    typer.trackUserSignedUp({
      active: true,
      profile: {
        email: "access.active@example.com",
        firstName: "Active",
        lastName: "User",
      },
      userAccess: { active: true, email: "active@example.com" },
    });

    typer.trackUserSignedUp({
      active: true,
      profile: {
        email: "access.inactive@example.com",
        firstName: "Inactive",
        lastName: "User",
      },
      userAccess: { active: false, status: "suspended" },
    });

    typer.trackUserSignedUp({
      active: true,
      profile: {
        email: "default.user@example.com",
        firstName: "Default",
        lastName: "Case",
      },
      userAccess: { active: true },
    });

    const events = await interceptor.waitForEvents(3);

    expect(events.map((e) => e.type)).toEqual(["track", "track", "track"]);
    expect(events.map((e) => e.properties)).toEqual([
      {
        active: true,
        profile: {
          email: "access.active@example.com",
          firstName: "Active",
          lastName: "User",
        },
        userAccess: { active: true, email: "active@example.com" },
      },
      {
        active: true,
        profile: {
          email: "access.inactive@example.com",
          firstName: "Inactive",
          lastName: "User",
        },
        userAccess: { active: false, status: "suspended" },
      },
      {
        active: true,
        profile: {
          email: "default.user@example.com",
          firstName: "Default",
          lastName: "Case",
        },
        userAccess: { active: true },
      },
    ]);
  });

  // ---- Nested variants (event variant containing a property variant) ----

  it("trackEventWithVariants serializes nested pageContext variants within each event variant", async () => {
    typer.trackEventWithVariants({
      deviceType: "desktop",
      firstName: "Nested",
      profile: {
        email: "nested.search@example.com",
        firstName: "Nested",
      },
      pageContext: { pageType: "search", query: "discriminated unions" },
    });

    typer.trackEventWithVariants({
      deviceType: "mobile",
      profile: {
        email: "nested.product@example.com",
        firstName: "Shopper",
      },
      pageContext: { pageType: "product", productId: "prod-456" },
    });

    typer.trackEventWithVariants({
      deviceType: "tablet",
      profile: {
        email: "nested.default@example.com",
        firstName: "Fallback",
      },
      pageContext: {
        pageType: "settings",
        pageData: { theme: "dark", language: "en" },
      },
    });

    typer.trackEventWithVariants({
      deviceType: "mobile",
      profile: {
        email: "nested.home@example.com",
        firstName: "HomeUser",
      },
      pageContext: { pageType: "home" },
    });

    const events = await interceptor.waitForEvents(4);

    expect(events.map((e) => e.type)).toEqual(["track", "track", "track", "track"]);
    expect(events.map((e) => e.event)).toEqual([
      "Event With Variants",
      "Event With Variants",
      "Event With Variants",
      "Event With Variants",
    ]);
    expect(events.map((e) => e.properties)).toEqual([
      {
        deviceType: "desktop",
        firstName: "Nested",
        profile: {
          email: "nested.search@example.com",
          firstName: "Nested",
        },
        pageContext: { pageType: "search", query: "discriminated unions" },
      },
      {
        deviceType: "mobile",
        profile: {
          email: "nested.product@example.com",
          firstName: "Shopper",
        },
        pageContext: { pageType: "product", productId: "prod-456" },
      },
      {
        deviceType: "tablet",
        profile: {
          email: "nested.default@example.com",
          firstName: "Fallback",
        },
        pageContext: {
          pageType: "settings",
          pageData: { theme: "dark", language: "en" },
        },
      },
      {
        deviceType: "mobile",
        profile: {
          email: "nested.home@example.com",
          firstName: "HomeUser",
        },
        pageContext: { pageType: "home" },
      },
    ]);
  });
});
