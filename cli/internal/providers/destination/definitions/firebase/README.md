# Firebase (`firebase`)

Firebase is a mobile analytics destination that runs only in device mode. RudderStack's mobile SDKs load the Firebase SDK in the app and log `identify`, `track`, and `screen` calls to Google Analytics for Firebase from there, mapping ecommerce events to Firebase's standard events where they match.

In a Firebase destination spec:

- `type: firebase`
- `definition_version: 1`

> [!NOTE]
> `firebase` is an [unverified destination type](../README.md#unverified-destination-types). Rudder CLI registers it only when the `unverifiedDestinations` experimental flag is on.

## Sample configuration

```yaml
version: rudder/v1
kind: destination
metadata:
  name: firebase-prod
spec:
  id: firebase-prod
  display_name: Firebase Production
  type: firebase
  definition_version: 1
  enabled: true
  config:
    event_filtering:
      blacklist:
        - Application Backgrounded
        - Debug Event

    connection_mode:
      android_kotlin: device
      ios_swift: device
    consent_management:
      android_kotlin:
        - provider: oneTrust
          consents:
            - analytics
```

The above example connects Kotlin and Swift apps. The Firebase project itself isn't configured here — the app bundles it, in the `google-services.json` or `GoogleService-Info.plist` file Firebase generates — so [`event_filtering`](#event-filtering) is the only Firebase-specific key.

## Config keys

`config` accepts only the keys listed below. The [shared config key rules](../README.md#config-key-rules) cover unknown keys, defaults, and immutability.

> [!NOTE]
> Firebase runs only in device mode, and receives `identify`, `track`, and `screen` calls from the mobile SDKs. Every key on this page configures what the SDK sends to Firebase in the app.

### Event filtering

Every source connects to Firebase in `device` mode, where the SDK applies the filter, so the filter applies to `track` events from every source.

#### `event_filtering` — object

Restricts which `track` events the SDK passes to Firebase, by event name.

- `whitelist` — array of event names to allow; every other `track` event is dropped.
- `blacklist` — array of event names to drop; every other `track` event is allowed.
- The two are mutually exclusive, and Rudder CLI enforces it — setting both fails validation.
- Each name is at most 100 characters, or a `{{ path || fallback }}` template.
- Omit the block entirely to filter nothing.

```yaml
event_filtering:
  whitelist:
    - Order Completed
    - Product Added
```

### Per-source keys

Both keys are objects keyed by the local source type — the tokens listed under [Source types](#source-types). A key naming a source type this destination doesn't support fails validation.

#### `connection_mode` — object

Maps each source type you connect to the mode its events reach Firebase in. Firebase accepts only `device`.

- An entry is required for each source type you connect — see [Connect a source](#connect-a-source).
- Any other mode fails validation — for example `cloud` for `android`.

```yaml
connection_mode:
  android: device
  ios: device
  react_native: device
```

#### `consent_management` — object

Consent provider configuration per source type. The entry shape, accepted providers, and the rules on `resolution_strategy` and `consents` are shared across all destination types — see [Consent management](../common/README.md).

## Source types

Firebase accepts events from these source types in the mentioned connection modes:

| Source type | Connection mode |
| :-----| :-----|
| `android` | `device` |
| `android_kotlin` | `device` |
| `ios` | `device` |
| `ios_swift` | `device` |
| `unity` | `device` |
| `react_native` | `device` |
| `flutter` | `device` |

Firebase accepts only mobile SDK sources, and only in `device` mode — the SDK loads Firebase in the app, and no events pass through RudderStack's servers. It doesn't accept `web`, `cordova`, or `cloud` sources.

Firebase doesn't accept `warehouse`, even in the dashboard.

## Connect a source

An event stream connection to this destination is checked against two rules at `validate` time.

**The source's type must be supported.** A source's type resolves to one of the tokens above before the check. A JavaScript source resolves to `web`, a Cordova source to `cordova`, and webhook and server-side SDK sources to `cloud` — none of which Firebase accepts — and reports, for example:

```text
destination 'firebase-prod' (type 'firebase') does not support source 'my-web-app':
source type 'web' is not among supported source types: android, android_kotlin, ios, ios_swift, unity, react_native, flutter
```

A Reverse ETL connection, whose source resolves to `warehouse`, is refused too:

```text
destination 'firebase-prod' (type 'firebase') does not accept rETL sources: source type 'warehouse' is not among supported source types: android, android_kotlin, ...
```

**The destination config must carry a `connection_mode` entry for that source type.** This lives on the destination spec, not on the connection spec. Without it:

```text
destination 'firebase-prod' config has no 'connection_mode' entry for source type 'android'
```

Firebase needs no additional config keys to connect a source of any type.

## Secrets

Firebase has no secret keys, and no credentials in its config at all. The app authenticates to Firebase with the configuration file it bundles.
