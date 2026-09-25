// Wire contract for the generated client.
//
// Every assertion here is written from the TRACKING PLAN's point of view: given
// a plan and a call, what must the event on the wire look like? None of it is
// written from what the generator currently emits.
//
// That distinction is the point. The existing suites were written by reading
// the generated output back, so when the generator put keys in the wrong place
// the tests recorded that as correct. Two shipped bugs came out of it: the
// camelCased property keys fixed in v0.24.0, and the context.traits routing
// these tests cover.
//
// Two rules keep that from happening again:
//   1. Assert against the plan, never against generated output.
//   2. Assert on a FOLLOWING event too, not only the call under test. Trait
//      persistence is invisible if you only inspect the call you just made,
//      which is why the identify bug reached production.
import type { RudderAnalytics } from "@rudderstack/analytics-js/bundled";
import { beforeEach, describe, expect, it } from "vitest";
import { RudderTyper } from "../RudderTyper/RudderTyper.ts";
import { RudderTyper as IdentitySections } from "../RudderTyper/IdentitySections.ts";
import { RudderTyper as EmptyIdentity } from "../RudderTyper/EmptyIdentity.ts";
import { interceptor } from "./eventInterceptor.ts";
import { freshAnalytics, traitsOf } from "./session.ts";

describe("wire contract: identify with identity_section context.traits", () => {
  let typer: IdentitySections;

  beforeEach(async () => {
    const analytics = await freshAnalytics();
    typer = new IdentitySections(() => analytics as RudderAnalytics);
  });

  it("puts the traits on the identify event", async () => {
    typer.identify("user-1", { email: "user@example.com", active: true });

    const [event] = await interceptor.waitForEvents(1);

    expect(event.type).toBe("identify");
    expect(traitsOf(event)).toEqual({ email: "user@example.com", active: true });
  });

  it("carries the traits on events that follow the identify", async () => {
    typer.identify("user-1", { email: "user@example.com", active: true });
    typer.trackUserSignedUp({ active: true });

    const [, followUp] = await interceptor.waitForEvents(2);

    // An identify tells the SDK who the user is. Every later event has to carry
    // those traits, otherwise the plan's identify rule describes a single event
    // rather than the session it is supposed to establish.
    expect(followUp.type).toBe("track");
    expect(traitsOf(followUp)).toEqual({ email: "user@example.com", active: true });
  });

  it("does not clear established traits when a later call omits them", async () => {
    typer.identify("user-1", { email: "user@example.com", active: true });
    (typer as unknown as { identify: (id: string) => void }).identify("user-1");
    typer.trackUserSignedUp({ active: true });

    const [, , followUp] = await interceptor.waitForEvents(3);

    // identify("user-1") means "this is who the user is", not "forget what you
    // knew about them". The JS SDK resets stored traits when the argument is
    // undefined but merges when it is an object, so the generated call has to
    // pass {} rather than nothing. v1 did this as `traits || {}`; v2 dropped it.
    expect(traitsOf(followUp)).toEqual({ email: "user@example.com", active: true });
  });

  it("merges a partial later identify into the established traits", async () => {
    typer.identify("user-1", { email: "user@example.com", active: true });
    (typer as unknown as { identify: (id: string, t: unknown) => void }).identify("user-1", { active: false });
    typer.trackUserSignedUp({ active: true });

    const [, , followUp] = await interceptor.waitForEvents(3);

    // Merging is the SDK's job, not the generated client's. Asserted here so a
    // future change cannot quietly reimplement it in generated code.
    expect(traitsOf(followUp)).toEqual({ email: "user@example.com", active: false });
  });

  it("carries the traits on later events for the anonymous overload too", async () => {
    (typer as unknown as { identify: (t: unknown) => void }).identify({
      email: "anon@example.com",
      active: false,
    });
    typer.trackUserSignedUp({ active: true });

    const [, followUp] = await interceptor.waitForEvents(2);

    expect(traitsOf(followUp)).toEqual({ email: "anon@example.com", active: false });
  });
});

describe("wire contract: group with identity_section traits", () => {
  let typer: IdentitySections;

  beforeEach(async () => {
    const analytics = await freshAnalytics();
    typer = new IdentitySections(() => analytics as RudderAnalytics);
  });

  it("puts group traits in the event's traits field", async () => {
    typer.group("org-1", { active: true, status: "active" });

    const [event] = await interceptor.waitForEvents(1);

    expect(event.type).toBe("group");
    expect(event.groupId).toBe("org-1");
    expect(event.traits).toEqual({ active: true, status: "active" });
  });
});

describe("wire contract: group with identity_section context.traits", () => {
  let typer: RudderTyper;

  beforeEach(async () => {
    const analytics = await freshAnalytics();
    typer = new RudderTyper(() => analytics);
  });

  it("puts group traits in the event's traits field", async () => {
    typer.group("org-1", { active: true, status: "active" });

    const [event] = await interceptor.waitForEvents(1);

    // A group event's traits belong in `traits`. That is what the warehouse
    // `groups` table and downstream group calls read. The identity_section a
    // plan uses to describe the schema must not relocate the payload.
    expect(event.type).toBe("group");
    expect(event.traits).toEqual({ active: true, status: "active" });
  });

  it("does not mix group traits into the user's traits", async () => {
    typer.identify("user-1", { email: "user@example.com", active: true });
    typer.group("org-1", { active: false, status: "suspended" });

    const [, groupEvent] = await interceptor.waitForEvents(2);

    // User traits and group traits are different identity scopes. Merging them
    // into one bag corrupts both: here `active` exists in both schemas, so one
    // silently overwrites the other.
    expect(traitsOf(groupEvent)).toEqual({ email: "user@example.com", active: true });
  });

  it("does not discard traits the caller supplied in options.context", async () => {
    // With a prior identify in place, this also pins the merge order against
    // the SDK's stored user traits rather than only the empty-session case.
    typer.identify("user-1", { email: "user@example.com", active: true });
    (typer as unknown as { group: (id: string, t: unknown, o: unknown) => void }).group(
      "org-1",
      { active: true },
      { context: { traits: { tenant: "acme" } } },
    );

    const [, event] = await interceptor.waitForEvents(2);

    expect(traitsOf(event)).toEqual({ email: "user@example.com", active: true, tenant: "acme" });
  });
});

describe("wire contract: track property keys", () => {
  let typer: RudderTyper;

  beforeEach(async () => {
    const analytics = await freshAnalytics();
    typer = new RudderTyper(() => analytics);
  });

  // Guards the v0.24.0 fix. Keys on the wire are the plan's, at every depth.
  it("sends the plan's property names, not the camelCased identifiers", async () => {
    typer.trackUserSignedUp({
      active: true,
      deviceType: "mobile",
      profile: { email: "a@example.com", firstName: "Ada", lastName: "L" },
    });

    const [event] = await interceptor.waitForEvents(1);

    expect(event.properties).toEqual({
      active: true,
      device_type: "mobile",
      profile: { email: "a@example.com", first_name: "Ada", last_name: "L" },
    });
  });
});

describe("wire contract: caller-supplied options survive", () => {
  let typer: RudderTyper;

  beforeEach(async () => {
    const analytics = await freshAnalytics();
    typer = new RudderTyper(() => analytics);
  });

  it("keeps caller context keys and integrations alongside the ruddertyper context", async () => {
    (typer as unknown as { trackUserSignedUp: (p: unknown, o: unknown) => void }).trackUserSignedUp(
      { active: true, profile: { email: "a@example.com", firstName: "Ada" } },
      { context: { custom: "keep-me" }, integrations: { All: false, Amplitude: true } },
    );

    const [event] = await interceptor.waitForEvents(1);
    const context = event.context as Record<string, unknown>;

    expect(context.custom).toBe("keep-me");
    expect(event.integrations).toEqual({ All: false, Amplitude: true });
    expect(context.ruddertyper).toEqual({
      platform: "typescript",
      rudderCLIVersion: "1.0.0",
      trackingPlanId: "plan_12345",
      trackingPlanVersion: 13,
    });
  });
});

describe("wire contract: identify and group rules with no properties", () => {
  let typer: EmptyIdentity;

  beforeEach(async () => {
    const analytics = await freshAnalytics();
    typer = new EmptyIdentity(() => analytics as RudderAnalytics);
  });

  it("identify does not clear traits the caller never supplied", async () => {
    const analytics = await freshAnalytics();
    // Traits established outside this plan — an untyped call, another plan, or
    // an earlier session. The generated identify takes no traits parameter, so
    // the caller has no way to protect them.
    analytics.identify("user-1", { email: "user@example.com" });
    typer = new EmptyIdentity(() => analytics as RudderAnalytics);

    typer.identify("user-1");
    typer.trackUserSignedUp({ active: true });

    const [, , followUp] = await interceptor.waitForEvents(3);

    expect(traitsOf(followUp)).toEqual({ email: "user@example.com" });
  });

  it("group does not clear group traits the caller never supplied", async () => {
    const analytics = await freshAnalytics();
    // Group traits established outside this plan. The SDK sources a group
    // event's traits from stored state, so a later group call that resets that
    // state empties the field every consumer reads.
    analytics.group("org-1", { plan: "enterprise" });
    typer = new EmptyIdentity(() => analytics as RudderAnalytics);

    typer.group("org-1");

    const [, second] = await interceptor.waitForEvents(2);

    expect(second.type).toBe("group");
    expect(second.traits).toEqual({ plan: "enterprise" });
  });
});
