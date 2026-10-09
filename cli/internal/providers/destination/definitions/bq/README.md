# BigQuery (`bq`)

Google BigQuery is a warehouse destination. RudderStack stages events as files in a Google Cloud Storage bucket, then loads them into a BigQuery dataset on a schedule.

In a BigQuery destination spec:

- `type: bq`
- `definition_version: 1`

## Sample configuration

```yaml
version: rudder/v1
kind: destination
metadata:
  name: bigquery-prod
spec:
  id: bigquery-prod
  display_name: BigQuery Production
  type: bq
  definition_version: 1
  enabled: true
  config:
    project: acme-analytics
    location: US
    bucket_name: rudder-bq-staging
    prefix: rudder/events
    namespace: rudder_events
    credentials: '{{ .BQ_CREDENTIALS }}'

    sync_frequency: "180"
    sync_start_at: "01:00"
    exclude_window:
      start_time: "02:00"
      end_time: "03:00"

    skip_users_table: true
    skip_tracks_table: false
    skip_views: false
    partition_column: loaded_at
    partition_type: day
    json_paths: context.traits,properties.metadata
    cleanup_object_storage_files: false

    underscore_divide_numbers: false
    allow_users_context_traits: false

    connection_mode:
      web: cloud
      android_kotlin: cloud
    consent_management:
      web:
        - provider: oneTrust
          consents:
            - analytics
            - marketing
```

The above example uses a `{{ .VAR }}` reference for `credentials` — see [Secrets](#secrets).

## Config keys

`config` accepts only the keys listed below. The [shared config key rules](../README.md#config-key-rules) cover unknown keys, defaults, and immutability.

> [!NOTE]
> BigQuery's string keys don't accept `{{ path || fallback }}` templates as a way around their constraints. A template is measured as literal text against the same rule, so an over-long one fails, and a key constrained by shape rather than length — `bucket_name`, `partition_column`, `partition_type` — rejects a template outright. Use `{{ .VAR }}` substitution for `credentials`, which has no pattern constraint.

### Connection

#### `project` — string, required

GCP project ID that holds the BigQuery dataset.

- At most 100 characters, and must not contain line breaks.

#### `location` — string

GCP region the dataset lives in, for example `US`, `EU`, or `asia-southeast1`.

- At most 100 characters, and must not contain line breaks.

#### `bucket_name` — string, required

Staging GCS bucket RudderStack writes event files to before loading them into BigQuery. The bucket must already exist, and should be co-located with the dataset so loads don't cross regions.

- 3 to 63 characters, matching `[a-z0-9][a-z0-9-._]{1,61}[a-z0-9]`.
- It must not start with `goog`, contain `google`, look like an IP address, or contain consecutive dots.

#### `prefix` — string

Folder prefix inside the staging bucket. RudderStack creates a folder with this prefix and writes all staged files beneath it.

- At most 100 characters, and must not contain line breaks.

#### `namespace` — string, immutable

Dataset RudderStack creates its tables in. Defaults to the source name, snake-cased, when omitted.

- At most 64 characters, and must not start with `pg_` in any capitalization.
- Can't be changed once the destination exists — the API rejects the update. Create a new destination instead.

#### `credentials` — string, required, secret

GCP service account JSON key. The account needs BigQuery dataset, table, and job permissions, plus read and write access to the staging bucket.

- The dashboard also offers Workload Identity Federation, which authenticates without a stored key. Rudder CLI doesn't support it, so `credentials` is always required.

Supply it as a `{{ .VAR }}` reference rather than a literal — see [Secrets](#secrets).

### Sync scheduling

#### `sync_frequency` — string, required

How often RudderStack syncs staged events into the dataset, in minutes. Written as a string, not a number.

- One of `5`, `10`, `15`, `30`, `60`, `180`, `360`, `720`, or `1440`.
- The dashboard defaults this field to `180`. Rudder CLI requires it explicitly.
- A spec that omits this key fails validation.

#### `sync_start_at` — string

Time of day, in UTC, that anchors the sync schedule. Subsequent syncs are computed from it at `sync_frequency` intervals. Written as `HH:MM` — the dashboard offers 15-minute steps.

- Not validated locally: any string is accepted, and a value the warehouse scheduler can't parse silently yields no scheduled times.

#### `exclude_window` — object

Daily window, in UTC, during which RudderStack doesn't sync. Omit the block entirely to sync around the clock.

- When present, both fields are required.
- `start_time` — string, when the window opens, `HH:MM`
- `end_time` — string, when the window closes, `HH:MM`
- Neither field's format is validated locally.

```yaml
exclude_window:
  start_time: "02:00"
  end_time: "03:00"
```

### Table behavior

#### `skip_users_table` — boolean, default `true`

Send `identify` events only to the `identifies` table, skipping the `users` table. The `users` table holds one row per unique user and is maintained with a merge, which can add significant time to each sync.

Set it to `false` to populate both tables.

#### `skip_tracks_table` — boolean, default `false`

Skip sending events to the `tracks` table. Per-event tables are unaffected.

#### `skip_views` — boolean, immutable, default `false`

Skip creating the `<table_name>_view` deduplication view alongside each table. The views cover the last 60 days and exist so queries don't return duplicate events. Skip them only if you deduplicate another way.

- Can't be changed once the destination exists — the API rejects the update.

#### `partition_column` — string, immutable, default `_PARTITIONTIME`

Column BigQuery partitions each table on:

- `_PARTITIONTIME` — ingestion time, when BigQuery received the data
- `loaded_at` — when RudderStack loaded the data into the warehouse
- `received_at` — when RudderStack received the event
- `timestamp` — event time corrected for client-side clock skew
- `sent_at` — when the client sent the event to RudderStack
- `original_timestamp` — when the event was generated at the source
- Can't be changed once the destination exists — the API rejects the update. Create a new destination instead.

#### `partition_type` — string, immutable, default `day`

Granularity of the partition: `hour` or `day`.

- Can't be changed once the destination exists, on the same terms as `partition_column`.

#### `json_paths` — string

Comma-separated dot-notation paths whose values are stored as JSON columns instead of being flattened into separate columns. Applies to every `track` event sent to this destination.

- Not validated locally.

```yaml
json_paths: context.traits,properties.metadata
```

#### `cleanup_object_storage_files` — boolean, default `false`

Delete staged files from the GCS bucket after a sync completes successfully.

### Legacy column naming

Both keys below preserve the column naming of destinations created before the behavior changed. Leave them at their defaults on a new destination. Neither can be changed once the destination exists — the API rejects the update.

#### `underscore_divide_numbers` — boolean, immutable, internal, default `false`

When `false`, numeric suffixes in column names are preserved: `v3` stays `v3` rather than being split into `v_3`.

#### `allow_users_context_traits` — boolean, immutable, internal, default `false`

When `false`, `context.traits.*` fields aren't promoted to top-level traits and are stored only as `context_traits_*` columns.

### Per-source keys

Both keys are objects keyed by the local source type — the tokens listed under [Source types](#source-types). A key naming a source type this destination doesn't support fails validation.

#### `connection_mode` — object

Maps each source type you connect to the mode its events reach BigQuery in, using the modes in [Source types](#source-types).

- An entry is required for each source type you connect — see [Connect a source](#connect-a-source).

```yaml
connection_mode:
  web: cloud
  android_kotlin: cloud
```

#### `consent_management` — object

Consent provider configuration per source type. The entry shape, accepted providers, and the rules on `resolution_strategy` and `consents` are shared across all destination types — see [Consent management](../common/README.md).

## Source types

BigQuery accepts events from these source types in the mentioned connection modes:

| Source type | Connection mode |
| :-----| :-----|
| `android` | `cloud` |
| `android_kotlin` | `cloud` |
| `ios` | `cloud` |
| `ios_swift` | `cloud` |
| `web` | `cloud` |
| `unity` | `cloud` |
| `cloud` | `cloud` |
| `react_native` | `cloud` |
| `flutter` | `cloud` |
| `cordova` | `cloud` |

Every source type is `cloud` only — events reach the dataset from RudderStack's servers, never in device mode.

BigQuery doesn't accept `warehouse`, even in the dashboard.

## Connect a source

An event stream connection to this destination is checked against two rules at `validate` time.

**The source's type must be supported.** A source's type resolves to one of the tokens above before the check — a JavaScript source resolves to `web`, and webhook and server-side SDK sources resolve to `cloud`. Every event stream source type is supported here, so only a Reverse ETL connection, whose source resolves to `warehouse`, fails it:

```text
destination 'bigquery-prod' (type 'bq') does not accept rETL sources: source type 'warehouse' is not among supported source types: android, android_kotlin, ...
```

**The destination config must carry a `connection_mode` entry for that source type.** This lives on the destination spec, not on the connection spec. Without it:

```text
destination 'bigquery-prod' config has no 'connection_mode' entry for source type 'web'
```

BigQuery needs no additional config keys to connect a source of any type.

## Secrets

`credentials` is the only secret key. Write it as a `{{ .VAR }}` reference — [Secrets](../README.md#secrets) covers supplying the values and what `import` writes:

```yaml
config:
  credentials: '{{ .BQ_CREDENTIALS }}'
```

`credentials` is JSON, so write its reference in single quotes, as above — see [Secrets](../README.md#secrets). Export the key file as is:

```bash
export RUDDER_BQ_CREDENTIALS="$(cat service-account.json)"
```
