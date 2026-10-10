# Adobe Analytics (`adobe_analytics`)

Adobe Analytics is a web and app analytics destination. RudderStack sends events to Adobe's Data Insertion API from its servers, or loads Adobe's own libraries in device mode — in the browser, and in Android and iOS apps.

In an Adobe Analytics destination spec:

- `type: adobe_analytics`
- `definition_version: 1`

> [!NOTE]
> `adobe_analytics` is an [unverified destination type](../README.md#unverified-destination-types). Rudder CLI registers it only when the `unverifiedDestinations` experimental flag is on.

## Sample configuration

```yaml
version: rudder/v1
kind: destination
metadata:
  name: adobe-analytics-prod
spec:
  id: adobe-analytics-prod
  display_name: Adobe Analytics Production
  type: adobe_analytics
  definition_version: 1
  enabled: true
  config:
    report_suite_ids: acmeprod,acmeglobal
    tracking_server_url: acme.sc.omtrdc.net
    tracking_server_secure_url: acme.sc.omtrdc.net
    marketing_cloud_org_id: 1A2B3C4D5E6F7A8B9C0D1E2F@AdobeOrg

    drop_visitor_id: false
    timestamp_option: disabled
    prefer_visitor_id: false

    rudder_events_to_adobe_events:
      - from: Signed Up
        to: event1
    track_page_name: true

    context_data_mapping:
      - from: section
        to: siteSection
    context_data_prefix: rs
    e_var_mapping:
      - from: plan
        to: "2"
    list_mapping:
      - from: interests
        to: "1"
        delimiter: ","
    custom_props_mapping:
      - from: category
        to: "3"
        delimiter: "|"

    event_merch_event_to_adobe_event:
      - from: Order Completed
        to: event5
    event_merch_properties:
      - revenue
    product_identifier: sku

    heartbeat_tracking_server_url: acme.hb.omtrdc.net
    events_to_types:
      - from: Video Playback Started
        to: heartbeatPlaybackStarted
      - from: Video Playback Completed
        to: heartbeatPlaybackCompleted

    event_filtering:
      blacklist:
        - Product Viewed

    connection_mode:
      web: device
      cloud: cloud
    consent_management:
      web:
        - provider: oneTrust
          consents:
            - analytics
```

The above example loads Adobe's library on web sources in `device` mode and sends server-side events in `cloud` mode, which needs `tracking_server_secure_url` — see [Connection](#connection). It sends `userId` as Adobe's visitor ID, and sets up video heartbeat tracking for the web library — see [Identity and timestamps](#identity-and-timestamps) and [Video heartbeat](#video-heartbeat).

## Config keys

`config` accepts only the keys listed below. The [shared config key rules](../README.md#config-key-rules) cover unknown keys, defaults, and immutability.

> [!NOTE]
> Adobe Analytics keys split by connection mode and platform. The groups below say which mode each key affects. Rudder CLI accepts every key whatever the mode — a key for another mode is stored and ignored.
>
> In `cloud` mode Adobe Analytics accepts `track`, `page`, and `screen`. In `device` mode, web sources send `page` and `track`; Android and iOS sources send `identify`, `track`, and `screen`.

### Connection

These keys tell Adobe Analytics where the data belongs. In web device mode, a report suite or tracking server the page already sets on Adobe's `s_account` or `s` object takes precedence over these keys.

#### `report_suite_ids` — string, required

Report suite IDs RudderStack sends events to, separated by commas — for example `acmeprod,acmeglobal`. They're listed on the Adobe Analytics settings page.

- Applies to `cloud` mode and to web sources in `device` mode. Android and iOS device mode take their report suite from the app's `ADBMobileConfig.json`.
- At most 300 characters, and must not contain line breaks.
- A `{{ path || fallback }}` template is accepted in place of a literal.
- A spec that omits this key fails validation with `'report_suite_ids' is required`.

#### `tracking_server_url` — string

Hostname of your Adobe Analytics tracking server — Adobe's `trackingServer` variable — without `http://` or `https://`, for example `acme.sc.omtrdc.net`.

- Applies to `device` mode on `web` sources only.
- At most 100 characters, and must not contain line breaks. Any value containing `.ngrok.io` is rejected.
- A `{{ path || fallback }}` template is accepted in place of a literal.

#### `tracking_server_secure_url` — string

Hostname of your Adobe Analytics tracking server for HTTPS requests — Adobe's `trackingServerSecure` variable — for example `acme.sc.omtrdc.net`. Cloud mode sends every event to this host.

- Applies to `cloud` mode and to web sources in `device` mode.
- Rudder CLI doesn't require this key, even when a source connects in `cloud` mode. Without it, RudderStack can't deliver cloud-mode events.
- At most 100 characters, and must not contain line breaks. Any value containing `.ngrok.io` is rejected.
- A `{{ path || fallback }}` template is accepted in place of a literal.

#### `marketing_cloud_org_id` — string

Your Adobe Marketing Cloud organization ID, for example `1A2B3C4D5E6F7A8B9C0D1E2F@AdobeOrg`. In web device mode, setting it starts Adobe's Visitor ID service (`visitorAPI.js`) with this ID; in cloud mode it's sent with each event.

- Applies to `cloud` mode and to web sources in `device` mode.
- At most 100 characters, and must not contain line breaks.

### Identity and timestamps

These keys apply in `cloud` mode and to web sources in `device` mode. Adobe doesn't accept both a visitor ID and a timestamp in one hit, so `drop_visitor_id`, `timestamp_option`, and `prefer_visitor_id` together decide which one each event carries:

- The visitor ID — the event's `userId` — is set when `drop_visitor_id` is `false` and `timestamp_option` is `disabled`, or `hybrid` with `prefer_visitor_id` on.
- The timestamp is set when `timestamp_option` is `enabled`, or `hybrid` with `prefer_visitor_id` off. In `cloud` mode it also needs `timestamp_optional_reporting`.

#### `drop_visitor_id` — boolean, default `true`

Don't send the event's `userId` as Adobe's visitor ID.

#### `timestamp_option` — string, default `disabled`

Whether your report suites accept timestamped data, non-timestamped data, or both.

- `disabled`, `enabled`, `hybrid`, or `optional`.
- `optional` sets neither a visitor ID nor a timestamp.

#### `prefer_visitor_id` — boolean, default `false`

With `timestamp_option: hybrid`, send the visitor ID instead of the timestamp.

#### `timestamp_optional_reporting` — boolean, default `false`

Send event timestamps in cloud mode, for report suites where timestamps are optional. The dashboard calls it **Send Both Timestamp and VisitorID for Timestamp Optional Reporting Suites**.

- Applies to `cloud` mode only. Cloud mode sends no timestamp unless this is `true`.
- Adobe advises against it, because it can put data out of order. It also needs **Optional Timestamp** turned on for the report suite.

#### `no_fallback_visitor_id` — boolean, default `false`

Don't set Adobe's fallback visitor ID. When `false`, RudderStack sets it from the event's `AdobeFallbackVisitorId` external ID, if there is one.

- Applies to `cloud` mode only.

### Event mapping

#### `rudder_events_to_adobe_events` — array of objects

Maps RudderStack event names to Adobe events.

- `from` — RudderStack event name.
- `to` — Adobe events to send for it, separated by commas, such as `event1,event2`. On Android and iOS, it's the action name sent to Adobe.
- Both are required in each entry, each at most 100 characters and without line breaks.
- Applies in every mode. In `cloud` mode, an event that's neither an ecommerce event nor mapped here fails, unless it carries `properties.overrideEventName`. That includes `page` and `screen` calls, which cloud mode matches by name.

```yaml
rudder_events_to_adobe_events:
  - from: Signed Up
    to: event1
  - from: Plan Upgraded
    to: event2,event3
```

#### `track_page_name` — boolean, default `true`

Send a `pageName` with every `track` event. The dashboard calls it **Enable pageName for Track Events**.

- Applies to `cloud` mode and to web sources in `device` mode. In `cloud` mode it affects only events whose `properties.overridePageView` is `true`.

### Variables

These keys map event fields to Adobe's custom variables. In every mapping, `from` and `to` are both required, each at most 100 characters and without line breaks. Index values are digit strings — write `"2"`, not `2`, or validation fails.

#### `context_data_mapping` — array of objects

Maps event fields to Adobe context data variables, which Adobe processing rules can read.

- `from` — event field, such as a property name.
- `to` — context data variable name.
- Applies in every mode.

```yaml
context_data_mapping:
  - from: section
    to: siteSection
```

#### `context_data_prefix` — string

Prefix added to context data variable names. The dashboard calls it **Prefix to prepend to all contextData properties**.

- In `cloud` mode, it's prepended to each variable name from `context_data_mapping`.
- In `device` mode, the SDK also sends event properties as context data under their own names, and prefixes those instead — `<prefix>.<property>` on web, `<prefix><property>` on Android and iOS.
- At most 100 characters, and must not contain line breaks.

#### `e_var_mapping` — array of objects

Maps event fields to Adobe eVars.

- `from` — event field, such as a property name.
- `to` — eVar index, written as a string: `"2"` for `eVar2`.
- Applies to `cloud` mode and to web sources in `device` mode.

```yaml
e_var_mapping:
  - from: plan
    to: "2"
```

#### `hier_mapping` — array of objects

Maps event fields to Adobe hierarchy variables, on the same terms as `e_var_mapping` — `to` is the index, such as `"1"` for `hier1`.

- Applies to `cloud` mode and to web sources in `device` mode.

#### `list_mapping` — array of objects

Maps event properties to Adobe list variables.

- `from` — event property holding an array, or a comma-separated string.
- `to` — list index, written as a string: `"1"` for `list1`.
- `delimiter` — `|`, `:`, `,`, `;`, or `/`, used to join the values. Set it on every entry: an entry without one isn't sent. The dashboard defaults it to `,`; Rudder CLI doesn't fill it in.
- Applies to `cloud` mode and to web sources in `device` mode.

```yaml
list_mapping:
  - from: interests
    to: "1"
    delimiter: ","
```

#### `custom_props_mapping` — array of objects

Maps event properties to Adobe props, on the same terms as `list_mapping` — `to` is the prop index, such as `"3"` for `prop3`.

- Set `delimiter` on every entry here too. Without one, `cloud` mode skips the entry, and web device mode joins the values with `|`.
- Applies to `cloud` mode and to web sources in `device` mode.

#### `mobile_event_mapping` — array of objects

The dashboard labels it **Map RudderStack field to Adobe mobile events**, but no connection mode reads it. Android and iOS device mode take event names from `rudder_events_to_adobe_events`, and context data from `context_data_mapping`.

Rudder CLI accepts it so that a destination already carrying a value keeps it. Leave it out of a new destination.

### Merchandising

These keys build Adobe's merchandising event and product strings for ecommerce events. They apply in `cloud` mode and to web sources in `device` mode, except `product_identifier`, which applies in every mode. In every mapping, `from` and `to` are both required, each at most 100 characters and without line breaks.

#### `event_merch_event_to_adobe_event` — array of objects

Maps RudderStack events to Adobe merchandising events of the currency or counter type, in the event string.

- `from` — RudderStack event name, such as `Order Completed`.
- `to` — Adobe event, such as `event5`.

#### `event_merch_properties` — array of strings

Event properties holding the currency or counter values for the events mapped in `event_merch_event_to_adobe_event`. With `revenue` listed, an `Order Completed` mapped to `event5` produces an event string such as `purchase,event5=19.9`.

- Each name is at most 100 characters, and must not contain line breaks.

#### `product_merch_event_to_adobe_event` — array of objects

Maps RudderStack events to Adobe merchandising events of the currency or counter type, in the product string. Same shape as `event_merch_event_to_adobe_event`.

#### `product_merch_properties` — array of strings

Properties holding the currency or counter values for the events mapped in `product_merch_event_to_adobe_event`. A name starting with `products.`, such as `products.price`, is read from each product.

- Each name is at most 100 characters, and must not contain line breaks.

#### `product_merch_evars_map` — array of objects

Maps properties to merchandising eVars in the product string, for the events mapped in `product_merch_event_to_adobe_event`. RudderStack joins them with `|` and appends them to each product.

- `from` — property name.
- `to` — eVar index, written as a string, such as `"4"`.

#### `product_identifier` — string, default `name`

Product field that identifies each product to Adobe, which accepts only one.

- `name`, `id`, or `sku`. With `id`, RudderStack reads `product_id`, falling back to `id`.

### Video heartbeat

These keys configure Adobe's heartbeat tracking for video, so they apply only to sources connected in `device` mode — web, Android, or iOS.

#### `heartbeat_tracking_server_url` — string

Hostname of your Adobe heartbeat tracking server, such as `acme.hb.omtrdc.net`. Adobe provides it. Setting it turns on heartbeat tracking — on web, RudderStack loads Adobe's heartbeat library in place of the standard one.

- At most 100 characters, and must not contain line breaks. Any value containing `.ngrok.io` is rejected.
- Templates are rejected — the value must be a literal.

#### `ssl_heartbeat` — boolean, default `true`

Send heartbeat calls over HTTPS.

#### `events_to_types` — array of objects

Maps RudderStack video events to Adobe heartbeat events.

- `from` — RudderStack event name. Required, at most 100 characters and without line breaks.
- `to` — one of `initHeartbeat`, `heartbeatPlaybackStarted`, `heartbeatPlaybackPaused`, `heartbeatPlaybackResumed`, `heartbeatPlaybackCompleted`, `heartbeatPlaybackInterrupted`, `heartbeatContentStarted`, `heartbeatContentComplete`, `heartbeatAdBreakStarted`, `heartbeatAdBreakCompleted`, `heartbeatAdStarted`, `heartbeatAdCompleted`, `heartbeatAdSkipped`, `heartbeatSeekStarted`, `heartbeatSeekCompleted`, `heartbeatBufferStarted`, `heartbeatBufferCompleted`, `heartbeatQualityUpdated`, or `heartbeatUpdatePlayhead`. Rudder CLI doesn't require it, but an entry without one maps nothing.
- On web, applies only when `heartbeat_tracking_server_url` is set. Android and iOS don't support `initHeartbeat` or `heartbeatUpdatePlayhead`.

```yaml
events_to_types:
  - from: Video Playback Started
    to: heartbeatPlaybackStarted
```

### Web library hosting

Both keys apply to `device` mode on `web` sources only. Each is at most 100 characters, without line breaks, and any value containing `.ngrok.io` is rejected. Templates are rejected — the value must be a literal.

#### `proxy_normal_url` — string

URL of your own copy of Adobe's JavaScript library. When omitted, the SDK loads RudderStack's hosted copy from `cdn.rudderlabs.com`.

- Used when `heartbeat_tracking_server_url` is unset.

#### `proxy_heartbeat_url` — string

URL of your own copy of Adobe's heartbeat library. When omitted, the SDK loads RudderStack's hosted copy.

- Used when `heartbeat_tracking_server_url` is set.

### Event filtering

> [!WARNING]
> Client-side event filtering is applied by the RudderStack SDK, so it affects only sources connected in `device` mode — web, Android, or iOS. Events sent in `cloud` mode reach Adobe Analytics unfiltered, whatever this block says.

#### `event_filtering` — object

Restricts which `track` events the SDK passes to Adobe Analytics, by event name.

- `whitelist` — array of event names to allow; every other `track` event is dropped.
- `blacklist` — array of event names to drop; every other `track` event is allowed.
- The two are mutually exclusive, and Rudder CLI enforces it — setting both fails validation.
- Each name is at most 100 characters, and must not contain line breaks.

### Per-source keys

Both keys are objects keyed by the local source type — the tokens listed under [Source types](#source-types). A key naming a source type this destination doesn't support fails validation.

#### `connection_mode` — object

Maps each source type you connect to the mode its events reach Adobe Analytics in, using the modes in [Source types](#source-types).

- An entry is required for each source type you connect — see [Connect a source](#connect-a-source).
- A mode the source type doesn't support on this destination fails validation — for example `device` for `react_native`.

```yaml
connection_mode:
  web: device
  ios: device
  cloud: cloud
```

#### `consent_management` — object

Consent provider configuration per source type. The entry shape, accepted providers, and the rules on `resolution_strategy` and `consents` are shared across all destination types — see [Consent management](../common/README.md).

## Source types

Adobe Analytics accepts events from these source types in the mentioned connection modes:

| Source type | Connection mode |
| :-----| :-----|
| `android` | `cloud`, `device` |
| `android_kotlin` | `cloud` |
| `ios` | `cloud`, `device` |
| `ios_swift` | `cloud` |
| `web` | `cloud`, `device` |
| `unity` | `cloud` |
| `react_native` | `cloud` |
| `flutter` | `cloud` |
| `cordova` | `cloud` |
| `cloud` | `cloud` |

`web`, `android`, and `ios` offer `device` mode. On mobile, device mode runs through the legacy Android (Java) and iOS (Obj-C) SDKs, which load Adobe's mobile SDK in the app; the Kotlin and Swift SDKs — `android_kotlin` and `ios_swift` — are `cloud` only.

The dashboard also accepts `warehouse` for Adobe Analytics, but this definition doesn't, so Rudder CLI can't connect a Reverse ETL source to it.

## Connect a source

An event stream connection to this destination is checked against two rules at `validate` time.

**The source's type must be supported.** A source's type resolves to one of the tokens above before the check — a JavaScript source resolves to `web`, and webhook and server-side SDK sources resolve to `cloud`. Every event stream source type is supported here, so only a Reverse ETL connection, whose source resolves to `warehouse`, fails it:

```text
destination 'adobe-analytics-prod' (type 'adobe_analytics') does not accept rETL sources: source type 'warehouse' is not among supported source types: android, android_kotlin, ...
```

**The destination config must carry a `connection_mode` entry for that source type.** This lives on the destination spec, not on the connection spec. Without it:

```text
destination 'adobe-analytics-prod' config has no 'connection_mode' entry for source type 'web'
```

Rudder CLI requires no additional config keys to connect a source of any type, in any mode — though `cloud` mode can't deliver events without [`tracking_server_secure_url`](#tracking_server_secure_url--string).

## Secrets

Adobe Analytics has no secret keys. Rudder CLI treats none of its keys as a secret, and the dashboard masks none, so every value is written to YAML in plain text and `rudder-cli import` brings every value back. In web device mode, the report suite IDs and tracking servers are embedded in the page's JavaScript anyway.
