# Customer.io (`customerio`)

Streaming destination. Customer.io receives events either from RudderStack's
servers (cloud mode) or directly from the Customer.io SDK in your app (device
mode, on web, Android and iOS).

In a destination spec:

- `type: customerio`
- `definition_version: 1`

## Modes

The mode a source uses is set per source type in `config.connection_mode`. Web,
`android` and `ios` accept `device`; every other source type is cloud only — see
[Source types](#source-types).

Each key below is badged with where it applies:

- `cloud` — events sent from RudderStack's servers
- `device` — events sent by the Customer.io SDK; supported platforms follow the
  badge where support is limited to some of them

A key is accepted by `validate` whatever your sources use; the badges tell you
where the setting takes effect.

## Example

```yaml
version: rudder/v1
kind: destination
metadata:
  name: customerio
spec:
  id: customerio
  display_name: Customer.io
  type: customerio
  definition_version: 1
  enabled: true
  config:
    site_id: abc123def456
    api_key: "{{ .CUSTOMERIO_API_KEY }}"
    datacenter: US

    api_version: v2
    user_id_identifier_type: email
    device_token_event_name: Device Token Registered

    event_filtering:
      whitelist:
        - Order Completed
        - Product Viewed

    # Device mode
    send_page_name_in_sdk:
      web: true
    data_use_in_app:
      web: true
    auto_track_device_attributes:
      android: true
      ios: true
    background_queue_min_number_of_tasks:
      android: "10"
    background_queue_seconds_delay:
      android: "30"

    connection_mode:
      web: device
      android: device
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

#### `site_id` — string, required
`cloud` `device`

Your Customer.io site ID. At most 100 characters.

#### `api_key` — string, required, secret
`cloud` `device`

Your Customer.io API key. At most 100 characters.

Supply it as a `{{ .VAR }}` reference — see [Secrets](#secrets).

#### `datacenter` — string, required
`cloud` `device`

Your Customer.io data center. One of `US` or `EU`.

### Cloud-mode delivery

#### `api_version` — string, default `v1`
`cloud`

The Customer.io API version used for cloud-mode delivery. `v1` uses the existing
per-endpoint behaviour; `v2` uses the unified `/v2/batch` API. Does not affect
device-mode SDK delivery.

#### `user_id_identifier_type` — string
`cloud`

Which Customer.io identifier receives the RudderStack `userId` in cloud mode. One
of `id`, `email`, `phone` or `cio_id`. **Required when `api_version` is `v2`.**
Does not affect device-mode SDK delivery.

#### `device_token_event_name` — string
`cloud` `device`

Name of the event fired immediately after setting the device token. At most 100
characters.

#### `event_filtering` — object
`device` · web

Determine which events are allowed to flow through to Customer.io or blocked, in
device mode connections. Client-side filtering is applied by the SDK, so it has
no effect on a cloud mode connection. Exactly one of the two lists may be set —
declaring both fails validation.

- `whitelist` — event names to allowlist
- `blacklist` — event names to denylist

Each entry is at most 100 characters. Omit the block to send every event.

## Device mode only

The keys below configure the Customer.io SDK and apply only in device mode. Each
is an object keyed by platform, using the local (snake_case) source type as the
key, and each supports the platforms listed on its badge.

`validate` accepts these keys whatever sources the project connects, so you can
declare a platform ahead of connecting a source of that type.

### Web

#### `send_page_name_in_sdk` — object
`device` · web

Send the page name from the SDK.

#### `data_use_in_app` — object
`device` · web

Send in-app messages to your website.

### Mobile

#### `auto_track_device_attributes` — object
`device` · android, ios

Automatically track device attributes from the SDK.

#### `background_queue_min_number_of_tasks` — object
`device` · android

Minimum number of queued tasks before the SDK flushes its background queue.
Written as a digit string, not a number. At most 100 characters.

#### `background_queue_seconds_delay` — object
`device` · android

Delay in seconds before the SDK flushes its background queue. Written as a digit
string, not a number. At most 100 characters.

## Source types

| Source type | Modes |
| --- | --- |
| `web` | cloud, device |
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

#### `connection_mode` — object

Selects the mode per source type. Values are constrained to the modes that
source type supports, so `device` is rejected for every type except `web`,
`android` and `ios`:

```yaml
connection_mode:
  web: device
  android: device
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
destination 'customerio' (type 'customerio') does not support source 'my-source':
source type 'amp' is not among supported source types: android, android_kotlin, ...
```

**The config must carry a `connection_mode` entry for that source type.** This
lives on the destination spec, not the connection spec. Without it:

```
destination 'customerio' config has no 'connection_mode' entry for source type 'web'
```

Customer.io requires no additional config keys to connect a source of any type.

## Secrets

`api_key` is the only secret key. Write it as a `{{ .VAR }}` reference and
supply the value at apply time:

```yaml
api_key: "{{ .CUSTOMERIO_API_KEY }}"
```

```sh
export RUDDER_CUSTOMERIO_API_KEY=...
rudder-cli apply

# or
rudder-cli apply --var-file secrets.vars.yaml
```

`rudder-cli import` writes `api_key` back as a `{{ .VAR }}` placeholder rather
than its value, since the API does not return secrets. Fill the placeholder in
before the first apply.
