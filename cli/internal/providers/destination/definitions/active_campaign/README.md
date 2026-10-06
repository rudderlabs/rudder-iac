# ActiveCampaign (`active_campaign`)

ActiveCampaign is an email marketing and CRM destination. RudderStack creates and updates contacts from `identify` calls, and records `track`, `page`, and `screen` activity against them.

In an ActiveCampaign destination spec:

- `type: active_campaign`
- `definition_version: 1`

## Sample configuration

```yaml
version: rudder/v1
kind: destination
metadata:
  name: activecampaign-prod
spec:
  id: activecampaign-prod
  display_name: ActiveCampaign Production
  type: active_campaign
  definition_version: 1
  enabled: true
  config:
    api_url: https://acme.api-us1.com
    api_key: "{{ .ACTIVECAMPAIGN_API_KEY }}"
    event_key: "{{ .ACTIVECAMPAIGN_EVENT_KEY }}"
    actid: "{{ .ACTIVECAMPAIGN_ACTID }}"

    connection_mode:
      web: hybrid
      cloud: cloud
    consent_management:
      web:
        - provider: oneTrust
          consents:
            - marketing
```

The above example connects web sources in `hybrid` mode, so `page` calls run through ActiveCampaign's site tracking script while other events go through its API — see [Source types](#source-types).

## Config keys

`config` accepts only the keys listed below. The [shared config key rules](../README.md#config-key-rules) cover unknown keys, defaults, and immutability.

### Connection

#### `api_url` — string, required

Your ActiveCampaign API URL, from **Settings** > **Developer** in your ActiveCampaign account.

- Must be a URL with a dotted hostname. The `http://` or `https://` scheme is optional.
- Any URL containing `.ngrok.io` is rejected.
- A `{{ path || fallback }}` template is accepted in place of a literal.

#### `api_key` — string, required, secret

Your ActiveCampaign API key, from the same **Developer** settings page as `api_url`.

- At most 100 characters, and must not contain line breaks.
- A `{{ path || fallback }}` template is accepted in place of a literal.

Supply it as a `{{ .VAR }}` reference rather than a literal — see [Secrets](#secrets).

### Event tracking

#### `event_key` — string, secret

Event key unique to your ActiveCampaign account, from **Settings** > **Tracking** > **Event Tracking**. RudderStack uses it to record `track` events.

- At most 100 characters, and must not contain line breaks.
- A `{{ path || fallback }}` template is accepted in place of a literal.

#### `actid` — string, secret

Your ActiveCampaign account ID for event tracking, from **Settings** > **Tracking** > **Event Tracking API**.

- At most 100 characters, and must not contain line breaks.
- A `{{ path || fallback }}` template is accepted in place of a literal.

### Per-source keys

Both keys are objects keyed by the local source type — the tokens listed under [Source types](#source-types). A key naming a source type this destination doesn't support fails validation.

#### `connection_mode` — object

Maps each source type you connect to the mode its events reach ActiveCampaign in, using the modes in [Source types](#source-types).

- An entry is required for each source type you connect — see [Connect a source](#connect-a-source).
- A mode the source type doesn't support on this destination fails validation — for example `device` for `android`.

```yaml
connection_mode:
  web: hybrid
  android_kotlin: cloud
```

#### `consent_management` — object

Consent provider configuration per source type. The entry shape, accepted providers, and the rules on `resolution_strategy` and `consents` are shared across all destination types — see [Consent management](../common/README.md).

## Source types

ActiveCampaign accepts events from these source types in the mentioned connection modes:

| Source type | Connection mode |
| :-----| :-----|
| `android` | `cloud` |
| `android_kotlin` | `cloud` |
| `ios` | `cloud` |
| `ios_swift` | `cloud` |
| `web` | `cloud`, `device`, `hybrid` |
| `unity` | `cloud` |
| `cloud` | `cloud` |
| `react_native` | `cloud` |
| `flutter` | `cloud` |
| `cordova` | `cloud` |
| `warehouse` | `cloud` |

Only `web` offers a choice. In `hybrid` mode, RudderStack sends `identify`, `track`, and `screen` events through ActiveCampaign's API and `page` events through ActiveCampaign's native web SDK, which is how you get site tracking.

`warehouse` is the token a Reverse ETL source resolves to — see [Source types](../README.md#source-types).

> [!NOTE]
> The dashboard additionally offers ActiveCampaign to AMP and Shopify sources. Rudder CLI doesn't manage those connections, so `amp` and `shopify` are invalid here.

## Connect a source

An event stream connection to this destination is checked against two rules at `validate` time.

**The source's type must be supported.** A source's type resolves to one of the tokens above before the check — a JavaScript source resolves to `web`, and webhook and server-side SDK sources resolve to `cloud`. An unsupported type reports:

```text
destination 'activecampaign-prod' (type 'active_campaign') does not support source 'my-source':
source type 'amp' is not among supported source types: android, android_kotlin, ...
```

**The destination config must carry a `connection_mode` entry for that source type.** This lives on the destination spec, not on the connection spec. Without it:

```text
destination 'activecampaign-prod' config has no 'connection_mode' entry for source type 'web'
```

ActiveCampaign needs no additional config keys to connect a source of any type, in any mode.

A Reverse ETL connection reaches this destination as source type `warehouse` and is checked against the same rules, so the config needs a `connection_mode.warehouse: cloud` entry for it.

## Secrets

`api_key`, `event_key`, and `actid` are the secret keys. Write each as a `{{ .VAR }}` reference and supply the value at apply time:

```yaml
config:
  api_key: "{{ .ACTIVECAMPAIGN_API_KEY }}"
  event_key: "{{ .ACTIVECAMPAIGN_EVENT_KEY }}"
  actid: "{{ .ACTIVECAMPAIGN_ACTID }}"
```

```bash
export RUDDER_ACTIVECAMPAIGN_API_KEY="..."
rudder-cli apply

# or
rudder-cli apply --var-file secrets.vars.yaml
```

Note that:

- The YAML that `rudder-cli import` writes may or may not include secret keys. Before you apply, make sure every secret key your configuration needs is present and populated through variable substitution.
- `api_url` isn't a secret and is imported as-is.
