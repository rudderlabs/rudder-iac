# BigQuery Stream (`bqstream`)

BigQuery Stream streams events into an existing BigQuery table through Google's streaming insert API, so rows are queryable within seconds rather than after a scheduled load.

In a BigQuery Stream destination spec:

- `type: bqstream`
- `definition_version: 1`

## Sample configuration

```yaml
version: rudder/v1
kind: destination
metadata:
  name: bigquery-stream-prod
spec:
  id: bigquery-stream-prod
  display_name: BigQuery Stream Production
  type: bqstream
  definition_version: 1
  enabled: true
  config:
    project_id: acme-analytics
    dataset_id: product_events
    table_id: product_inserts
    insert_id: productId
    credentials: "{{ .BQSTREAM_CREDENTIALS }}"

    connection_mode:
      web: cloud
      cloud: cloud
    consent_management:
      web:
        - provider: oneTrust
          consents:
            - analytics
```

The above example deduplicates on the `productId` event property. Leave `insert_id` out to stream without deduplication — see [Deduplication](#deduplication).

## Config keys

`config` accepts only the keys listed below. The [shared config key rules](../README.md#config-key-rules) cover unknown keys, defaults, and immutability.

> [!NOTE]
> BigQuery Stream accepts only `track` events, and RudderStack forwards each event's properties without fetching or checking the table schema. Make sure the `track` payload matches the columns of the table named in `table_id`.

### Table

#### `project_id` — string, required

GCP project ID that holds the dataset.

#### `dataset_id` — string, required

ID of the dataset, within `project_id`, that holds the table.

#### `table_id` — string, required

ID of the table RudderStack streams events into. The table must already exist.

### Deduplication

#### `insert_id` — string

Name of the event property whose value BigQuery uses as the `insertId` to deduplicate rows — for example `productId`. It names the property, not the value.

- The property's value must be a number or a string.
- Deduplication applies only when the value is present in the event's `properties`.

To pick the property per event, use a template with a fallback:

```yaml
insert_id: '{{ message.uniqueId || "productId" }}'
```

### Authentication

#### `credentials` — string, required, secret

Contents of the JSON key file for a GCP service account that can insert rows into the table.

Supply it as a `{{ .VAR }}` reference rather than a literal — see [Secrets](#secrets).

### Per-source keys

Both keys are objects keyed by the local source type — the tokens listed under [Source types](#source-types). A key naming a source type this destination doesn't support fails validation.

#### `connection_mode` — object

Maps each source type you connect to the mode its events reach BigQuery in, using the modes in [Source types](#source-types).

- An entry is required for each source type you connect — see [Connect a source](#connect-a-source).

```yaml
connection_mode:
  web: cloud
  cloud: cloud
```

#### `consent_management` — object

Consent provider configuration per source type. The entry shape, accepted providers, and the rules on `resolution_strategy` and `consents` are shared across all destination types — see [Consent management](../common/README.md).

## Source types

BigQuery Stream accepts events from these source types in the mentioned connection modes:

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

Every source type is `cloud` only — events reach the table from RudderStack's servers, never in device mode.

`warehouse` is the token a Reverse ETL source resolves to — see [Source types](../README.md#source-types).

> [!NOTE]
> The dashboard additionally offers BigQuery Stream to AMP and Shopify sources. Rudder CLI doesn't manage those connections, so `amp` and `shopify` are invalid here.

## Connect a source

An event stream connection to this destination is checked against two rules at `validate` time.

**The source's type must be supported.** A source's type resolves to one of the tokens above before the check — a JavaScript source resolves to `web`, and webhook and server-side SDK sources resolve to `cloud`. An unsupported type reports:

```text
destination 'bigquery-stream-prod' (type 'bqstream') does not support source 'my-source':
source type 'amp' is not among supported source types: android, android_kotlin, ...
```

**The destination config must carry a `connection_mode` entry for that source type.** This lives on the destination spec, not on the connection spec. Without it:

```text
destination 'bigquery-stream-prod' config has no 'connection_mode' entry for source type 'web'
```

BigQuery Stream needs no additional config keys to connect a source of any type.

A Reverse ETL connection reaches this destination as source type `warehouse` and is checked against the same rules, so the config needs a `connection_mode.warehouse: cloud` entry for it.

## Secrets

`credentials` is the only secret key. Write it as a `{{ .VAR }}` reference and supply the value at apply time:

```yaml
config:
  credentials: "{{ .BQSTREAM_CREDENTIALS }}"
```

```bash
export RUDDER_BQSTREAM_CREDENTIALS="$(cat service-account.json)"
rudder-cli apply

# or
rudder-cli apply --var-file secrets.vars.yaml
```

Note that:

- The YAML that `rudder-cli import` writes may or may not include secret keys. Before you apply, make sure every secret key your configuration needs is present and populated through variable substitution.
