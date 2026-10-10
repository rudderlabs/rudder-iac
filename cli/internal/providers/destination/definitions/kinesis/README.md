# Amazon Kinesis (`kinesis`)

Amazon Kinesis is a streaming destination. RudderStack writes each event as a record to a Kinesis data stream you own.

In an Amazon Kinesis destination spec:

- `type: kinesis`
- `definition_version: 1`

> [!NOTE]
> `kinesis` is an [unverified destination type](../README.md#unverified-destination-types). Rudder CLI registers it only when the `unverifiedDestinations` experimental flag is on.

## Sample configuration

```yaml
version: rudder/v1
kind: destination
metadata:
  name: kinesis-prod
spec:
  id: kinesis-prod
  display_name: Kinesis Production
  type: kinesis
  definition_version: 1
  enabled: true
  config:
    region: us-east-1
    stream: rudder-events-prod
    role_based_auth: true
    iam_role_arn: "arn:aws:iam::123456789012:role/RudderStackKinesisAccess"
    use_message_id: false

    connection_mode:
      web: cloud
      cloud: cloud
    consent_management:
      web:
        - provider: oneTrust
          consents:
            - analytics
```

The above example uses role-based authentication, so it carries no access keys. Which of `iam_role_arn`, `access_key_id`, and `access_key` you set depends on `role_based_auth` — see [Authentication](#authentication).

## Config keys

`config` accepts only the keys listed below. The [shared config key rules](../README.md#config-key-rules) cover unknown keys, defaults, and immutability.

### Stream

#### `region` — string, required

AWS region the Kinesis stream was created in, for example `us-east-1`.

- At most 100 characters, and must not contain line breaks.
- A `{{ path || fallback }}` template is accepted in place of a literal, and isn't measured against the length limit.

#### `stream` — string, required

Name of the Kinesis data stream RudderStack writes records to. The stream must already exist.

- At most 100 characters, and must not contain line breaks.
- Templates are accepted on the same terms as `region`.

#### `use_message_id` — boolean, default `false`

Use the event's `messageId` as the record's partition key. By default RudderStack partitions by `userId`, or `anonymousId` when `userId` is absent. `messageId` spreads records more evenly across the stream's shards.

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

ARN of the IAM role RudderStack assumes to write to the stream.

- Required when `role_based_auth` is `true`. Leave it unset otherwise.
- At most 100 characters, and must not contain line breaks.
- Templates are accepted.

#### `access_key_id` — string, required, secret

AWS access key ID authorizing RudderStack to write to the stream.

- Required when `role_based_auth` is `false`. Leave it unset otherwise.
- At most 100 characters, and must not contain line breaks.

Supply it as a `{{ .VAR }}` reference rather than a literal — see [Secrets](#secrets).

#### `access_key` — string, required, secret

AWS secret access key matching `access_key_id`.

- Required when `role_based_auth` is `false`.
- At most 100 characters, and must not contain line breaks.

> [!WARNING]
> RudderStack recommends role-based authentication. The access key method is deprecated.

> [!NOTE]
> Either method needs an IAM policy granting RudderStack `kinesis:PutRecord` on the stream. Role-based authentication on its own doesn't grant it.

### Per-source keys

Both keys are objects keyed by the local source type — the tokens listed under [Source types](#source-types). A key naming a source type this destination doesn't support fails validation.

#### `connection_mode` — object

Maps each source type you connect to the mode its events reach Kinesis in, using the modes in [Source types](#source-types).

- An entry is required for each source type you connect — see [Connect a source](#connect-a-source).

```yaml
connection_mode:
  web: cloud
  cloud: cloud
```

#### `consent_management` — object

Consent provider configuration per source type. The entry shape, accepted providers, and the rules on `resolution_strategy` and `consents` are shared across all destination types — see [Consent management](../common/README.md).

## Source types

Amazon Kinesis accepts events from these source types in the mentioned connection modes:

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

Every source type is `cloud` only — events reach the stream from RudderStack's servers, never in device mode.

The dashboard also accepts `warehouse` for Amazon Kinesis, but this definition doesn't, so Rudder CLI can't connect a Reverse ETL source to it.

## Connect a source

An event stream connection to this destination is checked against two rules at `validate` time.

**The source's type must be supported.** A source's type resolves to one of the tokens above before the check — a JavaScript source resolves to `web`, and webhook and server-side SDK sources resolve to `cloud`. Every event stream source type is supported here, so only a Reverse ETL connection, whose source resolves to `warehouse`, fails it:

```text
destination 'kinesis-prod' (type 'kinesis') does not accept rETL sources: source type 'warehouse' is not among supported source types: android, android_kotlin, ...
```

**The destination config must carry a `connection_mode` entry for that source type.** This lives on the destination spec, not on the connection spec. Without it:

```text
destination 'kinesis-prod' config has no 'connection_mode' entry for source type 'web'
```

Amazon Kinesis needs no additional config keys to connect a source of any type.

## Secrets

`access_key_id` and `access_key` are the secret keys, and apply only when `role_based_auth` is `false`. Write each as a `{{ .VAR }}` reference — [Secrets](../README.md#secrets) covers supplying the values and what `import` writes:

```yaml
config:
  region: us-east-1
  stream: rudder-events-prod
  role_based_auth: false
  access_key_id: "{{ .AWS_ACCESS_KEY_ID }}"
  access_key: "{{ .AWS_SECRET_ACCESS_KEY }}"
```

`iam_role_arn` isn't a secret — an ARN identifies a role but grants nothing on its own.
