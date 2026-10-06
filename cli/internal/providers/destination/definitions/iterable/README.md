# Iterable (`iterable`)

Iterable is a cross-channel marketing destination. RudderStack sends events to Iterable's API from its servers, or — for web sources — through Iterable's web SDK in device mode, which also shows in-app messages.

In an Iterable destination spec:

- `type: iterable`
- `definition_version: 1`

## Sample configuration

```yaml
version: rudder/v1
kind: destination
metadata:
  name: iterable-prod
spec:
  id: iterable-prod
  display_name: Iterable Production
  type: iterable
  definition_version: 1
  enabled: true
  config:
    api_key: "{{ .ITERABLE_API_KEY }}"
    data_center: USDC

    register_device_or_browser_api_key: "{{ .ITERABLE_MOBILE_WEB_KEY }}"
    prefer_user_id: true
    merge_nested_objects: true
    map_to_single_event: true
    track_all_pages: false
    track_categorized_pages: true
    track_named_pages: true

    package_name: acme-web
    initialisation_identifier:
      web: email
    get_in_app_event_mapping:
      web:
        - Viewed Pricing
    handle_links:
      web: open-all-new-tab
    close_button_position:
      web: top-right

    connection_mode:
      web: device
      cloud: cloud
    consent_management:
      web:
        - provider: oneTrust
          consents:
            - marketing
```

The above example connects web sources in `device` mode, which is what makes `package_name` required, and sends server-side events in `cloud` mode, where the page settings apply — see [Web device mode](#web-device-mode).

## Config keys

`config` accepts only the keys listed below. The [shared config key rules](../README.md#config-key-rules) cover unknown keys, defaults, and immutability.

> [!NOTE]
> In `cloud` mode Iterable accepts `identify`, `page`, `screen`, `track`, and `alias` events. In `device` mode, web sources send `identify` and `track`.

### Connection

#### `api_key` — string, required, secret

Your Iterable API key, from **Integrations** > **API Keys** in Iterable.

- At most 100 characters, and must not contain line breaks.
- A `{{ path || fallback }}` template is accepted in place of a literal.

#### `data_center` — string, required

Region of your Iterable project.

- `USDC` or `EUDC`.
- The dashboard defaults this field to `USDC`. Rudder CLI requires it explicitly.
- A spec that omits this key fails validation.

### Cloud mode

These keys apply to events sent in `cloud` mode. The dashboard shows them only when the destination has a cloud-mode connection.

#### `register_device_or_browser_api_key` — string, secret

Iterable mobile or web API key, used to register device and browser tokens. Needed when `identify` calls carry token information.

- At most 100 characters, and must not contain line breaks.
- A `{{ path || fallback }}` template is accepted in place of a literal.

#### `prefer_user_id` — boolean, default `true`

Create a new Iterable user when an event's `userId` doesn't match an existing one, using Iterable's `preferUserId` parameter. The dashboard calls this **Create new user if userID exists**.

#### `merge_nested_objects` — boolean, default `true`

Merge top-level objects on the user profile instead of overwriting them, using Iterable's `mergeNestedObjects` parameter. The dashboard calls this **Merge top-level objects**.

#### `map_to_single_event` — boolean, default `true`

Send every page under one event name — `Loaded a Page` for `page` calls and `Loaded a Screen` for `screen` calls.

#### `track_all_pages` — boolean, default `false`

Send every `page` event to Iterable.

#### `track_categorized_pages` — boolean, default `true`

Send `page` events that have a category.

#### `track_named_pages` — boolean, default `true`

Send `page` events that have a name.

### Web device mode

These keys configure Iterable's web SDK, so they apply only when `connection_mode.web` is `device`. Apart from `package_name`, each is keyed by `web`.

#### `package_name` — string, required

Name of the website Iterable shows in-app messages for.

- Required when `connection_mode.web` is `device`. Leave it unset otherwise.
- At most 100 characters, and must not contain line breaks.
- A `{{ path || fallback }}` template is accepted in place of a literal.

#### `initialisation_identifier` — object

Identifier the SDK uses to recognize a user across a session.

- `web` — `email` (default) or `userId`.

#### `get_in_app_event_mapping` — object

Event names that trigger fetching and showing in-app messages.

- `web` — array of event names, each at most 100 characters or a template.

```yaml
get_in_app_event_mapping:
  web:
    - Viewed Pricing
    - Started Checkout
```

#### `purchase_event_mapping` — object

Event names sent to Iterable as purchase events.

- `web` — array of event names, each at most 100 characters or a template.

#### `send_track_for_inapp` — object

Send a `track` event each time an in-app message is shown.

- `web` — boolean.

### In-app message display

These keys shape how in-app messages look and behave in the browser. They apply only when `connection_mode.web` is `device`, and each is keyed by `web`. Unless noted, the value is a string and isn't validated locally:

```yaml
animation_duration:
  web: "400"
top_offset:
  web: "10%"
handle_links:
  web: open-all-new-tab
close_button_position:
  web: top-right
is_required_to_dismiss_message:
  web: false
```

#### `animation_duration` — object

Time, in milliseconds, that messages take to animate in and out.

- `web` — string, for example `"400"`.

#### `display_interval` — object

Time, in milliseconds, to wait after a message closes before showing the next one.

- `web` — string.

#### `top_offset` — object

Space between the top of the screen and a message, in `px` or `%`. Doesn't apply to center, top, or full-screen messages.

- `web` — string, for example `"10%"`.

#### `right_offset` — object

Space between the right edge of the screen and a message, in `px` or `%`. Doesn't apply to center, top, or full-screen messages.

- `web` — string.

#### `bottom_offset` — object

Space between the bottom of the screen and a message, in `px` or `%`. Doesn't apply to center, top, or full-screen messages.

- `web` — string.

#### `handle_links` — object

How links inside a message open.

- `web` — `open-all-new-tab` (default), `open-all-same-tab`, or `external-new-tab`.

#### `on_open_screen_reader_message` — object

Text a screen reader announces when a message opens.

- `web` — string.

#### `on_open_node_to_take_focus` — object

DOM element that takes keyboard focus when a message opens.

- `web` — string, for example a CSS selector.

#### `close_button_color` — object

Color of the close button.

- `web` — string.

#### `close_button_size` — object, internal

Size of the close button.

- `web` — string.

#### `close_button_position` — object

Position of the close button on the message.

- `web` — `top-right` (default) or `top-left`.

#### `close_button_color_top_offset` — object

Space between the close button and the message's top border. Despite `color` in its name, this is an offset.

- `web` — string.

#### `close_button_color_side_offset` — object

Space between the close button and the message's side borders. Despite `color` in its name, this is an offset.

- `web` — string.

#### `icon_path` — object

Path to an image or SVG shown instead of the default close icon.

- `web` — string.

#### `is_required_to_dismiss_message` — object

Stop users from dismissing a message by clicking outside it.

- `web` — boolean.

### Event filtering

> [!WARNING]
> Client-side event filtering applies only when `connection_mode.web` is `device` — the dashboard shows these controls only for device-mode connections, and the SDK is what applies the filter. Rudder CLI accepts the block in any mode, but it has no effect on cloud-mode events.

#### `event_filtering` — object

Restricts which `track` events the SDK passes to Iterable, by event name.

- `whitelist` — array of event names to allow; every other `track` event is dropped.
- `blacklist` — array of event names to drop; every other `track` event is allowed.
- The two are mutually exclusive, and Rudder CLI enforces it — setting both fails validation.
- Each name is at most 100 characters, or a `{{ path || fallback }}` template.

### Per-source keys

Both keys are objects keyed by the local source type — the tokens listed under [Source types](#source-types). A key naming a source type this destination doesn't support fails validation.

#### `connection_mode` — object

Maps each source type you connect to the mode its events reach Iterable in, using the modes in [Source types](#source-types).

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

Iterable accepts events from these source types in the mentioned connection modes:

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

Only `web` offers `device` mode, which loads Iterable's web SDK — recommended if you use Iterable's web push notifications.

`warehouse` is the token a Reverse ETL source resolves to — see [Source types](../README.md#source-types).

> [!NOTE]
> The dashboard additionally offers Iterable to AMP and Shopify sources. Rudder CLI doesn't manage those connections, so `amp` and `shopify` are invalid here.

## Connect a source

An event stream connection to this destination is checked against two rules at `validate` time.

**The source's type must be supported.** A source's type resolves to one of the tokens above before the check — a JavaScript source resolves to `web`, and webhook and server-side SDK sources resolve to `cloud`. An unsupported type reports:

```text
destination 'iterable-prod' (type 'iterable') does not support source 'my-source':
source type 'amp' is not among supported source types: android, android_kotlin, ...
```

**The destination config must carry a `connection_mode` entry for that source type.** This lives on the destination spec, not on the connection spec. Without it:

```text
destination 'iterable-prod' config has no 'connection_mode' entry for source type 'web'
```

Iterable needs no additional config keys to connect a source of any type, in any mode. `package_name` is checked on the destination instead — see [Web device mode](#web-device-mode).

A Reverse ETL connection reaches this destination as source type `warehouse` and is checked against the same rules, so the config needs a `connection_mode.warehouse: cloud` entry for it.

## Secrets

`api_key` and `register_device_or_browser_api_key` are the secret keys. Write each as a `{{ .VAR }}` reference and supply the value at apply time:

```yaml
config:
  api_key: "{{ .ITERABLE_API_KEY }}"
  register_device_or_browser_api_key: "{{ .ITERABLE_MOBILE_WEB_KEY }}"
```

```bash
export RUDDER_ITERABLE_API_KEY="..."
rudder-cli apply

# or
rudder-cli apply --var-file secrets.vars.yaml
```

Note that:

- The YAML that `rudder-cli import` writes may or may not include secret keys. Before you apply, make sure every secret key your configuration needs is present and populated through variable substitution.
