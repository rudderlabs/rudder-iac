# Confluent Cloud (`confluent_cloud`)

Confluent Cloud is a streaming destination — a fully managed Apache Kafka service. RudderStack publishes each event as a JSON message to a topic in your Confluent Cloud cluster, keyed by the event's `userId`, or `anonymousId` when `userId` is absent, so one user's events stay in the same partition.

In a Confluent Cloud destination spec:

- `type: confluent_cloud`
- `definition_version: 1`

> [!NOTE]
> `confluent_cloud` is an [unverified destination type](../README.md#unverified-destination-types). Rudder CLI registers it only when the `unverifiedDestinations` experimental flag is on.

## Sample configuration

```yaml
version: rudder/v1
kind: destination
metadata:
  name: confluent-cloud-prod
spec:
  id: confluent-cloud-prod
  display_name: Confluent Cloud Production
  type: confluent_cloud
  definition_version: 1
  enabled: true
  config:
    bootstrap_server: pkc-a1b2c.us-east-1.aws.confluent.cloud:9092
    topic: rudder-events
    api_key: "{{ .CONFLUENT_API_KEY }}"
    api_secret: "{{ .CONFLUENT_API_SECRET }}"

    connection_mode:
      web: cloud
      cloud: cloud
    consent_management:
      web:
        - provider: oneTrust
          consents:
            - analytics
```

The above example keeps the API key pair out of the YAML with `{{ .VAR }}` references — see [Secrets](#secrets). Every key in it except the per-source keys is required.

## Config keys

`config` accepts only the keys listed below. The [shared config key rules](../README.md#config-key-rules) cover unknown keys, defaults, and immutability.

> [!NOTE]
> Confluent Cloud has no topic routing or Avro settings. For those, or for a self-managed cluster, see [Apache Kafka](../kafka/README.md).

### Connection

#### `bootstrap_server` — string, required

Bootstrap server of your Confluent Cloud cluster, as `host:port` — for example `pkc-a1b2c.us-east-1.aws.confluent.cloud:9092`. The cluster settings in Confluent Cloud show it.

- At most 100 characters, and must not contain line breaks.

#### `topic` — string, required

Topic RudderStack publishes events to. An event can name a different topic in `integrations.CONFLUENT_CLOUD.topic`, which takes precedence.

- At most 100 characters, and must not contain line breaks. Kafka's rules on topic names aren't checked locally.
- A `{{ path || fallback }}` template resolves per event, but is measured as literal text against the same limit.

### Authentication

RudderStack connects over TLS and authenticates with SASL/PLAIN, using `api_key` as the username and `api_secret` as the password.

#### `api_key` — string, required, secret

API key of your Confluent Cloud cluster, created in the Confluent Cloud console.

- At most 100 characters, and must not contain line breaks.

Supply it as a `{{ .VAR }}` reference rather than a literal — see [Secrets](#secrets).

#### `api_secret` — string, required, secret

Secret of `api_key`.

- At most 100 characters, and must not contain line breaks.

### Per-source keys

Both keys are objects keyed by the local source type — the tokens listed under [Source types](#source-types). A key naming a source type this destination doesn't support fails validation.

#### `connection_mode` — object

Maps each source type you connect to the mode its events reach Confluent Cloud in, using the modes in [Source types](#source-types).

- An entry is required for each source type you connect — see [Connect a source](#connect-a-source).

```yaml
connection_mode:
  web: cloud
  cloud: cloud
```

#### `consent_management` — object

Consent provider configuration per source type. The entry shape, accepted providers, and the rules on `resolution_strategy` and `consents` are shared across all destination types — see [Consent management](../common/README.md).

## Source types

Confluent Cloud accepts events from these source types in the mentioned connection modes:

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

Every source type is `cloud` only — events reach Confluent Cloud from RudderStack's servers, never in device mode.

The dashboard also accepts `warehouse` for Confluent Cloud, but this definition doesn't, so Rudder CLI can't connect a Reverse ETL source to it.

## Connect a source

An event stream connection to this destination is checked against two rules at `validate` time.

**The source's type must be supported.** A source's type resolves to one of the tokens above before the check — a JavaScript source resolves to `web`, and webhook and server-side SDK sources resolve to `cloud`. Every event stream source type is supported here, so only a Reverse ETL connection, whose source resolves to `warehouse`, fails it:

```text
destination 'confluent-cloud-prod' (type 'confluent_cloud') does not accept rETL sources: source type 'warehouse' is not among supported source types: android, android_kotlin, ...
```

**The destination config must carry a `connection_mode` entry for that source type.** This lives on the destination spec, not on the connection spec. Without it:

```text
destination 'confluent-cloud-prod' config has no 'connection_mode' entry for source type 'web'
```

Confluent Cloud needs no additional config keys to connect a source of any type.

## Secrets

`api_key` and `api_secret` are the secret keys. Write each as a `{{ .VAR }}` reference — [Secrets](../README.md#secrets) covers supplying the values and what `import` writes:

```yaml
config:
  api_key: "{{ .CONFLUENT_API_KEY }}"
  api_secret: "{{ .CONFLUENT_API_SECRET }}"
```
