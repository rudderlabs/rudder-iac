# Destination types

Every destination spec names a `type`, and that type decides which `config` keys the spec accepts. Each verified type documents those keys in a README in its own directory.

The rules in this file apply to every type. The per-type READMEs cover only what's specific to that destination.

## Supported destination types

Use the `type` value in YAML. Every type below takes `definition_version: 1`.

| Destination | Type |
| :-----| :-----|
| [ActiveCampaign](active_campaign/README.md) | `active_campaign` |
| [Amazon Redshift](rs/README.md) | `rs` |
| [Amazon S3](s3/README.md) | `s3` |
| [Amplitude](am/README.md) | `am` |
| [Attentive Tag](attentive_tag/README.md) | `attentive_tag` |
| [BigQuery](bq/README.md) | `bq` |
| [BigQuery Stream](bqstream/README.md) | `bqstream` |
| [Braze](braze/README.md) | `braze` |
| [Customer.io](customerio/README.md) | `customerio` |
| [Facebook Conversions](facebook_conversions/README.md) | `facebook_conversions` |
| [Facebook Pixel](facebook_pixel/README.md) | `facebook_pixel` |
| [Google Ads](googleads/README.md) | `googleads` |
| [Google Analytics 4](ga4/README.md) | `ga4` |
| [Google Cloud Storage](gcs/README.md) | `gcs` |
| [HTTP Webhook](http/README.md) | `http` |
| [HubSpot](hs/README.md) | `hs` |
| [Iterable](iterable/README.md) | `iterable` |
| [Mixpanel](mp/README.md) | `mp` |
| [PostgreSQL](postgres/README.md) | `postgres` |
| [PostHog](posthog/README.md) | `posthog` |
| [S3 Data Lake](s3_datalake/README.md) | `s3_datalake` |
| [Snowflake](snowflake/README.md) | `snowflake` |
| [TikTok Ads](tiktok_ads/README.md) | `tiktok_ads` |
| [Webhook](webhook/README.md) | `webhook` |

These are the verified definitions, which Rudder CLI always registers. The other definitions in this directory register only behind the `unverifiedDestinations` experimental flag and aren't documented here. Any other `type` fails validation.

To start from a destination already configured in the dashboard, run `rudder-cli import workspace` — the imported YAML uses the same keys the READMEs document.

## Config key rules

These rules hold for every destination type.

### Keys are snake_case

Local YAML uses snake_case config keys. Rudder CLI converts them to the API's camelCase at apply time. A camelCase key in the spec is treated as an unknown key.

### Unknown keys fail validation

`config` accepts only the keys its type declares. Anything else fails `rudder-cli validate` with:

```text
unknown config field "<key>"
```

A key that's valid for one destination type isn't valid for another. There are no shared optional extras beyond `connection_mode` and `consent_management`.

### Omitting a key with a default is safe

Where a key declares a default, Rudder CLI fills that value in before it validates the spec and before the spec reaches the API, because the backend applies the same default when it stores the destination. Omitting such a key is equivalent to writing its default: it doesn't produce a permanent diff on the next `apply`, and it's validated as though you had written the default — which can make another key required.

A default inside a nested block, such as `sdk_version.web`, fills in only when the spec writes that block. Omit the whole block and nothing is sent for it.

Keys without a default are sent only when you set them.

### Some keys can't change after the first apply

A key marked **immutable** in a type README is fixed once the destination exists. The API rejects an update that changes it.

> [!WARNING]
> Rudder CLI doesn't check immutability locally. `validate` accepts the change and `apply` sends it, so the failure surfaces at the API rather than in your pull request. To change an immutable key, create a new destination instead.

### Some keys aren't in the dashboard

A key marked **internal** in a type README isn't surfaced by the RudderStack dashboard. Rudder CLI accepts it so that a spec carrying an existing value, such as one written by `import`, keeps it on update. Set it only if you know your destination needs it.

## Source types

A destination declares the source types it accepts events from. Each type README lists its own set, drawn from the tokens below.

| Source type | Sources that resolve to it |
| :-----| :-----|
| `web` | JavaScript SDK |
| `android` | Android SDK |
| `android_kotlin` | Android Kotlin SDK |
| `ios` | iOS SDK |
| `ios_swift` | iOS Swift SDK |
| `react_native` | React Native SDK |
| `flutter` | Flutter SDK |
| `cordova` | Cordova SDK |
| `unity` | Unity SDK |
| `amp` | AMP Analytics |
| `shopify` | Shopify |
| `cloud` | Webhook sources and every server-side SDK — Node.js, Python, Go, Java, Ruby, PHP, .NET, Rust |
| `cloud_source` | Cloud app sources |
| `warehouse` | Reverse ETL sources |

A source's own definition resolves to exactly one token. Note that `cloud` and `cloud_source` are different tokens: a webhook or server-side SDK source resolves to `cloud`, while a cloud app source resolves to `cloud_source`.

Rudder CLI doesn't manage connections from AMP, Shopify, or cloud app sources, so `amp`, `shopify`, and `cloud_source` aren't valid for any type here, even where the dashboard offers the destination to them.

A Reverse ETL connection reaches its destination as a `warehouse` source, and is checked against the same destination config rules as an event stream connection. Reverse ETL connections sit behind the `retlConnectionSupport` experimental flag. A destination can't receive from both event stream and Reverse ETL sources in the same project.

## Connection modes

A connection mode describes how a given **source type** connecting to **this destination instance** routes its events:

| Connection mode | Description |
| :----| :----|
| `cloud` | The SDK sends events to RudderStack, which forwards them to the destination from its servers. |
| `device` | The SDK loads the destination's own SDK and sends events to it directly from the device or browser. |
| `hybrid` | Both at once — RudderStack forwards events from its servers, and also loads the destination's SDK for the features that need it, like in-app messages and push notifications. [ActiveCampaign](active_campaign/README.md), [Braze](braze/README.md), and [Google Analytics 4](ga4/README.md) support it on some source types. |

Note that:

- The connection mode is a property of the source-destination pair, not of the source or the destination alone.
- Two destinations fed by the same source can run in different modes.
- A source type supported in `device` or `hybrid` mode by one destination may be `cloud`-only on another.

### Where the mode is declared

`connection_mode` lives in the destination's `config` block, keyed by local source type — not on the connection spec.

```yaml
config:
  connection_mode:
    web: device
    android_kotlin: cloud
```

Naming a source type the destination doesn't support fails validation, and so does a mode that source type doesn't support on this destination.

An entry is required for every source type you connect. Without one, `validate` reports:

```text
destination '<id>' config has no 'connection_mode' entry for source type '<source_type>'
```

## Consent management

`consent_management` is accepted by every destination type, keyed by local source type. Each source type maps to an array of consent entries. The entry shape, accepted providers, and value rules are documented in [common/README.md](common/README.md).

## Secrets

Each type README lists the keys Rudder CLI treats as secrets. Write each one you use as a `{{ .VARIABLE_NAME }}` reference rather than a literal, and supply the value from the environment, prefixed with `RUDDER_`, or from a var file:

```bash
export RUDDER_BRAZE_REST_API_KEY="..."
rudder-cli apply

# or
rudder-cli apply --var-file secrets.vars.yaml
```

Note that:

- `validate` substitutes variables too, so every variable a spec references must be set when you run it. An undefined one stops the run.
- Substitution pastes each value into the YAML as is, before the YAML is parsed. A value containing double quotes, such as a JSON key file, needs a single-quoted reference: `'{{ .VAR }}'`. A value spanning several lines, such as a PEM key, needs its line breaks written as `\n` inside a double-quoted reference. The type READMEs show the form where a key needs it.
- `rudder-cli import` writes each secret the destination has set as a reference named after the spec `id` and the key, upper-cased with `-` turned into `_`. For example, `credentials` on a destination imported as `bigquery-prod` becomes `{{ .BIGQUERY_PROD_CREDENTIALS }}`, and a nested key adds its path, such as `{{ .ORDERS_WEBHOOK_HEADERS_0_TO }}`. Supply those variables before you apply. `import` writes every reference in double quotes, so switch a JSON-valued one, such as BigQuery's `credentials`, to single quotes.

See [variable substitution](../../../varsubst/README.md) for how references resolve.
