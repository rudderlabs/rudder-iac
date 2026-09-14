# HubSpot (`hs`)

Streaming destination. HubSpot receives events either from RudderStack's servers
(cloud mode) or directly from the HubSpot tracking script loaded on your site
(device mode, web only).

In a destination spec:

- `type: hs`
- `definition_version: 1`

## Modes

The mode a source uses is set per source type in `config.connection_mode`. Only
`web` accepts `device`; every other source type is cloud only — see
[Source types](#source-types).

Each key below is badged with where it applies:

- `cloud` — events sent from RudderStack's servers
- `device` — events sent by the HubSpot tracking script

A key is accepted by `validate` whatever your sources use; the badges tell you
where the setting takes effect.

## Example

```yaml
version: rudder/v1
kind: destination
metadata:
  name: hubspot
spec:
  id: hubspot
  display_name: HubSpot
  type: hs
  definition_version: 1
  enabled: true
  config:
    api_version: newApi
    access_token: "{{ .HUBSPOT_ACCESS_TOKEN }}"
    hub_id: "12345678"
    lookup_field: email
    do_association: true

    hubspot_events:
      - rs_event_name: Order Completed
        hubspot_event_name: pe12345678_order_completed
        event_properties:
          - from: revenue
            to: order_value
          - from: currency
            to: order_currency

    event_filtering:
      whitelist:
        - Order Completed
        - Product Viewed

    connection_mode:
      web: device
      cloud: cloud
    consent_management:
      web:
        - provider: oneTrust
          consents:
            - marketing
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

### Connection

#### `api_version` — string, required
`cloud`

The HubSpot API version to use. One of `newApi` (v3) or `legacyApi`.

#### `access_token` — string, required, secret
`cloud`

Your HubSpot private app access token. At most 100 characters.

Unlike most string keys in this destination, this one does not accept a
`{{ path || fallback }}` template. Supply it as a `{{ .VAR }}` reference, which
is substituted before validation runs — see [Secrets](#secrets).

#### `hub_id` — string
`cloud` `device` · web

Your HubSpot Hub ID, shown under your account name. At most 100 characters.

#### `lookup_field` — string
`cloud`

The HubSpot property name used to look up an existing record when upserting.
**Required when `api_version` is `newApi`.** At most 100 characters.

### Objects and events

#### `do_association` — boolean, default `false`

Create associations between object records.

#### `hubspot_events` — array of objects
`cloud`

Map RudderStack event names to HubSpot custom behavioural events. Each entry
takes:

- `rs_event_name` — the RudderStack event name, at most 100 characters
- `hubspot_event_name` — the HubSpot custom behavioural event name, at most 100
  characters
- `event_properties` — an optional array of `from` / `to` pairs mapping
  RudderStack event property names to HubSpot event property names, each at most
  100 characters

#### `event_filtering` — object
`device` · web

Client-side event filtering. Exactly one of the two lists may be set — declaring
both fails validation.

- `whitelist` — event names to allowlist
- `blacklist` — event names to denylist

Each entry is at most 100 characters. Omit the block to send every event.

## Source types

| Source type | Modes |
| --- | --- |
| `web` | cloud, device |
| `android` | cloud |
| `android_kotlin` | cloud |
| `ios` | cloud |
| `ios_swift` | cloud |
| `unity` | cloud |
| `react_native` | cloud |
| `flutter` | cloud |
| `cordova` | cloud |
| `cloud` | cloud |

## Per-source keys

Both keys below are objects keyed by the local source type. A key naming a
source type this destination does not support fails validation.

#### `connection_mode` \* — object

Selects the mode per source type. Values are constrained to the modes that
source type supports, so `device` is rejected for every type except `web`:

```yaml
connection_mode:
  web: device
  cloud: cloud
```

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
destination 'hubspot' (type 'hs') does not support source 'my-source':
source type 'amp' is not among supported source types: android, android_kotlin, ...
```

**The config must carry a `connection_mode` entry for that source type.** This
lives on the destination spec, not the connection spec. Without it:

```
destination 'hubspot' config has no 'connection_mode' entry for source type 'web'
```

HubSpot requires no additional config keys to connect a source of any type.

## Secrets

`access_token` is the only secret key. Write it as a `{{ .VAR }}` reference and
supply the value at apply time:

```yaml
access_token: "{{ .HUBSPOT_ACCESS_TOKEN }}"
```

```sh
export RUDDER_HUBSPOT_ACCESS_TOKEN=...
rudder-cli apply

# or
rudder-cli apply --var-file secrets.vars.yaml
```

`rudder-cli import` writes `access_token` back as a `{{ .VAR }}` placeholder
rather than its value, since the API does not return secrets. Fill the
placeholder in before the first apply.
