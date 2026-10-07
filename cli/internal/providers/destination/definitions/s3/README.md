# Amazon S3 (`s3`)

Amazon S3 is an object storage destination. RudderStack batches events and writes them as files into an S3 bucket you own.

In a S3 destination spec:

- `type: s3`
- `definition_version: 1`

## Sample configuration

```yaml
version: rudder/v1
kind: destination
metadata:
  name: amazon-s3-prod
spec:
  id: amazon-s3-prod
  display_name: Amazon S3 Production
  type: s3
  definition_version: 1
  enabled: true
  config:
    bucket_name: rudder-events-prod
    prefix: rudder/events
    role_based_auth: true
    iam_role_arn: "arn:aws:iam::123456789012:role/RudderStackS3Access"
    enable_sse: false

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

The above example uses role-based authentication, so it carries no access keys. Which of `iam_role_arn`, `access_key_id`, and `access_key` you set depends on `role_based_auth` — see [Authentication](#authentication).

## Config keys

`config` accepts only the keys listed below. The [shared config key rules](../README.md#config-key-rules) cover unknown keys, defaults, and immutability.

### Bucket

#### `bucket_name` — string, required

Name of the S3 bucket RudderStack writes event files to. The bucket must already exist.

- At most 100 characters, and must not contain line breaks.
- A `{{ path || fallback }}` template is accepted in place of a literal, and isn't measured against the length limit.

#### `prefix` — string

Folder prefix inside the bucket. RudderStack creates a folder with this name and writes all files beneath it, at `s3://<bucket_name>/<prefix>/`.

- At most 100 characters, and must not contain line breaks.
- Templates are accepted on the same terms as `bucket_name`.

#### `enable_sse` — boolean, default `false`

Enable server-side encryption. When `true`, RudderStack adds the header `x-amz-server-side-encryption: AES256` to each `PutObject` request.

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
- Templates are accepted.

#### `access_key_id` — string, required, secret

AWS access key ID authorizing RudderStack to write to the bucket.

- Required when `role_based_auth` is `false`. Leave it unset otherwise.

Supply it as a `{{ .VAR }}` reference rather than a literal — see [Secrets](#secrets).

#### `access_key` — string, required, secret

AWS secret access key matching `access_key_id`.

- Required when `role_based_auth` is `false`.

> [!WARNING]
> RudderStack recommends role-based authentication. The access key method is deprecated and will be discontinued.

> [!NOTE]
> Either method needs a bucket policy granting RudderStack write access. Role-based authentication on its own doesn't grant it.

### Per-source keys

Both keys are objects keyed by the local source type — the tokens listed under [Source types](#source-types). A key naming a source type this destination doesn't support fails validation.

#### `connection_mode` — object

Maps each source type you connect to the mode its events reach S3 in, using the modes in [Source types](#source-types).

- An entry is required for each source type you connect — see [Connect a source](#connect-a-source).

```yaml
connection_mode:
  web: cloud
  android_kotlin: cloud
```

#### `consent_management` — object

Consent provider configuration per source type. The entry shape, accepted providers, and the rules on `resolution_strategy` and `consents` are shared across all destination types — see [Consent management](../common/README.md).

## Source types

Amazon S3 accepts events from these source types in the mentioned connection modes:

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
| `warehouse` | `cloud` |

Every source type is `cloud` only — events reach the bucket from RudderStack's servers, never in device mode.

`warehouse` is the token a Reverse ETL source resolves to — see [Source types](../README.md#source-types).

## Connect a source

An event stream connection to this destination is checked against two rules at `validate` time.

**The source's type must be supported.** A source's type resolves to one of the tokens above before the check — a JavaScript source resolves to `web`, and webhook and server-side SDK sources resolve to `cloud`. Every type an event stream or Reverse ETL source can resolve to is supported here, so this check always passes.

**The destination config must carry a `connection_mode` entry for that source type.** This lives on the destination spec, not on the connection spec. Without it:

```text
destination 'amazon-s3-prod' config has no 'connection_mode' entry for source type 'web'
```

Amazon S3 needs no additional config keys to connect a source of any type.

A Reverse ETL connection reaches this destination as source type `warehouse` and is checked against the same rules, so the config needs a `connection_mode.warehouse: cloud` entry for it.

## Secrets

`access_key_id` and `access_key` are the secret keys, and apply only when `role_based_auth` is `false`. Write each as a `{{ .VAR }}` reference — [Secrets](../README.md#secrets) covers supplying the values and what `import` writes:

```yaml
config:
  bucket_name: rudder-events-prod
  role_based_auth: false
  access_key_id: "{{ .AWS_ACCESS_KEY_ID }}"
  access_key: "{{ .AWS_SECRET_ACCESS_KEY }}"
```

`iam_role_arn` isn't a secret — an ARN identifies a role but grants nothing on its own.
