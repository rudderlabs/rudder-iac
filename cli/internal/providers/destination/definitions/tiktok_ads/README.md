# TikTok Ads (`tiktok_ads`)

TikTok Ads is an advertising destination. RudderStack sends conversion events to TikTok's Events API from its servers, or loads the TikTok pixel in the browser in device mode.

In a TikTok Ads destination spec:

- `type: tiktok_ads`
- `definition_version: 1`

## Sample configuration

```yaml
version: rudder/v1
kind: destination
metadata:
  name: tiktok-ads-prod
spec:
  id: tiktok-ads-prod
  display_name: TikTok Ads Production
  type: tiktok_ads
  definition_version: 1
  enabled: true
  config:
    pixel_code: "{{ .TIKTOK_PIXEL_CODE }}"
    access_token: "{{ .TIKTOK_ACCESS_TOKEN }}"
    version: v2
    hash_user_properties: true

    events_to_standard:
      - from: Order Completed
        to: CompletePayment
      - from: Product Added
        to: AddToCart
    send_custom_events: false

    event_filtering:
      whitelist:
        - Order Completed
        - Product Added

    connection_mode:
      web: device
      cloud: cloud
    consent_management:
      web:
        - provider: oneTrust
          consents:
            - marketing
```

The above example loads the TikTok pixel on web sources in `device` mode and sends server-side events in `cloud` mode. `event_filtering` affects only the device-mode web traffic — see [Event filtering](#event-filtering).

## Config keys

`config` accepts only the keys listed below. The [shared config key rules](../README.md#config-key-rules) cover unknown keys, defaults, and immutability.

> [!NOTE]
> In `cloud` mode TikTok Ads accepts only `track` events. In `device` mode, web sources also send `identify` and `page`.

### Connection

#### `pixel_code` — string, required, secret

Code of the TikTok pixel that receives the events. Used in both modes.

- At most 100 characters, and must not contain line breaks.
- A `{{ path || fallback }}` template is accepted in place of a literal.

#### `access_token` — string, secret

TikTok long-term access token for the Events API. Cloud mode needs it; device mode doesn't.

- Rudder CLI doesn't require this key, even when a source connects in `cloud` mode. Without it, cloud-mode delivery fails at TikTok.

Supply it as a `{{ .VAR }}` reference rather than a literal — see [Secrets](#secrets).

#### `version` — string, default `v2`

TikTok Events API version for cloud mode.

- `v2` or `v1`. Use `v2` — TikTok announced the sunset of Events API 1.0.
- A `{{ path || fallback }}` template is accepted in place of a literal.

#### `hash_user_properties` — boolean, default `true`

SHA-256 hash contextual user properties such as `external_id`, `email`, and `phone_number` before sending them.

- Applies to `cloud` mode only.

### Event mapping

#### `events_to_standard` — array of objects

Maps RudderStack event names to TikTok standard events. Events without a mapping follow TikTok's default event mapping.

- `from` — RudderStack event name. At most 100 characters, and must not contain line breaks.
- `to` — one of `AddPaymentInfo`, `AddToCart`, `AddToWishlist`, `ClickButton`, `CompletePayment`, `CompleteRegistration`, `Contact`, `Download`, `InitiateCheckout`, `PlaceAnOrder`, `Search`, `SubmitForm`, `Subscribe`, `ViewContent`, `CustomizeProduct`, `FindLocation`, `Schedule`, `Purchase`, `Lead`, `ApplicationApproval`, `SubmitApplication`, or `StartTrial`.
- Both fields accept a `{{ path || fallback }}` template in place of a literal.

```yaml
events_to_standard:
  - from: Order Completed
    to: CompletePayment
```

#### `send_custom_events` — boolean, default `false`

Send events that don't map to a standard event as TikTok custom events. For custom events, only TikTok's standard fields are sent; other fields are dropped.

### Event filtering

> [!WARNING]
> Client-side event filtering is applied by the RudderStack SDK, so it affects only web sources connected in `device` mode. Events sent in `cloud` mode reach TikTok unfiltered, whatever this block says.

#### `event_filtering` — object

Restricts which `track` events the SDK passes to the TikTok pixel, by event name.

- `whitelist` — array of event names to allow; every other `track` event is dropped.
- `blacklist` — array of event names to drop; every other `track` event is allowed.
- The two are mutually exclusive, and Rudder CLI enforces it — setting both fails validation.
- Each name is at most 100 characters, or a `{{ path || fallback }}` template.
- Omit the block entirely to filter nothing.

```yaml
event_filtering:
  blacklist:
    - Product Viewed
```

### Per-source keys

Both keys are objects keyed by the local source type — the tokens listed under [Source types](#source-types). A key naming a source type this destination doesn't support fails validation.

#### `connection_mode` — object

Maps each source type you connect to the mode its events reach TikTok in, using the modes in [Source types](#source-types).

- An entry is required for each source type you connect — see [Connect a source](#connect-a-source).
- A mode the source type doesn't support on this destination fails validation — for example `device` for `android`.

```yaml
connection_mode:
  web: device
  cloud: cloud
```

#### `consent_management` — object

Consent provider configuration per source type. The entry shape, accepted providers, and the rules on `resolution_strategy` and `consents` are shared across all destination types — see [Consent management](../common/README.md).

## Source types

TikTok Ads accepts events from these source types in the mentioned connection modes:

| Source type | Connection mode |
| :-----| :-----|
| `android` | `cloud` |
| `android_kotlin` | `cloud` |
| `ios` | `cloud` |
| `ios_swift` | `cloud` |
| `web` | `cloud`, `device` |
| `unity` | `cloud` |
| `react_native` | `cloud` |
| `flutter` | `cloud` |
| `cordova` | `cloud` |
| `cloud` | `cloud` |
| `warehouse` | `cloud` |

Only `web` offers `device` mode, which loads the TikTok pixel in the browser.

`warehouse` is the token a Reverse ETL source resolves to — see [Source types](../README.md#source-types).

## Connect a source

An event stream connection to this destination is checked against two rules at `validate` time.

**The source's type must be supported.** A source's type resolves to one of the tokens above before the check — a JavaScript source resolves to `web`, and webhook and server-side SDK sources resolve to `cloud`. Every type an event stream or Reverse ETL source can resolve to is supported here, so this check always passes.

**The destination config must carry a `connection_mode` entry for that source type.** This lives on the destination spec, not on the connection spec. Without it:

```text
destination 'tiktok-ads-prod' config has no 'connection_mode' entry for source type 'web'
```

TikTok Ads needs no additional config keys to connect a source of any type, in any mode.

A Reverse ETL connection reaches this destination as source type `warehouse` and is checked against the same rules, so the config needs a `connection_mode.warehouse: cloud` entry for it.

## Secrets

`pixel_code` and `access_token` are the secret keys. Write each as a `{{ .VAR }}` reference — [Secrets](../README.md#secrets) covers supplying the values and what `import` writes:

```yaml
config:
  pixel_code: "{{ .TIKTOK_PIXEL_CODE }}"
  access_token: "{{ .TIKTOK_ACCESS_TOKEN }}"
```

In device mode the pixel code is embedded in the page's JavaScript, so treating it as a secret protects your YAML, not the value itself.
