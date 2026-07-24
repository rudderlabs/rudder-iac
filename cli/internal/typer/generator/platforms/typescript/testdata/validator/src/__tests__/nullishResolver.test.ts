import { describe, expect, it } from "vitest";
import { RudderTyper } from "../RudderTyper/RudderTyper.ts";
import { interceptor } from "./eventInterceptor.ts";

describe("RudderTyper nullish analytics resolver", () => {
  it("no-ops direct track wrappers when the resolver returns undefined", () => {
    const typer = new RudderTyper(() => undefined);

    expect(() =>
      typer.trackUserSignedUp({
        active: true,
        profile: { email: "missing-sdk@example.com", firstName: "Missing" },
      }),
    ).not.toThrow();

    expect(interceptor.events()).toEqual([]);
  });

  it("no-ops overloaded dispatcher wrappers when the resolver returns undefined", () => {
    const typer = new RudderTyper(() => undefined);

    expect(() =>
      typer.identify("user-without-sdk", {
        email: "missing-sdk@example.com",
      }),
    ).not.toThrow();

    expect(interceptor.events()).toEqual([]);
  });

  it("also no-ops when the resolver returns null", () => {
    const typer = new RudderTyper(() => null);

    expect(() => typer.group("group-without-sdk", { active: false })).not.toThrow();

    expect(interceptor.events()).toEqual([]);
  });
});
