# Marketo (`marketo`)

Marketo is a marketing automation destination. RudderStack creates and updates Marketo leads from `identify` events, and records `track` events as custom activities on those leads, from its servers.

In a Marketo destination spec:

- `type: marketo`
- `definition_version: 1`

> [!NOTE]
> `marketo` is an [unverified destination type](../README.md#unverified-destination-types). Rudder CLI registers it only when the `unverifiedDestinations` experimental flag is on.

## Sample configuration

```yaml
version: rudder/v1
kind: destination
metadata:
  name: marketo-prod
spec:
  id: marketo-prod
  display_name: Marketo Production
  type: marketo
  definition_version: 1
  enabled: true
  config:
    account_id: 123-ABC-456
    client_id: 53b1934e-8f2a-4c1d-9e3b-92612c41515f
    client_secret: "{{ .MARKETO_CLIENT_SECRET }}"

    track_anonymous_events: false
    create_if_not_exist: true

    rudder_events_mapping:
      - event: Product Clicked
        marketo_activity_id: "100001"
        marketo_primarykey: product_id
    custom_activity_property_map:
      - from: product_name
        to: productName
    lead_trait_mapping:
      - from: leadScore
        to: customLeadScore

    connection_mode:
      web: cloud
      cloud: cloud
    consent_management:
      web:
        - provider: oneTrust
          consents:
            - marketing
```

The above example records `Product Clicked` events as a custom activity and fails every other `track` event — Marketo receives only the events [`rudder_events_mapping`](#rudder_events_mapping--array-of-objects) names. Note that `marketo_activity_id` is a quoted string.

## Config keys

`config` accepts only the keys listed below. The [shared config key rules](../README.md#config-key-rules) cover unknown keys, defaults, and immutability.

> [!NOTE]
> Marketo accepts only `identify` and `track` events. RudderStack doesn't deliver any other event type to it.

### Connection

#### `account_id` — string, required

Munchkin account ID of your Marketo instance, such as `123-ABC-456`. RudderStack calls Marketo's REST API on `<account_id>.mktorest.com`. The dashboard calls it **Munchkin Account Id**.

- At most 100 characters, and must not contain line breaks.
- A `{{ path || fallback }}` template is accepted in place of a literal.
- A spec that omits this key fails validation with `'account_id' is required`.

#### `client_id` — string, required

Client ID of the Marketo REST API service RudderStack authenticates as — a custom LaunchPoint service tied to an API-only user.

- At most 100 characters, and must not contain line breaks.
- A `{{ path || fallback }}` template is accepted in place of a literal.

#### `client_secret` — string, required, secret

Client secret of the same REST API service.

- At most 100 characters, and must not contain line breaks.
- A `{{ path || fallback }}` template is accepted in place of a literal.

Supply it as a `{{ .VAR }}` reference rather than a literal — see [Secrets](#secrets).

### Lead matching

RudderStack finds the lead an event belongs to by its `marketoLeadId` external ID when the event carries one. Otherwise it looks the lead up by the event's email or, when there's no email, by its `userId` — or its `anonymousId` when there's no `userId` — stored in Marketo lead fields with those exact API names.

> [!NOTE]
> Create `userId` and `anonymousId` lead fields in Marketo before you send events. RudderStack looks leads up, and creates them, through those fields, so events fail without them.

#### `track_anonymous_events` — boolean, default `false`

Send `track` events that have no `userId`. RudderStack then matches the lead by email, or by `anonymousId`.

- When `false`, a `track` event without a `userId` fails.
- The dashboard's note on this setting reads as though turning it on requires a `userId` on every `track` event. It's the other way round: turning it on is what lets events without one through.

#### `create_if_not_exist` — boolean, default `true`

Create a lead when no existing lead matches the event. When `false`, an event without a matching lead fails.

### Mappings

None of the mapping fields is validated locally, so Rudder CLI accepts a row with a field missing — and every field is a string.

#### `rudder_events_mapping` — array of objects

Maps `track` event names to Marketo custom activities. A `track` event without a row here fails, so Marketo receives only the events you map.

- `event` — RudderStack event name.
- `marketo_activity_id` — ID of the custom activity type, from Marketo's admin settings. A digit string: quote it, since an unquoted number fails validation.
- `marketo_primarykey` — event property whose value fills the activity's primary field. An event without a value for it fails.

```yaml
rudder_events_mapping:
  - event: Product Clicked
    marketo_activity_id: "100001"
    marketo_primarykey: product_id
```

#### `custom_activity_property_map` — array of objects

Maps `track` event properties onto fields of the event's custom activity. The property named by `marketo_primarykey` fills the primary field and isn't sent through this map.

- `from` — event property name.
- `to` — API name of the custom activity field.

```yaml
custom_activity_property_map:
  - from: product_name
    to: productName
```

#### `lead_trait_mapping` — array of objects

Maps `identify` traits onto Marketo lead fields. RudderStack already maps standard traits, such as `email`, `firstName`, and `phone`, on its own. Use this for custom fields, or to send a trait to a different field.

- `from` — trait name.
- `to` — API name of the lead field.

```yaml
lead_trait_mapping:
  - from: leadScore
    to: customLeadScore
```

### Per-source keys

Both keys are objects keyed by the local source type — the tokens listed under [Source types](#source-types). A key naming a source type this destination doesn't support fails validation.

#### `connection_mode` — object

Maps each source type you connect to the mode its events reach Marketo in, using the modes in [Source types](#source-types).

- An entry is required for each source type you connect — see [Connect a source](#connect-a-source).

```yaml
connection_mode:
  web: cloud
  cloud: cloud
```

#### `consent_management` — object

Consent provider configuration per source type. The entry shape, accepted providers, and the rules on `resolution_strategy` and `consents` are shared across all destination types — see [Consent management](../common/README.md).

## Source types

Marketo accepts events from these source types in the mentioned connection modes:

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

Every source type is `cloud` only — events reach Marketo from RudderStack's servers, never in device mode.

The dashboard also accepts `warehouse` for Marketo, but this definition doesn't, so Rudder CLI can't connect a Reverse ETL source to it.

## Connect a source

An event stream connection to this destination is checked against two rules at `validate` time.

**The source's type must be supported.** A source's type resolves to one of the tokens above before the check — a JavaScript source resolves to `web`, and webhook and server-side SDK sources resolve to `cloud`. Every event stream source type is supported here, so only a Reverse ETL connection, whose source resolves to `warehouse`, fails it:

```text
destination 'marketo-prod' (type 'marketo') does not accept rETL sources: source type 'warehouse' is not among supported source types: android, android_kotlin, ...
```

**The destination config must carry a `connection_mode` entry for that source type.** This lives on the destination spec, not on the connection spec. Without it:

```text
destination 'marketo-prod' config has no 'connection_mode' entry for source type 'web'
```

Marketo needs no additional config keys to connect a source of any type.

## Secrets

`client_secret` is the only secret key. Write it as a `{{ .VAR }}` reference — [Secrets](../README.md#secrets) covers supplying the values and what `import` writes:

```yaml
config:
  client_secret: "{{ .MARKETO_CLIENT_SECRET }}"
```

`client_id` isn't a secret — it identifies the API service, but doesn't authenticate without the secret.
