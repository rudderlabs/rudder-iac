# rETL Warehouse Metadata Inventory

Which registered destination definitions declare `warehouse`, and with what rETL
metadata. The derivation is the one in
[source-type-mapping.md](source-type-mapping.md) "rETL metadata" and
"Per-source-type connect-time required keys".

- **Provenance**: every definition this backfill changed was derived from
  `rudderlabs/rudder-integrations-config` `develop` at
  `5f11c22c4bcad9ecca4ec37ad31bb7605b72b079` (2026-09-24) and is CLI definition
  version `1`; everything else listed was read at the same revision. DEX-821 read
  `9fa7b26`, 37 commits earlier. Between the two, no registered destination
  changed `supportedSourceTypes`, `supportedConnectionModes`, `syncBehaviours` or
  `supportsVisualMapper`. Only `customerio` changed a connectionMode-gated
  schema branch (see below).
- **Verified or not**: the registration block in
  `cli/internal/app/dependencies.go` `newDestinationRegistry` decides it; the
  last Reproduce command lists both sets.
- **Flow**: `retl/connection/flow.go` `ClassifyFlow` (mirrors config-backend
  `determineFlowTypeFromRequest`).
- **`supportedSourcesValidation`**: none. A recursive key walk over all 50
  registered destinations' db-config.json found no occurrence at this revision.

## Reproduce

```sh
SHA=5f11c22c4bcad9ecca4ec37ad31bb7605b72b079
raw() { gh api "repos/rudderlabs/rudder-integrations-config/contents/src/configurations/destinations/$1?ref=$SHA" -H 'Accept: application/vnd.github.raw'; }

# db-config: warehouse support, its modes and the two rETL fields.
raw "$TYPE/db-config.json" | jq -c '{
  warehouse: (.config.supportedSourceTypes | index("warehouse") != null),
  modes: .config.supportedConnectionModes.warehouse,
  syncBehaviours: (.config | if has("syncBehaviours") then .syncBehaviours else "absent" end),
  supportsVisualMapper: .config.supportsVisualMapper}'

# schema.json: the connectionMode-conditioned branches with required keys.
raw "$TYPE/schema.json" | jq -c '.configSchema.allOf[]?
  | select((.if | tostring | test("connectionMode")) and .then.required != null)
  | {if, required: .then.required}'

# Verified vs unverified, from the repo root: the types listed before the
# UnverifiedDestinations marker are verified, the rest unverified.
sed -n '/^func newDestinationRegistry/,/^}/p' cli/internal/app/dependencies.go |
  grep -oE 'UnverifiedDestinations|registering [a-z0-9_]+ destination' |
  sed -E 's/^registering ([a-z0-9_]+) destination$/\1/'
```

`$TYPE` is the definition's local type, except `linkedin_ads`, whose upstream
directory is `linkedIn_ads`.

## Verified definitions that declare `warehouse`

`active_campaign`, `am`, `attentive_tag`, `bqstream`, `braze`, `customerio`,
`facebook_conversions`, `facebook_pixel`, `ga4`, `gcs`, `hs`, `http` (DEX-821),
`iterable`, `mp`, `posthog`, `s3`, `tiktok_ads`, `webhook`.

Each declares `warehouse` with mode `cloud`, as every upstream
`supportedConnectionModes.warehouse` does. Unless noted below, upstream omits
`syncBehaviours`, so the field stays unset and takes the backend's
`upsert`/`mirror`/`full` fallback, and the definition declares no visual mapper
and no `ConnectionRequiredKeys[warehouse][cloud]`.

- Visual mapper: `am`, `braze`, `hs`, `iterable`.
- `braze` requires `rest_api_key` (allOf branch naming `warehouse: cloud`).
- `facebook_pixel` requires nothing: its only branch (negated `web: device`)
  does not name `connectionMode.warehouse`.
- `customerio`: `upsert`, `mirror`. It runs its own flow, which rETL refuses,
  and its upstream `siteID`/`apiKey` requiredness changed since `9fa7b26`.
- `iterable`'s only connectionMode branch (top-level `anyOf`, `packageName`)
  applies to `web: device` and does not reach `warehouse`.

## Verified definitions without `warehouse`

Upstream lists no `warehouse` source type, so rETL cannot reach them: `bq`,
`googleads`, `postgres`, `rs`, `s3_datalake`, `snowflake`. Use one of these, not a
backfilled destination, as a negative example in tests and docs.

## Unverified definitions

Unverified definitions keep their experimental gate; backfilling one is a
separate change, and so is promotion to verified.

- Declared already, with metadata completed in DEX-821:
  `bingads_offline_conversions` (`mirror`, visual mapper) and
  `customerio_audience` (`mirror`, destination-specific flow).
- Eligible upstream, deferred: `adj`, `adobe_analytics`, `confluent_cloud`, `ga`
  (visual mapper), `google_adwords_offline_conversions`, `googlepubsub`,
  `googlesheets`, `intercom` (visual mapper; its connectionMode branches do not
  name `warehouse`), `kafka`, `kinesis`, `linkedin_ads`, `marketo` (visual
  mapper), `redis`, `salesforce` (visual mapper), `slack`, `statsig`, `zendesk`.
  None overrides `syncBehaviours`.
- Not eligible (no `warehouse` upstream): `firebase`, `gtm`,
  `linkedin_insight_tag`, `qualtrics`, `sentry`, `snowpipe_streaming`, `vwo`.
