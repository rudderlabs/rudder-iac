# Amazon S3 (`s3`)

Object-storage destination. RudderStack writes event files to an S3 bucket from
its own servers. Every supported source type connects in cloud mode — there is
no device-mode variant.

In a destination spec:

- `type: s3`
- `definition_version: 1`

## Example

Role-based authentication, which is the recommended setup:

```yaml
version: rudder/v1
kind: destination
metadata:
  name: s3
spec:
  id: s3
  display_name: Amazon S3
  type: s3
  definition_version: 1
  enabled: true
  config:
    bucket_name: my-rudder-events
    prefix: rudder/events
    role_based_auth: true
    iam_role_arn: "arn:aws:iam::123456789012:role/RudderStackS3"
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

With access keys instead, replace the auth block:

```yaml
    role_based_auth: false
    access_key_id: "{{ .AWS_ACCESS_KEY_ID }}"
    access_key: "{{ .AWS_SECRET_ACCESS_KEY }}"
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

### Bucket

#### `bucket_name` — string, required

The name of your S3 bucket. At most 100 characters.

#### `prefix` — string

A path prefix RudderStack applies to every file it stores in the bucket. At most
100 characters.

#### `enable_sse` — boolean, default `false`

Enable server-side encryption on the objects RudderStack writes.

### Authentication

`role_based_auth` selects between the two authentication paths, and decides which
of the three keys below are required.

#### `role_based_auth` \* — boolean, required

Authenticate with an IAM role rather than access keys. Required — there is no
default, so every spec must state which path it uses.

#### `iam_role_arn` \* — string

ARN of the IAM role RudderStack assumes. **Required when `role_based_auth` is
`true`.** At most 100 characters.

#### `access_key_id` — string, secret

Your AWS access key ID. **Required when `role_based_auth` is `false`.** At most
100 characters.

Supply it as a `{{ .VAR }}` reference — see [Secrets](#secrets).

#### `access_key` — string, secret

Your AWS secret access key. **Required when `role_based_auth` is `false`.** At
most 100 characters.

Supply it as a `{{ .VAR }}` reference — see [Secrets](#secrets).

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
destination 's3' (type 's3') does not support source 'my-source':
source type 'amp' is not among supported source types: android, android_kotlin, ...
```

**The config must carry a `connection_mode` entry for that source type.** This
lives on the destination spec, not the connection spec. Without it:

```
destination 's3' config has no 'connection_mode' entry for source type 'web'
```

S3 requires no additional config keys to connect a source of any type.

## Secrets

`access_key_id` and `access_key` are the secret keys, and apply only when
`role_based_auth` is `false`. Write each as a `{{ .VAR }}` reference and supply
the value at apply time:

```yaml
access_key_id: "{{ .AWS_ACCESS_KEY_ID }}"
access_key: "{{ .AWS_SECRET_ACCESS_KEY }}"
```

```sh
export RUDDER_AWS_ACCESS_KEY_ID=...
export RUDDER_AWS_SECRET_ACCESS_KEY=...
rudder-cli apply

# or
rudder-cli apply --var-file secrets.vars.yaml
```

`rudder-cli import` writes each back as a `{{ .VAR }}` placeholder rather than
its value, since the API does not return secrets. Fill the placeholders in
before the first apply.
