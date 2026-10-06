# PostgreSQL (`postgres`)

PostgreSQL is a warehouse destination. RudderStack stages events as files in object storage, then loads them into a PostgreSQL database on a schedule.

In a PostgreSQL destination spec:

- `type: postgres`
- `definition_version: 1`

## Sample configuration

```yaml
version: rudder/v1
kind: destination
metadata:
  name: postgres-prod
spec:
  id: postgres-prod
  display_name: PostgreSQL Production
  type: postgres
  definition_version: 1
  enabled: true
  config:
    host: db.example.com
    port: "5432"
    database: analytics
    user: "{{ .PG_USER }}"
    password: "{{ .PG_PASSWORD }}"
    namespace: rudder_events

    ssl_mode: require
    use_ssh: false

    use_rudder_storage: false
    bucket_provider: S3
    bucket_name: acme-postgres-staging
    s3:
      role_based_auth: true
      iam_role_arn: "arn:aws:iam::123456789012:role/RudderStackS3"
    cleanup_object_storage_files: false

    sync_frequency: "180"
    sync_start_at: "01:00"
    exclude_window:
      start_time: "02:00"
      end_time: "03:00"

    prefer_append: true
    skip_users_table: true
    skip_tracks_table: false
    json_paths: context.traits,properties.metadata

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

The above example stages files in your own S3 bucket through an IAM role, so it carries no access keys, and uses `require`, so it needs no certificates. Which keys apply depends on several switches — see [Key dependencies](#key-dependencies).

## Config keys

`config` accepts only the keys listed below. The [shared config key rules](../README.md#config-key-rules) cover unknown keys, defaults, and immutability.

> [!NOTE]
> PostgreSQL's string keys don't accept `{{ path || fallback }}` templates as a way around their constraints — a template is measured as literal text against the same rule. Use `{{ .VAR }}` substitution for the secret keys.

### Key dependencies

`ssl_mode`, `use_ssh`, `use_rudder_storage`, and `bucket_provider` decide which other keys apply. Rudder CLI enforces every requirement below; a key outside its branch is accepted and ignored.

| Key | Required when |
| :-----| :-----|
| `client_key`, `client_cert`, `server_ca` | `ssl_mode` is `verify-ca` |
| `ssh` (all four fields) | `use_ssh` is `true` |
| `bucket_provider` | `use_rudder_storage` is `false` |
| `bucket_name` | `use_rudder_storage` is `false` and `bucket_provider` isn't `AZURE_BLOB` |
| `s3.iam_role_arn` | `bucket_provider` is `S3` and `s3.role_based_auth` is `true` |
| `access_key_id`, `s3.access_key` | `bucket_provider` is `S3` and `s3.role_based_auth` isn't `true` |
| `gcs.credentials` | `bucket_provider` is `GCS` |
| `azure.account_name`, `azure.container_name` | `bucket_provider` is `AZURE_BLOB` |
| `azure.account_key` | `bucket_provider` is `AZURE_BLOB` and `azure.use_sas_tokens` isn't `true` |
| `azure.sas_token` | `bucket_provider` is `AZURE_BLOB` and `azure.use_sas_tokens` is `true` |
| `access_key_id`, `minio.end_point`, `minio.secret_access_key`, `minio.use_ssl` | `bucket_provider` is `MINIO` |

Every storage requirement also assumes `use_rudder_storage` is `false`.

> [!WARNING]
> Three provider settings behave differently from the dashboard, which defaults them:
>
> - An omitted `s3.role_based_auth` counts as `false`, so Rudder CLI asks for access keys. Write `role_based_auth: true` to use `s3.iam_role_arn`.
> - An omitted `azure.use_sas_tokens` counts as `false`, so Rudder CLI asks for `azure.account_key`.
> - `minio.use_ssl` must be written out. The dashboard defaults it to `true`; a spec that omits it fails validation.

### Connection

#### `host` — string, required

Hostname of the PostgreSQL server.

- 1 to 200 characters, and must not contain line breaks.
- An `ngrok.io` host is rejected.

#### `port` — string, required

Port of the PostgreSQL server, written as a string — `"5432"`, not `5432`.

- At most 100 characters, and must not contain line breaks.

#### `database` — string, required

Name of the database RudderStack loads data into.

- At most 100 characters, and must not contain line breaks.

#### `user` — string, required, secret

Database user with the permissions RudderStack needs to create schemas and load tables.

- At most 100 characters, and must not contain line breaks.

#### `password` — string, required, secret

Password for `user`.

Supply it as a `{{ .VAR }}` reference rather than a literal — see [Secrets](#secrets).

#### `namespace` — string

Schema RudderStack creates its tables in. Defaults to the source name when omitted.

- At most 64 characters, and must not start with `pg_` in any capitalization.
- The PostgreSQL setup guide says the namespace can't be changed later, so treat it as fixed. It isn't marked immutable in the API schema, so a change isn't rejected.

### TLS

#### `ssl_mode` — string, required

How RudderStack secures its connection to the server.

- `disable` — no encryption.
- `require` — encrypted, without verifying the server.
- `verify-ca` — encrypted, and the server's certificate is checked against `server_ca`. Needs all three certificate keys below.

#### `client_key` — string, required, secret

Contents of the client key PEM file.

- Required when `ssl_mode` is `verify-ca`. Leave it unset otherwise.

#### `client_cert` — string, required, secret

Contents of the client certificate PEM file.

- Required when `ssl_mode` is `verify-ca`. Leave it unset otherwise.

#### `server_ca` — string, required

Contents of the server CA PEM file.

- Required when `ssl_mode` is `verify-ca`. Leave it unset otherwise.

### SSH tunnel

> [!NOTE]
> SSH tunneling is available on the Enterprise plan.

#### `use_ssh` — boolean, default `false`

Connect to PostgreSQL through an SSH tunnel via a bastion host.

#### `ssh` — object, required

Bastion host connection details.

- Required when `use_ssh` is `true`, with all four fields. Leave it unset otherwise.
- `host` — IP address or hostname of the bastion host. At most 100 characters.
- `port` — SSH port of the bastion host, as a string. At most 100 characters.
- `user` — user RudderStack logs in to the bastion host as. At most 100 characters. **Secret** — see [Secrets](#secrets).
- `public_key` — the public key RudderStack generates for this destination. At most 1000 characters.

```yaml
use_ssh: true
ssh:
  host: 203.0.113.10
  port: "22"
  user: "{{ .PG_SSH_USER }}"
  public_key: "ssh-rsa AAAA..."
```

RudderStack holds the private key; add `public_key` to the bastion host's `authorized_keys`. The key comes from RudderStack, so the practical route is to enable SSH on the destination in the dashboard, then import it to pick up the value.

### Object storage

`use_rudder_storage` decides whether RudderStack stages files in its own storage or in yours. With your own, `bucket_provider` picks the provider, and only that provider's block applies.

#### `use_rudder_storage` — boolean, required

Stage files in RudderStack-managed object storage instead of your own.

- Available only on RudderStack-hosted data planes. Self-hosted data planes must set `false` and configure a provider.
- The dashboard defaults this field to `false`. Rudder CLI requires it explicitly.

#### `bucket_provider` — string, required

Object storage provider for staging files.

- Required when `use_rudder_storage` is `false`. Leave it unset otherwise.
- One of `S3`, `GCS`, `AZURE_BLOB`, or `MINIO`.

#### `bucket_name` — string, required

Name of the staging bucket. The bucket must already exist. Azure uses `azure.container_name` instead.

- Required when `use_rudder_storage` is `false` and `bucket_provider` isn't `AZURE_BLOB`. Leave it unset otherwise.
- 3 to 63 characters, whichever provider you use, and must not contain line breaks.
- For `S3`: lowercase letters, digits, dots, and hyphens; not starting with `xn--`, no consecutive dots, not an IP address.
- For `GCS`: lowercase letters, digits, dots, hyphens, and underscores; not starting with `goog`, not containing `google`, no consecutive dots, not an IP address.
- For `MINIO`: lowercase letters, digits, dots, and hyphens; not an IP address.

#### `access_key_id` — string, required, secret

Access key ID for S3 or MinIO. It sits at the top level because both providers use it.

- Required when `bucket_provider` is `MINIO`, or `S3` with `s3.role_based_auth` not `true`. Leave it unset otherwise.
- At most 100 characters, and must not contain line breaks.

#### `cleanup_object_storage_files` — boolean, default `false`

Delete staged files after a sync completes successfully.

- Applies when `use_rudder_storage` is `false`.

#### `s3` — object, required

Amazon S3 settings.

- Required when `use_rudder_storage` is `false` and `bucket_provider` is `S3`. Leave it unset otherwise.
- `role_based_auth` — boolean. `true` to use `iam_role_arn`; omitted or `false` to use `access_key_id` and `access_key`.
- `iam_role_arn` — ARN of the IAM role RudderStack assumes. Required when `role_based_auth` is `true`. At most 100 characters.
- `access_key` — AWS secret access key matching `access_key_id`. Required when `role_based_auth` isn't `true`. At most 100 characters. **Secret**.

```yaml
bucket_provider: S3
bucket_name: acme-postgres-staging
s3:
  role_based_auth: true
  iam_role_arn: "arn:aws:iam::123456789012:role/RudderStackS3"
```

#### `gcs` — object, required

Google Cloud Storage settings.

- Required when `use_rudder_storage` is `false` and `bucket_provider` is `GCS`. Leave it unset otherwise.
- `credentials` — contents of the JSON key file for a service account that can create objects in the bucket. Required. **Secret**.

```yaml
bucket_provider: GCS
bucket_name: acme-postgres-staging
gcs:
  credentials: "{{ .PG_GCS_CREDENTIALS }}"
```

#### `azure` — object, required

Azure Blob Storage settings.

- Required when `use_rudder_storage` is `false` and `bucket_provider` is `AZURE_BLOB`. Leave it unset otherwise.
- `account_name` — storage account name. Required. At most 100 characters.
- `container_name` — staging container, which must already exist. Required. 3 to 63 characters of lowercase letters, digits, and single hyphens.
- `use_sas_tokens` — boolean. `true` to authenticate with `sas_token`; omitted or `false` to use `account_key`.
- `account_key` — storage account key. Required when `use_sas_tokens` isn't `true`. At most 100 characters. **Secret**.
- `sas_token` — shared access signature token. Required when `use_sas_tokens` is `true`. **Secret**.

```yaml
bucket_provider: AZURE_BLOB
azure:
  account_name: acmestorage
  container_name: rudder-staging
  use_sas_tokens: true
  sas_token: "{{ .PG_AZURE_SAS_TOKEN }}"
```

#### `minio` — object, required

MinIO settings. The access key ID goes in the top-level `access_key_id`.

- Required when `use_rudder_storage` is `false` and `bucket_provider` is `MINIO`. Leave it unset otherwise.
- `end_point` — MinIO server endpoint. Required. 1 to 100 characters; an `ngrok.io` endpoint is rejected.
- `secret_access_key` — MinIO secret access key. Required. At most 100 characters. **Secret**.
- `use_ssl` — boolean. Connect to MinIO over TLS. Required, even when `true`.

```yaml
bucket_provider: MINIO
bucket_name: rudder-staging
access_key_id: "{{ .MINIO_ACCESS_KEY_ID }}"
minio:
  end_point: minio.example.com:9000
  secret_access_key: "{{ .MINIO_SECRET_ACCESS_KEY }}"
  use_ssl: true
```

### Sync scheduling

#### `sync_frequency` — string, required

How often RudderStack syncs staged events into PostgreSQL, in minutes. Written as a string, not a number.

- One of `5`, `10`, `15`, `30`, `60`, `180`, `360`, `720`, or `1440`.
- The dashboard defaults this field to `180`. Rudder CLI requires it explicitly.
- A spec that omits this key fails validation.

#### `sync_start_at` — string

Time of day, in UTC, that anchors the sync schedule. Subsequent syncs are computed from it at `sync_frequency` intervals. Written as `HH:MM`.

- Not validated locally: any string is accepted, and a value the scheduler can't parse silently yields no scheduled times.

#### `exclude_window` — object

Daily window, in UTC, during which RudderStack doesn't sync. Omit the block entirely to sync around the clock.

- When present, both fields are required: `start_time` and `end_time`, each `HH:MM`.
- Neither field's format is validated locally.

### Table behavior

#### `prefer_append` — boolean, default `true`

Append incoming events to existing tables. Set it to `false` to merge instead, which guarantees no duplicates at the cost of noticeably longer syncs. This is what the dashboard calls **Warehouse Append**.

#### `skip_users_table` — boolean, default `true`

Send `identify` events only to the `identifies` table, skipping the `users` table. The `users` table holds one row per unique user and is maintained with a merge, which can add significant time to each sync.

#### `skip_tracks_table` — boolean, default `false`

Skip sending events to the `tracks` table. Per-event tables are unaffected.

#### `json_paths` — string

Comma-separated dot-notation paths whose values are stored as JSON columns instead of being flattened. Applies to every `track` event sent to this destination.

- Not validated locally.

### Legacy column naming

Both keys below preserve the column naming of destinations created before the behavior changed. Leave them at their defaults on a new destination. Neither can be changed once the destination exists — the API rejects the update.

#### `underscore_divide_numbers` — boolean, immutable, internal, default `false`

When `false`, numeric suffixes in column names are preserved: `v3` stays `v3` rather than being split into `v_3`.

#### `allow_users_context_traits` — boolean, immutable, internal, default `false`

When `false`, `context.traits.*` fields aren't promoted to top-level traits and are stored only as `context_traits_*` columns.

### Per-source keys

Both keys are objects keyed by the local source type — the tokens listed under [Source types](#source-types). A key naming a source type this destination doesn't support fails validation.

#### `connection_mode` — object

Maps each source type you connect to the mode its events reach PostgreSQL in, using the modes in [Source types](#source-types).

- An entry is required for each source type you connect — see [Connect a source](#connect-a-source).

```yaml
connection_mode:
  web: cloud
  cloud: cloud
```

#### `consent_management` — object

Consent provider configuration per source type. The entry shape, accepted providers, and the rules on `resolution_strategy` and `consents` are shared across all destination types — see [Consent management](../common/README.md).

## Source types

PostgreSQL accepts events from these source types in the mentioned connection modes:

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

Every source type is `cloud` only — events reach the warehouse from RudderStack's servers, never in device mode.

> [!NOTE]
> The dashboard additionally offers PostgreSQL to AMP, Shopify, and cloud app sources. Rudder CLI doesn't manage those connections, so `amp`, `shopify`, and `cloud_source` are invalid here.

## Connect a source

An event stream connection to this destination is checked against two rules at `validate` time.

**The source's type must be supported.** A source's type resolves to one of the tokens above before the check — a JavaScript source resolves to `web`, and webhook and server-side SDK sources resolve to `cloud`. An unsupported type reports:

```text
destination 'postgres-prod' (type 'postgres') does not support source 'my-source':
source type 'amp' is not among supported source types: android, android_kotlin, ...
```

**The destination config must carry a `connection_mode` entry for that source type.** This lives on the destination spec, not on the connection spec. Without it:

```text
destination 'postgres-prod' config has no 'connection_mode' entry for source type 'web'
```

PostgreSQL needs no additional config keys to connect a source of any type.

## Secrets

Rudder CLI treats eleven keys as secrets: `user`, `password`, `client_key`, `client_cert`, `access_key_id`, `s3.access_key`, `gcs.credentials`, `azure.account_key`, `azure.sas_token`, `minio.secret_access_key`, and `ssh.user`. Write each one you use as a `{{ .VAR }}` reference and supply the value at apply time:

```yaml
config:
  user: "{{ .PG_USER }}"
  password: "{{ .PG_PASSWORD }}"
```

```bash
export RUDDER_PG_USER="rudder"
export RUDDER_PG_PASSWORD="..."
rudder-cli apply

# or
rudder-cli apply --var-file secrets.vars.yaml
```

Note that:

- `server_ca` isn't a secret — a CA certificate is public by design.
- The YAML that `rudder-cli import` writes may or may not include secret keys. Before you apply, make sure every secret key your configuration needs is present and populated through variable substitution.
