# Braze (`braze`)

Braze is a customer engagement destination. RudderStack sends events to Braze's REST API from its servers, loads Braze's own SDKs in device mode, or combines the two in hybrid mode.

In a Braze destination spec:

- `type: braze`
- `definition_version: 1`

## Sample configuration

```yaml
version: rudder/v1
kind: destination
metadata:
  name: braze-prod
spec:
  id: braze-prod
  display_name: Braze Production
  type: braze
  definition_version: 1
  enabled: true
  config:
    data_center: US-03
    rest_api_key: "{{ .BRAZE_REST_API_KEY }}"
    use_platform_specific_api_keys: false
    app_key: "{{ .BRAZE_APP_KEY }}"

    enable_subscription_group_in_group_call: false
    enable_nested_array_operations: false
    send_purchase_event_with_extra_properties: false
    use_ecommerce_recommended_events: true
    support_dedup: true

    enable_braze_logging:
      web: false
    enable_push_notification:
      web: true
    allow_user_supplied_javascript:
      web: false
    track_anonymous_user:
      web: true

    connection_mode:
      web: hybrid
      ios_swift: device
      cloud: cloud
    consent_management:
      web:
        - provider: oneTrust
          consents:
            - marketing
```

The above example uses one app identifier key for every platform. Because some sources connect in `device` or `hybrid` mode and others in `cloud` or `hybrid`, it needs both `app_key` and `rest_api_key` — see [Key dependencies](#key-dependencies).

## Config keys

`config` accepts only the keys listed below. The [shared config key rules](../README.md#config-key-rules) cover unknown keys, defaults, and immutability.

### Key dependencies

Which keys Braze needs depends on the modes your sources connect in. Rudder CLI enforces each requirement below.

| Key | Required when |
| :-----| :-----|
| `rest_api_key` | Any source connects in `cloud` or `hybrid` mode. Also checked per connection — see [Connect a source](#connect-a-source). |
| `app_key` | `use_platform_specific_api_keys` is `false` and any source connects in `device` or `hybrid` mode |
| `android_api_key` | `use_platform_specific_api_keys` is `true` and `android`, `android_kotlin`, `react_native`, or `flutter` connects in `device` or `hybrid` mode |
| `ios_api_key` | `use_platform_specific_api_keys` is `true` and `ios`, `ios_swift`, `react_native`, or `flutter` connects in `device` or `hybrid` mode |
| `web_api_key` | `use_platform_specific_api_keys` is `true` and `web` connects in `device` or `hybrid` mode |

> [!WARNING]
> Write `use_platform_specific_api_keys` out whenever a source connects in `device` or `hybrid` mode. When it's omitted, neither Rudder CLI nor the API asks for `app_key` or any platform key — so the spec validates and applies, and device mode has no app identifier key to load Braze with.

### Connection

#### `data_center` — string, required

Braze data center of your account — visible in your Braze dashboard URL.

- One of `US-01` to `US-08`, `EU-01` to `EU-03`, or `AU-01`.
- A `{{ path || fallback }}` template is accepted in place of a literal.
- The dashboard defaults this field to `US-01`. Rudder CLI requires it explicitly.
- A spec that omits this key fails validation.

#### `rest_api_key` — string, required, secret

Braze REST API key, used whenever RudderStack calls Braze from its servers.

- Required when any source connects in `cloud` or `hybrid` mode. Leave it unset otherwise.
- At most 100 characters, and must not contain line breaks.
- A `{{ path || fallback }}` template is accepted in place of a literal.

The key needs the `users.track` permission, among others.

### App identifier keys

Braze's SDKs need an app identifier key. Use `app_key` for every platform, or set `use_platform_specific_api_keys: true` and give each platform its own key for separate attribution in Braze.

#### `use_platform_specific_api_keys` — boolean

Use a separate app identifier key per platform instead of the single `app_key`. The dashboard marks this setting as beta and defaults it to `false`.

- Applies when a source connects in `device` or `hybrid` mode.
- Has no default in Rudder CLI — see [Key dependencies](#key-dependencies).

#### `app_key` — string, required, secret

Braze default app identifier key, used for every platform.

- Required when `use_platform_specific_api_keys` is `false` and any source connects in `device` or `hybrid` mode.
- At most 100 characters, and must not contain line breaks.
- A `{{ path || fallback }}` template is accepted in place of a literal.

#### `android_api_key` — string, required, secret

App identifier key for Android. React Native and Flutter apps use it for their Android builds.

- Required when `use_platform_specific_api_keys` is `true` and `android`, `android_kotlin`, `react_native`, or `flutter` connects in `device` or `hybrid` mode.
- At most 100 characters, and must not contain line breaks. A template is accepted.

#### `ios_api_key` — string, required, secret

App identifier key for iOS. React Native and Flutter apps use it for their iOS builds.

- Required when `use_platform_specific_api_keys` is `true` and `ios`, `ios_swift`, `react_native`, or `flutter` connects in `device` or `hybrid` mode.
- At most 100 characters, and must not contain line breaks. A template is accepted.

#### `web_api_key` — string, required, secret

App identifier key for the web SDK.

- Required when `use_platform_specific_api_keys` is `true` and `web` connects in `device` or `hybrid` mode.
- At most 100 characters, and must not contain line breaks. A template is accepted.

The dashboard describes falling back to `app_key` when a platform key is blank, but Rudder CLI and the API both require each platform key whose condition is met.

### Event settings

#### `enable_subscription_group_in_group_call` — boolean, default `false`

Send the subscription group status in `group` events.

- Applies to `cloud` mode only.

#### `enable_nested_array_operations` — boolean, default `false`

Use Braze's nested custom attributes to update custom attribute objects. The dashboard calls this **Use Custom Attributes Operation**.

- Applies to `cloud` mode only.

#### `send_purchase_event_with_extra_properties` — boolean, default `false`

Include custom properties in purchase events.

- Applies to `cloud` mode only.

#### `use_ecommerce_recommended_events` — boolean, default `true`

Map RudderStack ecommerce `track` events such as **Product Viewed** and **Order Completed** to Braze's `ecommerce.*` recommended events. When `false`, they're sent as legacy custom and purchase events. The dashboard marks this setting as beta.

- Applies to both `cloud` and `device` mode.

#### `support_dedup` — boolean, default `false`

Deduplicate traits on `identify` and `track` calls, using Braze's `/users/export/ids` API to fetch existing attributes. If Braze's rate limit is hit, RudderStack sends all attributes without deduplicating.

### Web SDK settings

These keys configure Braze's web SDK, so they apply only when `connection_mode.web` is `device` or `hybrid`. Each is an object keyed by `web`, with a boolean value.

#### `enable_braze_logging` — object

Show Braze SDK logs in the browser console.

- `web` — boolean.

#### `enable_push_notification` — object

Use Braze web push notifications. Requires a service worker on your site.

- `web` — boolean.

#### `allow_user_supplied_javascript` — object

Enable HTML in-app messages, which can run JavaScript you supply in Braze.

- `web` — boolean.

#### `track_anonymous_user` — object

Track activity from anonymous users and send it to Braze.

- `web` — boolean.

### Event filtering

> [!WARNING]
> Client-side event filtering applies only to sources connected in `device` mode — the dashboard shows these controls only then, and the SDK is what applies the filter. Rudder CLI accepts the block in any mode, but it has no effect on cloud-mode events, or on the events hybrid mode sends through the API.

#### `event_filtering` — object

Restricts which `track` events the SDK passes to Braze, by event name.

- `whitelist` — array of event names to allow; every other `track` event is dropped.
- `blacklist` — array of event names to drop; every other `track` event is allowed.
- The two are mutually exclusive, and Rudder CLI enforces it — setting both fails validation.
- Each name is at most 100 characters, or a `{{ path || fallback }}` template.

### Per-source keys

Both keys are objects keyed by the local source type — the tokens listed under [Source types](#source-types). A key naming a source type this destination doesn't support fails validation.

#### `connection_mode` — object

Maps each source type you connect to the mode its events reach Braze in, using the modes in [Source types](#source-types). It also decides which API and app identifier keys are required.

- An entry is required for each source type you connect — see [Connect a source](#connect-a-source).
- A mode the source type doesn't support on this destination fails validation — for example `hybrid` for `react_native`.

```yaml
connection_mode:
  web: hybrid
  ios_swift: device
  react_native: device
  cloud: cloud
```

#### `consent_management` — object

Consent provider configuration per source type. The entry shape, accepted providers, and the rules on `resolution_strategy` and `consents` are shared across all destination types — see [Consent management](../common/README.md).

## Source types

Braze accepts events from these source types in the mentioned connection modes:

| Source type | Connection mode |
| :-----| :-----|
| `android` | `cloud`, `device`, `hybrid` |
| `android_kotlin` | `cloud`, `device`, `hybrid` |
| `ios` | `cloud`, `device`, `hybrid` |
| `ios_swift` | `cloud`, `device`, `hybrid` |
| `web` | `cloud`, `device`, `hybrid` |
| `unity` | `cloud` |
| `cloud` | `cloud` |
| `react_native` | `cloud`, `device` |
| `flutter` | `cloud`, `device` |
| `cordova` | `cloud` |
| `warehouse` | `cloud` |

In `hybrid` mode, RudderStack sends every user-generated event — `identify`, `track`, `page`, `screen`, `group`, and `alias` — through Braze's REST API, and loads Braze's SDK only for what needs it, such as in-app messages and push notifications.

`react_native` and `flutter` device mode covers their Android and iOS builds only. A Flutter app deployed to the web can reach Braze in `cloud` mode only.

`warehouse` is the token a Reverse ETL source resolves to — see [Source types](../README.md#source-types).

## Connect a source

An event stream connection to this destination is checked against three rules at `validate` time.

**The source's type must be supported.** A source's type resolves to one of the tokens above before the check — a JavaScript source resolves to `web`, and webhook and server-side SDK sources resolve to `cloud`. Every type an event stream or Reverse ETL source can resolve to is supported here, so this check always passes.

**The destination config must carry a `connection_mode` entry for that source type.** This lives on the destination spec, not on the connection spec. Without it:

```text
destination 'braze-prod' config has no 'connection_mode' entry for source type 'web'
```

**A source connecting in `cloud` or `hybrid` mode needs `rest_api_key`.** Device-mode connections don't. Without it:

```text
destination 'braze-prod' config is missing fields required to connect a 'cloud' source: rest_api_key
```

The app identifier keys aren't checked per connection — they're covered by the destination-level rules in [Key dependencies](#key-dependencies).

A Reverse ETL connection reaches this destination as source type `warehouse` and is checked against the same rules, so the config needs a `connection_mode.warehouse: cloud` entry for it, and `rest_api_key`.

## Secrets

Rudder CLI treats five keys as secrets: `rest_api_key`, `app_key`, `android_api_key`, `ios_api_key`, and `web_api_key`. Write each one you use as a `{{ .VAR }}` reference — [Secrets](../README.md#secrets) covers supplying the values and what `import` writes:

```yaml
config:
  rest_api_key: "{{ .BRAZE_REST_API_KEY }}"
  app_key: "{{ .BRAZE_APP_KEY }}"
```

In device and hybrid mode the app identifier keys are embedded in the app or page, so masking them protects your YAML, not the values themselves.
