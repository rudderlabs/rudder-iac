# Google Analytics 4 (`ga4`)

Google Analytics 4 is a web and app analytics destination. RudderStack sends events to GA4's Measurement Protocol from its servers, loads Google's own SDKs in device mode, or — for web — splits the two in hybrid mode.

In a Google Analytics 4 destination spec:

- `type: ga4`
- `definition_version: 1`

## Sample configuration

```yaml
version: rudder/v1
kind: destination
metadata:
  name: ga4-prod
spec:
  id: ga4-prod
  display_name: Google Analytics 4 Production
  type: ga4
  definition_version: 1
  enabled: true
  config:
    api_secret: "{{ .GA4_API_SECRET }}"
    client_type: gtag
    measurement_id: "{{ .GA4_MEASUREMENT_ID }}"

    debug_mode: false
    pii_properties_to_ignore:
      - pii_property: email
      - pii_property: phone

    capture_page_view:
      web: rs
    debug_view:
      web: false
    extend_page_view_params:
      web: true
    override_client_and_session_ids:
      web: true

    connection_mode:
      web: hybrid
      cloud: cloud
    consent_management:
      web:
        - provider: oneTrust
          consents:
            - analytics
```

The above example uses the gtag client with web sources in `hybrid` mode, which is where `override_client_and_session_ids` applies — see [Web SDK settings](#web-sdk-settings).

## Config keys

`config` accepts only the keys listed below. The [shared config key rules](../README.md#config-key-rules) cover unknown keys, defaults, and immutability.

> [!NOTE]
> In `cloud` mode GA4 accepts `track`, `group`, and `page` events. In `device` mode, web sources send `identify`, `track`, `page`, and `group`, and Android and iOS sources send `identify`, `track`, and `screen`. In `hybrid` mode, `page` goes through device mode and everything else through cloud mode.

### Connection

#### `api_secret` — string, required, secret

Measurement Protocol API secret of your GA4 data stream, from **Admin** > **Data Streams** > your stream > **Measurement Protocol API secrets**.

- At most 100 characters, and must not contain line breaks.
- A `{{ path || fallback }}` template is accepted in place of a literal.

#### `client_type` — string, required

Which Google client the data stream uses — gtag for websites, Firebase for apps.

- `gtag` — needs `measurement_id`. Required for `hybrid` mode.
- `firebase` — needs `firebase_app_id`.
- Templates aren't accepted.
- The dashboard defaults this field to `gtag`. Rudder CLI requires it explicitly.
- A spec that omits this key fails validation.

#### `measurement_id` — string, required, secret

Measurement ID of your GA4 web data stream, from **Admin** > **Data Streams** > your stream.

- Required when `client_type` is `gtag`. Leave it unset otherwise.
- Must start with `G-`, followed by 1 to 100 characters.
- A `{{ path || fallback }}` template is accepted in place of a literal.

#### `firebase_app_id` — string, required

Firebase App ID of your GA4 app data stream.

- Required when `client_type` is `firebase`. Leave it unset otherwise.
- At most 100 characters, and must not contain line breaks.
- A `{{ path || fallback }}` template is accepted in place of a literal.

### Event handling

#### `debug_mode` — boolean, default `false`

Send cloud-mode events to GA4's validation server instead of reporting them, so you can check validation responses in Live Events. Events sent this way don't appear in reports.

- Applies to events sent in `cloud` mode, including the cloud half of `hybrid` mode.

#### `pii_properties_to_ignore` — array of objects

User traits RudderStack drops before sending events to GA4 — for example email addresses and phone numbers.

- `pii_property` — the trait name. At most 100 characters, or a `{{ path || fallback }}` template.

```yaml
pii_properties_to_ignore:
  - pii_property: email
```

### Web SDK settings

These keys configure gtag in the browser, so they apply only when `connection_mode.web` is `device` or `hybrid`. The four objects are keyed by `web`.

#### `sdk_base_url` — string

Custom domain that serves gtag, loaded as `https://<domain>/gtag/js`. When omitted, `https://www.googletagmanager.com` is used.

- Applies when `connection_mode.web` is `device` or `hybrid`.
- Must be a domain URL, and any value containing `.ngrok.io` is rejected. Rudder CLI checks the format only when `client_type` is `gtag` and `connection_mode.web` is `device`.
- A `{{ path || fallback }}` template is accepted in place of a literal.

#### `server_container_url` — string

URL of your GA4 server-side Google Tag Manager container.

- Applies when `connection_mode.web` is `device` or `hybrid`.
- Not validated locally.

#### `capture_page_view` — object

How page views reach GA4.

- `web` — `rs` (default) to send them from RudderStack's `page` calls, or `gtag` to use GA4 Enhanced Measurement's automatic collection on each page load.
- Make sure your GA4 measurement configuration matches this setting.

#### `debug_view` — object

Show device-mode events in GA4's DebugView. In `hybrid` mode that covers the `page` calls only.

- `web` — boolean. The dashboard defaults it to `true`; Rudder CLI doesn't fill it in.

#### `extend_page_view_params` — object

Add `url` and `search` to the properties sent with each page view. GA4 limits unique properties per event name, so weigh this against your other properties.

- `web` — boolean.

#### `override_client_and_session_ids` — object

Replace gtag's `client_id` and `session_id` with RudderStack's `anonymousId` and `session_id`, so `page` calls from the browser and other calls from the server stitch into the same sessions.

- Applies when `connection_mode.web` is `hybrid`. Leave it unset otherwise.
- `web` — boolean.

> [!WARNING]
> Switching an existing device-mode setup to `hybrid` with this on can briefly spike session counts.

### Event filtering

> [!WARNING]
> Client-side event filtering applies only to sources connected in `device` mode — web, Android, or iOS. The dashboard shows these controls only then, and the SDK is what applies the filter. Rudder CLI accepts the block in any mode, but it has no effect on cloud-mode connections.

#### `event_filtering` — object

Restricts which `track` events the SDK passes to GA4, by event name.

- `whitelist` — array of event names to allow; every other `track` event is dropped.
- `blacklist` — array of event names to drop; every other `track` event is allowed.
- The two are mutually exclusive, and Rudder CLI enforces it — setting both fails validation.
- Each name is at most 100 characters, or a `{{ path || fallback }}` template.

### Per-source keys

Both keys are objects keyed by the local source type — the tokens listed under [Source types](#source-types). A key naming a source type this destination doesn't support fails validation.

#### `connection_mode` — object

Maps each source type you connect to the mode its events reach GA4 in, using the modes in [Source types](#source-types).

- An entry is required for each source type you connect — see [Connect a source](#connect-a-source).
- A mode the source type doesn't support on this destination fails validation — for example `hybrid` for `android`.

```yaml
connection_mode:
  web: hybrid
  android: device
  cloud: cloud
```

#### `consent_management` — object

Consent provider configuration per source type. The entry shape, accepted providers, and the rules on `resolution_strategy` and `consents` are shared across all destination types — see [Consent management](../common/README.md).

## Source types

Google Analytics 4 accepts events from these source types in the mentioned connection modes:

| Source type | Connection mode |
| :-----| :-----|
| `android` | `cloud`, `device` |
| `android_kotlin` | `cloud` |
| `ios` | `cloud`, `device` |
| `ios_swift` | `cloud` |
| `web` | `cloud`, `device`, `hybrid` |
| `unity` | `cloud` |
| `react_native` | `cloud` |
| `flutter` | `cloud` |
| `cordova` | `cloud` |
| `cloud` | `cloud` |
| `warehouse` | `cloud` |

`web`, `android`, and `ios` offer `device` mode, and `web` alone offers `hybrid`, which needs `client_type: gtag`.

`warehouse` is the token a Reverse ETL source resolves to — see [Source types](../README.md#source-types).

> [!NOTE]
> The dashboard additionally offers Google Analytics 4 to AMP and Shopify sources. Rudder CLI doesn't manage those connections, so `amp` and `shopify` are invalid here.

## Connect a source

An event stream connection to this destination is checked against two rules at `validate` time.

**The source's type must be supported.** A source's type resolves to one of the tokens above before the check — a JavaScript source resolves to `web`, and webhook and server-side SDK sources resolve to `cloud`. An unsupported type reports:

```text
destination 'ga4-prod' (type 'ga4') does not support source 'my-source':
source type 'amp' is not among supported source types: android, android_kotlin, ...
```

**The destination config must carry a `connection_mode` entry for that source type.** This lives on the destination spec, not on the connection spec. Without it:

```text
destination 'ga4-prod' config has no 'connection_mode' entry for source type 'web'
```

Google Analytics 4 needs no additional config keys to connect a source of any type, in any mode.

A Reverse ETL connection reaches this destination as source type `warehouse` and is checked against the same rules, so the config needs a `connection_mode.warehouse: cloud` entry for it.

## Secrets

`api_secret` and `measurement_id` are the secret keys. Write each as a `{{ .VAR }}` reference and supply the value at apply time:

```yaml
config:
  api_secret: "{{ .GA4_API_SECRET }}"
  measurement_id: "{{ .GA4_MEASUREMENT_ID }}"
```

```bash
export RUDDER_GA4_API_SECRET="..."
export RUDDER_GA4_MEASUREMENT_ID="G-XXXXXXXXXX"
rudder-cli apply

# or
rudder-cli apply --var-file secrets.vars.yaml
```

Note that:

- The YAML that `rudder-cli import` writes may or may not include secret keys. Before you apply, make sure every secret key your configuration needs is present and populated through variable substitution.
- In device and hybrid mode the measurement ID is embedded in the page's JavaScript, so masking it protects your YAML, not the value itself.
- `firebase_app_id` isn't a secret and is imported as-is.
