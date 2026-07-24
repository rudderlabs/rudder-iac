import { RudderAnalytics } from "@rudderstack/analytics-js/bundled";
import { beforeEach, describe, expect, it } from "vitest";
import { RudderTyper } from "../RudderTyper/RudderTyper.ts";
import {
  TEST_CONFIG_BE_URL,
  TEST_DATA_PLANE_URL,
  TEST_WRITE_KEY,
  interceptor,
} from "./eventInterceptor.ts";

describe("RudderTyper.page", () => {
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

  it("dispatches a page event with name and typed properties", async () => {
    typer.page("Dashboard", {
      profile: {
        email: "user@example.com",
        firstName: "Alice",
        lastName: "Johnson",
      },
    });

    const [event] = await interceptor.waitForEvents(1);

    expect(event.type).toBe("page");
    expect(event.name).toBe("Dashboard");
    expect(event.properties).toMatchObject({
      profile: {
        email: "user@example.com",
        firstName: "Alice",
        lastName: "Johnson",
      },
      name: "Dashboard",
    });
  });

  it("dispatches a page event with category, name, and properties", async () => {
    typer.page("Main Navigation", "Dashboard", {
      profile: {
        email: "user@example.com",
        firstName: "Alice",
        lastName: "Johnson",
      },
    });

    const [event] = await interceptor.waitForEvents(1);

    expect(event.type).toBe("page");
    expect(event.name).toBe("Dashboard");
    expect(event.category).toBe("Main Navigation");
    expect(event.properties).toMatchObject({
      profile: {
        email: "user@example.com",
        firstName: "Alice",
        lastName: "Johnson",
      },
      name: "Dashboard",
      category: "Main Navigation",
    });
  });

  it("dispatches a page event with properties only (no name)", async () => {
    typer.page({
      profile: {
        email: "anon@example.com",
        firstName: "Anonymous",
      },
    });

    const [event] = await interceptor.waitForEvents(1);

    expect(event.type).toBe("page");
    expect(event.properties).toMatchObject({
      profile: {
        email: "anon@example.com",
        firstName: "Anonymous",
      },
    });
  });

  it("merges the ruddertyper context into the dispatched page event", async () => {
    typer.page("Settings", {
      profile: { email: "ctx@example.com", firstName: "Ctx" },
    });

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
