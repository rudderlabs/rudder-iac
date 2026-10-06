# Facebook Conversions (`facebook_conversions`)

Facebook Conversions sends events to Meta's Conversions API from RudderStack's servers, so conversions reach Facebook without depending on the browser pixel.

In a Facebook Conversions destination spec:

- `type: facebook_conversions`
- `definition_version: 1`

## Sample configuration

```yaml
version: rudder/v1
kind: destination
metadata:
  name: fb-conversions-prod
spec:
  id: fb-conversions-prod
  display_name: Facebook Conversions Production
  type: facebook_conversions
  definition_version: 1
  enabled: true
  config:
    dataset_id: "{{ .FB_DATASET_ID }}"
    access_token: "{{ .FB_ACCESS_TOKEN }}"

    action_source: website
    events_to_events:
      - from: Order Completed
        to: Purchase
      - from: Product Added
        to: AddToCart

    test_destination: false
    limited_data_usage: false
    remove_external_id: false
    blacklist_pii_properties:
      - property: email
        hash: true
    whitelist_pii_properties:
      - property: city

    connection_mode:
      web: cloud
      cloud: cloud
    consent_management:
      web:
        - provider: oneTrust
          consents:
            - marketing
```

The above example hashes `email` instead of dropping it, and omits `test_event_code` because `test_destination` is `false` — see [Testing](#testing) and [Privacy](#privacy).

## Config keys

`config` accepts only the keys listed below. The [shared config key rules](../README.md#config-key-rules) cover unknown keys, defaults, and immutability.

> [!NOTE]
> Facebook Conversions accepts only `page`, `screen`, and `track` events. It doesn't accept `identify` — user data travels with each event instead.

### Connection

#### `dataset_id` — string, required, secret

ID of the Facebook dataset (formerly pixel) that receives the events, from the snippet on Facebook's dataset creation page.

- At most 100 characters, and must not contain line breaks.
- A `{{ path || fallback }}` template is accepted in place of a literal.

#### `access_token` — string, required, secret

Business access token from your Facebook Business account.

- At most 500 characters, and must not contain line breaks.
- A `{{ path || fallback }}` template is accepted in place of a literal.

Supply it as a `{{ .VAR }}` reference rather than a literal — see [Secrets](#secrets).

### Event settings

#### `action_source` — string, default `website`

Fallback `action_source` sent to Facebook when an event's properties don't carry one.

- One of `website`, `email`, `app`, `phone_call`, `chat`, `physical_store`, `system_generated`, or `other`.

#### `events_to_events` — array of objects

Maps RudderStack event names to Facebook standard events. Events without a mapping follow the default standard event mapping.

- `from` — RudderStack event name. At most 100 characters, and must not contain line breaks.
- `to` — one of `ViewContent`, `Search`, `AddToCart`, `AddToWishlist`, `InitiateCheckout`, `AddPaymentInfo`, `Purchase`, `PageView`, `Lead`, `CompleteRegistration`, `Contact`, `CustomizeProduct`, `Donate`, `FindLocation`, `Schedule`, `StartTrial`, `SubmitApplication`, or `Subscribe`.
- The dashboard's dropdown offers only the first 13 of those. Rudder CLI accepts all 18, matching what the API accepts.
- Both fields accept a `{{ path || fallback }}` template in place of a literal.

```yaml
events_to_events:
  - from: Order Completed
    to: Purchase
```

### Testing

> [!WARNING]
> The dashboard asks for `test_event_code` when `test_destination` is on. Rudder CLI doesn't enforce that pairing, so a spec with `test_destination: true` and no code passes `validate`.

#### `test_destination` — boolean, default `false`

Use this destination for testing, so events appear in real time under **Test Events** in your Facebook dashboard.

#### `test_event_code` — string

Test event code from your Facebook dataset's **Test Events** tab.

- Applies when `test_destination` is `true`. Leave it unset otherwise.
- At most 100 characters, and must not contain line breaks.
- A `{{ path || fallback }}` template is accepted in place of a literal.

### Privacy

The PII lists act only on Facebook's standard PII fields — `email`, `firstName`, `lastName`, `gender`, `city`, `country`, `phone`, `state`, `zip`, `birthday`, and their variants. They don't affect other properties, or whether `userId` and `anonymousId` are sent as `external_id`; `remove_external_id` controls that.

#### `limited_data_usage` — boolean, default `false`

Forward the event's `context.dataProcessingOptions` to Facebook as `data_processing_options`, `data_processing_options_country`, and `data_processing_options_state` — Meta's Limited Data Use flags.

#### `remove_external_id` — boolean, default `false`

Stop sending `userId` or `anonymousId` as `external_id`. When `true`, neither is sent. This is what the dashboard calls **Don't send external_id for user**.

#### `blacklist_pii_properties` — array of objects

Standard PII fields to drop — or, with `hash: true`, to SHA-256 hash and send. Every standard PII field is denylisted by default, so an entry here matters mainly to turn on hashing.

- `property` — the PII field name. At most 100 characters, and must not contain line breaks. A template is accepted.
- `hash` — boolean. `true` hashes the field and sends it; `false` or unset drops it.

```yaml
blacklist_pii_properties:
  - property: email
    hash: true
  - property: phone
    hash: true
```

An event whose `integrations.fb_conversions.hashed` is `true` is treated as already hashed and isn't hashed again.

#### `whitelist_pii_properties` — array of objects

Standard PII fields to send as they are, when present in the event's properties.

- `property` — the PII field name. At most 100 characters, and must not contain line breaks. A template is accepted.

```yaml
whitelist_pii_properties:
  - property: city
  - property: country
```

### Per-source keys

Both keys are objects keyed by the local source type — the tokens listed under [Source types](#source-types). A key naming a source type this destination doesn't support fails validation.

#### `connection_mode` — object

Maps each source type you connect to the mode its events reach Facebook in, using the modes in [Source types](#source-types).

- An entry is required for each source type you connect — see [Connect a source](#connect-a-source).

```yaml
connection_mode:
  web: cloud
  cloud: cloud
```

#### `consent_management` — object

Consent provider configuration per source type. The entry shape, accepted providers, and the rules on `resolution_strategy` and `consents` are shared across all destination types — see [Consent management](../common/README.md).

## Source types

Facebook Conversions accepts events from these source types in the mentioned connection modes:

| Source type | Connection mode |
| :-----| :-----|
| `android` | `cloud` |
| `android_kotlin` | `cloud` |
| `ios` | `cloud` |
| `ios_swift` | `cloud` |
| `web` | `cloud` |
| `unity` | `cloud` |
| `cloud` | `cloud` |
| `react_native` | `cloud` |
| `flutter` | `cloud` |
| `cordova` | `cloud` |
| `warehouse` | `cloud` |

Every source type is `cloud` only — events reach Facebook from RudderStack's servers, never in device mode. For browser-side tracking, use [Facebook Pixel](../facebook_pixel/README.md).

`warehouse` is the token a Reverse ETL source resolves to — see [Source types](../README.md#source-types).

> [!NOTE]
> The dashboard additionally offers Facebook Conversions to AMP and Shopify sources. Rudder CLI doesn't manage those connections, so `amp` and `shopify` are invalid here.

## Connect a source

An event stream connection to this destination is checked against two rules at `validate` time.

**The source's type must be supported.** A source's type resolves to one of the tokens above before the check — a JavaScript source resolves to `web`, and webhook and server-side SDK sources resolve to `cloud`. An unsupported type reports:

```text
destination 'fb-conversions-prod' (type 'facebook_conversions') does not support source 'my-source':
source type 'amp' is not among supported source types: android, android_kotlin, ...
```

**The destination config must carry a `connection_mode` entry for that source type.** This lives on the destination spec, not on the connection spec. Without it:

```text
destination 'fb-conversions-prod' config has no 'connection_mode' entry for source type 'web'
```

Facebook Conversions needs no additional config keys to connect a source of any type.

A Reverse ETL connection reaches this destination as source type `warehouse` and is checked against the same rules, so the config needs a `connection_mode.warehouse: cloud` entry for it.

## Secrets

`access_token` and `dataset_id` are the secret keys. Write each as a `{{ .VAR }}` reference and supply the value at apply time:

```yaml
config:
  dataset_id: "{{ .FB_DATASET_ID }}"
  access_token: "{{ .FB_ACCESS_TOKEN }}"
```

```bash
export RUDDER_FB_ACCESS_TOKEN="..."
export RUDDER_FB_DATASET_ID="..."
rudder-cli apply

# or
rudder-cli apply --var-file secrets.vars.yaml
```

Note that:

- The YAML that `rudder-cli import` writes may or may not include secret keys. Before you apply, make sure every secret key your configuration needs is present and populated through variable substitution.
