# Apache Kafka (`kafka`)

Apache Kafka is a streaming destination. RudderStack publishes each event as a message to a topic on your Kafka cluster — JSON by default, or Avro — keyed by the event's `userId`, or `anonymousId` when `userId` is absent, so one user's events stay in the same partition.

In an Apache Kafka destination spec:

- `type: kafka`
- `definition_version: 1`

> [!NOTE]
> `kafka` is an [unverified destination type](../README.md#unverified-destination-types). Rudder CLI registers it only when the `unverifiedDestinations` experimental flag is on.

## Sample configuration

```yaml
version: rudder/v1
kind: destination
metadata:
  name: kafka-prod
spec:
  id: kafka-prod
  display_name: Kafka Production
  type: kafka
  definition_version: 1
  enabled: true
  config:
    host_name: broker-1.kafka.example.com,broker-2.kafka.example.com
    port: "9093"
    topic: rudder-events

    ssl_enabled: true
    use_sasl: true
    sasl_type: sha512
    username: rudderstack
    password: "{{ .KAFKA_PASSWORD }}"

    enable_multi_topic: true
    event_type_to_topic_map:
      - from: identify
        to: rudder-identifies
      - from: page
        to: rudder-pages
    event_to_topic_map:
      - from: Order Completed
        to: rudder-orders

    connection_mode:
      web: cloud
      cloud: cloud
    consent_management:
      web:
        - provider: oneTrust
          consents:
            - analytics
```

The above example connects over SSL with SCRAM-SHA-512 authentication — see [SSL and SASL](#ssl-and-sasl). It sends `identify` and `page` events and `Order Completed` to their own topics, and every other event to `topic` — see [Topic routing](#topic-routing).

## Config keys

`config` accepts only the keys listed below. The [shared config key rules](../README.md#config-key-rules) cover unknown keys, defaults, and immutability.

> [!NOTE]
> A `{{ path || fallback }}` template resolves per event, so only the topic mappings can use one: `to` in `event_type_to_topic_map`, and both fields of `event_to_topic_map`. Rudder CLI accepts a template in most other string keys too, but the API rejects one in `host_name`, `port`, `topic`, `sasl_type`, `username`, `from` in `event_type_to_topic_map`, and the `ssh` fields — such a spec passes `validate` and fails at `apply`.

### Connection

#### `host_name` — string, required

Host name of your Kafka broker. To list several brokers, separate them with commas.

- Each name consists of dot-separated labels of letters, digits, and hyphens, with no port and no scheme — `port` supplies the port.
- Validation accepts a space after a comma, but RudderStack appends `:<port>` to each name exactly as written, so leave the spaces out.

#### `port` — string, required

Port the brokers listen on. Every name in `host_name` uses it.

- A whole number from `1` to `65535`, written as a string. An unquoted number fails validation.

```yaml
port: "9093"
```

#### `topic` — string, required

Topic RudderStack publishes events to, unless an event is routed elsewhere — see [Topic routing](#topic-routing).

- 1 to 249 characters: letters, digits, `.`, `_`, and `-`.

### SSL and SASL

`ssl_enabled` and `use_sasl` decide which of the other keys here apply. SASL works only over SSL: RudderStack doesn't support SASL on a plaintext connection.

#### `ssl_enabled` — boolean

Connect to the brokers over SSL.

- The dashboard defaults this field to `true`. Rudder CLI doesn't fill it in, and RudderStack treats an omitted value as `false`, so write `ssl_enabled: true` to connect over SSL.

#### `ca_certificate` — string

PEM-encoded CA certificate RudderStack verifies the brokers' certificates against. When it's omitted, RudderStack trusts the system's CAs, which is enough for brokers whose certificates chain to a public CA.

- Applies when `ssl_enabled` is `true`. Leave it unset otherwise.
- Not validated locally.

Write it as a YAML block scalar to keep its line breaks:

```yaml
ca_certificate: |
  -----BEGIN CERTIFICATE-----
  MIIDdzCCAl+gAwIBAgIEAgAAuTANBgkqhkiG9w0BAQUFADBaMQswCQYDVQQGEwJJ
  ...
  -----END CERTIFICATE-----
```

#### `use_sasl` — boolean

Authenticate to the brokers with SASL, using `sasl_type`, `username`, and `password`. The dashboard calls this **Enable SASL with SSL**.

- Takes effect only when `ssl_enabled` is `true`. With SSL off, RudderStack ignores it and connects without authenticating, and Rudder CLI doesn't require `username`.

#### `sasl_type` — string, default `plain`

SASL mechanism: `plain` for PLAIN, `sha256` for SCRAM-SHA-256, or `sha512` for SCRAM-SHA-512.

- Applies when `ssl_enabled` and `use_sasl` are both `true`.

#### `username` — string, required

SASL username.

- Required when `ssl_enabled` and `use_sasl` are both `true`. Leave it unset otherwise.
- 1 to 32 characters: letters, digits, `_`, and `-`.
- A missing value fails validation with `'username' is required when 'ssl_enabled' is true`, which names only the first of the two conditions.

#### `password` — string, secret

SASL password for `username`.

- Applies when `ssl_enabled` and `use_sasl` are both `true`. Rudder CLI doesn't require it.
- At most 100 characters, and must not contain line breaks.

Supply it as a `{{ .VAR }}` reference rather than a literal — see [Secrets](#secrets).

### Topic routing

RudderStack picks each event's topic in this order:

1. The topic the event names itself, in `integrations.Kafka.topic`.
2. A matching entry in `event_type_to_topic_map` or `event_to_topic_map`, when `enable_multi_topic` is `true`.
3. `topic`.

#### `enable_multi_topic` — boolean, default `false`

Route events to topics by event type and `track` event name, using `event_type_to_topic_map` and `event_to_topic_map`. When `false`, RudderStack ignores both mappings.

#### `event_type_to_topic_map` — array of objects

Maps event types to topics. `track` events are routed by name instead, through `event_to_topic_map`.

- Applies when `enable_multi_topic` is `true`.
- `from` — `identify`, `page`, `screen`, `group`, or `alias`.
- `to` — topic name. At most 100 characters, and must not contain line breaks. Unlike `topic`, its characters aren't checked. A `{{ path || fallback }}` template is accepted in place of a literal.

```yaml
event_type_to_topic_map:
  - from: identify
    to: rudder-identifies
```

#### `event_to_topic_map` — array of objects

Maps `track` event names to topics.

- Applies when `enable_multi_topic` is `true`.
- `from` — `track` event name, matched exactly, including case. At most 100 characters, and must not contain line breaks.
- `to` — topic name, on the same terms as in `event_type_to_topic_map`.
- Both fields accept a `{{ path || fallback }}` template in place of a literal.

```yaml
event_to_topic_map:
  - from: Order Completed
    to: rudder-orders
```

### Avro serialization

#### `convert_to_avro` — boolean

Serialize each event with Avro instead of publishing it as JSON. Each event selects its schema by ID, in `integrations.Kafka.schemaId`:

```json
{
  "integrations": {
    "Kafka": {
      "schemaId": "1"
    }
  }
}
```

- With this on, an event fails if it names no schema, names one that isn't in `avro_schemas`, or doesn't fit its schema. Serialization is strict: the schema has to cover every field in the event.

#### `avro_schemas` — array of objects, required

Avro schemas events can be serialized with.

- Required when `convert_to_avro` is `true`.
- `schema_id` — ID an event names to select the schema.
- `schema` — the Avro schema, as a JSON string.
- Both fields are required in every entry, even when `convert_to_avro` is `false`, and neither is validated further locally.
- A schema that doesn't parse stops delivery to this destination entirely, not just for the events that use it.

```yaml
convert_to_avro: true
avro_schemas:
  - schema_id: "1"
    schema: '{"type": "record", "name": "RudderEvent", "fields": [...]}'
```

#### `embed_avro_schema_id` — boolean

Prefix each Avro message with its schema ID, in the format Confluent Schema Registry consumers expect: a zero byte, then the ID as a 4-byte integer.

- Applies when `convert_to_avro` is `true`.
- Every `schema_id` must then be a whole number — an event using a non-numeric ID fails.

### SSH tunnel

Both keys below are internal: the dashboard offers no SSH settings for Kafka. Rudder CLI accepts them so that a destination already set up to connect through an SSH tunnel keeps that setup on update — see [Some keys aren't in the dashboard](../README.md#some-keys-arent-in-the-dashboard).

#### `use_ssh` — boolean, internal

Connect to the brokers through an SSH tunnel via a bastion host.

#### `ssh` — object, required, internal

Bastion host connection details. RudderStack logs in with a private key it holds for this destination; `public_key` is the matching public key, which belongs in the bastion host's `authorized_keys`.

- Required when `use_ssh` is `true`, with all four fields. Leave it unset otherwise.
- A spec that sets `use_ssh: true` and misses any of the four fields fails validation.
- `host` — host name or IP address of the bastion host. Validation accepts a `host:port` form, but RudderStack appends `port` to this value, so leave the port out.
- `port` — SSH port of the bastion host, from `1` to `65535`, written as a string.
- `user` — user RudderStack logs in as. 1 to 32 characters: letters, digits, `_`, and `-`.
- `public_key` — an `ssh-rsa`, `ssh-ed25519`, or `ssh-dss` public key, optionally followed by a `user@host` comment.

```yaml
use_ssh: true
ssh:
  host: bastion.example.com
  port: "22"
  user: rudderstack
  public_key: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl rudderstack@bastion"
```

### Per-source keys

Both keys are objects keyed by the local source type — the tokens listed under [Source types](#source-types). A key naming a source type this destination doesn't support fails validation.

#### `connection_mode` — object

Maps each source type you connect to the mode its events reach Kafka in, using the modes in [Source types](#source-types).

- An entry is required for each source type you connect — see [Connect a source](#connect-a-source).

```yaml
connection_mode:
  web: cloud
  cloud: cloud
```

#### `consent_management` — object

Consent provider configuration per source type. The entry shape, accepted providers, and the rules on `resolution_strategy` and `consents` are shared across all destination types — see [Consent management](../common/README.md).

## Source types

Apache Kafka accepts events from these source types in the mentioned connection modes:

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

Every source type is `cloud` only — events reach Kafka from RudderStack's servers, never in device mode.

The dashboard also accepts `warehouse` for Apache Kafka, but this definition doesn't, so Rudder CLI can't connect a Reverse ETL source to it.

## Connect a source

An event stream connection to this destination is checked against two rules at `validate` time.

**The source's type must be supported.** A source's type resolves to one of the tokens above before the check — a JavaScript source resolves to `web`, and webhook and server-side SDK sources resolve to `cloud`. Every event stream source type is supported here, so only a Reverse ETL connection, whose source resolves to `warehouse`, fails it:

```text
destination 'kafka-prod' (type 'kafka') does not accept rETL sources: source type 'warehouse' is not among supported source types: android, android_kotlin, ...
```

**The destination config must carry a `connection_mode` entry for that source type.** This lives on the destination spec, not on the connection spec. Without it:

```text
destination 'kafka-prod' config has no 'connection_mode' entry for source type 'web'
```

Apache Kafka needs no additional config keys to connect a source of any type.

## Secrets

`password` is the only secret key, and applies only when `ssl_enabled` and `use_sasl` are both `true`. Write it as a `{{ .VAR }}` reference — [Secrets](../README.md#secrets) covers supplying the values and what `import` writes:

```yaml
config:
  password: "{{ .KAFKA_PASSWORD }}"
```

`ca_certificate` isn't a secret — a CA certificate is public by design.
