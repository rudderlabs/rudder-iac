# Destination types

Every destination spec names a `type`, and that type decides which `config` keys the spec accepts. Each verified type documents those keys in a README in its own directory.

The rules in this file apply to every type. The per-type READMEs cover only what's specific to that destination. Breaking changes between releases are listed in [BREAKING_CHANGES.md](BREAKING_CHANGES.md).

## Supported destination types

Use the `type` value in YAML. Every type below takes `definition_version: 1`.

| Destination | Type |
| :-----| :-----|
| [ActiveCampaign](active_campaign/README.md) | `active_campaign` |
| [Adjust](adj/README.md) | `adj` |
| [Adobe Analytics](adobe_analytics/README.md) | `adobe_analytics` |
| [Amazon Kinesis](kinesis/README.md) | `kinesis` |
| [Amazon Redshift](rs/README.md) | `rs` |
| [Amazon S3](s3/README.md) | `s3` |
| [Amplitude](am/README.md) | `am` |
| [Apache Kafka](kafka/README.md) | `kafka` |
| [Attentive Tag](attentive_tag/README.md) | `attentive_tag` |
| [BigQuery](bq/README.md) | `bq` |
| [BigQuery Stream](bqstream/README.md) | `bqstream` |
| [Braze](braze/README.md) | `braze` |
| [Confluent Cloud](confluent_cloud/README.md) | `confluent_cloud` |
| [Customer.io](customerio/README.md) | `customerio` |
| [Facebook Conversions](facebook_conversions/README.md) | `facebook_conversions` |
| [Facebook Pixel](facebook_pixel/README.md) | `facebook_pixel` |
| [Firebase](firebase/README.md) | `firebase` |
| [Google Ads](googleads/README.md) | `googleads` |
| [Google Analytics](ga/README.md) | `ga` |
| [Google Analytics 4](ga4/README.md) | `ga4` |
| [Google Cloud Pub/Sub](googlepubsub/README.md) | `googlepubsub` |
| [Google Cloud Storage](gcs/README.md) | `gcs` |
| [Google Sheets](googlesheets/README.md) | `googlesheets` |
| [Google Tag Manager](gtm/README.md) | `gtm` |
| [HTTP Webhook](http/README.md) | `http` |
| [HubSpot](hs/README.md) | `hs` |
| [Intercom](intercom/README.md) | `intercom` |
| [Iterable](iterable/README.md) | `iterable` |
| [LinkedIn Insight Tag](linkedin_insight_tag/README.md) | `linkedin_insight_tag` |
| [Marketo](marketo/README.md) | `marketo` |
| [Mixpanel](mp/README.md) | `mp` |
| [PostgreSQL](postgres/README.md) | `postgres` |
| [PostHog](posthog/README.md) | `posthog` |
| [Qualtrics](qualtrics/README.md) | `qualtrics` |
| [Redis](redis/README.md) | `redis` |
| [S3 Data Lake](s3_datalake/README.md) | `s3_datalake` |
| [Salesforce](salesforce/README.md) | `salesforce` |
| [Sentry](sentry/README.md) | `sentry` |
| [Slack](slack/README.md) | `slack` |
| [Snowflake](snowflake/README.md) | `snowflake` |
| [Snowflake Streaming](snowpipe_streaming/README.md) | `snowpipe_streaming` |
| [Statsig](statsig/README.md) | `statsig` |
| [TikTok Ads](tiktok_ads/README.md) | `tiktok_ads` |
| [VWO](vwo/README.md) | `vwo` |
| [Webhook](webhook/README.md) | `webhook` |
| [Zendesk](zendesk/README.md) | `zendesk` |

These are the verified definitions, which Rudder CLI always registers. The [unverified destination types](#unverified-destination-types) register only behind an experimental flag. Any other `type` fails validation.

To start from a destination already configured in the dashboard, run `rudder-cli import workspace` — the imported YAML uses the same keys the READMEs document.

## Unverified destination types

The types below have definitions in Rudder CLI that aren't verified yet. Rudder CLI registers them only when the `unverifiedDestinations` experimental flag is on; without it, a spec using one of these types fails validation.

Turn the flag on with environment variables:

```bash
export RUDDERSTACK_CLI_EXPERIMENTAL=true
export RUDDERSTACK_X_UNVERIFIED_DESTINATIONS=true
```

Or persist it in `~/.rudder/config.json`:

```json
{
  "experimental": true,
  "flags": {
    "unverifiedDestinations": true
  }
}
```

`rudder-cli experimental enable unverifiedDestinations` writes only the flag. Set `experimental` as well, or the flag is ignored.

| Destination | Type |
| :-----| :-----|
| [BingAds Offline Conversions](bingads_offline_conversions/README.md) | `bingads_offline_conversions` |
| [Customer.io Audience](customerio_audience/README.md) | `customerio_audience` |
| [Facebook Custom Audience](fb_custom_audience/README.md) | `fb_custom_audience` |
| [Google Ads Offline Conversions](google_adwords_offline_conversions/README.md) | `google_adwords_offline_conversions` |
| [Google Ads Remarketing Lists](google_adwords_remarketing_lists/README.md) | `google_adwords_remarketing_lists` |
| [Iterable Audience](iterable_audience/README.md) | `iterable_audience` |
| [LinkedIn Ads](linkedin_ads/README.md) | `linkedin_ads` |

Every unverified type takes `definition_version: 1`, and the [config key rules](#config-key-rules) apply to them as they do to verified types.

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

A type models the keys its upstream `schema.json` declares, apart from the few its definition leaves out on purpose, such as the legacy `oneTrustCookieCategories` and `ketchConsentPurposes` blocks the backend folds into `consent_management`. An update or an import replaces the destination's whole config with the spec's, so a key the spec doesn't carry is removed: an empty value `import` pruned, or a key the type doesn't model. `apply --dry-run` shows what an import changes on the destination it adopts, field by field, under `Importable resources, changed on import`, including removed empty values, a transformation link the import connects, disconnects or switches to another transformation, and, under `Config (API keys the import removes)`, every remote field the import drops that local config can't hold, named by its API path (such as `consentManagement.amp` or `headers[0].description`) with its value summarised rather than printed; a destination it adopts unchanged stays under `Importable resources`. The API never returns some keys at all — keys outside the type's schema, and secret keys — so neither the plan nor the dry run can show those, and an update removes them from the stored config. A secret's value lives in a separate store and is kept.

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

A Reverse ETL connection reaches its destination as a `warehouse` source, and is checked against the same destination config rules as an event stream connection, except that it needs no `connection_mode` entry: `cloud` is the only mode a warehouse source runs in, so an absent entry means `cloud`. A destination can't receive from both event stream and Reverse ETL sources in the same project.

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

For an event stream source type that supports `cloud`, an omitted entry implies `cloud` (including when that source type also offers `device` or `hybrid`). A cloud-only `warehouse` source also needs no entry. Device-only source types still require an explicit mode. Without one, `validate` reports:

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
- `validate` warns with `destination/secret-reference` when a secret key holds anything but one reference with no default, and with `destination/secret-reference-unique` when one variable fills two secret keys.
- Substitution happens before YAML parsing. A one-line value containing double quotes, such as a compact JSON key file, needs a single-quoted reference: `'{{ .VAR }}'`. A multi-line value that fills a quoted reference keeps its line breaks in either quote style, including the single-quoted references generated by `import workspace`; use a YAML literal block in the var file for PEM certificates and keys.
- `rudder-cli import` writes a secret key as a reference named after the spec `id` and the key, upper-cased with `-` turned into `_`, such as `{{ .BRAZE_PROD_APP_KEY }}` for `app_key` on a destination imported as `braze-prod`, and lists it in a `secrets.vars.yaml` file next to the specs for you to fill in. It writes one for a secret the API returns with a value, and for one the API leaves out of its response that the config requires: required by the type's own config rules, such as Google Sheets' `credentials`, an HTTP `bearer_token` when `auth` is `bearerTokenAuth`, or S3's `access_key_id` and `access_key` when `role_based_auth` is `false`; or required to connect a source in the mode it runs in, such as Braze's `rest_api_key` when the same import brings a Reverse ETL connection to that destination. A secret the API returns as `null` or `""` is unset, so it is left out, and so is one the config can do without, such as Amplitude's `api_secret`: add it yourself if the destination uses it. Two imported resources whose ids and keys would give the same variable name, such as destinations `hook` with `bearer_token` and `hook-bearer` with `token`, or a destination and an account both imported as `pg` with a password, stop the import before anything is written: rename one. So does a new reference whose name a spec already in the project uses for a different key, such as an account `pg` adopted earlier; the same key of the same resource reusing its variable is fine.

See [variable substitution](../../../varsubst/README.md) for how references resolve.
