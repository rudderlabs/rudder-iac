# Webhook (`webhook`)

Streaming destination. RudderStack forwards each event as an HTTP request to an
endpoint you control. Every supported source type connects in cloud mode — there
is no device-mode variant.

In a destination spec:

- `type: webhook`
- `definition_version: 1`

## Example

```yaml
version: rudder/v1
kind: destination
metadata:
  name: webhook
spec:
  id: webhook
  display_name: Order Service Webhook
  type: webhook
  definition_version: 1
  enabled: true
  config:
    webhook_url: "https://hooks.example.com/rudderstack/events"
    webhook_method: POST
    headers:
      - from: Authorization
        to: "{{ .WEBHOOK_AUTH_HEADER }}"
      - from: X-Source
        to: rudderstack

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

#### `webhook_url` — string, required

The endpoint RudderStack sends events to.

Must be a public `http(s)` domain URL. Localhost and ngrok addresses are
rejected, so the endpoint has to be reachable from RudderStack's servers.

#### `webhook_method` — string, default `POST`

The HTTP method used for the request. One of `POST`, `PUT`, `PATCH`, `GET` or
`DELETE`.

#### `headers` — array of objects

Custom headers added to every request RudderStack makes to your endpoint. Each
entry takes:

- `from` — the header name, at most 1000 characters
- `to` — the header value, at most 1000 characters. **Treated as a secret** —
  see [Secrets](#secrets)

## Source types

Every supported source type connects in cloud mode only:

`web` · `android` · `android_kotlin` · `ios` · `ios_swift` · `unity` ·
`react_native` · `flutter` · `cordova` · `cloud`

## Per-source keys

Both keys below are objects keyed by the local source type. A key naming a
source type this destination does not support fails validation.

#### `connection_mode` \* — object

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
destination 'webhook' (type 'webhook') does not support source 'my-source':
source type 'amp' is not among supported source types: android, android_kotlin, ...
```

**The config must carry a `connection_mode` entry for that source type.** This
lives on the destination spec, not the connection spec. Without it:

```
destination 'webhook' config has no 'connection_mode' entry for source type 'web'
```

Webhook requires no additional config keys to connect a source of any type.

## Secrets

`headers.to` is the secret key — that is, the **value** of every custom header,
whatever its name. This is unusual: the secret is a field inside a repeated
block rather than a top-level key, so each header's value is masked
independently.

Header names (`headers.from`) are not secret and are stored in the clear.

Write header values as `{{ .VAR }}` references and supply them at apply time:

```yaml
headers:
  - from: Authorization
    to: "{{ .WEBHOOK_AUTH_HEADER }}"
```

```sh
export RUDDER_WEBHOOK_AUTH_HEADER="Bearer ..."
rudder-cli apply

# or
rudder-cli apply --var-file secrets.vars.yaml
```

`rudder-cli import` writes each header value back as a `{{ .VAR }}` placeholder
rather than its value, since the API does not return secrets. Fill the
placeholders in before the first apply.
