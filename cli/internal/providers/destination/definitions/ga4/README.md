# Google Analytics 4 (`ga4`)

Streaming destination. GA4 receives events either from RudderStack's servers via
the Measurement Protocol (cloud mode), directly from `gtag.js` or the Firebase
SDK in your app (device mode), or both at once on web (hybrid mode).

In a destination spec:

- `type: ga4`
- `definition_version: 1`

## Modes

The mode a source uses is set per source type in `config.connection_mode`. Web
accepts all three modes, `android` and `ios` accept cloud and device, and the
rest are cloud only — see [Source types](#source-types).

Each key below is badged with where it applies:

- `cloud` — events sent from RudderStack's servers
- `device` — events sent by `gtag.js` or the Firebase SDK; supported platforms
  follow the badge where support is limited to some of them

A key is accepted by `validate` whatever your sources use; the badges tell you
where the setting takes effect.

## Client type

`client_type` selects which GA4 client this destination talks to, and decides
which identifier is required:

| `client_type` | Required | Used for |
| --- | --- | --- |
| `gtag` | `measurement_id` | web sources |
| `firebase` | `firebase_app_id` | mobile sources |

## Example

```yaml
version: rudder/v1
kind: destination
metadata:
  name: ga4
spec:
  id: ga4
  display_name: Google Analytics 4
  type: ga4
  definition_version: 1
  enabled: true
  config:
    api_secret: "{{ .GA4_API_SECRET }}"
    client_type: gtag
    measurement_id: G-ABC1234567
    debug_mode: false

    sdk_base_url: "https://gtm.example.com"
    server_container_url: "https://sst.example.com"

    pii_properties_to_ignore:
      - pii_property: email
      - pii_property: phone

    event_filtering:
      whitelist:
        - Order Completed
        - Product Viewed

    # Device mode (web)
    capture_page_view:
      web: rs
    extend_page_view_params:
      web: true
    debug_view:
      web: false
    override_client_and_session_ids:
      web: true

    connection_mode:
      web: device
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

#### `api_secret` — string, required, secret
`cloud` `device`

The API secret generated through the Google Analytics dashboard. At most 100
characters.

Supply it as a `{{ .VAR }}` reference — see [Secrets](#secrets).

#### `client_type` — string, required
`cloud` `device`

The GA4 client to use. One of `gtag` or `firebase` — see
[Client type](#client-type).

#### `measurement_id` — string
`cloud` `device` · web

The identifier for a GA4 data stream. **Required when `client_type` is `gtag`.**
Must start with `G-`, and at most 101 characters in total.

#### `firebase_app_id` — string
`cloud` `device` · android, ios

The identifier for your Firebase app. **Required when `client_type` is
`firebase`.** At most 100 characters.

### Delivery and endpoints

#### `sdk_base_url` — string
`device` · web

Your GA4 custom domain URL, used instead of the default
`https://www.googletagmanager.com`.

Validated as a domain URL only when `client_type` is `gtag` **and** `web` is set
to `device` in `connection_mode` — in any other configuration the value is
accepted as-is.

#### `server_container_url` — string
`device` · web

Your GA4 server-side container URL. Not validated.

#### `debug_mode` — boolean, default `false`
`cloud`

Send events to GA4's validation server rather than reporting them. Validation
responses appear in Live Events, but the events do not reach your reports.

### Events and PII

#### `pii_properties_to_ignore` — array of objects
`cloud`

Filter sensitive PII fields out of events before sending them to GA4. Each entry
takes a `pii_property` name, at most 100 characters.

#### `event_filtering` — object
`cloud` `device`

Determine which events are blocked or allowed to flow through to GA4. Exactly one
of the two lists may be set — declaring both fails validation.

- `whitelist` — event names to allowlist
- `blacklist` — event names to denylist

Each entry is at most 100 characters. Omit the block to send every event.

## Device mode only

The keys below configure the browser-side integration and apply only to a web
source in device or hybrid mode. Each is an object keyed by platform.

`validate` accepts these keys whatever sources the project connects, so you can
declare them ahead of connecting a web source.

#### `capture_page_view` — object, default `{web: rs}`
`device` · web

Which layer sends page view events. `rs` sends them through the RudderStack JS
SDK; `gtag` leaves them to GA4's automatic Enhanced Measurement collection.

#### `extend_page_view_params` — object
`device` · web

Send `url` and `search`, along with any other custom property, on the `page` call
made by the RudderStack SDK.

#### `debug_view` — object
`device` · web

Surface your device-mode events in GA4 DebugView.

#### `override_client_and_session_ids` — object
`device` · web

Override the `gtag` client ID and session ID with RudderStack's, so attribution
stays unified across `page` and `track` events.

## Source types

| Source type | Modes |
| --- | --- |
| `web` | cloud, device, hybrid |
| `android` | cloud, device |
| `ios` | cloud, device |
| `android_kotlin` | cloud |
| `ios_swift` | cloud |
| `unity` | cloud |
| `react_native` | cloud |
| `flutter` | cloud |
| `cordova` | cloud |
| `cloud` | cloud |

## Per-source keys

Both keys below are objects keyed by the local source type. A key naming a
source type this destination does not support fails validation.

#### `connection_mode` \* — object

Selects the mode per source type. Values are constrained to the modes that source
type supports, so `hybrid` is accepted only for `web`, and `device` only for
`web`, `android` and `ios`:

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
destination 'ga4' (type 'ga4') does not support source 'my-source':
source type 'amp' is not among supported source types: android, android_kotlin, ...
```

**The config must carry a `connection_mode` entry for that source type.** This
lives on the destination spec, not the connection spec. Without it:

```
destination 'ga4' config has no 'connection_mode' entry for source type 'web'
```

GA4 requires no additional config keys to connect a source of any type.

## Secrets

`api_secret` is the only secret key. Write it as a `{{ .VAR }}` reference and
supply the value at apply time:

```yaml
api_secret: "{{ .GA4_API_SECRET }}"
```

```sh
export RUDDER_GA4_API_SECRET=...
rudder-cli apply

# or
rudder-cli apply --var-file secrets.vars.yaml
```

`rudder-cli import` writes `api_secret` back as a `{{ .VAR }}` placeholder rather
than its value, since the API does not return secrets. Fill the placeholder in
before the first apply.
