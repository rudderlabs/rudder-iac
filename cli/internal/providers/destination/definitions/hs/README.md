# HubSpot (`hs`)

HubSpot is a CRM and marketing destination. RudderStack creates and updates contacts from `identify` calls and records `track` events, from its servers or — for web sources — through HubSpot's own script in device mode.

In a HubSpot destination spec:

- `type: hs`
- `definition_version: 1`

## Sample configuration

```yaml
version: rudder/v1
kind: destination
metadata:
  name: hubspot-prod
spec:
  id: hubspot-prod
  display_name: HubSpot Production
  type: hs
  definition_version: 1
  enabled: true
  config:
    api_version: newApi
    access_token: "{{ .HUBSPOT_ACCESS_TOKEN }}"
    hub_id: "{{ .HUBSPOT_HUB_ID }}"
    lookup_field: email
    do_association: false

    hubspot_events:
      - rs_event_name: Order Completed
        hubspot_event_name: pe12345_order_completed
        event_properties:
          - from: revenue
            to: order_value

    connection_mode:
      web: cloud
      cloud: cloud
    consent_management:
      web:
        - provider: oneTrust
          consents:
            - marketing
```

The above example uses the new API with `email` as the upsert key, and connects web sources in `cloud` mode, so it omits `event_filtering` — see [Event filtering](#event-filtering).

## Config keys

`config` accepts only the keys listed below. The [shared config key rules](../README.md#config-key-rules) cover unknown keys, defaults, and immutability.

> [!NOTE]
> In `cloud` mode HubSpot accepts `identify` and `track` events. In `device` mode, web sources also send `page`.

### Connection

#### `api_version` — string, required

HubSpot API that RudderStack writes through.

- `newApi` — HubSpot's v3 API. Use this one.
- `legacyApi` — HubSpot's v1 API, which HubSpot has deprecated. It updates contacts only by email.
- The dashboard defaults this field to `newApi`. Rudder CLI requires it explicitly.
- A spec that omits this key fails validation.

#### `access_token` — string, required, secret

Access token of your HubSpot private app. Used by both API versions.

- At most 100 characters, and must not contain line breaks.
- Templates aren't accepted in place of a literal; a template is measured as text against the same limit.

Supply it as a `{{ .VAR }}` reference rather than a literal — see [Secrets](#secrets).

#### `hub_id` — string, secret

Your HubSpot Hub ID, shown under your account name in HubSpot.

- At most 100 characters, and must not contain line breaks.
- A `{{ path || fallback }}` template is accepted in place of a literal.

### New API settings

These keys apply when `api_version` is `newApi`.

#### `lookup_field` — string, required

HubSpot contact property RudderStack matches on to upsert contacts — for example `email`. Pass the same property, with the value to match, in the `identify` event's `traits`.

- Required when `api_version` is `newApi`. Leave it unset otherwise.
- At most 100 characters, and must not contain line breaks.
- A `{{ path || fallback }}` template is accepted in place of a literal.

Use a property that's unique in HubSpot. Unique properties enable batch upsert, which is much faster; non-unique ones fall back to a slower search-based flow.

#### `do_association` — boolean, default `false`

Create associations between object records. This is used with Reverse ETL sources.

- Applies when `api_version` is `newApi`. Leave it unset otherwise.

#### `hubspot_events` — array of objects

Maps RudderStack `track` events to HubSpot custom behavioral events, with optional property mappings.

- Applies when `api_version` is `newApi`. Leave it unset otherwise.
- `rs_event_name` — RudderStack event name.
- `hubspot_event_name` — internal name of the HubSpot custom behavioral event.
- `event_properties` — array of `from` (RudderStack property) and `to` (HubSpot property) pairs.
- Every string is at most 100 characters, or a `{{ path || fallback }}` template.

```yaml
hubspot_events:
  - rs_event_name: Order Completed
    hubspot_event_name: pe12345_order_completed
    event_properties:
      - from: revenue
        to: order_value
```

### Event filtering

> [!WARNING]
> Client-side event filtering applies only when `connection_mode.web` is `device` — the dashboard shows these controls only then, and the SDK is what applies the filter. Rudder CLI accepts the block in any mode, but events sent in `cloud` mode reach HubSpot unfiltered.

#### `event_filtering` — object

Restricts which `track` events the SDK passes to HubSpot's script, by event name.

- Applies when `connection_mode.web` is `device`. Leave it unset otherwise.
- `whitelist` — array of event names to allow; every other `track` event is dropped.
- `blacklist` — array of event names to drop; every other `track` event is allowed.
- The two are mutually exclusive, and Rudder CLI enforces it — setting both fails validation.
- Each name is at most 100 characters, or a `{{ path || fallback }}` template.

```yaml
connection_mode:
  web: device
event_filtering:
  whitelist:
    - Signed Up
```

### Per-source keys

Both keys are objects keyed by the local source type — the tokens listed under [Source types](#source-types). A key naming a source type this destination doesn't support fails validation.

#### `connection_mode` — object

Maps each source type you connect to the mode its events reach HubSpot in, using the modes in [Source types](#source-types).

- An entry is required for each source type you connect — see [Connect a source](#connect-a-source).
- A mode the source type doesn't support on this destination fails validation — for example `device` for `android`.

```yaml
connection_mode:
  web: device
  cloud: cloud
```

#### `consent_management` — object

Consent provider configuration per source type. The entry shape, accepted providers, and the rules on `resolution_strategy` and `consents` are shared across all destination types — see [Consent management](../common/README.md).

## Source types

HubSpot accepts events from these source types in the mentioned connection modes:

| Source type | Connection mode |
| :-----| :-----|
| `android` | `cloud` |
| `android_kotlin` | `cloud` |
| `ios` | `cloud` |
| `ios_swift` | `cloud` |
| `web` | `cloud`, `device` |
| `unity` | `cloud` |
| `cloud` | `cloud` |
| `react_native` | `cloud` |
| `flutter` | `cloud` |
| `cordova` | `cloud` |
| `warehouse` | `cloud` |

Only `web` offers `device` mode, which loads HubSpot's native script in the browser.

`warehouse` is the token a Reverse ETL source resolves to — see [Source types](../README.md#source-types).

## Connect a source

An event stream connection to this destination is checked against two rules at `validate` time.

**The source's type must be supported.** A source's type resolves to one of the tokens above before the check — a JavaScript source resolves to `web`, and webhook and server-side SDK sources resolve to `cloud`. Every type an event stream or Reverse ETL source can resolve to is supported here, so this check always passes.

**The destination config must carry a `connection_mode` entry for that source type.** This lives on the destination spec, not on the connection spec. Without it:

```text
destination 'hubspot-prod' config has no 'connection_mode' entry for source type 'web'
```

HubSpot needs no additional config keys to connect a source of any type, in any mode.

A Reverse ETL connection reaches this destination as source type `warehouse` and is checked against the same rules, so the config needs a `connection_mode.warehouse: cloud` entry for it.

## Secrets

`access_token` and `hub_id` are the secret keys. Write each as a `{{ .VAR }}` reference — [Secrets](../README.md#secrets) covers supplying the values and what `import` writes:

```yaml
config:
  access_token: "{{ .HUBSPOT_ACCESS_TOKEN }}"
  hub_id: "{{ .HUBSPOT_HUB_ID }}"
```
