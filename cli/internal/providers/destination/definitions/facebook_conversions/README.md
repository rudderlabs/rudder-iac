# Facebook Conversions (`facebook_conversions`)

Streaming destination. RudderStack sends events to the Facebook Conversions API
from its own servers. Every supported source type connects in cloud mode — there
is no device-mode variant.

In a destination spec:

- `type: facebook_conversions`
- `definition_version: 1`

## Example

```yaml
version: rudder/v1
kind: destination
metadata:
  name: facebook-conversions
spec:
  id: facebook-conversions
  display_name: Facebook Conversions
  type: facebook_conversions
  definition_version: 1
  enabled: true
  config:
    dataset_id: "1234567890"
    access_token: "{{ .FACEBOOK_CONVERSIONS_ACCESS_TOKEN }}"
    action_source: website
    limited_data_usage: false
    remove_external_id: false

    test_destination: true
    test_event_code: TEST12345

    events_to_events:
      - from: Order Completed
        to: Purchase
      - from: Product Viewed
        to: ViewContent

    blacklist_pii_properties:
      - property: email
        hash: true
      - property: phone
        hash: false
    whitelist_pii_properties:
      - property: plan_name

    connection_mode:
      cloud: cloud
      web: cloud
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

#### `dataset_id` — string, required

Your dataset ID, from the snippet created on the Facebook dataset creation page.
At most 100 characters.

#### `access_token` — string, required, secret

Your business access token from your Facebook business account. At most 500
characters.

Supply it as a `{{ .VAR }}` reference — see [Secrets](#secrets).

### Event delivery

#### `action_source` — string, default `website`

The fallback `action_source` to set when it is not found in the event
properties. One of `website`, `email`, `app`, `phone_call`, `chat`,
`physical_store`, `system_generated` or `other`.

#### `limited_data_usage` — boolean, default `false`

Enable Facebook's Limited Data Usage flag on outgoing events.

#### `remove_external_id` — boolean, default `false`

Send neither `userId` nor `anonymousId` as `external_id`.

#### `events_to_events` — array of objects

Map RudderStack event names to Facebook standard events. Each entry takes:

- `from` — the RudderStack event name, at most 100 characters
- `to` — the Facebook standard event, one of `ViewContent`, `Search`,
  `AddToCart`, `AddToWishlist`, `InitiateCheckout`, `AddPaymentInfo`,
  `Purchase`, `PageView`, `Lead`, `CompleteRegistration`, `Contact`,
  `CustomizeProduct`, `Donate`, `FindLocation`, `Schedule`, `StartTrial`,
  `SubmitApplication` or `Subscribe`

### PII handling

#### `blacklist_pii_properties` — array of objects

PII properties to denylist. Each entry takes:

- `property` — the property name, at most 100 characters
- `hash` — whether to hash the value rather than drop it

#### `whitelist_pii_properties` — array of objects

PII properties to allowlist. Each entry takes a `property` name, at most 100
characters.

### Testing

#### `test_destination` — boolean, default `false`

Mark this destination as being used for testing.

#### `test_event_code` — string

Your test event code, from the Facebook datasets dashboard. **Required when
`test_destination` is `true`.** At most 100 characters.

## Source types

Every supported source type connects in cloud mode only:

`web` · `android` · `android_kotlin` · `ios` · `ios_swift` · `unity` ·
`react_native` · `flutter` · `cordova` · `cloud`

## Per-source keys

Both keys below are objects keyed by the local source type. A key naming a
source type this destination does not support fails validation.

#### `connection_mode` \* — object

Selects the mode per source type. Every supported type accepts `cloud` only, so
each entry's value is `cloud`:

```yaml
connection_mode:
  web: cloud
  cloud: cloud
```

An entry is required for each source type you connect — see
[Connecting a source](#connecting-a-source).

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
destination 'facebook-conversions' (type 'facebook_conversions') does not support
source 'my-source': source type 'amp' is not among supported source types:
android, android_kotlin, ...
```

**The config must carry a `connection_mode` entry for that source type.** This
lives on the destination spec, not the connection spec. Without it:

```
destination 'facebook-conversions' config has no 'connection_mode' entry for
source type 'web'
```

Facebook Conversions requires no additional config keys to connect a source of
any type.

## Secrets

`access_token` is the only secret key. Write it as a `{{ .VAR }}` reference and
supply the value at apply time:

```yaml
access_token: "{{ .FACEBOOK_CONVERSIONS_ACCESS_TOKEN }}"
```

```sh
export RUDDER_FACEBOOK_CONVERSIONS_ACCESS_TOKEN=...
rudder-cli apply

# or
rudder-cli apply --var-file secrets.vars.yaml
```

`rudder-cli import` writes `access_token` back as a `{{ .VAR }}` placeholder
rather than its value, since the API does not return secrets. Fill the
placeholder in before the first apply.
