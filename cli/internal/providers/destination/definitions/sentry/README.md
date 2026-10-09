# Sentry (`sentry`)

Sentry is a web device mode error monitoring destination. RudderStack's JavaScript SDK loads Sentry's browser SDK, which reports your site's JavaScript errors to Sentry, and sets the Sentry user from each `identify` call so errors carry who hit them.

In a Sentry destination spec:

- `type: sentry`
- `definition_version: 1`

> [!NOTE]
> `sentry` is an [unverified destination type](../README.md#unverified-destination-types). Rudder CLI registers it only when the `unverifiedDestinations` experimental flag is on.

## Sample configuration

```yaml
version: rudder/v1
kind: destination
metadata:
  name: sentry-prod
spec:
  id: sentry-prod
  display_name: Sentry Production
  type: sentry
  definition_version: 1
  enabled: true
  config:
    dsn: https://4f8b2c1d9e7a4b3c8d2e1f0a9b8c7d6e@o991473.ingest.sentry.io/5948763
    environment: production
    custom_version_property: APP_VERSION
    release: web-app@2.3.12
    server_name: web-frontend
    logger: rudderstack
    debug_mode: false

    ignore_errors:
      - ResizeObserver loop limit exceeded
    allow_urls:
      - https://www.example.com
    deny_urls:
      - chrome-extension://
    include_paths:
      - example\.com/assets/

    connection_mode:
      web: device
    consent_management:
      web:
        - provider: oneTrust
          consents:
            - analytics
```

The above example reports the version in `window.APP_VERSION` as the release, falling back to `web-app@2.3.12` when the page doesn't define it — see [Error context](#error-context).

## Config keys

`config` accepts only the keys listed below. The [shared config key rules](../README.md#config-key-rules) cover unknown keys, defaults, and immutability.

> [!NOTE]
> Sentry runs only in web device mode, and acts only on `identify` calls, which set the user attached to subsequent errors. Every key on this page configures Sentry's browser SDK.

### Connection

#### `dsn` — string, required

Public DSN of your Sentry project — the URL Sentry's browser SDK reports errors to, from the project's **Client Keys (DSN)** settings. The dashboard calls it **Public DSN**.

- At most 300 characters, and must not contain line breaks.
- A `{{ path || fallback }}` template is accepted in place of a literal.
- The dashboard masks this field, but Rudder CLI doesn't treat it as a secret — see [Secrets](#secrets).

### Error context

These keys label the errors Sentry reports. None of them is validated locally.

#### `environment` — string

Environment Sentry files errors under, such as `production` or `staging`.

- Sentry silently drops an environment that contains a forward slash, a space, or a line break, or that is `None`.

#### `custom_version_property` — string

Name of a global `window` variable holding your app's version. When the page defines it, its value is sent as the release, taking precedence over `release`. The dashboard calls it **Set Release By Property**.

- A top-level variable name, not a dotted path: `APP_VERSION` reads `window.APP_VERSION`.

```yaml
custom_version_property: APP_VERSION
release: web-app@2.3.12
```

#### `release` — string

Release, or app version, Sentry attaches to each error, such as `web-app@2.3.12`.

- Used when `custom_version_property` is unset, or names a variable the page doesn't define.
- Sentry discards session data that has no release, so set `release`, `custom_version_property`, or both.

#### `server_name` — string

Server name Sentry attaches to each error, identifying the host the client runs on.

#### `logger` — string

Value of the `logger` tag set on every error Sentry reports.

### Error filtering

These keys decide which errors Sentry reports. Each entry is matched as plain text, not as a regular expression: an entry matches any message or URL that contains it.

#### `ignore_errors` — array of strings

Error messages Sentry doesn't report. An error whose message contains an entry is dropped.

- Each entry is at most 100 characters, or a `{{ path || fallback }}` template.

#### `allow_urls` — array of strings

Script URLs Sentry reports errors from. When the list is set, only errors raised by a script whose URL contains an entry are reported.

- Each entry must not contain line breaks, and any entry containing `.ngrok.io` is rejected.
- A `{{ path || fallback }}` template is accepted in place of a literal.

#### `deny_urls` — array of strings

Script URLs whose errors Sentry doesn't report — for example, browser extensions or third-party scripts. An error raised by a script whose URL contains an entry is dropped.

- Accepted on the same terms as `allow_urls`.

```yaml
deny_urls:
  - chrome-extension://
  - cdn.thirdparty.example
```

### Stack traces

#### `include_paths` — array of strings

Regular expressions matching the URLs of your app's own scripts. A stack frame whose file URL matches an entry is marked as your app's code, and every other frame as third-party, which Sentry collapses in the stack trace.

- An entry that isn't a valid regular expression is skipped.
- Each entry is at most 100 characters, or a `{{ path || fallback }}` template.

```yaml
include_paths:
  - example\.com/assets/
```

### Debugging

#### `debug_mode` — boolean, default `false`

Turn on the Sentry SDK's debug mode, which logs the SDK's diagnostic messages to the browser console. It doesn't stop errors from being sent to Sentry.

### Event filtering

> [!WARNING]
> Client-side event filtering applies only to `track` events, and Sentry acts only on `identify` calls, so this block has no effect on what Sentry receives. Rudder CLI accepts it all the same.

#### `event_filtering` — object

Restricts which `track` events the SDK passes to Sentry, by event name.

- `whitelist` — array of event names to allow; every other `track` event is dropped.
- `blacklist` — array of event names to drop; every other `track` event is allowed.
- The two are mutually exclusive, and Rudder CLI enforces it — setting both fails validation.
- Each name is at most 100 characters, or a `{{ path || fallback }}` template.

### Per-source keys

Both keys are objects keyed by the local source type — the tokens listed under [Source types](#source-types). A key naming a source type this destination doesn't support fails validation.

#### `connection_mode` — object

Maps each source type you connect to the mode its events reach Sentry in. Sentry accepts only `web: device`.

- An entry is required for each source type you connect — see [Connect a source](#connect-a-source).

```yaml
connection_mode:
  web: device
```

#### `consent_management` — object

Consent provider configuration per source type. The entry shape, accepted providers, and the rules on `resolution_strategy` and `consents` are shared across all destination types — see [Consent management](../common/README.md).

## Source types

Sentry accepts events from these source types in the mentioned connection modes:

| Source type | Connection mode |
| :-----| :-----|
| `web` | `device` |

Sentry accepts only web sources, and only in `device` mode — the SDK loads Sentry's browser SDK, and no events pass through RudderStack's servers.

Sentry doesn't accept `warehouse`, even in the dashboard.

## Connect a source

An event stream connection to this destination is checked against two rules at `validate` time.

**The source's type must be supported.** Sentry accepts only `web`, the type a JavaScript source resolves to. Every other source resolves to a type it doesn't accept — an iOS source to `ios`, an Android source to `android`, and webhook and server-side SDK sources to `cloud` — and reports, for example:

```text
destination 'sentry-prod' (type 'sentry') does not support source 'my-ios-app': source type 'ios' is not among supported source types: web
```

A Reverse ETL connection, whose source resolves to `warehouse`, is refused the same way:

```text
destination 'sentry-prod' (type 'sentry') does not accept rETL sources: source type 'warehouse' is not among supported source types: web
```

**The destination config must carry a `connection_mode` entry for that source type.** This lives on the destination spec, not on the connection spec. Without it:

```text
destination 'sentry-prod' config has no 'connection_mode' entry for source type 'web'
```

Sentry needs no additional config keys to connect a web source.

## Secrets

Rudder CLI treats none of Sentry's keys as secret.

The dashboard masks `dsn` (**Public DSN**). Rudder CLI doesn't treat it as a secret, so it's written to YAML in plain text. The DSN is embedded in the page's JavaScript in any case, so masking it would protect your YAML, not the value itself.
