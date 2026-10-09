# Google Tag Manager (`gtm`)

Google Tag Manager is a web device mode tag management destination. RudderStack's JavaScript SDK loads your Tag Manager container in the browser and pushes `track`, `page`, and `identify` calls to the container's data layer, where your tags and triggers act on them.

In a Google Tag Manager destination spec:

- `type: gtm`
- `definition_version: 1`

> [!NOTE]
> `gtm` is an [unverified destination type](../README.md#unverified-destination-types). Rudder CLI registers it only when the `unverifiedDestinations` experimental flag is on.

## Sample configuration

```yaml
version: rudder/v1
kind: destination
metadata:
  name: gtm-staging
spec:
  id: gtm-staging
  display_name: Google Tag Manager Staging
  type: gtm
  definition_version: 1
  enabled: true
  config:
    container_id: GTM-ABC1234
    server_url: https://tags.example.com
    environment_id: env-5
    authorization_token: d7bhUC_Zj4q1FRT5KWOuQ

    event_filtering:
      blacklist:
        - Credit Card Added

    connection_mode:
      web: device
    consent_management:
      web:
        - provider: oneTrust
          consents:
            - analytics
```

The above example loads the container's staging environment from a custom domain — see [Environments](#environments) — and keeps `Credit Card Added` events out of the data layer.

## Config keys

`config` accepts only the keys listed below. The [shared config key rules](../README.md#config-key-rules) cover unknown keys, defaults, and immutability.

> [!NOTE]
> Google Tag Manager runs only in web device mode. Every key on this page configures how the SDK loads the container, or which events it pushes to the data layer:
>
> - `track` — pushed under its event name, with `userId`, `anonymousId`, `traits`, `messageId`, and the event's properties.
> - `page` — pushed as `Viewed <category> <name> page`, `Viewed <name> page` when the call has no category, or `Viewed a Page` when it has no name, with the same fields.
> - `identify` — pushed as a `traits` object, with no event name.

### Container

#### `container_id` — string, required

ID of the Tag Manager container to load, in the form `GTM-XXXXXXX`, from the **Admin** section of Tag Manager.

- At most 100 characters, and must not contain line breaks.
- A `{{ path || fallback }}` template is accepted in place of a literal.

#### `server_url` — string

Domain that serves the container's `gtm.js` instead of `https://www.googletagmanager.com` — for example, a server-side Tag Manager container on your own domain. The SDK loads `<server_url>/gtm.js?id=<container_id>`. The dashboard calls it **Custom Domain URL**.

- Must be a domain URL, and any value containing `.ngrok.io` is rejected.
- A `{{ path || fallback }}` template is accepted in place of a literal.
- Write the scheme, and leave off the trailing slash. Rudder CLI accepts a value without `https://`, but the SDK uses the value as is, so `tags.example.com` alone loads the script relative to the current page.

```yaml
server_url: https://tags.example.com
```

### Environments

`environment_id` and `authorization_token` load a Tag Manager environment, such as staging, instead of the container's live version. Both come from the environment's snippet in Tag Manager — **Environments** > **Actions** > **Get Snippet** — as its `gtm_preview` and `gtm_auth` values. Leave both unset to load the live version.

#### `environment_id` — string

ID of the environment to load — the `gtm_preview` value, such as `env-5`.

- At most 100 characters, and must not contain line breaks.
- Templates aren't accepted in place of a literal; a template is measured as text against the same limit.
- Set it together with `authorization_token`. Rudder CLI accepts either one without the other.

#### `authorization_token` — string

Authorization token of that environment — the `gtm_auth` value.

- At most 100 characters, and must not contain line breaks.
- Templates aren't accepted in place of a literal; a template is measured as text against the same limit.
- Not secret — the SDK adds it to the `gtm.js` URL the browser requests.

### Event filtering

#### `event_filtering` — object

Restricts which `track` events the SDK pushes to the data layer, by event name. `page` and `identify` calls aren't filtered.

- `whitelist` — array of event names to allow; every other `track` event is dropped.
- `blacklist` — array of event names to drop; every other `track` event is allowed.
- The two are mutually exclusive, and Rudder CLI enforces it — setting both fails validation.
- Each name is at most 100 characters, or a `{{ path || fallback }}` template.
- Omit the block entirely to filter nothing.

```yaml
event_filtering:
  whitelist:
    - Order Completed
    - Product Added
```

### Per-source keys

Both keys are objects keyed by the local source type — the tokens listed under [Source types](#source-types). A key naming a source type this destination doesn't support fails validation.

#### `connection_mode` — object

Maps each source type you connect to the mode its events reach Google Tag Manager in. Google Tag Manager accepts only `web: device`.

- An entry is required for each source type you connect — see [Connect a source](#connect-a-source).

```yaml
connection_mode:
  web: device
```

#### `consent_management` — object

Consent provider configuration per source type. The entry shape, accepted providers, and the rules on `resolution_strategy` and `consents` are shared across all destination types — see [Consent management](../common/README.md).

## Source types

Google Tag Manager accepts events from these source types in the mentioned connection modes:

| Source type | Connection mode |
| :-----| :-----|
| `web` | `device` |

Google Tag Manager accepts only web sources, and only in `device` mode — the SDK loads the container in the browser, and no events pass through RudderStack's servers.

Google Tag Manager doesn't accept `warehouse`, even in the dashboard.

## Connect a source

An event stream connection to this destination is checked against two rules at `validate` time.

**The source's type must be supported.** Google Tag Manager accepts only `web`, the type a JavaScript source resolves to. Every other source resolves to a type it doesn't accept — an iOS source to `ios`, an Android source to `android`, and webhook and server-side SDK sources to `cloud` — and reports, for example:

```text
destination 'gtm-staging' (type 'gtm') does not support source 'my-ios-app': source type 'ios' is not among supported source types: web
```

A Reverse ETL connection, whose source resolves to `warehouse`, is refused the same way:

```text
destination 'gtm-staging' (type 'gtm') does not accept rETL sources: source type 'warehouse' is not among supported source types: web
```

**The destination config must carry a `connection_mode` entry for that source type.** This lives on the destination spec, not on the connection spec. Without it:

```text
destination 'gtm-staging' config has no 'connection_mode' entry for source type 'web'
```

Google Tag Manager needs no additional config keys to connect a web source.

## Secrets

Google Tag Manager has no secret keys. Rudder CLI treats none of its keys as secret, and the dashboard masks none of them, so `import` writes every value to YAML in plain text — including `authorization_token`, which the browser sends in its request for `gtm.js` anyway.
