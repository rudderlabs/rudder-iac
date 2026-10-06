# PostHog (`posthog`)

PostHog is a product analytics destination. RudderStack sends events to PostHog from its servers, or — for web sources — through PostHog's own SDK in device mode.

In a PostHog destination spec:

- `type: posthog`
- `definition_version: 1`

## Sample configuration

```yaml
version: rudder/v1
kind: destination
metadata:
  name: posthog-prod
spec:
  id: posthog-prod
  display_name: PostHog Production
  type: posthog
  definition_version: 1
  enabled: true
  config:
    api_key: "{{ .POSTHOG_API_KEY }}"
    endpoint: https://eu.posthog.com
    use_v2_group: true

    autocapture:
      web: true
    capture_page_view:
      web: false
    disable_session_recording:
      web: false
    enable_local_storage_persistence:
      web: true
    person_profiles:
      web: identified_only
    xhr_headers:
      - key: X-Environment
        value: production
    property_blacklist:
      - property: email

    connection_mode:
      web: device
      cloud: cloud
    consent_management:
      web:
        - provider: oneTrust
          consents:
            - analytics
```

The above example loads PostHog's SDK on web sources in `device` mode, where the web settings apply, and sends server-side events in `cloud` mode, where `use_v2_group` applies — see [Web device mode](#web-device-mode).

## Config keys

`config` accepts only the keys listed below. The [shared config key rules](../README.md#config-key-rules) cover unknown keys, defaults, and immutability.

### Connection

#### `api_key` — string, required, secret

Your PostHog team API key — the dashboard calls it **Team API Key**.

- At most 100 characters, and must not contain line breaks.
- Templates aren't accepted in place of a literal; a template is measured as text against the same limit.

#### `endpoint` — string

URL of your PostHog instance, without a trailing slash — for example `https://eu.posthog.com`, or your own host if you self-host PostHog. The dashboard calls it **Instance URL**. In device mode, PostHog's SDK loads from here, or from `https://app.posthog.com` when omitted.

- At most 100 characters, and must not contain line breaks.
- Any value containing `.ngrok.io` is rejected.

#### `use_v2_group` — boolean, default `false`

Send `group` traits as event properties — PostHog's updated group mapping. Turn this on; the old mapping is being deprecated.

- Applies to `cloud` mode only. Device mode always uses the updated mapping.

### Web device mode

These keys configure PostHog's SDK in the browser, so they apply only to web sources connected in `device` mode. Most are objects keyed by `web`; `xhr_headers` and `property_blacklist` are plain lists that apply to web all the same.

#### `autocapture` — object

Let PostHog's SDK autocapture clicks, form submissions, and other interactions.

- `web` — boolean.

#### `capture_page_view` — object

Let PostHog's SDK send a page view each time it loads on a page.

- `web` — boolean.

#### `disable_session_recording` — object

Stop PostHog from recording user sessions. `true` turns recording **off**.

- `web` — boolean.

#### `enable_local_storage_persistence` — object

Store PostHog's state in local storage instead of cookies, reducing the data held in cookies.

- `web` — boolean.

#### `person_profiles` — object

When PostHog creates a person profile.

- `web` — `always` (default) to create profiles for every user, or `identified_only` to capture anonymous events without profiles until a user is identified.

```yaml
person_profiles:
  web: identified_only
```

#### `xhr_headers` — array of objects

Extra headers the SDK adds to its requests to the PostHog API.

- `key` — header name. At most 100 characters, or a `{{ path || fallback }}` template.
- `value` — header value. Same limits. Not secret — it ships to the browser.
- A plain list, not keyed by `web`, though it applies only to web.

```yaml
xhr_headers:
  - key: X-Environment
    value: production
```

#### `property_blacklist` — array of objects

Traits and event properties the SDK doesn't send to PostHog.

- `property` — the trait or property name. Not validated locally.
- A plain list, not keyed by `web`, though it applies only to web.

```yaml
property_blacklist:
  - property: email
  - property: phone
```

### Event filtering

> [!WARNING]
> Client-side event filtering is applied by the RudderStack SDK, so it affects only web sources connected in `device` mode. Events sent in `cloud` mode aren't filtered by it.

#### `event_filtering` — object

Restricts which `track` events the SDK passes to PostHog, by event name.

- `whitelist` — array of event names to allow; every other `track` event is dropped.
- `blacklist` — array of event names to drop; every other `track` event is allowed.
- The two are mutually exclusive, and Rudder CLI enforces it — setting both fails validation.
- Each name is at most 100 characters, or a `{{ path || fallback }}` template.

### Per-source keys

Both keys are objects keyed by the local source type — the tokens listed under [Source types](#source-types). A key naming a source type this destination doesn't support fails validation.

#### `connection_mode` — object

Maps each source type you connect to the mode its events reach PostHog in, using the modes in [Source types](#source-types).

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

PostHog accepts events from these source types in the mentioned connection modes:

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

Only `web` offers `device` mode, which loads PostHog's SDK in the browser. In `cloud` mode PostHog accepts `alias`, `group`, `identify`, `page`, `screen`, and `track`; in `device` mode, web sources send `group`, `identify`, `page`, and `track`.

`warehouse` is the token a Reverse ETL source resolves to — see [Source types](../README.md#source-types).

> [!NOTE]
> The dashboard additionally offers PostHog to AMP and Shopify sources. Rudder CLI doesn't manage those connections, so `amp` and `shopify` are invalid here.

## Connect a source

An event stream connection to this destination is checked against two rules at `validate` time.

**The source's type must be supported.** A source's type resolves to one of the tokens above before the check — a JavaScript source resolves to `web`, and webhook and server-side SDK sources resolve to `cloud`. An unsupported type reports:

```text
destination 'posthog-prod' (type 'posthog') does not support source 'my-source':
source type 'amp' is not among supported source types: android, android_kotlin, ...
```

**The destination config must carry a `connection_mode` entry for that source type.** This lives on the destination spec, not on the connection spec. Without it:

```text
destination 'posthog-prod' config has no 'connection_mode' entry for source type 'web'
```

PostHog needs no additional config keys to connect a source of any type, in any mode.

A Reverse ETL connection reaches this destination as source type `warehouse` and is checked against the same rules, so the config needs a `connection_mode.warehouse: cloud` entry for it.

## Secrets

`api_key` is the only secret key. Write it as a `{{ .VAR }}` reference and supply the value at apply time:

```yaml
config:
  api_key: "{{ .POSTHOG_API_KEY }}"
```

```bash
export RUDDER_POSTHOG_API_KEY="phc_..."
rudder-cli apply

# or
rudder-cli apply --var-file secrets.vars.yaml
```

Note that:

- The YAML that `rudder-cli import` writes may or may not include secret keys. Before you apply, make sure every secret key your configuration needs is present and populated through variable substitution.
- In device mode the team API key is embedded in the page's JavaScript, so masking it protects your YAML, not the value itself.
- The dashboard also masks `endpoint` (**Instance URL**). Rudder CLI doesn't treat it as a secret, so it's written to YAML in plain text.
