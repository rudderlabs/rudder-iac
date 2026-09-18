# Snowflake (`snowflake`)

Warehouse destination. RudderStack stages events as files in object storage, then
loads them into Snowflake on a schedule. Every supported source type connects in
cloud mode — there is no device-mode variant.

In a destination spec:

- `type: snowflake`
- `definition_version: 1`

## Choosing your setup

Two independent switches decide which keys you need:

| Switch | Options |
| --- | --- |
| `use_key_pair_auth` | password, or key-pair |
| `use_rudder_storage` | RudderStack-hosted staging, or your own bucket via `cloud_provider` |

When you bring your own storage, `cloud_provider` picks one of three blocks —
`s3`, `gcp` or `azure` — and only that block's keys are required. The other
blocks are still accepted, so switching provider does not force you to delete
the old values.

## Example

Key-pair authentication, staging through your own S3 bucket with a role:

```yaml
version: rudder/v1
kind: destination
metadata:
  name: snowflake
spec:
  id: snowflake
  display_name: Snowflake
  type: snowflake
  definition_version: 1
  enabled: true
  config:
    account: xy12345.us-east-1
    database: ANALYTICS
    warehouse: RUDDER_WH
    user: RUDDER
    role: RUDDER_ROLE
    namespace: RUDDER_EVENTS

    use_key_pair_auth: true
    private_key: "{{ .SNOWFLAKE_PRIVATE_KEY }}"
    private_key_passphrase: "{{ .SNOWFLAKE_PRIVATE_KEY_PASSPHRASE }}"

    sync_frequency: "180"
    sync_start_at: "01:00"
    exclude_window:
      start_time: "02:00"
      end_time: "03:00"

    skip_tracks_table: false
    skip_users_table: true
    prefer_append: true
    manual_sync: false
    json_paths: context.traits,properties.metadata
    underscore_divide_numbers: false
    allow_users_context_traits: false

    use_rudder_storage: false
    cloud_provider: AWS
    bucket_name: my-rudder-staging
    prefix: rudder/events
    storage_integration: RUDDER_S3_INTEGRATION
    cleanup_object_storage_files: false
    s3:
      role_based_auth: true
      iam_role_arn: "arn:aws:iam::123456789012:role/RudderStackSnowflake"
      enable_sse: true

    connection_mode:
      cloud: cloud
      web: cloud
    consent_management:
      web:
        - provider: oneTrust
          consents:
            - analytics
```

## Config keys

`config` accepts only the keys documented here — anything else fails validation
with `unknown config field "<key>"`.

Keys that declare a default are filled in before the spec enters the resource
graph, matching what the backend stores, so omitting one is equivalent to
writing its default and does not produce a permanent diff.

A `*` after a key name marks a description written without a Terraform provider
source to draw on. Those need a closer review pass; the markers come out once the
wording is confirmed.

### Connection

#### `account` — string, required

Account ID of your Snowflake warehouse, taken from the Snowflake URL. At most 100
characters.

#### `database` — string, required

Name of the database. At most 100 characters.

#### `warehouse` — string, required

Name of the warehouse. At most 100 characters.

#### `user` — string, required

Name of the user. At most 100 characters.

#### `role` — string

Role for the user. The user's default role is used when omitted. At most 100
characters.

#### `namespace` — string

Schema name in the warehouse where RudderStack creates its tables.

### Authentication

`use_key_pair_auth` selects between the two paths and decides which keys below
are required.

#### `use_key_pair_auth` — boolean, required

Use key-pair authentication instead of password-based authentication. Required —
there is no default, so every spec states which path it uses.

#### `password` — string, secret

Password for the user. **Required when `use_key_pair_auth` is `false`.**

Supply it as a `{{ .VAR }}` reference — see [Secrets](#secrets).

#### `private_key` — string, secret

Private key for key-pair authentication. **Required when `use_key_pair_auth` is
`true`.**

Must be PEM-encoded, carrying `-----BEGIN PRIVATE KEY-----` and
`-----END PRIVATE KEY-----` headers (or their `ENCRYPTED` forms). A raw
base64 key body is rejected — the CLI does not add the headers for you.

#### `private_key_passphrase` — string, secret

Passphrase for the private key, if it is encrypted. At most 100 characters.

### Sync scheduling

#### `sync_frequency` \* — string, required

How often RudderStack syncs staged events into the warehouse, in minutes.
Written as a string. One of `5`, `10`, `15`, `30`, `60`, `180`, `360`, `720` or
`1440`.

#### `sync_start_at` \* — string

Time of day, in UTC, that anchors the sync schedule. Written as `HH:MM`. Not
validated locally.

#### `exclude_window` \* — object

Daily window, in UTC, during which RudderStack does not sync. When present, both
`start_time` and `end_time` are required. Neither format is validated locally.

#### `manual_sync` — boolean, default `false`

Enable manual sync mode, so syncs run only when you trigger them rather than on
the schedule above.

### Table behaviour

#### `skip_tracks_table` — boolean, default `false`

Skip sending event data to the `tracks` table.

#### `skip_users_table` — boolean, default `true`

Skip the `users` table, sending identify events only to `identifies`.

#### `prefer_append` — boolean, default `true`

Append rows on each sync rather than merging.

#### `json_paths` — string

Comma-separated dot-notation paths stored as JSON columns rather than flattened.

### Internal flags

Both keys below preserve the column naming of destinations created before the
behaviour changed. Leave them at their defaults on a new destination.

#### `underscore_divide_numbers` \* — boolean, default `false`

When `false`, numeric suffixes in column names are preserved: `v3` stays `v3`
rather than being split into `v_3`.

#### `allow_users_context_traits` \* — boolean, default `false`

When `false`, `context.traits.*` fields are stored only as `context_traits_*`
columns rather than promoted to top-level traits.

### Staging storage

#### `use_rudder_storage` — boolean, required

Use RudderStack-managed buckets for object storage rather than your own.

#### `cloud_provider` \* — string, default `AWS`

Which provider hosts your staging bucket, and therefore which of the three
blocks below applies. One of `AWS`, `GCP` or `AZURE`. **Required when
`use_rudder_storage` is `false`.**

#### `bucket_name` — string

Name of the staging bucket RudderStack writes to before loading into Snowflake.
**Required unless `use_rudder_storage` is `true` or `cloud_provider` is
`AZURE`** — the Azure block carries its own container name instead. At most 100
characters.

#### `prefix` — string

Folder prefix inside the bucket. RudderStack creates a folder with this prefix
and writes all data beneath it. At most 100 characters.

#### `storage_integration` — string

Name of the cloud storage integration created in Snowflake. **Required unless
`use_rudder_storage` is `true` or `cloud_provider` is `AWS`.** At most 100
characters.

#### `cleanup_object_storage_files` \* — boolean, default `false`

Delete the staged files from object storage after a sync completes successfully.

### Provider blocks

Exactly one block applies, chosen by `cloud_provider`. Keys in the other two
blocks are accepted but ignored.

#### `s3` \* — object

Applies when `cloud_provider` is `AWS`:

- `role_based_auth` — boolean, **required** for this provider. Authenticate with
  an IAM role rather than access keys
- `iam_role_arn` — **required when `role_based_auth` is `true`**, at most 100
  characters
- `access_key_id` — **secret**, required when `role_based_auth` is `false`, at
  most 100 characters
- `access_key` — **secret**, required when `role_based_auth` is `false`, at most
  100 characters
- `enable_sse` — boolean, default `false`. Enable server-side encryption on the
  bucket

#### `gcp` \* — object

Applies when `cloud_provider` is `GCP`:

- `credentials` — **secret**, **required** for this provider. GCP service account
  credentials JSON used to load data into your Cloud Storage bucket

#### `azure` \* — object

Applies when `cloud_provider` is `AZURE`:

- `container_name` — **required** for this provider. Name of the Azure container
  RudderStack writes to
- `account_name` — **required** for this provider, at most 100 characters
- `use_sas_tokens` — boolean, default `false`. Authenticate with a SAS token
  rather than an account key
- `account_key` — **secret**, required when `use_sas_tokens` is `false`, at most
  100 characters
- `sas_token` — **secret**, required when `use_sas_tokens` is `true`

## Source types

Every supported source type connects in cloud mode only:

`web` · `android` · `android_kotlin` · `ios` · `ios_swift` · `unity` ·
`react_native` · `flutter` · `cordova` · `cloud`

## Per-source keys

Both keys below are objects keyed by the local source type. A key naming a
source type this destination does not support fails validation.

#### `connection_mode` — object

Selects the mode per source type. Every supported type accepts `cloud` only, so
each entry's value is `cloud`:

```yaml
connection_mode:
  web: cloud
  cloud: cloud
```

An entry is required for each source type you connect — see
[Connecting a source](#connecting-a-source).

#### `consent_management` — object

Specify consent configuration data for multiple providers, per source type. The
entry shape, accepted providers, and the rules on `resolution_strategy` and
`consents` are shared across all destinations and documented in
[../common/README.md](../common/README.md).

## Connecting a source

An event stream connection to this destination is checked against two rules at
`validate` time.

**The source's type must be supported.** A source's type is mapped to one of the
tokens above first — a JavaScript source resolves to `web`, and webhook and
server-side SDK sources resolve to `cloud`. An unsupported type reports:

```
destination 'snowflake' (type 'snowflake') does not support source 'my-source':
source type 'amp' is not among supported source types: android, android_kotlin, ...
```

**The config must carry a `connection_mode` entry for that source type.** This
lives on the destination spec, not the connection spec. Without it:

```
destination 'snowflake' config has no 'connection_mode' entry for source type 'web'
```

Snowflake requires no additional config keys to connect a source of any type.

## Secrets

Snowflake has the largest set of secret keys of any destination here, and which
apply depends on your authentication path and storage provider:

| Key | Applies when |
| --- | --- |
| `password` | `use_key_pair_auth` is `false` |
| `private_key` | `use_key_pair_auth` is `true` |
| `private_key_passphrase` | the private key is encrypted |
| `s3.access_key_id`, `s3.access_key` | `cloud_provider` is `AWS` without role-based auth |
| `gcp.credentials` | `cloud_provider` is `GCP` |
| `azure.account_key` | `cloud_provider` is `AZURE` without SAS tokens |
| `azure.sas_token` | `cloud_provider` is `AZURE` with SAS tokens |

The four inside `s3`, `gcp` and `azure` are nested secrets — masked
independently, inside their provider block.

Write each as a `{{ .VAR }}` reference and supply the value at apply time:

```yaml
private_key: "{{ .SNOWFLAKE_PRIVATE_KEY }}"
s3:
  access_key: "{{ .AWS_SECRET_ACCESS_KEY }}"
```

```sh
export RUDDER_SNOWFLAKE_PRIVATE_KEY=...
rudder-cli apply

# or
rudder-cli apply --var-file secrets.vars.yaml
```

`rudder-cli import` writes each back as a `{{ .VAR }}` placeholder rather than
its value, since the API does not return secrets. Fill the placeholders in
before the first apply.
