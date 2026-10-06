# Snowflake (`snowflake`)

Snowflake is a warehouse destination. RudderStack stages events as files in object storage, then loads them into a Snowflake database on a schedule.

In a Snowflake destination spec:

- `type: snowflake`
- `definition_version: 1`

## Sample configuration

```yaml
version: rudder/v1
kind: destination
metadata:
  name: snowflake-prod
spec:
  id: snowflake-prod
  display_name: Snowflake Production
  type: snowflake
  definition_version: 1
  enabled: true
  config:
    account: xy12345.us-east-2.aws
    database: RUDDER_EVENTS
    warehouse: RUDDER_WAREHOUSE
    user: "{{ .SNOWFLAKE_USER }}"
    role: RUDDER_ROLE
    namespace: rudder_events

    use_key_pair_auth: true
    private_key: "{{ .SNOWFLAKE_PRIVATE_KEY }}"
    private_key_passphrase: "{{ .SNOWFLAKE_PRIVATE_KEY_PASSPHRASE }}"

    use_rudder_storage: false
    cloud_provider: AWS
    bucket_name: acme-snowflake-staging
    prefix: rudder
    storage_integration: RUDDER_S3_INTEGRATION
    s3:
      role_based_auth: true
      iam_role_arn: "arn:aws:iam::123456789012:role/RudderStackS3"
      enable_sse: false
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
    manual_sync: false

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

The above example authenticates with an encrypted key pair and stages files in your own S3 bucket through an IAM role, so it carries no password or access keys. Which keys apply depends on several switches — see [Key dependencies](#key-dependencies).

## Config keys

`config` accepts only the keys listed below. The [shared config key rules](../README.md#config-key-rules) cover unknown keys, defaults, and immutability.

### Key dependencies

`use_key_pair_auth`, `use_rudder_storage`, and `cloud_provider` decide which other keys apply. Rudder CLI enforces every requirement below; a key outside its branch is accepted and ignored.

| Key | Required when |
| :-----| :-----|
| `password` | `use_key_pair_auth` is `false` |
| `private_key` | `use_key_pair_auth` is `true` |
| `bucket_name` | `cloud_provider` is `AWS` or `GCP` |
| `storage_integration` | `cloud_provider` is `GCP` or `AZURE` |
| `s3.role_based_auth` | `cloud_provider` is `AWS` |
| `s3.iam_role_arn` | `cloud_provider` is `AWS` and `s3.role_based_auth` is `true` |
| `s3.access_key_id`, `s3.access_key` | `cloud_provider` is `AWS` and `s3.role_based_auth` is `false` |
| `gcp.credentials` | `cloud_provider` is `GCP` |
| `azure.container_name`, `azure.account_name` | `cloud_provider` is `AZURE` |
| `azure.account_key` | `cloud_provider` is `AZURE` and `azure.use_sas_tokens` is `false` |
| `azure.sas_token` | `cloud_provider` is `AZURE` and `azure.use_sas_tokens` is `true` |

Every storage requirement also assumes `use_rudder_storage` is `false`.

> [!WARNING]
> `validate` fills in defaults before it checks the spec, so an omitted key is checked as though you had written its default. Two keys are worth knowing about:
>
> - Omitting `cloud_provider` selects `AWS`, which makes `bucket_name` and `s3.role_based_auth` required.
> - Omitting `azure.use_sas_tokens` selects account key authentication, which makes `azure.account_key` required.

### Connection

#### `account` — string, required

Your Snowflake account identifier — the part of your Snowflake URL before `.snowflakecomputing.com`.

- At most 100 characters, and must not contain line breaks.
- A `{{ path || fallback }}` template is accepted in place of a literal.

#### `database` — string, required

Name of the Snowflake database RudderStack loads data into.

- At most 100 characters, and must not contain line breaks.
- A `{{ path || fallback }}` template is accepted in place of a literal.

#### `warehouse` — string, required

Name of the Snowflake virtual warehouse that runs the loads.

- At most 100 characters, and must not contain line breaks.
- A `{{ path || fallback }}` template is accepted in place of a literal.

#### `user` — string, required, secret

Snowflake user RudderStack connects as.

- At most 100 characters, and must not contain line breaks.
- A `{{ path || fallback }}` template is accepted in place of a literal.

#### `role` — string

Role RudderStack assumes. When omitted, the user's default role is used. The role needs permission to load data into `database`.

- At most 100 characters, and must not contain line breaks.

#### `namespace` — string, immutable

Schema RudderStack creates its tables in. Defaults to the source name when omitted.

- At most 64 characters, and must not start with `pg_` in any capitalization.
- A `{{ path || fallback }}` template is accepted in place of a literal.
- Can't be changed once the destination exists — the API rejects the update. Create a new destination instead.

### Authentication

#### `use_key_pair_auth` — boolean, required

Authenticate with a key pair instead of a password. Key pair authentication is the recommended method.

- The dashboard defaults this field to `false`. Rudder CLI requires it explicitly.
- A spec that omits this key fails validation.

#### `password` — string, required, secret

Password for `user`.

- Required when `use_key_pair_auth` is `false`. Leave it unset otherwise.

#### `private_key` — string, required, secret

PEM-encoded private key whose public half is assigned to `user` in Snowflake.

- Required when `use_key_pair_auth` is `true`. Leave it unset otherwise.
- Must include the delimiters: `-----BEGIN PRIVATE KEY-----` … `-----END PRIVATE KEY-----`, or the `ENCRYPTED PRIVATE KEY` equivalents. A bare base64 key body is rejected.
- Templates aren't accepted. `{{ .VAR }}` references are resolved before validation, so the resolved value is what must be PEM-shaped.

Load the key from a file rather than pasting it into YAML:

```bash
export RUDDER_SNOWFLAKE_PRIVATE_KEY="$(cat rsa_key.p8)"
```

#### `private_key_passphrase` — string, secret

Passphrase you set when encrypting `private_key`. Leave it out for an unencrypted key.

- Applies when `use_key_pair_auth` is `true`. Leave it unset otherwise.
- At most 100 characters, and must not contain line breaks.

> [!WARNING]
> Authentication fails if the key is encrypted and this passphrase is missing — Rudder CLI can't tell whether a key is encrypted, so `validate` doesn't catch it.

### Object storage

`use_rudder_storage` decides whether RudderStack stages files in its own storage or in yours. With your own, `cloud_provider` picks the provider, and only that provider's block applies.

#### `use_rudder_storage` — boolean, required

Stage files in RudderStack-managed object storage instead of your own.

- Available only on RudderStack-hosted data planes. Self-hosted data planes must set `false` and configure a provider.
- The dashboard defaults this field to `false`. Rudder CLI requires it explicitly.

#### `cloud_provider` — string, default `AWS`

Cloud that hosts your staging storage — normally the one your Snowflake account runs on.

- Applies when `use_rudder_storage` is `false`.
- One of `AWS`, `GCP`, or `AZURE`.

See [Key dependencies](#key-dependencies) for what omitting it makes required.

#### `bucket_name` — string, required

Name of the staging S3 or GCS bucket. The bucket must already exist. Azure uses `azure.container_name` instead.

- Required when `use_rudder_storage` is `false` and `cloud_provider` is `AWS` or `GCP`. Leave it unset otherwise.
- For `AWS`: lowercase letters, digits, dots, and hyphens; not starting with `xn--`, no consecutive dots, not an IP address.
- For `GCP`: lowercase letters, digits, dots, hyphens, and underscores; not starting with `goog`, not containing `google`, no consecutive dots, not an IP address.
- A `{{ path || fallback }}` template is accepted in place of a literal.

#### `prefix` — string

Folder prefix inside the staging bucket or container.

- Applies when `use_rudder_storage` is `false`.
- At most 100 characters, and must not contain line breaks.

#### `storage_integration` — string, required

Name of the Snowflake storage integration RudderStack uses to run `COPY` from the staging storage.

- Required when `use_rudder_storage` is `false` and `cloud_provider` is `GCP` or `AZURE`. Optional for `AWS`.
- At most 100 characters, and must not contain line breaks.
- A `{{ path || fallback }}` template is accepted in place of a literal.

#### `cleanup_object_storage_files` — boolean, default `false`

Delete staged files after a sync completes successfully.

- Applies when `use_rudder_storage` is `false`.

#### `s3` — object, required

Amazon S3 settings.

- Required when `use_rudder_storage` is `false` and `cloud_provider` is `AWS`. Leave it unset otherwise.
- `role_based_auth` — boolean. `true` to use `iam_role_arn`, `false` to use the access key pair. Required — the dashboard's default doesn't apply.
- `iam_role_arn` — ARN of the IAM role RudderStack assumes. Required when `role_based_auth` is `true`. At most 100 characters.
- `access_key_id`, `access_key` — AWS access key pair. Required when `role_based_auth` is `false`. At most 100 characters each. **Secret**.
- `enable_sse` — boolean, default `false`. Enable server-side encryption on the bucket.

```yaml
cloud_provider: AWS
bucket_name: acme-snowflake-staging
s3:
  role_based_auth: true
  iam_role_arn: "arn:aws:iam::123456789012:role/RudderStackS3"
```

#### `gcp` — object, required

Google Cloud Storage settings.

- Required when `use_rudder_storage` is `false` and `cloud_provider` is `GCP`. Leave it unset otherwise.
- `credentials` — contents of the JSON key file for a service account with `storage.objectCreator` access to the bucket. Required. **Secret**.

```yaml
cloud_provider: GCP
bucket_name: acme-snowflake-staging
storage_integration: RUDDER_GCS_INTEGRATION
gcp:
  credentials: "{{ .SNOWFLAKE_GCS_CREDENTIALS }}"
```

#### `azure` — object, required

Azure Blob Storage settings.

- Required when `use_rudder_storage` is `false` and `cloud_provider` is `AZURE`. Leave it unset otherwise.
- `container_name` — staging container, which must already exist. Required. 3 to 63 characters of lowercase letters, digits, and single hyphens.
- `account_name` — storage account name. Required. At most 100 characters.
- `use_sas_tokens` — boolean, default `false`. `true` to authenticate with `sas_token`; omitted or `false` to use `account_key`.
- `account_key` — storage account key. Required when `use_sas_tokens` is `false`. At most 100 characters. **Secret**.
- `sas_token` — shared access signature token. Required when `use_sas_tokens` is `true`. **Secret**.

```yaml
cloud_provider: AZURE
storage_integration: RUDDER_AZURE_INTEGRATION
azure:
  container_name: rudder-staging
  account_name: acmestorage
  use_sas_tokens: false
  account_key: "{{ .SNOWFLAKE_AZURE_ACCOUNT_KEY }}"
```

### Sync scheduling

#### `sync_frequency` — string, required

How often RudderStack syncs staged events into Snowflake, in minutes. Written as a string, not a number.

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

#### `manual_sync` — boolean, internal, default `false`

Enable manual sync mode.

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

Snowflake accepts events from these source types in the mentioned connection modes:

| Source type | Connection mode |
| :-----| :-----|
| `android` | `cloud` |
| `android_kotlin` | `cloud` |
| `ios` | `cloud` |
| `ios_swift` | `cloud` |
| `web` | `cloud` |
| `unity` | `cloud` |
| `react_native` | `cloud` |
| `flutter` | `cloud` |
| `cordova` | `cloud` |
| `cloud` | `cloud` |

Every source type is `cloud` only — events reach the warehouse from RudderStack's servers, never in device mode.

> [!NOTE]
> The dashboard additionally offers Snowflake to AMP, Shopify, and cloud app sources. Rudder CLI doesn't manage those connections, so `amp`, `shopify`, and `cloud_source` are invalid here.

## Connect a source

An event stream connection to this destination is checked against two rules at `validate` time.

**The source's type must be supported.** A source's type resolves to one of the tokens above before the check — a JavaScript source resolves to `web`, and webhook and server-side SDK sources resolve to `cloud`. An unsupported type reports:

```text
destination 'snowflake-prod' (type 'snowflake') does not support source 'my-source':
source type 'amp' is not among supported source types: android, android_kotlin, ...
```

**The destination config must carry a `connection_mode` entry for that source type.** This lives on the destination spec, not on the connection spec. Without it:

```text
destination 'snowflake-prod' config has no 'connection_mode' entry for source type 'web'
```

Snowflake needs no additional config keys to connect a source of any type.

## Secrets

Rudder CLI treats nine keys as secrets: `user`, `password`, `private_key`, `private_key_passphrase`, `s3.access_key_id`, `s3.access_key`, `gcp.credentials`, `azure.account_key`, and `azure.sas_token`. Write each one you use as a `{{ .VAR }}` reference and supply the value at apply time:

```yaml
config:
  user: "{{ .SNOWFLAKE_USER }}"
  private_key: "{{ .SNOWFLAKE_PRIVATE_KEY }}"
  private_key_passphrase: "{{ .SNOWFLAKE_PRIVATE_KEY_PASSPHRASE }}"
```

```bash
export RUDDER_SNOWFLAKE_USER="RUDDER_USER"
export RUDDER_SNOWFLAKE_PRIVATE_KEY="$(cat rsa_key.p8)"
export RUDDER_SNOWFLAKE_PRIVATE_KEY_PASSPHRASE="..."
rudder-cli apply

# or
rudder-cli apply --var-file secrets.vars.yaml
```

Note that:

- The YAML that `rudder-cli import` writes may or may not include secret keys. Before you apply, make sure every secret key your configuration needs is present and populated through variable substitution.
- `s3.iam_role_arn` and `storage_integration` aren't secrets — neither grants access on its own.
