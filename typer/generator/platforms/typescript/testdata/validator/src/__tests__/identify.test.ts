import { beforeEach, describe, expect, it } from "vitest";
import { RudderTyper } from "../RudderTyper/RudderTyper.ts";
import { interceptor } from "./eventInterceptor.ts";
import { freshAnalytics } from "./session.ts";

describe("RudderTyper.identify", () => {
  let typer: RudderTyper;

  beforeEach(async () => {
    const analytics = await freshAnalytics();
    typer = new RudderTyper(() => analytics);
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
