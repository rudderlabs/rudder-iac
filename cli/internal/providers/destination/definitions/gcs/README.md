# Google Cloud Storage (`gcs`)

Google Cloud Storage is an object storage destination. RudderStack batches events and writes them as files into a GCS bucket you own.

In a Google Cloud Storage destination spec:

- `type: gcs`
- `definition_version: 1`

## Sample configuration

```yaml
version: rudder/v1
kind: destination
metadata:
  name: gcs-events-prod
spec:
  id: gcs-events-prod
  display_name: GCS Events Production
  type: gcs
  definition_version: 1
  enabled: true
  config:
    bucket_name: rudder-events-prod
    prefix: rudder/events
    credentials: '{{ .GCS_CREDENTIALS }}'

    connection_mode:
      web: cloud
      cloud: cloud
    consent_management:
      web:
        - provider: oneTrust
          consents:
            - analytics
```

The above example supplies the service account key through a variable, which is how you should always set `credentials` — see [Secrets](#secrets).

## Config keys

`config` accepts only the keys listed below. The [shared config key rules](../README.md#config-key-rules) cover unknown keys, defaults, and immutability.

### Bucket

#### `bucket_name` — string, required

Name of the GCS bucket RudderStack writes event files to. The bucket must already exist.

- At most 100 characters, and must not contain line breaks.
- A `{{ path || fallback }}` template is accepted in place of a literal, and isn't measured against the length limit.

#### `prefix` — string

Folder prefix inside the bucket. RudderStack writes all files beneath it.

- At most 100 characters, and must not contain line breaks.
- Templates are accepted on the same terms as `bucket_name`.

### Authentication

#### `credentials` — string, secret

Contents of the JSON key file for a GCP service account that can create objects in the bucket.

- Rudder CLI doesn't require this key and doesn't validate its content.

Supply it as a `{{ .VAR }}` reference rather than a literal — see [Secrets](#secrets).

### Per-source keys

Both keys are objects keyed by the local source type — the tokens listed under [Source types](#source-types). A key naming a source type this destination doesn't support fails validation.

#### `connection_mode` — object

Maps each source type you connect to the mode its events reach GCS in, using the modes in [Source types](#source-types).

- An entry is required for each source type you connect — see [Connect a source](#connect-a-source).

```yaml
connection_mode:
  web: cloud
  cloud: cloud
```

#### `consent_management` — object

Consent provider configuration per source type. The entry shape, accepted providers, and the rules on `resolution_strategy` and `consents` are shared across all destination types — see [Consent management](../common/README.md).

## Source types

Google Cloud Storage accepts events from these source types in the mentioned connection modes:

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

Every source type is `cloud` only — events reach the bucket from RudderStack's servers, never in device mode.

`warehouse` is the token a Reverse ETL source resolves to — see [Source types](../README.md#source-types).

## Connect a source

An event stream connection to this destination is checked against two rules at `validate` time.

**The source's type must be supported.** A source's type resolves to one of the tokens above before the check — a JavaScript source resolves to `web`, and webhook and server-side SDK sources resolve to `cloud`. Every type an event stream or Reverse ETL source can resolve to is supported here, so this check always passes.

**The destination config must carry a `connection_mode` entry for that source type.** This lives on the destination spec, not on the connection spec. Without it:

```text
destination 'gcs-events-prod' config has no 'connection_mode' entry for source type 'web'
```

Google Cloud Storage needs no additional config keys to connect a source of any type.

A Reverse ETL connection reaches this destination as source type `warehouse` and is checked against the same rules, so the config needs a `connection_mode.warehouse: cloud` entry for it.

## Secrets

`credentials` is the only secret key. Write it as a `{{ .VAR }}` reference — [Secrets](../README.md#secrets) covers supplying the values and what `import` writes:

```yaml
config:
  credentials: '{{ .GCS_CREDENTIALS }}'
```

`credentials` is JSON, so write its reference in single quotes, as above — see [Secrets](../README.md#secrets). Export the key file as is:

```bash
export RUDDER_GCS_CREDENTIALS="$(cat service-account.json)"
```
