# Destination Breaking Changes

This document lists breaking changes to destination `config` between releases of `rudder-cli`: changes that require users to edit a destination spec, or re-import it, when they upgrade. Breaking changes elsewhere in the CLI are in the root [BREAKING_CHANGES.md](../../../../../BREAKING_CHANGES.md).

A destination type's history starts at the release that made it available. Changes from before that release aren't listed.

---

## v0.28.0

### Re-import destinations that already have a Reverse ETL connection

**Scope:** spec

**Types:** `active_campaign`, `am`, `attentive_tag`, `braze`, `bqstream`, `customerio`, `facebook_conversions`, `facebook_pixel`, `ga4`, `gcs`, `hs`, `iterable`, `mp`, `posthog`, `s3`, `tiktok_ads`, `webhook`

**Why:** the CLI now maps `connectionMode.warehouse` and `consentManagement.warehouse` on these types. Before, it dropped them. The webapp writes these keys when you connect a warehouse source, so specs imported earlier don't have them.

For those destinations, `apply --dry-run` now shows an update that removes the keys, and `apply` strips them along with any warehouse consent settings. To avoid this, re-import the affected destinations with `rudder-cli import workspace` before your next `apply`.

## v0.26.0

### Re-import HTTP Webhook destinations that already have a Reverse ETL connection

**Scope:** spec

**Types:** `http`

**Why:** the same change as in v0.28.0, made earlier for HTTP Webhook alone. The CLI started mapping `connectionMode.warehouse` and `consentManagement.warehouse` on `http` destinations, which it dropped before, so specs imported earlier don't have them and `apply` strips them.

Re-import the affected destinations with `rudder-cli import workspace` before your next `apply`.

### `use_native_sdk` removed from ActiveCampaign

**Scope:** spec

**Types:** `active_campaign`

**Why:** `use_native_sdk` is deprecated. RudderStack decides whether to load a destination's native SDK from `connection_mode`, so the CLI no longer models the key. A spec that still sets it fails validation with:

```text
unknown config field "use_native_sdk"
```

Remove the key. `connection_mode.web` decides whether ActiveCampaign's native SDK loads.

**Before:**

```yaml
config:
  use_native_sdk:
    web: true
  connection_mode:
    web: hybrid
```

**After:**

```yaml
config:
  connection_mode:
    web: hybrid
```
