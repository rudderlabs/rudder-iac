# Amplitude (`am`)

Amplitude is a product analytics destination. RudderStack sends events to Amplitude's HTTP API from its servers, or loads Amplitude's own SDKs in device mode on web, Android, iOS, React Native, and Flutter.

In an Amplitude destination spec:

- `type: am`
- `definition_version: 1`

## Sample configuration

```yaml
version: rudder/v1
kind: destination
metadata:
  name: amplitude-prod
spec:
  id: amplitude-prod
  display_name: Amplitude Production
  type: am
  definition_version: 1
  enabled: true
  config:
    api_key: "{{ .AMPLITUDE_API_KEY }}"
    api_secret: "{{ .AMPLITUDE_SECRET_KEY }}"
    residency_server: standard

    track_all_pages: false
    track_categorized_pages: true
    track_named_pages: true

    group_type_trait: company_type
    group_value_trait: company_name
    traits_to_set_once:
      - signup_date
    traits_to_increment:
      - login_count

    track_products_once: false
    track_revenue_per_product: true

    sdk_version:
      web: 2
    attribution:
      web: false
    track_session_events:
      web: true
      ios: true
    auto_capture:
      page_views:
        web: false
      web_vitals:
        web: true
    event_upload_threshold:
      web: "30"
      ios: "30"

    connection_mode:
      web: device
      ios: device
      cloud: cloud
    consent_management:
      web:
        - provider: oneTrust
          consents:
            - analytics
```

The above example loads Amplitude's SDKs on web and iOS in `device` mode, with web on SDK version 2 — which is what `auto_capture` and web `track_session_events` need. See [Web SDK](#web-sdk).

## Config keys

`config` accepts only the keys listed below. The [shared config key rules](../README.md#config-key-rules) cover unknown keys, defaults, and immutability.

> [!NOTE]
> Most Amplitude keys affect only one connection mode, or one platform's SDK. The groups below say which. Rudder CLI accepts every key whatever the mode — a key for the other mode is stored and ignored.
>
> In `cloud` mode Amplitude accepts `alias`, `group`, `identify`, `page`, `screen`, and `track`. In `device` mode, web sources send `identify`, `track`, `page`, and `group`; mobile sources send `identify`, `track`, and `screen`.

### Connection

#### `api_key` — string, required, secret

API key of your Amplitude project, from the project's **General** tab.

- At most 100 characters, and must not contain line breaks.
- A `{{ path || fallback }}` template is accepted in place of a literal.

#### `residency_server` — string, required

Amplitude data center RudderStack sends to.

- `standard` (US) or `EU`.
- The dashboard defaults this field to `standard`. Rudder CLI requires it explicitly.
- A spec that omits this key fails validation.

#### `api_secret` — string, secret

Secret key of your Amplitude project. RudderStack needs it to delete users, for example for GDPR requests. The dashboard calls it **Secret key**.

- At most 100 characters, and must not contain line breaks.
- A `{{ path || fallback }}` template is accepted in place of a literal.

### Page and screen events

`track_all_pages`, `track_categorized_pages`, and `track_named_pages` apply in `device` mode. On mobile they act on `screen` events rather than `page` events. Turning on more than one can send several Amplitude events for a single call. The custom name keys apply in `cloud` mode.

#### `track_all_pages` — boolean, default `false`

Send every `page` call to Amplitude as a `Loaded a Page` event.

- Applies to sources connected in `device` mode.

#### `track_categorized_pages` — boolean, default `true`

Send an event for each `page` call that has a category.

- Applies to sources connected in `device` mode.

#### `track_named_pages` — boolean, default `true`

Send an event for each `page` call that has a name.

- Applies to sources connected in `device` mode.

#### `use_user_defined_page_event_name` — boolean, default `false`

Name `page` events from `user_provided_page_event_string` instead of the default name.

- Applies to events sent in `cloud` mode.

#### `user_provided_page_event_string` — string

Format for `page` event names. Each `{{ field }}` is replaced with that field from the event payload — for example `Viewed {{ name }} page`.

- Applies when `use_user_defined_page_event_name` is `true`. Leave it unset otherwise.
- At most 200 characters, and must not contain line breaks.
- `{{ field }}`-style fields have no leading dot, so Rudder CLI's variable substitution leaves them alone.

#### `use_user_defined_screen_event_name` — boolean, default `false`

Name `screen` events from `user_provided_screen_event_string` instead of the default name.

- Applies to events sent in `cloud` mode.

#### `user_provided_screen_event_string` — string

Format for `screen` event names, on the same terms as `user_provided_page_event_string`.

- Applies when `use_user_defined_screen_event_name` is `true`. Leave it unset otherwise.
- At most 200 characters, and must not contain line breaks.

### Identify and group traits

The trait lists take arrays of trait names, each at most 100 characters or a `{{ path || fallback }}` template.

#### `group_type_trait` — string

Trait whose value becomes the Amplitude `group_type` in `group` calls — for example a trait holding `industry`.

- At most 100 characters, or a template.

#### `group_value_trait` — string

Trait whose value becomes the Amplitude `group_value` in `group` calls — for example `retail` for the `industry` type.

- At most 100 characters, or a template.

#### `traits_to_increment` — array of strings

`identify` traits Amplitude increments by their numeric value instead of overwriting.

#### `traits_to_set_once` — array of strings

`identify` traits Amplitude sets once and never overwrites.

#### `traits_to_append` — array of strings

`identify` traits whose values are appended to a list user property.

- Applies to `cloud` mode and to mobile sources in `device` mode — not to web device mode.

#### `traits_to_prepend` — array of strings

`identify` traits whose values are prepended to a list user property.

- Applies to `cloud` mode and to mobile sources in `device` mode — not to web device mode.

#### `enable_enhanced_user_operations` — boolean, default `false`

Enable Amplitude's enhanced user property operations.

- Applies to `cloud` mode.
- The dashboard shows this setting only to workspaces with the feature turned on.

### Ecommerce

#### `track_products_once` — boolean, default `false`

Send an event carrying a `products` array as one event, with the products as properties, instead of one event per product.

#### `track_revenue_per_product` — boolean, default `false`

Track each product's revenue individually. When `false`, the event carries the aggregate revenue of all products.

### Web SDK

These keys configure Amplitude's web SDK, so they apply only when `connection_mode.web` is `device`. Each object is keyed by `web`.

> [!WARNING]
> Several of these apply only to one web SDK version, set by `sdk_version.web`. `auto_capture` and `track_session_events.web` need version `2`; `track_new_campaigns` needs version `1`. A key for the other version is accepted and has no effect.

#### `sdk_version` — object

Version of Amplitude's web SDK RudderStack loads. Version 2 changes how the SDK behaves, so review the differences before switching an existing destination.

- `web` — `1` or `2`, written as a number, not a string.
- Rudder CLI fills in `2` only when the `sdk_version` block is present. Omit the block and no version is sent, so write it out.

```yaml
sdk_version:
  web: 2
```

#### `proxy_server_url` — object

Domain proxy that relays the SDK's requests to Amplitude.

- `web` — string. Must not start with `http://` and must not contain `.ngrok.io`. A template is accepted.
- The proxy must use HTTPS. Otherwise the proxy is ignored and data goes to Amplitude directly.

#### `version_name` — string

Version name sent to Amplitude with each event, for more detailed analysis.

- Applies when `connection_mode.web` is `device`.
- At most 100 characters, or a template.

#### `prefer_anonymous_id_for_device_id` — object

Use the RudderStack anonymous ID as Amplitude's device ID instead of the SDK's own.

- `web` — boolean.

#### `attribution` — object

**Disables** attribution tracking. `true` turns attribution off: GCLID, UTM parameters, and referrer aren't tracked. The dashboard calls this **Disable Attribution**.

- `web` — boolean.

> [!WARNING]
> Despite its name, `attribution: {web: true}` switches attribution **off**. To keep attribution, leave it `false` or unset.

#### `track_new_campaigns` — object

Save the referrer, URL parameters, and GCLID only **once per session**, ignoring new values that arrive later in the session. The dashboard calls this **Save Referrer, URL Params, GCLID only once per session** — it doesn't track new campaigns, whatever the name suggests.

- Applies when `sdk_version.web` is `1`. Leave it unset otherwise.
- `web` — boolean. The dashboard defaults it to `true`; Rudder CLI doesn't fill it in.

#### `auto_capture` — object

Interactions Amplitude's web SDK captures on its own. Each setting is an object keyed by `web`, with a boolean value that defaults to `false` when the setting is present.

- Applies when `sdk_version.web` is `2`. Leave it unset otherwise.
- `page_views` — emit `[Amplitude] Page Viewed` on navigation, in addition to RudderStack's `page` calls. Leave off if you already send `page` calls on navigation, or page views are counted twice.
- `page_url_enrichment` — attach the current page URL to every event.
- `web_vitals` — capture Core Web Vitals (LCP, FCP, INP, CLS, TTFB).
- `file_downloads` — capture clicks on download links.
- `frustration_interactions` — capture rage clicks, dead clicks, and error clicks.
- `network_tracking` — capture failed network requests.
- `element_interactions` — capture clicks and changes on every element. High volume, and can capture user-entered text.
- `form_interactions` — capture form submissions and field changes, including field values that may be PII.

```yaml
auto_capture:
  web_vitals:
    web: true
  file_downloads:
    web: true
```

### Mobile SDK

These keys configure Amplitude's mobile SDKs, so they apply only to mobile sources connected in `device` mode. Each is an object keyed by the source types listed, with a boolean value.

#### `map_device_brand` — boolean, default `false`

Send the device brand to Amplitude.

- A plain boolean, not keyed by source type. Applies to `android`, `ios`, `react_native`, and `flutter` in `device` mode.

#### `enable_location_listening` — object

Capture location for users who've granted the app location permission.

- `android`, `react_native`, `flutter` — booleans. The dashboard defaults them to `true`; Rudder CLI doesn't fill them in.

#### `use_advertising_id_for_device_id` — object

Send the Android Advertising ID as Amplitude's device ID.

- `android`, `react_native`, `flutter` — booleans.

#### `use_idfa_as_device_id` — object

Send the iOS IDFA as Amplitude's device ID.

- `ios`, `react_native`, `flutter` — booleans.

### Batching and sessions

These keys apply to both web and mobile SDKs, so they take effect for any source connected in `device` mode. Each is keyed by source type.

#### `event_upload_period_millis` — object

Time, in milliseconds, between the SDK's batch uploads.

- `web`, `android`, `ios`, `react_native`, `flutter` — digits only, written as a string: `"1000"`, not `1000`. At most 100 characters, or a template.
- The dashboard defaults it to `"1000"`; Rudder CLI doesn't fill it in.

```yaml
event_upload_period_millis:
  web: "1000"
  android: "5000"
```

#### `event_upload_threshold` — object

Minimum number of events the SDK collects before uploading a batch.

- `web`, `android`, `ios`, `react_native`, `flutter` — digits only, written as a string: `"30"`, not `30`. At most 100 characters, or a template.
- The dashboard defaults it to `"30"`; Rudder CLI doesn't fill it in.

#### `track_session_events` — object

Send session start and end events.

- `android`, `ios`, `react_native`, `flutter` — booleans.
- `web` — boolean, default `false` when the block is present. Applies only when `sdk_version.web` is `2`.

### Internal keys

The eight keys below are older Amplitude web SDK settings — the keys above are what configure today's integration. Rudder CLI accepts them so that a destination already carrying these values keeps them; without them in the spec, an update would drop whatever is stored. Leave them out of a new destination, and keep whatever `rudder-cli import` brings back.

Each is an object keyed by `web`, with a boolean value.

#### `batch_events` — object, internal

Batches events in the browser before uploading them. `event_upload_period_millis` and `event_upload_threshold` are the keys the dashboard exposes for batching.

#### `device_id_from_url_param` — object, internal

Sets the device ID from the `amp_device_id` URL parameter.

#### `force_https` — object, internal

Always uploads over HTTPS, rather than following the embedding page's protocol.

#### `track_gclid` — object, internal

Captures the `gclid` URL parameter alongside the user's `initial_gclid`. Attribution as a whole is controlled by `attribution`.

#### `track_referrer` — object, internal

Captures `referrer` and `referring_domain` for each session, alongside the user's initial values.

#### `track_utm_properties` — object, internal

Parses UTM parameters from the query string or `_utmz` cookie and sends them as user properties.

#### `save_params_referrer_once_per_session` — object, internal

Captures GCLID, referrer, and UTM parameters once per session. On web SDK version 1, `track_new_campaigns` is the key the dashboard exposes for this.

#### `unset_params_referrer_on_new_session` — object, internal

Resets `referrer` and UTM properties to null at the start of each session, instead of carrying the previous values forward.

### Event filtering

> [!WARNING]
> Client-side event filtering applies only to sources connected in `device` mode — the dashboard shows these controls only then, and the SDK is what applies the filter. Rudder CLI accepts the block in any mode, but it has no effect on cloud-mode events.

#### `event_filtering` — object

Restricts which `track` events the SDK passes to Amplitude, by event name.

- `whitelist` — array of event names to allow; every other `track` event is dropped.
- `blacklist` — array of event names to drop; every other `track` event is allowed.
- The two are mutually exclusive, and Rudder CLI enforces it — setting both fails validation.
- Each name is at most 100 characters, or a `{{ path || fallback }}` template.

### Per-source keys

Both keys are objects keyed by the local source type — the tokens listed under [Source types](#source-types). A key naming a source type this destination doesn't support fails validation.

#### `connection_mode` — object

Maps each source type you connect to the mode its events reach Amplitude in, using the modes in [Source types](#source-types).

- An entry is required for each source type you connect — see [Connect a source](#connect-a-source).
- A mode the source type doesn't support on this destination fails validation — for example `device` for `ios_swift`.

```yaml
connection_mode:
  web: device
  ios: device
  android_kotlin: cloud
```

#### `consent_management` — object

Consent provider configuration per source type. The entry shape, accepted providers, and the rules on `resolution_strategy` and `consents` are shared across all destination types — see [Consent management](../common/README.md).

## Source types

Amplitude accepts events from these source types in the mentioned connection modes:

| Source type | Connection mode |
| :-----| :-----|
| `android` | `cloud`, `device` |
| `android_kotlin` | `cloud` |
| `ios` | `cloud`, `device` |
| `ios_swift` | `cloud` |
| `web` | `cloud`, `device` |
| `unity` | `cloud` |
| `cloud` | `cloud` |
| `react_native` | `cloud`, `device` |
| `flutter` | `cloud`, `device` |
| `cordova` | `cloud` |
| `warehouse` | `cloud` |

`web`, `android`, `ios`, `react_native`, and `flutter` offer `device` mode. The Kotlin and Swift SDKs — `android_kotlin` and `ios_swift` — are `cloud` only.

`warehouse` is the token a Reverse ETL source resolves to — see [Source types](../README.md#source-types).

> [!NOTE]
> The dashboard additionally offers Amplitude to AMP and Shopify sources. Rudder CLI doesn't manage those connections, so `amp` and `shopify` are invalid here.

## Connect a source

An event stream connection to this destination is checked against two rules at `validate` time.

**The source's type must be supported.** A source's type resolves to one of the tokens above before the check — a JavaScript source resolves to `web`, and webhook and server-side SDK sources resolve to `cloud`. An unsupported type reports:

```text
destination 'amplitude-prod' (type 'am') does not support source 'my-source':
source type 'amp' is not among supported source types: android, android_kotlin, ...
```

**The destination config must carry a `connection_mode` entry for that source type.** This lives on the destination spec, not on the connection spec. Without it:

```text
destination 'amplitude-prod' config has no 'connection_mode' entry for source type 'web'
```

Amplitude needs no additional config keys to connect a source of any type, in any mode.

A Reverse ETL connection reaches this destination as source type `warehouse` and is checked against the same rules, so the config needs a `connection_mode.warehouse: cloud` entry for it.

## Secrets

`api_key` and `api_secret` are the secret keys. Write each as a `{{ .VAR }}` reference and supply the value at apply time:

```yaml
config:
  api_key: "{{ .AMPLITUDE_API_KEY }}"
  api_secret: "{{ .AMPLITUDE_SECRET_KEY }}"
```

```bash
export RUDDER_AMPLITUDE_API_KEY="..."
export RUDDER_AMPLITUDE_SECRET_KEY="..."
rudder-cli apply

# or
rudder-cli apply --var-file secrets.vars.yaml
```

Note that:

- The YAML that `rudder-cli import` writes may or may not include secret keys. Before you apply, make sure every secret key your configuration needs is present and populated through variable substitution.
- In device mode the API key is embedded in the app or page, so masking it protects your YAML, not the value itself. The secret key is used server-side, for user deletion.
