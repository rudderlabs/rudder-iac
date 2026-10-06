# Webhook (`webhook`)

Webhook sends each event, unmodified, to an endpoint you own. For request mapping, authentication modes, or batching, use [HTTP Webhook](../http/README.md) instead.

In a Webhook destination spec:

- `type: webhook`
- `definition_version: 1`

## Sample configuration

```yaml
version: rudder/v1
kind: destination
metadata:
  name: events-webhook-prod
spec:
  id: events-webhook-prod
  display_name: Events Webhook Production
  type: webhook
  definition_version: 1
  enabled: true
  config:
    webhook_url: https://hooks.example.com/rudderstack
    webhook_method: POST
    headers:
      - from: X-Signing-Key
        to: "{{ .WEBHOOK_SIGNING_KEY }}"
      - from: X-Environment
        to: "{{ .WEBHOOK_ENVIRONMENT }}"

    connection_mode:
      web: cloud
      cloud: cloud
    consent_management:
      web:
        - provider: oneTrust
          consents:
            - analytics
```

The above example sets two custom headers, and supplies both values through variables because every header value is a secret — see [Headers](#headers).

## Config keys

`config` accepts only the keys listed below. The [shared config key rules](../README.md#config-key-rules) cover unknown keys, defaults, and immutability.

### Endpoint

#### `webhook_url` — string, required

Endpoint RudderStack sends events to.

- Must be a public domain URL: `http` or `https`, at least one dot-separated label followed by an alphabetic top-level domain, an optional port, and an optional path.
- `localhost` addresses and `<name>.ngrok.io` addresses are rejected.
- A `{{ path || fallback }}` template is accepted in place of a literal.

To change the path per event, use a transformation.

#### `webhook_method` — string, default `POST`

HTTP method of the outgoing request.

- One of `POST`, `PUT`, `PATCH`, `GET`, or `DELETE`.

### Headers

> [!WARNING]
> In this destination, `from` is the header **name** and `to` is its **value**. That's the reverse of [HTTP Webhook's `headers`](../http/README.md#headers--array-of-objects), where `to` is the name.
>
> Swapping them doesn't fail validation — it sends a header named after your value.

#### `headers` — array of objects

Custom headers added to every request. Values that aren't strings are stringified before they're set.

- `from` — header name, for example `content-type`.
- `to` — header value, for example `application/json`. **Secret** — see [Secrets](#secrets).
- Each is at most 1000 characters, and must not contain line breaks. A `{{ path || fallback }}` template is accepted in place of a literal.

RudderStack adds `user-agent: RudderLabs` and `content-type: application/json` to `POST` and `PUT` requests on its own.

```yaml
headers:
  - from: X-Signing-Key
    to: "{{ .WEBHOOK_SIGNING_KEY }}"
```

To set a header from the event itself, use a transformation.

### Per-source keys

Both keys are objects keyed by the local source type — the tokens listed under [Source types](#source-types). A key naming a source type this destination doesn't support fails validation.

#### `connection_mode` — object

Maps each source type you connect to the mode its events reach the endpoint in, using the modes in [Source types](#source-types).

- An entry is required for each source type you connect — see [Connect a source](#connect-a-source).

```yaml
connection_mode:
  web: cloud
  cloud: cloud
```

#### `consent_management` — object

Consent provider configuration per source type. The entry shape, accepted providers, and the rules on `resolution_strategy` and `consents` are shared across all destination types — see [Consent management](../common/README.md).

## Source types

Webhook accepts events from these source types in the mentioned connection modes:

| Source type | Connection mode |
| :-----| :-----|
| `android` | `cloud` |
| `android_kotlin` | `cloud` |
| `ios` | `cloud` |
| `ios_swift` | `cloud` |
| `web` | `cloud` |
| `unity` | `cloud` |
| `cloud` | `cloud` |
| `react_native` | `cloud` |
| `flutter` | `cloud` |
| `cordova` | `cloud` |
| `warehouse` | `cloud` |

Every source type is `cloud` only — events reach the endpoint from RudderStack's servers, never in device mode.

`warehouse` is the token a Reverse ETL source resolves to — see [Source types](../README.md#source-types).

> [!NOTE]
> The dashboard additionally offers Webhook to AMP and Shopify sources. Rudder CLI doesn't manage those connections, so `amp` and `shopify` are invalid here.

## Connect a source

An event stream connection to this destination is checked against two rules at `validate` time.

**The source's type must be supported.** A source's type resolves to one of the tokens above before the check — a JavaScript source resolves to `web`, and webhook and server-side SDK sources resolve to `cloud`. An unsupported type reports:

```text
destination 'events-webhook-prod' (type 'webhook') does not support source 'my-source':
source type 'amp' is not among supported source types: android, android_kotlin, ...
```

**The destination config must carry a `connection_mode` entry for that source type.** This lives on the destination spec, not on the connection spec. Without it:

```text
destination 'events-webhook-prod' config has no 'connection_mode' entry for source type 'web'
```

Webhook needs no additional config keys to connect a source of any type.

A Reverse ETL connection reaches this destination as source type `warehouse` and is checked against the same rules, so the config needs a `connection_mode.warehouse: cloud` entry for it.

## Secrets

Every `headers` entry's `to` value — the header value — is secret. Write each as a `{{ .VAR }}` reference and supply the value at apply time:

```yaml
config:
  headers:
    - from: X-Signing-Key
      to: "{{ .WEBHOOK_SIGNING_KEY }}"
```

```bash
export RUDDER_WEBHOOK_SIGNING_KEY="..."
rudder-cli apply

# or
rudder-cli apply --var-file secrets.vars.yaml
```

Note that:

- Header values are masked as a group, so even a constant like `application/json` is treated as a secret.
- The YAML that `rudder-cli import` writes may or may not include secret keys. Before you apply, make sure every secret key your configuration needs is present and populated through variable substitution.
- Header names (`from`) aren't secret and are imported as-is.
