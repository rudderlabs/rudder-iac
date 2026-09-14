# TikTok Ads (`tiktok_ads`)

Streaming destination. TikTok Ads receives events either from RudderStack's
servers via the Events API (cloud mode) or directly from the TikTok Pixel loaded
on your site (device mode, web only).

In a destination spec:

- `type: tiktok_ads`
- `definition_version: 1`

## Modes

The mode a source uses is set per source type in `config.connection_mode`. Only
`web` accepts `device`; every other source type is cloud only — see
[Source types](#source-types).

Each key below is badged with where it applies:

- `cloud` — events sent from RudderStack's servers
- `device` — events sent by the TikTok Pixel

A key is accepted by `validate` whatever your sources use; the badges tell you
where the setting takes effect.

## Example

```yaml
version: rudder/v1
kind: destination
metadata:
  name: tiktok-ads
spec:
  id: tiktok-ads
  display_name: TikTok Ads
  type: tiktok_ads
  definition_version: 1
  enabled: true
  config:
    pixel_code: CJK3E2RC77UAAAABCDEF
    access_token: "{{ .TIKTOK_ADS_ACCESS_TOKEN }}"
    version: v2
    hash_user_properties: true
    send_custom_events: false

    events_to_standard:
      - from: Order Completed
        to: CompletePayment
      - from: Product Viewed
        to: ViewContent

    event_filtering:
      whitelist:
        - Order Completed
        - Product Viewed

    connection_mode:
      web: device
      cloud: cloud
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

#### `pixel_code` — string, required
`cloud` `device` · web

Your TikTok Pixel code. At most 100 characters.

#### `access_token` — string, secret
`cloud`

Your TikTok long-term access token, used by the Events API in cloud mode.

Not required by `validate`, since a device-mode-only destination does not need
it — but cloud-mode delivery fails without it. Supply it as a `{{ .VAR }}`
reference; see [Secrets](#secrets).

### Event delivery

#### `version` — string, default `v2`
`cloud`

The TikTok Events API version to send under. One of `v2` or `v1`.

#### `hash_user_properties` — boolean, default `true`
`cloud`

Hash contextual user properties with SHA-256 before sending.

#### `send_custom_events` — boolean, default `false`
`cloud` `device` · web

Send events that do not map to a TikTok standard event, rather than dropping
them.

#### `events_to_standard` — array of objects
`cloud` `device` · web

Map RudderStack event names to TikTok standard events. Each entry takes:

- `from` — the RudderStack event name, at most 100 characters
- `to` — the TikTok standard event, one of `AddPaymentInfo`, `AddToCart`,
  `AddToWishlist`, `ClickButton`, `CompletePayment`, `CompleteRegistration`,
  `Contact`, `Download`, `InitiateCheckout`, `PlaceAnOrder`, `Search`,
  `SubmitForm`, `Subscribe`, `ViewContent`, `CustomizeProduct`, `FindLocation`,
  `Schedule`, `Purchase`, `Lead`, `ApplicationApproval`, `SubmitApplication` or
  `StartTrial`

#### `event_filtering` — object
`device` · web

Client-side event filtering, applied by the TikTok Pixel. Exactly one of the two
lists may be set — declaring both fails validation.

- `whitelist` — event names to allowlist
- `blacklist` — event names to denylist

Each entry is at most 100 characters. Omit the block to send every event.

## Source types

| Source type | Modes |
| --- | --- |
| `web` | cloud, device |
| `android` | cloud |
| `android_kotlin` | cloud |
| `ios` | cloud |
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

Selects the mode per source type. Values are constrained to the modes that
source type supports, so `device` is rejected for every type except `web`:

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
destination 'tiktok-ads' (type 'tiktok_ads') does not support source 'my-source':
source type 'amp' is not among supported source types: android, android_kotlin, ...
```

**The config must carry a `connection_mode` entry for that source type.** This
lives on the destination spec, not the connection spec. Without it:

```
destination 'tiktok-ads' config has no 'connection_mode' entry for source type 'web'
```

TikTok Ads requires no additional config keys to connect a source of any type.

## Secrets

`access_token` is the only secret key. Write it as a `{{ .VAR }}` reference and
supply the value at apply time:

```yaml
access_token: "{{ .TIKTOK_ADS_ACCESS_TOKEN }}"
```

```sh
export RUDDER_TIKTOK_ADS_ACCESS_TOKEN=...
rudder-cli apply

# or
rudder-cli apply --var-file secrets.vars.yaml
```

`rudder-cli import` writes `access_token` back as a `{{ .VAR }}` placeholder
rather than its value, since the API does not return secrets. Fill the
placeholder in before the first apply.
