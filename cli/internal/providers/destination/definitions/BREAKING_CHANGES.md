# Destination Breaking Changes

This document lists breaking changes to destination `config` between releases of `rudder-cli`: changes that require users to edit a destination spec when they upgrade. Breaking changes elsewhere in the CLI are in the root [BREAKING_CHANGES.md](../../../../../BREAKING_CHANGES.md).

A destination type's history starts at the release that made it available. Changes from before that release aren't listed.

---

## v0.26.0 — 2026-09-17

### Key removal: ActiveCampaign `use_native_sdk`

**Scope:** spec

**Types:** `active_campaign`

**Deprecated in:** v0.26.0, the release that removed it

**Why:** `use_native_sdk` is the older way to choose device mode. Rudder CLI deprecates it in favor of `connection_mode` and no longer accepts the key, though destinations set up the older way still use it. A spec that still sets it fails validation with:

```text
unknown config field "use_native_sdk"
```

**Migration:** Remove `use_native_sdk`. To load ActiveCampaign's native SDK, set `connection_mode.web` to `device` or `hybrid`.

<!-- Convention for future entries:
     - Follow the root BREAKING_CHANGES.md convention, plus a required Types field:
       the affected type values, in alphabetical order.
     - Add unreleased entries under ## Upcoming (unreleased) above the newest
       release, creating the section if it's absent.
     - List every entry in the root file's Destination config section, which links
       to its heading, so renaming a heading means updating that link.
-->
