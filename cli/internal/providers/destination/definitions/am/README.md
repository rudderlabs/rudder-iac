# Amplitude (`am`)

Streaming destination. Amplitude receives events either from RudderStack's
servers (cloud mode) or directly from the Amplitude SDK embedded in your app
(device mode).

In a destination spec:

- `type: am`
- `definition_version: 1` — the only registered version

## Modes

Which mode a source uses is set per source type in `config.connection_mode`.
Five of the ten supported source types accept `device`; the rest are cloud only —
see [Source types](#source-types).

The mode decides which config keys have any effect, so each key below is
labelled:

- **cloud mode** — read by the RudderStack transformer when it builds the
  Amplitude payload
- **device mode** — read by the Amplitude SDK integration on the listed
  platforms
- **cloud and device mode** — read by both

Labels are derived from the integration sources, not from the key name. A key
labelled for one mode is still accepted by `validate` in any configuration; the
label tells you whether setting it changes anything.

## Example

```yaml
version: rudder/v1
kind: destination
metadata:
  name: amplitude
spec:
  id: amplitude
  display_name: Amplitude
  type: am
  definition_version: 1
  enabled: true
  config:
    api_key: "{{ .AMPLITUDE_API_KEY }}"
    api_secret: "{{ .AMPLITUDE_API_SECRET }}"
    residency_server: standard

    use_user_defined_page_event_name: true
    user_provided_page_event_string: "Viewed a Page"
    use_user_defined_screen_event_name: false
    user_provided_screen_event_string: "Viewed a Screen"

    group_type_trait: company
    group_value_trait: company_id
    traits_to_increment:
      - login_count
    traits_to_set_once:
      - signup_date
    traits_to_append:
      - viewed_categories
    traits_to_prepend:
      - recent_searches
    enable_enhanced_user_operations: true

    track_products_once: false
    track_revenue_per_product: true

    version_name: "1.4.0"
    map_device_brand: true
    event_filtering:
      whitelist:
        - Order Completed
        - Product Viewed

    track_all_pages: true
    track_categorized_pages: true
    track_named_pages: true

    # Device mode
    sdk_version:
      web: 2
    proxy_server_url:
      web: "https://amplitude-proxy.example.com"
    prefer_anonymous_id_for_device_id:
      web: true
    track_session_events:
      web: true
      android: true
      ios: true
    event_upload_period_millis:
      web: "1000"
      android: "30000"
    event_upload_threshold:
      web: "30"
      ios: "30"
    auto_capture:
      page_views:
        web: true
      web_vitals:
        web: true
    enable_location_listening:
      android: true
    use_advertising_id_for_device_id:
      android: false
    use_idfa_as_device_id:
      ios: false

    use_native_sdk:
      web: true
      android: true
    connection_mode:
      web: device
      android: device
      cloud: cloud
    consent_management:
      web:
        - provider: oneTrust
          consents:
            - analytics
```

## Config keys

`config` accepts only the keys documented here — anything else fails validation
with `unknown config field "<key>"`.

Keys that declare a default are filled in before the spec enters the resource
graph, matching what the backend stores, so omitting one is equivalent to
writing its default and does not produce a permanent diff.

### Connection

#### `api_key` — string, required
<sub>cloud and device mode</sub>

Amplitude project API key. At most 100 characters.

#### `api_secret` — string, secret
<sub>cloud mode</sub>

Amplitude project secret, used for the server-side delete and identify APIs. The
SDKs never receive it. At most 100 characters.

Supply it as a `{{ .VAR }}` reference — see [Secrets](#secrets).

#### `residency_server` — string, required
<sub>cloud and device mode</sub> — device: web, android

Amplitude data residency region. One of `standard` or `EU`. Determines which
Amplitude endpoint receives the events.

### Page and screen tracking

#### `use_user_defined_page_event_name` — boolean, default `false`
<sub>cloud mode</sub>

Send `page` calls under a fixed custom event name rather than the derived one.

#### `user_provided_page_event_string` — string
<sub>cloud mode</sub>

The event name used when the flag above is set. At most 200 characters.

#### `use_user_defined_screen_event_name` — boolean, default `false`
<sub>cloud mode</sub>

The `screen` equivalent of `use_user_defined_page_event_name`.

#### `user_provided_screen_event_string` — string
<sub>cloud mode</sub>

The event name used when the flag above is set. At most 200 characters.

#### `track_all_pages` — boolean, default `false`
<sub>device mode</sub> — web, android, ios

Send every `page` call to Amplitude as an event. The transformer does not read
this, so it has no effect in cloud mode.

#### `track_categorized_pages` — boolean, default `true`
<sub>device mode</sub> — web, android, ios

Send `page` calls that carry a category.

#### `track_named_pages` — boolean, default `true`
<sub>device mode</sub> — web, android, ios

Send `page` calls that carry a name.

### Identify and traits

#### `group_type_trait` — string
<sub>cloud and device mode</sub> — device: web, ios

Trait whose value names the Amplitude group *type*. Must be set together with
`group_value_trait`; Amplitude groups are ignored unless both are present. At
most 100 characters.

#### `group_value_trait` — string
<sub>cloud and device mode</sub> — device: web, ios

Trait whose value is the Amplitude group *value*. At most 100 characters.

#### `traits_to_increment` — string array
<sub>cloud and device mode</sub> — device: web, android, ios

Traits Amplitude should increment by the incoming value rather than overwrite.
Each entry at most 100 characters.

#### `traits_to_set_once` — string array
<sub>cloud and device mode</sub> — device: web, android, ios

Traits written only if not already set on the user. Each entry at most 100
characters.

#### `traits_to_append` — string array
<sub>cloud and device mode</sub> — device: android, ios

Traits appended to an existing list value. The web SDK reads only increment and
set-once, so this has no effect for a web source in device mode. Each entry at
most 100 characters.

#### `traits_to_prepend` — string array
<sub>cloud and device mode</sub> — device: android, ios

Traits prepended to an existing list value. Same web limitation as
`traits_to_append`. Each entry at most 100 characters.

#### `enable_enhanced_user_operations` — boolean, default `false`
<sub>cloud mode</sub>

Enable Amplitude's enhanced user property operations, which lets the increment,
set-once, append and prepend lists above apply to group identify calls as well.

### Ecommerce and revenue

#### `track_products_once` — boolean, default `false`
<sub>cloud and device mode</sub> — device: web, android, ios

Send one event carrying all products rather than one event per product.

#### `track_revenue_per_product` — boolean, default `false`
<sub>cloud and device mode</sub> — device: web, android, ios

Record revenue against each product individually rather than against the order
as a whole.

### Other

#### `version_name` — string
<sub>cloud and device mode</sub> — device: web, ios

App version reported to Amplitude with every event. At most 100 characters.

#### `map_device_brand` — boolean, default `false`
<sub>cloud and device mode</sub> — device: ios

Map the incoming device manufacturer to Amplitude's `device_brand` field.

#### `event_filtering` — object
<sub>cloud and device mode</sub>

Restrict which `track` events reach Amplitude. Exactly one of the two lists may
be set — declaring both fails validation.

- `whitelist` — string array; only these event names are forwarded
- `blacklist` — string array; these event names are dropped

Each entry is at most 100 characters. Omit the block entirely to forward every
event.

## Device mode only

The keys below are read only by the Amplitude SDK integrations, and each is
declared for specific source types. Each is an object keyed by platform, using
the local (snake_case) source type as the key.

The per-platform declarations mirror the destination's upstream per-source key
lists. They are **metadata, not a constraint**: `validate` accepts these keys
whatever sources the project connects, so a key set for a platform you do not
use is silently inert rather than an error.

### Web

#### `sdk_version` — object, default `{web: 2}`
<sub>web</sub>

Amplitude Browser SDK major version: `1` or `2`. Version 2 changes attribution
and session behaviour — new destinations should stay on `2`.

#### `proxy_server_url` — object
<sub>web</sub>

Route SDK traffic through your own proxy instead of Amplitude's endpoint.
Must not begin with `http://` and must not contain `.ngrok.io`.

#### `prefer_anonymous_id_for_device_id` — object
<sub>web</sub>

Use RudderStack's `anonymousId` as the Amplitude device ID rather than letting
the Amplitude SDK generate its own.

#### `attribution` — object
<sub>web</sub>

Enable Amplitude's attribution tracking.

#### `track_new_campaigns` — object
<sub>web</sub>

Start a new session when a new campaign is detected. Browser SDK v2 only — see
`sdk_version`.

#### Auto-capture

`auto_capture` groups the Browser SDK's automatic instrumentation. Each setting
is an object with a single boolean `web` key, and all eight default to `false`:

`page_views`, `page_url_enrichment`, `web_vitals`, `file_downloads`,
`frustration_interactions`, `network_tracking`, `element_interactions`,
`form_interactions`.

```yaml
auto_capture:
  page_views:
    web: true
  web_vitals:
    web: true
```

### Web and mobile

#### `event_upload_period_millis` — object
<sub>web, android, ios, react_native, flutter</sub>

How long the SDK buffers events before uploading, in milliseconds. Written as a
digit string, not a number. The web SDK falls back to `1000` when unset.

#### `event_upload_threshold` — object
<sub>web, android, ios, react_native, flutter</sub>

How many events the SDK buffers before uploading. Written as a digit string, not
a number. The web SDK falls back to `30` when unset.

#### `track_session_events` — object, default `{web: false}`
<sub>web, android, ios, react_native, flutter</sub>

Emit Amplitude's automatic session start and end events.

### Mobile

#### `enable_location_listening` — object
<sub>android, react_native, flutter</sub>

Let the Amplitude SDK collect device location. Requires the corresponding
location permission in the app.

#### `use_advertising_id_for_device_id` — object
<sub>android, react_native, flutter</sub>

Use the Android advertising ID as the Amplitude device ID.

#### `use_idfa_as_device_id` — object
<sub>ios, react_native, flutter</sub>

Use the iOS IDFA as the Amplitude device ID. The app must request tracking
authorisation for the IDFA to be available.

## Keys accepted but not delivered

The eight keys below are part of the CLI config model and pass validation, but
the destination's upstream per-source key lists do not carry them for any source
type, so the control plane delivers them to neither the transformer nor any SDK.
Setting them has no observable effect.

`force_https`, `track_gclid`, `track_referrer`,
`save_params_referrer_once_per_session`, `device_id_from_url_param`,
`batch_events`, `track_utm_properties`, `unset_params_referrer_on_new_session`

Each is an object with a single boolean `web` key. They correspond to options of
the Amplitude Browser SDK v1 and are retained so that importing an existing
destination does not drop values already stored against it.

## Source types

| Source type | Modes |
| --- | --- |
| `web` | cloud, device |
| `android` | cloud, device |
| `ios` | cloud, device |
| `react_native` | cloud, device |
| `flutter` | cloud, device |
| `android_kotlin` | cloud |
| `ios_swift` | cloud |
| `unity` | cloud |
| `cordova` | cloud |
| `cloud` | cloud |

## Per-source keys

All three keys below are objects keyed by the local source type. A key naming a
source type this destination does not support fails validation.

#### `connection_mode` — object

Selects the mode per source type. Values are constrained to the modes that
source type supports, so `device` is rejected for the five cloud-only types:

```yaml
connection_mode:
  web: device
  cloud: cloud
```

#### `use_native_sdk` — object

Boolean per source type, accepting `web`, `ios`, `android`, `react_native` and
`flutter`. Set alongside `connection_mode` to load the Amplitude SDK for that
platform.

#### `consent_management` — object

Consent-provider configuration per source type. The entry shape, accepted
providers, and the rules on `resolution_strategy` and `consents` are shared
across all destinations and documented in
[../common/README.md](../common/README.md).

## Connecting a source

An event stream connection to this destination is checked against two rules at
`validate` time.

**The source's type must be supported.** A source's type is mapped to one of the
tokens above first — a JavaScript source resolves to `web`, and webhook and
server-side SDK sources resolve to `cloud`. An unsupported type reports:

```
destination 'amplitude' (type 'am') does not support source 'my-source':
source type 'amp' is not among supported source types: android, android_kotlin, ...
```

**The config must carry an entry for that source type** under `connection_mode`
or `use_native_sdk`. This lives on the destination spec, not the connection spec.
Without it:

```
destination 'amplitude' config has no 'connection_mode' or 'use_native_sdk' entry
for source type 'web'
```

Amplitude requires no additional config keys to connect a source of any type.

## Secrets

`api_secret` is the only secret key. Write it as a `{{ .VAR }}` reference and
supply the value at apply time:

```yaml
api_secret: "{{ .AMPLITUDE_API_SECRET }}"
```

```sh
export RUDDER_AMPLITUDE_API_SECRET=...
rudder-cli apply

# or
rudder-cli apply --var-file secrets.vars.yaml
```

`rudder-cli import` writes `api_secret` back as a `{{ .VAR }}` placeholder rather
than its value, since the API does not return secrets. Fill the placeholder in
before the first apply.
