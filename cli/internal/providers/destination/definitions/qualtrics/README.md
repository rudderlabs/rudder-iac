# Qualtrics (`qualtrics`)

Qualtrics is a device mode destination for website and app feedback surveys. RudderStack loads Qualtrics' intercept code in the browser and passes it the names of `page` and `track` events, and loads Qualtrics' SDK in Android and iOS apps and sets `identify` traits on it, so your intercepts can target users by what they do and who they are.

In a Qualtrics destination spec:

- `type: qualtrics`
- `definition_version: 1`

> [!NOTE]
> `qualtrics` is an [unverified destination type](../README.md#unverified-destination-types). Rudder CLI registers it only when the `unverifiedDestinations` experimental flag is on.

## Sample configuration

```yaml
version: rudder/v1
kind: destination
metadata:
  name: qualtrics-prod
spec:
  id: qualtrics-prod
  display_name: Qualtrics Production
  type: qualtrics
  definition_version: 1
  enabled: true
  config:
    project_id: ZN_blw7AbCTWxCGung
    brand_id: acmecorp
    enable_generic_page_title:
      web: false

    event_filtering:
      blacklist:
        - Product Viewed

    connection_mode:
      web: device
      android: device
      ios: device
    consent_management:
      web:
        - provider: oneTrust
          consents:
            - analytics
```

The above example connects web, Android, and iOS sources, all in `device` mode. `enable_generic_page_title` and `event_filtering` affect only the web integration — see [Web settings](#web-settings) and [Event filtering](#event-filtering).

## Config keys

`config` accepts only the keys listed below. The [shared config key rules](../README.md#config-key-rules) cover unknown keys, defaults, and immutability.

> [!NOTE]
> Qualtrics runs only in device mode. On web it acts on `page` and `track` calls, and passes their names to Qualtrics as events your intercepts can target. On Android and iOS it acts only on `identify` calls, and sets their string and number traits as Qualtrics properties for intercept targeting.

### Connection

#### `project_id` — string, required

ID of the Qualtrics Website/App Feedback project whose intercepts RudderStack loads, such as `ZN_blw7AbCTWxCGung`. It's listed under the project's **Project IDs** settings, next to the brand ID.

- Used on web, Android, and iOS.
- Not validated locally beyond being present.
- The dashboard masks this field, but Rudder CLI doesn't treat it as a secret — see [Secrets](#secrets).

#### `brand_id` — string, required

ID of your organization's Qualtrics brand, listed with the project ID.

- Used on web, Android, and iOS.
- Not validated locally beyond being present.

### Web settings

#### `enable_generic_page_title` — object

Send every `page` call to Qualtrics as `Viewed a Page`, instead of a name built from the call's category and name.

- `web` — boolean. The dashboard defaults it to `false`; Rudder CLI doesn't fill it in.
- Applies to `device` mode on `web` sources only — the Android and iOS integrations ignore `page` calls. `web` is the only key the object accepts.
- When `false` or unset, a `page` call is sent as `Viewed <category> <name> Page`, or `Viewed <name> Page` when it has no category, so give your `page` calls a name. The category comes from the call's category, or from `properties.category`. A call with neither a name nor a category isn't sent.

```yaml
enable_generic_page_title:
  web: true
```

### Event filtering

#### `event_filtering` — object

Restricts which `track` events the SDK passes to Qualtrics, by event name. `page` calls aren't filtered.

- `whitelist` — array of event names to allow; every other `track` event is dropped.
- `blacklist` — array of event names to drop; every other `track` event is allowed.
- The two are mutually exclusive, and Rudder CLI enforces it — setting both fails validation.
- Each name is at most 100 characters, or a `{{ path || fallback }}` template.
- Omit the block entirely to filter nothing.

Filtering changes nothing for Android and iOS sources: their Qualtrics integrations act only on `identify` calls, which aren't filtered.

### Per-source keys

Both keys are objects keyed by the local source type — the tokens listed under [Source types](#source-types). A key naming a source type this destination doesn't support fails validation.

#### `connection_mode` — object

Maps each source type you connect to the mode its events reach Qualtrics in. Qualtrics accepts only `device`, for `web`, `android`, and `ios`.

- An entry is required for each source type you connect — see [Connect a source](#connect-a-source).
- A mode the source type doesn't support on this destination fails validation — for example `cloud` for `web`.

```yaml
connection_mode:
  web: device
  android: device
  ios: device
```

#### `consent_management` — object

Consent provider configuration per source type. The entry shape, accepted providers, and the rules on `resolution_strategy` and `consents` are shared across all destination types — see [Consent management](../common/README.md).

## Source types

Qualtrics accepts events from these source types in the mentioned connection modes:

| Source type | Connection mode |
| :-----| :-----|
| `web` | `device` |
| `android` | `device` |
| `ios` | `device` |

Every source type is `device` only — RudderStack loads Qualtrics' own code in the browser or app, and no events pass through RudderStack's servers. The Kotlin and Swift SDKs — `android_kotlin` and `ios_swift` — aren't supported.

Qualtrics doesn't accept `warehouse`, even in the dashboard.

## Connect a source

An event stream connection to this destination is checked against two rules at `validate` time.

**The source's type must be supported.** Qualtrics accepts `web`, `android`, and `ios` — the types JavaScript, Android, and iOS SDK sources resolve to. Every other source resolves to a type it doesn't accept — an iOS Swift source to `ios_swift`, an Android Kotlin source to `android_kotlin`, and webhook and server-side SDK sources to `cloud` — and reports, for example:

```text
destination 'qualtrics-prod' (type 'qualtrics') does not support source 'my-swift-app': source type 'ios_swift' is not among supported source types: web, android, ios
```

A Reverse ETL connection, whose source resolves to `warehouse`, is refused the same way:

```text
destination 'qualtrics-prod' (type 'qualtrics') does not accept rETL sources: source type 'warehouse' is not among supported source types: web, android, ios
```

**The destination config must carry a `connection_mode` entry for that source type.** This lives on the destination spec, not on the connection spec. Without it:

```text
destination 'qualtrics-prod' config has no 'connection_mode' entry for source type 'web'
```

Qualtrics needs no additional config keys to connect a source of any supported type.

## Secrets

Rudder CLI treats none of Qualtrics' keys as secret.

The dashboard masks `project_id` (**Project ID**). Rudder CLI doesn't treat it as a secret, so it's written to YAML in plain text. On web the project ID is part of the intercept code's URL in the page, so masking it would protect your YAML, not the value itself.
