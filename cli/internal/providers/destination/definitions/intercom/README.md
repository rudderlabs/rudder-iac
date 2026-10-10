# Intercom (`intercom`)

Intercom is a customer messaging destination. RudderStack sends `identify`, `track`, and `group` events to Intercom's REST API from its servers, or loads Intercom's own SDKs in device mode — the Messenger on web, and the mobile SDKs on Android and iOS.

In an Intercom destination spec:

- `type: intercom`
- `definition_version: 1`

> [!NOTE]
> `intercom` is an [unverified destination type](../README.md#unverified-destination-types). Rudder CLI registers it only when the `unverifiedDestinations` experimental flag is on.

## Sample configuration

```yaml
version: rudder/v1
kind: destination
metadata:
  name: intercom-prod
spec:
  id: intercom-prod
  display_name: Intercom Production
  type: intercom
  definition_version: 1
  enabled: true
  config:
    app_id: fll5vd90
    api_key: "{{ .INTERCOM_ACCESS_TOKEN }}"
    api_version: v2
    api_server: standard
    send_anonymous_id: true

    mobile_api_key_android: android_sdk-3f8a1c2d4e5b6a7980c1d2e3f4a5b6c7d8e9f0a1
    mobile_api_key_ios: ios_sdk-9b8c7d6e5f4a3b2c1d0e9f8a7b6c5d4e3f2a1b0c

    event_filtering:
      blacklist:
        - Credit Card Added

    connection_mode:
      web: device
      android: device
      ios: device
      cloud: cloud
    consent_management:
      web:
        - provider: oneTrust
          consents:
            - analytics
```

The above example loads Intercom's SDKs on web, Android, and iOS in `device` mode, which use `app_id` and the mobile API keys, and sends server-side events in `cloud` mode with `api_key` — see [Credential requirements](#credential-requirements).

## Config keys

`config` accepts only the keys listed below. The [shared config key rules](../README.md#config-key-rules) cover unknown keys, defaults, and immutability.

> [!NOTE]
> Most Intercom keys affect only one connection mode, or one platform's SDK. Each key says which. Rudder CLI accepts every key whatever the mode — a key for the other mode is stored and ignored.
>
> In `cloud` mode Intercom accepts `identify`, `track`, and `group` events. In `device` mode, web sources send `identify`, `track`, and `page`, and Android and iOS sources send `identify` and `track`.

### Credential requirements

Device mode loads Intercom with `app_id`; cloud mode calls Intercom's API with `api_key`. Rudder CLI checks for them twice:

- **On the destination.** `app_id` is required when `connection_mode` is set and every entry in it maps `web`, `android`, or `ios` to `device`. `api_key` is required when every entry maps `web`, `android`, `ios`, `unity`, `react_native`, `flutter`, or `cordova` to `cloud`. A `connection_mode` that mixes modes, or that names `cloud`, `android_kotlin`, or `ios_swift`, requires neither here, and neither does a spec without `connection_mode`.
- **On each connection.** A `web`, `android`, or `ios` source connecting in `device` mode needs `app_id`. A source connecting in `cloud` mode needs `api_key` — except `cloud`, `android_kotlin`, and `ios_swift`, which aren't checked. See [Connect a source](#connect-a-source).

A spec that misses either key where the destination check requires it fails validation.

In practice: set `app_id` if anything connects in `device` mode, and `api_key` if anything connects in `cloud` mode. Cloud-mode delivery fails at Intercom without `api_key`, whatever the source type — including the three source types Rudder CLI doesn't check.

Android and iOS device mode also need their platform's mobile API key, which Rudder CLI never requires — see [Mobile device mode](#mobile-device-mode).

### Connection

#### `app_id` — string, required

ID of your Intercom workspace, which Intercom's web Messenger and mobile SDKs load with. The dashboard calls it **App Id**.

- Required when a `web`, `android`, or `ios` source connects in `device` mode — see [Credential requirements](#credential-requirements).
- Applies to `device` mode only.
- 1 to 100 characters, and must not contain line breaks.
- A `{{ path || fallback }}` template is accepted in place of a literal.

#### `api_key` — string, required, secret

Intercom access token RudderStack calls Intercom's REST API with. The dashboard calls it **Access Token**.

- Required when a source connects in `cloud` mode — see [Credential requirements](#credential-requirements).
- Applies to `cloud` mode only.
- 1 to 100 characters, and must not contain line breaks.
- A `{{ path || fallback }}` template is accepted in place of a literal.

Supply it as a `{{ .VAR }}` reference rather than a literal — see [Secrets](#secrets).

### Cloud mode

These keys configure how RudderStack calls Intercom's API, so they apply only to events sent in `cloud` mode. The dashboard shows them only then.

#### `api_version` — string, default `v2`

Intercom REST API version RudderStack sends to. The dashboard calls it **Intercom REST API Version**.

- `v2` — the latest API version (2.10).
- `v1` — API version 1.4, kept for backward compatibility.

#### `api_server` — string, default `standard`

Region of your Intercom workspace. The dashboard calls it **API Server**.

- `standard` (US), `eu`, or `au`.
- Applies when `api_version` is `v2`. With `v1`, RudderStack always calls the US endpoint.

```yaml
api_version: v2
api_server: eu
```

#### `send_anonymous_id` — boolean, default `false`

Send the event's `anonymousId` as the Intercom user ID when `userId` is absent. The dashboard calls it **Send AnonymousId as Secondary UserId**.

- Without it, an `identify` or `track` event carrying neither a `userId` nor an email fails, because Intercom needs one of them.

#### `update_last_request_at` — boolean, default `true`

Set the user's last seen time in Intercom to the time of each `identify` call. The dashboard calls it **Enable this to update the last seen to the current time**.

- Applies when `api_version` is `v1`. It has no effect with `v2`.

```yaml
api_version: v1
update_last_request_at: false
```

### Mobile device mode

These keys configure Intercom's mobile SDKs. Each is a plain string, not keyed by source type, though it applies to one platform only.

> [!WARNING]
> Rudder CLI doesn't require these keys, even when `android` or `ios` connects in `device` mode. Without the platform's key, the SDK doesn't initialize Intercom, so no device-mode events from that platform reach it.

#### `mobile_api_key_android` — string

Intercom API key for Android, from the Android installation settings in Intercom. The Android SDK initializes Intercom with it and `app_id`. The dashboard calls it **Android API Key**.

- Applies to `device` mode on `android` sources only.
- At most 100 characters, and must not contain line breaks.
- A `{{ path || fallback }}` template is accepted in place of a literal.

#### `mobile_api_key_ios` — string

Intercom API key for iOS, used on the same terms as `mobile_api_key_android`. The dashboard calls it **iOS API Key**.

- Applies to `device` mode on `ios` sources only.
- At most 100 characters, and must not contain line breaks.
- A `{{ path || fallback }}` template is accepted in place of a literal.

### Event filtering

> [!WARNING]
> Client-side event filtering applies only to sources connected in `device` mode — `web`, `android`, or `ios`. The dashboard shows these controls only then, and the SDK is what applies the filter. Rudder CLI accepts the block in any mode, but it has no effect on cloud-mode events.

#### `event_filtering` — object

Restricts which `track` events the SDK passes to Intercom, by event name.

- `whitelist` — array of event names to allow; every other `track` event is dropped.
- `blacklist` — array of event names to drop; every other `track` event is allowed.
- The two are mutually exclusive, and Rudder CLI enforces it — setting both fails validation.
- Each name is at most 100 characters, or a `{{ path || fallback }}` template.

### Per-source keys

Both keys are objects keyed by the local source type — the tokens listed under [Source types](#source-types). A key naming a source type this destination doesn't support fails validation.

#### `connection_mode` — object

Maps each source type you connect to the mode its events reach Intercom in, using the modes in [Source types](#source-types).

- An entry is required for each source type you connect — see [Connect a source](#connect-a-source).
- A mode the source type doesn't support on this destination fails validation — for example `device` for `android_kotlin`.
- The entries also decide which credentials the destination needs — see [Credential requirements](#credential-requirements).

```yaml
connection_mode:
  web: device
  ios: device
  cloud: cloud
```

#### `consent_management` — object

Consent provider configuration per source type. The entry shape, accepted providers, and the rules on `resolution_strategy` and `consents` are shared across all destination types — see [Consent management](../common/README.md).

## Source types

Intercom accepts events from these source types in the mentioned connection modes:

| Source type | Connection mode |
| :-----| :-----|
| `android` | `cloud`, `device` |
| `android_kotlin` | `cloud` |
| `ios` | `cloud`, `device` |
| `ios_swift` | `cloud` |
| `web` | `cloud`, `device` |
| `unity` | `cloud` |
| `cloud` | `cloud` |
| `react_native` | `cloud` |
| `flutter` | `cloud` |
| `cordova` | `cloud` |

`web`, `android`, and `ios` offer `device` mode. The Kotlin and Swift SDKs — `android_kotlin` and `ios_swift` — are `cloud` only.

The dashboard also accepts `warehouse` for Intercom, but this definition doesn't, so Rudder CLI can't connect a Reverse ETL source to it.

## Connect a source

An event stream connection to this destination is checked against three rules at `validate` time.

**The source's type must be supported.** A source's type resolves to one of the tokens above before the check — a JavaScript source resolves to `web`, and webhook and server-side SDK sources resolve to `cloud`. Every event stream source type is supported here, so only a Reverse ETL connection, whose source resolves to `warehouse`, fails it:

```text
destination 'intercom-prod' (type 'intercom') does not accept rETL sources: source type 'warehouse' is not among supported source types: android, android_kotlin, ...
```

**The destination config must carry a `connection_mode` entry for that source type.** This lives on the destination spec, not on the connection spec. Without it:

```text
destination 'intercom-prod' config has no 'connection_mode' entry for source type 'web'
```

**The config must carry the credential for the connection's mode.** A `web`, `android`, or `ios` source connecting in `device` mode needs `app_id`:

```text
destination 'intercom-prod' config is missing fields required to connect a 'android' source: app_id
```

A source connecting in `cloud` mode needs `api_key`, except `cloud`, `android_kotlin`, and `ios_swift`, which connect without it:

```text
destination 'intercom-prod' config is missing fields required to connect a 'web' source: api_key
```

## Secrets

`api_key` is the only secret key. Write it as a `{{ .VAR }}` reference — [Secrets](../README.md#secrets) covers supplying the values and what `import` writes:

```yaml
config:
  api_key: "{{ .INTERCOM_ACCESS_TOKEN }}"
```

`app_id` and the mobile API keys aren't secrets — device mode embeds them in the page or app, and `import` writes them as they are.
