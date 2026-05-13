import { RudderAnalytics } from "@rudderstack/analytics-js/bundled";
import { beforeEach, describe, expect, it } from "vitest";
import { RudderTyper } from "../RudderTyper/RudderTyper.ts";
import {
  TEST_CONFIG_BE_URL,
  TEST_DATA_PLANE_URL,
  TEST_WRITE_KEY,
  interceptor,
} from "./eventInterceptor.ts";

describe("RudderTyper.identify", () => {
  let typer: RudderTyper;

  beforeEach(async () => {
    const analytics = new RudderAnalytics();
    analytics.load(TEST_WRITE_KEY, TEST_DATA_PLANE_URL, {
      configUrl: TEST_CONFIG_BE_URL,
      logLevel: "ERROR",
      queueOptions: { maxItems: 1, batch: { enabled: false } },
      sessions: { autoTrack: false },
      uaChTrackLevel: "none",
    });
    await new Promise<void>((resolve) => analytics.ready(() => resolve()));
    typer = new RudderTyper(analytics);
  });

  it("dispatches an identify event with the provided userId and traits", async () => {
    typer.identify("user-123-abc", {
      email: "john.doe@example.com",
      active: true,
    });

    const [event] = await interceptor.waitForEvents(1);

    expect(event.type).toBe("identify");
    expect(event.userId).toBe("user-123-abc");
    const traits = (event.context as { traits?: Record<string, unknown> })?.traits;
    expect(traits).toEqual({
      email: "john.doe@example.com",
      active: true,
    });
  });

  it("merges the ruddertyper context into the dispatched event", async () => {
    typer.identify("user-123-abc", { email: "ada@example.com" });

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
