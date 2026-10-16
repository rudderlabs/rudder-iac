# Facebook Pixel (`facebook_pixel`)

Facebook Pixel is an advertising destination. RudderStack loads the Meta Pixel in the browser for web sources in device mode, and sends events to Meta's Conversions API from its servers in cloud mode.

In a Facebook Pixel destination spec:

- `type: facebook_pixel`
- `definition_version: 1`

## Sample configuration

```yaml
version: rudder/v1
kind: destination
metadata:
  name: fb-pixel-prod
spec:
  id: fb-pixel-prod
  display_name: Facebook Pixel Production
  type: facebook_pixel
  definition_version: 1
  enabled: true
  config:
    pixel_id: "{{ .FB_PIXEL_ID }}"
    access_token: "{{ .FB_ACCESS_TOKEN }}"

    standard_page_call: false
    value_field_identifier: properties.price
    advanced_mapping: true
    use_updated_mapping: true
    events_to_events:
      - from: Order Completed
        to: Purchase

    limited_data_usage: false
    test_destination: false
    remove_external_id: false
    blacklist_pii_properties:
      - property: email
        hash: true

    auto_config:
      web: true
    legacy_conversion_pixel_id:
      - from: Signed Up
        to: "{{ .FB_LEGACY_PIXEL_ID }}"

    connection_mode:
      web: device
      cloud: cloud
    consent_management:
      web:
        - provider: oneTrust
          consents:
            - marketing
```

The above example loads the Pixel on web sources in `device` mode and sends server-side events in `cloud` mode. Because a source connects in `cloud` mode, it needs `access_token` — see [Access token requirements](#access-token-requirements).

## Config keys

`config` accepts only the keys listed below. The [shared config key rules](../README.md#config-key-rules) cover unknown keys, defaults, and immutability.

> [!NOTE]
> In `cloud` mode Facebook Pixel accepts `identify`, `page`, `screen`, and `track` events. In `device` mode, web sources send `page` and `track`.

### Access token requirements

The Conversions API needs `access_token`; the Pixel in the browser doesn't. Rudder CLI checks for it twice:

- **On the destination.** `access_token` is required unless `connection_mode` is set and either maps `web` to `device` or leaves `web` out. A spec with no `connection_mode` at all, or with `web: cloud`, must carry it.
- **On each connection.** Connecting any source in `cloud` mode requires `access_token` — only `web` in `device` mode is exempt. See [Connect a source](#connect-a-source).

In practice: if anything connects in `cloud` mode, set `access_token`.

### Connection

#### `pixel_id` — string, required, secret

ID of your Facebook Pixel, from the snippet on Facebook's Pixel creation page.

- At most 100 characters, and must not contain line breaks.
- A `{{ path || fallback }}` template is accepted in place of a literal.

#### `access_token` — string, required, secret

Business access token from your Facebook Business account, used by the Conversions API in cloud mode.

- Required when a source connects in `cloud` mode, or when `connection_mode` is absent or maps `web` to `cloud`. See [Access token requirements](#access-token-requirements).
- At most 300 characters, and must not contain line breaks.
- A `{{ path || fallback }}` template is accepted in place of a literal.

### Event settings

#### `standard_page_call` — boolean, default `false`

Send a standard `PageView` event for every `page` and `screen` call.

#### `value_field_identifier` — string, default `properties.price`

Event property RudderStack maps to Facebook's `value` field — used for events such as **Product Viewed** and **Product Added**.

- `properties.price` or `properties.value`.

#### `advanced_mapping` — boolean, default `false`

Turn on Facebook's advanced matching, sending user information with Pixel events. This is what the dashboard calls **Enable Advanced Matching**.

#### `events_to_events` — array of objects

Maps RudderStack event names to Facebook standard events.

- `from` — RudderStack event name. At most 100 characters, and must not contain line breaks.
- `to` — one of `ViewContent`, `Search`, `AddToCart`, `AddToWishlist`, `InitiateCheckout`, `AddPaymentInfo`, `Purchase`, `PageView`, `Lead`, `CompleteRegistration`, `Contact`, `CustomizeProduct`, `Donate`, `FindLocation`, `Schedule`, `StartTrial`, `SubmitApplication`, or `Subscribe`.
- The dashboard's dropdown offers only the first 13 of those. Rudder CLI accepts all 18, matching what the API accepts.
- Both fields accept a `{{ path || fallback }}` template in place of a literal.

### Testing and privacy

> [!WARNING]
> The dashboard asks for `test_event_code` when `test_destination` is on. Rudder CLI doesn't enforce that pairing.

#### `test_destination` — boolean, default `false`

Use this destination for testing, so events appear in real time under **Test Events** in your Facebook dashboard.

#### `test_event_code` — string

Test event code from your Facebook dataset's **Test Events** tab.

- Applies when `test_destination` is `true`. Leave it unset otherwise.
- At most 100 characters, and must not contain line breaks.
- A `{{ path || fallback }}` template is accepted in place of a literal.

#### `limited_data_usage` — boolean, default `false`

Forward the event's `context.dataProcessingOptions` to Facebook — Meta's Limited Data Use flags.

#### `remove_external_id` — boolean, default `false`

Stop sending `userId` or `anonymousId` as `external_id`. When `true`, neither is sent.

#### `blacklist_pii_properties` — array of objects

Standard PII fields to drop — or, with `hash: true`, to SHA-256 hash and send. Standard PII fields are denylisted by default, so an entry here matters mainly to turn on hashing.

- `property` — the PII field name. At most 100 characters, and must not contain line breaks. A template is accepted.
- `hash` — boolean. `true` hashes the field and sends it; `false` or unset drops it.

#### `whitelist_pii_properties` — array of objects

Standard PII fields to send as they are, when present in the event's properties.

- `property` — the PII field name. At most 100 characters, and must not contain line breaks. A template is accepted.

### Web device mode

These keys configure the Pixel in the browser, so they apply only when `connection_mode.web` is `device`.

#### `use_updated_mapping` — boolean, default `false`

Map user traits to Facebook's fields instead of sending them unmodified. Turn this on; the old mapping is being deprecated.

#### `auto_config` — object

Let the Pixel send button clicks and page metadata to improve ad delivery — Meta's automatic configuration.

- `web` — boolean. The dashboard defaults it to `true`; Rudder CLI doesn't fill it in.

#### `legacy_conversion_pixel_id` — array of objects

Sends specific events to a legacy conversion Pixel instead of `pixel_id`.

- `from` — RudderStack event name. At most 100 characters, or a template.
- `to` — ID of the legacy conversion Pixel. At most 100 characters, or a template. **Secret** — see [Secrets](#secrets).
- A plain list, not keyed by `web`, though it applies only to web.

```yaml
legacy_conversion_pixel_id:
  - from: Signed Up
    to: "{{ .FB_LEGACY_PIXEL_ID }}"
```

### Event filtering

> [!WARNING]
> Client-side event filtering is applied by the RudderStack SDK, so it affects only web sources connected in `device` mode. Events sent in `cloud` mode aren't filtered by it.

#### `event_filtering` — object

Restricts which `track` events the SDK passes to the Pixel, by event name.

- `whitelist` — array of event names to allow; every other `track` event is dropped.
- `blacklist` — array of event names to drop; every other `track` event is allowed.
- The two are mutually exclusive, and Rudder CLI enforces it — setting both fails validation.
- Each name is at most 100 characters, or a `{{ path || fallback }}` template.

### Per-source keys

Both keys are objects keyed by the local source type — the tokens listed under [Source types](#source-types). A key naming a source type this destination doesn't support fails validation.

#### `connection_mode` — object

Maps each source type you connect to the mode its events reach Facebook in, using the modes in [Source types](#source-types). It also decides whether `access_token` is required.

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

Facebook Pixel accepts events from these source types in the mentioned connection modes:

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
| `warehouse` | `cloud` |

Only `web` offers `device` mode, which loads the Meta Pixel in the browser.

`warehouse` is the token a Reverse ETL source resolves to — see [Source types](../README.md#source-types).

## Connect a source

An event stream connection to this destination is checked against three rules at `validate` time.

**The source's type must be supported.** A source's type resolves to one of the tokens above before the check — a JavaScript source resolves to `web`, and webhook and server-side SDK sources resolve to `cloud`. Every type an event stream or Reverse ETL source can resolve to is supported here, so this check always passes.

**The destination config must carry a `connection_mode` entry for that source type.** This lives on the destination spec, not on the connection spec. Without it:

```text
destination 'fb-pixel-prod' config has no 'connection_mode' entry for source type 'web'
```

**A source connecting in `cloud` mode needs `access_token`.** Every source type requires it in `cloud` mode; only `web` in `device` mode doesn't. Without it:

```text
destination 'fb-pixel-prod' config is missing fields required to connect a 'cloud' source: access_token
```

A Reverse ETL connection reaches this destination as source type `warehouse` and is checked against the same rules, so the config needs a `connection_mode.warehouse: cloud` entry for it, and `access_token`.

## Secrets

Rudder CLI treats three keys as secrets: `pixel_id`, `access_token`, and every `legacy_conversion_pixel_id` entry's `to` value. Write each one you use as a `{{ .VAR }}` reference — [Secrets](../README.md#secrets) covers supplying the values and what `import` writes:

```yaml
config:
  pixel_id: "{{ .FB_PIXEL_ID }}"
  access_token: "{{ .FB_ACCESS_TOKEN }}"
```

In device mode the Pixel IDs are embedded in the page's JavaScript, so masking them protects your YAML, not the values themselves.
