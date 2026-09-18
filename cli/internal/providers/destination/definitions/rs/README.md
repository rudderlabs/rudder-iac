# Amazon Redshift (`rs`)

Warehouse destination. RudderStack stages events as files in an S3 bucket, then
loads them into a Redshift cluster or serverless workgroup on a schedule. Every
supported source type connects in cloud mode — there is no device-mode variant.

In a destination spec:

- `type: rs`
- `definition_version: 1`

## Choosing your setup

Three independent switches decide which keys you need. Each is required or
defaulted, so every spec makes all three choices explicitly or by default.

| Switch | Options |
| --- | --- |
| `use_iam_for_auth` | password-based connection, or IAM |
| `use_serverless` (IAM only) | provisioned cluster, or serverless workgroup |
| `use_rudder_storage` | RudderStack-hosted staging bucket, or your own S3 bucket |

## Example

IAM authentication against a provisioned cluster, staging through your own
bucket with a role:

```yaml
version: rudder/v1
kind: destination
metadata:
  name: redshift
spec:
  id: redshift
  display_name: Amazon Redshift
  type: rs
  definition_version: 1
  enabled: true
  config:
    database: analytics
    user: rudderstack
    namespace: rudder_events

    use_iam_for_auth: true
    iam_role_arn_for_auth: "arn:aws:iam::123456789012:role/RudderStackRedshift"
    cluster_region: us-east-1
    use_serverless: false
    cluster_id: analytics-cluster

    use_ssh: true
    ssh:
      host: bastion.example.com
      port: "22"
      user: rudder
      public_key: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5rudder rudder@example"

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
    bucket_name: my-rudder-staging
    prefix: rudder/events
    role_based_auth: true
    iam_role_arn: "arn:aws:iam::123456789012:role/RudderStackS3"
    enable_sse: true
    cleanup_object_storage_files: false

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

#### `database` — string, required

The database name in your Redshift instance where the data will be sent. At most
100 characters.

#### `user` — string, required

The name of the user with read/write access to that database. At most 100
characters.

#### `namespace` — string

The schema name where RudderStack creates all its tables. Defaults to the source
name when omitted.

### Authentication

`use_iam_for_auth` selects between the two paths and decides which keys below are
required.

#### `use_iam_for_auth` \* — boolean, required

Authenticate with an IAM role rather than a host, port and password. Required —
there is no default, so every spec states which path it uses.

#### `host` — string

The host name of your Redshift service. **Required when `use_iam_for_auth` is
`false`.**

#### `port` — string

The port associated with the Redshift database instance. **Required when
`use_iam_for_auth` is `false`.** At most 100 characters.

#### `password` — string, secret

The password for the user above. **Required when `use_iam_for_auth` is `false`.**

Supply it as a `{{ .VAR }}` reference — see [Secrets](#secrets).

#### `iam_role_arn_for_auth` \* — string

ARN of the IAM role RudderStack assumes to connect. **Required when
`use_iam_for_auth` is `true`.** At most 100 characters.

#### `cluster_region` \* — string

AWS region your Redshift cluster or workgroup runs in. **Required when
`use_iam_for_auth` is `true`.** At most 255 characters.

#### `use_serverless` \* — boolean, default `false`

Connect to a Redshift Serverless workgroup rather than a provisioned cluster.
**Required when `use_iam_for_auth` is `true`.**

#### `cluster_id` \* — string

Identifier of the provisioned Redshift cluster. **Required when
`use_iam_for_auth` is `true` and `use_serverless` is `false`.** At most 255
characters.

#### `workgroup_name` \* — string

Name of the Redshift Serverless workgroup. **Required when `use_iam_for_auth` is
`true` and `use_serverless` is `true`.** At most 255 characters.

### SSH tunnel

#### `use_ssh` \* — boolean, default `false`

Reach Redshift through an SSH tunnel rather than connecting directly.

#### `ssh` \* — object

SSH tunnel settings, grouped under one block. **All four fields are required when
`use_ssh` is `true`**, including when the block itself is omitted:

- `host` — bastion host name, at most 100 characters
- `port` — bastion port, at most 100 characters
- `user` — SSH user, at most 100 characters
- `public_key` — the SSH public key

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

RudderStack writes files to S3 before loading them into Redshift.
`use_rudder_storage` decides whose bucket that is.

#### `use_rudder_storage` — boolean, required

Use the RudderStack-hosted object storage rather than your own bucket.

#### `bucket_name` — string

The name of your S3 bucket. **Required when `use_rudder_storage` is `false`.**

#### `prefix` \* — string

Path prefix applied to files RudderStack writes into the bucket.

#### `role_based_auth` \* — boolean, default `true`

Authenticate to the bucket with an IAM role rather than access keys.

#### `iam_role_arn` \* — string

ARN of the IAM role used for bucket access. **Required when
`use_rudder_storage` is `false` and `role_based_auth` is `true`.** At most 100
characters.

#### `access_key_id` — string, secret

Your AWS access key ID, used when `role_based_auth` is `false`. At most 100
characters.

#### `access_key` — string, secret

Your AWS secret access key, used when `role_based_auth` is `false`. At most 100
characters.

#### `enable_sse` — boolean, default `false`

Enable server-side encryption on the staged objects.

#### `cleanup_object_storage_files` \* — boolean, default `false`

Delete the staged files from the bucket after a sync completes successfully.

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
destination 'redshift' (type 'rs') does not support source 'my-source':
source type 'amp' is not among supported source types: android, android_kotlin, ...
```

**The config must carry a `connection_mode` entry for that source type.** This
lives on the destination spec, not the connection spec. Without it:

```
destination 'redshift' config has no 'connection_mode' entry for source type 'web'
```

Redshift requires no additional config keys to connect a source of any type.

## Secrets

`password`, `access_key_id` and `access_key` are the secret keys. Which apply
depends on your setup: `password` only with password-based authentication, the
two AWS keys only when staging through your own bucket without a role.

Write each as a `{{ .VAR }}` reference and supply the value at apply time:

```yaml
password: "{{ .REDSHIFT_PASSWORD }}"
```

```sh
export RUDDER_REDSHIFT_PASSWORD=...
rudder-cli apply

# or
rudder-cli apply --var-file secrets.vars.yaml
```

Note that `ssh.public_key` is **not** a secret — a public key is not sensitive,
and it is stored and returned in the clear.

`rudder-cli import` writes each secret back as a `{{ .VAR }}` placeholder rather
than its value, since the API does not return secrets. Fill the placeholders in
before the first apply.
