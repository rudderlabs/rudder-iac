# Adjust (`adj`)

Adjust is a mobile attribution destination. RudderStack sends `track` events to Adjust's server-to-server API from its servers, or loads Adjust's SDK in the app in device mode on Android, iOS, Unity, and Flutter.

In an Adjust destination spec:

- `type: adj`
- `definition_version: 1`

> [!NOTE]
> `adj` is an [unverified destination type](../README.md#unverified-destination-types). Rudder CLI registers it only when the `unverifiedDestinations` experimental flag is on.

## Sample configuration

```yaml
version: rudder/v1
kind: destination
metadata:
  name: adjust-prod
spec:
  id: adjust-prod
  display_name: Adjust Production
  type: adj
  definition_version: 1
  enabled: true
  config:
    app_token: t1yurrb968zk
    environment: true

    custom_mappings:
      - from: Order Completed
        to: tf4gm5
      - from: Install Attributed
        to: k2pd7x
    partner_params_keys:
      - from: revenue
        to: price

    enable_install_attribution_tracking:
      android_kotlin: true
      ios_swift: true

    event_filtering:
      blacklist:
        - Application Backgrounded

    connection_mode:
      android_kotlin: device
      ios_swift: device
      cloud: cloud
    consent_management:
      android_kotlin:
        - provider: oneTrust
          consents:
            - marketing
```

The above example loads Adjust's SDK in Kotlin and Swift apps in `device` mode and sends server-side events in `cloud` mode. `environment` and `partner_params_keys` affect only the cloud-mode traffic, and `enable_install_attribution_tracking` only the device-mode apps — see [Install attribution](#install-attribution).

## Config keys

`config` accepts only the keys listed below. The [shared config key rules](../README.md#config-key-rules) cover unknown keys, defaults, and immutability.

> [!NOTE]
> Several Adjust keys affect only one connection mode, or only some SDKs. Each key says which. Rudder CLI accepts every key whatever the mode — a key for the other mode is stored and ignored.
>
> In `cloud` mode Adjust accepts only `track` events. In `device` mode, mobile sources send `identify` and `track`.

### Connection

#### `app_token` — string, required

Token of your Adjust app, which identifies it to Adjust in both modes. The dashboard calls it **APP Token**.

- At most 100 characters, and must not contain line breaks.
- Templates aren't accepted in place of a literal; a template is measured as text against the same limit.

#### `environment` — boolean, default `false`

Send events to Adjust's production environment. When `false` — the default — they go to the sandbox environment. The dashboard calls it **Send to Production Environment on Adjust**.

- Applies to `cloud` mode only. Device mode ignores it: iOS SDK v2 always sends to production, and the other SDKs pick sandbox or production from the log level they run with.

#### `delay` — string

Delay, in seconds, before Adjust's SDK initializes in the app. Written as a string. The dashboard calls it **Delay Time (in seconds)**.

- Applies to `device` mode on `unity` sources, and on `ios` sources running iOS SDK v2. Other SDKs initialize Adjust without a delay.
- Values below `0` or above `10` are clamped to that range, and a value that isn't a number means no delay.
- At most 100 characters, and must not contain line breaks. Not otherwise validated locally.
- Templates aren't accepted in place of a literal; a template is measured as text against the same limit.

```yaml
delay: "5"
```

### Event mapping

#### `custom_mappings` — array of objects

Maps RudderStack event names to Adjust event tokens. Only mapped events reach Adjust: a `track` event without a mapping isn't delivered, in either mode. The dashboard calls it **Map Events to Adjust Event Tokens**.

- `from` — RudderStack event name.
- `to` — Adjust event token. Create the event token in Adjust before you map to it.
- Each is at most 100 characters, or a `{{ path || fallback }}` template.

```yaml
custom_mappings:
  - from: Order Completed
    to: tf4gm5
```

#### `partner_params_keys` — array of objects

Maps `track` event properties to Adjust partner parameters, which Adjust forwards to the partners set up in your Adjust account. Properties without a mapping aren't sent as partner parameters. The dashboard calls it **Rudderstack Parameters to Partner Parameters**.

- `from` — name of the property in the event's `properties`.
- `to` — Adjust partner parameter key. RudderStack sends each value as a string.
- Each is at most 100 characters, or a `{{ path || fallback }}` template.
- Applies to `cloud` mode only. In device mode the SDK ignores this list, and sets only `anonymousId` and `userId` as partner parameters.

```yaml
partner_params_keys:
  - from: revenue
    to: price
```

### Install attribution

#### `enable_install_attribution_tracking` — object

Track an `Install Attributed` event when Adjust attributes the app install, with Adjust's tracker, network, campaign, click label, creative, and ad group in its properties. The SDK tracks it like any other event, so it reaches every destination the source is connected to. The dashboard calls it **Enable Install Attribution**.

- `android`, `android_kotlin`, `ios`, `ios_swift` — booleans.
- Applies to `device` mode on those source types. On `ios`, iOS SDK v2 ignores it.
- To record the event in Adjust too, map `Install Attributed` to an event token in `custom_mappings`.

```yaml
enable_install_attribution_tracking:
  android_kotlin: true
  ios_swift: true
```

### Event filtering

> [!WARNING]
> Client-side event filtering applies only to sources connected in `device` mode — the dashboard marks it as applicable only to device-mode integrations, and the SDK is what applies the filter. Rudder CLI accepts the block in any mode, but it has no effect on cloud-mode events.

#### `event_filtering` — object

Restricts which `track` events the SDK passes to Adjust, by event name.

- `whitelist` — array of event names to allow; every other `track` event is dropped.
- `blacklist` — array of event names to drop; every other `track` event is allowed.
- The two are mutually exclusive, and Rudder CLI enforces it — setting both fails validation.
- Each name is at most 100 characters, or a `{{ path || fallback }}` template.

### Per-source keys

Both keys are objects keyed by the local source type — the tokens listed under [Source types](#source-types). A key naming a source type this destination doesn't support fails validation.

#### `connection_mode` — object

Maps each source type you connect to the mode its events reach Adjust in, using the modes in [Source types](#source-types).

- An entry is required for each source type you connect — see [Connect a source](#connect-a-source).
- A mode the source type doesn't support on this destination fails validation — for example `device` for `react_native`.

```yaml
connection_mode:
  android: device
  ios: device
  cloud: cloud
```

#### `consent_management` — object

Consent provider configuration per source type. The entry shape, accepted providers, and the rules on `resolution_strategy` and `consents` are shared across all destination types — see [Consent management](../common/README.md).

## Source types

Adjust accepts events from these source types in the mentioned connection modes:

| Source type | Connection mode |
| :-----| :-----|
| `android` | `cloud`, `device` |
| `android_kotlin` | `cloud`, `device` |
| `ios` | `cloud`, `device` |
| `ios_swift` | `cloud`, `device` |
| `unity` | `cloud`, `device` |
| `react_native` | `cloud` |
| `flutter` | `cloud`, `device` |
| `cordova` | `cloud` |
| `cloud` | `cloud` |

`android`, `android_kotlin`, `ios`, `ios_swift`, `unity`, and `flutter` offer `device` mode. `react_native` and `cordova` are `cloud` only.

Adjust doesn't accept `web` sources, even in the dashboard. The dashboard also accepts `warehouse` for Adjust, but this definition doesn't, so Rudder CLI can't connect a Reverse ETL source to it.

## Connect a source

An event stream connection to this destination is checked against two rules at `validate` time.

**The source's type must be supported.** A source's type resolves to one of the tokens above before the check — webhook and server-side SDK sources resolve to `cloud`. A JavaScript source resolves to `web`, which Adjust doesn't accept, and reports, for example:

```text
destination 'adjust-prod' (type 'adj') does not support source 'my-web-app':
source type 'web' is not among supported source types: android, android_kotlin, ios, ios_swift, unity, react_native, flutter, cordova, cloud
```

A Reverse ETL connection, whose source resolves to `warehouse`, is refused too:

```text
destination 'adjust-prod' (type 'adj') does not accept rETL sources: source type 'warehouse' is not among supported source types: android, android_kotlin, ...
```

**The destination config must carry a `connection_mode` entry for that source type.** This lives on the destination spec, not on the connection spec. Without it:

```text
destination 'adjust-prod' config has no 'connection_mode' entry for source type 'android'
```

Adjust needs no additional config keys to connect a source of any type, in any mode.

## Secrets

Adjust has no secret keys. Rudder CLI treats none of its keys as secret, and the dashboard masks none of them, so `import` writes every value to YAML in plain text — including `app_token`, which device mode embeds in the app anyway.
