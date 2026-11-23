# Customer.io (`customerio`)

Customer.io is a messaging and marketing automation destination. RudderStack sends events to Customer.io's APIs from its servers, or through Customer.io's own SDKs in device mode on web, Android, and iOS.

In a Customer.io destination spec:

- `type: customerio`
- `definition_version: 1`

## Sample configuration

```yaml
version: rudder/v1
kind: destination
metadata:
  name: customerio-prod
spec:
  id: customerio-prod
  display_name: Customer.io Production
  type: customerio
  definition_version: 1
  enabled: true
  config:
    site_id: "{{ .CUSTOMERIO_SITE_ID }}"
    api_key: "{{ .CUSTOMERIO_API_KEY }}"
    datacenter: US
    api_version: v2
    user_id_identifier_type: id
    device_token_event_name: Device Token Registered

    sdk_version:
      web: v2
    write_key:
      web: "{{ .CUSTOMERIO_WRITE_KEY }}"
    anonymous_in_app:
      web: false
    send_page_name_in_sdk:
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

The above example sends server-side events through the v2 API, and connects web and Android sources in `device` mode, where the SDK settings apply. Web loads the v2 JavaScript client, so it needs `write_key` — see [Key dependencies](#key-dependencies).

## Config keys

`config` accepts only the keys listed below. The [shared config key rules](../README.md#config-key-rules) cover unknown keys, defaults, and immutability.

### Key dependencies

Which credentials Customer.io needs depends on the modes your sources connect in, and on the web SDK version. Rudder CLI enforces each requirement below.

| Key | Required when |
| :-----| :-----|
| `api_key` | Always, unless `web` is the only source type and connects in `device` mode. Also checked per connection — see [Connect a source](#connect-a-source). |
| `site_id` | Always, unless `web` is the only source type and connects in `device` mode with `sdk_version.web` set to `v2`. Also checked per connection. |
| `write_key` | `connection_mode.web` is `device` and `sdk_version.web` is `v2` |
| `user_id_identifier_type` | `api_version` is `v2`, including when it's omitted |

> [!WARNING]
> Write `sdk_version` out to use the v2 web client. When the block is omitted, nothing fills it in — the destination is stored without a version, which Customer.io's web SDK and the API's checks both read as `v1`. The `v2` default applies only inside an `sdk_version` block you write.

### Connection

#### `site_id` — string, required, secret

Your Customer.io site ID.

- Required unless `web` is the only source type and connects in `device` mode with `sdk_version.web` set to `v2`. See [Key dependencies](#key-dependencies).
- At most 100 characters, and must not contain line breaks.
- A `{{ path || fallback }}` template is accepted in place of a literal.

#### `api_key` — string, required, secret

Your Customer.io Tracking API key, paired with `site_id`.

- Required unless `web` is the only source type and connects in `device` mode.
- At most 100 characters, and must not contain line breaks.
- A `{{ path || fallback }}` template is accepted in place of a literal.

Supply it as a `{{ .VAR }}` reference rather than a literal — see [Secrets](#secrets).

#### `datacenter` — string, required

Region of your Customer.io account.

- `US` or `EU`.
- The dashboard defaults this field to `US`. Rudder CLI requires it explicitly.
- A spec that omits this key fails validation.

#### `device_token_event_name` — string

Name of the event your app fires right after it sets the device token, so RudderStack can send the token to Customer.io immediately. Applies in both modes.

- At most 100 characters, and must not contain line breaks.
- A `{{ path || fallback }}` template is accepted in place of a literal.

### Cloud mode

These keys apply to events sent in `cloud` mode. They have no effect on device-mode delivery.

#### `api_version` — string, default `v2`

Customer.io API that delivers cloud-mode events.

- `v2` — Customer.io's unified batch API. Needs `user_id_identifier_type`.
- `v1` — Customer.io's per-endpoint APIs.
- Omitting this key is the same as setting `v2`.

#### `user_id_identifier_type` — string, required

Customer.io identifier that receives the RudderStack `userId`.

- Required when `api_version` is `v2` — including when you omit `api_version`, since `v2` is the default. Leave it unset only with `v1`.
- One of `id`, `email`, `phone`, or `cio_id`. With `phone`, the value must read as E.164 — for example `+15551234567`.

> [!WARNING]
> `userId` is sent as this identifier for every event, with no fallback — choose `id` and send an email address, and it's sent as an ID. A wrong choice attaches events to the wrong profiles or creates new ones, and that can't be reversed. With `v2`, events without a `userId` fail.

### Device mode

These keys configure Customer.io's SDKs, so they apply only to sources connected in `device` mode. Each is keyed by the source type it applies to.

#### `sdk_version` — object

Customer.io JavaScript client that web device mode loads. `v1` loads the legacy snippet and uses `site_id`. `v2` loads the Data Pipelines JavaScript client and needs `write_key`. Switching an existing destination to `v2` changes where your web events land.

- `web` — `v1` or `v2`. Defaults to `v2`, but only inside an `sdk_version` block you write — an omitted block is read as `v1`.
- Applies when `connection_mode.web` is `device`.

```yaml
sdk_version:
  web: v2
```

#### `write_key` — object

Write key of your Customer.io Data Pipelines JavaScript source. On `v2`, web events go to this source instead of the Track API.

- `web` — string. Required when `connection_mode.web` is `device` and `sdk_version.web` is `v2`.
- At most 100 characters, and must not contain line breaks. A `{{ path || fallback }}` template is measured as literal text against the same limit.

#### `anonymous_in_app` — object

Show in-app messages to website visitors you haven't identified yet. Needs in-app messaging turned on in your Customer.io workspace, and a Customer.io plan that supports anonymous in-app messages. In-app messages for identified users need no setting on `v2`.

- `web` — boolean. The dashboard defaults it to `false`.
- Applies when `sdk_version.web` is `v2`.

#### `send_page_name_in_sdk` — object

Send the `page` name to Customer.io. When `false`, Customer.io records the page URL instead.

- `web` — boolean. The dashboard defaults it to `true`; Rudder CLI doesn't fill it in.

#### `data_use_in_app` — object

Enable Customer.io in-app messages on your website.

- `web` — boolean. The dashboard defaults it to `false`.
- Applies to the `v1` web client. Rudder CLI also accepts it with `v2`, where it has no effect.

#### `auto_track_device_attributes` — object

Let Customer.io's mobile SDK track device attributes automatically. Set `false` to track them yourself.

- `android`, `ios` — booleans. The dashboard defaults both to `true`; Rudder CLI doesn't fill them in.

```yaml
auto_track_device_attributes:
  android: true
  ios: false
```

#### `background_queue_min_number_of_tasks` — object

Minimum number of tasks the Android SDK keeps in its background queue.

- `android` — a whole number written as a string, `"10"` rather than `10`. At most 100 characters, or a template.
- The dashboard defaults it to `"10"`; Rudder CLI doesn't fill it in.

#### `background_queue_seconds_delay` — object

Delay, in seconds, that the Android SDK holds events in its background queue.

- `android` — a whole number written as a string, `"30"` rather than `30`. At most 100 characters, or a template.
- The dashboard defaults it to `"30"`; Rudder CLI doesn't fill it in.

### Event filtering

> [!WARNING]
> Client-side event filtering applies only when `connection_mode.web` is `device` — the dashboard shows these controls only then, and the SDK is what applies the filter. Rudder CLI accepts the block in any mode, but it has no effect on other connections.

#### `event_filtering` — object

Restricts which `track` events the web SDK passes to Customer.io, by event name.

- Applies when `connection_mode.web` is `device`. Leave it unset otherwise.
- `whitelist` — array of event names to allow; every other `track` event is dropped.
- `blacklist` — array of event names to drop; every other `track` event is allowed.
- The two are mutually exclusive, and Rudder CLI enforces it — setting both fails validation.
- Each name is at most 100 characters, or a `{{ path || fallback }}` template.

### Per-source keys

Both keys are objects keyed by the local source type — the tokens listed under [Source types](#source-types). A key naming a source type this destination doesn't support fails validation.

#### `connection_mode` — object

Maps each source type you connect to the mode its events reach Customer.io in, using the modes in [Source types](#source-types).

- An entry is required for each source type you connect — see [Connect a source](#connect-a-source).
- A mode the source type doesn't support on this destination fails validation — for example `device` for `android_kotlin`.

```yaml
connection_mode:
  web: device
  android: device
  ios: cloud
```

#### `consent_management` — object

Consent provider configuration per source type. The entry shape, accepted providers, and the rules on `resolution_strategy` and `consents` are shared across all destination types — see [Consent management](../common/README.md).

## Source types

Customer.io accepts events from these source types in the mentioned connection modes:

| Source type | Connection mode |
| :-----| :-----|
| `android` | `cloud`, `device` |
| `android_kotlin` | `cloud` |
| `ios` | `cloud`, `device` |
| `ios_swift` | `cloud` |
| `web` | `cloud`, `device` |
| `unity` | `cloud` |
| `cloud` | `cloud` |
| `react_native` | `cloud` |
| `flutter` | `cloud` |
| `cordova` | `cloud` |
| `warehouse` | `cloud` |

`web`, `android`, and `ios` offer `device` mode. The Kotlin and Swift SDKs — `android_kotlin` and `ios_swift` — are `cloud` only.

`warehouse` is the token a Reverse ETL source resolves to — see [Source types](../README.md#source-types).

## Connect a source

An event stream connection to this destination is checked against three rules at `validate` time.

**The source's type must be supported.** A source's type resolves to one of the tokens above before the check — a JavaScript source resolves to `web`, and webhook and server-side SDK sources resolve to `cloud`. Every type an event stream or Reverse ETL source can resolve to is supported here, so this check always passes.

**The destination config must carry a `connection_mode` entry for that source type.** This lives on the destination spec, not on the connection spec. Without it:

```text
destination 'customerio-prod' config has no 'connection_mode' entry for source type 'web'
```

**Every source needs `api_key` and `site_id`, except `web` connecting in `device` mode.** Without them:

```text
destination 'customerio-prod' config is missing fields required to connect a 'cloud' source: api_key, site_id
```

What `web` needs in `device` mode depends on the other source types and on `sdk_version` — see [Key dependencies](#key-dependencies).

Rudder CLI accepts `warehouse` in `connection_mode`, but refuses a Reverse ETL connection to Customer.io, which runs a destination-specific Reverse ETL flow the CLI doesn't support:

```text
destination api type "CUSTOMERIO" uses a destination-specific rETL flow, which is not supported
```

## Secrets

`api_key` and `site_id` are the secret keys. Write each as a `{{ .VAR }}` reference — [Secrets](../README.md#secrets) covers supplying the values and what `import` writes:

```yaml
config:
  site_id: "{{ .CUSTOMERIO_SITE_ID }}"
  api_key: "{{ .CUSTOMERIO_API_KEY }}"
```

In web device mode the site ID (`v1`) or `write_key` (`v2`) is embedded in the page's JavaScript, so masking it protects your YAML, not the value itself. `write_key` isn't treated as a secret.
