# BigQuery (`bq`)

Warehouse destination. RudderStack stages events as files in a Google Cloud
Storage bucket, then loads them into a BigQuery dataset on a schedule.

In a destination spec:

- `type: bq`
- `definition_version: 1` — the only registered version

## Example

Every config key this destination accepts:

```yaml
version: rudder/v1
kind: destination
metadata:
  name: bigquery
spec:
  id: bigquery
  display_name: BigQuery
  type: bq
  definition_version: 1
  enabled: true
  config:
    project: my-gcp-project
    location: US
    bucket_name: my-rudder-staging-bucket
    prefix: rudder/events
    namespace: rudder_events
    credentials: "{{ .BQ_CREDENTIALS }}"

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

## Config keys

`config` accepts only the keys below — anything else fails validation with
`unknown config field "<key>"`.

Keys that declare a default are filled in before the spec enters the resource
graph, because the backend applies the same defaults when it stores the
destination. Omitting one is equivalent to writing its default; it does not
produce a permanent diff.

### Connection

#### `project` — string, required

GCP project ID that holds the BigQuery dataset.

At most 100 characters, and must not contain line breaks.

#### `location` — string

GCP region the dataset lives in, for example `US`, `EU` or `asia-southeast1`.

At most 100 characters, and must not contain line breaks.

#### `bucket_name` — string, required

Staging GCS bucket RudderStack writes event files to before loading them into
BigQuery. The bucket must already exist, and should be co-located with the
dataset so loads do not cross regions.

3 to 63 characters, matching `[a-z0-9][a-z0-9-._]{1,61}[a-z0-9]`. Must not start
with `goog`, contain `google`, look like an IP address, or contain consecutive
dots.

#### `prefix` — string

Folder prefix inside the staging bucket. RudderStack creates a folder with this
prefix and writes all staged files beneath it.

At most 100 characters, and must not contain line breaks.

#### `namespace` — string, rs-immutable

Dataset RudderStack creates its tables in. Defaults to the source name,
snake-cased, when omitted.

At most 64 characters, and must not start with `pg_` in any capitalisation.

Cannot be changed once the destination exists — the API rejects the update. The
CLI does not check this locally: `validate` accepts a change and `apply` sends
it, failing at the API. Create a new destination instead.

#### `credentials` — string, required, secret

GCP service-account JSON key. The account needs BigQuery dataset, table and job
permissions plus read and write access to the staging bucket.

Supply it as a `{{ .VAR }}` reference rather than a literal — see
[Secrets](#secrets).

### Sync scheduling

#### `sync_frequency` — string, required

How often RudderStack syncs staged events into the dataset, in minutes. Written
as a string, not a number.

One of `5`, `10`, `15`, `30`, `60`, `180`, `360`, `720` or `1440`.

#### `sync_start_at` — string

Time of day, in UTC, that anchors the sync schedule; subsequent syncs are
computed from it at `sync_frequency` intervals. Written as `HH:MM` — the
dashboard offers 15-minute steps.

Not validated locally: any string is accepted, and a value the warehouse
scheduler cannot parse silently yields no scheduled times.

#### `exclude_window` — object

Daily window, in UTC, during which RudderStack does not sync. Omit the block
entirely to sync around the clock.

When present, both fields are required:

- `start_time` — string, window opens, `HH:MM`
- `end_time` — string, window closes, `HH:MM`

Neither field's format is validated locally.

### Table behaviour

#### `skip_users_table` — boolean, default `true`

Send `identify` events only to the `identifies` table, skipping the `users`
table. The `users` table holds one row per unique user and is maintained with a
merge, which can add significant time to each sync.

Set it to `false` to populate both tables.

#### `skip_tracks_table` — boolean, default `false`

Skip sending events to the `tracks` table. Per-event tables are unaffected.

#### `skip_views` — boolean, default `false`, rs-immutable

Skip creating the `<table_name>_view` de-duplication view alongside each table.
The views cover the last 60 days and exist so queries do not return duplicate
events; skip them only if you deduplicate another way.

Cannot be changed once the destination exists — the API rejects the update. The
CLI does not check this locally: `validate` accepts a change and `apply` sends
it, failing at the API.

#### `partition_column` — string, default `_PARTITIONTIME`, rs-immutable

Column BigQuery partitions each table on:

- `_PARTITIONTIME` — ingestion time, when BigQuery received the data
- `loaded_at` — when RudderStack loaded the data into the warehouse
- `received_at` — when RudderStack received the event
- `timestamp` — event time corrected for client-side clock skew
- `sent_at` — when the client sent the event to RudderStack
- `original_timestamp` — when the event was generated at the source

Cannot be changed once the destination exists — the API rejects the update. The
CLI does not check this locally: `validate` accepts a change and `apply` sends
it, failing at the API. Create a new destination instead.

#### `partition_type` — string, default `day`, rs-immutable

Granularity of the partition: `hour` or `day`.

Cannot be changed once the destination exists, on the same terms as
`partition_column`.

#### `json_paths` — string

Comma-separated dot-notation paths whose values are stored as JSON columns
instead of being flattened into separate columns — for example
`context.traits,properties.metadata`. Applies to every `track` event sent to this
destination.

Not validated locally.

#### `cleanup_object_storage_files` — boolean, default `false`

Delete staged files from the GCS bucket after a sync completes successfully.

### Internal flags

Both keys below exist to preserve the column naming of destinations created
before the behaviour changed. Leave them at their defaults on a new destination.
Neither can be changed once the destination exists — the API rejects the update,
and the CLI does not check for it locally.

#### `underscore_divide_numbers` — boolean, default `false`, rs-immutable

When `false`, numeric suffixes in column names are preserved: `v3` stays `v3`
rather than being split into `v_3`.

#### `allow_users_context_traits` — boolean, default `false`, rs-immutable

When `false`, `context.traits.*` fields are not promoted to top-level traits and
are stored only as `context_traits_*` columns.

### Per-source keys

Both keys are objects keyed by the **local** source type — the snake_case names
listed under [Source types](#source-types). A key naming a source type this
destination does not support fails validation.

#### `connection_mode` — object

Maps a source type to the mode its events reach BigQuery in. Every source type
supports exactly one mode, `cloud`, so every entry's value is `cloud`.

```yaml
connection_mode:
  web: cloud
  android_kotlin: cloud
```

An entry is required for each source type you connect — see
[Connecting a source](#connecting-a-source).

#### `consent_management` — object

Consent-provider configuration per source type. The entry shape, the accepted
providers, and the rules on `resolution_strategy` and `consents` are shared
across all destinations and documented in
[../common/README.md](../common/README.md).

```yaml
consent_management:
  web:
    - provider: oneTrust
      consents:
        - analytics
        - marketing
```

## Source types

BigQuery accepts events from these source types:

`android` · `android_kotlin` · `ios` · `ios_swift` · `web` · `unity` · `cloud` ·
`react_native` · `flutter` · `cordova`

All of them connect in `cloud` mode only — events are sent from RudderStack's
servers, never from the device SDK.

## Connecting a source

An event stream connection to this destination is checked against two rules at
`validate` time.

**The source's type must be supported.** A source's type is mapped to one of the
tokens above before the check — a JavaScript source resolves to `web`, and
webhook and server-side SDK sources resolve to `cloud`. An unsupported type
reports:

```
destination 'bigquery' (type 'bq') does not support source 'my-source':
source type 'amp' is not among supported source types: android, android_kotlin, ...
```

**The destination config must carry a** `connection_mode` **entry for that source
type.** This lives on the destination spec, not on the connection spec. Without
it:

```
destination 'bigquery' config has no 'connection_mode' entry for source type 'web'
```

BigQuery requires no additional config keys to connect a source of any type.

## Secrets

`credentials` is the only secret key. Write it as a `{{ .VAR }}` reference and
supply the value at apply time, either from the environment or from a var file:

```yaml
credentials: "{{ .BQ_CREDENTIALS }}"
```

```sh
export RUDDER_BQ_CREDENTIALS="$(cat service-account.json)"
rudder-cli apply

# or
rudder-cli apply --var-file secrets.vars.yaml
```

`rudder-cli import` writes `credentials` back as a `{{ .VAR }}` placeholder
rather than its value, since the API does not return secrets. Fill the
placeholder in before the first apply.
