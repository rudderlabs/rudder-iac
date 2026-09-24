# rETL Warehouse Metadata Inventory

Which registered destination definitions declare `warehouse`, and with what rETL
metadata. The derivation is the one in
[source-type-mapping.md](source-type-mapping.md) "rETL metadata" and
"Per-source-type connect-time required keys". It is applied to a whole
definition at once: `warehouse` source type, its connection modes,
`SyncBehaviours`, `SupportsVisualMapper` and `ConnectionRequiredKeys`.

- **Upstream revision**: `rudderlabs/rudder-integrations-config` `develop` at
  `5f11c22c4bcad9ecca4ec37ad31bb7605b72b079` (2026-09-24). DEX-821 read
  `9fa7b26`, 37 commits earlier. Between the two, no registered destination
  changed `supportedSourceTypes`, `supportedConnectionModes`, `syncBehaviours` or
  `supportsVisualMapper`. Only `customerio` changed a connectionMode-gated
  schema branch (see below).
- **CLI definition version**: `1` for every definition below.
- **Flow selection**: rudder-config-backend
  `src/modules/retl/api-gateway/connection-config/assembler.ts`
  `determineFlowTypeFromRequest`. A destination in `DESTINATION_SPECIFIC_REGISTRY`
  (`constants.ts`: `CUSTOMERIO`, `CUSTOMERIO_AUDIENCE`) runs its own flow, which
  the CLI refuses. Otherwise a destination with `supportsVisualMapper` and an
  `object` runs object mapping, and everything else runs the JSON mapper, which
  drops `mirror`. The CLI mirrors this in `retl/connection/flow.go`
  `ClassifyFlow`.
- **`supportedSourcesValidation`**: none. A recursive key walk over all 50
  registered destinations' db-config.json found no occurrence at this revision,
  and a repo-wide code search found only a historical CHANGELOG entry.

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
```

`$TYPE` is the definition's local type, except `linkedin_ads`, whose upstream
directory is `linkedIn_ads`.

## Verified definitions that declare `warehouse`

Each declares `warehouse` with mode `cloud`, as every upstream
`supportedConnectionModes.warehouse` does. Sync behaviours read `default` where
upstream omits `syncBehaviours`, which leaves the field unset and takes the
backend's `upsert`/`mirror`/`full` fallback.

| Type | Sync behaviours | Visual mapper | `ConnectionRequiredKeys[warehouse][cloud]` | Added in |
| --- | --- | --- | --- | --- |
| `active_campaign` | default | no | — | DEX-834 |
| `am` | default | yes | — | DEX-834 |
| `attentive_tag` | default | no | — | DEX-834 |
| `bqstream` | default | no | — | DEX-834 |
| `braze` | default | yes | `rest_api_key` (allOf branch naming `warehouse: cloud`) | DEX-834 |
| `customerio` | `upsert`, `mirror` | no | — (see below) | DEX-834 |
| `facebook_conversions` | default | no | — | DEX-834 |
| `facebook_pixel` | default | no | `access_token` (negated `web: device` branch) | DEX-834 |
| `ga4` | default | no | — | DEX-834 |
| `gcs` | default | no | — | DEX-834 |
| `hs` | default | yes | — | DEX-834 |
| `http` | default | no | — | DEX-821 |
| `iterable` | default | yes | — | DEX-834 |
| `mp` | default | no | — | DEX-834 |
| `posthog` | default | no | — | DEX-834 |
| `s3` | default | no | — | DEX-834 |
| `tiktok_ads` | default | no | — | DEX-834 |
| `webhook` | default | no | — | DEX-834 |

- `customerio` runs the destination-specific flow, so rETL refuses it whatever
  its metadata says. Since `9fa7b26`, upstream no longer requires `siteID` and
  `apiKey` outright. It requires them for every connection mode except
  web-only device mode, and `siteID` also depends on `sdkVersion`. The CLI
  still tags both keys `required` outright, so a connect-time entry would add
  nothing, as on every other source type of this definition. The same upstream
  change added `sdkVersion`, `writeKey` and `anonymousInApp` (all web-only),
  which the definition does not model yet. That is config-surface drift, and
  it is outside this inventory.
- `iterable`'s only connectionMode branch (top-level `anyOf`, `packageName`)
  applies to `web: device` and does not reach `warehouse`.
- `posthog`'s schema.json declares no `connectionMode` property; the definition
  models `connection_mode` from before this inventory, and `warehouse` joins it
  like any other source type.

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
