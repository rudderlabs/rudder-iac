# Amplitude (`am`)

Streaming destination. Amplitude receives events either from RudderStack's
servers (cloud mode) or directly from the Amplitude SDK embedded in your app
(device mode).

In a destination spec:

- `type: am`
- `definition_version: 1`

## Modes

The mode a source uses is set per source type in `config.connection_mode`. Five
of the ten supported source types accept `device`; the rest are cloud only — see
[Source types](#source-types).

Each key below is badged with where it applies:

- `cloud` — events sent from RudderStack's servers
- `device` — events sent by the Amplitude SDK; supported platforms follow the
  badge where support is limited to some of them

A key is accepted by `validate` whatever your sources use; the badges tell you
where the setting takes effect.

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
    track_all_pages: true
    track_categorized_pages: true
    track_named_pages: true

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

    # Device mode
    sdk_version:
      web: 2
    proxy_server_url:
      web: "https://amplitude-proxy.example.com"
    prefer_anonymous_id_for_device_id:
      web: true
    attribution:
      web: true
    track_new_campaigns:
      web: true
    auto_capture:
      page_views:
        web: true
      web_vitals:
        web: true
    event_upload_period_millis:
      web: "1000"
      android: "30000"
    event_upload_threshold:
      web: "30"
      ios: "30"
    track_session_events:
      web: true
      android: true
      ios: true
    enable_location_listening:
      android: true
    use_advertising_id_for_device_id:
      android: false
    use_idfa_as_device_id:
      ios: false

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

A `*` after a key name marks a description written without a Terraform provider
source to draw on. Those need a closer review pass; the markers come out once the
wording is confirmed.

### Connection

#### `api_key` — string, required
`cloud` `device`

Your Amplitude project API key. At most 100 characters.

#### `api_secret` — string, secret
`cloud`

The Amplitude API secret key, required for user deletion. At most 100
characters.

Supply it as a `{{ .VAR }}` reference — see [Secrets](#secrets).

#### `residency_server` \* — string, required
`cloud` `device` · web, android

Amplitude data residency region, which determines the endpoint events are sent
to. One of `standard` or `EU`.

### Page and screen tracking

#### `use_user_defined_page_event_name` \* — boolean, default `false`
`cloud`

Send `page` calls under a fixed custom event name rather than the derived one.

#### `user_provided_page_event_string` \* — string
`cloud`

The event name used when the flag above is set. At most 200 characters.

#### `use_user_defined_screen_event_name` \* — boolean, default `false`
`cloud`

The `screen` equivalent of `use_user_defined_page_event_name`.

#### `user_provided_screen_event_string` \* — string
`cloud`

The event name used when the flag above is set. At most 200 characters.

#### `track_all_pages` — boolean, default `false`
`device` · web, android, ios

Send an event named `Loaded a page` / `Loaded a Screen` to Amplitude.

#### `track_categorized_pages` — boolean, default `true`
`device` · web, android, ios

When `category` is present in a `page` / `screen` call, send an event named
`Viewed {category} page` / `Viewed {category} Screen`.

#### `track_named_pages` — boolean, default `true`
`device` · web, android, ios

When `name` is present in a `page` call, send an event named
`Viewed {name} page`.

### Identify and traits

#### `group_type_trait` — string
`cloud` `device` · web, ios

Used as the `groupType` in `group` calls. At most 100 characters.

#### `group_value_trait` — string
`cloud` `device` · web, ios

Used as the `groupValue` in `group` calls. At most 100 characters.

#### `traits_to_increment` — string array
`cloud` `device` · web, android, ios

Traits whose value is incremented at Amplitude by the value provided against the
trait in an `identify` call. Each entry at most 100 characters.

#### `traits_to_set_once` — string array
`cloud` `device` · web, android, ios

Traits set once at Amplitude, with the value provided against the trait in an
`identify` call. Each entry at most 100 characters.

#### `traits_to_append` — string array
`cloud` `device` · android, ios

Traits whose value is appended to the corresponding trait array at Amplitude.
Each entry at most 100 characters.

#### `traits_to_prepend` — string array
`cloud` `device` · android, ios

Traits whose value is prepended to the corresponding trait array at Amplitude.
Each entry at most 100 characters.

#### `enable_enhanced_user_operations` \* — boolean, default `false`
`cloud`

Enable Amplitude's enhanced user property operations, extending the increment,
set-once, append and prepend lists above to group identify calls.

### Ecommerce and revenue

#### `track_products_once` — boolean, default `false`
`cloud` `device` · web, android, ios

When the event payload contains an array of products, track the event under its
original name with all products as a property. Otherwise each product is tracked
as `Product purchased`.

#### `track_revenue_per_product` — boolean, default `false`
`cloud` `device` · web, android, ios

When the payload contains multiple products, track each product's revenue
individually.

### Other

#### `version_name` — string
`cloud` `device` · web, ios

Set as the `versionName` of the Amplitude SDK. At most 100 characters.

#### `map_device_brand` — boolean, default `false`
`cloud` `device` · ios

Send the device brand information (`context.device.brand`) to Amplitude.

#### `event_filtering` — object
`cloud` `device`

Filter which events are sent to Amplitude. Exactly one of the two lists may be
set — declaring both fails validation.

- `whitelist` — event names to allowlist
- `blacklist` — event names to denylist

Each entry is at most 100 characters. Omit the block to send every event.

## Device mode only

The keys below configure the Amplitude SDK and apply only in device mode. Each
is an object keyed by platform, using the local (snake_case) source type as the
key, and each supports the platforms listed on its badge.

`validate` accepts these keys whatever sources the project connects, so you can
declare a platform ahead of connecting a source of that type.

### Web

#### `sdk_version` — object, default `{web: 2}`
`device` · web

The Amplitude Browser SDK version to load for web sources: `1` or `2`.

#### `proxy_server_url` \* — object
`device` · web

Route SDK traffic through your own proxy instead of Amplitude's endpoint. Must
not begin with `http://` and must not contain `.ngrok.io`.

#### `prefer_anonymous_id_for_device_id` — object
`device` · web

Set the device ID to the `anonymousId` generated by the RudderStack SDK, or the
one set via `setAnonymousId()`.

#### `attribution` \* — object
`device` · web

Enable Amplitude's attribution tracking.

#### `track_new_campaigns` \* — object
`device` · web

Start a new session when a new campaign is detected. Browser SDK v2 only — see
`sdk_version`.

#### Auto-capture

`auto_capture` configures the AutoCapture settings of Amplitude Browser SDK v2
for web sources. Each setting is an object with a single boolean `web` key, and
all eight default to `false`:

- `page_views` — track page views automatically
- `page_url_enrichment` — enrich page view events with URL properties
- `web_vitals` — capture web vitals automatically
- `file_downloads` — capture file downloads automatically
- `frustration_interactions` — capture frustration interactions automatically
- `network_tracking` — capture network requests automatically
- `element_interactions` — capture element interactions automatically
- `form_interactions` — capture form interactions automatically

```yaml
auto_capture:
  page_views:
    web: true
  web_vitals:
    web: true
```

### Web and mobile

#### `event_upload_period_millis` — object
`device` · web, android, ios, react_native, flutter

How long the SDK waits before uploading batched events, in milliseconds. Written
as a digit string, not a number.

#### `event_upload_threshold` — object
`device` · web, android, ios, react_native, flutter

The minimum number of events the Amplitude SDK batches together before
uploading. Written as a digit string, not a number.

#### `track_session_events` — object, default `{web: false}`
`device` · web, android, ios, react_native, flutter

Track Amplitude's session events.

### Mobile

#### `enable_location_listening` — object
`device` · android, react_native, flutter

Activate location listening in the Amplitude SDK.

#### `use_advertising_id_for_device_id` — object
`device` · android, react_native, flutter

Set the advertising ID as the device ID.

#### `use_idfa_as_device_id` — object
`device` · ios, react_native, flutter

Set the IDFA as the device ID.

## Keys accepted but not delivered

The eight keys below are accepted for compatibility with destinations configured
against earlier versions of the Amplitude integration. They are not part of the
current integration and do not affect event delivery in either mode.

Each is an object with a single boolean `web` key, carried over from Amplitude
Browser SDK v1:

- `force_https` — always upload over HTTPS rather than the embedding site's
  protocol
- `track_gclid` — capture the `gclid` URL parameter along with the user's
  `initial_gclid`
- `track_referrer` — capture `referrer` and `referring_domain` per session,
  along with `initial_referrer` and `initial_referring_domain`
- `track_utm_properties` — parse UTM parameters from the query string or `_utmz`
  cookie and include them as user properties
- `save_params_referrer_once_per_session` — track `gclid`, referrer and UTM
  parameters once per session
- `unset_params_referrer_on_new_session` — set `referrer` and `utm_parameter` to
  null on a new session rather than carrying the existing values forward
- `device_id_from_url_param` — set the device ID from the `amp_device_id` URL
  parameter
- `batch_events` — batch events together before upload

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

Both keys below are objects keyed by the local source type. A key naming a
source type this destination does not support fails validation.

#### `connection_mode` \* — object

Selects the mode per source type. Values are constrained to the modes that
source type supports, so `device` is rejected for the five cloud-only types:

```yaml
connection_mode:
  web: device
  cloud: cloud
```

#### `consent_management` — object

Specify consent configuration data for multiple providers, per source type. The
entry shape, accepted providers, and the rules on `resolution_strategy` and
`consents` are shared across all destinations and documented in
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

**The config must carry a `connection_mode` entry for that source type.** This
lives on the destination spec, not the connection spec. Without it:

```
destination 'amplitude' config has no 'connection_mode' entry for source type 'web'
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
