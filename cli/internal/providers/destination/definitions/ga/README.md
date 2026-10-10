# Google Analytics (`ga`)

Google Analytics (Universal Analytics) is a web analytics destination. RudderStack sends events to Google Analytics' Measurement Protocol from its servers, or — for web sources — loads Google's `analytics.js` library in device mode.

In a Google Analytics destination spec:

- `type: ga`
- `definition_version: 1`

> [!NOTE]
> `ga` is an [unverified destination type](../README.md#unverified-destination-types). Rudder CLI registers it only when the `unverifiedDestinations` experimental flag is on.

> [!WARNING]
> RudderStack has deprecated the Google Analytics destination, since Google has sunset Universal Analytics. To send events to Google Analytics 4, use the [`ga4`](../ga4/README.md) type instead.

## Sample configuration

```yaml
version: rudder/v1
kind: destination
metadata:
  name: google-analytics-prod
spec:
  id: google-analytics-prod
  display_name: Google Analytics Production
  type: ga
  definition_version: 1
  enabled: true
  config:
    tracking_id: UA-123456789-1
    send_user_id: true
    anonymize_ip: true
    enhanced_ecommerce: true

    dimensions:
      - from: plan
        to: dimension1
    metrics:
      - from: age
        to: metric1
    content_groupings:
      - from: section
        to: contentGroup1

    enable_server_side_identify: true
    server_side_identify:
      event_category: user_type
      event_action: User Enriched

    track_categorized_pages:
      web: true
    track_named_pages:
      web: true
    sample_rate:
      web: "100"
    site_speed_sample_rate:
      web: "1"

    event_filtering:
      blacklist:
        - Cart Viewed

    connection_mode:
      web: device
      cloud: cloud
    consent_management:
      web:
        - provider: oneTrust
          consents:
            - analytics
```

The above example loads `analytics.js` on web sources in `device` mode, where the `web`-keyed settings apply, and sends server-side events in `cloud` mode, where `enable_server_side_identify` lets `identify` calls through — see [Web device mode](#web-device-mode) and [Cloud mode](#cloud-mode).

## Config keys

`config` accepts only the keys listed below. The [shared config key rules](../README.md#config-key-rules) cover unknown keys, defaults, and immutability.

> [!NOTE]
> Google Analytics settings split by connection mode. The groups below say which mode each key affects. Rudder CLI accepts every key whatever the mode — a key for the other mode is stored and ignored.
>
> In `cloud` mode Google Analytics accepts `identify`, `page`, `screen`, and `track`. In `device` mode, web sources send `page` and `track`.

### Connection

#### `tracking_id` — string, required

Tracking ID of your Universal Analytics property, from **Admin** > **Tracking Info** > **Tracking Code** in Google Analytics.

- `UA-`, `YT-`, or `MO-`, then digits, a hyphen, and up to 100 more digits — for example `UA-123456789-1`.
- A Google Analytics 4 measurement ID (`G-…`) is rejected. GA4 properties take the [`ga4`](../ga4/README.md) type.
- A `{{ path || fallback }}` template is accepted in place of a literal.
- A spec that omits this key fails validation with `'tracking_id' is required`.

### Tracking

These keys apply in both modes.

#### `send_user_id` — boolean, default `false`

Send the event's `userId` to Google Analytics as the user ID, for identified visitors. The dashboard calls it **Send user-id to GA**.

#### `anonymize_ip` — boolean, default `false`

Anonymize visitors' IP addresses.

#### `include_search` — boolean, default `false`

Include the query string in the page path sent with page views. The dashboard calls it **Include the Querystring in Page Views**.

#### `non_interaction` — boolean, default `false`

Mark every event as a non-interaction hit, which doesn't count toward bounce rate. An event's own `properties.nonInteraction` overrides it.

#### `enhanced_ecommerce` — boolean, default `false`

Send ecommerce events through Google Analytics enhanced ecommerce. It must be turned on in Google Analytics too.

#### `enhanced_link_attribution` — boolean, default `false`

Turn on Google Analytics enhanced link attribution, for more detailed reports of the links visitors click. In `cloud` mode, the link ID comes from the event's `properties.linkid`.

#### `double_click` — boolean, default `false`

Turn on Google's advertising features — remarketing, display ads, and demographic reports. The dashboard calls it **Remarketing, Display Ads and Demographic Reports**.

- In web `device` mode, it loads Google's display features plugin.
- In `cloud` mode, RudderStack instead marks each hit with the Measurement Protocol's `npa=1`, which disables ads personalization for that hit.

### Custom dimensions and content groupings

These keys apply in both modes. In each mapping, `from` and `to` are each at most 100 characters, or a `{{ path || fallback }}` template.

#### `dimensions` — array of objects

Maps event properties and traits to Google Analytics custom dimensions, which must exist in Google Analytics first.

- `from` — event property or trait name.
- `to` — custom dimension, written `dimension<index>`, such as `dimension1`.

```yaml
dimensions:
  - from: plan
    to: dimension1
```

#### `metrics` — array of objects

Maps numeric event properties and traits to custom metrics.

- `from` — event property or trait name.
- `to` — custom metric, written `metric<index>`, such as `metric1`.

#### `content_groupings` — array of objects

Maps event properties and traits to content groupings, which group pages together in reports.

- `from` — event property or trait name.
- `to` — content grouping, written `contentGroup<index>`, such as `contentGroup1`.

#### `custom_mappings` — array of objects, internal

An older, combined form of `dimensions` and `metrics`: an entry whose `to` starts with `cd` is a custom dimension, and one starting with `cm` is a custom metric. Cloud mode reads it only when neither `dimensions` nor `metrics` is set. The dashboard keeps this field hidden.

Leave it out of a new destination, and keep whatever `rudder-cli import` brings back.

### Cloud mode

These keys apply to events sent in `cloud` mode.

#### `enable_server_side_identify` — boolean, default `false`

Send `identify` calls to Google Analytics as events. When `false`, `identify` events sent in `cloud` mode fail with an error.

#### `server_side_identify` — object

Event category and action for the events `identify` calls become.

- `event_category` — name of the `identify` trait whose value RudderStack sends as the event category. When the event has no such trait, the category is `All`.
- `event_action` — event action.
- Set both or neither — Rudder CLI rejects one without the other. Omit the block to send the category `All` and the action `User Enriched`.
- Each is at most 100 characters, or a `{{ path || fallback }}` template.

```yaml
server_side_identify:
  event_category: user_type
  event_action: User Enriched
```

> [!WARNING]
> Despite its name, and the dashboard's **Server Side Identify Event Category** label, `event_category` isn't the category itself. RudderStack sends the value of the trait it names — `event_category: user_type` sends each user's `user_type` trait.

#### `disable_md5` — boolean, default `false`

Don't fall back to an MD5 hash of `userId` for the client ID. RudderStack takes the client ID from the `integrations` object, the `gaExternalId` external ID, `traits.cid`, or `anonymousId`, in that order. When none is set, it hashes `userId`, unless this is `true`. The dashboard calls it **Disable Md5 encryption from Client ID**.

### User deletion

#### `rudder_delete_account_id` — string

ID of the Google account RudderStack uses to delete users from Google Analytics, for example on a suppress-with-delete regulation. The dashboard calls it **Data regulation account**, and creates the account when you sign in with Google.

- Applies whatever the connection mode — deletion runs from RudderStack's servers.
- At most 100 characters, or a `{{ path || fallback }}` template.

### Web device mode

These keys configure `analytics.js` in the browser, so they apply only when `connection_mode.web` is `device`. Each is an object keyed by `web`. Where the dashboard has a default, the SDK applies the same value when the key is unset.

#### `track_categorized_pages` — object

Also send a non-interaction event for each `page` call that has a category.

- `web` — boolean. Unset means `true`, the dashboard's default.

#### `track_named_pages` — object

Also send a non-interaction event for each `page` call that has a name.

- `web` — boolean. Unset means `true`.

#### `use_rich_event_names` — object

Name the events that `track_categorized_pages` and `track_named_pages` send `Viewed <category> Page` and `Viewed <name> Page` — for example `Viewed Signup Page`.

- `web` — boolean.

#### `sample_rate` — object

Percentage of users to track.

- `web` — a percentage written as a string: `"50"`, not `50`. Rudder CLI checks only that it's at most 100 characters, or a template. Unset means `"100"`.

```yaml
sample_rate:
  web: "50"
```

#### `site_speed_sample_rate` — object

Percentage of users whose site speed data is collected.

- `web` — a percentage written as a string, on the same terms as `sample_rate`. Unset means `"1"`.

#### `domain` — object

Domain the `_ga` cookie is set on. The dashboard calls it **Cookie Domain Name**.

- `web` — string, at most 100 characters, or a template. Unset means `auto`.

#### `optimize` — object

Google Optimize container ID. With it set, the SDK loads the Optimize plugin.

- `web` — string, at most 100 characters, or a template.

#### `set_all_mapped_props` — object

Set the mapped custom dimensions and metrics on the page, so every later event on that page carries them. When `false`, they're sent only with the event they came from. The dashboard calls it **Set Custom Dimensions & Metrics to the Page**.

- `web` — boolean. Unset means `true`.

#### `reset_custom_dimensions_on_page` — object

Properties whose custom dimensions are cleared on each `page` call, before new values are set. Each name must also appear as a `from` in `dimensions`.

- `web` — array of property names, each at most 100 characters or a template.

```yaml
reset_custom_dimensions_on_page:
  web:
    - plan
```

#### `named_tracker` — object

Send events through a tracker named `rudderGATracker` instead of the default tracker.

- `web` — boolean.

#### `use_google_amp_client_id` — object

Use the AMP client ID, so users are recognized across AMP and non-AMP pages.

- `web` — boolean.

### Event filtering

> [!WARNING]
> Client-side event filtering is applied by the RudderStack SDK, so it affects only web sources connected in `device` mode. Events sent in `cloud` mode reach Google Analytics unfiltered, whatever this block says.

#### `event_filtering` — object

Restricts which `track` events the SDK passes to Google Analytics, by event name.

- `whitelist` — array of event names to allow; every other `track` event is dropped.
- `blacklist` — array of event names to drop; every other `track` event is allowed.
- The two are mutually exclusive, and Rudder CLI enforces it — setting both fails validation.
- Each name is at most 100 characters, or a `{{ path || fallback }}` template.

### Per-source keys

Both keys are objects keyed by the local source type — the tokens listed under [Source types](#source-types). A key naming a source type this destination doesn't support fails validation.

#### `connection_mode` — object

Maps each source type you connect to the mode its events reach Google Analytics in, using the modes in [Source types](#source-types).

- An entry is required for each source type you connect — see [Connect a source](#connect-a-source).
- A mode the source type doesn't support on this destination fails validation — for example `device` for `ios`.

```yaml
connection_mode:
  web: device
  cloud: cloud
```

#### `consent_management` — object

Consent provider configuration per source type. The entry shape, accepted providers, and the rules on `resolution_strategy` and `consents` are shared across all destination types — see [Consent management](../common/README.md).

## Source types

Google Analytics accepts events from these source types in the mentioned connection modes:

| Source type | Connection mode |
| :-----| :-----|
| `android` | `cloud` |
| `android_kotlin` | `cloud` |
| `ios` | `cloud` |
| `ios_swift` | `cloud` |
| `web` | `cloud`, `device` |
| `unity` | `cloud` |
| `cloud` | `cloud` |
| `react_native` | `cloud` |
| `flutter` | `cloud` |
| `cordova` | `cloud` |

Only `web` offers `device` mode, which loads `analytics.js` in the browser.

The dashboard also accepts `warehouse` for Google Analytics, but this definition doesn't, so Rudder CLI can't connect a Reverse ETL source to it.

## Connect a source

An event stream connection to this destination is checked against two rules at `validate` time.

**The source's type must be supported.** A source's type resolves to one of the tokens above before the check — a JavaScript source resolves to `web`, and webhook and server-side SDK sources resolve to `cloud`. Every event stream source type is supported here, so only a Reverse ETL connection, whose source resolves to `warehouse`, fails it:

```text
destination 'google-analytics-prod' (type 'ga') does not accept rETL sources: source type 'warehouse' is not among supported source types: android, android_kotlin, ...
```

**The destination config must carry a `connection_mode` entry for that source type.** This lives on the destination spec, not on the connection spec. Without it:

```text
destination 'google-analytics-prod' config has no 'connection_mode' entry for source type 'web'
```

Google Analytics needs no additional config keys to connect a source of any type, in any mode.

## Secrets

Google Analytics has no secret keys. Rudder CLI treats none of its keys as a secret, and the dashboard masks none, so every value is written to YAML in plain text and `rudder-cli import` brings every value back.

Note that:

- In web device mode the tracking ID is embedded in the page's JavaScript anyway.
- `rudder_delete_account_id` only identifies the Google account connected in the dashboard. The account's credentials stay in RudderStack.
