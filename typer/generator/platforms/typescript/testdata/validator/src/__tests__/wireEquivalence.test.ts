// Equivalence harness for DAW-3732 (throwaway — delete once the option is picked).
//
// Asserts ONE thing: whatever the generated client is called with, the payload
// that leaves the SDK uses the TRACKING PLAN's property keys, at every depth.
//
// Inputs below are written in plan keys. Option 2 (verbatim fields) takes them
// as-is. Option 1 (camelCase fields + runtime remap) needs the call site
// camelCased, so EQUIV_CAMEL=1 camelCases the INPUT only — never the expectation.
// Same file, same expectations, both branches: that is the equivalence.
//
// Calls go through `as any` on purpose. Field-name typing is already covered by
// the real suite; this harness is only about what goes on the wire.
import { RudderAnalytics } from "@rudderstack/analytics-js/bundled";
import { beforeEach, describe, expect, it } from "vitest";
import { RudderTyper } from "../RudderTyper/RudderTyper.ts";
import {
  TEST_CONFIG_BE_URL,
  TEST_DATA_PLANE_URL,
  TEST_WRITE_KEY,
  interceptor,
} from "./eventInterceptor.ts";

const CAMEL_INPUT = process.env.EQUIV_CAMEL === "1";

// Mirrors FormatPropertyName for the plain snake_case keys used below.
const camel = (k: string) => k.replace(/_([a-z0-9])/g, (_, c: string) => c.toUpperCase());

function adapt<T>(value: T): T {
  if (!CAMEL_INPUT) return value;
  if (Array.isArray(value)) return value.map(adapt) as unknown as T;
  if (value === null || typeof value !== "object") return value;
  return Object.fromEntries(
    Object.entries(value as Record<string, unknown>).map(([k, v]) => [camel(k), adapt(v)]),
  ) as T;
}

// ---- fixtures, in plan keys ----

const USER_SIGNED_UP = {
  active: true,
  device_type: "mobile",
  // nested object behind a custom type
  profile: { email: "a@example.com", first_name: "Bob", last_name: "Williams" },
  // ARRAY of objects — Option 1 emits a flat string entry for this key
  profile_list: [
    { email: "b@example.com", first_name: "Ann" },
    { email: "c@example.com", first_name: "Cy", last_name: "Doe" },
  ],
  // union-typed custom type — Option 1 cannot pick a branch at runtime
  feature_config: { feature_flag: true },
  // two levels of nesting
  context: {
    ip_address: "10.0.0.1",
    nested_context: {
      profile: { email: "d@example.com", first_name: "Dee" },
    },
  },
};

const EVENT_WITH_VARIANTS = {
  device_type: "mobile",
  profile: { email: "e@example.com", first_name: "Eve" },
  page_context: { page_type: "product", product_id: "sku-1" },
};

const PAGE_PROPERTIES = {
  profile: { email: "f@example.com", first_name: "Fay", last_name: "Ng" },
};

const IDENTIFY_TRAITS = { active: true, email: "g@example.com" };
const GROUP_TRAITS = { active: true, status: "active" };

// Every key that may appear on the wire, at any depth, for the fixtures above.
const PLAN_KEYS = new Set([
  "active", "device_type", "profile", "email", "first_name", "last_name",
  "profile_list", "feature_config", "feature_flag", "context", "ip_address",
  "nested_context", "page_context", "page_type", "product_id", "status",
]);

function offendingKeys(value: unknown, path = "", out: string[] = []): string[] {
  if (Array.isArray(value)) {
    value.forEach((v, i) => offendingKeys(v, `${path}[${i}]`, out));
    return out;
  }
  if (value === null || typeof value !== "object") return out;
  for (const [k, v] of Object.entries(value as Record<string, unknown>)) {
    if (!PLAN_KEYS.has(k)) out.push(`${path}.${k}`);
    offendingKeys(v, `${path}.${k}`, out);
  }
  return out;
}

describe(`wire keys match the tracking plan (input=${CAMEL_INPUT ? "camelCase" : "plan keys"})`, () => {
  let typer: any;

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
    typer = new RudderTyper(() => analytics) as any;
  });

  it("track: flat, nested, array-of-objects and union-typed properties", async () => {
    typer.trackUserSignedUp(adapt(USER_SIGNED_UP));
    const [event] = await interceptor.waitForEvents(1);

    expect(offendingKeys(event.properties)).toEqual([]);
    expect(event.properties).toEqual(USER_SIGNED_UP);
  });

  it("track: union-typed props (variants)", async () => {
    typer.trackEventWithVariants(adapt(EVENT_WITH_VARIANTS));
    const [event] = await interceptor.waitForEvents(1);

    expect(offendingKeys(event.properties)).toEqual([]);
    expect(event.properties).toEqual(EVENT_WITH_VARIANTS);
  });

  it("page: nested properties", async () => {
    typer.page("Home", adapt(PAGE_PROPERTIES));
    const [event] = await interceptor.waitForEvents(1);
    // The SDK injects its own page context (name, path, referrer, ...), so scope
    // the check to the plan-defined subtree.
    const props = event.properties as Record<string, unknown>;

    expect(offendingKeys(props.profile)).toEqual([]);
    expect(props.profile).toEqual(PAGE_PROPERTIES.profile);
  });

  it("identify: traits", async () => {
    typer.identify("user-1", adapt(IDENTIFY_TRAITS));
    const [event] = await interceptor.waitForEvents(1);

    expect(offendingKeys(event.traits)).toEqual([]);
  });

  it("group: traits", async () => {
    typer.group("group-1", adapt(GROUP_TRAITS));
    const [event] = await interceptor.waitForEvents(1);

    expect(offendingKeys(event.traits ?? {})).toEqual([]);
  });
});
