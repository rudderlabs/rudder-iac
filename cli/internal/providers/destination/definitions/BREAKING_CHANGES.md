# Destination Breaking Changes

This document lists breaking changes to destination `config` between releases of `rudder-cli`: changes that require users to edit a destination spec when they upgrade. Breaking changes elsewhere in the CLI are in the root [BREAKING_CHANGES.md](../../../../../BREAKING_CHANGES.md).

A destination type's history starts at the first release that made it available without an experimental flag.

---

## v0.28.0 — 2026-10-08

### Source type added: `warehouse`

**Scope:** spec

**Types:** `active_campaign`, `am`, `attentive_tag`, `bqstream`, `braze`, `customerio`, `facebook_conversions`, `facebook_pixel`, `ga4`, `gcs`, `hs`, `iterable`, `mp`, `posthog`, `s3`, `tiktok_ads`, `webhook`

**Why:** Rudder CLI doesn't manage Reverse ETL connections yet, but it now manages a destination's `warehouse` entries in `connection_mode` and `consent_management`, which the dashboard writes when you connect a Reverse ETL source. Earlier releases dropped these entries when reading a destination, so a spec imported by one of them lacks the entries but showed no diff. Now that same spec plans an update that removes them, along with any warehouse consent settings.

**Migration:** For each destination connected to a Reverse ETL source, add `connection_mode.warehouse: cloud` to its spec, and any `consent_management.warehouse` entries it has, before your next `apply`. `apply --dry-run` shows the entries an update would remove. Re-importing doesn't help: `import workspace` skips destinations the project already manages.

### Keys added: Customer.io v2 web client

**Scope:** spec

**Types:** `customerio`

**Why:** Rudder CLI now manages `sdk_version.web`, `write_key.web` and `anonymous_in_app.web`, which the dashboard writes for Customer.io's v2 web client. Earlier releases dropped these keys when reading a destination, so a spec imported by one of them lacks the keys but showed no diff. Now that same spec plans an update that removes them, and a destination on the v2 web client falls back to v1.

**Migration:** Add the destination's `sdk_version.web`, `write_key.web` and `anonymous_in_app.web` values from the dashboard to its spec before your next `apply`. `apply --dry-run` shows the keys an update would remove.

---

## v0.27.0 — 2026-09-22

### Default changed: Customer.io `api_version`

**Scope:** spec

**Types:** `customerio`

**Why:** `api_version` now defaults to `v2` instead of `v1`, and Rudder CLI applies defaults before it validates a spec. `v2` requires `user_id_identifier_type`, so a spec that sets neither key, which ran on `v1` before, now fails validation with:

```text
'user_id_identifier_type' is required when 'api_version' is v2
```

Setting only `user_id_identifier_type` passes validation but moves the destination to the v2 API.

**Migration:** Set `api_version: v1` to stay on the v1 API, or set `user_id_identifier_type` to move to `v2`.

---

## v0.26.0 — 2026-09-17

### Source type added: `warehouse` on HTTP Webhook

**Scope:** spec

**Types:** `http`

**Why:** Rudder CLI doesn't manage Reverse ETL connections yet, but it now manages an `http` destination's `warehouse` entries in `connection_mode` and `consent_management`, which the dashboard writes when you connect a Reverse ETL source. Earlier releases dropped these entries when reading a destination, so a spec imported by one of them lacks the entries but showed no diff. Now that same spec plans an update that removes them, along with any warehouse consent settings.

**Migration:** For each destination connected to a Reverse ETL source, add `connection_mode.warehouse: cloud` to its spec, and any `consent_management.warehouse` entries it has, before your next `apply`. `apply --dry-run` shows the entries an update would remove. Re-importing doesn't help: `import workspace` skips destinations the project already manages.

### Key removal: ActiveCampaign `use_native_sdk`

**Scope:** spec

**Types:** `active_campaign`

**Deprecated in:** none. Removed in v0.26.0 with no deprecation period.

**Why:** `use_native_sdk` was the older way to choose device mode. Destinations still store it as `useNativeSDK`, but the backend now derives that from `connectionMode`, so Rudder CLI sets device mode through `connection_mode` alone. A spec that still sets the key fails validation with:

```text
unknown config field "use_native_sdk"
```

**Migration:** Remove `use_native_sdk` and set `connection_mode.web`: `device` or `hybrid` to keep loading ActiveCampaign's native SDK, `cloud` otherwise. Earlier releases accepted `use_native_sdk.web` in place of a `connection_mode.web` entry, so a spec that connects a web source without one now fails with:

```text
destination '<id>' config has no 'connection_mode' entry for source type 'web'
```

<!-- Convention for future entries:
     - Follow the root BREAKING_CHANGES.md convention, plus a required Types field:
       the affected type values, in alphabetical order.
     - Add unreleased entries under ## Upcoming (unreleased) above the newest
       release, creating the section if it's absent.
-->
