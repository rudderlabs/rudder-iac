# Braze (`braze`)

Streaming destination. Braze receives events from RudderStack's servers (cloud
mode), directly from the Braze SDK in your app (device mode), or both at once
(hybrid mode).

In a destination spec:

- `type: braze`
- `definition_version: 1`

## Modes

The mode a source uses is set per source type in `config.connection_mode`. Braze
is the one destination that supports a third mode:

- `cloud` — events sent from RudderStack's servers
- `device` — events sent by the Braze SDK in your app
- `hybrid` — both, with some calls sent from the device and the rest from the
  server

Five source types accept all three modes; `react_native` and `flutter` accept
cloud and device; the rest are cloud only — see [Source types](#source-types).

Each key below is badged with where it applies. Platforms follow the badge where
support is limited to some of them. A key is accepted by `validate` whatever your
sources use; the badges tell you where the setting takes effect.

**Which mode you choose decides which credentials are required.** See
[Credentials](#credentials) below — this is the part of Braze's config most
likely to fail validation.

## Example

```yaml
version: rudder/v1
kind: destination
metadata:
  name: braze
spec:
  id: braze
  display_name: Braze
  type: braze
  definition_version: 1
  enabled: true
  config:
    data_center: US-03
    rest_api_key: "{{ .BRAZE_REST_API_KEY }}"

    use_platform_specific_api_keys: true
    web_api_key: "{{ .BRAZE_WEB_API_KEY }}"
    android_api_key: "{{ .BRAZE_ANDROID_API_KEY }}"
    ios_api_key: "{{ .BRAZE_IOS_API_KEY }}"

    support_dedup: true
    enable_subscription_group_in_group_call: true
    enable_nested_array_operations: false
    send_purchase_event_with_extra_properties: true
    use_ecommerce_recommended_events: true

    event_filtering:
      whitelist:
        - Order Completed
        - Product Viewed

    # Device mode (web)
    track_anonymous_user:
      web: true
    enable_braze_logging:
      web: false
    enable_push_notification:
      web: true
    allow_user_supplied_javascript:
      web: false

    connection_mode:
      web: hybrid
      android: device
      ios: device
      cloud: cloud
    consent_management:
      web:
        - provider: oneTrust
          consents:
            - marketing
```

## Credentials

Which keys are required depends on the modes declared in `connection_mode`, and
on whether you use one app key or platform-specific ones.

| Situation | Required |
| --- | --- |
| Any source type in `cloud` or `hybrid` | `rest_api_key` |
| Any source type in `device` or `hybrid`, with `use_platform_specific_api_keys` unset or `false` | `app_key` |
| Any of `web` in `device` or `hybrid`, with `use_platform_specific_api_keys` `true` | `web_api_key` |
| Any of `android`, `android_kotlin`, `react_native`, `flutter` in `device` or `hybrid`, with `use_platform_specific_api_keys` `true` | `android_api_key` |
| Any of `ios`, `ios_swift`, `react_native`, `flutter` in `device` or `hybrid`, with `use_platform_specific_api_keys` `true` | `ios_api_key` |

`react_native` and `flutter` appear in both the Android and iOS rows: a source of
either type running in device mode needs both platform keys.

Separately, connecting any source to this destination requires `rest_api_key` to
be present — see [Connecting a source](#connecting-a-source).

## Config keys

`config` accepts only the keys documented here — anything else fails validation
with `unknown config field "<key>"`.

Keys that declare a default are filled in before the spec enters the resource
graph, matching what the backend stores, so omitting one is equivalent to
writing its default and does not produce a permanent diff.

A `*` after a key name marks a description written without a Terraform provider
source to draw on. Those need a closer review pass; the markers come out once the
wording is confirmed.

### Connection

#### `data_center` — string, required
`cloud` `device`

Your Braze data center. One of `US-01` through `US-08`, `EU-01` through `EU-03`,
or `AU-01`.

#### `rest_api_key` — string, secret
`cloud`

Your Braze REST API key. **Required when any source type runs in `cloud` or
`hybrid` mode.** At most 100 characters.

Supply it as a `{{ .VAR }}` reference — see [Secrets](#secrets).

#### `app_key` — string
`cloud` `device`

Your Braze app key. **Required when any source type runs in `device` or `hybrid`
mode and `use_platform_specific_api_keys` is unset or `false`.** At most 100
characters.

#### `use_platform_specific_api_keys` \* — boolean
`device`

Use a separate Braze app key per platform instead of the single `app_key`. When
`true`, the three keys below replace it.

#### `web_api_key` \* — string
`device` · web

Braze app key for web. **Required when `use_platform_specific_api_keys` is `true`
and `web` runs in `device` or `hybrid` mode.** At most 100 characters.

#### `android_api_key` \* — string
`device` · android, android_kotlin, react_native, flutter

Braze app key for Android. **Required when `use_platform_specific_api_keys` is
`true` and any Android-family source type runs in `device` or `hybrid` mode.** At
most 100 characters.

#### `ios_api_key` \* — string
`device` · ios, ios_swift, react_native, flutter

Braze app key for iOS. **Required when `use_platform_specific_api_keys` is `true`
and any iOS-family source type runs in `device` or `hybrid` mode.** At most 100
characters.

### Event delivery

#### `support_dedup` — boolean, default `false`
`cloud` `device` · android

Deduplicate traits on `identify` and `track` calls, sending only values that
changed since the previous call.

#### `enable_subscription_group_in_group_call` — boolean, default `false`
`cloud`

Enable subscription groups in `group` calls.

#### `enable_nested_array_operations` — boolean, default `false`
`cloud`

Enable Braze's custom attribute operations for nested arrays.

#### `send_purchase_event_with_extra_properties` — boolean, default `false`
`cloud`

Send purchase events with their custom properties attached.

#### `use_ecommerce_recommended_events` \* — boolean, default `true`
`cloud` `device`

Map ecommerce events to Braze's recommended event names rather than sending them
under their original names.

#### `event_filtering` — object
`cloud` `device`

Filter which events are sent to Braze. Exactly one of the two lists may be set —
declaring both fails validation.

- `whitelist` — event names to allowlist
- `blacklist` — event names to denylist

Each entry is at most 100 characters. Omit the block to send every event.

## Device mode only

The keys below configure the Braze Web SDK and apply only to a web source in
device or hybrid mode. Each is an object with a single boolean `web` key.

`validate` accepts these keys whatever sources the project connects, so you can
declare them ahead of connecting a web source.

#### `track_anonymous_user` — object
`device` · web

Track events for users who have not been identified.

#### `enable_braze_logging` — object
`device` · web

Surface Braze SDK logs in the browser console.

#### `enable_push_notification` — object
`device` · web

Enable web push notifications. Requires a service worker set up in your
application.

#### `allow_user_supplied_javascript` — object
`device` · web

Enable HTML in-app messages, which may contain user-supplied JavaScript.

## Source types

| Source type | Modes |
| --- | --- |
| `web` | cloud, device, hybrid |
| `android` | cloud, device, hybrid |
| `android_kotlin` | cloud, device, hybrid |
| `ios` | cloud, device, hybrid |
| `ios_swift` | cloud, device, hybrid |
| `react_native` | cloud, device |
| `flutter` | cloud, device |
| `unity` | cloud |
| `cordova` | cloud |
| `cloud` | cloud |

## Per-source keys

Both keys below are objects keyed by the local source type. A key naming a
source type this destination does not support fails validation.

#### `connection_mode` — object

Selects the mode per source type. Values are constrained to the modes that source
type supports, so `hybrid` is rejected for `react_native`, `flutter`, `unity`,
`cordova` and `cloud`:

```yaml
connection_mode:
  web: hybrid
  android: device
  cloud: cloud
```

#### `consent_management` — object

Specify consent configuration data for multiple providers, per source type. The
entry shape, accepted providers, and the rules on `resolution_strategy` and
`consents` are shared across all destinations and documented in
[../common/README.md](../common/README.md).

## Connecting a source

An event stream connection to this destination is checked against three rules at
`validate` time.

**The source's type must be supported.** A source's type is mapped to one of the
tokens above first — a JavaScript source resolves to `web`, and webhook and
server-side SDK sources resolve to `cloud`. An unsupported type reports:

```
destination 'braze' (type 'braze') does not support source 'my-source':
source type 'amp' is not among supported source types: android, android_kotlin, ...
```

**The config must carry a `connection_mode` entry for that source type.** This
lives on the destination spec, not the connection spec. Without it:

```
destination 'braze' config has no 'connection_mode' entry for source type 'web'
```

**The config must carry `rest_api_key`.** Unlike most destinations, Braze
declares a config key that a source needs in order to connect at all — every
source type requires `rest_api_key` in `cloud` mode, and the five hybrid-capable
types require it in `hybrid` mode too:

```
destination 'braze' config is missing fields required to connect a 'web' source:
rest_api_key
```

## Secrets

`rest_api_key` is the only key registered as a secret. Write it as a `{{ .VAR }}`
reference and supply the value at apply time:

```yaml
rest_api_key: "{{ .BRAZE_REST_API_KEY }}"
```

```sh
export RUDDER_BRAZE_REST_API_KEY=...
rudder-cli apply

# or
rudder-cli apply --var-file secrets.vars.yaml
```

`rudder-cli import` writes `rest_api_key` back as a `{{ .VAR }}` placeholder
rather than its value, since the API does not return secrets. Fill the
placeholder in before the first apply.

The app keys — `app_key`, `web_api_key`, `android_api_key` and `ios_api_key` —
are not registered as secrets, because they are embedded in client-side
applications and returned by the API. They are shown above as `{{ .VAR }}`
references for consistency, but that is a convention rather than a requirement.
