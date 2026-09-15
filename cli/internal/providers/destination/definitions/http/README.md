# HTTP (`http`)

Streaming destination. RudderStack sends each event as an HTTP request to an
endpoint you define, with full control over the method, body format, headers,
query parameters and payload mapping. Every supported source type connects in
cloud mode — there is no device-mode variant.

Where [Webhook](../webhook/README.md) forwards events to a URL with optional
static headers, this destination lets you reshape the request itself.

In a destination spec:

- `type: http`
- `definition_version: 1`

## Example

```yaml
version: rudder/v1
kind: destination
metadata:
  name: http
spec:
  id: http
  display_name: Orders API
  type: http
  definition_version: 1
  enabled: true
  config:
    api_url: "https://api.example.com/v1/events"
    method: POST
    format: JSON

    auth: bearerTokenAuth
    bearer_token: "{{ .HTTP_BEARER_TOKEN }}"

    is_default_mapping: false
    properties_mapping:
      - to: $.user.id
        from: $.userId
      - to: $.order.total
        from: $.properties.revenue
      - to: $.source
        from: rudderstack

    query_params:
      - to: tenant
        from: acme
      - to: event
        from: $.event

    headers:
      - to: X-Request-Id
        from: $.messageId
      - to: X-Source
        from: rudderstack

    path_params:
      - path: $.properties.orderId

    is_batching_enabled: true
    max_batch_size: "50"

    event_filtering:
      whitelist:
        - Order Completed
        - Product Viewed

    connection_mode:
      cloud: cloud
      web: cloud
    consent_management:
      web:
        - provider: oneTrust
          consents:
            - analytics
```

## Config keys

`config` accepts only the keys documented here — anything else fails validation
with `unknown config field "<key>"`.

Keys that declare a default are filled in before the spec enters the resource
graph, matching what the backend stores, so omitting one is equivalent to
writing its default and does not produce a permanent diff.

A `*` after a key name marks a description written without a Terraform provider
source to draw on. Those need a closer review pass; the markers come out once the
wording is confirmed.

### Request

#### `api_url` — string, required

The base URL for your HTTP endpoint.

Must be an `http(s)` URL with a resolvable domain and an optional port and path.
Localhost and ngrok addresses are rejected, so the endpoint has to be reachable
from RudderStack's servers.

#### `method` — string, required

The HTTP method used when sending requests. One of `POST`, `PUT`, `PATCH`, `GET`
or `DELETE`.

#### `format` — string, required

The body format for the outgoing request. One of `JSON`, `XML` or `FORM`.

#### `xml_root_key` — string

The XML root key used as a common prefix for mapped fields. At most 100
characters.

### Authentication

`auth` selects the scheme and decides which of the keys below are required.

#### `auth` — string, required

The authentication method used for the request. One of `noAuth`, `basicAuth`,
`bearerTokenAuth` or `apiKeyAuth`.

#### `username` — string

Username for basic authentication. **Required when `auth` is `basicAuth`.** 1 to
100 characters, no whitespace.

#### `password` — string, secret

Password for basic authentication. **Required when `auth` is `basicAuth`.** At
most 100 characters.

#### `bearer_token` — string, secret

The bearer token used for authorization. **Required when `auth` is
`bearerTokenAuth`.** 1 to 2048 characters, no whitespace.

#### `api_key_name` — string

The header name carrying the API key. **Required when `auth` is `apiKeyAuth`.**
1 to 100 characters.

#### `api_key_value` — string, secret

The API key value. **Required when `auth` is `apiKeyAuth`.** 1 to 100
characters, no whitespace.

### Payload mapping

The four mapping blocks below share a convention: a `from` value is either a
**JSON path** into the RudderStack payload (`$.userId`,
`$.properties.revenue`) or a **literal constant** (`rudderstack`). Which forms
are accepted differs slightly per block, since header and query values allow
different character sets than payload fields.

#### `is_default_mapping` — boolean, default `true`

Send the event payload as-is. Leave it on to forward RudderStack's payload
unchanged; turn it off to shape the request with `properties_mapping`.

#### `properties_mapping` — array of objects

Map the outgoing request payload using JSON path keys and values from the
RudderStack payload or constants. Each entry takes:

- `to` — a JSON path into the outgoing payload, written with the `$.` prefix
  (`$.user.id`). A plain key is rejected
- `from` — a JSON path into the RudderStack payload, or a constant

#### `query_params` — array of objects

Map query parameter keys to values from the RudderStack payload or constants.
Each entry takes:

- `to` — the query parameter name, a plain token rather than a JSON path
- `from` — a JSON path into the RudderStack payload, or a constant

#### `headers` — array of objects

Build custom request headers using constants or values from the RudderStack
payload. Each entry takes:

- `to` — the header name
- `from` — a JSON path into the RudderStack payload, or a constant

#### `path_params` — array of objects

Path parameters, appended to `api_url` in the order listed. Each entry takes a
`path`, which is either a JSON path into the RudderStack payload or a constant.

### Batching and filtering

#### `is_batching_enabled` — boolean, default `false`

Batch JSON payloads into a single request rather than sending one request per
event.

#### `max_batch_size` — string

The maximum number of events per batch. **Required when `is_batching_enabled` is
`true`.** Written as a digit string, from `1` to `100`.

#### `event_filtering` — object

Choose whether to allowlist or denylist events. Exactly one of the two lists may
be set — declaring both fails validation.

- `whitelist` — event names to allowlist
- `blacklist` — event names to denylist

Each entry is at most 100 characters. Omit the block to send every event.

## Source types

Every supported source type connects in cloud mode only:

`web` · `android` · `android_kotlin` · `ios` · `ios_swift` · `unity` ·
`react_native` · `flutter` · `cordova` · `cloud`

## Per-source keys

Both keys below are objects keyed by the local source type. A key naming a
source type this destination does not support fails validation.

#### `connection_mode` — object

Selects the mode per source type. Every supported type accepts `cloud` only, so
each entry's value is `cloud`:

```yaml
connection_mode:
  web: cloud
  cloud: cloud
```

An entry is required for each source type you connect — see
[Connecting a source](#connecting-a-source).

#### `consent_management` — object

Specify consent configuration data for multiple providers, per source type. The
entry shape, accepted providers, and the rules on `resolution_strategy` and
`consents` are shared across all destinations and documented in
[../common/README.md](../common/README.md).

## Connecting a source

An event stream connection to this destination is checked against two rules at
`validate` time.

**The source's type must be supported.** A source's type is mapped to one of the
tokens above first — a JavaScript source resolves to `web`, and webhook and
server-side SDK sources resolve to `cloud`. An unsupported type reports:

```
destination 'http' (type 'http') does not support source 'my-source':
source type 'amp' is not among supported source types: android, android_kotlin, ...
```

**The config must carry a `connection_mode` entry for that source type.** This
lives on the destination spec, not the connection spec. Without it:

```
destination 'http' config has no 'connection_mode' entry for source type 'web'
```

HTTP requires no additional config keys to connect a source of any type.

## Secrets

`password`, `bearer_token` and `api_key_value` are the secret keys — one per
authentication scheme, so at most one applies to any given destination. Write
whichever you use as a `{{ .VAR }}` reference and supply the value at apply time:

```yaml
bearer_token: "{{ .HTTP_BEARER_TOKEN }}"
```

```sh
export RUDDER_HTTP_BEARER_TOKEN=...
rudder-cli apply

# or
rudder-cli apply --var-file secrets.vars.yaml
```

`rudder-cli import` writes each back as a `{{ .VAR }}` placeholder rather than
its value, since the API does not return secrets. Fill the placeholders in
before the first apply.
