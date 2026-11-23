# Amazon Redshift (`rs`)

Amazon Redshift is a warehouse destination. RudderStack stages events as files in S3, then loads them into a Redshift cluster or Serverless workgroup on a schedule.

In an Amazon Redshift destination spec:

- `type: rs`
- `definition_version: 1`

## Sample configuration

```yaml
version: rudder/v1
kind: destination
metadata:
  name: redshift-prod
spec:
  id: redshift-prod
  display_name: Redshift Production
  type: rs
  definition_version: 1
  enabled: true
  config:
    database: analytics
    user: "{{ .REDSHIFT_USER }}"
    namespace: rudder_events

    use_iam_for_auth: true
    iam_role_arn_for_auth: "arn:aws:iam::123456789012:role/RudderStackRedshift"
    cluster_region: us-east-1
    use_serverless: false
    cluster_id: analytics-cluster

    use_ssh: false

    use_rudder_storage: false
    bucket_name: acme-redshift-staging
    prefix: rudder
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

The above example connects to a provisioned cluster with IAM authentication and stages files in your own bucket through an IAM role, so it carries no password or access keys. Which keys apply depends on several switches — see [Key dependencies](#key-dependencies).

## Config keys

`config` accepts only the keys listed below. The [shared config key rules](../README.md#config-key-rules) cover unknown keys, defaults, and immutability.

### Key dependencies

`use_iam_for_auth`, `use_serverless`, `use_rudder_storage`, `role_based_auth`, and `use_ssh` decide which other keys apply. Rudder CLI enforces the requirements marked **Enforced**; the rest are accepted whatever the switch says, so a key that doesn't apply is stored and ignored.

| Key | Applies when | Required then |
| :-----| :-----| :-----|
| `host`, `port`, `password` | `use_iam_for_auth` is `false` | Enforced |
| `iam_role_arn_for_auth`, `cluster_region` | `use_iam_for_auth` is `true` | Enforced |
| `cluster_id` | `use_iam_for_auth` is `true` and `use_serverless` is `false` | Enforced |
| `workgroup_name` | `use_iam_for_auth` is `true` and `use_serverless` is `true` | Enforced |
| `ssh` | `use_ssh` is `true` | Enforced, all four fields |
| `bucket_name` | `use_rudder_storage` is `false` | Enforced |
| `iam_role_arn` | `use_rudder_storage` is `false` and `role_based_auth` is `true`, which is the default | Enforced |
| `prefix`, `role_based_auth`, `enable_sse`, `cleanup_object_storage_files` | `use_rudder_storage` is `false` | No |
| `access_key_id`, `access_key` | `use_rudder_storage` is `false` and `role_based_auth` is `false` | No |

> [!WARNING]
> `validate` fills in defaults before it checks the spec, so an omitted key is checked as though you had written its default. Two keys are worth knowing about:
>
> - Omitting `use_serverless` selects a provisioned cluster, which makes `cluster_id` required.
> - With your own storage, omitting `role_based_auth` selects role-based authentication, which makes `iam_role_arn` required.

### Connection

#### `database` — string, required

Name of the Redshift database RudderStack loads data into.

- At most 100 characters, and must not contain line breaks.
- A `{{ path || fallback }}` template is accepted in place of a literal.

#### `user` — string, required, secret

Database user with read and write access to `database`.

- At most 100 characters, and must not contain line breaks.
- A `{{ path || fallback }}` template is accepted in place of a literal.

#### `namespace` — string

Schema RudderStack creates its tables in. Defaults to the source name when omitted.

- At most 64 characters, and must not start with `pg_` in any capitalization.
- A `{{ path || fallback }}` template is accepted in place of a literal.
- The Redshift setup guide says the namespace can't be changed later, so treat it as fixed. Unlike BigQuery's, it isn't marked immutable in the API schema, so a change isn't rejected.

### Authentication

`use_iam_for_auth` selects between a database password and the RudderStack IAM role. With IAM, `use_serverless` then selects between a provisioned cluster and a Serverless workgroup.

#### `use_iam_for_auth` — boolean, required

Authenticate with the RudderStack IAM role instead of a database password.

- The dashboard defaults this field to `false`. Rudder CLI requires it explicitly.
- A spec that omits this key fails validation.

#### `host` — string, required

Hostname of the Redshift cluster endpoint.

- Required when `use_iam_for_auth` is `false`. Leave it unset otherwise.
- 1 to 255 characters, and must not contain line breaks. An `ngrok.io` host is rejected.
- A `{{ path || fallback }}` template is accepted in place of a literal.

#### `port` — string, required

Port of the Redshift cluster endpoint, written as a string — `"5439"`, not `5439`.

- Required when `use_iam_for_auth` is `false`. Leave it unset otherwise.
- At most 100 characters, and must not contain line breaks.

#### `password` — string, required, secret

Password for `user`.

- Required when `use_iam_for_auth` is `false`. Leave it unset otherwise.

Supply it as a `{{ .VAR }}` reference rather than a literal — see [Secrets](#secrets).

#### `iam_role_arn_for_auth` — string, required

ARN of the RudderStack IAM role used to obtain database credentials.

- Required when `use_iam_for_auth` is `true`. Leave it unset otherwise.
- At most 100 characters, and must not contain line breaks.
- A `{{ path || fallback }}` template is accepted in place of a literal.

#### `cluster_region` — string, required

AWS region of the cluster or workgroup — for example `us-east-1`.

- Required when `use_iam_for_auth` is `true`. Leave it unset otherwise.
- 1 to 255 characters, and must not contain line breaks.

#### `use_serverless` — boolean, default `false`

Connect to a Redshift Serverless workgroup instead of a provisioned cluster.

- Applies when `use_iam_for_auth` is `true`.

See [Key dependencies](#key-dependencies) for what omitting it makes required.

#### `cluster_id` — string, required

Identifier of the provisioned Redshift cluster.

- Required when `use_iam_for_auth` is `true` and `use_serverless` is `false`. Leave it unset otherwise.
- 1 to 255 characters, and must not contain line breaks.

#### `workgroup_name` — string, required

Name of the Redshift Serverless workgroup.

- Required when `use_iam_for_auth` is `true` and `use_serverless` is `true`. Leave it unset otherwise.
- 1 to 255 characters, and must not contain line breaks.

### SSH tunnel

> [!NOTE]
> SSH tunneling is available on the Enterprise plan.

#### `use_ssh` — boolean, default `false`

Connect to Redshift through an SSH tunnel via a bastion host.

#### `ssh` — object, required

Bastion host connection details.

- Required when `use_ssh` is `true`, with all four fields. Leave it unset otherwise.
- `host` — IP address or hostname of the bastion host. At most 100 characters.
- `port` — SSH port of the bastion host, as a string. At most 100 characters.
- `user` — user RudderStack logs in to the bastion host as. **Secret** — see [Secrets](#secrets).
- `public_key` — the public key RudderStack generates for this destination.

```yaml
use_ssh: true
ssh:
  host: 203.0.113.10
  port: "22"
  user: "{{ .REDSHIFT_SSH_USER }}"
  public_key: "ssh-rsa AAAA..."
```

RudderStack holds the private key; add `public_key` to the bastion host's `authorized_keys`. The key comes from RudderStack, so the practical route is to enable SSH on the destination in the dashboard, then import it to pick up the value.

### Object storage

`use_rudder_storage` decides whether RudderStack stages files in its own bucket or in yours. The keys below it apply only to your own bucket.

#### `use_rudder_storage` — boolean, required

Stage files in RudderStack-managed object storage instead of your own S3 bucket.

- Available only on RudderStack-hosted data planes. Self-hosted data planes must set `false` and configure a bucket.
- The dashboard defaults this field to `false`. Rudder CLI requires it explicitly.

#### `bucket_name` — string, required

Name of the staging S3 bucket. The bucket must already exist.

- Required when `use_rudder_storage` is `false`. Leave it unset otherwise.
- 3 to 63 characters: lowercase letters, digits, dots, and hyphens, starting and ending with a letter or digit. Must not start with `xn--`, contain consecutive dots, or look like an IPv4 address.
- A `{{ path || fallback }}` template is accepted in place of a literal.

#### `prefix` — string

Folder prefix inside the staging bucket, at `s3://<bucket_name>/<prefix>/`.

- Applies when `use_rudder_storage` is `false`. Leave it unset otherwise.
- At most 100 characters, with no whitespace.

#### `role_based_auth` — boolean, default `true`

Access the staging bucket with an IAM role (`iam_role_arn`) rather than an access key pair.

- Applies when `use_rudder_storage` is `false`. Leave it unset otherwise.

See [Key dependencies](#key-dependencies) for what leaving it at its default makes required.

#### `iam_role_arn` — string, required

ARN of the IAM role RudderStack assumes to read and write the staging bucket.

- Required when `use_rudder_storage` is `false` and `role_based_auth` is `true`. Leave it unset otherwise.
- At most 100 characters, and must not contain line breaks.

#### `access_key_id` — string, secret

AWS access key ID for the staging bucket.

- Applies when `use_rudder_storage` is `false` and `role_based_auth` is `false`. Leave it unset otherwise.
- Not required even then — neither Rudder CLI nor the API asks for it.
- At most 100 characters, and must not contain line breaks.

#### `access_key` — string, secret

AWS secret access key matching `access_key_id`.

- Applies on the same terms as `access_key_id`, and is likewise not required.
- At most 100 characters, and must not contain line breaks.

> [!WARNING]
> Access key authentication is deprecated. Use `role_based_auth: true` with `iam_role_arn`.

#### `enable_sse` — boolean, default `false`

Enable server-side encryption on the staging bucket.

- Applies when `use_rudder_storage` is `false`.

#### `cleanup_object_storage_files` — boolean, default `false`

Delete staged files from the bucket after a sync completes successfully.

- Applies when `use_rudder_storage` is `false`.

### Sync scheduling

#### `sync_frequency` — string, required

How often RudderStack syncs staged events into Redshift, in minutes. Written as a string, not a number.

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

```yaml
exclude_window:
  start_time: "02:00"
  end_time: "03:00"
```

### Table behavior

#### `prefer_append` — boolean, default `true`

Append incoming events to existing tables. Set it to `false` to merge instead, which guarantees no duplicates at the cost of noticeably longer syncs. This is what the dashboard calls **Warehouse Append**.

Appending can let duplicates through — most often SDK retries, and especially against data older than 7 days.

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

Maps each source type you connect to the mode its events reach Redshift in, using the modes in [Source types](#source-types).

- An entry is required for each source type you connect — see [Connect a source](#connect-a-source).

```yaml
connection_mode:
  web: cloud
  cloud: cloud
```

#### `consent_management` — object

Consent provider configuration per source type. The entry shape, accepted providers, and the rules on `resolution_strategy` and `consents` are shared across all destination types — see [Consent management](../common/README.md).

## Source types

Amazon Redshift accepts events from these source types in the mentioned connection modes:

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

## Connect a source

An event stream connection to this destination is checked against two rules at `validate` time.

**The source's type must be supported.** A source's type resolves to one of the tokens above before the check — a JavaScript source resolves to `web`, and webhook and server-side SDK sources resolve to `cloud`. Every event stream source type is supported here, so only a Reverse ETL connection, whose source resolves to `warehouse`, fails it:

```text
destination 'redshift-prod' (type 'rs') does not accept rETL sources: source type 'warehouse' is not among supported source types: android, android_kotlin, ...
```

**The destination config must carry a `connection_mode` entry for that source type.** This lives on the destination spec, not on the connection spec. Without it:

```text
destination 'redshift-prod' config has no 'connection_mode' entry for source type 'web'
```

Amazon Redshift needs no additional config keys to connect a source of any type.

## Secrets

Rudder CLI treats five keys as secrets: `user`, `password`, `access_key_id`, `access_key`, and `ssh.user`. Write each one you use as a `{{ .VAR }}` reference — [Secrets](../README.md#secrets) covers supplying the values and what `import` writes:

```yaml
config:
  user: "{{ .REDSHIFT_USER }}"
  password: "{{ .REDSHIFT_PASSWORD }}"
```

`iam_role_arn_for_auth` and `iam_role_arn` aren't secrets — an ARN identifies a role but grants nothing on its own.
