# Google Ads (`googleads`)

Google Ads is a web device mode destination. RudderStack's JavaScript SDK loads the Google tag (`gtag.js`) in the browser and sends page loads, clicks, and conversions to Google Ads from there.

In a Google Ads destination spec:

- `type: googleads`
- `definition_version: 1`

## Sample configuration

```yaml
version: rudder/v1
kind: destination
metadata:
  name: google-ads-prod
spec:
  id: google-ads-prod
  display_name: Google Ads Production
  type: googleads
  definition_version: 1
  enabled: true
  config:
    conversion_id: "{{ .GOOGLE_ADS_CONVERSION_ID }}"
    v2: true
    allow_identify: false
    event_mapping_from_config:
      - from: Signed Up
        to: Signup

    track_conversions: true
    enable_conversion_events_filtering: true
    events_to_track_conversions:
      - Order Completed
    track_dynamic_remarketing: false

    page_load_conversions:
      - label: AbC-D_efG-h12_34-567
        name: Pricing
    default_page_conversion: XyZ-a_BcD-e98_76-543
    click_event_conversions:
      - label: QrS-t_UvW-x45_67-890
        name: Order Completed

    send_page_view: true
    conversion_linker: true
    disable_ad_personalization: false
    enable_conversion_label: false
    allow_enhanced_conversions: false

    connection_mode:
      web: device
    consent_management:
      web:
        - provider: oneTrust
          consents:
            - marketing
```

The above example sends only `Order Completed` as a conversion event and leaves dynamic remarketing off, so it omits the remarketing event list — see [Conversion and remarketing tracking](#conversion-and-remarketing-tracking).

## Config keys

`config` accepts only the keys listed below. The [shared config key rules](../README.md#config-key-rules) cover unknown keys, defaults, and immutability.

> [!NOTE]
> Google Ads runs only in web device mode, and receives `identify`, `track`, and `page` calls from the SDK. Every key on this page configures what the Google tag does in the browser.

### Connection

#### `conversion_id` — string, required, secret

Your Google Ads conversion ID, which identifies the account the Google tag reports to.

- Must start with `AW-`, followed by at most 100 characters.
- A `{{ path || fallback }}` template is accepted in place of a literal.

#### `sdk_base_url` — string

Full URL that loads the Google tag script. RudderStack appends `?id=<conversion_id>` to it as-is, so it must resolve to `gtag.js`. When omitted, `https://www.googletagmanager.com/gtag/js` is used.

- Must be a domain URL, at most 500 characters. The scheme is optional.
- A `{{ path || fallback }}` template is accepted in place of a literal.

To serve the tag from your own domain, set up Google tag gateway for advertisers and enter the gateway URL that serves `gtag.js`.

#### `v2` — boolean, default `true`

Follow the Ecommerce Event Spec when sending `track` events. This is what the dashboard calls **Ecommerce event spec support for track events**.

#### `allow_identify` — boolean, default `false`

Send `identify` calls to Google Ads, where they define the user data for enhanced conversions.

#### `event_mapping_from_config` — array of objects

Maps RudderStack event names to standard Google Ads events.

- `from` — RudderStack event name. At most 100 characters, or a `{{ path || fallback }}` template.
- `to` — one of `Lead`, `PageVisit`, `ViewCategory`, `Signup`, `WatchVideo`, `Checkout`, `Search`, `AddToCart`, or `purchase`. Note the lowercase `purchase`. Templates aren't accepted here.

```yaml
event_mapping_from_config:
  - from: Signed Up
    to: Signup
```

### Conversion and remarketing tracking

Each tracking switch gates a filtering toggle, which gates an event list. Rudder CLI accepts every key regardless of the switches above it, so a list whose toggle is off is stored and ignored.

#### `track_conversions` — boolean, default `true`

Send conversion events to Google Ads.

#### `enable_conversion_events_filtering` — boolean, default `false`

Treat only the events in `events_to_track_conversions` as conversions. When `false`, every event is a conversion event.

- Applies when `track_conversions` is `true`. Leave it unset otherwise.

#### `events_to_track_conversions` — array of strings

Event names to send as conversion events.

- Applies when `track_conversions` and `enable_conversion_events_filtering` are both `true`. Leave it unset otherwise.
- Each name is at most 100 characters, or a `{{ path || fallback }}` template.

#### `track_dynamic_remarketing` — boolean, default `false`

Send dynamic remarketing events.

#### `enable_dynamic_remarketing_events_filtering` — boolean, default `false`

Treat only the events in `events_to_track_dynamic_remarketing` as remarketing events. When `false`, every event is a remarketing event.

- Applies when `track_dynamic_remarketing` is `true`. Leave it unset otherwise.

#### `events_to_track_dynamic_remarketing` — array of strings

Event names to send as dynamic remarketing events.

- Applies when `track_dynamic_remarketing` and `enable_dynamic_remarketing_events_filtering` are both `true`. Leave it unset otherwise.
- Each name is at most 100 characters, or a `{{ path || fallback }}` template.

#### `dynamic_remarketing` — object, internal

Per-source dynamic remarketing flag, keyed by `web`. Remarketing is controlled by `track_dynamic_remarketing` instead.

- `web` — boolean.

Leave it unset on a new destination. An imported spec may carry it from older configurations.

### Page and click conversions

A conversion label identifies the conversion action in Google Ads.

#### `page_load_conversions` — array of objects

Conversions fired when a named `page` event loads.

- `label` — the conversion label from Google Ads.
- `name` — name of the `page` event that fires the conversion.
- Each is at most 100 characters, or a `{{ path || fallback }}` template.

```yaml
page_load_conversions:
  - label: AbC-D_efG-h12_34-567
    name: Pricing
```

#### `default_page_conversion` — string

Conversion label used for `page` events that don't match an entry in `page_load_conversions`.

- At most 100 characters, or a `{{ path || fallback }}` template.

#### `click_event_conversions` — array of objects

Conversions fired by named `track` events.

- `label` — the conversion label from Google Ads.
- `name` — name of the `track` event that fires the conversion.
- Each is at most 100 characters, or a `{{ path || fallback }}` template.

### Tag behavior

#### `send_page_view` — boolean, default `true`

Send `page` events to Google Ads automatically.

#### `conversion_linker` — boolean, default `true`

Let the Google tag set first-party cookies on your domain for conversion measurement. Turning it off can make conversion measurement less accurate.

#### `disable_ad_personalization` — boolean, default `false`

Disable ad personalization for the events this tag sends.

#### `enable_conversion_label` — boolean, default `false`

Label every conversion event `conversion`. When `false`, each conversion keeps its event name as the label.

#### `allow_enhanced_conversions` — boolean, default `false`

Send enhanced conversions programmatically. Pair it with `allow_identify` to supply the user data.

### Event filtering

#### `event_filtering` — object

Restricts which events the SDK passes to Google Ads, by event name.

- `whitelist` — array of event names to allow; every other `track` event is dropped.
- `blacklist` — array of event names to drop; every other `track` event is allowed.
- The two are mutually exclusive, and Rudder CLI enforces it — setting both fails validation.
- Each name is at most 100 characters, or a `{{ path || fallback }}` template.

This is separate from `events_to_track_conversions`: filtering decides whether an event reaches Google Ads at all, and the conversion list decides which of those count as conversions.

### Per-source keys

Both keys are objects keyed by the local source type — the tokens listed under [Source types](#source-types). A key naming a source type this destination doesn't support fails validation.

#### `connection_mode` — object

Maps each source type you connect to the mode its events reach Google Ads in. Google Ads accepts only `web: device`.

- An entry is required for each source type you connect — see [Connect a source](#connect-a-source).

```yaml
connection_mode:
  web: device
```

#### `consent_management` — object

Consent provider configuration per source type. The entry shape, accepted providers, and the rules on `resolution_strategy` and `consents` are shared across all destination types — see [Consent management](../common/README.md).

## Source types

Google Ads accepts events from these source types in the mentioned connection modes:

| Source type | Connection mode |
| :-----| :-----|
| `web` | `device` |

Google Ads accepts only web sources, and only in `device` mode — the SDK loads the Google tag in the browser, and no events pass through RudderStack's servers.

## Connect a source

An event stream connection to this destination is checked against two rules at `validate` time.

**The source's type must be supported.** A source's type resolves to one of the tokens above before the check — a JavaScript source resolves to `web`, and webhook and server-side SDK sources resolve to `cloud`. Any source other than a JavaScript source reports:

```text
destination 'google-ads-prod' (type 'googleads') does not support source 'my-source':
source type 'cloud' is not among supported source types: web
```

**The destination config must carry a `connection_mode` entry for that source type.** This lives on the destination spec, not on the connection spec. Without it:

```text
destination 'google-ads-prod' config has no 'connection_mode' entry for source type 'web'
```

Google Ads needs no additional config keys to connect a web source.

## Secrets

`conversion_id` is the only secret key. Write it as a `{{ .VAR }}` reference and supply the value at apply time:

```yaml
config:
  conversion_id: "{{ .GOOGLE_ADS_CONVERSION_ID }}"
```

```bash
export RUDDER_GOOGLE_ADS_CONVERSION_ID="AW-123456789"
rudder-cli apply

# or
rudder-cli apply --var-file secrets.vars.yaml
```

Note that:

- In device mode the conversion ID is embedded in the page's JavaScript, so treating it as a secret keeps it out of your YAML but doesn't hide it from the browser.
- The YAML that `rudder-cli import` writes may or may not include secret keys. Before you apply, make sure every secret key your configuration needs is present and populated through variable substitution.
