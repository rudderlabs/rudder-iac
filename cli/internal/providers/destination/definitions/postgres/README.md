# PostgreSQL (`postgres`)

Warehouse destination. RudderStack stages events as files in object storage, then
loads them into a PostgreSQL database on a schedule. Every supported source type
connects in cloud mode — there is no device-mode variant.

In a destination spec:

- `type: postgres`
- `definition_version: 1`

## Choosing your setup

Two independent switches decide which keys you need:

| Switch | Options |
| --- | --- |
| `use_ssh` | connect directly, or through an SSH tunnel |
| `use_rudder_storage` | RudderStack-hosted staging, or your own bucket via `bucket_provider` |

When you bring your own storage, `bucket_provider` picks one of four blocks —
`s3`, `gcs`, `azure` or `minio` — and only that block's keys are required. The
other blocks are still accepted, so switching provider does not force you to
delete the old values.

## Example

Direct connection over SSL, staging through your own S3 bucket with access keys:

```yaml
version: rudder/v1
kind: destination
metadata:
  name: postgres
spec:
  id: postgres
  display_name: PostgreSQL
  type: postgres
  definition_version: 1
  enabled: true
  config:
    host: warehouse.example.com
    port: "5432"
    database: analytics
    user: rudderstack
    password: "{{ .POSTGRES_PASSWORD }}"
    namespace: rudder_events

    ssl_mode: verify-ca
    client_key: "{{ .POSTGRES_CLIENT_KEY }}"
    client_cert: "{{ .POSTGRES_CLIENT_CERT }}"
    server_ca: "{{ .POSTGRES_SERVER_CA }}"

    use_ssh: false

    sync_frequency: "180"
    sync_start_at: "01:00"
    exclude_window:
      start_time: "02:00"
      end_time: "03:00"

    skip_tracks_table: false
    skip_users_table: true
    prefer_append: true
    json_paths: context.traits,properties.metadata
    underscore_divide_numbers: false
    allow_users_context_traits: false

    use_rudder_storage: false
    bucket_provider: S3
    bucket_name: my-rudder-staging
    access_key_id: "{{ .AWS_ACCESS_KEY_ID }}"
    cleanup_object_storage_files: false
    s3:
      role_based_auth: false
      access_key: "{{ .AWS_SECRET_ACCESS_KEY }}"

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

#### `host` — string, required

The host name of your PostgreSQL database.

#### `port` — string, required

The port of your PostgreSQL database.

#### `database` — string, required

The name of your PostgreSQL database. At most 100 characters.

#### `user` — string, required

The username of your PostgreSQL database. At most 100 characters.

#### `password` — string, required, secret

The password for that user.

Supply it as a `{{ .VAR }}` reference — see [Secrets](#secrets).

#### `namespace` — string

The schema name where RudderStack creates all its tables. Defaults to the source
name when omitted.

### TLS

#### `ssl_mode` — string, required

How the connection negotiates TLS.

#### `client_key` — string

Client private key, used by the SSL modes that require client certificates.

#### `client_cert` — string

Client certificate, used by the SSL modes that require client certificates.

#### `server_ca` — string

Server certificate authority, used to verify the database's certificate.

### SSH tunnel

#### `use_ssh` \* — boolean, default `false`

Reach PostgreSQL through an SSH tunnel rather than connecting directly.

#### `ssh` \* — object

SSH tunnel settings, grouped under one block. **All four fields are required when
`use_ssh` is `true`**, including when the block itself is omitted:

- `host` — bastion host name, at most 100 characters
- `port` — bastion port, at most 100 characters
- `user` — SSH user, at most 100 characters
- `public_key` — the SSH public key, at most 1000 characters

### Sync scheduling

#### `sync_frequency` — string, required

How often RudderStack syncs staged events into the database, in minutes. Written
as a string. One of `5`, `10`, `15`, `30`, `60`, `180`, `360`, `720` or `1440`.

#### `sync_start_at` \* — string

Time of day, in UTC, that anchors the sync schedule. Written as `HH:MM`. Not
validated locally.

#### `exclude_window` \* — object

Daily window, in UTC, during which RudderStack does not sync. When present, both
`start_time` and `end_time` are required. Neither format is validated locally.

### Table behaviour

#### `skip_tracks_table` \* — boolean, default `false`

Skip sending event data to the `tracks` table.

#### `skip_users_table` \* — boolean, default `true`

Skip the `users` table, sending identify events only to `identifies`.

#### `prefer_append` \* — boolean, default `true`

Append rows on each sync rather than merging.

#### `json_paths` \* — string

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

#### `bucket_provider` \* — string

Which provider hosts your staging bucket, and therefore which of the four blocks
below applies. One of `S3`, `GCS`, `AZURE_BLOB` or `MINIO`. **Required when
`use_rudder_storage` is `false`.**

#### `bucket_name` \* — string

Name of the staging bucket RudderStack writes to before loading into PostgreSQL.

#### `access_key_id` \* — string, secret

Access key ID for the staging bucket. Unlike the other credentials, this one sits
at the top level rather than inside a provider block, because two providers need
it: **required when `bucket_provider` is `MINIO`, and when it is `S3` with
`s3.role_based_auth` `false`.** At most 100 characters.

#### `cleanup_object_storage_files` \* — boolean, default `false`

Delete the staged files from object storage after a sync completes successfully.

### Provider blocks

Exactly one block applies, chosen by `bucket_provider`. Keys in the other three
are accepted but ignored.

#### `s3` \* — object

Applies when `bucket_provider` is `S3`:

- `role_based_auth` — boolean. Authenticate with an IAM role rather than access
  keys
- `iam_role_arn` — **required when `role_based_auth` is `true`**, at most 100
  characters
- `access_key` — **secret**, required when `role_based_auth` is `false`, at most
  100 characters. Pairs with the top-level `access_key_id`

#### `gcs` \* — object

Applies when `bucket_provider` is `GCS`:

- `credentials` — **secret**, **required** for this provider. GCP service account
  credentials JSON

#### `azure` \* — object

Applies when `bucket_provider` is `AZURE_BLOB`:

- `container_name` — **required** for this provider
- `account_name` — **required** for this provider, at most 100 characters
- `use_sas_tokens` — boolean. Authenticate with a SAS token rather than an
  account key
- `account_key` — **secret**, required when `use_sas_tokens` is `false`, at most
  100 characters
- `sas_token` — **secret**, required when `use_sas_tokens` is `true`

#### `minio` \* — object

Applies when `bucket_provider` is `MINIO`:

- `end_point` — **required** for this provider. Address of your MinIO server
- `secret_access_key` — **secret**, **required** for this provider
- `use_ssl` — boolean. Connect to MinIO over HTTPS

MinIO also requires the top-level `access_key_id`.

## Source types

Every supported source type connects in cloud mode only:

`web` · `android` · `android_kotlin` · `ios` · `ios_swift` · `unity` ·
`react_native` · `flutter` · `cordova` · `cloud`

## Per-source keys

Both keys below are objects keyed by the local source type. A key naming a
source type this destination does not support fails validation.

#### `connection_mode` \* — object

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
destination 'postgres' (type 'postgres') does not support source 'my-source':
source type 'amp' is not among supported source types: android, android_kotlin, ...
```

**The config must carry a `connection_mode` entry for that source type.** This
lives on the destination spec, not the connection spec. Without it:

```
destination 'postgres' config has no 'connection_mode' entry for source type 'web'
```

PostgreSQL requires no additional config keys to connect a source of any type.

## Secrets

Which secrets apply depends on your storage provider:

| Key | Applies when |
| --- | --- |
| `password` | always — the database password |
| `access_key_id` | `bucket_provider` is `MINIO`, or `S3` without role-based auth |
| `s3.access_key` | `bucket_provider` is `S3` without role-based auth |
| `gcs.credentials` | `bucket_provider` is `GCS` |
| `azure.account_key` | `bucket_provider` is `AZURE_BLOB` without SAS tokens |
| `azure.sas_token` | `bucket_provider` is `AZURE_BLOB` with SAS tokens |
| `minio.secret_access_key` | `bucket_provider` is `MINIO` |

The four inside `s3`, `gcs`, `azure` and `minio` are nested secrets — masked
independently, inside their provider block.

Note that `client_key`, `client_cert` and `server_ca` are **not** registered as
secrets, so they are stored and returned in the clear. `ssh.public_key` is not a
secret either, since a public key is not sensitive.

Write each secret as a `{{ .VAR }}` reference and supply the value at apply time:

```yaml
password: "{{ .POSTGRES_PASSWORD }}"
s3:
  access_key: "{{ .AWS_SECRET_ACCESS_KEY }}"
```

```sh
export RUDDER_POSTGRES_PASSWORD=...
rudder-cli apply

# or
rudder-cli apply --var-file secrets.vars.yaml
```

`rudder-cli import` writes each back as a `{{ .VAR }}` placeholder rather than
its value, since the API does not return secrets. Fill the placeholders in
before the first apply.
