# HTTP Webhook (`http`)

HTTP Webhook sends each event to an HTTP endpoint you own. You choose the method, body format, authentication, and — when the default passthrough doesn't suit the endpoint — how RudderStack maps the event onto the outgoing request.

In an HTTP Webhook destination spec:

- `type: http`
- `definition_version: 1`

## Sample configuration

```yaml
version: rudder/v1
kind: destination
metadata:
  name: orders-webhook-prod
spec:
  id: orders-webhook-prod
  display_name: Orders Webhook Production
  type: http
  definition_version: 1
  enabled: true
  config:
    api_url: https://api.example.com/v1/events
    method: POST
    format: JSON

    auth: apiKeyAuth
    api_key_name: "{{ .WEBHOOK_API_KEY_NAME }}"
    api_key_value: "{{ .WEBHOOK_API_KEY }}"

    is_default_mapping: false
    properties_mapping:
      - to: $.eventName
        from: $.event
      - to: $.customer.id
        from: $.userId
    query_params:
      - to: source
        from: rudderstack
    headers:
      - to: X-Request-Source
        from: "{{ .WEBHOOK_HEADER_VALUE }}"
    path_params:
      - path: $.properties.accountId

    is_batching_enabled: true
    max_batch_size: "50"

    event_filtering:
      whitelist:
        - Order Completed
        - Order Refunded

    connection_mode:
      web: cloud
      cloud: cloud
    consent_management:
      web:
        - provider: oneTrust
          consents:
            - analytics
```

The above example uses API key authentication and custom request mapping. `xml_root_key` is omitted because it applies only when `format` is `XML`. Several keys apply only in certain combinations — see [Key dependencies](#key-dependencies).

## Config keys

`config` accepts only the keys listed below. The [shared config key rules](../README.md#config-key-rules) cover unknown keys, defaults, and immutability.

### Key dependencies

Several keys only take effect in combination with another key's value. The CLI enforces one of these — `max_batch_size` — and accepts the rest whatever the other key says, so a key that doesn't apply is stored and ignored rather than rejected.

| Key | Applies when |
| :-----| :-----|
| `username`, `password` | `auth` is `basicAuth` |
| `bearer_token` | `auth` is `bearerTokenAuth` |
| `api_key_name`, `api_key_value` | `auth` is `apiKeyAuth` |
| `xml_root_key` | `format` is `XML` |
| `properties_mapping` | `is_default_mapping` is `false` |
| `is_batching_enabled` | `format` is `JSON` |
| `max_batch_size` | `is_batching_enabled` is `true` **and** `format` is `JSON` |

### Endpoint

#### `api_url` — string, required

Endpoint RudderStack sends events to. Both `http` and `https` are accepted; an `https` endpoint needs a valid TLS certificate for delivery to succeed.

- Must be a public domain URL: a scheme, at least one dot-separated label followed by an alphabetic top-level domain, an optional port, and an optional path.
- A bare hostname, a bare IP address, `localhost` in any form, and `ngrok.io` addresses are all rejected.

To append path or query segments built from the event, use `path_params` and `query_params` rather than writing them into this value.

#### `method` — string, required

HTTP method of the outgoing request: `POST`, `PUT`, `PATCH`, `GET`, or `DELETE`.

- The dashboard defaults this to `POST`. Rudder CLI requires it explicitly.
- A spec that omits this key fails validation.

#### `format` — string, required

Body format of the outgoing request: `JSON`, `XML`, or `FORM`.

- The dashboard defaults this to `JSON`. Rudder CLI requires it explicitly.
- A spec that omits this key fails validation.
- The value also decides whether `xml_root_key` and `is_batching_enabled` apply.

#### `xml_root_key` — string

Root key wrapping every mapped field in the request body.

- Applies when `format` is `XML`.
- At most 100 characters, and must not contain line breaks.

### Authentication

#### `auth` — string, required

Authentication method for the request: `noAuth`, `basicAuth`, `bearerTokenAuth`, or `apiKeyAuth`.

- The dashboard defaults this to `noAuth`. Rudder CLI requires it explicitly.
- A spec that omits this key fails validation.
- The value decides which credential keys below are required.

#### `username` — string, required, secret

Username for basic authentication.

- Required when `auth` is `basicAuth`.
- 1 to 100 characters, and must not contain line breaks.

#### `password` — string, required, secret

Password for basic authentication.

- Required when `auth` is `basicAuth`.
- At most 100 characters, and must not contain line breaks.

#### `bearer_token` — string, required, secret

Token sent in the `Authorization` header.

- Required when `auth` is `bearerTokenAuth`.
- 1 to 2048 characters, and must not contain line breaks.

#### `api_key_name` — string, required, secret

Name of the header carrying the API key — for example `X-Api-Key`, which is what the dashboard prefills.

- Required when `auth` is `apiKeyAuth`.
- 1 to 100 characters, with no whitespace.

Rudder CLI treats the header **name** as a secret as well as its value, so write it as a `{{ .VAR }}` reference — see [Secrets](#secrets).

#### `api_key_value` — string, required, secret

Value of the API key header.

- Required when `auth` is `apiKeyAuth`.
- 1 to 100 characters, and must not contain line breaks.

### Request mapping

By default the source event is sent through unmodified. The keys in this group either turn that off and reshape the body, or add to the request URL and headers — those additions apply either way.

Each mapping entry pairs a `from` (where the value comes from) with a `to` (where it lands on the outgoing request). A `from` is a JSONPath into the source event, such as `$.properties.orderId`, or a bare constant such as `rudderstack`. A quoted constant like `"EUR"` is not valid.

> [!WARNING]
> If you define more than one mapping for the same key, only the first is used and the rest are ignored.

#### `is_default_mapping` — boolean, default `true`

Send the source event payload as it is, without applying `properties_mapping`. This is what the dashboard calls **Send the event payload as is**.

Set it to `false` to shape the body yourself. When `format` is `XML` and this stays `true`, the unmodified payload is sent under `xml_root_key`.

#### `properties_mapping` — array of objects

Maps fields of the source event onto the request body.

- Applies when `is_default_mapping` is `false`.
- `to` — JSONPath of the field on the outgoing request. Must be a JSONPath or empty; a bare token like `properties.value` is rejected.
- `from` — JSONPath into the source event, or a constant of up to 100 characters.

```yaml
properties_mapping:
  - to: $.messageType
    from: $.type
  - to: $.customer.firstName
    from: $.traits.firstName
```

#### `query_params` — array of objects

Query parameters appended to `api_url`.

- `to` — parameter name. Must be a plain token of up to 100 characters, not a JSONPath.
- `from` — JSONPath into the source event, or a constant of up to 100 characters.

```yaml
query_params:
  - to: source
    from: rudderstack
  - to: order_id
    from: $.properties.orderId
```

#### `headers` — array of objects

Headers added to the outgoing request.

- `to` — header name. Must be a plain token of up to 100 characters, not a JSONPath.
- `from` — JSONPath into the source event, or a constant of up to 100 characters. **Secret** — see [Secrets](#secrets).

```yaml
headers:
  - to: X-Request-Source
    from: rudderstack
```

#### `path_params` — array of objects

Path segments appended to `api_url`, in the order listed.

- `path` — JSONPath into the source event, or a plain token of up to 100 characters. A quoted value like `"order"` is rejected.

```yaml
path_params:
  - path: accounts
  - path: $.properties.accountId
```

### Batching

#### `is_batching_enabled` — boolean, default `false`

Collect events and send them as a JSON array — `[{event1},{event2},...]` — instead of one request per event.

- Applies when `format` is `JSON`.
- A batch is sent when it reaches `max_batch_size`, or after five seconds, whichever comes first.

#### `max_batch_size` — string, required

Largest number of events in one batch, written as a string rather than a number.

- Required when `is_batching_enabled` is `true`.
- Applies when `format` is `JSON`.
- A string integer from `1` to `100`.

```yaml
is_batching_enabled: true
max_batch_size: "50"
```

### Event filtering

#### `event_filtering` — object

Restricts which `track` events reach the destination, by event name.

- `whitelist` — array of event names to allow; every other `track` event is dropped.
- `blacklist` — array of event names to drop; every other `track` event is allowed.
- The two are mutually exclusive, and this the CLI does enforce — setting both fails validation. Omit the block entirely to filter nothing.
- Each name is at most 100 characters, or a `{{ path || fallback }}` template.

```yaml
event_filtering:
  whitelist:
    - Order Completed
    - Order Refunded
```

> [!WARNING]
> Client-side event filtering is applied by the device SDK, so it covers destinations reached in device mode. HTTP Webhook connects in `cloud` mode only, for every source type, which is why the dashboard doesn't offer these controls for it. Rudder CLI accepts and sends the keys regardless.
>
> Filter events with a transformation instead.

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

HTTP Webhook accepts events from these source types in the mentioned connection modes:

| Source type | Connection mode |
| :-----| :-----|
| `android` | `cloud` |
| `android_kotlin` | `cloud` |
| `ios` | `cloud` |
| `ios_swift` | `cloud` |
| `web` | `cloud` |
| `unity` | `cloud` |
| `react_native` | `cloud` |
| `flutter` | `cloud` |
| `cordova` | `cloud` |
| `cloud` | `cloud` |
| `warehouse` | `cloud` |

Every source type is `cloud` only — events reach the endpoint from RudderStack's servers, never in device mode.

`warehouse` is the token a Reverse ETL source resolves to — see [Source types](../README.md#source-types).

> [!NOTE]
> The dashboard additionally offers HTTP Webhook to AMP and Shopify sources. Rudder CLI doesn't manage those connections, so `amp` and `shopify` are invalid here.

## Connect a source

An event stream connection to this destination is checked against two rules at `validate` time.

**The source's type must be supported.** A source's type resolves to one of the tokens above before the check — a JavaScript source resolves to `web`, and webhook and server-side SDK sources resolve to `cloud`. An unsupported type reports:

```text
destination 'orders-webhook-prod' (type 'http') does not support source 'my-source':
source type 'amp' is not among supported source types: android, android_kotlin, ...
```

**The destination config must carry a `connection_mode` entry for that source type.** This lives on the destination spec, not on the connection spec. Without it:

```text
destination 'orders-webhook-prod' config has no 'connection_mode' entry for source type 'web'
```

HTTP Webhook needs no additional config keys to connect a source of any type.

A Reverse ETL connection reaches this destination as source type `warehouse` and is checked against the same rules, so the config needs a `connection_mode.warehouse: cloud` entry for it.

## Secrets

Rudder CLI treats six keys as secrets: `username`, `password`, `bearer_token`, `api_key_name`, `api_key_value`, and every `headers` entry's `from` value. Write each as a `{{ .VAR }}` reference and supply the value at apply time:

```yaml
config:
  auth: apiKeyAuth
  api_key_name: "{{ .WEBHOOK_API_KEY_NAME }}"
  api_key_value: "{{ .WEBHOOK_API_KEY }}"
  headers:
    - to: X-Request-Source
      from: "{{ .WEBHOOK_HEADER_VALUE }}"
```

```bash
export RUDDER_WEBHOOK_API_KEY="..."
rudder-cli apply

# or
rudder-cli apply --var-file secrets.vars.yaml
```

Note that:

- `api_key_name` is secret even though it's the header name, not the key itself.
- Every header value is secret, including constants such as `application/json`.
- The YAML that `rudder-cli import` writes may or may not include secret keys. Before you apply, make sure every secret key your configuration needs is present and populated through variable substitution.
