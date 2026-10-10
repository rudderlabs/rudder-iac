# Statsig (`statsig`)

Statsig is a feature flagging and experimentation destination. RudderStack forwards events unchanged to Statsig's RudderStack integration from its servers, authenticating with your Statsig project's server secret key.

In a Statsig destination spec:

- `type: statsig`
- `definition_version: 1`

> [!NOTE]
> `statsig` is an [unverified destination type](../README.md#unverified-destination-types). Rudder CLI registers it only when the `unverifiedDestinations` experimental flag is on.

## Sample configuration

```yaml
version: rudder/v1
kind: destination
metadata:
  name: statsig-prod
spec:
  id: statsig-prod
  display_name: Statsig Production
  type: statsig
  definition_version: 1
  enabled: true
  config:
    secret_key: "{{ .STATSIG_SERVER_SECRET_KEY }}"

    connection_mode:
      web: cloud
      android_kotlin: cloud
      cloud: cloud
    consent_management:
      web:
        - provider: oneTrust
          consents:
            - analytics
```

The above example connects web, Android Kotlin, and server-side sources, each in `cloud`, the only mode Statsig offers — see [Source types](#source-types). The secret key is a `{{ .VAR }}` reference — see [Secrets](#secrets).

## Config keys

`config` accepts only the keys listed below. The [shared config key rules](../README.md#config-key-rules) cover unknown keys, defaults, and immutability.

> [!NOTE]
> Statsig accepts `alias`, `group`, `identify`, `page`, `screen`, and `track` events. Statsig takes them in only once its RudderStack integration is enabled in the Statsig console.

### Connection

#### `secret_key` — string, required, secret

Server secret key of your Statsig project, from the **API Keys** tab of the project settings.

- At most 200 characters, and must not contain line breaks.
- A `{{ path || fallback }}` template is accepted in place of a literal, and isn't measured against the length limit.

Supply it as a `{{ .VAR }}` reference rather than a literal — see [Secrets](#secrets).

### Per-source keys

Both keys are objects keyed by the local source type — the tokens listed under [Source types](#source-types). A key naming a source type this destination doesn't support fails validation.

#### `connection_mode` — object

Maps each source type you connect to the mode its events reach Statsig in, using the modes in [Source types](#source-types).

- An entry is required for each source type you connect — see [Connect a source](#connect-a-source).

```yaml
connection_mode:
  web: cloud
  android_kotlin: cloud
  cloud: cloud
```

#### `consent_management` — object

Consent provider configuration per source type. The entry shape, accepted providers, and the rules on `resolution_strategy` and `consents` are shared across all destination types — see [Consent management](../common/README.md).

## Source types

Statsig accepts events from these source types in the mentioned connection modes:

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

Every source type is `cloud` only — events reach Statsig from RudderStack's servers, never in device mode.

The dashboard also accepts `warehouse` for Statsig, but this definition doesn't, so Rudder CLI can't connect a Reverse ETL source to it.

## Connect a source

An event stream connection to this destination is checked against two rules at `validate` time.

**The source's type must be supported.** A source's type resolves to one of the tokens above before the check — a JavaScript source resolves to `web`, and webhook and server-side SDK sources resolve to `cloud`. Every event stream source type is supported here, so only a Reverse ETL connection, whose source resolves to `warehouse`, fails it:

```text
destination 'statsig-prod' (type 'statsig') does not accept rETL sources: source type 'warehouse' is not among supported source types: android, android_kotlin, ...
```

**The destination config must carry a `connection_mode` entry for that source type.** This lives on the destination spec, not on the connection spec. Without it:

```text
destination 'statsig-prod' config has no 'connection_mode' entry for source type 'web'
```

Statsig needs no additional config keys to connect a source of any type.

## Secrets

`secret_key` is the only secret key. Write it as a `{{ .VAR }}` reference — [Secrets](../README.md#secrets) covers supplying the values and what `import` writes:

```yaml
config:
  secret_key: "{{ .STATSIG_SERVER_SECRET_KEY }}"
```
