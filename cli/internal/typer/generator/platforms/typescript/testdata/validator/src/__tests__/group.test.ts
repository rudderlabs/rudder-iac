import { RudderAnalytics } from "@rudderstack/analytics-js/bundled";
import { beforeEach, describe, expect, it } from "vitest";
import { RudderTyper } from "../RudderTyper/RudderTyper.ts";
import {
  TEST_CONFIG_BE_URL,
  TEST_DATA_PLANE_URL,
  TEST_WRITE_KEY,
  interceptor,
} from "./eventInterceptor.ts";

describe("RudderTyper.group", () => {
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

  it("dispatches a group event with groupId and traits routed through context.traits", async () => {
    typer.group("company-xyz-789", { active: true, status: "active" });

    const [event] = await interceptor.waitForEvents(1);

    expect(event.type).toBe("group");
    expect(event.groupId).toBe("company-xyz-789");
    const contextTraits = (event.context as Record<string, unknown>)?.traits;
    expect(contextTraits).toEqual({ active: true, status: "active" });
  });

  it("dispatches a group event with required-only traits", async () => {
    typer.group("org-456", { active: false });

    const [event] = await interceptor.waitForEvents(1);

    expect(event.type).toBe("group");
    expect(event.groupId).toBe("org-456");
    const contextTraits = (event.context as Record<string, unknown>)?.traits;
    expect(contextTraits).toEqual({ active: false });
  });

  it("merges the ruddertyper context into the dispatched group event", async () => {
    typer.group("company-xyz-789", { active: true });

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
