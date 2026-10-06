# Attentive Tag (`attentive_tag`)

Attentive Tag is an SMS and email marketing destination. RudderStack sends `identify` and `track` events to Attentive to subscribe users and record their activity.

In an Attentive Tag destination spec:

- `type: attentive_tag`
- `definition_version: 1`

## Sample configuration

```yaml
version: rudder/v1
kind: destination
metadata:
  name: attentive-prod
spec:
  id: attentive-prod
  display_name: Attentive Production
  type: attentive_tag
  definition_version: 1
  enabled: true
  config:
    api_key: "{{ .ATTENTIVE_API_KEY }}"
    sign_up_source_id: "123456"
    enable_new_identify_flow: false

    connection_mode:
      web: cloud
      cloud: cloud
    consent_management:
      web:
        - provider: oneTrust
          consents:
            - marketing
```

The above example keeps the legacy subscription flow for `identify` calls. `sign_up_source_id` is written as a quoted digit string — see [Connection](#connection).

## Config keys

`config` accepts only the keys listed below. The [shared config key rules](../README.md#config-key-rules) cover unknown keys, defaults, and immutability.

> [!NOTE]
> Attentive Tag accepts only `identify` and `track` events. RudderStack drops other event types for this destination.

### Connection

#### `api_key` — string, required, secret

API key of the app you created in your Attentive dashboard.

- At most 100 characters, and must not contain line breaks.
- A `{{ path || fallback }}` template is accepted in place of a literal.

Supply it as a `{{ .VAR }}` reference rather than a literal — see [Secrets](#secrets).

#### `sign_up_source_id` — string

ID of the sign-up method, from **Sign-up Units** in your Attentive dashboard, that RudderStack attributes subscriptions to.

- Digits only. Quote it in YAML so it stays a string — `"123456"`, not `123456`.
- A `{{ path || fallback }}` template is accepted in place of a literal.

### Identify behavior

#### `enable_new_identify_flow` — boolean, default `false`

Sync `identify` calls through Attentive's Identity and Custom Attributes APIs instead of the legacy subscribe and unsubscribe flows. The dashboard marks this setting as beta.

### Per-source keys

Both keys are objects keyed by the local source type — the tokens listed under [Source types](#source-types). A key naming a source type this destination doesn't support fails validation.

#### `connection_mode` — object

Maps each source type you connect to the mode its events reach Attentive in, using the modes in [Source types](#source-types).

- An entry is required for each source type you connect — see [Connect a source](#connect-a-source).

```yaml
connection_mode:
  web: cloud
  cloud: cloud
```

#### `consent_management` — object

Consent provider configuration per source type. The entry shape, accepted providers, and the rules on `resolution_strategy` and `consents` are shared across all destination types — see [Consent management](../common/README.md).

## Source types

Attentive Tag accepts events from these source types in the mentioned connection modes:

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

Every source type is `cloud` only — events reach Attentive from RudderStack's servers, never in device mode.

`warehouse` is the token a Reverse ETL source resolves to — see [Source types](../README.md#source-types).

> [!NOTE]
> The dashboard additionally offers Attentive Tag to AMP and Shopify sources. Rudder CLI doesn't manage those connections, so `amp` and `shopify` are invalid here.

## Connect a source

An event stream connection to this destination is checked against two rules at `validate` time.

**The source's type must be supported.** A source's type resolves to one of the tokens above before the check — a JavaScript source resolves to `web`, and webhook and server-side SDK sources resolve to `cloud`. An unsupported type reports:

```text
destination 'attentive-prod' (type 'attentive_tag') does not support source 'my-source':
source type 'amp' is not among supported source types: android, android_kotlin, ...
```

**The destination config must carry a `connection_mode` entry for that source type.** This lives on the destination spec, not on the connection spec. Without it:

```text
destination 'attentive-prod' config has no 'connection_mode' entry for source type 'web'
```

Attentive Tag needs no additional config keys to connect a source of any type.

A Reverse ETL connection reaches this destination as source type `warehouse` and is checked against the same rules, so the config needs a `connection_mode.warehouse: cloud` entry for it.

## Secrets

`api_key` is the only secret key. Write it as a `{{ .VAR }}` reference and supply the value at apply time:

```yaml
config:
  api_key: "{{ .ATTENTIVE_API_KEY }}"
```

```bash
export RUDDER_ATTENTIVE_API_KEY="..."
rudder-cli apply

# or
rudder-cli apply --var-file secrets.vars.yaml
```

Note that:

- The YAML that `rudder-cli import` writes may or may not include secret keys. Before you apply, make sure every secret key your configuration needs is present and populated through variable substitution.
