# Google Cloud Pub/Sub (`googlepubsub`)

Google Cloud Pub/Sub is a streaming destination. RudderStack publishes each event as a JSON message to the Pub/Sub topic its name or type maps to, optionally with message attributes copied from the event.

In a Google Cloud Pub/Sub destination spec:

- `type: googlepubsub`
- `definition_version: 1`

> [!NOTE]
> `googlepubsub` is an [unverified destination type](../README.md#unverified-destination-types). Rudder CLI registers it only when the `unverifiedDestinations` experimental flag is on.

## Sample configuration

```yaml
version: rudder/v1
kind: destination
metadata:
  name: pubsub-prod
spec:
  id: pubsub-prod
  display_name: Pub/Sub Production
  type: googlepubsub
  definition_version: 1
  enabled: true
  config:
    project_id: acme-analytics
    credentials: '{{ .PUBSUB_CREDENTIALS }}'

    event_to_topic_map:
      - from: Order Completed
        to: rudder-orders
      - from: identify
        to: rudder-users
      - from: "*"
        to: rudder-events
    event_to_attribute_map:
      - from: Order Completed
        to: order_id
      - from: Order Completed
        to: currency

    connection_mode:
      web: cloud
      cloud: cloud
    consent_management:
      web:
        - provider: oneTrust
          consents:
            - analytics
```

The above example sends `Order Completed` and `identify` events to their own topics, and every other event to `rudder-events` — see [Topic mapping](#topic-mapping). `credentials` is a JSON key, so its reference is single-quoted — see [Secrets](#secrets).

## Config keys

`config` accepts only the keys listed below. The [shared config key rules](../README.md#config-key-rules) cover unknown keys, defaults, and immutability.

### Connection

#### `project_id` — string, required

ID of the GCP project that holds the topics.

- At most 100 characters, and must not contain line breaks.
- Validation accepts a `{{ path || fallback }}` template, but RudderStack connects with the project ID as written, so use a literal.

#### `credentials` — string, required, secret

GCP service account JSON key RudderStack publishes with. The service account needs the **Pub/Sub Publisher** role on the topics.

- Only checked for presence locally. RudderStack accepts only a service account key: any other kind of Google credential fails at delivery.

Supply it as a `{{ .VAR }}` reference rather than a literal — see [Secrets](#secrets).

### Topic mapping

Each event is matched against the `from` field of both mappings by its event name first, then by its type — `track`, `page`, `identify`, and so on — then by `*`, which matches every event. Matching ignores case.

#### `event_to_topic_map` — array of objects

Maps events to the topics RudderStack publishes them to.

- `from` — an event name, an event type, or `*`.
- `to` — ID of the topic, exactly as it appears in Pub/Sub. Topic IDs are case-sensitive.
- Each field is at most 100 characters, and must not contain line breaks.
- Validation accepts a `{{ path || fallback }}` template in either field, but RudderStack prepares a publisher for each `to` value as written, so keep `to` literal.

Rudder CLI doesn't require this key, but an event that matches no entry isn't delivered. Map `*` to send everything else to one topic:

```yaml
event_to_topic_map:
  - from: Order Completed
    to: rudder-orders
  - from: "*"
    to: rudder-events
```

Quote `"*"` — unquoted, YAML reads it as an alias.

#### `event_to_attribute_map` — array of objects

Adds message attributes copied from the event. Each entry names a field whose value RudderStack attaches as an attribute.

- `from` — an event name, an event type, or `*`, matched as described above. Only one set of entries applies to an event: those for its name, or else its type, or else `*`.
- `to` — path of the field, in dot notation. RudderStack looks it up at the event's root, then in `properties`, `traits`, and `context.traits`. The attribute takes the last segment of the path as its name, so `metadata.order_id` becomes `order_id`.
- Repeat a `from` to attach several attributes. A field the event doesn't carry adds no attribute.
- Attribute values are strings. A number or boolean is converted, and an object or array is sent as JSON.
- Each field is at most 100 characters, and must not contain line breaks. Both accept a `{{ path || fallback }}` template in place of a literal.

```yaml
event_to_attribute_map:
  - from: Order Completed
    to: order_id
  - from: Order Completed
    to: metadata.coupon
```

### Per-source keys

Both keys are objects keyed by the local source type — the tokens listed under [Source types](#source-types). A key naming a source type this destination doesn't support fails validation.

#### `connection_mode` — object

Maps each source type you connect to the mode its events reach Pub/Sub in, using the modes in [Source types](#source-types).

- An entry is required for each source type you connect — see [Connect a source](#connect-a-source).

```yaml
connection_mode:
  web: cloud
  cloud: cloud
```

#### `consent_management` — object

Consent provider configuration per source type. The entry shape, accepted providers, and the rules on `resolution_strategy` and `consents` are shared across all destination types — see [Consent management](../common/README.md).

## Source types

Google Cloud Pub/Sub accepts events from these source types in the mentioned connection modes:

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

Every source type is `cloud` only — events reach Pub/Sub from RudderStack's servers, never in device mode.

The dashboard also accepts `warehouse` for Google Cloud Pub/Sub, but this definition doesn't, so Rudder CLI can't connect a Reverse ETL source to it.

## Connect a source

An event stream connection to this destination is checked against two rules at `validate` time.

**The source's type must be supported.** A source's type resolves to one of the tokens above before the check — a JavaScript source resolves to `web`, and webhook and server-side SDK sources resolve to `cloud`. Every event stream source type is supported here, so only a Reverse ETL connection, whose source resolves to `warehouse`, fails it:

```text
destination 'pubsub-prod' (type 'googlepubsub') does not accept rETL sources: source type 'warehouse' is not among supported source types: android, android_kotlin, ...
```

**The destination config must carry a `connection_mode` entry for that source type.** This lives on the destination spec, not on the connection spec. Without it:

```text
destination 'pubsub-prod' config has no 'connection_mode' entry for source type 'web'
```

Google Cloud Pub/Sub needs no additional config keys to connect a source of any type.

## Secrets

`credentials` is the only secret key. Write it as a `{{ .VAR }}` reference — [Secrets](../README.md#secrets) covers supplying the values and what `import` writes:

```yaml
config:
  credentials: '{{ .PUBSUB_CREDENTIALS }}'
```

`credentials` is JSON, so write its reference in single quotes, as above — see [Secrets](../README.md#secrets). Export the key file as is:

```bash
export RUDDER_PUBSUB_CREDENTIALS="$(cat service-account.json)"
```
