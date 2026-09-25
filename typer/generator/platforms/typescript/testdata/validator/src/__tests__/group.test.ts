import { beforeEach, describe, expect, it } from "vitest";
import { RudderTyper } from "../RudderTyper/RudderTyper.ts";
import { interceptor } from "./eventInterceptor.ts";
import { freshAnalytics } from "./session.ts";

describe("RudderTyper.group", () => {
  let typer: RudderTyper;

  beforeEach(async () => {
    const analytics = await freshAnalytics();
    typer = new RudderTyper(() => analytics);
  });

  it("dispatches a group event with groupId and traits in the event's traits field", async () => {
    typer.group("company-xyz-789", { active: true, status: "active" });

    const [event] = await interceptor.waitForEvents(1);

    expect(event.type).toBe("group");
    expect(event.groupId).toBe("company-xyz-789");
    expect(event.traits).toEqual({ active: true, status: "active" });
  });

  it("dispatches a group event with required-only traits", async () => {
    typer.group("org-456", { active: false });

    const [event] = await interceptor.waitForEvents(1);

    expect(event.type).toBe("group");
    expect(event.groupId).toBe("org-456");
    expect(event.traits).toEqual({ active: false });
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
