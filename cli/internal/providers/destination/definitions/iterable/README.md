# Iterable (`iterable`)

Streaming destination. Iterable receives events either from RudderStack's servers
(cloud mode) or directly from the Iterable Web SDK loaded on your site (device
mode, web only).

Device mode also brings Iterable's in-app messages, and most of this
destination's config surface exists to position and style them.

In a destination spec:

- `type: iterable`
- `definition_version: 1`

## Modes

The mode a source uses is set per source type in `config.connection_mode`. Only
`web` accepts `device`; every other source type is cloud only — see
[Source types](#source-types).

Each key below is badged with where it applies:

- `cloud` — events sent from RudderStack's servers
- `device` — events sent by the Iterable Web SDK

A key is accepted by `validate` whatever your sources use; the badges tell you
where the setting takes effect.

## Example

```yaml
version: rudder/v1
kind: destination
metadata:
  name: iterable
spec:
  id: iterable
  display_name: Iterable
  type: iterable
  definition_version: 1
  enabled: true
  config:
    api_key: "{{ .ITERABLE_API_KEY }}"
    data_center: USDC
    register_device_or_browser_api_key: "{{ .ITERABLE_JWT_API_KEY }}"

    prefer_user_id: true
    merge_nested_objects: true

    map_to_single_event: true
    track_all_pages: false
    track_categorized_pages: true
    track_named_pages: true

    event_filtering:
      whitelist:
        - Order Completed
        - Product Viewed

    # Device mode (web)
    package_name: com.example.web
    initialisation_identifier:
      web: email
    get_in_app_event_mapping:
      web:
        - Product Viewed
    purchase_event_mapping:
      web:
        - Order Completed
    send_track_for_inapp:
      web: true

    handle_links:
      web: open-all-new-tab
    display_interval:
      web: "30000"
    animation_duration:
      web: "400"
    is_required_to_dismiss_message:
      web: false

    top_offset:
      web: "10%"
    right_offset:
      web: "5%"
    bottom_offset:
      web: "0"
    icon_path:
      web: "/assets/iterable-icon.svg"

    close_button_position:
      web: top-right
    close_button_color:
      web: "#333333"
    close_button_size:
      web: "16px"
    close_button_color_top_offset:
      web: "8px"
    close_button_color_side_offset:
      web: "8px"

    on_open_screen_reader_message:
      web: "In-app message opened"
    on_open_node_to_take_focus:
      web: "#main"

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

#### `api_key` — string, required
`cloud` `device` · web

Your Iterable API key. At most 100 characters.

#### `data_center` \* — string, required
`cloud` `device` · web

The Iterable data centre your project lives in. One of `USDC` or `EUDC`.

#### `register_device_or_browser_api_key` \* — string, secret
`device` · web

API key used to register a device or browser for push and in-app messages. At
most 100 characters.

Supply it as a `{{ .VAR }}` reference — see [Secrets](#secrets).

### Identity and payload

#### `prefer_user_id` \* — boolean, default `true`
`cloud`

Identify users by `userId` in preference to `email` when both are present.

#### `merge_nested_objects` \* — boolean, default `true`
`cloud`

Merge nested objects into the existing user profile rather than replacing them
wholesale.

### Page tracking

#### `map_to_single_event` — boolean, default `true`
`cloud` `device` · web

Map all page calls to a single event name rather than deriving one per page.

#### `track_all_pages` — boolean, default `false`
`cloud` `device` · web

Track every `page` call.

#### `track_categorized_pages` — boolean, default `true`
`cloud` `device` · web

Track `page` calls that carry a category.

#### `track_named_pages` — boolean, default `true`
`cloud` `device` · web

Track `page` calls that carry a name.

#### `event_filtering` \* — object
`device` · web

Filter which events are sent to Iterable in device mode connections. Client-side
filtering is applied by the SDK, so it has no effect on a cloud mode connection.
Exactly one of the two lists may be set — declaring both fails validation.

- `whitelist` — event names to allowlist
- `blacklist` — event names to denylist

Each entry is at most 100 characters. Omit the block to send every event.

## Device mode only

The keys below configure the Iterable Web SDK and apply only to a web source in
device mode. Except `package_name`, each is an object with a single `web` key.

`validate` accepts these keys whatever sources the project connects, so you can
declare them ahead of connecting a web source.

### Setup

#### `package_name` — string
`device` · web

The package name the SDK registers under. **Required when `connection_mode.web`
is `device`** — this is the one key here whose absence fails validation. At most
100 characters.

#### `initialisation_identifier` — object, default `{web: email}`
`device` · web

Which identifier identifies a user across a session. One of `email` or `userId`.

### Event mapping

#### `get_in_app_event_mapping` — object
`device` · web

Event names that trigger a `getInApp` messages fetch. A list of event names under
`web`, each at most 100 characters.

#### `purchase_event_mapping` — object
`device` · web

Event names treated as purchases. A list of event names under `web`, each at most
100 characters.

#### `send_track_for_inapp` — object
`device` · web

Fire a `track` event when a web in-app push is delivered.

### In-app message behaviour

#### `handle_links` — object, default `{web: open-all-new-tab}`
`device` · web

How links inside in-app messages open. One of `open-all-new-tab`,
`open-all-same-tab` or `external-new-tab`.

#### `display_interval` — object
`device` · web

Wait time before the next message is shown, in milliseconds.

#### `animation_duration` — object
`device` · web

Time for messages to animate in and out, in milliseconds.

#### `is_required_to_dismiss_message` — object
`device` · web

Prevent the user dismissing an in-app message by clicking outside it.

### In-app message layout

Each is an object with a single string `web` key. Offsets accept a pixel or
percentage value.

- **`top_offset`** — space between the top of the screen and the message
- **`right_offset`** — space between the right of the screen and the message
- **`bottom_offset`** — space between the bottom of the screen and the message
- **`icon_path`** — custom pathname for the message icon

### Close button

Each is an object with a single string `web` key, except `close_button_position`
which takes `top-right` or `top-left`.

- **`close_button_position`** — corner the button sits in, default
  `{web: top-right}`
- **`close_button_color`** — colour of the close button
- **`close_button_size`** — size of the close button
- **`close_button_color_top_offset`** — space between the button and the
  container top
- **`close_button_color_side_offset`** — space between the button and the
  container side

### Accessibility

#### `on_open_screen_reader_message` — object
`device` · web

Text read out by a screen reader when a message opens.

#### `on_open_node_to_take_focus` — object
`device` · web

Selector for the element that takes focus when a message opens.

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

Setting `web` to `device` also makes `package_name` required.

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
destination 'iterable' (type 'iterable') does not support source 'my-source':
source type 'amp' is not among supported source types: android, android_kotlin, ...
```

**The config must carry a `connection_mode` entry for that source type.** This
lives on the destination spec, not the connection spec. Without it:

```
destination 'iterable' config has no 'connection_mode' entry for source type 'web'
```

Iterable requires no additional config keys to connect a source of any type.

## Secrets

`register_device_or_browser_api_key` is the only secret key. Write it as a
`{{ .VAR }}` reference and supply the value at apply time:

```yaml
register_device_or_browser_api_key: "{{ .ITERABLE_JWT_API_KEY }}"
```

```sh
export RUDDER_ITERABLE_JWT_API_KEY=...
rudder-cli apply

# or
rudder-cli apply --var-file secrets.vars.yaml
```

`api_key` is **not** registered as a secret, so it is stored and returned in the
clear. Only the device/browser registration key is masked.

`rudder-cli import` writes the secret back as a `{{ .VAR }}` placeholder rather
than its value, since the API does not return secrets. Fill the placeholder in
before the first apply.
