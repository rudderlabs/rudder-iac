# Snowflake Streaming (`snowpipe_streaming`)

Snowflake Streaming is a warehouse destination. RudderStack streams events as rows into tables in a Snowflake database through the Snowpipe Streaming API, in near real time, without staging files or a sync schedule.

In a Snowflake Streaming destination spec:

- `type: snowpipe_streaming`
- `definition_version: 1`

> [!NOTE]
> `snowpipe_streaming` is an [unverified destination type](../README.md#unverified-destination-types). Rudder CLI registers it only when the `unverifiedDestinations` experimental flag is on.

## Sample configuration

```yaml
version: rudder/v1
kind: destination
metadata:
  name: snowflake-streaming-prod
spec:
  id: snowflake-streaming-prod
  display_name: Snowflake Streaming Production
  type: snowpipe_streaming
  definition_version: 1
  enabled: true
  config:
    account: xy12345.us-east-2.aws
    database: RUDDER_EVENTS
    warehouse: RUDDER_WAREHOUSE
    user: RUDDER_USER
    role: RUDDER_ROLE
    namespace: rudder_events

    private_key: "{{ .SNOWFLAKE_PRIVATE_KEY }}"
    private_key_passphrase: "{{ .SNOWFLAKE_PRIVATE_KEY_PASSPHRASE }}"

    enable_iceberg: false

    skip_tracks_table: false
    json_paths: track.properties.metadata,identify.traits.address

    underscore_divide_numbers: false
    allow_users_context_traits: false

    connection_mode:
      web: cloud
      cloud: cloud
    consent_management:
      web:
        - provider: oneTrust
          consents:
            - analytics
```

The above example authenticates with an encrypted key pair and writes standard Snowflake tables. Iceberg tables are a choice you make when you create the destination — see [Iceberg tables](#iceberg-tables).

## Config keys

`config` accepts only the keys listed below. The [shared config key rules](../README.md#config-key-rules) cover unknown keys, defaults, and immutability.

### Connection

#### `account` — string, required

Your Snowflake account identifier — the part of your Snowflake URL before `.snowflakecomputing.com`.

- At most 100 characters, and must not contain line breaks.
- A `{{ path || fallback }}` template is accepted in place of a literal.

#### `database` — string, required

Name of the Snowflake database RudderStack streams data into.

- At most 100 characters, and must not contain line breaks.
- A `{{ path || fallback }}` template is accepted in place of a literal.

#### `warehouse` — string, required

Name of the Snowflake virtual warehouse RudderStack runs its connection checks and schema changes on. Rows are streamed without it, so a small warehouse is enough.

- At most 100 characters, and must not contain line breaks.
- A `{{ path || fallback }}` template is accepted in place of a literal.

#### `user` — string, required

Snowflake user RudderStack connects as. The user authenticates with `private_key` — Snowflake Streaming doesn't support password authentication.

- At most 100 characters, and must not contain line breaks.
- A `{{ path || fallback }}` template is accepted in place of a literal.

#### `role` — string

Role RudderStack assumes. When omitted, the user's default role is used. The role needs permission to create schemas and tables in `database`.

- At most 100 characters, and must not contain line breaks.
- A `{{ path || fallback }}` template is accepted in place of a literal.

#### `namespace` — string, required, immutable

Schema RudderStack creates its tables in.

- At most 64 characters, and must not start with `pg_` in any capitalization.
- A `{{ path || fallback }}` template is accepted in place of a literal.
- Unlike [Snowflake](../snowflake/README.md#namespace--string-immutable), there's no fallback to the source name. A spec that omits this key fails validation with `'namespace' is required`.
- Can't be changed once the destination exists — the API rejects the update. Create a new destination instead.

### Authentication

Snowflake Streaming authenticates with a key pair only.

#### `private_key` — string, required, secret

PEM-encoded private key whose public half is assigned to `user` in Snowflake.

- Must include the delimiters: `-----BEGIN PRIVATE KEY-----` … `-----END PRIVATE KEY-----`, or the `ENCRYPTED PRIVATE KEY` equivalents. A bare base64 key body is rejected, and so is a PKCS#1 `RSA PRIVATE KEY`.
- Templates aren't accepted. `{{ .VAR }}` references are resolved before validation, so the resolved value is what must be PEM-shaped.

Load the key from a file with its line breaks written as `\n`, which the double-quoted reference turns back into line breaks:

```bash
export RUDDER_SNOWFLAKE_PRIVATE_KEY="$(awk '{printf "%s\\n", $0}' rsa_key.p8)"
```

Supply it as a `{{ .VAR }}` reference rather than a literal — see [Secrets](#secrets).

#### `private_key_passphrase` — string, secret

Passphrase you set when encrypting `private_key`. Leave it out for an unencrypted key.

- At most 100 characters, and must not contain line breaks.

> [!WARNING]
> Authentication fails if the key is encrypted and this passphrase is missing. Rudder CLI doesn't check that an `ENCRYPTED PRIVATE KEY` comes with a passphrase, so `validate` doesn't catch it.

### Iceberg tables

By default RudderStack creates standard Snowflake tables. With `enable_iceberg`, it creates Snowflake-managed Iceberg tables instead, which store their data as Parquet files in your own cloud storage. In Iceberg tables, JSON values are stored as `VARCHAR` strings and timestamps as `TIMESTAMP_NTZ`, without a time zone.

#### `enable_iceberg` — boolean, immutable, default `false`

Create Snowflake-managed Iceberg tables instead of standard tables.

- Requires `external_volume`.
- Can't be changed once the destination exists — the API rejects the update. Create a new destination instead.

#### `external_volume` — string, required

Name of the Snowflake external volume Iceberg tables keep their data and metadata files on. The volume must already exist.

- Required when `enable_iceberg` is `true`. Leave it unset otherwise — the dashboard shows it only when Iceberg is on, and RudderStack ignores it without Iceberg.
- At most 100 characters, and must not contain line breaks.
- RudderStack reads it only when it creates a table, so changing it later leaves existing tables on the old volume. To move to another volume, create a new destination.

A spec that sets `enable_iceberg: true` without it fails validation with `'external_volume' is required when 'enable_iceberg' is true`.

```yaml
enable_iceberg: true
external_volume: RUDDER_ICEBERG_VOLUME
```

### Table behavior

Snowflake Streaming only appends rows. It never writes a `users` table — `identify` events go to the `identifies` table only — and doesn't merge, so it has no keys for either.

#### `skip_tracks_table` — boolean, default `false`

Skip sending events to the `tracks` table. Per-event tables are unaffected.

#### `json_paths` — string

Comma-separated dot-notation paths whose values are stored as JSON columns instead of being flattened. Start each path with the event type it applies to, such as `track.properties.metadata`. A path without one applies to `track` events only.

- Not validated locally.

```yaml
json_paths: track.properties.metadata,identify.traits.address
```

### Legacy column naming

Both keys below preserve the column naming of destinations created before the behavior changed. Leave them at their defaults on a new destination. Neither can be changed once the destination exists — the API rejects the update.

#### `underscore_divide_numbers` — boolean, immutable, internal, default `false`

When `false`, numeric suffixes in column names are preserved: `v3` stays `v3` rather than being split into `v_3`.

#### `allow_users_context_traits` — boolean, immutable, internal, default `false`

When `false`, `context.traits.*` fields aren't promoted to top-level traits and are stored only as `context_traits_*` columns.

### Per-source keys

Both keys are objects keyed by the local source type — the tokens listed under [Source types](#source-types). A key naming a source type this destination doesn't support fails validation.

#### `connection_mode` — object

Maps each source type you connect to the mode its events reach Snowflake in, using the modes in [Source types](#source-types).

- An entry is required for each source type you connect — see [Connect a source](#connect-a-source).

```yaml
connection_mode:
  web: cloud
  cloud: cloud
```

#### `consent_management` — object

Consent provider configuration per source type. The entry shape, accepted providers, and the rules on `resolution_strategy` and `consents` are shared across all destination types — see [Consent management](../common/README.md).

## Source types

Snowflake Streaming accepts events from these source types in the mentioned connection modes:

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

Every source type is `cloud` only — events reach Snowflake from RudderStack's servers, never in device mode.

Snowflake Streaming doesn't accept `warehouse`, even in the dashboard.

## Connect a source

An event stream connection to this destination is checked against two rules at `validate` time.

**The source's type must be supported.** A source's type resolves to one of the tokens above before the check — a JavaScript source resolves to `web`, and webhook and server-side SDK sources resolve to `cloud`. Every event stream source type is supported here, so only a Reverse ETL connection, whose source resolves to `warehouse`, fails it:

```text
destination 'snowflake-streaming-prod' (type 'snowpipe_streaming') does not accept rETL sources: source type 'warehouse' is not among supported source types: android, android_kotlin, ...
```

**The destination config must carry a `connection_mode` entry for that source type.** This lives on the destination spec, not on the connection spec. Without it:

```text
destination 'snowflake-streaming-prod' config has no 'connection_mode' entry for source type 'web'
```

Snowflake Streaming needs no additional config keys to connect a source of any type.

## Secrets

`private_key` and `private_key_passphrase` are the secret keys. Write each one you use as a `{{ .VAR }}` reference — [Secrets](../README.md#secrets) covers supplying the values and what `import` writes:

```yaml
config:
  private_key: "{{ .SNOWFLAKE_PRIVATE_KEY }}"
  private_key_passphrase: "{{ .SNOWFLAKE_PRIVATE_KEY_PASSPHRASE }}"
```

Note that:

- `private_key` spans several lines, so pass it with its line breaks written as `\n` — see [`private_key`](#private_key--string-required-secret).
- Unlike [Snowflake](../snowflake/README.md#secrets), `user` isn't a secret here, so `import` writes it in plain text.
