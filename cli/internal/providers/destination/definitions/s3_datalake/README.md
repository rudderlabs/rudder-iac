# S3 Data Lake (`s3_datalake`)

S3 Data Lake is a data lake destination. RudderStack writes events as Parquet files into an S3 bucket on a schedule, and can register their schema in the AWS Glue Data Catalog so you can query them with Athena.

In an S3 Data Lake destination spec:

- `type: s3_datalake`
- `definition_version: 1`

## Sample configuration

```yaml
version: rudder/v1
kind: destination
metadata:
  name: s3-datalake-prod
spec:
  id: s3-datalake-prod
  display_name: S3 Data Lake Production
  type: s3_datalake
  definition_version: 1
  enabled: true
  config:
    bucket_name: acme-datalake-prod
    prefix: rudder
    namespace: web_events
    enable_sse: false
    cleanup_object_storage_files: false

    use_glue: true
    region: us-east-1
    time_window_layout: dt=2006-01-02

    role_based_auth: true
    iam_role_arn: "arn:aws:iam::123456789012:role/RudderStackDatalake"

    sync_frequency: "180"
    sync_start_at: "01:00"

    skip_users_table: true
    skip_tracks_table: false

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

The above example registers the schema on AWS Glue with date-only partitions, and uses role-based authentication, so it carries no access keys — see [AWS Glue](#aws-glue) and [Authentication](#authentication).

## Config keys

`config` accepts only the keys listed below. The [shared config key rules](../README.md#config-key-rules) cover unknown keys, defaults, and immutability.

### Storage

RudderStack writes each table to `s3://<bucket_name>/<prefix>/rudder-datalake/<namespace>/<table>/`.

#### `bucket_name` — string, required

Name of the S3 bucket RudderStack writes the data lake into. The bucket must already exist.

- 3 to 63 characters: lowercase letters, digits, dots, and hyphens, starting and ending with a letter or digit.
- Must not start with `xn--`, contain consecutive dots, or look like an IPv4 address.
- A `{{ path || fallback }}` template is accepted in place of a literal.

#### `prefix` — string

Folder prefix inside the bucket, placed before `rudder-datalake/`.

- Not validated locally.

#### `namespace` — string, immutable

Folder under `rudder-datalake/` that holds this destination's tables. When `use_glue` is `true`, it's also the name of the Glue database RudderStack creates the table definitions in. Defaults to the source name when omitted.

- At most 64 characters, and must not start with `pg_` in any capitalization.
- A `{{ path || fallback }}` template is accepted in place of a literal.
- Can't be changed once the destination exists — the API rejects the update. Create a new destination instead.

#### `enable_sse` — boolean, default `false`

Enable server-side encryption on the files RudderStack writes.

#### `cleanup_object_storage_files` — boolean, default `false`

Delete intermediate object storage files after a sync completes successfully.

### AWS Glue

#### `use_glue` — boolean, default `false`

Register each table's schema in the AWS Glue Data Catalog, so you can query the data lake without running a crawler. This is what the dashboard calls **Register schema on AWS Glue**.

The IAM role or user needs Glue permissions, plus Lake Formation permissions if your account uses them.

#### `region` — string, required

AWS region of the Glue Data Catalog — for example `us-east-1`.

- Required when `use_glue` is `true`. Leave it unset otherwise.
- Not validated beyond being present.

#### `time_window_layout` — string, immutable

Partition layout of the folders under each table, written as a Go time layout.

- Applies when `use_glue` is `true`. Leave it unset otherwise.
- Omit it, or set `""`, for hourly folders — `YYYY/MM/DD/HH`.
- Set `dt=2006-01-02` for daily Hive-style folders — `dt=YYYY-MM-DD`.
- Not validated locally. Those two values are the ones the dashboard offers.
- Can't be changed once the destination exists — the API rejects the update.

### Authentication

`role_based_auth` selects the authentication method, and decides which of the remaining three keys are required.

> [!WARNING]
> Rudder CLI checks only that the keys the selected method needs are present. It doesn't reject the keys belonging to the other method, so a spec carrying both an `iam_role_arn` and an access key pair passes `validate` and applies.
>
> Leave the unused method's keys out — otherwise you store credentials the destination never reads.

#### `role_based_auth` — boolean, required

Whether to authenticate with an IAM role. Set it to `true` to use `iam_role_arn`, or `false` to use the access key pair.

- The dashboard defaults this field to `true`. Rudder CLI requires it explicitly.
- A spec that omits this key fails validation with `'role_based_auth' is required`.

#### `iam_role_arn` — string, required

ARN of the IAM role RudderStack assumes to write to the bucket.

- Required when `role_based_auth` is `true`. Leave it unset otherwise.
- At most 100 characters, and must not contain line breaks.
- A `{{ path || fallback }}` template is accepted in place of a literal.

#### `access_key_id` — string, required, secret

AWS access key ID authorizing RudderStack to write to the bucket.

- Required when `role_based_auth` is `false`. Leave it unset otherwise.
- Must not be empty or contain line breaks. There's no upper length limit.

Supply it as a `{{ .VAR }}` reference rather than a literal — see [Secrets](#secrets).

#### `access_key` — string, required, secret

AWS secret access key matching `access_key_id`.

- Required when `role_based_auth` is `false`. Leave it unset otherwise.
- Must not be empty or contain line breaks. There's no upper length limit.

### Sync scheduling

#### `sync_frequency` — string, default `180`

How often RudderStack syncs staged events into the data lake, in minutes. Written as a string, not a number.

- One of `5`, `10`, `15`, `30`, `60`, `180`, `360`, `720`, or `1440`.
- Optional here, unlike on the other warehouse destinations, where Rudder CLI requires it.

#### `sync_start_at` — string

Time of day, in UTC, that anchors the sync schedule. Subsequent syncs are computed from it at `sync_frequency` intervals. Written as `HH:MM`.

- Not validated locally: any string is accepted, and a value the scheduler can't parse silently yields no scheduled times.

### Table behavior

#### `skip_users_table` — boolean, default `true`

Send `identify` events only to the `identifies` table, skipping the `users` table. The `users` table holds one row per unique user and is maintained with a merge, which can add significant time to each sync.

#### `skip_tracks_table` — boolean, default `false`

Skip sending events to the `tracks` table. Per-event tables are unaffected.

### Legacy column naming

Both keys below preserve the column naming of destinations created before the behavior changed. Leave them at their defaults on a new destination. Neither can be changed once the destination exists — the API rejects the update.

#### `underscore_divide_numbers` — boolean, immutable, internal, default `false`

When `false`, numeric suffixes in column names are preserved: `v3` stays `v3` rather than being split into `v_3`.

#### `allow_users_context_traits` — boolean, immutable, internal, default `false`

When `false`, `context.traits.*` fields aren't promoted to top-level traits and are stored only as `context_traits_*` columns.

### Per-source keys

Both keys are objects keyed by the local source type — the tokens listed under [Source types](#source-types). A key naming a source type this destination doesn't support fails validation.

#### `connection_mode` — object

Maps each source type you connect to the mode its events reach the data lake in, using the modes in [Source types](#source-types).

- An entry is required for each source type you connect — see [Connect a source](#connect-a-source).

```yaml
connection_mode:
  web: cloud
  cloud: cloud
```

#### `consent_management` — object

Consent provider configuration per source type. The entry shape, accepted providers, and the rules on `resolution_strategy` and `consents` are shared across all destination types — see [Consent management](../common/README.md).

## Source types

S3 Data Lake accepts events from these source types in the mentioned connection modes:

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

Every source type is `cloud` only — events reach the bucket from RudderStack's servers, never in device mode.

> [!NOTE]
> The dashboard additionally offers S3 Data Lake to AMP, Shopify, and cloud app sources. Rudder CLI doesn't manage those connections, so `amp`, `shopify`, and `cloud_source` are invalid here.

## Connect a source

An event stream connection to this destination is checked against two rules at `validate` time.

**The source's type must be supported.** A source's type resolves to one of the tokens above before the check — a JavaScript source resolves to `web`, and webhook and server-side SDK sources resolve to `cloud`. An unsupported type reports:

```text
destination 's3-datalake-prod' (type 's3_datalake') does not support source 'my-source':
source type 'amp' is not among supported source types: android, android_kotlin, ...
```

**The destination config must carry a `connection_mode` entry for that source type.** This lives on the destination spec, not on the connection spec. Without it:

```text
destination 's3-datalake-prod' config has no 'connection_mode' entry for source type 'web'
```

S3 Data Lake needs no additional config keys to connect a source of any type.

## Secrets

`access_key_id` and `access_key` are the secret keys, and apply only when `role_based_auth` is `false`. Write each as a `{{ .VAR }}` reference and supply the value at apply time:

```yaml
config:
  bucket_name: acme-datalake-prod
  role_based_auth: false
  access_key_id: "{{ .AWS_ACCESS_KEY_ID }}"
  access_key: "{{ .AWS_SECRET_ACCESS_KEY }}"
```

```bash
export RUDDER_AWS_ACCESS_KEY_ID="AKIAIOSFODNN7EXAMPLE"
export RUDDER_AWS_SECRET_ACCESS_KEY="wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
rudder-cli apply

# or
rudder-cli apply --var-file secrets.vars.yaml
```

Note that:

- A `{{ .VAR }}` reference satisfies the requirement check, so a spec using access keys validates before the values are supplied.
- The YAML that `rudder-cli import` writes may or may not include secret keys. Before you apply, make sure every secret key your configuration needs is present and populated through variable substitution.
- `iam_role_arn` isn't a secret — an ARN identifies a role but grants nothing on its own.
