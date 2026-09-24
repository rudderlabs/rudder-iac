# Google Ads (`googleads`)

Streaming destination. Google Ads receives events from the global site tag
(`gtag.js`) loaded on your site. It is the one destination here that supports a
single source type in a single mode: **`web` in `device` mode only**. There is no
cloud-mode variant and no mobile support, so none of the keys below carry a mode
badge.

In a destination spec:

- `type: googleads`
- `definition_version: 1`

## Example

```yaml
version: rudder/v1
kind: destination
metadata:
  name: google-ads
spec:
  id: google-ads
  display_name: Google Ads
  type: googleads
  definition_version: 1
  enabled: true
  config:
    conversion_id: AW-123456789
    v2: true
    conversion_linker: true
    send_page_view: true
    disable_ad_personalization: false
    allow_identify: false
    allow_enhanced_conversions: false

    default_page_conversion: page_view_label
    page_load_conversions:
      - name: Home Page
        label: AbCdEfGhIj
    click_event_conversions:
      - name: Order Completed
        label: KlMnOpQrSt

    track_conversions: true
    enable_conversion_label: true
    enable_conversion_events_filtering: true
    events_to_track_conversions:
      - Order Completed
      - Signup

    track_dynamic_remarketing: true
    enable_dynamic_remarketing_events_filtering: true
    events_to_track_dynamic_remarketing:
      - Product Viewed
    dynamic_remarketing:
      web: true

    event_mapping_from_config:
      - from: Order Completed
        to: purchase
      - from: Product List Viewed
        to: ViewCategory

    event_filtering:
      whitelist:
        - Order Completed
        - Product Viewed

    connection_mode:
      web: device
    consent_management:
      web:
        - provider: oneTrust
          consents:
            - marketing
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

#### `conversion_id` — string, required

Your Google Ads conversion ID. Must start with `AW-`, and at most 103 characters
in total.

#### `v2` \* — boolean, default `true`

Use the version 2 integration behaviour.

#### `sdk_base_url` \* — string

Load `gtag.js` from an alternative domain rather than Google's own. Must be a
domain URL, at most 500 characters. Leave unset to use the default.

### Page and tag behaviour

#### `conversion_linker` — boolean, default `true`

Let the global site tag set first-party cookies on your domain. Enabled by
default; disable it if you do not want `gtag.js` setting cookies.

#### `send_page_view` — boolean, default `true`

Have Google Ads automatically send your `page` events.

#### `disable_ad_personalization` — boolean, default `false`

Programmatically disable ad personalization.

#### `allow_identify` \* — boolean, default `false`

Send `identify` calls to Google Ads in addition to `page` and `track`.

#### `allow_enhanced_conversions` \* — boolean, default `false`

Send hashed first-party customer data alongside conversions, so Google can match
conversions more accurately.

### Conversions

#### `default_page_conversion` — string

The conversion label used for page conversions when no specific one matches. At
most 100 characters.

#### `page_load_conversions` — array of objects

Page-load conversions, configurable for multiple pages. Each entry takes a
`name` and a `label`, each at most 100 characters.

#### `click_event_conversions` — array of objects

Conversions fired from `track` calls. Each entry takes a `name` and a `label`,
each at most 100 characters.

#### `track_conversions` \* — boolean, default `true`

Track conversion events.

#### `enable_conversion_label` \* — boolean, default `false`

Send a conversion label with each conversion event.

#### `enable_conversion_events_filtering` \* — boolean, default `false`

Restrict conversion tracking to the events listed in
`events_to_track_conversions` rather than tracking all of them.

#### `events_to_track_conversions` \* — string array

Event names to track as conversions, applied when
`enable_conversion_events_filtering` is `true`. Each entry at most 100
characters.

### Dynamic remarketing

#### `dynamic_remarketing` — object

Enable Google Ads' Dynamic Remarketing feature for event tracking. Object with a
single boolean `web` key.

#### `track_dynamic_remarketing` \* — boolean, default `false`

Track dynamic remarketing events.

#### `enable_dynamic_remarketing_events_filtering` \* — boolean, default `false`

Restrict dynamic remarketing to the events listed in
`events_to_track_dynamic_remarketing` rather than tracking all of them.

#### `events_to_track_dynamic_remarketing` \* — string array

Event names to track for dynamic remarketing, applied when
`enable_dynamic_remarketing_events_filtering` is `true`. Each entry at most 100
characters.

### Event mapping and filtering

#### `event_mapping_from_config` \* — array of objects

Map RudderStack event names to Google Ads event names. Each entry takes:

- `from` — the RudderStack event name, at most 100 characters
- `to` — the Google Ads event, one of `Lead`, `PageVisit`, `ViewCategory`,
  `Signup`, `WatchVideo`, `Checkout`, `Search`, `AddToCart` or `purchase`

#### `event_filtering` — object

Determine which events are blocked or allowed to flow through to Google Ads.
Exactly one of the two lists may be set — declaring both fails validation.

- `whitelist` — event names to allowlist
- `blacklist` — event names to denylist

Each entry is at most 100 characters. Omit the block to send every event.

## Source types

Google Ads supports one source type in one mode:

| Source type | Modes |
| --- | --- |
| `web` | device |

A connection from any other source type fails validation.

## Per-source keys

Both keys below are objects keyed by the local source type. A key naming a
source type this destination does not support fails validation — which for this
destination means anything other than `web`.

#### `connection_mode` \* — object

Selects the mode per source type. `web` supports `device` only, so that is the
only accepted entry:

```yaml
connection_mode:
  web: device
```

#### `consent_management` — object

Specify consent configuration data for multiple providers, per source type. The
entry shape, accepted providers, and the rules on `resolution_strategy` and
`consents` are shared across all destinations and documented in
[../common/README.md](../common/README.md).

## Connecting a source

An event stream connection to this destination is checked against two rules at
`validate` time.

**The source's type must be supported.** Since `web` is the only supported type,
connecting anything else — including a server-side or webhook source, which
resolves to `cloud` — reports:

```
destination 'google-ads' (type 'googleads') does not support source 'my-source':
source type 'cloud' is not among supported source types: web
```

**The config must carry a `connection_mode` entry for that source type.** This
lives on the destination spec, not the connection spec. Without it:

```
destination 'google-ads' config has no 'connection_mode' entry for source type 'web'
```

Google Ads requires no additional config keys to connect a source.

## Secrets

Google Ads registers no secret keys. Its conversion ID and labels are embedded in
the page by the global site tag, so they are not treated as sensitive and are
returned in full by the API.
